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
// Bei einer sonstigen Leistung an einen Unternehmer im übrigen
// Gemeinschaftsgebiet weist die gültige USt-IdNr. den Empfänger als Unternehmer
// aus; daran hängt der Leistungsort im Ausland (§ 3a Abs. 2 UStG, Art. 18
// MwStVO). Wer sie nicht prüft, trägt das Risiko, und die Bestätigungsanfrage
// nach § 18e UStG ist die Sorgfalt, die das Gesetz erwartet.
//
// Der Dienst blockiert das Ausstellen einer solchen Rechnung, wenn die
// Bestätigung negativ ausfällt. Er blockiert es nicht, wenn das Bundeszentralamt
// nicht erreichbar ist — dann steht der Anwender vor einer Entscheidung, die er
// mit einem dokumentierten Grund trifft. Ein local-first-Programm darf nicht
// stehenbleiben, weil eine fremde Behörde gerade wartet.
type VatIDService struct {
	auditor[uint]
	repo         domain.VatIDCheckRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	newClient    func() VatIDClient
	now          func() time.Time
}

// VatIDClient ist die Bestätigungsanfrage beim Bundeszentralamt für Steuern.
type VatIDClient interface {
	Check(ctx context.Context, req vatid.Request) (*vatid.Result, error)
	Endpoint() string
}

// NewVatIDService wires die Bestätigungsanfrage.
func NewVatIDService(
	repo domain.VatIDCheckRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *VatIDService {
	return &VatIDService{
		repo: repo, contactRepo: contactRepo, settingsRepo: settingsRepo, auditor: auditor[uint]{auditRepo: auditRepo, entity: "VAT_ID_CHECK"},
		newClient: func() VatIDClient { return vatid.New(vatid.DefaultEndpoint) },
		now:       time.Now,
	}
}

// SetClientFactory ersetzt den Erzeuger des Clients: der Kern legt seinen
// Netzwächter davor, Tests einen httptest-Server.
func (s *VatIDService) SetClientFactory(f func() VatIDClient) { s.newClient = f }

// SetClock ersetzt die Uhr. Nur für Tests — die Frist von 90 Tagen richtet sich nach ihr.
func (s *VatIDService) SetClock(now func() time.Time) { s.now = now }

// Checks liefert den Verlauf der Abfragen zu einem Kontakt.
func (s *VatIDService) Checks(ctx context.Context, contactID uint) ([]domain.VatIDCheck, error) {
	if s.repo == nil {
		return []domain.VatIDCheck{}, nil
	}
	return s.repo.FindByContact(ctx, contactID)
}

// VatIDQuery ist eine vorbereitete Bestätigungsanfrage. Sie trennt das Netz
// vom Schreiben: vorbereitet und abgelegt wird in der Datenbank, gefragt wird
// dazwischen ohne Schreibsperre.
type VatIDQuery struct {
	Contact  *domain.Contact
	Request  vatid.Request
	client   VatIDClient
	endpoint string
}

// PrepareCheck liest, was die Anfrage braucht. Es schreibt nichts.
func (s *VatIDService) PrepareCheck(ctx context.Context, contactID uint) (*VatIDQuery, error) {
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
	street, postalCode, city := contact.PostalAddress()
	client := s.newClient()
	return &VatIDQuery{
		Contact: contact,
		Request: vatid.Request{
			OwnVatID:    strings.TrimSpace(s.ownVatID(ctx)),
			VatID:       strings.TrimSpace(contact.VatID),
			CompanyName: contactCompanyName(contact),
			City:        city,
			PostalCode:  postalCode,
			Street:      street,
		},
		client: client, endpoint: client.Endpoint(),
	}, nil
}

// Run fragt beim Bundeszentralamt an. Es ist der einzige Netzaufruf der
// Anfrage und gehört nicht in einen Schreibvorgang.
func (q *VatIDQuery) Run(ctx context.Context) (*vatid.Result, error) {
	return q.client.Check(ctx, q.Request)
}

// Check führt die qualifizierte Bestätigungsanfrage aus und hebt das Ergebnis
// auf. Der Kern teilt sie in PrepareCheck, Run und RecordCheck, damit das Netz
// keine Schreibsperre hält.
func (s *VatIDService) Check(ctx context.Context, contactID uint) (*domain.VatIDCheck, error) {
	query, err := s.PrepareCheck(ctx, contactID)
	if err != nil {
		return nil, err
	}
	result, runErr := query.Run(ctx)
	check, err := s.RecordCheck(ctx, query, result, runErr)
	if err != nil {
		return nil, err
	}
	if runErr != nil {
		return nil, runErr
	}
	return check, nil
}

// RecordCheck hebt das Ergebnis einer Anfrage auf.
//
// Auch das negative: gerade es ist der Nachweis, dass geprüft wurde. Nur wo gar
// keine Auskunft kam, wird nichts gespeichert — eine nicht beantwortete Anfrage
// ist kein Ergebnis über den Geschäftspartner. Sie steht nur im Protokoll, und
// RecordCheck gibt (nil, nil) zurück: der Eintrag soll bleiben, den
// ausgebliebenen Bescheid meldet der Aufrufer nach dem Abschluss.
func (s *VatIDService) RecordCheck(
	ctx context.Context, query *VatIDQuery, result *vatid.Result, runErr error,
) (*domain.VatIDCheck, error) {
	contact := query.Contact
	if runErr != nil {
		return nil, s.audit(ctx, domain.AuditActionUpdate, contact.ID, fmt.Sprintf(
			"Bestätigungsanfrage für %s ohne Ergebnis: %v", contact.Name, runErr))
	}

	check := &domain.VatIDCheck{
		ContactID: contact.ID,
		VatID:     query.Request.VatID,
		OwnVatID:  query.Request.OwnVatID,
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
		Endpoint:         query.endpoint,
	}
	if err := s.repo.Save(ctx, check); err != nil {
		return nil, fmt.Errorf("das Ergebnis der Bestätigungsanfrage ließ sich nicht speichern: %w", err)
	}
	if err := s.audit(ctx, domain.AuditActionUpdate, contact.ID, fmt.Sprintf(
		"Bestätigungsanfrage für %s (%s): %s", contact.Name, check.VatID, check.Summary())); err != nil {
		return nil, err
	}
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

// EnsureConfirmed ist die Prüfung vor dem Ausstellen einer Leistung an einen
// Unternehmer im übrigen Gemeinschaftsgebiet.
//
// Sie fragt nicht selbst beim Bundeszentralamt an: das Ausstellen ist ein
// Schreibvorgang, und ein Netzaufruf darin hielte die Schreibsperre. Gefragt
// wird vorher (core.Tenant.CheckVatID); hier zählt das abgelegte Ergebnis.
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
			"für eine Leistung an ein Unternehmen in einem anderen Mitgliedstaat braucht %s eine USt-IdNr. "+
				"(§ 3a Abs. 2 UStG)", contact.Name)
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
				"jeder Anfrage. Tragen Sie sie unter Unternehmen › Stammdaten › Steuern ein und stellen "+
				"Sie die Rechnung an %s danach aus. Ein Grund hilft hier nicht weiter: die Abfrage ist nicht "+
				"fehlgeschlagen, sie ist noch nicht möglich", contact.Name)
	}

	if check, err := s.latestAnswer(ctx, contact.ID, status.VatID); err != nil {
		return err
	} else if check != nil && check.Status != domain.VatIDValid {
		return fmt.Errorf(
			"das Bundeszentralamt für Steuern hat die USt-IdNr. %s von %s nicht bestätigt. %s. Die "+
				"Steuerschuld geht nur auf einen Empfänger über, der als Unternehmer ausgewiesen ist "+
				"(§ 3a Abs. 2, § 13b Abs. 1 UStG)",
			check.VatID, contact.Name, check.Summary())
	}

	// Keine Auskunft. Das ist kein negatives Ergebnis, und es darf nicht wie
	// eines wirken — aber es ist auch kein Nachweis.
	if strings.TrimSpace(overrideReason) != "" {
		return s.audit(ctx, domain.AuditActionUpdate, contact.ID, fmt.Sprintf(
			"Rechnung an %s ohne Bestätigung der USt-IdNr. ausgestellt: %s",
			contact.Name, strings.TrimSpace(overrideReason)))
	}
	return fmt.Errorf(
		"für die USt-IdNr. von %s liegt keine Bestätigung vor: aus den letzten %d Tagen fehlt eine Auskunft des "+
			"Bundeszentralamts. Ohne Bestätigung ist der Leistungsort im Ausland nicht belegt. "+
			"Die Abfrage ist zu wiederholen, sobald das Netz erreichbar ist, oder die Rechnung geht mit einem "+
			"festgehaltenen Grund hinaus — der nächste Prüflauf führt sie dann als offen",
		contact.Name, domain.VatIDCheckValidityDays)
}

