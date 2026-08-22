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

// AccountingService answers questions about the journal: account balances, the
// Kontoblatt, the Summen- und Saldenliste and the dashboard figures.
//
// It only reads. Everything that writes goes through JournalService, so the
// booking rules live in exactly one place.
type AccountingService struct {
	accountRepo  domain.AccountRepository
	journalRepo  domain.JournalRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	journalSvc   *JournalService
	fiscalYear   int
}

// NewAccountingService creates the reporting service.
func NewAccountingService(
	accountRepo domain.AccountRepository,
	journalRepo domain.JournalRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	journalSvc *JournalService,
	fiscalYear int,
) *AccountingService {
	return &AccountingService{
		accountRepo:  accountRepo,
		journalRepo:  journalRepo,
		contactRepo:  contactRepo,
		settingsRepo: settingsRepo,
		journalSvc:   journalSvc,
		fiscalYear:   fiscalYear,
	}
}

// SetFiscalYear updates the active fiscal year.
func (s *AccountingService) SetFiscalYear(year int) {
	s.fiscalYear = year
	if s.journalSvc != nil {
		s.journalSvc.SetFiscalYear(year)
	}
}

// GetFiscalYear returns the active fiscal year.
func (s *AccountingService) GetFiscalYear() int { return s.fiscalYear }

// GetAccounts returns the chart of accounts with the turnover booked in the
// active fiscal year folded in.
func (s *AccountingService) GetAccounts(ctx context.Context) ([]domain.Account, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	turnovers, err := s.collectedTurnovers(ctx)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		acc := &accounts[i]
		var t domain.AccountTurnover
		for number, booked := range turnovers {
			if accounting.Covers(*acc, number) {
				t.Debit += booked.Debit
				t.Credit += booked.Credit
				t.Count += booked.Count
				t.Aggregated += booked.Aggregated
			}
		}
		applyTurnover(acc, t)
	}

	return accounts, nil
}

// collectedTurnovers folds the Personenkonten into their Sammelkonto.
//
// Open items are booked on the partner's own account (10000-69999 / 70000-99999),
// but those accounts are not part of the chart of accounts. Without folding them
// into 1200 and 3300 they would be missing from the balance sheet and from the
// Summen- und Saldenliste — which would then no longer add up.
func (s *AccountingService) collectedTurnovers(ctx context.Context) (map[string]domain.AccountTurnover, error) {
	raw, err := s.journalRepo.AccountTurnovers(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}

	collected := make(map[string]domain.AccountTurnover, len(raw))
	for number, t := range raw {
		target := number
		if kind, ok := domain.LedgerAccountKind(number); ok {
			if kind == domain.ContactTypeCustomer {
				target = domain.AccountForderungenLuL
			} else {
				target = domain.AccountVerbindlichkeitenLuL
			}
		}
		existing := collected[target]
		existing.Debit += t.Debit
		existing.Credit += t.Credit
		existing.Count += t.Count
		if target != number {
			existing.Aggregated++
		}
		collected[target] = existing
	}
	return collected, nil
}

// applyTurnover sets the sums and the balance in the account's natural
// direction: Aktiva and Aufwand carry a debit balance, Passiva, Eigenkapital and
// Ertrag a credit balance.
func applyTurnover(acc *domain.Account, t domain.AccountTurnover) {
	acc.DebitSum = t.Debit
	acc.CreditSum = t.Credit
	acc.BookingsCount = t.Count
	acc.AggregatedAccounts = t.Aggregated

	switch acc.Type {
	case domain.AccountTypeLiability, domain.AccountTypeEquity, domain.AccountTypeRevenue:
		acc.Balance = t.Credit - t.Debit
	default:
		acc.Balance = t.Debit - t.Credit
	}
}

// GetAccountByNumber returns one account including its turnover.
func (s *AccountingService) GetAccountByNumber(ctx context.Context, number string) (*domain.Account, error) {
	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}
	acc, ok := chart.Lookup(number)
	if !ok {
		return nil, fmt.Errorf("Konto %s ist im SKR04 nicht vorhanden", number)
	}

	turnovers, err := s.collectedTurnovers(ctx)
	if err != nil {
		return nil, err
	}
	applyTurnover(&acc, turnovers[number])
	return &acc, nil
}

