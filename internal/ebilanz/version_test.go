package ebilanz

import (
	"strings"
	"testing"
)

// Anwendungszeiträume nach Leitfaden B.18 und den BMF-Schreiben zu 6.9 und
// 6.10: Regelfassung für Wirtschaftsjahre, die nach dem 31.12.2025 bzw.
// 31.12.2026 beginnen; zulässig schon für das Jahr davor.
func TestChooseVersion(t *testing.T) {
	cases := []struct {
		begin, end, requested string
		version               string
		regular               bool
		allowed               string
	}{
		{"2025-01-01", "2025-12-31", "", "6.9", false, "6.9"},
		{"2026-01-01", "2026-12-31", "", "6.9", true, "6.9,6.10"},
		{"2026-01-01", "2026-12-31", "6.10", "6.10", false, "6.9,6.10"},
		{"2027-01-01", "2027-12-31", "", "6.10", true, "6.10"},
		// Abweichendes Wirtschaftsjahr 2026/27: es beginnt vor dem Stichtag
		// der Regelfassung 6.10.
		{"2026-07-01", "2027-06-30", "", "6.9", true, "6.9,6.10"},
	}
	for _, c := range cases {
		got, err := ChooseVersion(c.begin, c.end, c.requested)
		if err != nil {
			t.Errorf("%s–%s: %v", c.begin, c.end, err)
			continue
		}
		if got.Version != c.version || got.Regular != c.regular || strings.Join(got.Allowed, ",") != c.allowed {
			t.Errorf("%s–%s (%q): %+v, erwartet %s regulär=%v zulässig %s", c.begin, c.end, c.requested, got, c.version, c.regular, c.allowed)
		}
		if got.Reason == "" {
			t.Errorf("%s–%s: Begründung fehlt", c.begin, c.end)
		}
	}
	for _, c := range [][3]string{
		{"2024-01-01", "2024-12-31", ""},
		{"2028-01-01", "2028-12-31", ""},
		{"2025-01-01", "2025-12-31", "6.10"},
		{"2025-01-01", "", ""},
	} {
		if _, err := ChooseVersion(c[0], c[1], c[2]); err == nil {
			t.Errorf("%v: Fehler erwartet", c)
		}
	}
}

// Beide Fassungen laden mit Herkunft und Lizenzvermerk; die Wurzeln von
// Bilanz und GuV sind Summenmussfelder ihrer Rolle.
func TestEmbeddedTaxonomies(t *testing.T) {
	for _, version := range Versions() {
		tax, err := LoadTaxonomy(version)
		if err != nil {
			t.Fatal(err)
		}
		if tax.Version != version || tax.Copyright != "(c) XBRL Deutschland e.V. - see http://www.xbrl.de" ||
			!strings.Contains(tax.Notice, "Abgeleiteter Auszug, bearbeitet") || !strings.Contains(tax.Source, "SHA-256") {
			t.Errorf("%s: Kopf unvollständig: %q %q %q", version, tax.Copyright, tax.Notice, tax.Source)
		}
		for _, root := range [][2]string{{RoleBalanceSheet, "bs.ass"}, {RoleBalanceSheet, "bs.eqLiab"}, {RoleIncomeStatement, "is.netIncome"}} {
			el, ok := tax.Element(root[1])
			if !ok || !el.InRole(root[0]) || el.Req != ReqSumMandatory || len(tax.CalcChildren(root[0], root[1])) == 0 {
				t.Errorf("%s: %s fehlt oder ist keine Summe der Rolle %s", version, root[1], root[0])
			}
		}
		if tax.QName("bs.ass") != "de-gaap-ci:bs.ass" || tax.QName("genInfo.company.id.name") != "de-gcd:genInfo.company.id.name" {
			t.Errorf("%s: Präfixe falsch", version)
		}
	}
	if _, err := LoadTaxonomy("6.8"); err == nil {
		t.Error("nicht eingebettete Fassung geladen")
	}
}
