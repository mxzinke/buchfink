package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AccountingService handles core double-entry bookkeeping, Kontenübersicht, SuSa and GoBD hash chain generation.
type AccountingService struct {
	accountRepo  domain.AccountRepository
	bookingRepo  domain.BookingRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	fiscalYear   int
}

func NewAccountingService(
	accountRepo domain.AccountRepository,
	bookingRepo domain.BookingRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *AccountingService {
	return &AccountingService{
		accountRepo:  accountRepo,
		bookingRepo:  bookingRepo,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
		fiscalYear:   fiscalYear,
	}
}

// SetFiscalYear updates the active fiscal year filter.
func (s *AccountingService) SetFiscalYear(year int) {
	s.fiscalYear = year
}

// GetFiscalYear returns the active fiscal year.
func (s *AccountingService) GetFiscalYear() int {
	return s.fiscalYear
}

// GetAccounts returns all SKR04 accounts with dynamically calculated balances for the active fiscal year.
func (s *AccountingService) GetAccounts(ctx context.Context) ([]domain.Account, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// Read all bookings for active fiscal year to compute sums in memory fast
	bookings, err := s.bookingRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}

	debitTotals := make(map[string]float64)
	creditTotals := make(map[string]float64)
	counts := make(map[string]int)

	for _, b := range bookings {
		debitTotals[b.DebitAccount] += b.Amount
		creditTotals[b.CreditAccount] += b.Amount
		counts[b.DebitAccount]++
		counts[b.CreditAccount]++
	}

	for i := range accounts {
		acc := &accounts[i]
		var debitSum, creditSum float64
		var count int

		if acc.IsRange && acc.RangeStart != "" && acc.RangeEnd != "" {
			for num, d := range debitTotals {
				if num == acc.Number || (len(num) == 4 && num >= acc.RangeStart && num <= acc.RangeEnd) {
					debitSum += d
				}
			}
			for num, c := range creditTotals {
				if num == acc.Number || (len(num) == 4 && num >= acc.RangeStart && num <= acc.RangeEnd) {
					creditSum += c
				}
			}
			for num, cnt := range counts {
				if num == acc.Number || (len(num) == 4 && num >= acc.RangeStart && num <= acc.RangeEnd) {
					count += cnt
				}
			}
		} else {
			debitSum = debitTotals[acc.Number]
			creditSum = creditTotals[acc.Number]
			count = counts[acc.Number]
		}

		acc.DebitSum = debitSum
		acc.CreditSum = creditSum
		acc.BookingsCount = count

		switch acc.Type {
		case domain.AccountTypeAsset, domain.AccountTypeExpense:
			acc.Balance = debitSum - creditSum
		case domain.AccountTypeLiability, domain.AccountTypeEquity, domain.AccountTypeRevenue:
			acc.Balance = creditSum - debitSum
		case domain.AccountTypeStatistical:
			acc.Balance = debitSum - creditSum
		default:
			acc.Balance = debitSum - creditSum
		}
	}

	return accounts, nil
}

// GetAccountByNumber returns account info by account number.
func (s *AccountingService) GetAccountByNumber(ctx context.Context, number string) (*domain.Account, error) {
	acc, err := s.accountRepo.FindByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	debitSum, creditSum, _ := s.bookingRepo.CalculateAccountSums(ctx, acc.Number, s.fiscalYear)
	acc.DebitSum = debitSum
	acc.CreditSum = creditSum

	switch acc.Type {
	case domain.AccountTypeAsset, domain.AccountTypeExpense:
		acc.Balance = debitSum - creditSum
	case domain.AccountTypeLiability, domain.AccountTypeEquity, domain.AccountTypeRevenue:
		acc.Balance = creditSum - debitSum
	default:
		acc.Balance = debitSum - creditSum
	}

	return acc, nil
}

// GetAccountBookings returns all journal entries referencing the given account.
func (s *AccountingService) GetAccountBookings(ctx context.Context, accountNumber string) ([]domain.BookingEntry, error) {
	return s.bookingRepo.FindByAccount(ctx, accountNumber, s.fiscalYear)
}

