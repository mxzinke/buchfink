package domain

import "testing"

func TestCheckDateIsStrict(t *testing.T) {
	for _, valid := range []string{"2026-01-01", "2028-02-29", "2026-12-31"} {
		if err := CheckDate("Datum", valid); err != nil {
			t.Errorf("%s ist ein Kalenderdatum: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "2026-13-45", "2026-02-29", "2026-2-3", "01.03.2026", "2026-03-01T00:00:00Z", "2026-03-01 "} {
		if err := CheckDate("Datum", invalid); err == nil {
			t.Errorf("%q darf nicht als Datum durchgehen", invalid)
		}
	}
}

func TestFiscalYearFirstDayMatchesFiscalYearOfDate(t *testing.T) {
	for _, start := range []int{1, 4, 7, 10} {
		first := FiscalYearFirstDay(2026, start)
		if got := GetFiscalYearForDate(first, start); got != 2026 {
			t.Errorf("Beginn %d: %s liegt im Geschäftsjahr %d, erwartet 2026", start, first, got)
		}
	}
}