// NeedsCheck meldet, ob vor einer Leistung an den Geschäftspartner beim
// Bundeszentralamt anzufragen ist: er hat eine USt-IdNr., die eigene ist
// erfasst, und eine frische Bestätigung fehlt.
func (s *VatIDService) NeedsCheck(ctx context.Context, contact *domain.Contact) (bool, error) {
	status, err := s.Status(ctx, contact)
	if err != nil {
		return false, err
	}
	return status.VatID != "" && !status.Confirmed && s.ownVatID(ctx) != "", nil
}

// latestAnswer ist die jüngste Antwort zur Nummer aus der Frist, gleich wie sie
// ausfiel, oder nil.
func (s *VatIDService) latestAnswer(ctx context.Context, contactID uint, vatID string) (*domain.VatIDCheck, error) {
	if s.repo == nil {
		return nil, nil
	}
	checks, err := s.repo.FindByContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	horizon := s.now().Add(-domain.VatIDCheckValidityDays * 24 * time.Hour)
	for i := range checks {
		if checks[i].VatID != vatID {
			continue
		}
		checked, err := time.Parse(time.RFC3339, checks[i].CheckedAt)
		if err != nil || checked.Before(horizon) {
			return nil, nil
		}
		return &checks[i], nil
	}
	return nil, nil
}

// germanTimestamp rendert einen RFC3339-Zeitpunkt als deutsches Datum.
func germanTimestamp(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Local().Format("02.01.2006 15:04")
}
