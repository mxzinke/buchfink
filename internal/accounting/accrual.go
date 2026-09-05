package accounting

import (
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Rechnung hinter der Rechnungsabgrenzung.
//
// § 250 Abs. 1 HGB nennt den Maßstab: aktiviert wird, was „Aufwand für eine
// bestimmte Zeit nach diesem Tag" darstellt. Der Aufwand wird also zeitanteilig
// verteilt, und die einzige Frage ist, in welchen Einheiten. Buchfink kennt
// zwei — Monate und Tage — und wählt nicht selbst: das Verfahren ist eine
// Einstellung des Mandanten, weil § 252 Abs. 1 Nr. 6 HGB die Stetigkeit der
// Bewertungsmethoden verlangt.

// AccrualUnit ist eine Zeiteinheit der Verteilung mit ihrem Geschäftsjahr.
type AccrualUnit struct {
	// Start ist der erste Tag der Einheit (Monatserster bzw. der Tag selbst).
	Start      string
	FiscalYear int
}

// accrualUnits zerlegt einen Zeitraum in die Einheiten des Verfahrens.
//
// Beim monatsgenauen Verfahren zählt der angefangene Monat voll — aber nur
// einmal. Ein Zwölfmonatsvertrag bleibt ein Zwölftelvertrag, gleich ob er am
// 1. oder am 15. eines Monats beginnt: gezählt werden die Monate von seinem
// Beginn bis zu seinem Ende, und der Rest, der über den letzten vollen Monat
// hinausragt, füllt den angefangenen Anfangsmonat auf. Zählte auch der
// angefangene Endmonat noch einmal voll, ergäbe ein Vertrag vom 15.12.2026 bis
// zum 14.12.2027 dreizehn Einheiten und damit 12/13 statt 11/12 als
// Abgrenzung.
//
// Die Einheiten tragen die Monatsersten ab dem Anfangsmonat als Kennung; der
// über sie hinausragende Rest des Endmonats gehört rechnerisch zum
// angefangenen Anfangsmonat. Wo diese Vereinfachung nicht genügt, gibt es das
// taggenaue Verfahren.
func accrualUnits(from, to string, method domain.AccrualMethod, startMonth int) ([]AccrualUnit, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("%q ist kein Datum (erwartet JJJJ-MM-TT)", from)
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("%q ist kein Datum (erwartet JJJJ-MM-TT)", to)
	}
	if end.Before(start) {
		return nil, nil
	}

	var units []AccrualUnit
	if method == domain.AccrualDaily {
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			day := d.Format("2006-01-02")
			units = append(units, AccrualUnit{Start: day, FiscalYear: domain.GetFiscalYearForDate(day, startMonth)})
		}
		return units, nil
	}

	count := monthlyUnitCount(start, end)
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		day := cursor.Format("2006-01-02")
		units = append(units, AccrualUnit{Start: day, FiscalYear: domain.GetFiscalYearForDate(day, startMonth)})
		cursor = cursor.AddDate(0, 1, 0)
	}
	return units, nil
}

// monthlyUnitCount ist die Zahl der Monatseinheiten eines Zeitraums.
//
// Gerechnet wird über das offene Ende: der Zeitraum reicht bis zum Tag nach dem
// letzten Tag. Passen n volle Monate genau hinein, sind es n Einheiten; bleibt
// ein Rest, zählt er als angefangener Monat voll. Für einen Vertrag vom
// Monatsersten bis zum Ende eines Monats ist das genau die Zahl der Monate; für
// den 15.12.2026 bis 14.12.2027 sind es zwölf und nicht dreizehn.
func monthlyUnitCount(start, end time.Time) int {
	limit := end.AddDate(0, 0, 1)
	months := 0
	for start.AddDate(0, months+1, 0).Before(limit) {
		months++
	}
	// months ist jetzt die Zahl der vollen Monate vor dem letzten; passt der
	// nächste genau bis zum offenen Ende, ist er der letzte volle, sonst ist er
	// der angefangene — in beiden Fällen zählt er.
	return months + 1
}

// AccrualShare ist der Teil eines Gesamtbetrags, der nach dem Stichtag liegt.
//
// Beispiel: eine Versicherungsprämie von 1.200 € für die zwölf Monate ab dem
// 1. Dezember. Zum 31.12. sind elf der zwölf Monate noch nicht verbraucht, also
// sind 1.100 € abzugrenzen.
func AccrualShare(
	total domain.Cents, start, end, cutoff string, method domain.AccrualMethod, startMonth int,
) (domain.Cents, error) {
	if total <= 0 {
		return 0, fmt.Errorf("ohne Betrag lässt sich nichts abgrenzen")
	}
	all, err := accrualUnits(start, end, method, startMonth)
	if err != nil {
		return 0, err
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("der Zeitraum %s bis %s ist leer", start, end)
	}
	after := 0
	for _, u := range all {
		if u.Start > cutoff {
			after++
		}
	}
	if after == 0 {
		return 0, nil
	}
	if after == len(all) {
		return total, nil
	}
	return domain.MulRound(total, int64(after), int64(len(all))), nil
}

