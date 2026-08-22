// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AllocationRequest assigns part of a payment to one open item.
type AllocationRequest struct {
	OpenItemEntryID uint `json:"openItemEntryId"`
	// SettledAmount is the amount the open item is reduced by. With a Skonto
	// that is the full remaining amount even though less cash moves.
	SettledAmount    domain.Cents          `json:"settledAmount"`
	DifferenceKind   domain.DifferenceKind `json:"differenceKind"`
	DifferenceAmount domain.Cents          `json:"differenceAmount"`
}

// cashAmount is what actually moves on the liquid account for this allocation.
// A Skonto or a rounding difference reduces it, a bank fee increases it.
func (a AllocationRequest) cashAmount() domain.Cents {
	switch a.DifferenceKind {
	case domain.DifferenceSkonto, domain.DifferenceRounding, domain.DifferenceCurrency:
		return a.SettledAmount - a.DifferenceAmount
	case domain.DifferenceBankFee:
		return a.SettledAmount + a.DifferenceAmount
	default:
		return a.SettledAmount
	}
}

// PaymentRequest settles one or more open items from a single payment.
type PaymentRequest struct {
	BankTxID       *uint               `json:"bankTxId,omitempty"`
	PaymentAccount string              `json:"paymentAccount"`
	PaymentDate    string              `json:"paymentDate"`
	ValueDate      string              `json:"valueDate,omitempty"`
	Description    string              `json:"description,omitempty"`
	Allocations    []AllocationRequest `json:"allocations"`
}

// PaymentService settles open items and books the differences.
type PaymentService struct {
	journalSvc     *JournalService
	journalRepo    domain.JournalRepository
	allocationRepo domain.PaymentAllocationRepository
	contactRepo    domain.ContactRepository
	bankRepo       domain.BankRepository
	fiscalYear     int
}

// NewPaymentService creates the payment matching service.
func NewPaymentService(
	journalSvc *JournalService,
	journalRepo domain.JournalRepository,
	allocationRepo domain.PaymentAllocationRepository,
	contactRepo domain.ContactRepository,
	bankRepo domain.BankRepository,
	fiscalYear int,
) *PaymentService {
	return &PaymentService{
		journalSvc:     journalSvc,
		journalRepo:    journalRepo,
		allocationRepo: allocationRepo,
		contactRepo:    contactRepo,
		bankRepo:       bankRepo,
		fiscalYear:     fiscalYear,
	}
}

// SetFiscalYear updates the year the open items are computed for.
func (s *PaymentService) SetFiscalYear(year int) { s.fiscalYear = year }

// OpenItems lists the unsettled receivables and payables.
func (s *PaymentService) OpenItems(ctx context.Context) ([]domain.OpenItem, error) {
	entries, err := s.journalRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}
	settled, err := s.allocationRepo.SettledByOpenItem(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}

	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]domain.Contact, len(contacts))
	for _, c := range contacts {
		byID[c.ID] = c
	}

	// A Generalumkehr cancels the open item of the entry it reverses.
	reversed := map[uint]bool{}
	for i := range entries {
		if entries[i].ReversalOfID != nil {
			reversed[*entries[i].ReversalOfID] = true
		}
	}

	var items []domain.OpenItem
	for i := range entries {
		entry := &entries[i]
		if entry.Kind == domain.EntryKindReversal || reversed[entry.ID] {
			continue
		}
		// Payments themselves do not create open items.
		if entry.Source == domain.EntrySourcePayment {
			continue
		}

		line, ok := ledgerLine(entry)
		if !ok || entry.ContactID == nil {
			continue
		}
		contact, ok := byID[*entry.ContactID]
		if !ok {
			continue
		}

		gross := line.Amount
		alreadySettled := settled[entry.ID]
		open := gross - alreadySettled
		if open == 0 {
			continue
		}

		items = append(items, domain.OpenItem{
			EntryID:        entry.ID,
			EntryNumber:    entry.EntryNumber,
			ContactID:      contact.ID,
			ContactName:    contact.Name,
			ContactType:    contact.Type,
			LedgerAccount:  line.Account,
			DocumentNumber: entry.DocumentNumber,
			DocumentDate:   entry.DocumentDate,
			DueDate:        dueDate(entry, contact),
			GrossAmount:    gross,
			SettledAmount:  alreadySettled,
			OpenAmount:     open,
			TaxRate:        documentTaxRate(entry),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DueDate != items[j].DueDate {
			return items[i].DueDate < items[j].DueDate
		}
		return items[i].EntryNumber < items[j].EntryNumber
	})
	return items, nil
}