// GetAccountLedger builds the Kontoblatt: every line touching the account, with
// its counter accounts and a running balance.
func (s *AccountingService) GetAccountLedger(ctx context.Context, accountNumber string) (*domain.AccountLedger, error) {
	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}

	acc, ok := chart.Lookup(accountNumber)
	if !ok {
		return nil, fmt.Errorf("Konto %s ist im SKR04 nicht vorhanden", accountNumber)
	}

	entries, err := s.journalRepo.FindByAccount(ctx, accountNumber, s.fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("Buchungen für Konto %s konnten nicht geladen werden: %w", accountNumber, err)
	}

	debitPositive := acc.Type != domain.AccountTypeLiability &&
		acc.Type != domain.AccountTypeEquity &&
		acc.Type != domain.AccountTypeRevenue

	rows := make([]domain.AccountLedgerRow, 0, len(entries))
	var running, totalDebit, totalCredit domain.Cents

	for i := range entries {
		entry := &entries[i]
		for _, line := range entry.Lines {
			if line.Account != accountNumber {
				continue
			}

			var debit, credit domain.Cents
			if line.Side == domain.SideDebit {
				debit = line.Amount
				totalDebit += debit
			} else {
				credit = line.Amount
				totalCredit += credit
			}

			if debitPositive {
				running += debit - credit
			} else {
				running += credit - debit
			}

			rows = append(rows, domain.AccountLedgerRow{
				EntryID:         entry.ID,
				EntryNumber:     entry.EntryNumber,
				BookingDate:     entry.BookingDate,
				DocumentDate:    entry.DocumentDate,
				DocumentNumber:  entry.DocumentNumber,
				Description:     entry.Description,
				Kind:            entry.Kind,
				Side:            line.Side,
				DebitAmount:     debit,
				CreditAmount:    credit,
				RunningBalance:  running,
				CounterAccounts: counterAccounts(entry, accountNumber, chart),
				TaxKey:          line.TaxKey,
			})
		}
	}

	acc.DebitSum = totalDebit
	acc.CreditSum = totalCredit
	acc.Balance = running
	acc.BookingsCount = len(rows)

	return &domain.AccountLedger{
		Account:        acc,
		FiscalYear:     s.fiscalYear,
		OpeningBalance: 0,
		TotalDebit:     totalDebit,
		TotalCredit:    totalCredit,
		ClosingBalance: running,
		RowCount:       len(rows),
		Rows:           rows,
	}, nil
}

// counterAccounts lists the accounts on the opposite side of an entry. With
// multi-line bookings there can be several, which is why the Kontoblatt shows a
// list rather than the single "Gegenkonto" a two-account model implied.
func counterAccounts(entry *domain.JournalEntry, account string, chart *accounting.Chart) []domain.CounterAccount {
	var own domain.Side
	for _, l := range entry.Lines {
		if l.Account == account {
			own = l.Side
			break
		}
	}

	var result []domain.CounterAccount
	for _, l := range entry.Lines {
		if l.Account == account || l.Side == own {
			continue
		}
		result = append(result, domain.CounterAccount{
			Account: l.Account,
			Name:    chart.Name(l.Account),
			Amount:  l.Amount,
		})
	}
	return result
}

// GetSuSaOverview builds the Summen- und Saldenliste grouped by Kontenklasse.
func (s *AccountingService) GetSuSaOverview(ctx context.Context) (*domain.SuSaOverview, error) {
	accounts, err := s.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	classNames := map[int]string{
		0: "Klasse 0: Anlagevermögenskonten",
		1: "Klasse 1: Umlaufvermögenskonten",
		2: "Klasse 2: Eigenkapital- & Fremdkapitalkonten",
		3: "Klasse 3: Fremdkapitalkonten",
		4: "Klasse 4: Betriebliche Erträge",
		5: "Klasse 5: Betriebliche Aufwendungen (Material / Fremdleistungen)",
		6: "Klasse 6: Betriebliche Aufwendungen (Personal / AfA / Sonstige)",
		7: "Klasse 7: Weitere Erträge & Aufwendungen (Finanzen / Steuern)",
		8: "Klasse 8: Künftige Verwendung durch DATEV",
		9: "Klasse 9: Vortrags-, Kapital- & statistische Konten",
	}

	classes := make([]domain.SuSaClassSummary, 10)
	for i := range classes {
		classes[i] = domain.SuSaClassSummary{
			Kontenklasse:     i,
			KontenklasseName: classNames[i],
			Accounts:         []domain.Account{},
		}
	}

	var totalDebit, totalCredit, saldoDebit, saldoCredit domain.Cents

	for _, a := range accounts {
		// Accounts without movement would bury the list; the chart has 1.855 of
		// them and at most a few dozen carry bookings.
		if a.BookingsCount == 0 {
			continue
		}

		kk := a.Kontenklasse
		if kk < 0 || kk > 9 {
			kk = 0
		}
		cls := &classes[kk]
		cls.Accounts = append(cls.Accounts, a)
		cls.TotalDebit += a.DebitSum
		cls.TotalCredit += a.CreditSum
		cls.AccountsCount++

		var sd, sc domain.Cents
		switch {
		case a.DebitSum > a.CreditSum:
			sd = a.DebitSum - a.CreditSum
		case a.CreditSum > a.DebitSum:
			sc = a.CreditSum - a.DebitSum
		}
		cls.TotalSaldoDebit += sd
		cls.TotalSaldoCredit += sc

		totalDebit += a.DebitSum
		totalCredit += a.CreditSum
		saldoDebit += sd
		saldoCredit += sc
	}

	// With integer cents this is an exact equality, not a tolerance check. If it
	// ever fails, an unbalanced entry reached the journal.
	difference := totalDebit - totalCredit

	return &domain.SuSaOverview{
		FiscalYear:       s.fiscalYear,
		TotalDebit:       totalDebit,
		TotalCredit:      totalCredit,
		TotalSaldoDebit:  saldoDebit,
		TotalSaldoCredit: saldoCredit,
		IsBalanced:       difference == 0,
		Difference:       difference,
		Classes:          classes,
	}, nil
}

