package repository

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Land, Steuerberater, Ausschlussmerkmale und die getrennte Hausnummer
// überstehen das Speichern; die ELSTER-Steuernummer wird abgeleitet.
func TestObligationProfileRoundTrip(t *testing.T) {
	repo, ctx := newProfileDB(t)
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.FederalState = " by "
	cfg.TaxNumber = "181/815/08155"
	cfg.Street, cfg.HouseNumber = "Hauptstraße 12a", ""
	cfg.TaxAdvisor = domain.TaxAdvisor{Engaged: true, SinceYear: 2025,
		Declarations: []string{domain.DeclarationCorporateTax, domain.DeclarationTradeTax}}
	cfg.FiscalUnity = true
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("speichern: %v", err)
	}
	got, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.FederalState != "BY" || got.TaxNumberElster != "9181081508155" {
		t.Errorf("Land %q, ELSTER-Steuernummer %q", got.FederalState, got.TaxNumberElster)
	}
	if got.Street != "Hauptstraße" || got.HouseNumber != "12a" || got.StreetLine() != "Hauptstraße 12a" {
		t.Errorf("Anschrift %q / %q", got.Street, got.HouseNumber)
	}
	if !got.TaxAdvisor.Engaged || got.TaxAdvisor.SinceYear != 2025 || len(got.TaxAdvisor.Declarations) != 2 ||
		got.TaxAdvisor.Declarations[1] != domain.DeclarationTradeTax {
		t.Errorf("Steuerberater %+v", got.TaxAdvisor)
	}
	if !got.FiscalUnity || got.NonProfit || got.InLiquidation {
		t.Errorf("Ausschlussmerkmale %v %v %v", got.NonProfit, got.InLiquidation, got.FiscalUnity)
	}

	got.FederalState = "SN"
	if err := repo.UpdateCompanySettings(ctx, got); err == nil {
		t.Error("eine bayerische Steuernummer passt nicht zu Sachsen")
	}
	got.FederalState, got.TaxAdvisor.Declarations = "BY", []string{"est"}
	if err := repo.UpdateCompanySettings(ctx, got); err == nil {
		t.Error("eine unbekannte Erklärung für den Steuerberater wird abgewiesen")
	}
}