// Settle books a payment against one or more open items.
func (s *PaymentService) Settle(ctx context.Context, req PaymentRequest) (*domain.JournalEntry, error) {
	if len(req.Allocations) == 0 {
		return nil, fmt.Errorf("es wurde kein offener Posten ausgewählt")
	}
	if req.PaymentDate == "" {
		req.PaymentDate = time.Now().Format("2006-01-02")
	}

	// A bank payment takes its account and amount from the statement, so neither
	// can be mistyped.
	var bankTx *domain.BankTransaction
	if req.BankTxID != nil {
		tx, err := s.bankRepo.FindByID(ctx, *req.BankTxID)
		if err != nil {
			return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", *req.BankTxID, err)
		}
		if tx.MatchStatus == domain.MatchStatusMatched {
			return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet", tx.ID)
		}
		bankTx = tx
		req.PaymentAccount = tx.LedgerAccount
		req.PaymentDate = tx.BookingDate
		req.ValueDate = tx.ValueDate
	}
	if req.PaymentAccount == "" {
		return nil, fmt.Errorf("Zahlungsmittel fehlt (Kasse, Bank oder Kreditkarte)")
	}

	openItems, err := s.OpenItems(ctx)
	if err != nil {
		return nil, err
	}
	byEntry := make(map[uint]domain.OpenItem, len(openItems))
	for _, o := range openItems {
		byEntry[o.EntryID] = o
	}

	var lines []domain.JournalLine
	var totalCash domain.Cents
	var contactID *uint
	allocations := make([]domain.PaymentAllocation, 0, len(req.Allocations))

	for i, alloc := range req.Allocations {
		item, ok := byEntry[alloc.OpenItemEntryID]
		if !ok {
			return nil, fmt.Errorf("Zuordnung %d: die Buchung %d hat keinen offenen Posten", i+1, alloc.OpenItemEntryID)
		}
		if alloc.SettledAmount <= 0 {
			return nil, fmt.Errorf("Zuordnung %d: der Ausgleichsbetrag muss größer als null sein", i+1)
		}
		if alloc.SettledAmount > item.OpenAmount {
			return nil, fmt.Errorf(
				"Zuordnung %d: %s ist mit %s € offen, zugeordnet wurden aber %s €",
				i+1, item.DocumentNumber, item.OpenAmount, alloc.SettledAmount)
		}
		if alloc.DifferenceKind == "" {
			alloc.DifferenceKind = domain.DifferenceNone
		}
		if alloc.DifferenceKind != domain.DifferenceNone && alloc.DifferenceAmount <= 0 {
			return nil, fmt.Errorf("Zuordnung %d: für die Differenzart %q fehlt der Betrag", i+1, alloc.DifferenceKind)
		}

		// Settling a payable is a debit on the Kreditorenkonto, settling a
		// receivable a credit on the Debitorenkonto.
		settleSide := domain.SideDebit
		direction := domain.DirectionIncoming
		if item.ContactType == domain.ContactTypeCustomer {
			settleSide = domain.SideCredit
			direction = domain.DirectionOutgoing
		}

		cid := item.ContactID
		lines = append(lines, domain.JournalLine{
			Side:      settleSide,
			Account:   item.LedgerAccount,
			ContactID: &cid,
			Amount:    alloc.SettledAmount,
			Text:      item.DocumentNumber,
		})
		if contactID == nil {
			contactID = &cid
		}

		diffLines, err := s.differenceLines(alloc, item, direction, settleSide)
		if err != nil {
			return nil, fmt.Errorf("Zuordnung %d: %w", i+1, err)
		}
		lines = append(lines, diffLines...)

		cash := alloc.cashAmount()
		if cash < 0 {
			return nil, fmt.Errorf("Zuordnung %d: die Differenz ist größer als der Ausgleichsbetrag", i+1)
		}
		totalCash += cash

		allocations = append(allocations, domain.PaymentAllocation{
			OpenItemEntryID:  item.EntryID,
			BankTxID:         req.BankTxID,
			ContactID:        item.ContactID,
			SettledAmount:    alloc.SettledAmount,
			CashAmount:       cash,
			DifferenceKind:   alloc.DifferenceKind,
			DifferenceAmount: alloc.DifferenceAmount,
		})
	}

	// The cash side is what the statement says. Checking it here is what turns a
	// mis-entered Skonto into an error message instead of a silent wrong booking.
	if bankTx != nil && totalCash != bankTx.Amount.Abs() {
		return nil, fmt.Errorf(
			"die Zuordnung ergibt %s €, der Bankumsatz lautet aber über %s €",
			totalCash, bankTx.Amount.Abs())
	}
	if totalCash == 0 {
		return nil, fmt.Errorf("die Zahlung hat einen Betrag von null")
	}

	// Money leaving is a credit on the liquid account, money arriving a debit.
	cashSide := domain.SideCredit
	if lines[0].Side == domain.SideCredit {
		cashSide = domain.SideDebit
	}
	lines = append(lines, domain.JournalLine{Side: cashSide, Account: req.PaymentAccount, Amount: totalCash})

	description := req.Description
	if description == "" {
		description = paymentDescription(req, byEntry)
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.PaymentDate,
		DocumentDate:       req.PaymentDate,
		ServiceDateFrom:    req.PaymentDate,
		ServiceDateTo:      req.PaymentDate,
		ValueDate:          req.ValueDate,
		Description:        description,
		Source:             domain.EntrySourcePayment,
		ContactID:          contactID,
		BankTxID:           req.BankTxID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}

	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	for i := range allocations {
		allocations[i].PaymentEntryID = created.ID
	}
	if err := s.allocationRepo.Create(ctx, allocations); err != nil {
		return nil, fmt.Errorf("die Zahlungszuordnung konnte nicht gespeichert werden: %w", err)
	}

	if bankTx != nil {
		if err := s.bankRepo.SetMatchStatus(ctx, bankTx.ID, domain.MatchStatusMatched); err != nil {
			return nil, fmt.Errorf("der Bankumsatz konnte nicht als zugeordnet markiert werden: %w", err)
		}
	}

	return created, nil
}

