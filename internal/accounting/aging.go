package accounting

import (
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// agingBands sind die Grenzen der Altersstruktur in Tagen seit Fälligkeit.
//
// Nicht fällig, 1–30, 31–60, 61–90, über 90 — die Staffelung, die jede
// Offene-Posten-Liste verwendet und an der sich ablesen lässt, wann eine
// Forderung zweifelhaft wird.
var agingBands = []struct {
	key      domain.AgingBucketKey
	label    string
	maxDays  int
	unbound  bool
	notDue   bool
	noDueDay bool
}{
	{key: domain.AgingNotDue, label: "Nicht fällig", notDue: true},
	{key: domain.AgingUpTo30, label: "1 bis 30 Tage überfällig", maxDays: 30},
	{key: domain.AgingUpTo60, label: "31 bis 60 Tage überfällig", maxDays: 60},
	{key: domain.AgingUpTo90, label: "61 bis 90 Tage überfällig", maxDays: 90},
	{key: domain.AgingOver90, label: "Über 90 Tage überfällig", unbound: true},
	{key: domain.AgingWithoutDate, label: "Ohne Fälligkeit", noDueDay: true},
}

var maturityBands = []struct {
	key   domain.MaturityBandKey
	label string
}{
	{domain.MaturityUpToOneYear, "Restlaufzeit bis zu einem Jahr"},
	{domain.MaturityOneToFive, "Restlaufzeit über ein bis fünf Jahre"},
	{domain.MaturityOverFiveYears, "Restlaufzeit über fünf Jahre"},
	{domain.MaturityUndated, "Ohne Fälligkeit"},
}

// AgeOpenItems gliedert die offenen Posten zum Stichtag nach Alter und
// Restlaufzeit.
//
// cutoff ist der Stichtag im Format YYYY-MM-DD; leer heißt: heute. Er wird
// übergeben und nicht aus der Uhr genommen, weil eine Auswertung, die sich beim
// Testen nicht auf einen Tag festlegen lässt, nicht prüfbar ist — und weil die
// Angabe unter der Bilanz den Abschlussstichtag braucht und nicht den heutigen
// Tag.
//
// Gerechnet wird mit dem offenen Betrag, nicht mit dem Bruttobetrag: was bezahlt
// ist, ist nicht überfällig. Ein negativer offener Betrag — eine Überzahlung —
// geht mit seinem Betrag ein, damit die Summe der Bänder die Summe der Posten
// bleibt.
func AgeOpenItems(items []domain.OpenItem, cutoff string) domain.OpenItemsAging {
	if cutoff == "" {
		cutoff = time.Now().UTC().Format("2006-01-02")
	}
	result := domain.OpenItemsAging{
		Cutoff:    cutoff,
		Reference: "§ 268 Abs. 4 und 5 HGB",
		Sides: []domain.OpenItemsAgingSide{
			{Side: "receivables", Label: "Forderungen aus Lieferungen und Leistungen"},
			{Side: "payables", Label: "Verbindlichkeiten aus Lieferungen und Leistungen"},
		},
	}
	for i := range result.Sides {
		for _, band := range agingBands {
			result.Sides[i].Buckets = append(result.Sides[i].Buckets,
				domain.AgingBucket{Key: band.key, Label: band.label})
		}
		for _, band := range maturityBands {
			result.Sides[i].Maturities = append(result.Sides[i].Maturities,
				domain.MaturityBand{Key: band.key, Label: band.label})
		}
	}

	oneYear := addYears(cutoff, 1)
	fiveYears := addYears(cutoff, 5)

	for i := range items {
		item := &items[i]
		if item.OpenAmount == 0 {
			continue
		}
		side := &result.Sides[0]
		if item.ContactType != domain.ContactTypeCustomer {
			side = &result.Sides[1]
		}
		amount := item.OpenAmount
		if amount < 0 {
			amount = -amount
		}
		side.Total += amount
		side.Items++

		addTo(side.Buckets, agingKey(item.DueDate, cutoff), amount)
		addToMaturity(side.Maturities, maturityKey(item.DueDate, cutoff, oneYear, fiveYears), amount)
	}
	result.EnsureLists()
	return result
}

// agingKey ordnet einen Posten seinem Altersband zu.
func agingKey(dueDate, cutoff string) domain.AgingBucketKey {
	if dueDate == "" {
		return domain.AgingWithoutDate
	}
	if dueDate >= cutoff {
		// Am Fälligkeitstag selbst ist noch nichts überfällig: die Zahlung ist
		// an diesem Tag zu leisten, nicht am Tag davor.
		return domain.AgingNotDue
	}
	days := daysBetween(dueDate, cutoff)
	for _, band := range agingBands {
		if band.notDue || band.noDueDay {
			continue
		}
		if band.unbound || days <= band.maxDays {
			return band.key
		}
	}
	return domain.AgingOver90
}

// maturityKey ordnet einen Posten seinem Restlaufzeitband zu.
//
// Ein überfälliger Posten hat eine Restlaufzeit bis zu einem Jahr: er ist
// sofort fällig. Etwas anderes anzugeben hieße, eine Verbindlichkeit, die schon
// gezahlt werden müsste, als langfristig auszuweisen.
func maturityKey(dueDate, cutoff, oneYear, fiveYears string) domain.MaturityBandKey {
	switch {
	case dueDate == "":
		return domain.MaturityUndated
	case dueDate <= oneYear:
		return domain.MaturityUpToOneYear
	case dueDate <= fiveYears:
		return domain.MaturityOneToFive
	default:
		return domain.MaturityOverFiveYears
	}
}

func addTo(buckets []domain.AgingBucket, key domain.AgingBucketKey, amount domain.Cents) {
	for i := range buckets {
		if buckets[i].Key == key {
			buckets[i].Amount += amount
			buckets[i].Items++
			return
		}
	}
}

func addToMaturity(bands []domain.MaturityBand, key domain.MaturityBandKey, amount domain.Cents) {
	for i := range bands {
		if bands[i].Key == key {
			bands[i].Amount += amount
			bands[i].Items++
			return
		}
	}
}

// daysBetween zählt die Tage zwischen zwei ISO-Daten. Ein unlesbares Datum
// liefert null — dann fällt der Posten in das erste Band statt in das
// schlimmste, und niemand mahnt aufgrund eines Parserfehlers.
func daysBetween(from, to string) int {
	a, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0
	}
	b, err := time.Parse("2006-01-02", to)
	if err != nil {
		return 0
	}
	return int(b.Sub(a).Hours() / 24)
}

// addYears verschiebt ein ISO-Datum um ganze Jahre.
func addYears(iso string, years int) string {
	if shifted := AddYearsISO(iso, years); shifted != "" {
		return shifted
	}
	return iso
}

// AddYearsISO verschiebt ein ISO-Datum um ganze Jahre und liefert den leeren
// String, wenn sich das Datum nicht lesen lässt.
//
// Sie steht hier und nicht in einem Hilfspaket, weil sie außer der
// Altersstruktur nur die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO braucht —
// zwei Aufrufer rechtfertigen kein Paket, aber sehr wohl eine gemeinsame
// Funktion.
func AddYearsISO(iso string, years int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return t.AddDate(years, 0, 0).Format("2006-01-02")
}
