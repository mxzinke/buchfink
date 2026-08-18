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

	accSvc := service.NewAccountingService(accRepo, bookingRepo, auditRepo, 2024)
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
