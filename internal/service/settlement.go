package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// settlement bündelt offene Posten, Zahlungsausgleich, Bankaufteilung und
// Ausbuchung. Die schreibenden Einstiege schließen Journal, Zuordnungen,
// Anlagenminderung und Protokoll in eine Transaktion ein.
type settlement struct {
	journalSvc *JournalService
	*openItemReader
	bankRepo  domain.BankRepository
	assets    AssetRegister
	auditRepo domain.AuditRepository
	loans     LoanLineSource
}

// Settle bucht eine Zahlung gegen offene Posten.
func (s *settlement) Settle(ctx context.Context, req PaymentRequest) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.settle(ctx, req)
	})
}

func (s *settlement) settle(ctx context.Context, req PaymentRequest) (*domain.JournalEntry, error) {
	if len(req.Allocations) == 0 {
		return nil, fmt.Errorf("es wurde kein offener Posten ausgewählt")
	}
	if req.PaymentDate == "" {
		req.PaymentDate = todayLocal()
	}

	// Bankkonto und Zahlungsdatum kommen aus dem Kontoauszug.
	var bankTx *domain.BankTransaction
	if req.BankTxID != nil {
		tx, err := s.openBankTx(ctx, *req.BankTxID)
		if err != nil {
			return nil, err
		}
		bankTx = tx
		req.PaymentAccount = tx.LedgerAccount
		req.PaymentDate = tx.BookingDate
		req.ValueDate = tx.ValueDate
	}
	if req.PaymentAccount == "" {
		return nil, fmt.Errorf("das Bankkonto der Zahlung fehlt")
	}
	if !isLiquidAccount(req.PaymentAccount) {
		return nil, fmt.Errorf(
			"Konto %s ist kein Zahlungsmittelkonto (Bank 1800 bis 1850 oder Geldtransit 1460)", req.PaymentAccount)
	}

	parts, err := s.allocate(ctx, req.Allocations, req.BankTxID)
	if err != nil {
		return nil, err
	}

	// Auch mit Skonto muss der Zahlbetrag den Bankumsatz genau treffen.
	if bankTx != nil && parts.cash != bankTx.Amount.Abs() {
		return nil, fmt.Errorf(
			"die Zuordnung ergibt %s €, der Bankumsatz lautet aber über %s €",
			parts.cash, bankTx.Amount.Abs())
	}
	if parts.cash == 0 {
		return nil, fmt.Errorf("die Zahlung hat einen Betrag von null")
	}

	lines := append(parts.lines, domain.JournalLine{
		Side: parts.lines[0].Side.Opposite(), Account: req.PaymentAccount, Amount: parts.cash,
	})

	description := req.Description
	if description == "" {
		description = paymentDescription(req, parts.items)
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.PaymentDate,
		DocumentDate:       req.PaymentDate,
		ServiceDateFrom:    req.PaymentDate,
		ServiceDateTo:      req.PaymentDate,
		ValueDate:          req.ValueDate,
		Description:        description,
		Source:             domain.EntrySourcePayment,
		ContactID:          parts.contactID,
		BankTxID:           req.BankTxID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	return s.postSettlement(ctx, entry, parts)
}

// allocationParts sind die Zeilen und Vermerke, die eine Zahlung auf offene
// Posten verteilt: Ausgleich je Posten, Differenzzeilen, Zuordnungen und die
// Minderung von Anschaffungskosten.
type allocationParts struct {
	lines       []domain.JournalLine
	allocations []domain.PaymentAllocation
	reductions  []assetCostReduction
	items       map[uint]domain.OpenItem
	contactID   *uint
	// cash ist, was auf dem Zahlungsmittelkonto fließt.
	cash domain.Cents
}

// openBankTx liest einen Bankumsatz, der noch zu buchen ist.
func (s *settlement) openBankTx(ctx context.Context, id uint) (*domain.BankTransaction, error) {
	tx, err := s.bankRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", id, err)
	}
	if tx.MatchStatus == domain.MatchStatusMatched {
		return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet", tx.ID)
	}
	return tx, nil
}

