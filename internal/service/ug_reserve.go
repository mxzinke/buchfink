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

func (s *ClosingService) SetReservePosting(receipts closingReceiptFiler, tx domain.TxRunner) {
	s.reserveReceipts, s.reserveTx = receipts, tx
}

// LegalReserve belongs to the balance sheet of the profit year (§ 5a GmbHG).
// Transfers use a result-appropriation account and never reduce taxable profit.
func (s *ClosingService) LegalReserve(ctx context.Context, year int) (*LegalReserveState, error) {
	state := &LegalReserveState{Year: year}
	if s.settingsRepo == nil {
		return state, nil
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil || !isEntrepreneurialCompany(settings.LegalForm) {
		return state, nil
	}
	turnovers, err := s.journalRepo.AccountTurnovers(ctx, year)
	if err != nil {
		return nil, err
	}
	capital := turnovers[domain.AccountGezeichnetesKapital]
	if capital.Credit-capital.Debit >= 2500000 {
		return state, nil
	}
	state.Applies = true
	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}
	state.NetIncome = netIncomeOf(turnovers, chart)
	loss, profit := turnovers[domain.AccountVerlustvortrag], turnovers[domain.AccountGewinnvortrag]
	state.LossCarryForward = max(domain.Cents(0), loss.Debit-loss.Credit-(profit.Credit-profit.Debit))
	base := max(domain.Cents(0), state.NetIncome-state.LossCarryForward)
	// Round up to a cent so the statutory quarter is never undershot.
	state.Required = base / 4
	if base%4 != 0 {
		state.Required++
	}
	allocation := turnovers[legalReserveExpenseAccount]
	state.Booked = allocation.Debit - allocation.Credit
	state.Difference = state.Required - state.Booked
	period, err := s.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	state.Date = period.EndDate
	return state, nil
}
func (s *ClosingService) EnsureLegalReserve(ctx context.Context, year int) error {
	reserve, err := s.LegalReserve(ctx, year)
	if err != nil {
		return err
	}
	if reserve.Applies && reserve.Difference != 0 {
		return fmt.Errorf("die gesetzliche UG-Rücklage für %d ist noch nicht vollständig im Abschluss gebucht: erforderlich %s €, gebucht %s €. Bitte unter Jahresabschluss die Rücklage bilden bzw. anpassen", year, reserve.Required, reserve.Booked)
	}
	return nil
}
func (s *ClosingService) BookLegalReserve(ctx context.Context, year int) (*domain.JournalEntry, error) {
	if s.reserveTx == nil {
		return nil, fmt.Errorf("Rücklagenbuchung ist nicht eingerichtet")
	}
	var created *domain.JournalEntry
	err := s.reserveTx.RunInTx(ctx, func(ctx context.Context) error {
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
