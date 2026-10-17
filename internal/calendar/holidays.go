// Package calendar rechnet Feiertage und Fristenden.
//
// Fällt das Ende einer Frist auf einen Sonnabend, Sonntag oder gesetzlichen
// Feiertag, endet sie mit Ablauf des nächsten Werktags (§ 108 Abs. 3 AO).
// Gesetzliche Feiertage sind Landesrecht; maßgebend ist das Land am Sitz.
//
// Bundesweit gelten Neujahr, Karfreitag, Ostermontag, 1. Mai, Christi
// Himmelfahrt, Pfingstmontag, 3. Oktober (Art. 2 Abs. 2 Einigungsvertrag) und
// die beiden Weihnachtstage. Die übrigen Tage folgen den Feiertagsgesetzen der
// Länder, gelesen am Wortlaut für Bayern (Art. 1 FTG), Brandenburg (§ 2 FTG)
// und Sachsen (§ 1 SächsSFG), für die übrigen Länder nach der Übersicht in
// „Feiertag (Deutschland)" (Wikipedia, Stand Oktober 2026) mit Verweis auf die
// Landesgesetze.
//
// Feiertage, die nur in einzelnen Gemeinden gelten (Fronleichnam in Teilen
// Sachsens und Thüringens, Mariä Himmelfahrt in Bayern, Friedensfest in
// Augsburg), kennt das Paket nicht. Eine Frist erscheint dort einen Tag zu
// früh, nie zu spät.
package calendar

import (
	"sort"
	"time"
)

// Die Bundesländer als Kürzel.
const (
	BadenWuerttemberg     = "BW"
	Bayern                = "BY"
	Berlin                = "BE"
	Brandenburg           = "BB"
	Bremen                = "HB"
	Hamburg               = "HH"
	Hessen                = "HE"
	MecklenburgVorpommern = "MV"
	Niedersachsen         = "NI"
	NordrheinWestfalen    = "NW"
	RheinlandPfalz        = "RP"
	Saarland              = "SL"
	Sachsen               = "SN"
	SachsenAnhalt         = "ST"
	SchleswigHolstein     = "SH"
	Thueringen            = "TH"
)

// State ist ein Bundesland mit Kürzel und Namen.
type State struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var states = []State{
	{BadenWuerttemberg, "Baden-Württemberg"}, {Bayern, "Bayern"}, {Berlin, "Berlin"},
	{Brandenburg, "Brandenburg"}, {Bremen, "Bremen"}, {Hamburg, "Hamburg"}, {Hessen, "Hessen"},
	{MecklenburgVorpommern, "Mecklenburg-Vorpommern"}, {Niedersachsen, "Niedersachsen"},
	{NordrheinWestfalen, "Nordrhein-Westfalen"}, {RheinlandPfalz, "Rheinland-Pfalz"},
	{Saarland, "Saarland"}, {Sachsen, "Sachsen"}, {SachsenAnhalt, "Sachsen-Anhalt"},
	{SchleswigHolstein, "Schleswig-Holstein"}, {Thueringen, "Thüringen"},
}

// States liefert die Bundesländer in der üblichen Reihenfolge.
func States() []State {
	out := make([]State, len(states))
	copy(out, states)
	return out
}

// ValidState meldet, ob code ein Bundesland ist.
func ValidState(code string) bool {
	for _, s := range states {
		if s.Code == code {
			return true
		}
	}
	return false
}

// StateName ist der Name eines Bundeslandes, sonst das Kürzel selbst.
func StateName(code string) string {
	for _, s := range states {
		if s.Code == code {
			return s.Name
		}
	}
	return code
}

// Holiday ist ein gesetzlicher Feiertag.
type Holiday struct {
	Date time.Time
	Name string
}

// Easter ist der Ostersonntag des gregorianischen Kalenders (Gaußsche
// Osterformel in der Fassung von Meeus/Jones/Butcher).
func Easter(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := (h+l-7*m+114)%31 + 1
	return date(year, time.Month(month), day)
}