// differenceLines books the difference between the settled amount and the cash
// that moved.
func (s *PaymentService) differenceLines(
	alloc AllocationRequest,
	item domain.OpenItem,
	direction domain.Direction,
	settleSide domain.Side,
) ([]domain.JournalLine, error) {
	if alloc.DifferenceKind == domain.DifferenceNone {
		return nil, nil
	}

	// The difference sits on the side opposite the settlement, except for a bank
	// fee, which is an additional expense on the same side as the settlement.
	opposite := domain.SideCredit
	if settleSide == domain.SideCredit {
		opposite = domain.SideDebit
	}

	switch alloc.DifferenceKind {
	case domain.DifferenceSkonto:
		// § 17 UStG: a Skonto reduces the taxable amount, so the VAT booked on
		// the original document has to be corrected along with it. Booking only
		// the net part would leave the VAT return overstated.
		account, err := domain.SkontoAccount(direction, item.TaxRate)
		if err != nil {
			return nil, err
		}
		net := item.TaxRate.NetFromGross(alloc.DifferenceAmount)
		tax := alloc.DifferenceAmount - net

		lines := []domain.JournalLine{
			{Side: opposite, Account: account, Amount: net, Text: "Skonto"},
		}
		if tax != 0 {
			taxAccount := domain.AccountVorsteuer19
			key := "SKONTO_VST"
			if direction == domain.DirectionOutgoing {
				taxAccount = domain.AccountUmsatzsteuer19
				key = "SKONTO_UST"
			}
			if item.TaxRate == domain.TaxRateReduced {
				taxAccount = domain.AccountVorsteuer7
				if direction == domain.DirectionOutgoing {
					taxAccount = domain.AccountUmsatzsteuer7
				}
			}
			lines = append(lines, domain.JournalLine{
				Side: opposite, Account: taxAccount, Amount: tax,
				TaxKey: key, TaxBase: net, Text: "Steuerkorrektur Skonto (§ 17 UStG)",
			})
		}
		return lines, nil

	case domain.DifferenceBankFee:
		return []domain.JournalLine{{
			Side: settleSide, Account: domain.AccountNebenkostenGeld,
			Amount: alloc.DifferenceAmount, Text: "Bankgebühr",
		}}, nil

	case domain.DifferenceRounding, domain.DifferenceCurrency:
		text := "Rundungsdifferenz"
		if alloc.DifferenceKind == domain.DifferenceCurrency {
			text = "Kursdifferenz"
		}
		// On the credit side the difference is income, on the debit side expense.
		account := "6300" // Sonstige betriebliche Aufwendungen
		if opposite == domain.SideCredit {
			account = "4830" // Sonstige betriebliche Erträge
		}
		return []domain.JournalLine{{
			Side: opposite, Account: account, Amount: alloc.DifferenceAmount, Text: text,
		}}, nil
	}

	return nil, fmt.Errorf("unbekannte Differenzart %q", alloc.DifferenceKind)
}

// ledgerLine returns the Personenkonto line of an entry, which is the one that
// carries the open item.
func ledgerLine(entry *domain.JournalEntry) (domain.JournalLine, bool) {
	for _, l := range entry.Lines {
		if domain.IsLedgerAccount(l.Account) {
			return l, true
		}
	}
	return domain.JournalLine{}, false
}

// documentTaxRate derives the VAT rate of a document from its tax lines. It is
// needed to correct the taxable amount when a Skonto is granted later.
func documentTaxRate(entry *domain.JournalEntry) domain.TaxRate {
	for _, l := range entry.Lines {
		if l.TaxKey == "" || l.TaxBase == 0 {
			continue
		}
		// Recover the rate from base and amount rather than parsing the key.
		for _, rate := range []domain.TaxRate{domain.TaxRateStandard, domain.TaxRateReduced} {
			if rate.Tax(l.TaxBase) == l.Amount {
				return rate
			}
		}
	}
	return domain.TaxRateNone
}

func dueDate(entry *domain.JournalEntry, contact domain.Contact) string {
	days := contact.PaymentTermsDays
	if days <= 0 {
		days = 14
	}
	t, err := time.Parse("2006-01-02", entry.DocumentDate)
	if err != nil {
		return entry.DocumentDate
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func paymentDescription(req PaymentRequest, items map[uint]domain.OpenItem) string {
	if len(req.Allocations) == 1 {
		if item, ok := items[req.Allocations[0].OpenItemEntryID]; ok {
			return fmt.Sprintf("Zahlung %s – %s", item.DocumentNumber, item.ContactName)
		}
	}
	return fmt.Sprintf("Sammelzahlung über %d Belege", len(req.Allocations))
}
