package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// OpeningBalanceLine ist ein Sachkonto der Eröffnungsbilanz mit seinem Wert.
type OpeningBalanceLine struct {
	Account string       `json:"account"`
	Side    domain.Side  `json:"side"`
	Amount  domain.Cents `json:"amount"`
	// LegacyRef ist die Kennung, unter der der Posten im Altsystem geführt
	// wurde.
	LegacyRef string `json:"legacyRef,omitempty"`
}

// OpeningOpenItem ist ein offener Posten des Umsteigers.
//
// Er wird einzeln übernommen und nicht als Summe auf dem Sammelkonto: eine
// Forderung über 1.200 € gegen einen Kunden ist ein Posten mit einer Nummer und
// einer Fälligkeit, und ohne beides ließe sich sie nach der Übernahme nicht
// mehr ausgleichen — die Offene-Posten-Liste des ersten Jahres wäre leer,
// während in der Bilanz 40.000 € Forderungen stünden.
type OpenOpeningItem struct {
	ContactID      uint         `json:"contactId"`
	Amount         domain.Cents `json:"amount"`
	DocumentNumber string       `json:"documentNumber"`
	DocumentDate   string       `json:"documentDate"`
	DueDate        string       `json:"dueDate,omitempty"`
	LegacyRef      string       `json:"legacyRef,omitempty"`
}

// OpeningBalanceRequest ist die Eröffnungsbilanz des Umsteigers.
type OpeningBalanceRequest struct {
	FiscalYear int `json:"fiscalYear"`
	// Date ist der Buchungstag; leer heißt der erste Tag des Geschäftsjahres.
	Date string `json:"date,omitempty"`
	// ReceiptID ist der Beleg mit der Schlussbilanz des Altsystems. Er ist
	// Pflicht: eine Eröffnungsbilanz ohne Beleg wäre eine Behauptung über
	// Zahlen, die aus einem anderen System stammen (§ 146 Abs. 1 AO, GoBD
	// Rz. 61).
	ReceiptID *uint `json:"receiptId,omitempty"`
	// LegacySystem benennt das Altsystem, aus dem übernommen wird.
	LegacySystem string               `json:"legacySystem,omitempty"`
	Accounts     []OpeningBalanceLine `json:"accounts"`
	Receivables  []OpenOpeningItem    `json:"receivables"`
	Payables     []OpenOpeningItem    `json:"payables"`
}

