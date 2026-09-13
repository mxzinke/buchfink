package domain

import (
	"strings"
	"testing"
)

func TestSplitLegacyPostalLineReadsCommonVariants(t *testing.T) {
	for _, tc := range []struct{ line, postalCode, city string }{
		{"80331 München", "80331", "München"},
		{"D-80331 München", "80331", "München"},
		{"A-1010 Wien", "1010", "Wien"},
		{"1012 AB Amsterdam", "1012 AB", "Amsterdam"},
		{"61118 Bad Vilbel", "61118", "Bad Vilbel"},
		{"München", "", "München"},
		{"London SW1A 1AA", "", "London SW1A 1AA"},
		{"", "", ""},
	} {
		postalCode, city := SplitLegacyPostalLine(tc.line)
		if postalCode != tc.postalCode || city != tc.city {
			t.Errorf("%q: %q / %q, erwartet %q / %q", tc.line, postalCode, city, tc.postalCode, tc.city)
		}
	}
}

func TestCompanyAddressLinesNameTheCountryOnlyAbroad(t *testing.T) {
	s := CompanySettings{Street: "Postfach 10 20 30", AddressAddition: "c/o Kanzlei Rot", PostalCode: "80331", City: "München"}
	if got := strings.Join(s.AddressLines(), " | "); got != "Postfach 10 20 30 | c/o Kanzlei Rot | 80331 München" {
		t.Errorf("Inland: %q", got)
	}
	s = CompanySettings{Street: "Stephansplatz 1", PostalCode: "1010", City: "Wien", CountryCode: "AT"}
	if got := strings.Join(s.AddressLines(), " | "); got != "Stephansplatz 1 | 1010 Wien | Österreich" {
		t.Errorf("Ausland: %q", got)
	}
}

func TestValidateProfile(t *testing.T) {
	valid := CompanySettings{
		PostalCode: "80331", FoundedOn: "2020-03-01", RegisteredOn: "2020-04-15",
		ShareCapital: 2_500_000,
		Shareholders: []CompanyShareholder{{Name: "Anna Bauer", ShareCapital: 1_500_000}, {Name: "Ben Conrad", ShareCapital: 1_000_000}},
	}
	if err := valid.ValidateProfile(); err != nil {
		t.Fatalf("gültige Stammdaten abgewiesen: %v", err)
	}
	for name, mutate := range map[string]func(*CompanySettings){
		"deutsche PLZ mit vier Ziffern": func(s *CompanySettings) { s.PostalCode = "8033" },
		"Eintragung vor Gründung":       func(s *CompanySettings) { s.RegisteredOn = "2020-02-01" },
		"unvollständiges Datum":         func(s *CompanySettings) { s.FoundedOn = "2020-3-1" },
		"Anteile ohne Stammkapital":     func(s *CompanySettings) { s.Shareholders[1].ShareCapital = 900_000 },
		"Gesellschafter ohne Namen":     func(s *CompanySettings) { s.Shareholders[0].Name = " " },
		"Ländercode":                    func(s *CompanySettings) { s.CountryCode = "Deutschland" },
	} {
		s := valid
		s.Shareholders = append([]CompanyShareholder{}, valid.Shareholders...)
		mutate(&s)
		if err := s.ValidateProfile(); err == nil {
			t.Errorf("%s: angenommen", name)
		}
	}
	foreign := CompanySettings{PostalCode: "1012 AB", CountryCode: "NL"}
	if err := foreign.ValidateProfile(); err != nil {
		t.Errorf("ausländische Postleitzahl abgewiesen: %v", err)
	}
}