// GetAccountLedger generates the detailed ledger (Kontoblatt) for an account with running balances and counterpart accounts.
func (s *AccountingService) GetAccountLedger(ctx context.Context, accountNumber string) (*domain.AccountLedger, error) {
	acc, err := s.accountRepo.FindByNumber(ctx, accountNumber)
	if err != nil {
		return nil, fmt.Errorf("konto %s nicht gefunden: %w", accountNumber, err)
	}

	// Fetch all accounts for counterpart name lookup
	allAccounts, _ := s.accountRepo.FindAll(ctx)
	accNameMap := make(map[string]string, len(allAccounts))
	for _, a := range allAccounts {
		accNameMap[a.Number] = a.Name
	}

	bookings, err := s.bookingRepo.FindByAccount(ctx, acc.Number, s.fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Laden der Buchungen für Konto %s: %w", accountNumber, err)
	}

	entries := make([]domain.AccountLedgerBooking, 0, len(bookings))
	var runningBalance float64
	var totalDebit, totalCredit float64

	isDebitPositive := acc.Type == domain.AccountTypeAsset || acc.Type == domain.AccountTypeExpense || acc.Type == domain.AccountTypeStatistical

	for _, b := range bookings {
		isDebit := b.DebitAccount == acc.Number || (acc.IsRange && len(b.DebitAccount) == 4 && b.DebitAccount >= acc.RangeStart && b.DebitAccount <= acc.RangeEnd)
		isCredit := b.CreditAccount == acc.Number || (acc.IsRange && len(b.CreditAccount) == 4 && b.CreditAccount >= acc.RangeStart && b.CreditAccount <= acc.RangeEnd)

		var dir string
		var debitAmt, creditAmt float64
		var counterAcc, counterName string

		if isDebit && !isCredit {
			dir = "SOLL"
			debitAmt = b.Amount
			totalDebit += debitAmt
			counterAcc = b.CreditAccount
			counterName = accNameMap[counterAcc]
			if isDebitPositive {
				runningBalance += debitAmt
			} else {
				runningBalance -= debitAmt
			}
		} else if isCredit && !isDebit {
			dir = "HABEN"
			creditAmt = b.Amount
			totalCredit += creditAmt
			counterAcc = b.DebitAccount
			counterName = accNameMap[counterAcc]
			if isDebitPositive {
				runningBalance -= creditAmt
			} else {
				runningBalance += creditAmt
			}
		} else if isDebit && isCredit {
			// Booking to self (Umbuchung)
			dir = "SOLL/HABEN"
			debitAmt = b.Amount
			creditAmt = b.Amount
			totalDebit += debitAmt
			totalCredit += creditAmt
			counterAcc = acc.Number
			counterName = acc.Name
		}

		bCopy := b
		bCopy.DebitAccountName = accNameMap[b.DebitAccount]
		bCopy.CreditAccountName = accNameMap[b.CreditAccount]

		entries = append(entries, domain.AccountLedgerBooking{
			Booking:        bCopy,
			CounterAccount: counterAcc,
			CounterName:    counterName,
			Direction:      dir,
			DebitAmount:    debitAmt,
			CreditAmount:   creditAmt,
			RunningBalance: runningBalance,
		})
	}

	accCopy := *acc
	accCopy.DebitSum = totalDebit
	accCopy.CreditSum = totalCredit
	accCopy.Balance = runningBalance
	accCopy.BookingsCount = len(bookings)

	return &domain.AccountLedger{
		Account:        accCopy,
		FiscalYear:     s.fiscalYear,
		OpeningBalance: 0.0,
		TotalDebit:     totalDebit,
		TotalCredit:    totalCredit,
		ClosingBalance: runningBalance,
		BookingsCount:  len(bookings),
		Entries:        entries,
	}, nil
}

