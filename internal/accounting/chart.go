package accounting

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// Chart resolves account numbers against the SKR04 catalog.
//
// The catalog lists 1.612 concrete accounts plus 243 ranges such as
// "4400-4409 Erlöse 19 % USt". A range is a compact notation for ten usable
// accounts, not an account itself: a company books to 4400 or 4407, never to the
// literal string "4400-4409". Resolving that distinction in one place keeps it
// out of the query layer, where it previously lived as string comparisons on
// account numbers.
type Chart struct {
	concrete map[string]domain.Account
	ranges   []chartRange
}

type chartRange struct {
	start, end int
	account    domain.Account
}

// NewChart indexes a chart of accounts for lookup.
func NewChart(accounts []domain.Account) *Chart {
	c := &Chart{concrete: make(map[string]domain.Account, len(accounts))}

	for _, a := range accounts {
		if a.IsRange && a.RangeStart != "" && a.RangeEnd != "" {
			start, err1 := strconv.Atoi(a.RangeStart)
			end, err2 := strconv.Atoi(a.RangeEnd)
			if err1 == nil && err2 == nil && start <= end {
				c.ranges = append(c.ranges, chartRange{start: start, end: end, account: a})
				continue
			}
		}
		c.concrete[a.Number] = a
	}

	sort.Slice(c.ranges, func(i, j int) bool { return c.ranges[i].start < c.ranges[j].start })
	return c
}

// Lookup resolves an account number to its catalog classification. Numbers
// covered by a range inherit the range's name and classification but keep their
// own number.
func (c *Chart) Lookup(number string) (domain.Account, bool) {
	if a, ok := c.concrete[number]; ok {
		return a, true
	}

	n, err := strconv.Atoi(number)
	if err != nil {
		return domain.Account{}, false
	}

	idx := sort.Search(len(c.ranges), func(i int) bool { return c.ranges[i].end >= n })
	if idx < len(c.ranges) && c.ranges[idx].start <= n && n <= c.ranges[idx].end {
		a := c.ranges[idx].account
		a.Number = number
		a.IsRange = false
		a.RangeStart = ""
		a.RangeEnd = ""
		return a, true
	}

	return domain.Account{}, false
}

// Name returns a display name for an account number, falling back to the number
// itself for accounts outside the catalog (e.g. Personenkonten).
func (c *Chart) Name(number string) string {
	if a, ok := c.Lookup(number); ok {
		return a.Name
	}
	return number
}

// EnsurePostable reports why an account may not be booked to, or nil if it may.
func (c *Chart) EnsurePostable(number string) error {
	if number == "" {
		return fmt.Errorf("Konto fehlt")
	}

	// A number written as a range is the catalog's grouping notation and never a
	// bookable account.
	if _, err := strconv.Atoi(number); err != nil {
		return fmt.Errorf("Konto %q ist keine gültige Kontonummer", number)
	}

	acc, ok := c.Lookup(number)
	if !ok {
		return fmt.Errorf("Konto %s ist im SKR04 nicht vorhanden", number)
	}
	if acc.IsReserved {
		return fmt.Errorf("Konto %s ist im SKR04 als reserviert gekennzeichnet und darf nicht bebucht werden", number)
	}
	if !acc.IsActive {
		return fmt.Errorf("Konto %s (%s) ist deaktiviert", number, acc.Name)
	}

	// Kontenklasse 8 is held free for future DATEV use in SKR04 and carries no
	// meaning today. It is a trap worth naming: in SKR03 class 8 holds the
	// revenue accounts, so 8400 looks like "Erlöse 19 %" to anyone coming from
	// there — in SKR04 revenue is 4400.
	if acc.Kontenklasse == 8 {
		return fmt.Errorf(
			"Konto %s liegt in Kontenklasse 8, die im SKR04 für künftige DATEV-Verwendung freigehalten wird. "+
				"Erlöse liegen im SKR04 in Kontenklasse 4 (z. B. 4400 Erlöse 19 %% USt) – Klasse 8 ist der SKR03-Erlösbereich",
			number,
		)
	}

	return nil
}

// All returns every concrete and range entry, for catalog display.
func (c *Chart) All() []domain.Account {
	out := make([]domain.Account, 0, len(c.concrete)+len(c.ranges))
	for _, a := range c.concrete {
		out = append(out, a)
	}
	for _, r := range c.ranges {
		out = append(out, r.account)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// Covers reports whether a catalog entry (concrete or range) covers a given
// account number. Used to fold booked turnover into the catalog view.
func Covers(entry domain.Account, number string) bool {
	if entry.Number == number {
		return true
	}
	if !entry.IsRange || entry.RangeStart == "" || entry.RangeEnd == "" {
		return false
	}
	n, err := strconv.Atoi(number)
	if err != nil {
		return false
	}
	start, err1 := strconv.Atoi(entry.RangeStart)
	end, err2 := strconv.Atoi(entry.RangeEnd)
	if err1 != nil || err2 != nil {
		return false
	}
	return n >= start && n <= end
}
