package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

const legalReserveExpenseAccount = "7765"

type LegalReserveState struct {
	Applies          bool         `json:"applies"`
	Year             int          `json:"year"`
	Date             string       `json:"date"`
	NetIncome        domain.Cents `json:"netIncome"`
	LossCarryForward domain.Cents `json:"lossCarryForward"`
	Required         domain.Cents `json:"required"`
	Booked           domain.Cents `json:"booked"`
	Difference       domain.Cents `json:"difference"`
}

func (s *ClosingService) SetReservePosting(receipts closingReceiptFiler) {
	s.reserveReceipts = receipts
}

func (s *ClosingService) LegalReserve(ctx context.Context, year int) (*LegalReserveState, error) {
	return s.checks.LegalReserve(ctx, year)
}
func (s *ClosingService) EnsureLegalReserve(ctx context.Context, year int) error {
	return s.checks.EnsureLegalReserve(ctx, year)
}
func (s *ClosingService) BookLegalReserve(ctx context.Context, year int) (*domain.JournalEntry, error) {
	if s.reserveReceipts == nil {
		return nil, fmt.Errorf("Rücklagenbuchung ist nicht eingerichtet")
	}
	var created *domain.JournalEntry
	err := s.journalSvc.InTx(ctx, func(ctx context.Context) error {
		state, err := s.LegalReserve(ctx, year)
		if err != nil {
			return err
		}
		if !state.Applies || state.Difference == 0 {
			return fmt.Errorf("für %d ist keine weitere gesetzliche Rücklage zu buchen", year)
		}
		amount, debit, credit := state.Difference, legalReserveExpenseAccount, domain.AccountGesetzlicheRuecklage
		if amount < 0 {
			amount, debit, credit = -amount, credit, debit
		}
		lines := []domain.JournalLine{{Side: domain.SideDebit, Account: debit, Amount: amount}, {Side: domain.SideCredit, Account: credit, Amount: amount}}
		description := fmt.Sprintf("Gesetzliche UG-Rücklage %d gemäß § 5a Abs. 3 GmbHG", year)
		entry := &domain.JournalEntry{FiscalYear: year, BookingDate: state.Date, DocumentDate: state.Date, ServiceDateFrom: state.Date, ServiceDateTo: state.Date, Source: domain.EntrySourceClosing, Description: description, Lines: lines}
		if err := s.journalSvc.ValidatePostable(ctx, entry); err != nil {
			return err
		}
		voucher, err := selfIssuedVoucher(ctx, s.reserveReceipts, year, closingVoucher{Kind: "ug_ruecklage", FiscalYear: year, Date: state.Date, Description: description, Explanation: "Ein Viertel des um den Verlustvortrag geminderten Jahresüberschusses, auf volle Cent aufgerundet. Bereits gebuchte Zuführungen sind berücksichtigt.", Calculation: state, Lines: lines})
		if err != nil {
			return err
		}
		attachVoucher(entry, voucher)
		created, err = postWithVoucher(ctx, s.journalSvc, s.reserveReceipts, entry, voucher)
		return err
	})
	return created, err
}
