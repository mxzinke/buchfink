package accounting

import (
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Verzug und Verzugszinsen (§§ 286, 288 BGB).
//
// Die Rechnung ist einfach und wird trotzdem regelmäßig falsch gemacht, weil
// drei Größen zusammenkommen, die sich unabhängig voneinander ändern: der
// Basiszinssatz wechselt halbjährlich, der Zuschlag hängt davon ab, ob ein
// Verbraucher beteiligt ist, und gezählt wird taggenau. Deshalb rechnet
// Buchfink es aus, statt eine Zahl zu schätzen — und deshalb steht die Rechnung
// hier und nicht im Dienst: sie ist eine Regel und keine Ablaufsteuerung.

// Die Zuschläge des § 288 BGB in Hundertsteln eines Prozentpunktes.
const (
	// DefaultInterestPointsConsumer sind die fünf Prozentpunkte des § 288
	// Abs. 1 Satz 2 BGB — der Regelfall, wenn ein Verbraucher beteiligt ist.
	DefaultInterestPointsConsumer = 500
	// DefaultInterestPointsBusiness sind die neun Prozentpunkte des § 288
	// Abs. 2 BGB. Sie gelten bei Entgeltforderungen aus Rechtsgeschäften, an
	// denen kein Verbraucher beteiligt ist.
	DefaultInterestPointsBusiness = 900
)

// DefaultInterestLumpSum ist die Pauschale des § 288 Abs. 5 BGB: 40 € bei einer
// Entgeltforderung gegen einen Schuldner, der kein Verbraucher ist. Sie wird auf
// einen geschuldeten Schadensersatz für Rechtsverfolgungskosten angerechnet und
// fällt je Forderung einmal an, nicht je Mahnung.
const DefaultInterestLumpSum = domain.Cents(4000)

// DefaultGraceDays ist die Frist des § 286 Abs. 3 Satz 1 BGB: dreißig Tage nach
// Fälligkeit und Zugang der Rechnung kommt der Schuldner auch ohne Mahnung in
// Verzug. Gegenüber einem Verbraucher gilt das nur, wenn er in der Rechnung auf
// diese Folge hingewiesen wurde (§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB) — der
// Hinweis gehört auf die Rechnung, nicht in diese Rechnung.
const DefaultGraceDays = 30

// DefaultInterestDaysPerYear ist der Nenner der Tageszinsformel.
//
// 365 und nicht 360: die Zinsmethode der Handelsbräuche (30/360) ist eine
// Vereinbarung, und für den gesetzlichen Verzugszins gibt es keine. Gerechnet
// wird deshalb mit den tatsächlichen Tagen über ein Jahr von 365 Tagen — auch im
// Schaltjahr, weil sonst zwei gleich lange Verzüge verschieden hohe Zinsen
// trügen, je nachdem, in welchem Jahr sie lagen.
const DefaultInterestDaysPerYear = 365

// Die datierte Tabelle des Basiszinssatzes selbst (BaseRatePeriod,
// defaultBaseRates, DefaultBaseRates) steht in tax_params.go, bei den übrigen
// Werten, die der Gesetzgeber bzw. die Bundesbank über die Zeit ändert. Hier
// steht, was mit ihr gerechnet wird.

// BaseRateAt liefert den Satz, der an einem Tag galt.
//
// Die Tabelle wird als aufsteigend sortiert vorausgesetzt; MergeBaseRates
// stellt das her. Vor dem ersten Eintrag gibt es keinen Satz — dann sagt die
// Funktion das, statt null zu liefern: ein Zins von null Prozent wäre eine
// Aussage, und sie wäre falsch.
func BaseRateAt(rates []BaseRatePeriod, date string) (BaseRatePeriod, error) {
	var found BaseRatePeriod
	ok := false
	for _, r := range rates {
		if r.ValidFrom <= date {
			found, ok = r, true
			continue
		}
		break
	}
	if !ok {
		return BaseRatePeriod{}, fmt.Errorf(
			"für den %s ist kein Basiszinssatz hinterlegt; er ist in den Einstellungen nachzutragen", date)
	}
	return found, nil
}

// MergeBaseRates legt die gepflegten Sätze über die hinterlegten und sortiert.
//
// Der gepflegte Satz gewinnt: er ist die Bekanntgabe, die jemand nachgetragen
// hat, und sie ist jünger als jede Auslieferung des Programms.
func MergeBaseRates(stored []BaseRatePeriod) []BaseRatePeriod {
	byDate := map[string]BaseRatePeriod{}
	for _, r := range DefaultBaseRates() {
		byDate[r.ValidFrom] = r
	}
	for _, r := range stored {
		if r.ValidFrom == "" {
			continue
		}
		byDate[r.ValidFrom] = r
	}
	out := make([]BaseRatePeriod, 0, len(byDate))
	for _, r := range byDate {
		out = append(out, r)
	}
	sortBaseRates(out)
	return out
}

func sortBaseRates(rates []BaseRatePeriod) {
	for i := 1; i < len(rates); i++ {
		for j := i; j > 0 && rates[j].ValidFrom < rates[j-1].ValidFrom; j-- {
			rates[j], rates[j-1] = rates[j-1], rates[j]
		}
	}
}

// DefaultInterestStart ist der erste Tag, für den Verzugszinsen laufen.
//
// § 286 Abs. 3 BGB lässt den Verzug mit Ablauf von dreißig Tagen nach
// Fälligkeit und Zugang der Rechnung eintreten; verzinst wird ab dem Tag
// danach. Der Zugang wird mit der Fälligkeit gleichgesetzt — Buchfink kennt den
// Zugangstag nicht, und die Fälligkeit setzt ihn ohnehin voraus.
func DefaultInterestStart(dueDate string) (string, error) {
	due, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return "", fmt.Errorf("%q ist kein gültiges Fälligkeitsdatum (erwartet JJJJ-MM-TT)", dueDate)
	}
	return due.AddDate(0, 0, DefaultGraceDays+1).Format("2006-01-02"), nil
}

