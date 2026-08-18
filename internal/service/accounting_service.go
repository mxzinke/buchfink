package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// AccountingService handles core double-entry bookkeeping and GoBD hash chain generation.
type AccountingService struct {
	accountRepo domain.AccountRepository
	bookingRepo domain.BookingRepository
	auditRepo   domain.AuditRepository
	fiscalYear  int
}

func NewAccountingService(
	accountRepo domain.AccountRepository,
	bookingRepo domain.BookingRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *AccountingService {
	return &AccountingService{
		accountRepo: accountRepo,
		bookingRepo: bookingRepo,
		auditRepo:   auditRepo,
		fiscalYear:  fiscalYear,
	}
}

// GetAccounts returns all SKR04 accounts with dynamically calculated balances.
func (s *AccountingService) GetAccounts(ctx context.Context) ([]domain.Account, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	for i := range accounts {
		debit, credit, _ := s.bookingRepo.CalculateAccountSums(ctx, accounts[i].Number)
		switch accounts[i].Type {
		case domain.AccountTypeAsset, domain.AccountTypeExpense:
			accounts[i].Balance = debit - credit
		case domain.AccountTypeLiability, domain.AccountTypeRevenue:
			accounts[i].Balance = credit - debit
		default:
			accounts[i].Balance = debit - credit
		}
	}

	return accounts, nil
}

// GetBookings returns all journal entries.
func (s *AccountingService) GetBookings(ctx context.Context) ([]domain.BookingEntry, error) {
	return s.bookingRepo.FindAll(ctx)
}

// CreateBooking appends a new double-entry record to the cryptographic hash chain.
func (s *AccountingService) CreateBooking(ctx context.Context, b *domain.BookingEntry) (*domain.BookingEntry, error) {
	last, err := s.bookingRepo.GetLastEntry(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous booking: %w", err)
	}

	prevHash := domain.GenesisHash
	var nextID uint = 1
	if last != nil {
		prevHash = last.EntryHash
		nextID = last.ID + 1
	}

	if b.BookingNumber == "" {
		b.BookingNumber = fmt.Sprintf("B-%d-%04d", s.fiscalYear, nextID)
	}
	if b.Date == "" {
		b.Date = time.Now().Format("2006-01-02")
	}
	if b.ValueDate == "" {
		b.ValueDate = b.Date
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

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionCreate,
		"BOOKING",
		fmt.Sprintf("%d", b.ID),
		fmt.Sprintf("Buchung %s (%.2f %s, %s an %s)", b.BookingNumber, b.Amount, b.Currency, b.DebitAccount, b.CreditAccount),
	)

	return b, nil
}

// StornoBooking creates a compensating booking to cancel a previous booking according to GoBD rules.
func (s *AccountingService) StornoBooking(ctx context.Context, bookingID uint, reason string) (*domain.BookingEntry, error) {
	orig, err := s.bookingRepo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("original booking ID %d not found: %w", bookingID, err)
	}

	stornoEntry := &domain.BookingEntry{
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

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionStorno,
		"BOOKING",
		fmt.Sprintf("%d", created.ID),
		fmt.Sprintf("Stornierung von Buchung ID %d (%s)", bookingID, reason),
	)

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
		"NR:%s|D:%s|VD:%s|DEB:%s|CRE:%s|AMT:%.2f|CUR:%s|TX:%s|RH:%s|PREV:%s|ST:%t|STID:%s",
		b.BookingNumber,
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

// VerifyIntegrity traverses the entire hash chain and validates block continuity.
func (s *AccountingService) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	entries, err := s.bookingRepo.FindAll(ctx)
	if err != nil {
		return domain.IntegrityCheckResult{}, err
	}

	if len(entries) == 0 {
		return domain.IntegrityCheckResult{
			IsValid:          true,
			TotalEntries:     0,
			CheckedEntries:   0,
			Message:          "Keine Buchungen vorhanden. Die Kette ist intakt (Genesis-Zustand).",
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
					"Integritätsfehler bei Buchung %s (ID %d): PreviousHash stimmt nicht mit dem Vorgänger überein.",
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
					"Integritätsfehler bei Buchung %s (ID %d): Daten wurden nachträglich manipuliert.",
					entry.BookingNumber, entry.ID,
				),
				LastVerifiedHash: expectedPrev,
				CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
			}, nil
		}

		expectedPrev = entry.EntryHash
	}

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionIntegrityCheck,
		"HASH_CHAIN",
		"ALL",
		fmt.Sprintf("%d Buchungen erfolgreich auf GoBD-Konformität geprüft", len(entries)),
	)

	return domain.IntegrityCheckResult{
		IsValid:          true,
		TotalEntries:     len(entries),
		CheckedEntries:   len(entries),
		Message:          fmt.Sprintf("Alle %d Buchungen sind kryptografisch intakt und GoBD-konform verkettet.", len(entries)),
		LastVerifiedHash: expectedPrev,
		CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
	}, nil
}

// GetFinancialSummary aggregates KPIs (Revenues, Expenses, Bank balance, OPOS).
func (s *AccountingService) GetFinancialSummary(ctx context.Context) (*domain.FinancialSummary, error) {
	totalRev, _ := s.bookingRepo.CalculateTypeSums(ctx, domain.AccountTypeRevenue)
	totalExp, _ := s.bookingRepo.CalculateTypeSums(ctx, domain.AccountTypeExpense)

	bankDebit, bankCredit, _ := s.bookingRepo.CalculateAccountSums(ctx, "1800")
	recDebit, recCredit, _ := s.bookingRepo.CalculateAccountSums(ctx, "1200")
	payDebit, payCredit, _ := s.bookingRepo.CalculateAccountSums(ctx, "3300")

	cashflow, _ := s.bookingRepo.GetMonthlyCashflow(ctx)

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
