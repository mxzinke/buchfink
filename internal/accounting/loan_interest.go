package accounting

import (
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// LoanMovement ist eine Bewegung des Darlehensbestands: Auszahlung erhöht,
// Tilgung mindert ihn.
type LoanMovement struct {
	Date   string
	Amount domain.Cents
}

// LoanInterestSegment ist ein Zeitraum mit gleichem Bestand.
type LoanInterestSegment struct {
	From    string       `json:"from"`
	To      string       `json:"to"`
	Balance domain.Cents `json:"balance"`
	Days    int          `json:"days"`
}

// LoanInterestResult ist die Zinsrechnung eines Zeitraums.
type LoanInterestResult struct {
	Amount   domain.Cents          `json:"amount"`
	YearDays int                   `json:"yearDays"`
	Segments []LoanInterestSegment `json:"segments"`
}

// LoanInterest rechnet die Zinsen eines Darlehens für den Zeitraum from bis to,
// beide Tage eingeschlossen, auf den taggenauen Bestand. Eine Bewegung zählt ab
// ihrem Tag: der Auszahlungstag wird verzinst, der Rückzahlungstag nicht mehr.
// Gerundet wird einmal am Ende, kaufmännisch auf den Cent.
func LoanInterest(movements []LoanMovement, rateBP int, method domain.DayCount, from, to string) (LoanInterestResult, error) {
	start, err := time.Parse(domain.DateLayout, from)
	if err != nil {
		return LoanInterestResult{}, fmt.Errorf("Zinsbeginn %q ist kein gültiges Datum", from)
	}
	last, err := time.Parse(domain.DateLayout, to)
	if err != nil {
		return LoanInterestResult{}, fmt.Errorf("Zinsende %q ist kein gültiges Datum", to)
	}
	if last.Before(start) {
		return LoanInterestResult{}, fmt.Errorf("das Zinsende %s liegt vor dem Beginn %s",
			domain.GermanDate(to), domain.GermanDate(from))
	}
	if rateBP < 0 {
		return LoanInterestResult{}, fmt.Errorf("negativer Zinssatz")
	}
	yearDays := 365
	switch method {
	case domain.DayCountAct365:
	case domain.DayCountAct360, domain.DayCount30E360:
		yearDays = 360
	default:
		return LoanInterestResult{}, fmt.Errorf("unbekannte Zinsmethode %q", method)
	}

	sorted := append([]LoanMovement(nil), movements...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })

	end := last.AddDate(0, 0, 1)
	var balance domain.Cents
	i := 0
	for ; i < len(sorted) && sorted[i].Date <= from; i++ {
		balance += sorted[i].Amount
	}

	result := LoanInterestResult{YearDays: yearDays, Segments: make([]LoanInterestSegment, 0, 2)}
	var numerator int64
	cursor := start
	for cursor.Before(end) {
		next := end
		if i < len(sorted) {
			if change, err := time.Parse(domain.DateLayout, sorted[i].Date); err == nil && change.Before(end) {
				next = change
			}
		}
		days := dayCount(method, cursor, next)
		if days > 0 {
			result.Segments = append(result.Segments, LoanInterestSegment{
				From: cursor.Format(domain.DateLayout), To: next.AddDate(0, 0, -1).Format(domain.DateLayout),
				Balance: balance, Days: days,
			})
			if balance > 0 {
				numerator += int64(balance) * int64(rateBP) * int64(days)
			}
		}
		cursor = next
		for ; i < len(sorted) && sorted[i].Date <= cursor.Format(domain.DateLayout); i++ {
			balance += sorted[i].Amount
		}
	}
	denominator := int64(10000) * int64(yearDays)
	result.Amount = domain.Cents((numerator + denominator/2) / denominator)
	return result, nil
}

// dayCount zählt die Zinstage von a (eingeschlossen) bis b (ausgeschlossen).
func dayCount(method domain.DayCount, a, b time.Time) int {
	if method != domain.DayCount30E360 {
		return int(b.Sub(a).Hours() / 24)
	}
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return 360*(y2-y1) + 30*(int(m2)-int(m1)) + min(d2, 30) - min(d1, 30)
}