// GetSuSaOverview calculates the official Summen- und Saldenliste (Soll-Haben-Übersicht) grouped by Kontenklasse 0-9.
func (s *AccountingService) GetSuSaOverview(ctx context.Context) (*domain.SuSaOverview, error) {
	accounts, err := s.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	// Class containers 0..9
	classNames := map[int]string{
		0: "Klasse 0: Anlagevermögenskonten",
		1: "Klasse 1: Umlaufvermögenskonten",
		2: "Klasse 2: Eigenkapital- & Fremdkapitalkonten",
		3: "Klasse 3: Fremdkapitalkonten (Verbindlichkeiten)",
		4: "Klasse 4: Betriebliche Erträge",
		5: "Klasse 5: Betriebliche Aufwendungen (Material / Fremdleistungen)",
		6: "Klasse 6: Betriebliche Aufwendungen (Personal / AfA / Sonstige)",
		7: "Klasse 7: Weitere Erträge & Aufwendungen (Finanzen / Steuern)",
		8: "Klasse 8: Freie Kontenklasse / Sonderkonten",
		9: "Klasse 9: Vortrags-, Kapital- & statistische Konten",
	}

	classMap := make(map[int]*domain.SuSaClassSummary)
	for i := 0; i <= 9; i++ {
		classMap[i] = &domain.SuSaClassSummary{
			Kontenklasse:     i,
			KontenklasseName: classNames[i],
			Accounts:         make([]domain.Account, 0),
		}
	}

	var grandTotalDebit, grandTotalCredit float64
	var grandSaldoDebit, grandSaldoCredit float64

	for _, a := range accounts {
		kk := a.Kontenklasse
		if kk < 0 || kk > 9 {
			kk = 0
		}

		classSummary := classMap[kk]
		classSummary.Accounts = append(classSummary.Accounts, a)
		classSummary.TotalDebit += a.DebitSum
		classSummary.TotalCredit += a.CreditSum

		var saldoDebit, saldoCredit float64
		if a.DebitSum > a.CreditSum {
			saldoDebit = a.DebitSum - a.CreditSum
		} else if a.CreditSum > a.DebitSum {
			saldoCredit = a.CreditSum - a.DebitSum
		}

		classSummary.TotalSaldoDebit += saldoDebit
		classSummary.TotalSaldoCredit += saldoCredit
		classSummary.AccountsCount++

		grandTotalDebit += a.DebitSum
		grandTotalCredit += a.CreditSum
		grandSaldoDebit += saldoDebit
		grandSaldoCredit += saldoCredit
	}

	classes := make([]domain.SuSaClassSummary, 10)
	for i := 0; i <= 9; i++ {
		classes[i] = *classMap[i]
	}

	diff := math.Abs(grandTotalDebit - grandTotalCredit)
	isBalanced := diff < 0.009

	return &domain.SuSaOverview{
		FiscalYear:       s.fiscalYear,
		TotalDebit:       grandTotalDebit,
		TotalCredit:      grandTotalCredit,
		TotalSaldoDebit:  grandSaldoDebit,
		TotalSaldoCredit: grandSaldoCredit,
		IsBalanced:       isBalanced,
		Difference:       diff,
		Classes:          classes,
	}, nil
}

// GetSKR04Catalog returns the complete static SKR04 2026 catalog including metadata, legend, and positions.
func (s *AccountingService) GetSKR04Catalog(ctx context.Context) (*accounting.SKR04Catalog, error) {
	return accounting.GetSKR04Catalog()
}

// GetBookings returns all journal entries for the active fiscal year.
func (s *AccountingService) GetBookings(ctx context.Context) ([]domain.BookingEntry, error) {
	return s.bookingRepo.FindAll(ctx, s.fiscalYear)
}

// GetAllBookings returns all journal entries across all fiscal years.
func (s *AccountingService) GetAllBookings(ctx context.Context) ([]domain.BookingEntry, error) {
	return s.bookingRepo.FindAll(ctx, 0)
}

// CreateBooking appends a new double-entry record to the cryptographic hash chain.
func (s *AccountingService) CreateBooking(ctx context.Context, b *domain.BookingEntry) (*domain.BookingEntry, error) {
	if b.Date == "" {
		b.Date = time.Now().Format("2006-01-02")
	}
	if b.ValueDate == "" {
		b.ValueDate = b.Date
	}

	// Auto-assign fiscal year from date if not explicitly set
	if b.FiscalYear == 0 {
		startMonth := 1
		if s.settingsRepo != nil {
			if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil && cfg.FiscalYearStartMonth > 0 {
				startMonth = cfg.FiscalYearStartMonth
			}
		}
		b.FiscalYear = domain.GetFiscalYearForDate(b.Date, startMonth)
	}

	last, err := s.bookingRepo.GetLastEntry(ctx, b.FiscalYear)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous booking for fiscal year %d: %w", b.FiscalYear, err)
	}

	prevHash := domain.GenesisHash
	var nextID uint = 1
	if last != nil {
		prevHash = last.EntryHash
		nextID = last.ID + 1
	}

	if b.BookingNumber == "" {
		b.BookingNumber = fmt.Sprintf("B-%d-%04d", b.FiscalYear, nextID)
	}
	if b.Currency == "" {
		b.Currency = "EUR"
	}
	if b.ExchangeRate == 0 {
		b.ExchangeRate = 1.0
	}

	if b.TaxCode == "" {
		b.TaxCode = "NONE"
	}

	b.PreviousHash = prevHash
	b.EntryHash = s.CalculateEntryHash(b, b.PreviousHash)

	if err := s.bookingRepo.Create(ctx, b); err != nil {
		return nil, fmt.Errorf("failed to persist booking entry: %w", err)
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(
			ctx,
			domain.AuditActionCreate,
			"BOOKING",
			fmt.Sprintf("%d", b.ID),
			fmt.Sprintf("Buchung %s (%.2f %s, %s an %s, GJ %d)", b.BookingNumber, b.Amount, b.Currency, b.DebitAccount, b.CreditAccount, b.FiscalYear),
		)
	}

	return b, nil
}

