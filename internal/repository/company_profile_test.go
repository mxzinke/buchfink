package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func newProfileDB(t *testing.T) (*settingsRepositoryGorm, context.Context) {
	t.Helper()
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	return &settingsRepositoryGorm{db: db}, context.Background()
}

func TestCompanySettingsKeepShareholdersUnlessSent(t *testing.T) {
	repo, ctx := newProfileDB(t)
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.FoundedOn, cfg.ShareCapital = "2019-05-02", 2_500_000
	cfg.Shareholders = []domain.CompanyShareholder{{Name: "Anna Bauer", ShareCapital: 1_500_000}, {Name: "Ben Conrad", ShareCapital: 1_000_000}}
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	// Ein Aufrufer, der die Liste nicht kennt, lässt sie stehen.
	cfg.Shareholders = nil
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.FoundedOn != "2019-05-02" || len(got.Shareholders) != 2 || got.Shareholders[1].Name != "Ben Conrad" {
		t.Fatalf("Gründungsdaten nicht erhalten: %+v", got)
	}

	got.Shareholders = []domain.CompanyShareholder{{Name: "Anna Bauer", ShareCapital: 900}}
	if err := repo.UpdateCompanySettings(ctx, got); err == nil {
		t.Fatal("Anteile, die das Stammkapital nicht ergeben, angenommen")
	}
}

func TestCompanySettingsReadBankDetailsFromTheInvoiceAccount(t *testing.T) {
	repo, ctx := newProfileDB(t)
	accounts := NewBankAccountRepository(repo.db)
	for _, account := range []domain.BankAccount{
		{IBAN: "DE89370400440532013000", Name: "Geschäftskonto", LedgerAccount: "1800", BIC: "COBADEFFXXX", BankName: "Commerzbank"},
		{IBAN: "DE02120300000000202051", Name: "Rücklage", LedgerAccount: "1810", BankName: "DKB", IsInvoiceAccount: true},
	} {
		if err := accounts.Save(ctx, &account); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IBAN != "DE02120300000000202051" || cfg.BankName != "DKB" || cfg.BIC != "" {
		t.Fatalf("Bankverbindung nicht vom Konto für Rechnungen: %+v", cfg)
	}
}
