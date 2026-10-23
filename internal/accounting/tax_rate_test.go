package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Steuersatz folgt dem Leistungsdatum (UNV-03 K2).
//
// Die Senkung des Zweiten Corona-Steuerhilfegesetzes galt für Umsätze vom
// 1. Juli bis zum 31. Dezember 2020. Eine im Jahr 2026 nacherfasste Rechnung
// über eine Leistung aus dieser Zeit hat 16 %, und ein Programm, das dort
// 19 % vorschlägt, führt auf eine Buchung, die weder zur Rechnung noch zur
// Voranmeldung passt.
func TestTaxRateFollowsTheServiceDate(t *testing.T) {
	cases := []struct {
		date              string
		standard, reduced domain.TaxRate
	}{
		{"2019-12-31", 1900, 700},
		{"2020-06-30", 1900, 700},
		{"2020-07-01", 1600, 500},
		{"2020-12-31", 1600, 500},
		{"2021-01-01", 1900, 700},
		{"2026-03-01", 1900, 700},
	}
	for _, c := range cases {
		standard, err := TaxRateFor(c.date, false)
		if err != nil {
			t.Fatalf("%s: %v", c.date, err)
		}
		if standard != c.standard {
			t.Errorf("%s: Regelsatz %s, erwartet %s", c.date, standard.Label(), c.standard.Label())
		}
		reduced, err := TaxRateFor(c.date, true)
		if err != nil {
			t.Fatalf("%s: %v", c.date, err)
		}
		if reduced != c.reduced {
			t.Errorf("%s: ermäßigter Satz %s, erwartet %s",
				c.date, reduced.Label(), c.reduced.Label())
		}
	}

	// Vor der geführten Zeit wird nicht geraten.
	if _, err := TaxRateFor("2006-12-31", false); err == nil {
		t.Error("für einen Tag ohne hinterlegten Satz darf kein Satz herauskommen")
	}
	if _, err := TaxRateFor("", false); err == nil {
		t.Error("ohne Datum lässt sich kein Satz bestimmen")
	}
}