// StornoBooking creates a compensating booking to cancel a previous booking according to GoBD rules.
func (s *AccountingService) StornoBooking(ctx context.Context, bookingID uint, reason string) (*domain.BookingEntry, error) {
	orig, err := s.bookingRepo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("original booking ID %d not found: %w", bookingID, err)
	}

	if orig.IsStorno || orig.StornoForID != nil {
		return nil, fmt.Errorf("Buchung %s ist bereits eine Stornobuchung und kann nicht erneut storniert werden", orig.BookingNumber)
	}

	existingStorno, err := s.bookingRepo.FindByStornoForID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing storno for booking ID %d: %w", bookingID, err)
	}
	if existingStorno != nil {
		return nil, fmt.Errorf("Buchung %s (ID %d) wurde bereits durch Stornobuchung %s (ID %d) storniert", orig.BookingNumber, orig.ID, existingStorno.BookingNumber, existingStorno.ID)
	}

	stornoEntry := &domain.BookingEntry{
		FiscalYear:    orig.FiscalYear,
		BookingNumber: fmt.Sprintf("STORNO-%s", orig.BookingNumber),
		Date:          time.Now().Format("2006-01-02"),
		ValueDate:     orig.ValueDate,
		Description:   fmt.Sprintf("STORNO zu %s: %s (Grund: %s)", orig.BookingNumber, orig.Description, reason),
		DebitAccount:  orig.CreditAccount, // Swapped
		CreditAccount: orig.DebitAccount,  // Swapped
		Amount:        orig.Amount,
		Currency:      orig.Currency,
		ExchangeRate:  orig.ExchangeRate,
		TaxCode:       orig.TaxCode,
		TaxAmount:     orig.TaxAmount,
		ReceiptNumber: orig.ReceiptNumber,
		ReceiptHash:   orig.ReceiptHash,
		IsStorno:      true,
		StornoForID:   &orig.ID,
	}

	created, err := s.CreateBooking(ctx, stornoEntry)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(
			ctx,
			domain.AuditActionStorno,
			"BOOKING",
			fmt.Sprintf("%d", created.ID),
			fmt.Sprintf("Stornierung von Buchung ID %d (%s)", bookingID, reason),
		)
	}

	return created, nil
}

// CalculateEntryHash calculates the SHA256 cryptographic digest of a booking entry.
func (s *AccountingService) CalculateEntryHash(b *domain.BookingEntry, prevHash string) string {
	stornoIDStr := ""
	if b.StornoForID != nil {
		stornoIDStr = fmt.Sprintf("%d", *b.StornoForID)
	}
	taxCode := b.TaxCode
	if taxCode == "" {
		taxCode = "NONE"
	}
	payload := fmt.Sprintf(
		"NR:%s|FY:%d|D:%s|VD:%s|DEB:%s|CRE:%s|AMT:%.2f|CUR:%s|TX:%s|RH:%s|PREV:%s|ST:%t|STID:%s",
		b.BookingNumber,
		b.FiscalYear,
		b.Date,
		b.ValueDate,
		b.DebitAccount,
		b.CreditAccount,
		b.Amount,
		b.Currency,
		taxCode,
		b.ReceiptHash,
		prevHash,
		b.IsStorno,
		stornoIDStr,
	)
	hashBytes := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hashBytes[:])
}

