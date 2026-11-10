package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// withElsterTaxNumber trägt die Steuernummer der Testumgebung im
// 13-stelligen Bundesschema ein (Bayern: 143/815/08151 → 9143081508151).
func withElsterTaxNumber(t *testing.T, env *testEnv) {
	t.Helper()
	repo := repository.NewSettingsRepository(env.db)
	cfg, err := repo.GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.TaxNumber = "9143081508151"
	if err := repo.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Steuernummer: %v", err)
	}
}

// Ohne Steuernummer im Bundesschema entsteht keine E-Bilanz: sie ist Kennung
// der Instanz und Pflichtangabe der Stammdaten (Leitfaden B.14.3.1.18).
func TestEBilanzExportNeedsTheElsterTaxNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	svc := NewEBilanzService(env.statements(t), repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db))

	report, err := svc.MappingReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Prüfbericht: %v", err)
	}
	if report.CanExport {
		t.Fatal("Export ohne Steuernummer zugelassen")
	}
	if _, err := svc.ExportXBRLWith(ctx, 2026, EBilanzOptions{}); err == nil || !strings.Contains(err.Error(), "Steuernummer") {
		t.Errorf("Fehlermeldung nennt die Steuernummer nicht: %v", err)
	}
}

// Die Abweichungen der Steuerbilanz je Konto: Abzinsung mit 5,5 % und die
// steuerlich verbotene Drohverlustrückstellung (§ 6 Abs. 1 Nr. 3a Buchst. e,
// § 5 Abs. 4a EStG).
func TestTaxAdjustmentsPerAccount(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)
	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Rückbau",
		Amount: 1_000_000, ExpectedOn: "2029-12-30", Reason: "Mietvertrag § 12",
	}); err != nil {
		t.Fatalf("Rückstellung: %v", err)
	}
	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionPendingLoss, Text: "Festpreisauftrag",
		Amount: 300_000, ExpectedOn: "2027-03-31", Reason: "Kalkulation",
	}); err != nil {
		t.Fatalf("Drohverlustrückstellung: %v", err)
	}

	adjustments, err := m.register.Adjustments(ctx, 2026)
	if err != nil {
		t.Fatalf("Abweichungen: %v", err)
	}
	var discount, pendingLoss bool
	for _, adj := range adjustments {
		if adj.Account == "" || adj.IncomeAccount == "" || adj.Prior != 0 {
			t.Errorf("unvollständig: %+v", adj)
		}
		switch {
		case strings.HasPrefix(adj.Reason, "Abzinsung"):
			// 9.563,17 € handelsrechtlich; steuerlich mit 5,5 % weniger.
			discount = adj.Current > 0 && adj.Current < 956_317
		case strings.HasPrefix(adj.Reason, "Drohverlust"):
			pendingLoss = adj.Current == 300_000
		}
	}
	if !discount || !pendingLoss {
		t.Errorf("Abweichungen unvollständig: %+v", adjustments)
	}
}

// Mit Abweichungen geht die E-Bilanz als Handelsbilanz mit Überleitung
// (Leitfaden B.13, B.15).
func TestEBilanzWithAdjustmentsIsACommercialBalanceWithTransfer(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	withElsterTaxNumber(t, env)
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionPendingLoss, Text: "Festpreisauftrag",
		Amount: 300_000, ExpectedOn: "2027-03-31", Reason: "Kalkulation",
	}); err != nil {
		t.Fatalf("Drohverlustrückstellung: %v", err)
	}
	svc := NewEBilanzService(env.statements(t), repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db))
	svc.SetAdjustmentSource(m.register)

	res, err := svc.Generate(ctx, 2026, EBilanzOptions{})
	if err != nil {
		t.Fatalf("E-Bilanz: %v", err)
	}
	if !res.Report.CanExport {
		t.Fatalf("Export blockiert: %+v", res.Report.Findings)
	}
	text := string(res.Instance)
	if res.Report.AccountingStandard != "HGBM" || !strings.Contains(text, "<de-gaap-ci:hbst.transfer>") ||
		!strings.Contains(text, "reportElements.STU") {
		t.Error("keine Handelsbilanz mit Überleitung")
	}
}

// Wahlrechte der Übermittlung: Fassung 6.10 für 2026 und die ELSTER-Hülle mit
// dem Stichtag; der Export steht im Protokoll.
func TestEBilanzExportWithOptionsIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	withElsterTaxNumber(t, env)
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	audit := repository.NewAuditRepository(env.db)
	svc := NewEBilanzService(env.statements(t), repository.NewSettingsRepository(env.db), audit)

	out, err := svc.ExportXBRLWith(ctx, 2026, EBilanzOptions{Version: "6.10", Envelope: true})
	if err != nil {
		t.Fatalf("E-Bilanz: %v", err)
	}
	if !strings.Contains(out, "<ebilanz:stichtag>20261231</ebilanz:stichtag>") ||
		!strings.Contains(out, "de-gaap-ci-2026-04-01-shell-fiscal.xsd") {
		t.Error("Hülle oder Fassung fehlt")
	}
	entries, err := audit.FindAll(ctx, 20)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	logged := false
	for _, e := range entries {
		if e.EntityType == "EBILANZ" && strings.Contains(e.Details, "Taxonomie 6.10") {
			logged = true
		}
	}
	if !logged {
		t.Error("der Export steht nicht im Protokoll")
	}
}