// repentanceDay ist der Buß- und Bettag: der Mittwoch vor dem 23. November.
func repentanceDay(year int) time.Time {
	d := date(year, time.November, 22)
	for d.Weekday() != time.Wednesday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func in(state string, codes ...string) bool {
	for _, c := range codes {
		if c == state {
			return true
		}
	}
	return false
}

// Holidays liefert die gesetzlichen Feiertage eines Jahres in einem Land,
// nach Datum sortiert. Ohne Land (leeres Kürzel) sind es die bundesweiten.
func Holidays(year int, state string) []Holiday {
	easter := Easter(year)
	out := []Holiday{
		{date(year, time.January, 1), "Neujahr"},
		{easter.AddDate(0, 0, -2), "Karfreitag"},
		{easter.AddDate(0, 0, 1), "Ostermontag"},
		{date(year, time.May, 1), "Tag der Arbeit"},
		{easter.AddDate(0, 0, 39), "Christi Himmelfahrt"},
		{easter.AddDate(0, 0, 50), "Pfingstmontag"},
		{date(year, time.October, 3), "Tag der Deutschen Einheit"},
		{date(year, time.December, 25), "1. Weihnachtstag"},
		{date(year, time.December, 26), "2. Weihnachtstag"},
	}
	add := func(when bool, d time.Time, name string) {
		if when {
			out = append(out, Holiday{d, name})
		}
	}
	add(in(state, BadenWuerttemberg, Bayern, SachsenAnhalt), date(year, time.January, 6), "Heilige Drei Könige")
	add((state == Berlin && year >= 2019) || (state == MecklenburgVorpommern && year >= 2023),
		date(year, time.March, 8), "Internationaler Frauentag")
	add(state == Berlin && (year == 2020 || year == 2025), date(year, time.May, 8), "Tag der Befreiung")
	add(in(state, BadenWuerttemberg, Bayern, Hessen, NordrheinWestfalen, RheinlandPfalz, Saarland),
		easter.AddDate(0, 0, 60), "Fronleichnam")
	add(state == Saarland, date(year, time.August, 15), "Mariä Himmelfahrt")
	add(state == Thueringen && year >= 2019, date(year, time.September, 20), "Weltkindertag")
	// 2017 war der Reformationstag einmalig in allen Ländern Feiertag; Bremen,
	// Hamburg, Niedersachsen und Schleswig-Holstein führen ihn seit 2018.
	add(in(state, Brandenburg, MecklenburgVorpommern, Sachsen, SachsenAnhalt, Thueringen) ||
		(in(state, Bremen, Hamburg, Niedersachsen, SchleswigHolstein) && year >= 2018) ||
		(year == 2017 && state != ""),
		date(year, time.October, 31), "Reformationstag")
	add(in(state, BadenWuerttemberg, Bayern, NordrheinWestfalen, RheinlandPfalz, Saarland),
		date(year, time.November, 1), "Allerheiligen")
	add(state == Sachsen, repentanceDay(year), "Buß- und Bettag")

	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

// HolidayName nennt den Feiertag an einem Tag, falls einer ist.
func HolidayName(day time.Time, state string) (string, bool) {
	day = date(day.Year(), day.Month(), day.Day())
	for _, h := range Holidays(day.Year(), state) {
		if h.Date.Equal(day) {
			return h.Name, true
		}
	}
	return "", false
}

// IsWorkday meldet, ob ein Tag weder Sonnabend noch Sonntag noch gesetzlicher
// Feiertag ist.
func IsWorkday(day time.Time, state string) bool {
	if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	_, holiday := HolidayName(day, state)
	return !holiday
}

// NonWorkdayReason nennt, warum ein Tag kein Werktag ist: „Sonnabend",
// „Sonntag" oder der Name des Feiertags. Leer heißt Werktag.
func NonWorkdayReason(day time.Time, state string) string {
	switch day.Weekday() {
	case time.Saturday:
		return "Sonnabend"
	case time.Sunday:
		return "Sonntag"
	}
	name, _ := HolidayName(day, state)
	return name
}

// NextWorkday ist das Fristende nach § 108 Abs. 3 AO: der Tag selbst, wenn er
// ein Werktag ist, sonst der nächstfolgende Werktag.
func NextWorkday(day time.Time, state string) time.Time {
	day = date(day.Year(), day.Month(), day.Day())
	for !IsWorkday(day, state) {
		day = day.AddDate(0, 0, 1)
	}
	return day
}