// VerifyIntegrity traverses the hash chain for the active fiscal year and validates block continuity.
func (s *AccountingService) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	entries, err := s.bookingRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return domain.IntegrityCheckResult{}, err
	}

	if len(entries) == 0 {
		return domain.IntegrityCheckResult{
			IsValid:          true,
			TotalEntries:     0,
			CheckedEntries:   0,
			Message:          fmt.Sprintf("Keine Buchungen im Geschäftsjahr %d vorhanden. Die Buchhaltung ist bereit.", s.fiscalYear),
			LastVerifiedHash: domain.GenesisHash,
			CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
		}, nil
	}

	expectedPrev := domain.GenesisHash
	for i, entry := range entries {
		if entry.PreviousHash != expectedPrev {
			brokenID := entry.ID
			return domain.IntegrityCheckResult{
				IsValid:        false,
				TotalEntries:   len(entries),
				CheckedEntries: i,
				FirstBrokenID:  &brokenID,
				Message: fmt.Sprintf(
					"Unstimmigkeit bei Buchung %s (ID %d): Vorgänger-Referenz weicht ab.",
					entry.BookingNumber, entry.ID,
				),
				LastVerifiedHash: expectedPrev,
				CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
			}, nil
		}

		calculated := s.CalculateEntryHash(&entry, entry.PreviousHash)
		if calculated != entry.EntryHash {
			brokenID := entry.ID
			return domain.IntegrityCheckResult{
				IsValid:        false,
				TotalEntries:   len(entries),
				CheckedEntries: i,
				FirstBrokenID:  &brokenID,
				Message: fmt.Sprintf(
					"Unstimmigkeit bei Buchung %s (ID %d): Daten wurden nach der Buchung verändert.",
					entry.BookingNumber, entry.ID,
				),
				LastVerifiedHash: expectedPrev,
				CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
			}, nil
		}

		expectedPrev = entry.EntryHash
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(
			ctx,
			domain.AuditActionIntegrityCheck,
			"HASH_CHAIN",
			fmt.Sprintf("GJ_%d", s.fiscalYear),
			fmt.Sprintf("%d Buchungen in Geschäftsjahr %d erfolgreich auf Unveränderbarkeit geprüft", len(entries), s.fiscalYear),
		)
	}

	return domain.IntegrityCheckResult{
		IsValid:          true,
		TotalEntries:     len(entries),
		CheckedEntries:   len(entries),
		Message:          fmt.Sprintf("Alle %d Buchungen in Geschäftsjahr %d sind vollständig und unverändert.", len(entries), s.fiscalYear),
		LastVerifiedHash: expectedPrev,
		CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
	}, nil
}

// GetFinancialSummary aggregates KPIs (Revenues, Expenses, Bank balance, OPOS) for the active fiscal year.
func (s *AccountingService) GetFinancialSummary(ctx context.Context) (*domain.FinancialSummary, error) {
	totalRev, _ := s.bookingRepo.CalculateTypeSums(ctx, domain.AccountTypeRevenue, s.fiscalYear)
	totalExp, _ := s.bookingRepo.CalculateTypeSums(ctx, domain.AccountTypeExpense, s.fiscalYear)

	bankDebit, bankCredit, _ := s.bookingRepo.CalculateAccountSums(ctx, "1800", s.fiscalYear)
	recDebit, recCredit, _ := s.bookingRepo.CalculateAccountSums(ctx, "1200", s.fiscalYear)
	payDebit, payCredit, _ := s.bookingRepo.CalculateAccountSums(ctx, "3300", s.fiscalYear)

	cashflow, _ := s.bookingRepo.GetMonthlyCashflow(ctx, s.fiscalYear)

	return &domain.FinancialSummary{
		TotalRevenue:    totalRev,
		TotalExpenses:   totalExp,
		NetIncome:       totalRev - totalExp,
		BankBalance:     bankDebit - bankCredit,
		OpenReceivables: recDebit - recCredit,
		OpenPayables:    payCredit - payDebit,
		CashflowHistory: cashflow,
	}, nil
}

// GetAvailableFiscalYears returns all distinct fiscal years containing bookings plus the active/current year.
func (s *AccountingService) GetAvailableFiscalYears(ctx context.Context) []int {
	startMonth := 1
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil && cfg.FiscalYearStartMonth > 0 {
			startMonth = cfg.FiscalYearStartMonth
		}
	}

	currentFY := domain.GetFiscalYearForDate(time.Now().Format("2006-01-02"), startMonth)
	yearsMap := map[int]bool{
		currentFY: true,
	}

	if s.fiscalYear > 0 {
		yearsMap[s.fiscalYear] = true
	}

	if dbYears, err := s.bookingRepo.GetAvailableFiscalYears(ctx); err == nil {
		for _, y := range dbYears {
			if y > 0 {
				yearsMap[y] = true
			}
		}
	}

	var result []int
	for y := range yearsMap {
		result = append(result, y)
	}
	sort.Ints(result)
	return result
}
