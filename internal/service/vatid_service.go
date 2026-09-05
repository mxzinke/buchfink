package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/vatid"
)

// VatIDService holt und verwahrt die Bestätigungen ausländischer USt-IdNr.
//
// § 6a Abs. 1 Satz 1 Nr. 4 UStG macht die gültige, vom Bestimmungsland erteilte
// USt-IdNr. des Abnehmers zur materiellen Voraussetzung der steuerfreien
// innergemeinschaftlichen Lieferung. Wer sie nicht prüft, trägt das Risiko: die
// Vertrauensschutzregel des § 6a Abs. 4 UStG greift nur für den, der die
// Sorgfalt eines ordentlichen Kaufmanns angewandt hat — und die qualifizierte
// Bestätigungsanfrage nach § 18e UStG ist genau das.
//
// Der Dienst blockiert das Ausstellen einer steuerfreien Lieferung, wenn die
// Bestätigung negativ ausfällt. Er blockiert es nicht, wenn das Bundeszentralamt
// nicht erreichbar ist — dann steht der Anwender vor einer Entscheidung, die er
// mit einem dokumentierten Grund trifft. Ein local-first-Programm darf nicht
// stehenbleiben, weil eine fremde Behörde gerade wartet.
type VatIDService struct {
	repo         domain.VatIDCheckRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	// client wird je Aufruf aus der eingestellten Adresse gebaut. Ein
	// festgehaltener Client hielte die Adresse von damals fest.
	newClient func(endpoint string) *vatid.Client
	now       func() time.Time
}

// SettingVatIDEndpoint ist der Schlüssel der einstellbaren Adresse.
const SettingVatIDEndpoint = "vatid_endpoint"

// NewVatIDService wires die Bestätigungsanfrage.
func NewVatIDService(
	repo domain.VatIDCheckRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *VatIDService {
	return &VatIDService{
		repo: repo, contactRepo: contactRepo, settingsRepo: settingsRepo, auditRepo: auditRepo,
		newClient: vatid.New,
		now:       time.Now,
	}
}

// SetClientFactory ersetzt den Erzeuger des Clients. Nur für Tests: sie legen
// einen httptest-Server vor die Schnittstelle.
func (s *VatIDService) SetClientFactory(f func(endpoint string) *vatid.Client) { s.newClient = f }

// SetClock ersetzt die Uhr. Nur für Tests — die Frist von 90 Tagen hängt an ihr.
func (s *VatIDService) SetClock(now func() time.Time) { s.now = now }

// Endpoint liefert die eingestellte Adresse des Bundeszentralamts.
func (s *VatIDService) Endpoint(ctx context.Context) string {
	if s.settingsRepo == nil {
		return vatid.DefaultEndpoint
	}
	value, err := s.settingsRepo.Get(ctx, SettingVatIDEndpoint)
	if err != nil || strings.TrimSpace(value) == "" {
		return vatid.DefaultEndpoint
	}
	return strings.TrimSpace(value)
}

// Checks liefert den Verlauf der Abfragen zu einem Kontakt.
func (s *VatIDService) Checks(ctx context.Context, contactID uint) ([]domain.VatIDCheck, error) {
	if s.repo == nil {
		return []domain.VatIDCheck{}, nil
	}
	return s.repo.FindByContact(ctx, contactID)
}

// Check führt die qualifizierte Bestätigungsanfrage aus und hebt das Ergebnis
// auf.
//
// Auch das negative: gerade es ist der Nachweis, dass geprüft wurde. Nur wo gar
// keine Auskunft kam, wird nichts gespeichert — eine nicht beantwortete Anfrage
// ist kein Ergebnis über den Geschäftspartner.
func (s *VatIDService) Check(ctx context.Context, contactID uint) (*domain.VatIDCheck, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("die Bestätigungsabfrage ist nicht eingerichtet")
	}
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("der Geschäftspartner konnte nicht geladen werden: %w", err)
	}
	if strings.TrimSpace(contact.VatID) == "" {
		return nil, fmt.Errorf("%s hat keine USt-IdNr., die zu bestätigen wäre", contact.Name)
	}
	own := s.ownVatID(ctx)

	street, postalCode, city := contact.PostalAddress()
	endpoint := s.Endpoint(ctx)
	result, err := s.newClient(endpoint).Check(ctx, vatid.Request{
		OwnVatID:    strings.TrimSpace(own),
		VatID:       strings.TrimSpace(contact.VatID),
		CompanyName: contactCompanyName(contact),
		City:        city,
		PostalCode:  postalCode,
		Street:      street,
	})
	if err != nil {
		s.audit(ctx, contactID, fmt.Sprintf(
			"Bestätigungsanfrage für %s ohne Ergebnis: %v", contact.Name, err))
		return nil, err
	}

	check := &domain.VatIDCheck{
		ContactID: contactID,
		VatID:     strings.TrimSpace(contact.VatID),
		OwnVatID:  strings.TrimSpace(own),
		CheckedAt: result.CheckedAt.Format(time.RFC3339),
		Status:    result.Status,
		// Der Ergebniscode und der rohe Datensatz sind der Nachweis. Sie werden
		// aufgehoben, wie sie kamen: eine aufbereitete Zusammenfassung ist gegenüber
		// dem Finanzamt nichts wert.
		ResultCode:       result.Code,
		ResultText:       result.Text,
		RequestID:        result.RequestID,
		NameResult:       result.Name,
		CityResult:       result.City,
		PostalCodeResult: result.PostalCode,
		StreetResult:     result.Street,
		RawResponse:      result.Raw,
		Endpoint:         endpoint,
	}
	if err := s.repo.Save(ctx, check); err != nil {
		return nil, fmt.Errorf("das Ergebnis der Bestätigungsanfrage ließ sich nicht speichern: %w", err)
	}
	s.audit(ctx, contactID, fmt.Sprintf(
		"Bestätigungsanfrage für %s (%s): %s", contact.Name, check.VatID, check.Summary()))
	return check, nil
}

