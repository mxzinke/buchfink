package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// ContactService manages debtor/creditor master data and their Personenkonten.
type ContactService struct {
	contactRepo domain.ContactRepository
	journalRepo domain.JournalRepository
	numberRepo  domain.NumberRangeRepository
	auditRepo   domain.AuditRepository
	// vatIDs ist die Bestätigungsabfrage. Sie ist optional: ohne sie bleibt der
	// Hinweis beim Speichern allgemein.
	vatIDs     vatIDStatusSource
	fiscalYear int
}

// NewContactService creates the master data service.
func NewContactService(
	contactRepo domain.ContactRepository,
	journalRepo domain.JournalRepository,
	numberRepo domain.NumberRangeRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		journalRepo: journalRepo,
		numberRepo:  numberRepo,
		auditRepo:   auditRepo,
		fiscalYear:  fiscalYear,
	}
}

// SetFiscalYear updates the year the open item balances are computed for.
func (s *ContactService) SetFiscalYear(year int) { s.fiscalYear = year }

// GetContacts returns all business partners with the balance of their
// Personenkonto (the open item total) filled in.
func (s *ContactService) GetContacts(ctx context.Context) ([]domain.Contact, error) {
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if s.journalRepo == nil {
		return contacts, nil
	}

	turnovers, err := s.journalRepo.AccountTurnovers(ctx, s.fiscalYear)
	if err != nil {
		return contacts, nil
	}

	for i := range contacts {
		t := turnovers[contacts[i].LedgerAccount]
		if contacts[i].Type == domain.ContactTypeCustomer {
			contacts[i].OpenAmount = t.Debit - t.Credit
		} else {
			contacts[i].OpenAmount = t.Credit - t.Debit
		}
	}
	return contacts, nil
}

// SaveContact stores a business partner, assigning a Personenkonto from the
// DATEV number range on first save.
func (s *ContactService) SaveContact(ctx context.Context, c *domain.Contact) error {
	c.VatIDNotice = ""
	if c.Type != domain.ContactTypeCustomer && c.Type != domain.ContactTypeVendor {
		return fmt.Errorf("Kontakttyp muss Kunde (Debitor) oder Lieferant (Kreditor) sein")
	}
	if c.Name == "" {
		return fmt.Errorf("Name des Geschäftspartners fehlt")
	}

	action := domain.AuditActionUpdate
	if c.ID == 0 {
		action = domain.AuditActionCreate
	}

	if c.LedgerAccount == "" {
		account, err := s.allocateLedgerAccount(ctx, c.Type)
		if err != nil {
			return err
		}
		c.LedgerAccount = account
	}

	if c.CountryCode == "" {
		c.CountryCode = "DE"
	}
	// Wer nur die alte einzeilige Anschrift schickt, bekommt sie zerlegt: die
	// Rechnung braucht Straße, PLZ und Ort in Feldern (§ 14 Abs. 4 Nr. 1 UStG,
	// BT-50/52/53). Was sich nicht sicher zerlegen lässt, bleibt liegen und
	// fällt auf der Kontaktseite als unvollständig auf — eine geratene Straße
	// wäre schlechter als eine fehlende.
	c.MigrateAddress()
	// Ohne Zielformat gilt der Regelfall. Ein leeres Feld wäre kein „kein
	// Format", sondern ein Kontakt, an den sich keine Rechnung stellen ließe.
	if c.EInvoiceProfile == "" {
		c.EInvoiceProfile = domain.EInvoiceProfileZUGFeRD
	}
	// Ein unbekanntes Zielformat wird abgewiesen und nicht gespeichert. Vorher
	// wurde nur das leere Feld belegt: ein Tippfehler oder ein Format aus der
	// Fachplanung, das Buchfink nicht erzeugt („xrechnung_ubl"), stand danach am
	// Kontakt und wirkte beim Ausstellen still wie ZUGFeRD — der Empfänger bekam
	// ein anderes Dokument, als an ihm hinterlegt war.
	if err := c.EInvoiceProfile.Validate(); err != nil {
		return err
	}

	if err := s.contactRepo.Save(ctx, c); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, action, "CONTACT", fmt.Sprintf("%d", c.ID),
			fmt.Sprintf("Geschäftspartner %s, Personenkonto %s (%s)", c.Name, c.LedgerAccount, c.Type))
	}
	c.VatIDNotice = s.vatIDNotice(ctx, c)
	return nil
}

// vatIDNotice ist der Hinweis zur Bestätigung einer EU-USt-IdNr.
//
// Er hält nichts an. Die Bestätigungsanfrage ist beim Ausstellen einer
// steuerfreien innergemeinschaftlichen Lieferung zwingend (§ 6a Abs. 1 Satz 1
// Nr. 4 UStG); beim Erfassen des Kontakts ist sie eine gute Gewohnheit, und ein
// Netzaufruf, den niemand angefordert hat, gehört nicht in das Speichern von
// Stammdaten. Der Hinweis sagt deshalb, was ist, und bietet die Abfrage an.
func (s *ContactService) vatIDNotice(ctx context.Context, c *domain.Contact) string {
	if strings.TrimSpace(c.VatID) == "" || !c.IsEUCounterparty() {
		return ""
	}
	if s.vatIDs != nil {
		if status, err := s.vatIDs.Status(ctx, c); err == nil && status != nil {
			if status.Confirmed {
				return status.Note
			}
			return status.Note + " Hole die qualifizierte Bestätigungsanfrage nach § 18e UStG nach: " +
				"beim Ausstellen einer steuerfreien innergemeinschaftlichen Lieferung ist sie zwingend."
		}
	}
	return fmt.Sprintf(
		"%s trägt die USt-IdNr. %s aus einem anderen Mitgliedstaat. Lass sie beim Bundeszentralamt "+
			"bestätigen (§ 18e UStG) — beim Ausstellen einer steuerfreien innergemeinschaftlichen "+
			"Lieferung ist die gültige Nummer materielle Voraussetzung der Befreiung "+
			"(§ 6a Abs. 1 Satz 1 Nr. 4 UStG).", c.Name, c.VatID)
}

