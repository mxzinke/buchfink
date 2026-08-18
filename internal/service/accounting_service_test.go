package service_test

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

func setupTestServices(t *testing.T) (*service.AccountingService, *service.BankService, *service.InvoiceService, *service.EBilanzService, domain.SettingsRepository) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	if err := repository.SeedDefaultsIfEmpty(ctx, db, 2024); err != nil {
		t.Fatalf("failed to seed defaults: %v", err)
	}

	accRepo := repository.NewAccountRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	bankRepo := repository.NewBankRepository(db)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	accSvc := service.NewAccountingService(accRepo, bookingRepo, settingsRepo, auditRepo, 2024)
	bankSvc := service.NewBankService(bankRepo, accSvc, auditRepo)
	invSvc := service.NewInvoiceService(invoiceRepo, contactRepo, settingsRepo, auditRepo)
	ebilanzSvc := service.NewEBilanzService(accSvc, settingsRepo, auditRepo)

	return accSvc, bankSvc, invSvc, ebilanzSvc, settingsRepo
}

func TestAccountingService_CreateAndVerifyChain(t *testing.T) {
	ctx := context.Background()
	accSvc, _, _, _, _ := setupTestServices(t)

	// 1. Initial integrity check on empty journal
	res, err := accSvc.VerifyIntegrity(ctx)
	if err != nil || !res.IsValid {
		t.Fatalf("expected valid initial chain, got %v (err: %v)", res, err)
	}

	// 2. Create first booking
	b1 := &domain.BookingEntry{
		Description:   "Zahlungseingang Kunde",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        1190.00,
		Currency:      "EUR",
		TaxCode:       "UST19",
		ReceiptNumber: "RE-2024-001",
	}
	created1, err := accSvc.CreateBooking(ctx, b1)
	if err != nil {
		t.Fatalf("failed to create booking 1: %v", err)
	}
	if created1.PreviousHash != domain.GenesisHash {
		t.Fatalf("expected genesis prev hash, got %s", created1.PreviousHash)
	}

	// 3. Create second booking
	b2 := &domain.BookingEntry{
		Description:   "Serverkosten",
		DebitAccount:  "6800",
		CreditAccount: "1800",
		Amount:        89.25,
		Currency:      "EUR",
		ReceiptNumber: "BE-2024-001",
	}
	created2, err := accSvc.CreateBooking(ctx, b2)
	if err != nil {
		t.Fatalf("failed to create booking 2: %v", err)
	}
	if created2.PreviousHash != created1.EntryHash {
		t.Fatalf("expected prev hash %s, got %s", created1.EntryHash, created2.PreviousHash)
	}

	// 4. Verify chain with 2 bookings
	res2, err := accSvc.VerifyIntegrity(ctx)
	if err != nil || !res2.IsValid || res2.CheckedEntries != 2 {
		t.Fatalf("expected valid chain with 2 entries, got %+v (err: %v)", res2, err)
	}

	// 5. Test Storno
	storno, err := accSvc.StornoBooking(ctx, created2.ID, "Falsches Konto")
	if err != nil {
		t.Fatalf("failed to storno booking: %v", err)
	}
	if !storno.IsStorno || storno.DebitAccount != "1800" || storno.CreditAccount != "6800" {
		t.Fatalf("expected reversed accounts on storno, got debit=%s, credit=%s", storno.DebitAccount, storno.CreditAccount)
	}

	// 5b. Prevent duplicate storno on the same booking
	_, err = accSvc.StornoBooking(ctx, created2.ID, "Zweiter Versuch")
	if err == nil {
		t.Fatalf("expected error when trying to storno already stornoed booking, got nil")
	}

	// 5c. Prevent storno of a storno entry
	_, err = accSvc.StornoBooking(ctx, storno.ID, "Storno des Stornos")
	if err == nil {
		t.Fatalf("expected error when trying to storno a storno entry, got nil")
	}

	// 6. Chain should still be 100% valid with 3 bookings
	res3, err := accSvc.VerifyIntegrity(ctx)
	if err != nil || !res3.IsValid || res3.CheckedEntries != 3 {
		t.Fatalf("expected valid chain with 3 entries, got %+v", res3)
	}

	// 7. Test Financial Summary
	summary, err := accSvc.GetFinancialSummary(ctx)
	if err != nil {
		t.Fatalf("failed to get financial summary: %v", err)
	}
	if summary.TotalRevenue != 1190.00 {
		t.Fatalf("expected revenue 1190.00, got %f", summary.TotalRevenue)
	}
	// Serverkosten (89.25) wurde storniert -> TotalExpenses sollte 0 sein
	if summary.TotalExpenses != 0.00 {
		t.Fatalf("expected expense 0.00 after storno, got %f", summary.TotalExpenses)
	}
	if summary.NetIncome != 1190.00 {
		t.Fatalf("expected net income 1190.00, got %f", summary.NetIncome)
	}
}

