package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AccountingService answers questions about the journal: account balances, the
// Kontoblatt, the Summen- und Saldenliste and the dashboard figures.
//
// It only reads. Everything that writes goes through JournalService, so the
// booking rules live in exactly one place.
type AccountingService struct {
	serviceClock
	fiscalCalendar
	accountRepo domain.AccountRepository
	journalRepo domain.JournalRepository
	contactRepo domain.ContactRepository
	journalSvc  *JournalService
	// auditRepo protokolliert die Herausgabe von Daten (QUE-02 K2). Freiwillig
	// wie überall: ohne ihn rechnet der Dienst weiter, nur schreibt er nichts.
	auditRepo domain.AuditRepository
}

// SetAuditRepo hängt das Protokoll an. Gebraucht wird es für die Herausgabe der
// gefilterten Journalmenge — ein Lesezugriff auf personenbezogene Daten.
func (s *AccountingService) SetAuditRepo(repo domain.AuditRepository) { s.auditRepo = repo }

// NewAccountingService creates the reporting service.
func NewAccountingService(
	accountRepo domain.AccountRepository,
	journalRepo domain.JournalRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	journalSvc *JournalService,
) *AccountingService {
	return &AccountingService{
		accountRepo:    accountRepo,
		journalRepo:    journalRepo,
		contactRepo:    contactRepo,
		fiscalCalendar: fiscalCalendar{settingsRepo: settingsRepo},
		journalSvc:     journalSvc,
	}
}

// GetAccounts returns the chart of accounts with the turnover booked in the
// fiscal year folded in.
func (s *AccountingService) GetAccounts(ctx context.Context, year int) ([]domain.Account, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	return s.AccountsForYear(ctx, year)
}

// AccountsForYear liefert denselben Kontenplan mit den Salden eines beliebigen
// Geschäftsjahres.
//
// Die Bilanz braucht das für die Vorjahresspalte: § 265 Abs. 2 HGB verlangt zu
// jedem Posten den Betrag des vorhergehenden Geschäftsjahres, und den kann nur
// liefern, wer die Salden eines anderen Jahres beschaffen kann, ohne das aktive
// Jahr umzuschalten.
func (s *AccountingService) AccountsForYear(ctx context.Context, year int) ([]domain.Account, error) {
	return s.AccountsForYearAt(ctx, year, "")
}

