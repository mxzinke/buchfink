package einvoice

// checkedRules is what Validate implements.
//
// It sits next to the standard's own inventory on purpose, because two tests
// keep it honest: one fails if Buchfink claims a rule EN 16931 does not define,
// the other if a rule is claimed here without a check anywhere that reports it.
// A claim nobody can verify is worth as little as no claim.
//
// The four rules the standard's own reference implementation carries as
// `true()` — BR-CO-05 to BR-CO-08, which ask whether a reason code and a free
// text reason mean the same thing — are listed as checked because Buchfink does
// exactly what the norm does with them: nothing. Claiming otherwise in either
// direction would misstate the coverage.
var checkedRules = buildCheckedRules()

func buildCheckedRules() map[string]bool {
	rules := map[string]bool{}
	add := func(ids ...string) {
		for _, id := range ids {
			rules[id] = true
		}
	}

	// Dokument, Beteiligte, Positionen
	add("BR-01", "BR-02", "BR-03", "BR-04", "BR-05", "BR-06", "BR-07", "BR-08",
		"BR-09", "BR-10", "BR-11", "BR-12", "BR-13", "BR-14", "BR-15", "BR-16",
		"BR-17", "BR-18", "BR-19", "BR-20", "BR-21", "BR-22", "BR-23", "BR-24",
		"BR-25", "BR-26", "BR-27", "BR-28", "BR-29", "BR-30")
	// Nachlässe, Zuschläge, Zahlung, Unterlagen, Bezüge, Artikel
	add("BR-31", "BR-32", "BR-33", "BR-36", "BR-37", "BR-38", "BR-41", "BR-42",
		"BR-43", "BR-44", "BR-45", "BR-46", "BR-47", "BR-48", "BR-49", "BR-50",
		"BR-51", "BR-52", "BR-53", "BR-54", "BR-55", "BR-56", "BR-57", "BR-61",
		"BR-62", "BR-63", "BR-64", "BR-65")
	// Rechenregeln und Zusammenhänge
	add("BR-CO-03", "BR-CO-04", "BR-CO-05", "BR-CO-06", "BR-CO-07", "BR-CO-08",
		"BR-CO-09", "BR-CO-10", "BR-CO-11", "BR-CO-12", "BR-CO-13", "BR-CO-14",
		"BR-CO-15", "BR-CO-16", "BR-CO-17", "BR-CO-18", "BR-CO-19", "BR-CO-20",
		"BR-CO-21", "BR-CO-22", "BR-CO-23", "BR-CO-24", "BR-CO-26")
	// Nachkommastellen
	add("BR-DEC-01", "BR-DEC-02", "BR-DEC-05", "BR-DEC-06", "BR-DEC-09",
		"BR-DEC-10", "BR-DEC-11", "BR-DEC-12", "BR-DEC-13", "BR-DEC-14",
		"BR-DEC-15", "BR-DEC-16", "BR-DEC-17", "BR-DEC-18", "BR-DEC-19",
		"BR-DEC-20", "BR-DEC-23", "BR-DEC-24", "BR-DEC-25", "BR-DEC-27",
		"BR-DEC-28")
	// Codelisten
	add("BR-CL-01", "BR-CL-03", "BR-CL-04", "BR-CL-05", "BR-CL-06", "BR-CL-07",
		"BR-CL-08", "BR-CL-10", "BR-CL-11", "BR-CL-13", "BR-CL-14", "BR-CL-15",
		"BR-CL-16", "BR-CL-17", "BR-CL-18", "BR-CL-19", "BR-CL-20", "BR-CL-21",
		"BR-CL-22", "BR-CL-23", "BR-CL-24", "BR-CL-25", "BR-CL-26")
	// Split Payment
	add("BR-B-01", "BR-B-02")
	// Die Steuerkategorie-Familien entstehen aus derselben Tabelle wie die
	// Prüfung. Eine getippte Liste liefe früher oder später auseinander.
	for _, spec := range categorySpecs {
		for _, n := range []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "10"} {
			rules["BR-"+spec.family+"-"+n] = true
		}
	}
	add("BR-IC-11", "BR-IC-12", "BR-O-11", "BR-O-12", "BR-O-13", "BR-O-14")

	return rules
}