func TestAccountingService_AccountLedgerAndSuSa(t *testing.T) {
	ctx := context.Background()
	accSvc, _, _, _, _ := setupTestServices(t)

	// Create test bookings
	_, err := accSvc.CreateBooking(ctx, &domain.BookingEntry{
		Description:   "Umsatz Software",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        2000.00,
	})
	if err != nil {
		t.Fatalf("failed to create booking 1: %v", err)
	}

	_, err = accSvc.CreateBooking(ctx, &domain.BookingEntry{
		Description:   "Büromiete",
		DebitAccount:  "6500",
		CreditAccount: "1800",
		Amount:        500.00,
	})
	if err != nil {
		t.Fatalf("failed to create booking 2: %v", err)
	}

	// 1. Test GetAccounts
	accounts, err := accSvc.GetAccounts(ctx)
	if err != nil {
		t.Fatalf("failed to get accounts: %v", err)
	}
	if len(accounts) < 1800 {
		t.Fatalf("expected > 1800 SKR04 accounts, got %d", len(accounts))
	}

	// Find Bank (1800)
	var bank *domain.Account
	for i := range accounts {
		if accounts[i].Number == "1800" {
			bank = &accounts[i]
			break
		}
	}
	if bank == nil {
		t.Fatalf("account 1800 not found")
	}
	if bank.DebitSum != 2000.00 || bank.CreditSum != 500.00 || bank.Balance != 1500.00 {
		t.Fatalf("bank sums mismatch: debit=%f, credit=%f, balance=%f", bank.DebitSum, bank.CreditSum, bank.Balance)
	}

	// 2. Test GetAccountLedger
	ledger, err := accSvc.GetAccountLedger(ctx, "1800")
	if err != nil {
		t.Fatalf("failed to get account ledger for 1800: %v", err)
	}
	if ledger.TotalDebit != 2000.00 || ledger.TotalCredit != 500.00 || ledger.ClosingBalance != 1500.00 {
		t.Fatalf("ledger totals mismatch: debit=%f, credit=%f, closing=%f", ledger.TotalDebit, ledger.TotalCredit, ledger.ClosingBalance)
	}
	if len(ledger.Entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(ledger.Entries))
	}
	if ledger.Entries[0].Direction != "SOLL" || ledger.Entries[0].DebitAmount != 2000.00 || ledger.Entries[0].CounterAccount != "4400" {
		t.Fatalf("entry 0 mismatch: %+v", ledger.Entries[0])
	}
	if ledger.Entries[1].Direction != "HABEN" || ledger.Entries[1].CreditAmount != 500.00 || ledger.Entries[1].CounterAccount != "6500" {
		t.Fatalf("entry 1 mismatch: %+v", ledger.Entries[1])
	}

	// 3. Test GetSuSaOverview (Summen- und Saldenliste)
	susa, err := accSvc.GetSuSaOverview(ctx)
	if err != nil {
		t.Fatalf("failed to get SuSa: %v", err)
	}
	if !susa.IsBalanced {
		t.Fatalf("expected balanced SuSa, got diff=%f", susa.Difference)
	}
	if susa.TotalDebit != 2500.00 || susa.TotalCredit != 2500.00 {
		t.Fatalf("expected total debit/credit 2500.00, got debit=%f, credit=%f", susa.TotalDebit, susa.TotalCredit)
	}
	if len(susa.Classes) != 10 {
		t.Fatalf("expected 10 Kontenklassen, got %d", len(susa.Classes))
	}

	// 4. Test GetSKR04Catalog
	cat, err := accSvc.GetSKR04Catalog(ctx)
	if err != nil || cat == nil {
		t.Fatalf("failed to get SKR04 catalog: %v", err)
	}
	if len(cat.Positions) == 0 || len(cat.Accounts) == 0 {
		t.Fatalf("expected non-empty positions and accounts in catalog")
	}
}
