package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Bei Uneinbringlichkeit werden Forderungsverlust und Steuerkorrektur im
// Zeitraum ihres Eintritts gebucht (§ 17 Abs. 2 Nr. 1 i. V. m. Abs. 1 UStG).

// WriteOffRequest bucht einen offenen Posten als uneinbringlich aus.
type WriteOffRequest struct {
	OpenItemEntryID uint `json:"openItemEntryId"`
	// Amount ist der auszubuchende Bruttobetrag. Null heißt: der ganze offene
	// Betrag.
	Amount domain.Cents `json:"amount"`
	Date   string       `json:"date"`
	// Reason begründet die Uneinbringlichkeit.
	Reason    string `json:"reason"`
	ReceiptID uint   `json:"receiptId"`
}

// WriteOff bucht eine uneinbringliche Forderung aus.
func (s *settlement) WriteOff(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.writeOff(ctx, req)
	})
}

func (s *settlement) previewWriteOff(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf(
			"zu jeder Ausbuchung gehört eine Begründung: woran die Forderung gescheitert ist, entscheidet, " +
				"ob sie uneinbringlich im Sinne des § 17 Abs. 2 Nr. 1 UStG ist")
	}
	if req.Date == "" {
		return nil, fmt.Errorf("der Tag der Uneinbringlichkeit fehlt")
	}
	if req.ReceiptID == 0 {
		return nil, fmt.Errorf("der Beleg der Uneinbringlichkeit fehlt")
	}

	openItems, err := s.OpenItems(ctx, "")
	if err != nil {
		return nil, err
	}
	var item *domain.OpenItem
	for i := range openItems {
		if openItems[i].EntryID == req.OpenItemEntryID {
			item = &openItems[i]
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("die Buchung %d hat keinen offenen Posten", req.OpenItemEntryID)
	}
	if item.ContactType != domain.ContactTypeCustomer {
		return nil, fmt.Errorf(
			"%s ist eine Verbindlichkeit. Ihr Wegfall ist ein Erlass oder eine Verjährung und wird als "+
				"Ertrag gebucht, nicht als Forderungsverlust", item.DocumentNumber)
	}
	if item.IsCredit() {
		return nil, fmt.Errorf("%s ist eine Gutschrift an den Kunden und keine Forderung", item.DocumentNumber)
	}
	original, err := s.journalRepo.FindByID(ctx, item.EntryID)
	if err != nil {
		return nil, err
	}
	if original.BookingDate > req.Date {
		return nil, fmt.Errorf("die Ausbuchung liegt vor der Forderung")
	}
	amount := req.Amount
	if amount == 0 {
		amount = item.OpenAmount
	}
	if amount <= 0 || amount > item.OpenAmount {
		return nil, fmt.Errorf(
			"%s ist mit %s € offen, ausgebucht werden sollen aber %s €",
			item.DocumentNumber, item.OpenAmount, amount)
	}
	if item.TaxTreatment == "" {
		return nil, fmt.Errorf(
			"der Steuerfall von %s lässt sich nicht bestimmen; ohne ihn lässt sich die Steuerkorrektur nach "+
				"§ 17 Abs. 2 Nr. 1 UStG nicht zutreffend buchen", item.DocumentNumber)
	}

	// Nur beim steuerpflichtigen Inlandsumsatz steckt die Steuer im offenen
	// Betrag; bei einem steuerfreien Umsatz ist der ganze Betrag Entgelt.
	rate := domain.TaxRateNone
	if item.TaxTreatment == domain.TaxTreatmentDomestic {
		rate = item.TaxRate
	}
	net := rate.NetFromGross(amount)

	account, err := accounting.WriteOffAccountFor(rate)
	if err != nil {
		return nil, err
	}

	lines := []domain.JournalLine{{
		Side: domain.SideDebit, Account: account, Amount: net,
		Text: "Forderungsverlust " + item.DocumentNumber,
	}}

	legs, err := s.journalSvc.TaxResolver().Resolve(
		domain.DirectionOutgoing, item.TaxTreatment, item.TaxRate, net)
	if err != nil {
		return nil, fmt.Errorf("die Steuerkorrektur der Ausbuchung ließ sich nicht auflösen: %w", err)
	}
	// Netto und Steuer müssen auch bei Teilausbuchungen den offenen Betrag treffen.
	legs = absorbRounding(legs, amount-net)
	for _, leg := range legs {
		if leg.Amount == 0 {
			continue
		}
		line := taxLegLine(leg)
		// Gegenseite der Rechnung; der Schlüssel bleibt für die Voranmeldung erhalten.
		line.Side = leg.Side.Opposite()
		line.Text = "Steuerkorrektur Forderungsverlust (§ 17 Abs. 2 Nr. 1 UStG)"
		lines = append(lines, line)
	}

	balances, err := specificBalances(ctx, s.journalRepo, req.Date)
	if err != nil {
		return nil, err
	}
	for _, account := range []string{accounting.AccountEWBShortTerm, accounting.AccountEWBLongTerm} {
		later, err := s.journalRepo.FindByAccountRange(ctx, account, req.Date, "")
		if err != nil {
			return nil, err
		}
		for _, entry := range later {
			if entry.BookingDate <= req.Date {
				continue
			}
			for _, line := range entry.Lines {
				if line.AllowanceEntryID == item.EntryID {
					return nil, fmt.Errorf("zur Forderung besteht eine spätere Wertberichtigungsbuchung (%s); zuerst diese berichtigen", entry.EntryNumber)
				}
			}
		}
	}
	remaining := net
	for _, account := range []string{accounting.AccountEWBShortTerm, accounting.AccountEWBLongTerm} {
		used := balances[account][item.EntryID]
		if used < 0 {
			return nil, fmt.Errorf("die Einzelwertberichtigung der Forderung ist negativ")
		}
		if used > remaining && amount != item.OpenAmount {
			used = remaining
		}
		if used == 0 {
			continue
		}
		lines = append(lines, domain.JournalLine{Side: domain.SideDebit, Account: account, Amount: used, AllowanceEntryID: item.EntryID, Text: "Auflösung Einzelwertberichtigung"}, domain.JournalLine{Side: domain.SideCredit, Account: accounting.AccountEWBIncome, Amount: used, Text: "Auflösung Einzelwertberichtigung"})
		remaining -= used
	}
	contactID := item.ContactID
	lines = append(lines, domain.JournalLine{
		Side:      item.SettleSide(),
		Account:   item.LedgerAccount,
		ContactID: &contactID,
		Amount:    amount,
		Text:      item.DocumentNumber,
	})

	entry := &domain.JournalEntry{
		ReceiptID:       &req.ReceiptID,
		BookingDate:     req.Date,
		DocumentDate:    req.Date,
		ServiceDateFrom: req.Date,
		ServiceDateTo:   req.Date,
		Description: fmt.Sprintf("Ausbuchung %s (%s): %s",
			item.DocumentNumber, item.ContactName, req.Reason),
		// Der Ausgleich erzeugt keinen eigenen offenen Posten.
		Source:             domain.EntrySourcePayment,
		DocumentNumber:     item.DocumentNumber,
		TaxTreatment:       item.TaxTreatment,
		ContactID:          &contactID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	if err := s.journalSvc.ValidatePostable(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *settlement) writeOff(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	preview, err := s.previewWriteOff(ctx, req)
	if err != nil {
		return nil, err
	}
	entry, err := s.journalSvc.Post(ctx, preview)
	if err != nil {
		return nil, err
	}
	receipt, err := s.journalSvc.receiptRepo.FindByID(ctx, req.ReceiptID)
	if err != nil {
		return nil, err
	}
	if receipt.Status != domain.ReceiptStatusSealed {
		if err := s.journalSvc.receiptRepo.Seal(ctx, req.ReceiptID, entry.ID); err != nil {
			return nil, err
		}
	}
	amount := entry.Lines[len(entry.Lines)-1].Amount

	if err := s.allocationRepo.Create(ctx, []domain.PaymentAllocation{{
		OpenItemEntryID:  req.OpenItemEntryID,
		PaymentEntryID:   entry.ID,
		ContactID:        *entry.ContactID,
		SettledAmount:    amount,
		CashAmount:       0,
		DifferenceKind:   domain.DifferenceWriteoff,
		DifferenceAmount: amount,
	}}); err != nil {
		return nil, fmt.Errorf("die Ausbuchung ließ sich dem Posten nicht zuordnen: %w", err)
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "OPEN_ITEM",
		fmt.Sprintf("%d", req.OpenItemEntryID), fmt.Sprintf(
			"Offener Posten %s über %s € als uneinbringlich ausgebucht (%s): %s",
			entry.DocumentNumber, amount, entry.EntryNumber, req.Reason)); err != nil {
		return nil, err
	}
	return entry, nil
}

// absorbRounding korrigiert eine einzelne Steuerzeile auf Brutto minus Netto.
// Zwei gegenläufige Steuerzeilen müssen sich unverändert saldieren.
func absorbRounding(legs []domain.TaxLeg, wanted domain.Cents) []domain.TaxLeg {
	index, count := -1, 0
	for i := range legs {
		if legs[i].Amount != 0 {
			index, count = i, count+1
		}
	}
	if count != 1 {
		return legs
	}
	legs[index].Amount = wanted
	return legs
}
