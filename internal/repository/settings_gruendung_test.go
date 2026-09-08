package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Firmenzusatz folgt der Gründung, nicht einer Einstellung.
//
// Bis zur Eintragung ist die Gesellschaft Vorgesellschaft und führt „i. G.".
// Der Zustand wird abgeleitet und nirgends gespeichert — sonst ginge er
// irgendwann mit dem Eintragungsdatum auseinander, und der Zusatz bliebe an
// einer längst eingetragenen Gesellschaft stehen.
func TestCompanySettingsDeriveInGruendungFromTheFoundation(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	settings := NewSettingsRepository(db)
	if err := settings.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", LegalForm: "GmbH",
		FiscalYear: 2026, FiscalYearStartMonth: 1, Currency: "EUR", SKR: "SKR04",
		VatPeriod: "quarter", TaxationType: "SOLL",
	}); err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}

	// Ohne Gründung: kein Zusatz. Der Regelfall jedes Mandanten, der nicht
	// gerade gegründet hat.
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("GetCompanySettings: %v", err)
	}
	if cfg.InGruendung {
		t.Error("ohne erfasste Gründung ist niemand in Gründung")
	}
	if got := cfg.FirmName(); got != "Pfennig Ventures GmbH" {
		t.Errorf("Firma = %q, erwartet den Namen ohne Zusatz", got)
	}

	foundations := NewFoundationRepository(db)
	f := &domain.Foundation{NotarizedOn: "2026-03-15", ShareCapital: 2_500_000}
	if err := foundations.Save(ctx, f); err != nil {
		t.Fatalf("Gründung: %v", err)
	}

	cfg, err = settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("GetCompanySettings: %v", err)
	}
	if !cfg.InGruendung {
		t.Error("zwischen Beurkundung und Eintragung ist die Gesellschaft Vorgesellschaft")
	}
	if got := cfg.FirmName(); got != "Pfennig Ventures GmbH i. G." {
		t.Errorf("Firma = %q, erwartet den Zusatz i. G.", got)
	}

	f.RegisteredOn = "2026-05-04"
	if err := foundations.Save(ctx, f); err != nil {
		t.Fatalf("Eintragung: %v", err)
	}

	cfg, err = settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("GetCompanySettings: %v", err)
	}
	if cfg.InGruendung {
		t.Error("mit der Eintragung endet die Vorgesellschaft")
	}
	if got := cfg.FirmName(); got != "Pfennig Ventures GmbH" {
		t.Errorf("Firma = %q, erwartet den Namen ohne Zusatz", got)
	}
	// Gespeichert wird der Zusatz nie: das Eingabefeld führt den rohen Namen.
	if cfg.CompanyName != "Pfennig Ventures GmbH" {
		t.Errorf("gespeicherter Name = %q, der Zusatz gehört nicht in die Datenbank", cfg.CompanyName)
	}
}

// Ein leerer Name bleibt leer — ein alleinstehendes „i. G." wäre keine Firma.
func TestFirmNameLeavesAnEmptyNameEmpty(t *testing.T) {
	cfg := &domain.CompanySettings{InGruendung: true}
	if got := cfg.FirmName(); got != "" {
		t.Errorf("Firma = %q, erwartet leer", got)
	}
}