// allocate verteilt eine Zahlung auf offene Posten.
func (s *settlement) allocate(
	ctx context.Context, requests []AllocationRequest, bankTxID *uint,
) (*allocationParts, error) {
	openItems, err := s.OpenItems(ctx, "")
	if err != nil {
		return nil, err
	}
	parts := &allocationParts{items: make(map[uint]domain.OpenItem, len(openItems))}
	for _, o := range openItems {
		parts.items[o.EntryID] = o
	}
	parts.allocations = make([]domain.PaymentAllocation, 0, len(requests))

	for i, alloc := range requests {
		item, ok := parts.items[alloc.OpenItemEntryID]
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
		// Eine Ausbuchung ohne Geldfluss gehört in WriteOff.
		if alloc.DifferenceKind == domain.DifferenceWriteoff {
			return nil, fmt.Errorf(
				"Zuordnung %d: die Ausbuchung ist keine Zahlung. Eine uneinbringliche Forderung wird "+
					"über „Forderung ausbuchen\" ohne Zahlungsmittel ausgebucht, mit Begründung", i+1)
		}
		if alloc.DifferenceKind != domain.DifferenceNone && alloc.DifferenceAmount <= 0 {
			return nil, fmt.Errorf("Zuordnung %d: für die Differenzart %q fehlt der Betrag", i+1, alloc.DifferenceKind)
		}

		cid := item.ContactID
		parts.lines = append(parts.lines, domain.JournalLine{
			Side:      item.SettleSide(),
			Account:   item.LedgerAccount,
			ContactID: &cid,
			Amount:    alloc.SettledAmount,
			Text:      item.DocumentNumber,
		})
		if parts.contactID == nil {
			parts.contactID = &cid
		}

		diffLines, reduction, err := s.differenceLines(ctx, alloc, item)
		if err != nil {
			return nil, fmt.Errorf("Zuordnung %d: %w", i+1, err)
		}
		parts.lines = append(parts.lines, diffLines...)
		if reduction != nil {
			parts.reductions = append(parts.reductions, *reduction)
		}

		cash := alloc.cashAmount(item.SettleSide())
		if cash < 0 {
			return nil, fmt.Errorf("Zuordnung %d: die Differenz ist größer als der Ausgleichsbetrag", i+1)
		}
		parts.cash += cash

		parts.allocations = append(parts.allocations, domain.PaymentAllocation{
			OpenItemEntryID:  item.EntryID,
			BankTxID:         bankTxID,
			ContactID:        item.ContactID,
			SettledAmount:    alloc.SettledAmount,
			CashAmount:       cash,
			DifferenceKind:   alloc.DifferenceKind,
			DifferenceAmount: alloc.DifferenceAmount,
		})
	}
	return parts, nil
}

// postSettlement überlässt den Bankstatus dem Journal-Effekt. Zuordnungen und
// Anlagenminderungen brauchen die Zuordnungsangaben außerhalb der Buchung.
func (s *settlement) postSettlement(
	ctx context.Context, entry *domain.JournalEntry, parts *allocationParts,
) (*domain.JournalEntry, error) {
	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	if len(parts.allocations) > 0 {
		for i := range parts.allocations {
			parts.allocations[i].PaymentEntryID = created.ID
		}
		if err := s.allocationRepo.Create(ctx, parts.allocations); err != nil {
			return nil, fmt.Errorf("die Zahlungszuordnung konnte nicht gespeichert werden: %w", err)
		}
	}

	// Die Anlagenbewegung verweist auf die Zahlungsbuchung.
	for _, reduction := range parts.reductions {
		entryID := created.ID
		if _, err := s.assets.RecordCostAdjustment(ctx, CostAdjustmentRequest{
			AssetID:        reduction.assetID,
			Date:           entry.BookingDate,
			Amount:         reduction.amount,
			Reduction:      true,
			Note:           reduction.note,
			JournalEntryID: &entryID,
		}); err != nil {
			return nil, fmt.Errorf(
				"die Minderung der Anschaffungskosten ließ sich nicht in das Anlagenverzeichnis "+
					"übernehmen, deshalb wurde die Zahlung nicht gebucht: %w", err)
		}
	}

	return created, nil
}

func paymentDescription(req PaymentRequest, items map[uint]domain.OpenItem) string {
	if len(req.Allocations) == 1 {
		if item, ok := items[req.Allocations[0].OpenItemEntryID]; ok {
			return fmt.Sprintf("Zahlung %s – %s", item.DocumentNumber, item.ContactName)
		}
	}
	return fmt.Sprintf("Sammelzahlung über %d Belege", len(req.Allocations))
}
