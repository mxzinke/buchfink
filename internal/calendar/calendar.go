// Package calendar bestimmt Kalendertage der deutschen Buchführung.
package calendar

import (
	"time"
	_ "time/tzdata"
)

var location = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return loc
}()

// Location ist die Zeitzone für Buchungstage und Fristen.
func Location() *time.Location { return location }

// Day wandelt einen Zeitpunkt vor der Datumsbildung nach Deutschland um.
func Day(at time.Time) string { return at.In(location).Format("2006-01-02") }

// Today liefert den deutschen Kalendertag aus der übergebenen Uhr.
func Today(now func() time.Time) string {
	if now == nil {
		now = time.Now
	}
	return Day(now())
}