// InterestSegment ist ein Abschnitt der Zinsrechnung mit einem Satz.
type InterestSegment struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
	// BasisPoints ist der Basiszinssatz des Abschnitts, TotalPoints der Satz
	// einschließlich des Zuschlags nach § 288 BGB.
	BasisPoints int          `json:"basisPoints"`
	TotalPoints int          `json:"totalPoints"`
	Amount      domain.Cents `json:"amount"`
}

// InterestResult ist die Zinsrechnung eines Postens.
type InterestResult struct {
	Principal domain.Cents      `json:"principal"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	Days      int               `json:"days"`
	Amount    domain.Cents      `json:"amount"`
	Segments  []InterestSegment `json:"segments"`
}

// EnsureLists ersetzt eine nicht belegte Abschnittsliste durch eine leere.
func (r *InterestResult) EnsureLists() {
	if r.Segments == nil {
		r.Segments = make([]InterestSegment, 0)
	}
}

// DefaultInterest rechnet die Verzugszinsen einer Forderung taggenau.
//
// from ist der erste Tag des Verzugs, to der Stichtag (das Datum des
// Mahnschreibens). Gezählt werden die Tage dazwischen, den Stichtag nicht
// mitgerechnet: am Tag der Zahlung ist der Schuldner nicht mehr in Verzug.
//
// Gerechnet wird abschnittsweise, weil der Basiszinssatz halbjährlich wechselt.
// Jeder Abschnitt wird für sich auf Cent gerundet, und die Summe der Abschnitte
// ist der Betrag — das Mahnschreiben führt die Abschnitte auf, und eine Summe,
// die nicht der Summe der ausgewiesenen Zeilen entspricht, ist ein Fehler in den
// Augen dessen, der sie nachrechnet.
func DefaultInterest(
	principal domain.Cents, from, to string, isConsumer bool, rates []BaseRatePeriod,
) (InterestResult, error) {
	out := InterestResult{Principal: principal, From: from, To: to}
	out.EnsureLists()

	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return out, fmt.Errorf("%q ist kein gültiger Verzugsbeginn (erwartet JJJJ-MM-TT)", from)
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return out, fmt.Errorf("%q ist kein gültiger Stichtag (erwartet JJJJ-MM-TT)", to)
	}
	if principal <= 0 || !end.After(start) {
		return out, nil
	}

	surcharge := DefaultInterestPointsBusiness
	if isConsumer {
		surcharge = DefaultInterestPointsConsumer
	}

	cursor := start
	for cursor.Before(end) {
		day := cursor.Format("2006-01-02")
		base, err := BaseRateAt(rates, day)
		if err != nil {
			return out, err
		}
		next := nextRateChange(rates, day)
		segmentEnd := end
		if next != "" {
			if changed, err := time.Parse("2006-01-02", next); err == nil && changed.Before(end) {
				segmentEnd = changed
			}
		}
		days := int(segmentEnd.Sub(cursor).Hours() / 24)
		if days <= 0 {
			break
		}
		points := base.BasisPoints + surcharge
		segment := InterestSegment{
			From: day, To: segmentEnd.Format("2006-01-02"), Days: days,
			BasisPoints: base.BasisPoints, TotalPoints: points,
		}
		if points > 0 {
			segment.Amount = roundDiv(
				int64(principal)*int64(points)*int64(days),
				int64(10000)*DefaultInterestDaysPerYear)
		}
		out.Segments = append(out.Segments, segment)
		out.Amount += segment.Amount
		out.Days += days
		cursor = segmentEnd
	}
	return out, nil
}

// nextRateChange liefert den nächsten Stichtag nach einem Tag, oder "".
func nextRateChange(rates []BaseRatePeriod, date string) string {
	for _, r := range rates {
		if r.ValidFrom > date {
			return r.ValidFrom
		}
	}
	return ""
}

// roundDiv teilt kaufmännisch: die halbe Einheit geht nach oben, bei negativen
// Beträgen nach unten. Ohne die Rundung fehlte je Abschnitt bis zu ein Cent, und
// eine Zinsforderung, die um Cents zu niedrig ist, ist genauso falsch wie eine
// zu hohe.
func roundDiv(numerator, denominator int64) domain.Cents {
	if denominator == 0 {
		return 0
	}
	if numerator < 0 {
		return -roundDiv(-numerator, denominator)
	}
	return domain.Cents((numerator*2 + denominator) / (denominator * 2))
}
