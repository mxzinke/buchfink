package domain

import "testing"

// Der Berichtigungszeitraum läuft ab der erstmaligen Verwendung und endet mit
// einem ganzen Kalendermonat (§ 15a Abs. 1 UStG i. V. m. § 45 UStDV). Bei einem
// Zugang mitten im Jahr reicht er deshalb ins sechste Kalenderjahr.
func TestCorrectionPeriodEndFollowsParagraph45(t *testing.T) {
	cases := []struct {
		acquisition string
		years       int
		want        string
		why         string
	}{
		{"2026-01-15", 5, "2030-12-31",
			"der Zeitraum endet am 14.01.2031, vor dem 16. — der Januar 2031 bleibt außen vor"},
		{"2026-12-20", 5, "2031-12-31",
			"der Zeitraum endet am 19.12.2031, nach dem 15. — der Dezember 2031 zählt voll"},
		{"2026-07-01", 5, "2031-06-30",
			"der Zeitraum endet am 30.06.2031, nach dem 15. — der Juni 2031 zählt voll"},
		{"2026-01-02", 5, "2030-12-31",
			"der Zeitraum endet am 01.01.2031, vor dem 16."},
		{"2026-06-16", 10, "2036-05-31",
			"zehn Jahre für ein Grundstück, Ende am 15.06.2036 — der Juni bleibt außen vor"},
		{"2026-06-17", 10, "2036-06-30",
			"Ende am 16.06.2036, nach dem 15. — der Juni zählt voll"},
	}
	for _, c := range cases {
		got, err := CorrectionPeriodEndDate(c.acquisition, c.years)
		if err != nil {
			t.Fatalf("Zeitraum ab %s: %v", c.acquisition, err)
		}
		if got != c.want {
			t.Errorf("Zugang %s, %d Jahre: Ende %s — erwartet %s (%s)",
				c.acquisition, c.years, got, c.want, c.why)
		}
	}

	if _, err := CorrectionPeriodEndDate("2026-13-01", 5); err == nil {
		t.Error("ein unmögliches Datum muss auffallen")
	}
	if _, err := CorrectionPeriodEndDate("2026-01-15", 0); err == nil {
		t.Error("ohne Zeitraum lässt sich kein Ende bestimmen")
	}
}

// Der Anteil eines Jahres bemisst sich nach den Monaten, die in den Zeitraum
// fallen — im Anfangs- und im Schlussjahr weniger als zwölf.
func TestMonthsInWindowWeighsTheFirstAndLastYear(t *testing.T) {
	c := &InputTaxCorrection{
		AcquisitionDate: "2026-07-01", CorrectionPeriodYears: 5,
		PeriodEnd: "2031-06-30", FirstFiscalYear: 2026, LastFiscalYear: 2031,
	}
	cases := []struct {
		from, to string
		want     int
	}{
		{"2026-01-01", "2026-12-31", 6},  // Juli bis Dezember
		{"2027-01-01", "2027-12-31", 12}, // volles Jahr
		{"2031-01-01", "2031-12-31", 6},  // Januar bis Juni
		{"2032-01-01", "2032-12-31", 0},  // nach dem Zeitraum
		{"2025-01-01", "2025-12-31", 0},  // davor
		{"2026-03-01", "2026-12-31", 6},  // Rumpfgeschäftsjahr, ganz im Zeitraum
	}
	for _, tc := range cases {
		if got := c.MonthsInWindow(tc.from, tc.to); got != tc.want {
			t.Errorf("Monate in %s bis %s: %d — erwartet %d", tc.from, tc.to, got, tc.want)
		}
	}

	// Ein Eintrag aus der Zeit vor dem gespeicherten Ende leitet es ab, statt
	// den Zeitraum still um ein Jahr zu kürzen.
	legacy := &InputTaxCorrection{AcquisitionDate: "2026-07-01", CorrectionPeriodYears: 5}
	if got := legacy.PeriodEndDate(); got != "2031-06-30" {
		t.Errorf("abgeleitetes Ende %s — erwartet 2031-06-30", got)
	}
	if got := legacy.MonthsInWindow("2031-01-01", "2031-12-31"); got != 6 {
		t.Errorf("Monate 2031 ohne gespeichertes Ende: %d — erwartet 6", got)
	}
}

// Das Jahr der erstmaligen Verwendung und das letzte Jahr des Zeitraums
// gehören beide dazu.
func TestCoversYearIncludesFirstAndLastYear(t *testing.T) {
	c := &InputTaxCorrection{FirstFiscalYear: 2026, LastFiscalYear: 2031}
	for _, year := range []int{2026, 2027, 2031} {
		if !c.CoversYear(year) {
			t.Errorf("das Jahr %d liegt im Berichtigungszeitraum", year)
		}
	}
	for _, year := range []int{2025, 2032} {
		if c.CoversYear(year) {
			t.Errorf("das Jahr %d liegt außerhalb des Berichtigungszeitraums", year)
		}
	}
}