// vatIDStatusSource ist der Ausschnitt der Bestätigungsabfrage, den die
// Kontaktverwaltung für ihren Hinweis braucht.
type vatIDStatusSource interface {
	Status(ctx context.Context, contact *domain.Contact) (*VatIDStatus, error)
}

// SetVatIDStatusSource koppelt die Bestätigungsabfrage an die Kontaktverwaltung.
// Ohne sie bleibt der Hinweis allgemein statt konkret.
func (s *ContactService) SetVatIDStatusSource(src vatIDStatusSource) { s.vatIDs = src }

// allocateLedgerAccount takes the next free number from the Debitoren or
// Kreditoren range. Numbers are never reused, so a deleted partner does not free
// its account for a different one — an old booking must stay attributable.
func (s *ContactService) allocateLedgerAccount(ctx context.Context, kind domain.ContactType) (string, error) {
	if s.numberRepo == nil {
		return "", fmt.Errorf("Nummernkreis für Personenkonten ist nicht verfügbar")
	}

	key := domain.NumberRangeDebitor
	if kind == domain.ContactTypeVendor {
		key = domain.NumberRangeCreditor
	}

	for attempt := 0; attempt < 100; attempt++ {
		seq, err := s.numberRepo.Allocate(ctx, key, 0)
		if err != nil {
			return "", err
		}
		account, err := domain.FormatLedgerAccount(kind, seq)
		if err != nil {
			return "", err
		}
		// Guard against a counter that fell behind manually assigned accounts.
		existing, err := s.contactRepo.FindByLedgerAccount(ctx, account)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return account, nil
		}
	}
	return "", fmt.Errorf("es konnte kein freies Personenkonto vergeben werden")
}

// DeleteContact removes a business partner that carries no bookings.
func (s *ContactService) DeleteContact(ctx context.Context, id uint) error {
	contact, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if s.journalRepo != nil {
		entries, err := s.journalRepo.FindByAccount(ctx, contact.LedgerAccount, 0)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf(
				"%s hat %d Buchungen auf Personenkonto %s und kann nicht gelöscht werden",
				contact.Name, len(entries), contact.LedgerAccount,
			)
		}
	}

	if err := s.contactRepo.Delete(ctx, id); err != nil {
		return err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "CONTACT", fmt.Sprintf("%d", id),
			fmt.Sprintf("Geschäftspartner %s (Personenkonto %s) gelöscht", contact.Name, contact.LedgerAccount))
	}
	return nil
}

// ExemptionCertificateWarning ist eine ablaufende oder abgelaufene
// Freistellungsbescheinigung nach § 48b EStG.
//
// Wer eine Bauleistung bezieht, hat nach § 48 EStG 15 % der Gegenleistung
// einzubehalten — es sei denn, der Leistende legt eine gültige Bescheinigung
// vor. Der Bauabzug selbst ist nicht Teil von Buchfink; die Frist wird trotzdem
// überwacht, weil eine abgelaufene Bescheinigung sonst erst auffällt, wenn die
// Haftung schon entstanden ist.
type ExemptionCertificateWarning struct {
	ContactID  uint   `json:"contactId"`
	Name       string `json:"name"`
	Number     string `json:"number"`
	ValidUntil string `json:"validUntil"`
	// State ist „expiring" oder „expired".
	State string `json:"state"`
	Note  string `json:"note"`
}

// ExemptionCertificateWarnings liefert die Bescheinigungen, die in den nächsten
// 30 Tagen ablaufen oder abgelaufen sind.
//
// today wird übergeben und nicht aus der Uhr genommen: eine Frist, die sich beim
// Testen nicht setzen lässt, ist eine Frist, die nicht geprüft wird. Leer heißt:
// heute.
func (s *ContactService) ExemptionCertificateWarnings(
	ctx context.Context, today string,
) ([]ExemptionCertificateWarning, error) {
	if today == "" {
		today = time.Now().Format("2006-01-02")
	}
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ExemptionCertificateWarning, 0)
	for i := range contacts {
		c := &contacts[i]
		state := c.ExemptionCertificateState(today)
		if state != "expiring" && state != "expired" {
			continue
		}
		note := fmt.Sprintf(
			"Die Freistellungsbescheinigung von %s läuft am %s ab. Ohne sie sind bei einer "+
				"Bauleistung 15 %% der Gegenleistung einzubehalten (§ 48 EStG).",
			c.Name, c.ExemptionCertificateValidUntil)
		if state == "expired" {
			note = fmt.Sprintf(
				"Die Freistellungsbescheinigung von %s ist am %s abgelaufen. Ohne sie sind bei einer "+
					"Bauleistung 15 %% der Gegenleistung einzubehalten und an das Finanzamt abzuführen "+
					"(§ 48 EStG).",
				c.Name, c.ExemptionCertificateValidUntil)
		}
		out = append(out, ExemptionCertificateWarning{
			ContactID: c.ID, Name: c.Name,
			Number:     c.ExemptionCertificateNumber,
			ValidUntil: c.ExemptionCertificateValidUntil,
			State:      state, Note: note,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ValidUntil < out[j].ValidUntil })
	return out, nil
}
