package domain

import (
	"strings"
	"testing"
)

func TestSplitStreet(t *testing.T) {
	cases := []struct{ line, street, number string }{
		{"Hauptstraße 12a", "Hauptstraße", "12a"},
		{"Hauptstraße 12 a", "Hauptstraße", "12 a"},
		{"Straße des 17. Juni 135", "Straße des 17. Juni", "135"},
		{"Am Markt 3-5", "Am Markt", "3-5"},
		{"Am Markt", "Am Markt", ""},
		{"Postfach 12 34 56", "Postfach 12 34 56", ""},
		{"  Lindenallee   7 ", "Lindenallee", "7"},
		{"", "", ""},
	}
	for _, c := range cases {
		street, number := SplitStreet(c.line)
		if street != c.street || number != c.number {
			t.Errorf("%q → %q / %q, erwartet %q / %q", c.line, street, number, c.street, c.number)
		}
	}
}

// Eine Hausnummer, die schon im eigenen Feld steht, bleibt dort; die Zeile
// setzt beides wieder zusammen.
func TestNormalizeAddressKeepsAnExplicitHouseNumber(t *testing.T) {
	s := CompanySettings{Street: "Hauptstraße 1", HouseNumber: ""}
	s.NormalizeAddress()
	if s.Street != "Hauptstraße" || s.HouseNumber != "1" || s.StreetLine() != "Hauptstraße 1" {
		t.Errorf("aufgeteilt: %q / %q", s.Street, s.HouseNumber)
	}
	s = CompanySettings{Street: "Weg 9", HouseNumber: "4b"}
	s.NormalizeAddress()
	if s.Street != "Weg 9" || s.HouseNumber != "4b" {
		t.Errorf("die eingetragene Hausnummer wurde verändert: %q / %q", s.Street, s.HouseNumber)
	}
}

// § 149 Abs. 3 AO gilt nur für die Erklärungen, die der Berater erstellt.
func TestTaxAdvisorPrepares(t *testing.T) {
	none := TaxAdvisor{}
	if none.Prepares(DeclarationCorporateTax, 2026) {
		t.Error("ohne Beauftragung erstellt niemand die Erklärung")
	}
	all := TaxAdvisor{Engaged: true}
	if !all.Prepares(DeclarationAnnualVat, 2020) {
		t.Error("ohne Auswahl übernimmt der Berater alle Erklärungen")
	}
	some := TaxAdvisor{Engaged: true, SinceYear: 2026, Declarations: []string{DeclarationCorporateTax, DeclarationTradeTax}}
	if some.Prepares(DeclarationCorporateTax, 2025) {
		t.Error("vor dem Jahr der Beauftragung gilt die Frist ohne Berater")
	}
	if !some.Prepares(DeclarationTradeTax, 2026) || some.Prepares(DeclarationAnnualVat, 2026) {
		t.Error("nur die ausgewählten Erklärungen")
	}
	if err := (TaxAdvisor{Declarations: []string{"est"}}).Validate(); err == nil {
		t.Error("eine unbekannte Erklärung wird abgewiesen")
	}
}

func TestValidateProfileChecksStateAndTaxNumber(t *testing.T) {
	s := CompanySettings{FederalState: "XY"}
	if err := s.ValidateProfile(); err == nil || !strings.Contains(err.Error(), "Bundesland") {
		t.Errorf("unbekanntes Land: %v", err)
	}
	s = CompanySettings{FederalState: "SN", TaxNumber: "101/815/08154"}
	if err := s.ValidateProfile(); err == nil || !strings.Contains(err.Error(), "Steuernummer") {
		t.Errorf("Steuernummer aus Sachsen-Anhalt in Sachsen: %v", err)
	}
	s = CompanySettings{FederalState: "SN", TaxNumber: "201/123/12340"}
	if err := s.ValidateProfile(); err != nil {
		t.Errorf("passende Steuernummer: %v", err)
	}
	s.ResolveTaxNumberElster()
	if s.TaxNumberElster != "3201012312340" {
		t.Errorf("ELSTER-Steuernummer = %q", s.TaxNumberElster)
	}
}

func TestUnsupportedCompanies(t *testing.T) {
	if err := (&CompanySettings{}).Unsupported(); err != nil {
		t.Errorf("eine gewöhnliche GmbH: %v", err)
	}
	for name, s := range map[string]CompanySettings{
		"KStG":       {NonProfit: true},
		"§ 71 GmbHG": {InLiquidation: true},
		"§§ 14, 17":  {FiscalUnity: true},
	} {
		err := s.Unsupported()
		if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "Steuerberater") {
			t.Errorf("%s: %v", name, err)
		}
	}
}
