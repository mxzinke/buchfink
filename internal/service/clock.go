package service

import (
	"time"

	"github.com/buchfink/buchfink/internal/calendar"
)

type serviceClock struct{ now func() time.Time }

// SetClock setzt die Uhr vor der Verwendung des Dienstes.
func (c *serviceClock) SetClock(now func() time.Time) { c.now = now }
func (c *serviceClock) today() string                 { return calendar.Today(c.now) }

func (c *serviceClock) instant() time.Time {
	if c.now == nil {
		return time.Now().UTC()
	}
	return c.now().UTC()
}
