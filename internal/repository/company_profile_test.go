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

func TestBackfillCompanyProfileMovesLegacyValues(t *testing.T) {
	repo, ctx := newProfileDB(t)
	db := repo.db
	for key, value := range map[string]string{
		"zip_city": "D-80331 München", "country": "Deutschland",
		"iban": "de89 3704 0044 0532 0130 00", "bic": "cobadeffxxx", "bank_name": "Commerzbank",
	} {
		if err := repo.Set(ctx, key, value); err != nil {
			t.Fatal(err)
		}
	}
	// Konto 1800 ist bereits mit einer anderen IBAN belegt.
	if err := NewBankAccountRepository(db).Save(ctx, &domain.BankAccount{IBAN: "DE02120300000000202051", Name: "Alt", LedgerAccount: "1800"}); err != nil {
		t.Fatal(err)
	}
	foundation := &domain.Foundation{
		NotarizedOn: "2021-02-03", ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{{Name: "Anna Bauer", ShareCapital: 2_500_000, Kind: domain.ContributionCash}},
	}
	if err := NewFoundationRepository(db).Save(ctx, foundation); err != nil {
		t.Fatal(err)
	}

	if err := BackfillCompanyProfile(db); err != nil {
		t.Fatal(err)
	}
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PostalCode != "80331" || cfg.City != "München" || cfg.CountryCode != "DE" {
		t.Errorf("Anschrift nicht zerlegt: %q %q %q", cfg.PostalCode, cfg.City, cfg.CountryCode)
	}
	if cfg.IBAN != "DE89370400440532013000" || cfg.BIC != "COBADEFFXXX" || cfg.BankName != "Commerzbank" {
		t.Errorf("Bankverbindung nicht übernommen: %+v", cfg)
	}
	if cfg.FoundedOn != "2021-02-03" || cfg.ShareCapital != 2_500_000 || len(cfg.Shareholders) != 1 {
		t.Errorf("Gründungsdaten nicht übernommen: %+v", cfg)
	}
	stored, err := NewBankAccountRepository(db).FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range stored {
		if account.IBAN == "DE89370400440532013000" && account.LedgerAccount != "1810" {
			t.Errorf("Konto auf belegtem Buchungskonto angelegt: %+v", account)
		}
	}
	for _, key := range []string{"zip_city", "country", "iban", "bic", "bank_name"} {
		if _, err := repo.Get(ctx, key); err == nil {
			t.Errorf("alter Schlüssel %s steht noch da", key)
		}
	}
}

func TestBackfillCompanyProfileKeepsWhatItCannotMove(t *testing.T) {
	repo, ctx := newProfileDB(t)
	for key, value := range map[string]string{"country": "Atlantis", "iban": "DE00123456789012345678"} {
		if err := repo.Set(ctx, key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := BackfillCompanyProfile(repo.db); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"country", "iban"} {
		if value, err := repo.Get(ctx, key); err != nil || value == "" {
			t.Errorf("%s gelöscht, obwohl nicht überführbar", key)
		}
	}
}
