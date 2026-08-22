// ACHTUNG: Dieser Prüfer ist abgelöst.
//
// Die EN-16931-Prüfung liegt in `internal/einvoice`. Sie läuft auf einem
// semantischen Modell statt auf CII-Structs, deckt alle 223 Geschäftsregeln ab
// statt 170, liest neben CII auch UBL, und XRechnung und ZUGFeRD sitzen als
// Schichten darüber.
//
// Was hier steht, hängt nur noch am Buchungspfad (`internal/service`), der
// weiterhin die CIIInvoice-Struktur verwendet. Das Umhängen ist der zweite
// Schritt und für sich zu machen — bis dahin gilt: **neue Regeln kommen ins
// Modul, nicht hierher.** Zwei Prüfer im Baum sind genau die Stelle, an der
// jemand den falschen bearbeitet.

package invoice

import "sort"

// en16931BusinessRules lists every business rule of the EN 16931 CII validation
// artefact, as of the release this list was taken from.
//
// Only the identifiers are kept here. They are facts about the standard, and
// they exist for one purpose: to measure how far Buchfink's own checker reaches.
// A "Teilprüfung" that cannot say which fraction it covers is a claim, not a
// measurement — with this list the coverage is a number, and a test fails the
// moment the implementation claims a rule that is not in the standard.
//
// The rule texts themselves are deliberately not copied: they belong to the
// EUPL-licensed validation artefact, and Buchfink ships under MIT.
//
// Source: ConnectingEurope/eInvoicing-EN16931, cii/schematron.
var en16931BusinessRules = []string{
	"BR-01", "BR-02", "BR-03", "BR-04", "BR-05", "BR-06",
	"BR-07", "BR-08", "BR-09", "BR-10", "BR-11", "BR-12",
	"BR-13", "BR-14", "BR-15", "BR-16", "BR-17", "BR-18",
	"BR-19", "BR-20", "BR-21", "BR-22", "BR-23", "BR-24",
	"BR-25", "BR-26", "BR-27", "BR-28", "BR-29", "BR-30",
	"BR-31", "BR-32", "BR-33", "BR-36", "BR-37", "BR-38",
	"BR-41", "BR-42", "BR-43", "BR-44", "BR-45", "BR-46",
	"BR-47", "BR-48", "BR-49", "BR-50", "BR-51", "BR-52",
	"BR-53", "BR-54", "BR-55", "BR-56", "BR-57", "BR-61",
	"BR-62", "BR-63", "BR-64", "BR-65", "BR-AE-01", "BR-AE-02",
	"BR-AE-03", "BR-AE-04", "BR-AE-05", "BR-AE-06", "BR-AE-07", "BR-AE-08",
	"BR-AE-09", "BR-AE-10", "BR-AF-01", "BR-AF-02", "BR-AF-03", "BR-AF-04",
	"BR-AF-05", "BR-AF-06", "BR-AF-07", "BR-AF-08", "BR-AF-09", "BR-AF-10",
	"BR-AG-01", "BR-AG-02", "BR-AG-03", "BR-AG-04", "BR-AG-05", "BR-AG-06",
	"BR-AG-07", "BR-AG-08", "BR-AG-09", "BR-AG-10", "BR-B-01", "BR-B-02",
	"BR-CL-01", "BR-CL-03", "BR-CL-04", "BR-CL-05", "BR-CL-06", "BR-CL-07",
	"BR-CL-08", "BR-CL-10", "BR-CL-11", "BR-CL-13", "BR-CL-14", "BR-CL-15",
	"BR-CL-16", "BR-CL-17", "BR-CL-18", "BR-CL-19", "BR-CL-20", "BR-CL-21",
	"BR-CL-22", "BR-CL-23", "BR-CL-24", "BR-CL-25", "BR-CL-26", "BR-CO-03",
	"BR-CO-04", "BR-CO-05", "BR-CO-06", "BR-CO-07", "BR-CO-08", "BR-CO-09",
	"BR-CO-10", "BR-CO-11", "BR-CO-12", "BR-CO-13", "BR-CO-14", "BR-CO-15",
	"BR-CO-16", "BR-CO-17", "BR-CO-18", "BR-CO-19", "BR-CO-20", "BR-CO-21",
	"BR-CO-22", "BR-CO-23", "BR-CO-24", "BR-CO-26", "BR-DEC-01", "BR-DEC-02",
	"BR-DEC-05", "BR-DEC-06", "BR-DEC-09", "BR-DEC-10", "BR-DEC-11", "BR-DEC-12",
	"BR-DEC-13", "BR-DEC-14", "BR-DEC-15", "BR-DEC-16", "BR-DEC-17", "BR-DEC-18",
	"BR-DEC-19", "BR-DEC-20", "BR-DEC-23", "BR-DEC-24", "BR-DEC-25", "BR-DEC-27",
	"BR-DEC-28", "BR-E-01", "BR-E-02", "BR-E-03", "BR-E-04", "BR-E-05",
	"BR-E-06", "BR-E-07", "BR-E-08", "BR-E-09", "BR-E-10", "BR-G-01",
	"BR-G-02", "BR-G-03", "BR-G-04", "BR-G-05", "BR-G-06", "BR-G-07",
	"BR-G-08", "BR-G-09", "BR-G-10", "BR-IC-01", "BR-IC-02", "BR-IC-03",
	"BR-IC-04", "BR-IC-05", "BR-IC-06", "BR-IC-07", "BR-IC-08", "BR-IC-09",
	"BR-IC-10", "BR-IC-11", "BR-IC-12", "BR-O-01", "BR-O-02", "BR-O-03",
	"BR-O-04", "BR-O-05", "BR-O-06", "BR-O-07", "BR-O-08", "BR-O-09",
	"BR-O-10", "BR-O-11", "BR-O-12", "BR-O-13", "BR-O-14", "BR-S-01",
	"BR-S-02", "BR-S-03", "BR-S-04", "BR-S-05", "BR-S-06", "BR-S-07",
	"BR-S-08", "BR-S-09", "BR-S-10", "BR-Z-01", "BR-Z-02", "BR-Z-03",
	"BR-Z-04", "BR-Z-05", "BR-Z-06", "BR-Z-07", "BR-Z-08", "BR-Z-09",
	"BR-Z-10",
}

