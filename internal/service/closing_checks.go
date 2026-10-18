package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// StatementBuilder liefert die Zahlen und Pflichtangaben für die Abschlussprüfung.
type StatementBuilder interface {
	Build(context.Context, int, domain.StatementDepth) (*domain.FinancialStatement, error)
}

// closingChecks teilen die lesenden Prüfungen von Abschluss und Periodenverwaltung.
type closingChecks struct {
	journalRepo     domain.JournalRepository
	accountRepo     domain.AccountRepository
	settingsRepo    domain.SettingsRepository
	journalSvc      *JournalService
	periods         PeriodSource
	statementSource StatementBuilder
}

// LegalReserve belongs to the balance sheet of the profit year (§ 5a GmbHG).
// Transfers use a result-appropriation account and never reduce taxable profit.
func (s *closingChecks) LegalReserve(ctx context.Context, year int) (*LegalReserveState, error) {
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
	period, err := s.periods.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	state.Date = period.EndDate
	return state, nil
}

func (s *closingChecks) EnsureLegalReserve(ctx context.Context, year int) error {
	reserve, err := s.LegalReserve(ctx, year)
	if err != nil {
		return err
	}
	if reserve.Applies && reserve.Difference != 0 {
		return fmt.Errorf("die gesetzliche UG-Rücklage für %d ist noch nicht vollständig im Abschluss gebucht: erforderlich %s €, gebucht %s €. Bitte unter Jahresabschluss die Rücklage bilden bzw. anpassen", year, reserve.Required, reserve.Booked)
	}
	return nil
}

func (s *closingChecks) EnsureStatementReady(ctx context.Context, year int) error {
	if s.statementSource == nil {
		return nil
	}
	fs, err := s.statementSource.Build(ctx, year, domain.DepthFull)
	if err != nil {
		return err
	}
	if len(fs.Header.Missing) > 0 {
		return fmt.Errorf("für den Abschluss fehlen Unternehmensangaben: %s", strings.Join(fs.Header.Missing, ", "))
	}
	if len(fs.Notes.Missing) > 0 {
		return fmt.Errorf("bitte unter Abschlussbausteine → Anhang ergänzen: %s. Wenn nichts anzugeben ist, halten Sie dies ausdrücklich fest", strings.Join(fs.Notes.Missing, ", "))
	}
	return nil
}

func (s *closingChecks) chart(ctx context.Context) (*accounting.Chart, error) {
	if s.journalSvc != nil {
		return s.journalSvc.Chart(ctx)
	}
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Kontenplan konnte nicht geladen werden: %w", err)
	}
	return accounting.NewChart(accounts), nil
}