// ownVatID liefert die eigene USt-IdNr. aus den Unternehmensdaten.
//
// Sie gehört zu jeder Bestätigungsanfrage: § 18e UStG lässt sie nur dem zu, der
// selbst eine USt-IdNr. hat, und das Bundeszentralamt beantwortet eine Anfrage
// ohne sie nicht.
func (s *VatIDService) ownVatID(ctx context.Context) string {
	if s.settingsRepo == nil {
		return ""
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil {
		return ""
	}
	return strings.TrimSpace(settings.VatID)
}

// contactCompanyName liefert den Namen, unter dem der Partner geführt wird. Die
// Firma geht vor: sie ist das, was im Register des Bestimmungslandes steht.
func contactCompanyName(c *domain.Contact) string {
	if strings.TrimSpace(c.Company) != "" {
		return strings.TrimSpace(c.Company)
	}
	return strings.TrimSpace(c.Name)
}

// VatIDStatus ist der Bestätigungsstand eines Geschäftspartners.
type VatIDStatus struct {
	ContactID uint   `json:"contactId"`
	VatID     string `json:"vatId"`
	// Confirmed sagt, ob eine gültige Bestätigung innerhalb der Frist vorliegt.
	Confirmed bool               `json:"confirmed"`
	Latest    *domain.VatIDCheck `json:"latest,omitempty"`
	// ValidityDays ist die Frist, mit der Buchfink rechnet.
	ValidityDays int    `json:"validityDays"`
	Note         string `json:"note"`
}

// Status beantwortet, ob eine frische Bestätigung vorliegt — ohne das Netz zu
// bemühen.
func (s *VatIDService) Status(ctx context.Context, contact *domain.Contact) (*VatIDStatus, error) {
	out := &VatIDStatus{
		ContactID:    contact.ID,
		VatID:        strings.TrimSpace(contact.VatID),
		ValidityDays: domain.VatIDCheckValidityDays,
	}
	if out.VatID == "" {
		out.Note = fmt.Sprintf("%s hat keine USt-IdNr. erfasst.", contact.Name)
		return out, nil
	}
	if s.repo == nil {
		out.Note = "Die Bestätigungsabfrage ist nicht eingerichtet."
		return out, nil
	}
	latest, err := s.repo.FindLatestValid(ctx, contact.ID, out.VatID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		out.Note = fmt.Sprintf(
			"Für die USt-IdNr. %s liegt keine gültige Bestätigung vor.", out.VatID)
		return out, nil
	}
	out.Latest = latest
	out.Confirmed = latest.Fresh(s.now())
	if out.Confirmed {
		out.Note = fmt.Sprintf(
			"Die USt-IdNr. %s wurde am %s bestätigt (Abfrage-Nr. %s).",
			out.VatID, germanTimestamp(latest.CheckedAt), latest.RequestID)
		return out, nil
	}
	out.Note = fmt.Sprintf(
		"Die letzte Bestätigung der USt-IdNr. %s stammt vom %s und ist damit älter als %d Tage. "+
			"Bei laufenden Geschäftsbeziehungen wird regelmäßig neu abgefragt.",
		out.VatID, germanTimestamp(latest.CheckedAt), domain.VatIDCheckValidityDays)
	return out, nil
}

// StatusForContact ist derselbe Bestätigungsstand, nur über die Kennung des
// Geschäftspartners gefragt.
//
// Der Prüflauf hat die Rechnung und nicht den Kontakt; ihn dafür ein
// Kontaktverzeichnis führen zu lassen, hieße die Kartei ein zweites Mal
// anzuschließen.
func (s *VatIDService) StatusForContact(ctx context.Context, contactID uint) (*VatIDStatus, error) {
	if s.contactRepo == nil {
		return nil, fmt.Errorf("die Kontaktverwaltung ist nicht eingerichtet")
	}
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("der Geschäftspartner konnte nicht geladen werden: %w", err)
	}
	return s.Status(ctx, contact)
}