// ValidationRules documents exactly which rules ValidateEN16931 implements.
//
// It is part of the result rather than a comment, because "validated" without
// the list of what was checked tells the reader nothing they can act on.
//
// The list lives next to the standard's own inventory on purpose: two tests keep
// it honest. One fails if Buchfink claims a rule the standard does not define,
// the other if a rule is claimed here without a check in en16931.go that reports
// it. A claim nobody can check is worth as little as no claim at all.
func ValidationRules() []string {
	rules := []string{
		"BR-01", "BR-02", "BR-03", "BR-04", "BR-05", "BR-06", "BR-07", "BR-08",
		"BR-09", "BR-10", "BR-11", "BR-12", "BR-13", "BR-14", "BR-15", "BR-16",
		"BR-21", "BR-22", "BR-24", "BR-25", "BR-26",
		"BR-31", "BR-32", "BR-33", "BR-36", "BR-37", "BR-38",
		"BR-41", "BR-42", "BR-43", "BR-44",
		"BR-45", "BR-46", "BR-47", "BR-48",
		"BR-CL-03", "BR-CL-04", "BR-CL-05", "BR-CL-14", "BR-CL-18",
		"BR-CO-04", "BR-CO-09", "BR-CO-10", "BR-CO-11", "BR-CO-12", "BR-CO-13",
		"BR-CO-14", "BR-CO-15", "BR-CO-16", "BR-CO-17", "BR-CO-18", "BR-CO-19",
		"BR-CO-26",
		"BR-DEC-01", "BR-DEC-02", "BR-DEC-05", "BR-DEC-06", "BR-DEC-09",
		"BR-DEC-10", "BR-DEC-11", "BR-DEC-12", "BR-DEC-13", "BR-DEC-14",
		"BR-DEC-15",
		"BR-DEC-16", "BR-DEC-17", "BR-DEC-18", "BR-DEC-19", "BR-DEC-20",
		"BR-DEC-23", "BR-DEC-24", "BR-DEC-25", "BR-DEC-27", "BR-DEC-28",
		"BR-IC-11", "BR-IC-12",
		"BR-O-11", "BR-O-12", "BR-O-13", "BR-O-14",
	}
	// Die Kategorie-Familien haben alle denselben Zuschnitt, deshalb entsteht
	// ihr Anteil an der Liste aus derselben Tabelle, aus der auch die Prüfung
	// entsteht. Eine getippte Liste liefe früher oder später auseinander.
	for _, spec := range categorySpecs {
		for _, n := range []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "10"} {
			rules = append(rules, "BR-"+spec.family+"-"+n)
		}
	}
	sort.Strings(rules)
	return rules
}

// EN16931RuleCount is the number of business rules the standard defines for CII.
func EN16931RuleCount() int { return len(en16931BusinessRules) }

// EN16931AllRules returns every rule identifier of the standard.
func EN16931AllRules() []string {
	out := make([]string, len(en16931BusinessRules))
	copy(out, en16931BusinessRules)
	return out
}

// EN16931UncheckedRules returns the rules Buchfink does not check.
//
// It is exported on purpose. A user who needs to know whether a specific rule
// was looked at can ask, instead of inferring it from a coverage percentage.
func EN16931UncheckedRules() []string {
	checked := map[string]bool{}
	for _, r := range ValidationRules() {
		checked[r] = true
	}
	var out []string
	for _, r := range en16931BusinessRules {
		if !checked[r] {
			out = append(out, r)
		}
	}
	return out
}
