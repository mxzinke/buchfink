package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestBankAccountsConfigureAtomicallyAndRouteMultiAccountStatements(t *testing.T) {
	e := newTestEnv(t)
	svc := e.banking(t)
	ctx := context.Background()
	registry := repository.NewBankAccountRepository(e.db)
	svc.SetAccountRegistry(registry, repository.NewSettingsRepository(e.db), repository.NewTxRunner(e.db))
	accounts := []domain.BankAccount{
		{IBAN: "DE89370400440532013000", Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR"},
		{IBAN: "DE02120300000000202051", Name: "Steuerrücklage", LedgerAccount: "1810", Currency: "EUR"},
	}
	bad := append([]domain.BankAccount{}, accounts...)
	bad[1].LedgerAccount = "1800"
	if err := svc.ConfigureAccounts(ctx, bad); err == nil {
		t.Fatal("two IBANs on one ledger accepted")
	}
	stored, err := registry.FindAll(ctx)
	if err != nil || len(stored) != 0 {
		t.Fatalf("failed setup partially persisted: %+v %v", stored, err)
	}
	if err := svc.ConfigureAccounts(ctx, accounts); err != nil {
		t.Fatal(err)
	}
	// Two statement blocks with separate IBANs in one original file.
	start, end := strings.Index(sampleCAMT, "<Stmt>"), strings.Index(sampleCAMT, "</Stmt>")+len("</Stmt>")
	second := strings.ReplaceAll(sampleCAMT[start:end], accounts[0].IBAN, accounts[1].IBAN)
	input := sampleCAMT[:end] + second + sampleCAMT[end:]
	count, err := svc.ImportCAMT053(ctx, strings.NewReader(input), "")
	if err != nil || count != 4 {
		t.Fatalf("multi-account import: %d, %v", count, err)
	}
	transactions, err := repository.NewBankRepository(e.db).FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	for _, tx := range transactions {
		want := "1800"
		if tx.AccountIBAN == accounts[1].IBAN {
			want = "1810"
		}
		if tx.LedgerAccount != want || tx.StatementReceiptID == nil {
			t.Fatalf("wrong account or missing original: %+v", tx)
		}
	}
	count, err = svc.ImportCAMT053(ctx, strings.NewReader(strings.ReplaceAll(input, "<BkToCstmrStmt>", "<BkToCstmrStmt>\n")), "")
	if err != nil || count != 0 {
		t.Fatalf("repackaged original duplicated transactions: %d %v", count, err)
	}
	accounts[0].LedgerAccount = "1820"
	if err := svc.ConfigureAccounts(ctx, accounts[:1]); err == nil {
		t.Fatal("historical ledger silently changed")
	}
	settings, err := repository.NewSettingsRepository(e.db).GetCompanySettings(ctx)
	if err != nil || settings.IBAN != "DE89370400440532013000" {
		t.Fatalf("invoice bank replaced: %+v %v", settings, err)
	}
}

// withInvoiceAccount richtet das Bankkonto ein, dessen Verbindung auf
// Rechnungen steht.
func (e *testEnv) withInvoiceAccount(t *testing.T, iban, bic, bankName string) {
	t.Helper()
	account := domain.BankAccount{
		IBAN: iban, Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR",
		BIC: bic, BankName: bankName, IsInvoiceAccount: true,
	}
	if err := repository.NewBankAccountRepository(e.db).Save(context.Background(), &account); err != nil {
		t.Fatalf("Bankkonto für Rechnungen einrichten: %v", err)
	}
}

func TestInvoiceAccountIsChosenExplicitlyAndSurvivesImports(t *testing.T) {
	e := newTestEnv(t)
	svc := e.banking(t)
	ctx := context.Background()
	registry := repository.NewBankAccountRepository(e.db)
	settings := repository.NewSettingsRepository(e.db)
	svc.SetAccountRegistry(registry, settings, repository.NewTxRunner(e.db))

	accounts := []domain.BankAccount{
		{IBAN: "DE89 3704 0044 0532 0130 00", Name: "Geschäftskonto", LedgerAccount: "1800", BIC: "cobadeffxxx", BankName: "Commerzbank"},
		{IBAN: "DE02120300000000202051", Name: "Steuerrücklage", LedgerAccount: "1810"},
	}
	if err := svc.ConfigureAccounts(ctx, accounts); err != nil {
		t.Fatal(err)
	}
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil || cfg.IBAN != "DE89370400440532013000" || cfg.BIC != "COBADEFFXXX" || cfg.BankName != "Commerzbank" {
		t.Fatalf("das erste Konto steht nicht auf Rechnungen: %+v %v", cfg, err)
	}

	if err := svc.SetInvoiceAccount(ctx, "DE02120300000000202051"); err != nil {
		t.Fatal(err)
	}
	// Ein erneuter Import desselben Kontos nimmt die Auswahl nicht zurück.
	if err := svc.ConfigureAccounts(ctx, accounts[:1]); err != nil {
		t.Fatal(err)
	}
	stored, err := registry.FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	flagged := 0
	for _, account := range stored {
		if account.IsInvoiceAccount {
			flagged++
			if account.IBAN != "DE02120300000000202051" {
				t.Errorf("falsches Konto für Rechnungen: %+v", account)
			}
		}
	}
	if flagged != 1 {
		t.Errorf("%d Konten für Rechnungen markiert, erwartet eins", flagged)
	}

	if err := svc.SetInvoiceAccount(ctx, "DE75512108001245126199"); err == nil {
		t.Error("unbekanntes Konto als Rechnungskonto angenommen")
	}
	bad := []domain.BankAccount{{IBAN: "DE75512108001245126199", Name: "Neu", LedgerAccount: "1820", BIC: "KEINBIC"}}
	if err := svc.ConfigureAccounts(ctx, bad); err == nil {
		t.Error("ungültige BIC angenommen")
	}
}
