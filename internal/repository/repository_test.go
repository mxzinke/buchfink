package repository_test

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestGORMRepositories(t *testing.T) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("failed to init in-memory db: %v", err)
	}

	// 1. Account Repository
	accRepo := repository.NewAccountRepository(db)
	acc := &domain.Account{
		Number:   "1800",
		Name:     "Bank",
		Type:     domain.AccountTypeAsset,
		Category: "Liquide Mittel",
	}
	if err := accRepo.Create(ctx, acc); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	foundAcc, err := accRepo.FindByNumber(ctx, "1800")
	if err != nil || foundAcc.Name != "Bank" {
		t.Fatalf("expected account Bank, got %v (err: %v)", foundAcc, err)
	}

	count, _ := accRepo.Count(ctx)
	if count != 1 {
		t.Fatalf("expected 1 account, got %d", count)
	}

	// 2. Booking Repository
	bookingRepo := repository.NewBookingRepository(db)
	booking := &domain.BookingEntry{
		FiscalYear:    2024,
		BookingNumber: "B-2024-0001",
		Date:          "2024-01-15",
		ValueDate:     "2024-01-15",
		Description:   "Erlöse Barverkauf",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        1000.0,
		Currency:      "EUR",
		PreviousHash:  domain.GenesisHash,
		EntryHash:     "mockhash123",
	}
	if err := bookingRepo.Create(ctx, booking); err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	last, err := bookingRepo.GetLastEntry(ctx, 2024)
	if err != nil || last == nil || last.BookingNumber != "B-2024-0001" {
		t.Fatalf("expected last entry B-2024-0001, got %v", last)
	}

	debitSum, creditSum, err := bookingRepo.CalculateAccountSums(ctx, "1800", 2024)
	if err != nil || debitSum != 1000.0 || creditSum != 0.0 {
		t.Fatalf("expected debit 1000 and credit 0, got debit=%f, credit=%f", debitSum, creditSum)
	}

	// Test FindByStornoForID
	stornoBooking := &domain.BookingEntry{
		FiscalYear:    2024,
		BookingNumber: "STORNO-B-2024-0001",
		Date:          "2024-01-16",
		ValueDate:     "2024-01-16",
		Description:   "Storno zu B-2024-0001",
		DebitAccount:  "4400",
		CreditAccount: "1800",
		Amount:        1000.0,
		Currency:      "EUR",
		PreviousHash:  booking.EntryHash,
		EntryHash:     "mockhash456",
		IsStorno:      true,
		StornoForID:   &booking.ID,
	}
	if err := bookingRepo.Create(ctx, stornoBooking); err != nil {
		t.Fatalf("failed to create storno booking: %v", err)
	}

	foundStorno, err := bookingRepo.FindByStornoForID(ctx, booking.ID)
	if err != nil || foundStorno == nil || foundStorno.BookingNumber != "STORNO-B-2024-0001" {
		t.Fatalf("expected storno booking STORNO-B-2024-0001, got %v (err: %v)", foundStorno, err)
	}

	notFoundStorno, err := bookingRepo.FindByStornoForID(ctx, 99999)
	if err != nil || notFoundStorno != nil {
		t.Fatalf("expected nil for non-existent storno, got %v (err: %v)", notFoundStorno, err)
	}

	// 3. Bank Repository
	bankRepo := repository.NewBankRepository(db)
	txs := []domain.BankTransaction{
		{
			AccountIBAN:      "DE89370400440532013000",
			BookingDate:      "2024-04-15",
			ValueDate:        "2024-04-15",
			Amount:           500.0,
			Currency:         "EUR",
			CounterpartyName: "Kunde XY",
			EndToEndID:       "E2E-999",
			MatchStatus:      domain.MatchStatusUnmatched,
		},
	}
	inserted, err := bankRepo.CreateBatch(ctx, txs)
	if err != nil || inserted != 1 {
		t.Fatalf("expected 1 inserted bank tx, got %d (err: %v)", inserted, err)
	}

	allTxs, err := bankRepo.FindAll(ctx, 0)
	if err != nil || len(allTxs) != 1 {
		t.Fatalf("expected 1 tx in list, got %d", len(allTxs))
	}

	if err := bankRepo.MarkMatched(ctx, allTxs[0].ID, 1); err != nil {
		t.Fatalf("failed to mark matched: %v", err)
	}

	// 4. Contact Repository
	contactRepo := repository.NewContactRepository(db)
	contact := &domain.Contact{
		Type:             domain.ContactTypeCustomer,
		Number:           "K-10001",
		Name:             "Acme Corp",
		PaymentTermsDays: 14,
	}
	if err := contactRepo.Save(ctx, contact); err != nil {
		t.Fatalf("failed to save contact: %v", err)
	}

	foundContact, err := contactRepo.FindByNumber(ctx, "K-10001")
	if err != nil || foundContact.Name != "Acme Corp" {
		t.Fatalf("expected contact Acme Corp, got %v", foundContact)
	}

	// 5. Invoice Repository
	invoiceRepo := repository.NewInvoiceRepository(db)
	inv := &domain.Invoice{
		InvoiceNumber: "RE-2024-001",
		Date:          "2024-01-01",
		DueDate:       "2024-01-15",
		ContactID:     contact.ID,
		ContactName:   contact.Name,
		NetAmount:     1000.0,
		TaxAmount:     190.0,
		GrossAmount:   1190.0,
		Currency:      "EUR",
		Status:        domain.InvoiceStatusIssued,
		Items: []domain.InvoiceItem{
			{Position: 1, Description: "Entwicklungsleistung", Quantity: 10, UnitPrice: 100, TotalNet: 1000, TotalGross: 1190},
		},
	}
	if err := invoiceRepo.Save(ctx, inv); err != nil {
		t.Fatalf("failed to save invoice: %v", err)
	}

	foundInv, err := invoiceRepo.FindByNumber(ctx, "RE-2024-001")
	if err != nil || len(foundInv.Items) != 1 {
		t.Fatalf("expected invoice with 1 item, got %v", foundInv)
	}

	// 6. Audit Repository
	auditRepo := repository.NewAuditRepository(db)
	if err := auditRepo.Log(ctx, domain.AuditActionCreate, "TEST", "1", "Test Audit Log"); err != nil {
		t.Fatalf("failed to log audit entry: %v", err)
	}
	logs, err := auditRepo.FindAll(ctx, 10)
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}

	// 7. Settings Repository
	settingsRepo := repository.NewSettingsRepository(db)
	settings := &domain.CompanySettings{
		CompanyName: "Test GmbH",
		FiscalYear:  2024,
		TaxNumber:   "12/345/67890",
	}
	if err := settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		t.Fatalf("failed to update company settings: %v", err)
	}

	loadedSettings, err := settingsRepo.GetCompanySettings(ctx)
	if err != nil || loadedSettings.CompanyName != "Test GmbH" {
		t.Fatalf("expected company name 'Test GmbH', got %v", loadedSettings)
	}
}

func TestSeedDefaults(t *testing.T) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	if err := repository.SeedDefaultsIfEmpty(ctx, db, 2024); err != nil {
		t.Fatalf("failed to seed defaults: %v", err)
	}

	accRepo := repository.NewAccountRepository(db)
	count, err := accRepo.Count(ctx)
	if err != nil || count == 0 {
		t.Fatalf("expected seeded accounts, got count %d (err: %v)", count, err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	settings, err := settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings.FiscalYear != 2024 || settings.SKR != "SKR04" {
		t.Fatalf("expected default company settings (fiscal year 2024, SKR04), got %v", settings)
	}
}