// OpeningBalancePreview ist die Vorschau auf die Eröffnungsbuchungen.
type OpeningBalancePreview struct {
	FiscalYear  int                    `json:"fiscalYear"`
	Date        string                 `json:"date"`
	Entries     []*domain.JournalEntry `json:"entries"`
	DebitTotal  domain.Cents           `json:"debitTotal"`
	CreditTotal domain.Cents           `json:"creditTotal"`
	Balanced    bool                   `json:"balanced"`
	Messages    []string               `json:"messages"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (p *OpeningBalancePreview) EnsureLists() {
	if p.Entries == nil {
		p.Entries = make([]*domain.JournalEntry, 0)
	}
	for _, entry := range p.Entries {
		entry.EnsureLists()
	}
	if p.Messages == nil {
		p.Messages = make([]string, 0)
	}
}

// PreviewOpeningBalance baut die Eröffnungsbuchungen, ohne sie zu schreiben.
//
// Vorschau und Buchung teilen sich denselben Bau: eine Vorschau, die etwas
// anderes zeigt als das, was gebucht wird, ist schlimmer als keine.
func (s *JournalService) PreviewOpeningBalance(
	ctx context.Context, req OpeningBalanceRequest,
) (*OpeningBalancePreview, error) {
	entries, preview, err := s.buildOpeningBalance(ctx, req)
	if err != nil {
		return nil, err
	}
	preview.Entries = entries
	preview.EnsureLists()
	return preview, nil
}

// BookOpeningBalance bucht die Eröffnungsbilanz eines Umsteigers.
//
// Gegen die Saldenvortragskonten 9000 (Sachkonten), 9008 (Debitoren) und 9009
// (Kreditoren): die Gegenbuchung des Vortrags ist kein Aufwand und kein Ertrag,
// sie schließt nur die Buchung. Jede Buchung trägt EntrySourceOpening und die
// Herkunftskennung aus dem Altsystem.
//
// Alles oder nichts: geprüft wird jede Buchung, bevor die erste geschrieben
// wird. Eine halb erfasste Eröffnungsbilanz wäre eine Bilanz, die nicht aufgeht
// und sich nur durch Stornos wieder auflösen ließe.
func (s *JournalService) BookOpeningBalance(
	ctx context.Context, req OpeningBalanceRequest,
) (*OpeningBalancePreview, error) {
	entries, preview, err := s.buildOpeningBalance(ctx, req)
	if err != nil {
		return nil, err
	}
	if !preview.Balanced {
		return nil, fmt.Errorf(
			"die Eröffnungsbilanz ist nicht ausgeglichen: Soll %s €, Haben %s €. Prüfe die Schlussbilanz des Altsystems",
			preview.DebitTotal, preview.CreditTotal)
	}
	if err := s.ensureFirstOpeningBalance(ctx, preview.FiscalYear); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := s.ValidatePostable(ctx, entry); err != nil {
			return nil, fmt.Errorf("die Eröffnungsbilanz wurde nicht gebucht: %w", err)
		}
	}

	booked := make([]*domain.JournalEntry, 0, len(entries))
	for _, entry := range entries {
		created, err := s.Post(ctx, entry)
		if err != nil {
			return nil, fmt.Errorf(
				"die Eröffnungsbilanz ist nach %d von %d Buchungen gescheitert: %w. Storniere die bereits erfassten Buchungen, bevor du es erneut versuchst",
				len(booked), len(entries), err)
		}
		booked = append(booked, created)
	}

	if note := s.sealOpeningReceipt(ctx, *req.ReceiptID, booked); note != "" {
		preview.Messages = append(preview.Messages, note)
	}

	preview.Entries = booked
	preview.EnsureLists()
	return preview, nil
}

// sealOpeningReceipt versiegelt die Schlussbilanz des Altsystems, nachdem die
// Eröffnungsbuchungen stehen.
//
// Ohne die Versiegelung bliebe der Beleg im Status „abgelegt": der Prüflauf
// meldete ihn dauerhaft als abgelegt, aber nicht gebucht — blockierend, sobald
// sein Eingang vor dem Stichtag liegt, also bei jeder Festschreibung des ersten
// Jahres —, und seine Kopfdaten ließen sich über SaveHeader weiter ändern,
// obwohl Buchungen auf ihn verweisen und seinen Hash tragen. Versiegelt ist er
// das, was er ist: ein gebuchter Beleg.
//
// Das Versiegeln ist ein zweiter Schreibvorgang nach den Buchungen und darf sie
// nicht zurücknehmen — sie stehen bereits im Journal. Scheitert es, wird das
// gemeldet und nicht als Fehler behandelt; ReceiptService.Get holt das Siegel
// beim nächsten Lesen des Belegs nach.
func (s *JournalService) sealOpeningReceipt(ctx context.Context, receiptID uint, booked []*domain.JournalEntry) string {
	if s.receiptRepo == nil || len(booked) == 0 {
		return ""
	}
	if err := s.receiptRepo.Seal(ctx, receiptID, booked[0].ID); err != nil {
		return fmt.Sprintf(
			"Die Eröffnungsbuchungen stehen, der Beleg der Schlussbilanz ließ sich aber nicht versiegeln: %v. "+
				"Öffne den Beleg in der Belegliste — das Siegel wird dann nachgeholt.", err)
	}
	return ""
}

// ensureFirstOpeningBalance lässt die Eröffnungsbilanz nur einmal und nur im
// ersten Geschäftsjahr zu.
//
// Beides gehört ins Backend und nicht nur in die Oberfläche: der Baustein wird
// dort zwar nur im ersten Jahr angeboten, aber eine Regel, die an der Sichtbarkeit
// eines Knopfes hängt, ist keine. Ein zweiter Aufruf bucht sonst dieselben Werte
// noch einmal — die Bilanz stimmte weiterhin (jede Buchung ist für sich
// ausgeglichen), nur eben mit doppelten Beständen, und die Offene-Posten-Liste
// führte jede übernommene Forderung zweimal.
//
// Ein vorhandener Saldenvortrag zählt mit: er trägt dieselbe Quelle und dieselben
// Werte. Ist er da, ist das Jahr nicht das erste, und die Bestände sind schon da.
func (s *JournalService) ensureFirstOpeningBalance(ctx context.Context, fiscalYear int) error {
	existing, err := s.journalRepo.FindAll(ctx, fiscalYear)
	if err != nil {
		return fmt.Errorf("die vorhandenen Buchungen des Jahres %d ließen sich nicht lesen: %w", fiscalYear, err)
	}
	for i := range existing {
		if existing[i].Source != domain.EntrySourceOpening {
			continue
		}
		return fmt.Errorf(
			"im Geschäftsjahr %d stehen bereits Eröffnungsbuchungen (%s). Eine zweite Eröffnungsbilanz "+
				"verdoppelte die Bestände; eine Korrektur läuft über den Storno der vorhandenen Buchungen",
			fiscalYear, existing[i].EntryNumber)
	}

	prior, err := s.journalRepo.FindAll(ctx, fiscalYear-1)
	if err != nil {
		return fmt.Errorf("die Buchungen des Vorjahres ließen sich nicht lesen: %w", err)
	}
	if len(prior) > 0 {
		return fmt.Errorf(
			"das Geschäftsjahr %d ist nicht das erste: im Vorjahr stehen %d Buchungen. Die Bestände kommen "+
				"dann aus dem Saldenvortrag des Jahresabschlusses und nicht aus einer Eröffnungsbilanz",
			fiscalYear, len(prior))
	}
	return nil
}

// buildOpeningBalance baut die Buchungen und die Kennzahlen der Vorschau.
func (s *JournalService) buildOpeningBalance(
	ctx context.Context, req OpeningBalanceRequest,
) ([]*domain.JournalEntry, *OpeningBalancePreview, error) {
	if req.FiscalYear == 0 {
		req.FiscalYear = s.fiscalYear
	}
	if req.FiscalYear == 0 {
		return nil, nil, fmt.Errorf("ohne Geschäftsjahr lässt sich keine Eröffnungsbilanz erfassen")
	}
	date := req.Date
	if date == "" {
		date = fmt.Sprintf("%d-01-01", req.FiscalYear)
	}
	if req.ReceiptID == nil {
		return nil, nil, fmt.Errorf(
			"zur Eröffnungsbilanz gehört die Schlussbilanz des Altsystems als Beleg. Lege sie zuerst ab (§ 146 Abs. 1 AO)")
	}
	if len(req.Accounts) == 0 && len(req.Receivables) == 0 && len(req.Payables) == 0 {
		return nil, nil, fmt.Errorf("die Eröffnungsbilanz enthält keine Werte")
	}

	preview := &OpeningBalancePreview{FiscalYear: req.FiscalYear, Date: date}
	source := req.LegacySystem
	if source == "" {
		source = "Altsystem"
	}

	entries := make([]*domain.JournalEntry, 0, 1+len(req.Receivables)+len(req.Payables))

	// Die Sachkonten in einer Buchung: sie sind ein Vorgang — die Übernahme der
	// Schlussbilanz —, und je Konto eine eigene Buchung machte aus einer Bilanz
	// hundert Buchungssätze, die einzeln nichts aussagen.
	if len(req.Accounts) > 0 {
		entry := s.newOpeningEntry(req, date, fmt.Sprintf("Eröffnungsbilanz %d aus %s", req.FiscalYear, source))
		var debit, credit domain.Cents
		for i, line := range req.Accounts {
			if line.Amount <= 0 {
				return nil, nil, fmt.Errorf("Zeile %d (%s): der Wert der Eröffnungsbilanz muss positiv sein", i+1, line.Account)
			}
			if line.Side != domain.SideDebit && line.Side != domain.SideCredit {
				return nil, nil, fmt.Errorf("Zeile %d (%s): ungültige Buchungsseite %q", i+1, line.Account, line.Side)
			}
			entry.Lines = append(entry.Lines, domain.JournalLine{
				Position: len(entry.Lines) + 1,
				Side:     line.Side,
				Account:  line.Account,
				Amount:   line.Amount,
				Text:     strings.TrimSpace("Vortrag " + line.LegacyRef),
			})
			if line.Side == domain.SideDebit {
				debit += line.Amount
			} else {
				credit += line.Amount
			}
		}
		// Die Gegenbuchung auf 9000 gleicht die Buchung aus. Sie ist die
		// Differenz der beiden Seiten und steht auf der Seite, die fehlt.
		if diff := debit - credit; diff != 0 {
			side := domain.SideCredit
			amount := diff
			if diff < 0 {
				side = domain.SideDebit
				amount = -diff
			}
			entry.Lines = append(entry.Lines, domain.JournalLine{
				Position: len(entry.Lines) + 1,
				Side:     side,
				Account:  domain.AccountSaldenvortraegeSachkonten,
				Amount:   amount,
				Text:     "Saldenvortrag Sachkonten",
			})
		}
		entries = append(entries, entry)
	}

	// Je offener Posten eine Buchung: der Posten ist die Einheit, die später
	// ausgeglichen wird, und eine Sammelbuchung ließe sich nicht ausgleichen.
	for i := range req.Receivables {
		entry, err := s.openingItemEntry(ctx, req, date, source, &req.Receivables[i], domain.ContactTypeCustomer)
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, entry)
	}
	for i := range req.Payables {
		entry, err := s.openingItemEntry(ctx, req, date, source, &req.Payables[i], domain.ContactTypeVendor)
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, entry)
	}

	// Der Beleg-Hash steht an jeder Eröffnungsbuchung.
	//
	// Er ist die Klammer zwischen Buchung und Beleg: die Journalkette deckt mit
	// ihm die Dateien der Schlussbilanz und (seit BEL-02) ihre Kopfdaten ab.
	// Ohne ihn verwiese die Eröffnungsbilanz nur über eine Nummer auf ein
	// Dokument, dessen Austausch niemandem auffiele — bei ausgerechnet den
	// Buchungen, deren Werte aus einem fremden System stammen und für die es
	// keinen anderen Nachweis gibt (§ 146 Abs. 1 AO, GoBD Rz. 61).
	if s.receiptRepo != nil {
		receipt, err := s.receiptRepo.FindByID(ctx, *req.ReceiptID)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"der Beleg %d mit der Schlussbilanz des Altsystems wurde nicht gefunden: %w", *req.ReceiptID, err)
		}
		if receipt == nil {
			return nil, nil, fmt.Errorf(
				"der Beleg %d mit der Schlussbilanz des Altsystems wurde nicht gefunden", *req.ReceiptID)
		}
		for _, entry := range entries {
			entry.ReceiptHash = receipt.ReceiptHash
		}
	}

	for _, entry := range entries {
		preview.DebitTotal += entry.DebitTotal()
		preview.CreditTotal += entry.CreditTotal()
	}
	preview.Balanced = preview.DebitTotal == preview.CreditTotal
	preview.Messages = append(preview.Messages, fmt.Sprintf(
		"%d Buchungen: %d Sachkontenvortrag, %d Forderungen, %d Verbindlichkeiten.",
		len(entries), boolToInt(len(req.Accounts) > 0), len(req.Receivables), len(req.Payables)))
	if !preview.Balanced {
		preview.Messages = append(preview.Messages,
			"Soll und Haben stimmen nicht überein. Eine Eröffnungsbilanz, die nicht aufgeht, lässt sich nicht buchen.")
	}
	return entries, preview, nil
}

// openingItemEntry baut die Vortragsbuchung eines offenen Postens.
func (s *JournalService) openingItemEntry(
	ctx context.Context, req OpeningBalanceRequest, date, source string,
	item *OpenOpeningItem, kind domain.ContactType,
) (*domain.JournalEntry, error) {
	if item.Amount <= 0 {
		return nil, fmt.Errorf("der offene Posten %s hat keinen Betrag", item.DocumentNumber)
	}
	if s.contactRepo == nil {
		return nil, fmt.Errorf("ohne Kontaktverwaltung lassen sich keine Personenkonten vortragen")
	}
	contact, err := s.contactRepo.FindByID(ctx, item.ContactID)
	if err != nil {
		return nil, fmt.Errorf("der Geschäftspartner des offenen Postens %s wurde nicht gefunden: %w",
			item.DocumentNumber, err)
	}
	if contact.Type != kind {
		return nil, fmt.Errorf("%s ist kein %s und kann den Posten %s nicht tragen",
			contact.Name, contactKindLabel(kind), item.DocumentNumber)
	}
	if err := ensureNotBlocked(contact); err != nil {
		return nil, err
	}

	side := domain.SideDebit
	contra := domain.AccountSaldenvortraegeDebitoren
	if kind == domain.ContactTypeVendor {
		side = domain.SideCredit
		contra = domain.AccountSaldenvortraegeKreditoren
	}

	documentDate := item.DocumentDate
	if documentDate == "" {
		documentDate = date
	}
	entry := s.newOpeningEntry(req, date, fmt.Sprintf(
		"Vortrag offener Posten %s (%s) aus %s", item.DocumentNumber, contact.Name, source))
	entry.DocumentDate = documentDate
	entry.ServiceDateFrom = documentDate
	entry.ServiceDateTo = documentDate
	entry.DocumentNumber = item.DocumentNumber
	entry.ContactID = &contact.ID
	// Die Fälligkeit kommt aus dem Altsystem und wird übernommen: sie wurde
	// dort vereinbart, und die Altersstruktur des ersten Jahres hinge sonst am
	// Zahlungsziel des Kontakts statt an dem, was mit ihm ausgemacht war.
	entry.DueDate = item.DueDate
	if item.LegacyRef != "" {
		entry.LegacyRef = item.LegacyRef
	}
	entry.Lines = []domain.JournalLine{
		{
			Position: 1, Side: side, Account: contact.LedgerAccount,
			Amount: item.Amount, ContactID: &contact.ID,
			Text: item.DocumentNumber,
		},
		{
			Position: 2, Side: side.Opposite(), Account: contra,
			Amount: item.Amount, Text: "Saldenvortrag Personenkonto",
		},
	}
	return entry, nil
}

// newOpeningEntry ist der gemeinsame Kopf jeder Eröffnungsbuchung.
func (s *JournalService) newOpeningEntry(req OpeningBalanceRequest, date, description string) *domain.JournalEntry {
	entry := &domain.JournalEntry{
		FiscalYear:      req.FiscalYear,
		BookingDate:     date,
		DocumentDate:    date,
		ServiceDateFrom: date,
		ServiceDateTo:   date,
		Description:     description,
		Source:          domain.EntrySourceOpening,
		Kind:            domain.EntryKindNormal,
		ReceiptID:       req.ReceiptID,
		Currency:        "EUR",
	}
	if req.LegacySystem != "" {
		entry.LegacyRef = req.LegacySystem
	}
	return entry
}

func contactKindLabel(kind domain.ContactType) string {
	if kind == domain.ContactTypeVendor {
		return "Lieferant"
	}
	return "Kunde"
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