// AccrualRelease ist eine geplante Auflösung eines Geschäftsjahres.
type AccrualRelease struct {
	FiscalYear int          `json:"fiscalYear"`
	Date       string       `json:"date"`
	Amount     domain.Cents `json:"amount"`
}

// AccrualReleasePlan verteilt den abgegrenzten Betrag auf die Geschäftsjahre
// nach dem Stichtag.
//
// Ein Posten kann über mehrere Jahre laufen — eine Versicherung über drei Jahre,
// ein Disagio über die Laufzeit des Darlehens. Der Plan entsteht einmal bei der
// Bildung und wird gespeichert: was gebucht wird, muss auch dann noch dasselbe
// sein, wenn jemand später das Verfahren umstellt.
//
// Die Rundungsdifferenz landet im letzten Jahr. Sie irgendwo anders zu lassen
// hieße, den Posten nicht vollständig aufzulösen — und ein Rest auf 1900, der
// zu nichts mehr gehört, fällt erst Jahre später auf.
func AccrualReleasePlan(
	deferred domain.Cents, start, end, cutoff string, method domain.AccrualMethod, startMonth int,
) ([]AccrualRelease, error) {
	return AccrualReleasePlanFor(
		deferred, start, end, cutoff, method, startMonth, domain.AccrualReleaseYearly)
}

// AccrualReleasePlanFor ist derselbe Plan in wählbarem Takt.
//
// Beim jährlichen Takt entsteht je Geschäftsjahr eine Auflösung am ersten Tag,
// beim monatlichen je Monat eine am Monatsersten. Die Jahreszahlen sind in
// beiden Fällen dieselben; verschieden ist nur, wann der Aufwand innerhalb des
// Jahres ankommt — und das ist genau die Frage, die eine unterjährige
// Auswertung stellt.
func AccrualReleasePlanFor(
	deferred domain.Cents, start, end, cutoff string, method domain.AccrualMethod,
	startMonth int, schedule domain.AccrualReleaseSchedule,
) ([]AccrualRelease, error) {
	if deferred <= 0 {
		return nil, fmt.Errorf("ohne abgegrenzten Betrag gibt es nichts aufzulösen")
	}
	all, err := accrualUnits(start, end, method, startMonth)
	if err != nil {
		return nil, err
	}

	// Reihenfolge bewahren: eine Map allein verlöre sie, und der Plan wird in
	// dieser Reihenfolge gebucht. Der Schlüssel ist beim jährlichen Takt das
	// Geschäftsjahr und beim monatlichen der Kalendermonat.
	type bucket struct {
		key        string
		fiscalYear int
		date       string
		count      int
	}
	var order []string
	buckets := map[string]*bucket{}
	total := 0
	for _, u := range all {
		if u.Start <= cutoff {
			continue
		}
		key := fmt.Sprintf("%d", u.FiscalYear)
		date := fiscalYearFirstDay(u.FiscalYear, startMonth)
		if schedule == domain.AccrualReleaseMonthly {
			key = u.Start[:7]
			date = key + "-01"
		}
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
			buckets[key] = &bucket{key: key, fiscalYear: u.FiscalYear, date: date}
		}
		buckets[key].count++
		total++
	}
	if total == 0 {
		return nil, fmt.Errorf("nach dem Stichtag %s liegt kein Zeitraum mehr", cutoff)
	}

	plan := make([]AccrualRelease, 0, len(order))
	var allocated domain.Cents
	for i, key := range order {
		b := buckets[key]
		amount := domain.MulRound(deferred, int64(b.count), int64(total))
		if i == len(order)-1 {
			// Die Rundungsdifferenz landet im letzten Schritt. Sie irgendwo
			// anders zu lassen hieße, den Posten nicht vollständig aufzulösen —
			// und ein Rest auf 1900, der zu nichts mehr gehört, fällt erst
			// Jahre später auf.
			amount = deferred - allocated
		}
		allocated += amount
		plan = append(plan, AccrualRelease{FiscalYear: b.fiscalYear, Date: b.date, Amount: amount})
	}
	return plan, nil
}

// fiscalYearFirstDay ist der erste Tag eines Geschäftsjahres. Bei abweichendem
// Geschäftsjahr beginnt das Jahr im Kalenderjahr, das seine Jahreszahl trägt.
func fiscalYearFirstDay(fiscalYear, startMonth int) string {
	if startMonth <= 0 || startMonth > 12 {
		startMonth = 1
	}
	return fmt.Sprintf("%04d-%02d-01", fiscalYear, startMonth)
}