// GetSKR04Catalog returns the complete static SKR04 2026 catalog.
func (s *AccountingService) GetSKR04Catalog(ctx context.Context) (*accounting.SKR04Catalog, error) {
	return accounting.GetSKR04Catalog()
}

// GetEntries returns the journal of the active fiscal year, with account names
// resolved for display.
func (s *AccountingService) GetEntries(ctx context.Context) ([]domain.JournalEntry, error) {
	entries, err := s.journalRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}
	return s.decorate(ctx, entries)
}

// GetAllEntries returns the journal across all fiscal years.
func (s *AccountingService) GetAllEntries(ctx context.Context) ([]domain.JournalEntry, error) {
	entries, err := s.journalRepo.FindAll(ctx, 0)
	if err != nil {
		return nil, err
	}
	return s.decorate(ctx, entries)
}

func (s *AccountingService) decorate(ctx context.Context, entries []domain.JournalEntry) ([]domain.JournalEntry, error) {
	chart, err := s.chart(ctx)
	if err != nil {
		return entries, nil
	}

	ledgerNames := map[string]string{}
	if s.contactRepo != nil {
		if contacts, err := s.contactRepo.FindAll(ctx); err == nil {
			for _, c := range contacts {
				ledgerNames[c.LedgerAccount] = c.Name
			}
		}
	}

	for i := range entries {
		for j := range entries[i].Lines {
			line := &entries[i].Lines[j]
			if name, ok := ledgerNames[line.Account]; ok {
				line.AccountName = name
				continue
			}
			line.AccountName = chart.Name(line.Account)
		}
	}
	return entries, nil
}

// GetFinancialSummary aggregates the dashboard KPIs.
func (s *AccountingService) GetFinancialSummary(ctx context.Context) (*domain.FinancialSummary, error) {
	accounts, err := s.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	var revenue, expenses domain.Cents
	for _, a := range accounts {
		switch a.Type {
		case domain.AccountTypeRevenue:
			revenue += a.Balance
		case domain.AccountTypeExpense:
			expenses += a.Balance
		}
	}

	turnovers, err := s.collectedTurnovers(ctx)
	if err != nil {
		return nil, err
	}

	var bank domain.Cents
	for _, acc := range domain.LiquidAccounts() {
		t := turnovers[acc]
		bank += t.Debit - t.Credit
	}

	receivable := turnovers[domain.AccountForderungenLuL]
	payable := turnovers[domain.AccountVerbindlichkeitenLuL]
	receivables := receivable.Debit - receivable.Credit
	payables := payable.Credit - payable.Debit

	cashflow, err := s.journalRepo.MonthlyCashflow(ctx, s.fiscalYear, domain.LiquidAccounts())
	if err != nil {
		return nil, err
	}

	return &domain.FinancialSummary{
		TotalRevenue:    revenue,
		TotalExpenses:   expenses,
		NetIncome:       revenue - expenses,
		BankBalance:     bank,
		OpenReceivables: receivables,
		OpenPayables:    payables,
		CashflowHistory: cashflow,
	}, nil
}

// GetAvailableFiscalYears lists the years that hold bookings, plus the current
// and the active one.
func (s *AccountingService) GetAvailableFiscalYears(ctx context.Context) []int {
	startMonth := 1
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil && cfg.FiscalYearStartMonth > 0 {
			startMonth = cfg.FiscalYearStartMonth
		}
	}

	years := map[int]bool{
		domain.GetFiscalYearForDate(time.Now().Format("2006-01-02"), startMonth): true,
	}
	if s.fiscalYear > 0 {
		years[s.fiscalYear] = true
	}
	if dbYears, err := s.journalRepo.GetAvailableFiscalYears(ctx); err == nil {
		for _, y := range dbYears {
			if y > 0 {
				years[y] = true
			}
		}
	}

	result := make([]int, 0, len(years))
	for y := range years {
		result = append(result, y)
	}
	sort.Ints(result)
	return result
}

func (s *AccountingService) chart(ctx context.Context) (*accounting.Chart, error) {
	if s.journalSvc != nil {
		return s.journalSvc.Chart(ctx)
	}
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return accounting.NewChart(accounts), nil
}