// AccountsForYearAt liefert denselben Kontenplan mit den Salden bis zu einem
// Stichtag. Leerer Stichtag heißt: das ganze Geschäftsjahr.
func (s *AccountingService) AccountsForYearAt(ctx context.Context, year int, cutoff string) ([]domain.Account, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	turnovers, err := s.collectedTurnoversAt(ctx, year, cutoff)
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
func (s *AccountingService) collectedTurnovers(ctx context.Context, year int) (map[string]domain.AccountTurnover, error) {
	return s.collectedTurnoversAt(ctx, year, "")
}

// collectedTurnoversAt tut dasselbe bis zu einem Stichtag.
func (s *AccountingService) collectedTurnoversAt(ctx context.Context, year int, cutoff string) (map[string]domain.AccountTurnover, error) {
	raw, err := s.journalRepo.AccountTurnoversUntil(ctx, year, cutoff)
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

// GetAccountLedger builds the Kontoblatt of a fiscal year: every line touching
// the account, with its counter accounts and a running balance.
func (s *AccountingService) GetAccountLedger(ctx context.Context, year int, accountNumber string) (*domain.AccountLedger, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	return s.accountLedger(ctx, year, accountNumber, "", "", false)
}

// GetAccountLedgerRange ist dasselbe Kontoblatt für ein Datumsfenster über alle
// Geschäftsjahre hinweg.
//
// Ein Prüfer fragt nach einem Zeitraum, nicht nach einem Geschäftsjahr: „das
// Bankkonto von Oktober bis März" liegt über einem Jahreswechsel, und ein
// Kontoblatt, das an Silvester aufhört, beantwortet die Frage nicht.
func (s *AccountingService) GetAccountLedgerRange(ctx context.Context, accountNumber, from, to string) (*domain.AccountLedger, error) {
	return s.accountLedger(ctx, 0, accountNumber, from, to, true)
}

func (s *AccountingService) accountLedger(
	ctx context.Context, year int, accountNumber, from, to string, byRange bool,
) (*domain.AccountLedger, error) {
	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}

	acc, ok := chart.Lookup(accountNumber)
	if !ok {
		return nil, fmt.Errorf("Konto %s ist im SKR04 nicht vorhanden", accountNumber)
	}

	var entries []domain.JournalEntry
	if byRange {
		entries, err = s.journalRepo.FindByAccountRange(ctx, accountNumber, from, to)
	} else {
		entries, err = s.journalRepo.FindByAccount(ctx, accountNumber, year)
	}
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
		FiscalYear:     year,
		From:           from,
		To:             to,
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

	// Belegt statt nil: das Kontoblatt läuft je Zeile über die Gegenkonten.
	result := make([]domain.CounterAccount, 0, len(entry.Lines))
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
func (s *AccountingService) GetSuSaOverview(ctx context.Context, year int) (*domain.SuSaOverview, error) {
	return s.GetSuSaOverviewAt(ctx, year, "")
}

// GetSuSaOverviewAt ist dieselbe Liste zu einem Stichtag.
//
// Der Stichtag gehört in die Abfrage und nicht hinter sie: eine Liste, die das
// ganze Jahr summiert und danach Zeilen ausblendet, zeigte Salden, die es zum
// Stichtag nie gab. Ein Prüfer fragt aber nach dem Stand an einem Tag —
// zum 30.06., zum Tag einer Übernahme, zum Ende eines Voranmeldungszeitraums.
func (s *AccountingService) GetSuSaOverviewAt(ctx context.Context, year int, cutoff string) (*domain.SuSaOverview, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	accounts, err := s.AccountsForYearAt(ctx, year, cutoff)
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
		FiscalYear:       year,
		Cutoff:           cutoff,
		TotalDebit:       totalDebit,
		TotalCredit:      totalCredit,
		TotalSaldoDebit:  saldoDebit,
		TotalSaldoCredit: saldoCredit,
		IsBalanced:       difference == 0,
		Difference:       difference,
		Classes:          classes,
	}, nil
}

// GetEntries returns the journal of the active fiscal year, with account names
// resolved for display.
func (s *AccountingService) GetEntries(ctx context.Context, year int) ([]domain.JournalEntry, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return nil, err
	}
	return s.decorate(ctx, entries)
}

// Entry liefert eine Buchung mit Kontonamen.
func (s *AccountingService) Entry(ctx context.Context, id uint) (*domain.JournalEntry, error) {
	entry, err := s.journalRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Buchung %d wurde nicht gefunden: %w", id, err)
	}
	decorated, err := s.decorate(ctx, []domain.JournalEntry{*entry})
	if err != nil || len(decorated) != 1 {
		return entry, err
	}
	return &decorated[0], nil
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
	// Leer statt nil: das leere Journal ist der Zustand jedes neuen Mandanten,
	// und als `null` nähme `entries.map` in der Ansicht den ganzen Baum mit.
	if entries == nil {
		entries = make([]domain.JournalEntry, 0)
	}
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

// GetAvailableFiscalYears lists the years that hold bookings, plus the current
// one.
func (s *AccountingService) GetAvailableFiscalYears(ctx context.Context) []int {
	startMonth := s.fiscalYearStartMonth(ctx)

	years := map[int]bool{
		domain.GetFiscalYearForDate(s.today(), startMonth): true,
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

// FilterEntries beantwortet eine eingeschränkte Frage an das Journal
// (PRF-01 K3).
//
// Gefiltert wird über alle Geschäftsjahre, wenn der Filter keinen Zeitraum
// nennt, und sonst über die Jahre, die der Zeitraum berührt. Der Grund ist die
// Frage selbst: „alle Buchungen auf Konto 6300 über 10.000 €" ist keine Frage
// an ein Geschäftsjahr, sondern an die Buchführung — und eine Antwort, die
// stillschweigend beim 1. Januar aufhört, ist die falsche.
func (s *AccountingService) FilterEntries(
	ctx context.Context, filter accounting.JournalFilter,
) (*accounting.JournalFilterResult, error) {
	entries, err := s.GetAllEntries(ctx)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	if accounts, err := s.accountRepo.FindAll(ctx); err == nil {
		for i := range accounts {
			names[accounts[i].Number] = accounts[i].Name
		}
	}
	result := accounting.FilterJournal(entries, filter, func(number string) string {
		return names[number]
	})
	return &result, nil
}
