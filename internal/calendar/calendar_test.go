package calendar

import (
	"testing"
	"time"
)

func TestGermanDayIndependentOfSourceZone(t *testing.T) {
	for _, tc := range []struct{ instant, day string }{
		{"2026-12-31T23:30:00Z", "2027-01-01"},
		{"2026-06-30T22:30:00Z", "2026-07-01"},
		{"2026-03-29T00:30:00Z", "2026-03-29"},
		{"2026-03-29T01:30:00Z", "2026-03-29"},
		{"2026-10-25T00:45:00Z", "2026-10-25"},
		{"2026-10-25T01:15:00Z", "2026-10-25"},
	} {
		at, _ := time.Parse(time.RFC3339, tc.instant)
		for _, zone := range []string{"UTC", "Pacific/Kiritimati", "America/Los_Angeles", "Europe/Berlin"} {
			loc, err := time.LoadLocation(zone)
			if err != nil {
				t.Fatal(err)
			}
			if got := Today(func() time.Time { return at.In(loc) }); got != tc.day {
				t.Errorf("%s in %s: %s, erwartet %s", tc.instant, zone, got, tc.day)
			}
		}
	}
}
