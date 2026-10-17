package calendar

import "time"

// Layout ist das Datumsformat der Fristen (JJJJ-MM-TT).
const Layout = "2006-01-02"

// Parse liest ein Datum JJJJ-MM-TT.
func Parse(iso string) (time.Time, error) { return time.Parse(Layout, iso) }

// Format schreibt ein Datum als JJJJ-MM-TT.
func Format(t time.Time) string { return t.Format(Layout) }

// Date ist ein Kalendertag ohne Uhrzeit.
func Date(year int, month time.Month, day int) time.Time { return date(year, month, day) }

// EndOfMonth ist der letzte Tag eines Monats.
func EndOfMonth(year int, month time.Month) time.Time {
	return date(year, month+1, 1).AddDate(0, 0, -1)
}

// AddMonths verschiebt einen Tag um Monate nach § 188 Abs. 2 und 3 BGB, auf
// die § 108 Abs. 1 AO verweist: die Frist endet mit dem Tag des letzten
// Monats, der durch seine Zahl dem Anfangstag entspricht; fehlt dieser Tag,
// mit dem letzten Tag des Monats.
func AddMonths(t time.Time, months int) time.Time {
	first := date(t.Year(), t.Month(), 1).AddDate(0, months, 0)
	last := EndOfMonth(first.Year(), first.Month())
	if t.Day() > last.Day() {
		return last
	}
	return date(first.Year(), first.Month(), t.Day())
}

// EndOfMonthsAfter ist das Ende des n-ten Monats nach dem Monat, in dem t
// liegt: „sieben Monate nach Ablauf des Kalenderjahres" ist für den 31.12.
// das Ende des Juli.
func EndOfMonthsAfter(t time.Time, months int) time.Time {
	m := date(t.Year(), t.Month(), 1).AddDate(0, months, 0)
	return EndOfMonth(m.Year(), m.Month())
}

// Shift wendet § 108 Abs. 3 AO auf ein gesetzliches Fristende an. Es liefert
// das verschobene Ende und den Grund der Verschiebung; der Grund ist leer,
// wenn der Tag ein Werktag ist.
func Shift(statutory time.Time, state string) (time.Time, string) {
	reason := NonWorkdayReason(statutory, state)
	if reason == "" {
		return statutory, ""
	}
	return NextWorkday(statutory, state), reason
}
