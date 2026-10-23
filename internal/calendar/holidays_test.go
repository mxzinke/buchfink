package calendar

import (
	"testing"
	"time"
)

func mustParse(t *testing.T, iso string) time.Time {
	t.Helper()
	d, err := Parse(iso)
	if err != nil {
		t.Fatalf("%s: %v", iso, err)
	}
	return d
}

// Ostersonntage nach der Tabelle der Osterdaten.
func TestEaster(t *testing.T) {
	for year, want := range map[int]string{
		2019: "2019-04-21", 2024: "2024-03-31", 2025: "2025-04-20", 2026: "2026-04-05",
		2027: "2027-03-28", 2028: "2028-04-16", 2029: "2029-04-01", 2038: "2038-04-25",
	} {
		if got := Format(Easter(year)); got != want {
			t.Errorf("Ostern %d = %s, erwartet %s", year, got, want)
		}
	}
}

// Bewegliche Feiertage hängen am Osterdatum: Karfreitag −2, Ostermontag +1,
// Himmelfahrt +39, Pfingstmontag +50, Fronleichnam +60.
func TestMovableHolidays2026(t *testing.T) {
	want := map[string]string{
		"2026-04-03": "Karfreitag",
		"2026-04-06": "Ostermontag",
		"2026-05-14": "Christi Himmelfahrt",
		"2026-05-25": "Pfingstmontag",
		"2026-06-04": "Fronleichnam",
		"2026-11-18": "Buß- und Bettag",
	}
	for iso, name := range want {
		state := Bayern
		if name == "Buß- und Bettag" {
			state = Sachsen
		}
		got, ok := HolidayName(mustParse(t, iso), state)
		if !ok || got != name {
			t.Errorf("%s in %s = %q, erwartet %q", iso, state, got, name)
		}
	}
}

// Landesfeiertage gelten nur im Land, und die jüngeren erst ab ihrem Jahr.
func TestStateHolidays(t *testing.T) {
	cases := []struct {
		iso, state string
		holiday    bool
	}{
		{"2026-01-06", Bayern, true},
		{"2026-01-06", NordrheinWestfalen, false},
		{"2026-06-04", NordrheinWestfalen, true},
		{"2026-06-04", Sachsen, false}, // nur in einzelnen Gemeinden
		{"2026-06-04", Berlin, false},
		{"2026-08-15", Saarland, true},
		{"2026-08-15", Bayern, false}, // nur in Gemeinden mit überwiegend katholischer Bevölkerung
		{"2026-10-31", Niedersachsen, true},
		{"2017-10-31", Bayern, true}, // einmalig 500 Jahre Reformation
		{"2016-10-31", Hamburg, false},
		{"2026-11-01", NordrheinWestfalen, true},
		{"2026-11-01", Hessen, false},
		{"2026-03-09", Berlin, false},
		{"2027-03-08", Berlin, true},
		{"2022-03-08", MecklenburgVorpommern, false},
		{"2023-03-08", MecklenburgVorpommern, true},
		{"2026-09-21", Thueringen, false},
		{"2027-09-20", Thueringen, true},
		{"2025-05-08", Berlin, true},
		{"2026-05-08", Berlin, false},
		{"2026-10-03", "", true},
		{"2026-10-31", "", false},
	}
	for _, c := range cases {
		_, got := HolidayName(mustParse(t, c.iso), c.state)
		if got != c.holiday {
			t.Errorf("%s in %q: Feiertag = %v, erwartet %v", c.iso, c.state, got, c.holiday)
		}
	}
}

// § 108 Abs. 3 AO: Sonnabend, Sonntag und Feiertag schieben das Fristende auf
// den nächsten Werktag.
func TestShiftToNextWorkday(t *testing.T) {
	cases := []struct {
		statutory, state, want, reason string
	}{
		// Erklärungsfrist 2026 ohne Berater: der 31.07.2027 ist ein Sonnabend.
		{"2027-07-31", Bayern, "2027-08-02", "Sonnabend"},
		// Berater-Frist 2025: der 28.02.2027 ist ein Sonntag.
		{"2027-02-28", Berlin, "2027-03-01", "Sonntag"},
		// Zusammenfassende Meldung November: 25.12. und 26.12. sind Feiertage,
		// der 26.12.2026 zugleich Sonnabend.
		{"2026-12-25", Hamburg, "2026-12-28", "1. Weihnachtstag"},
		// Voranmeldung April 2029 mit Fälligkeit an Christi Himmelfahrt.
		{"2029-05-10", Niedersachsen, "2029-05-11", "Christi Himmelfahrt"},
		// Reformationstag nur in den Ländern, die ihn führen.
		{"2028-10-31", Sachsen, "2028-11-01", "Reformationstag"},
		{"2028-10-31", Bayern, "2028-10-31", ""},
		// Reformationstag (Di) in Thüringen, Allerheiligen (Mi) in Bayern.
		{"2028-11-01", Bayern, "2028-11-02", "Allerheiligen"},
		{"2026-03-10", Bayern, "2026-03-10", ""},
	}
	for _, c := range cases {
		got, reason := Shift(mustParse(t, c.statutory), c.state)
		if Format(got) != c.want || reason != c.reason {
			t.Errorf("%s in %s → %s (%q), erwartet %s (%q)", c.statutory, c.state, Format(got), reason, c.want, c.reason)
		}
	}
}

// Ein langes Wochenende: Pfingstsonntag, Pfingstmontag in Bayern.
func TestNextWorkdaySkipsSeveralDays(t *testing.T) {
	// Sonnabend 23.05.2026, Sonntag, Pfingstmontag 25.05.2026.
	if got := Format(NextWorkday(mustParse(t, "2026-05-23"), Bayern)); got != "2026-05-26" {
		t.Errorf("nächster Werktag = %s, erwartet 2026-05-26", got)
	}
}

// § 188 Abs. 2 und 3 BGB: der Monat endet am Tag gleicher Zahl, fehlt er, am
// Monatsletzten.
func TestAddMonths(t *testing.T) {
	cases := map[string]string{
		"2026-03-15": "2026-04-15",
		"2026-01-31": "2026-02-28",
		"2028-01-31": "2028-02-29",
		"2026-12-31": "2027-01-31",
	}
	for from, want := range cases {
		if got := Format(AddMonths(mustParse(t, from), 1)); got != want {
			t.Errorf("%s + 1 Monat = %s, erwartet %s", from, got, want)
		}
	}
	if got := Format(EndOfMonthsAfter(mustParse(t, "2026-12-31"), 7)); got != "2027-07-31" {
		t.Errorf("sieben Monate nach Ablauf 2026 = %s, erwartet 2027-07-31", got)
	}
	if got := Format(EndOfMonthsAfter(mustParse(t, "2026-06-30"), 6)); got != "2026-12-31" {
		t.Errorf("sechs Monate nach dem 30.06. = %s, erwartet 2026-12-31", got)
	}
}

func TestStates(t *testing.T) {
	if len(States()) != 16 {
		t.Fatalf("%d Länder, erwartet 16", len(States()))
	}
	if !ValidState("NW") || ValidState("XX") || ValidState("") {
		t.Error("Prüfung der Kürzel stimmt nicht")
	}
	if StateName("TH") != "Thüringen" {
		t.Errorf("Name = %q", StateName("TH"))
	}
}
