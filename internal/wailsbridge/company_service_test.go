package wailsbridge

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Ein Darlehensvertrag entsteht in den Stammdaten: nur mit einem
// Gesellschafter aus der Gesellschafterliste, berichtigt nur mit Grund, und
// die Liste zeigt ihn mit dem Namen des Gesellschafters.
func TestShareholderLoansAreRegisteredInTheApp(t *testing.T) {
	b := testBridge(t)
	settings, err := b.GetCompanySettings()
	if err != nil {
		t.Fatal(err)
	}
	settings.LegalForm = "GmbH"
	settings.ShareCapital = 2500000
	settings.Shareholders = []domain.CompanyShareholder{{Name: "Anna Bauer", ShareCapital: 2500000}}
	if err := b.UpdateCompanySettings(*settings); err != nil {
		t.Fatalf("Gesellschafterliste: %v", err)
	}
	anna, err := b.SaveContact(domain.Contact{Type: domain.ContactTypeVendor, Name: "Anna Bauer", CountryCode: "DE"})
	if err != nil {
		t.Fatalf("Kontakt: %v", err)
	}
	stranger, err := b.SaveContact(domain.Contact{Type: domain.ContactTypeVendor, Name: "Bernd Fremd", CountryCode: "DE"})
	if err != nil {
		t.Fatalf("Kontakt: %v", err)
	}

	req := service.LoanRequest{
		ContactID: anna.ID, Direction: domain.LoanToCompany, Principal: 1000000,
		InterestRateBP: 350, StartDate: "2026-03-01", MaturityDate: "2028-02-29",
	}
	if _, err := b.RegisterShareholderLoan(service.LoanRequest{
		ContactID: stranger.ID, Direction: req.Direction, Principal: req.Principal,
		InterestRateBP: req.InterestRateBP, StartDate: req.StartDate,
	}); err == nil || !strings.Contains(err.Error(), "Gesellschafterliste") {
		t.Errorf("Darlehen eines Dritten angenommen: %v", err)
	}
	view, err := b.RegisterShareholderLoan(req)
	if err != nil {
		t.Fatalf("Darlehen erfassen: %v", err)
	}
	if view.Loan.Account == "" || view.Loan.InterestAccount == "" {
		t.Errorf("Konten nicht abgeleitet: %+v", view.Loan)
	}

	loans, err := b.GetShareholderLoans()
	if err != nil {
		t.Fatal(err)
	}
	if len(loans) != 1 || loans[0].ContactName != "Anna Bauer" || loans[0].Loan.Principal != 1000000 {
		t.Fatalf("Darlehen in der Liste: %+v", loans)
	}

	req.InterestRateBP = 400
	if _, err := b.CorrectShareholderLoan(view.Loan.ID, req, " "); err == nil {
		t.Error("Berichtigung ohne Grund angenommen")
	}
	corrected, err := b.CorrectShareholderLoan(view.Loan.ID, req, "Nachtrag zum Vertrag vom 1. März")
	if err != nil {
		t.Fatalf("Berichtigen: %v", err)
	}
	if corrected.Loan.InterestRateBP != 400 {
		t.Errorf("Zinssatz nach Berichtigung: %d", corrected.Loan.InterestRateBP)
	}
}

// Die Einrichtung einer Gründung setzt den Starttag der Bücher als
// Gründungsdatum; das erste Geschäftsjahr beginnt dort.
func TestSetupOfAFoundingStartsTheBooksAtTheNotarization(t *testing.T) {
	b := newBridge(t)
	if err := b.SetupApplication(t.TempDir()+"/mandant", domain.CompanySettings{
		CompanyName: "Nordlicht Systeme GmbH", LegalForm: "GmbH", FiscalYear: 2026,
		FiscalYearStartMonth: 1, FoundedOn: "2026-05-04", FederalState: "HH", VatPeriod: "unknown",
	}); err != nil {
		t.Fatalf("Einrichtung: %v", err)
	}
	years, err := b.GetFiscalYears()
	if err != nil {
		t.Fatal(err)
	}
	if len(years) != 1 || years[0].StartDate != "2026-05-04" || !years[0].IsShort {
		t.Fatalf("erstes Geschäftsjahr: %+v", years)
	}
}

// Ohne Mandanten ist die Liste leer und nicht null.
func TestShareholderLoansWithoutTenantAreEmpty(t *testing.T) {
	loans, err := newBridge(t).GetShareholderLoans()
	if err != nil || loans == nil || len(loans) != 0 {
		t.Fatalf("ohne Mandanten: %v %v", loans, err)
	}
}

func TestFormatElsterTaxNumber(t *testing.T) {
	b := newBridge(t)
	got, err := b.FormatElsterTaxNumber("27/123/45678", "HH")
	if err != nil || got != "2227012345678" {
		t.Errorf("Hamburg: %q %v", got, err)
	}
	if got, err := b.FormatElsterTaxNumber("", "HH"); err != nil || got != "" {
		t.Errorf("leer: %q %v", got, err)
	}
	if _, err := b.FormatElsterTaxNumber("27/123/4567", "HH"); err == nil {
		t.Error("zu kurze Steuernummer angenommen")
	}
}

// Die App öffnet nur die Anleitungen aus der festen Liste.
func TestAgentGuidesAreFixed(t *testing.T) {
	for _, client := range []string{"", "claude-code", "claude-desktop", "opencode", "codex", "pi"} {
		url, err := agentGuideURL(client)
		if err != nil || !strings.HasPrefix(url, agentGuideBase) || !strings.HasSuffix(url, ".md") {
			t.Errorf("%q: %q %v", client, url, err)
		}
	}
	if _, err := agentGuideURL("https://example.com"); err == nil {
		t.Error("fremde Adresse angenommen")
	}
	if err := newBridge(t).OpenAgentGuide("unbekannt"); err == nil {
		t.Error("unbekannte Anleitung geöffnet")
	}
}