// EnsureConfirmed ist die Prüfung vor dem Ausstellen einer steuerfreien
// Lieferung.
//
// Sie gibt einen Fehler zurück, wo die Ausstellung zu unterbleiben hat, und nil,
// wo sie laufen darf. Die drei Fälle sind bewusst verschieden behandelt: eine
// frische Bestätigung genügt, eine negative Antwort hält die Rechnung an, und
// eine ausgebliebene Antwort lässt sie nur mit einem festgehaltenen Grund
// passieren.
func (s *VatIDService) EnsureConfirmed(
	ctx context.Context, contact *domain.Contact, overrideReason string,
) error {
	status, err := s.Status(ctx, contact)
	if err != nil {
		return err
	}
	if status.VatID == "" {
		return fmt.Errorf(
			"für eine steuerfreie Lieferung in einen anderen Mitgliedstaat braucht %s eine USt-IdNr. "+
				"(§ 6a Abs. 1 Satz 1 Nr. 4 UStG)", contact.Name)
	}
	if status.Confirmed {
		return nil
	}

	// Fehlt die eigene USt-IdNr., kommt die Anfrage gar nicht zustande. Dieser
	// Fall darf nicht in den Zweig der ausgebliebenen Auskunft laufen: dort ließe
	// er sich mit einem Grund übersteuern, und die Rechnung ginge ohne
	// Bestätigung hinaus, obwohl der Mangel in den eigenen Stammdaten liegt und
	// ohne Netz in einer Minute behoben ist. Deshalb ohne Übersteuerung.
	if s.ownVatID(ctx) == "" {
		return fmt.Errorf(
			"für die Bestätigungsanfrage fehlt die eigene USt-IdNr. Sie ist nach § 18e UStG Teil "+
				"jeder Anfrage — trage sie in den Unternehmensdaten ein und stelle die Rechnung an "+
				"%s danach aus. Ein Grund hilft hier nicht weiter: die Abfrage ist nicht "+
				"fehlgeschlagen, sie ist noch nicht möglich", contact.Name)
	}

	check, err := s.Check(ctx, contact.ID)
	if err != nil {
		// Keine Auskunft. Das ist kein negatives Ergebnis, und es darf nicht wie
		// eines wirken — aber es ist auch kein Nachweis.
		if strings.TrimSpace(overrideReason) != "" {
			s.audit(ctx, contact.ID, fmt.Sprintf(
				"Rechnung an %s ohne Bestätigung der USt-IdNr. ausgestellt: %s (Fehler: %v)",
				contact.Name, strings.TrimSpace(overrideReason), err))
			return nil
		}
		return fmt.Errorf(
			"die USt-IdNr. von %s ließ sich nicht bestätigen: %w. Ohne Bestätigung ist die "+
				"Steuerbefreiung nicht belegt. Stelle die Rechnung mit einem festgehaltenen Grund aus "+
				"oder wiederhole die Abfrage später — der nächste Prüflauf führt sie als offen",
			contact.Name, err)
	}
	if check.Status == domain.VatIDValid {
		return nil
	}
	return fmt.Errorf(
		"das Bundeszentralamt für Steuern hat die USt-IdNr. %s von %s nicht bestätigt. %s. Eine "+
			"steuerfreie innergemeinschaftliche Lieferung setzt eine gültige, vom Bestimmungsland "+
			"erteilte USt-IdNr. des Abnehmers voraus (§ 6a Abs. 1 Satz 1 Nr. 4 UStG)",
		check.VatID, contact.Name, check.Summary())
}

// germanTimestamp rendert einen RFC3339-Zeitpunkt als deutsches Datum.
func germanTimestamp(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Local().Format("02.01.2006 15:04")
}

func (s *VatIDService) audit(ctx context.Context, contactID uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "VAT_ID_CHECK",
		fmt.Sprintf("%d", contactID), details)
}
