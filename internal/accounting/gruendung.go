package accounting

import (
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Was eine Kapitalgesellschaft aufbringen muss, bevor sie angemeldet werden darf.
//
// Der Katalog steht hier als Tabelle und nicht als Kette von if-Abfragen an den
// Aufrufstellen, aus demselben Grund wie der Rechtsformkatalog in
// domain/legalform.go: eine Regel, die an drei Stellen ausformuliert ist, ist
// drei Regeln, sobald sich eine ändert.
//

// FoundationRules is what one legal form demands before the Anmeldung.
type FoundationRules struct {
	// LegalForm ist die Schreibweise aus domain.LegalFormCatalog.
	LegalForm string `json:"legalForm"`
	// MinShareCapital ist das gesetzliche Mindestkapital.
	MinShareCapital domain.Cents `json:"minShareCapital"`
	// PaidInPerShareQuota ist der Anteil jeder einzelnen Geldeinlage, der vor der
	// Anmeldung eingezahlt sein muss: ein Viertel bei der GmbH, alles bei der UG.
	PaidInPerShareQuota float64 `json:"paidInPerShareQuota"`
	// PaidInFloor ist die Untergrenze für die Summe aller Einlagen, unabhängig
	// von der Aufteilung in Geschäftsanteile.
	//
	// Bei der GmbH ist das die Hälfte des *Mindest*stammkapitals, also stets
	// 12.500 € — nicht die Hälfte des tatsächlichen Stammkapitals
	// (§ 7 Abs. 2 Satz 2 GmbHG). Wer eine GmbH mit 100.000 € gründet, schuldet
	// vor der Anmeldung 25.000 € (ein Viertel je Anteil), nicht 50.000 €.
	PaidInFloor domain.Cents `json:"paidInFloor"`
	// PaidInFloorIsFullCapital sagt, dass die Untergrenze das volle Stammkapital
	// ist und deshalb nicht als fester Betrag danebenstehen kann (UG).
	PaidInFloorIsFullCapital bool `json:"paidInFloorIsFullCapital"`
	// CashOnly schließt die Sacheinlage aus (§ 5a Abs. 2 Satz 2 GmbHG).
	CashOnly bool `json:"cashOnly"`
	// LegalReserve verlangt die gesetzliche Rücklage nach § 5a Abs. 3 GmbHG.
	LegalReserve bool `json:"legalReserve"`
	// Reference ist die Fundstelle der Einzahlungsregel.
	Reference string `json:"reference"`
	// Note sagt, was die Rechtsform im Gründungsweg besonders macht.
	Note string `json:"note"`
}

// Der Katalog. Beträge in Cent, wie überall im System.
var foundationRules = []FoundationRules{
	{
		LegalForm:           "GmbH",
		MinShareCapital:     2_500_000,
		PaidInPerShareQuota: 0.25,
		PaidInFloor:         1_250_000,
		Reference:           "§ 7 Abs. 2 GmbHG",
		Note: "Auf jeden Geschäftsanteil ist ein Viertel des Nennbetrags einzuzahlen; " +
			"zusammen müssen die Einlagen die Hälfte des Mindeststammkapitals erreichen, " +
			"also 12.500 € (§ 7 Abs. 2 GmbHG). Sacheinlagen sind vor der Anmeldung " +
			"vollständig zu bewirken (§ 7 Abs. 3 GmbHG).",
	},
	{
		LegalForm:                "UG (haftungsbeschränkt)",
		MinShareCapital:          100,
		PaidInPerShareQuota:      1.0,
		PaidInFloorIsFullCapital: true,
		CashOnly:                 true,
		LegalReserve:             true,
		Reference:                "§ 5a Abs. 2 GmbHG",
		Note: "Die Anmeldung darf erst erfolgen, wenn das Stammkapital in voller Höhe " +
			"eingezahlt ist; Sacheinlagen sind ausgeschlossen (§ 5a Abs. 2 GmbHG). " +
			"In den Jahresabschluss ist ein Viertel des um einen Verlustvortrag " +
			"geminderten Jahresüberschusses in die gesetzliche Rücklage einzustellen " +
			"(§ 5a Abs. 3 GmbHG).",
	},
}

// FoundationRulesFor returns the rules of a Rechtsform. Der zweite
// Rückgabewert meldet, ob für sie Regeln hinterlegt sind.
func FoundationRulesFor(legalForm string) (FoundationRules, bool) {
	for _, r := range foundationRules {
		if r.LegalForm == legalForm {
			return r, true
		}
	}
	return FoundationRules{}, false
}

// FoundationLegalForms returns the Kapitalaufbringungsregeln of GmbH and UG.
func FoundationLegalForms() []FoundationRules {
	out := make([]FoundationRules, len(foundationRules))
	copy(out, foundationRules)
	return out
}

// RequiredPaidIn computes what must be contributed before the Anmeldung.
//
// Gerechnet wird je Geschäftsanteil und dann gegen die Untergrenze geprüft, weil
// beides nebeneinander gilt: die Viertelregel je Anteil und der Gesamtbetrag.
// Eine Sacheinlage zählt dabei mit ihrem vollen Nennbetrag, weil sie vollständig
// zu bewirken ist und § 7 Abs. 2 Satz 2 GmbHG sie mit dem Gesamtnennbetrag in
// die Rechnung nimmt.
func (r FoundationRules) RequiredPaidIn(f *domain.Foundation) domain.Cents {
	var sum domain.Cents
	for _, s := range f.Shareholders {
		sum += r.RequiredPerShare(s)
	}

	floor := r.PaidInFloor
	if r.PaidInFloorIsFullCapital {
		floor = f.ShareCapital
	}
	if sum < floor {
		return floor
	}
	return sum
}

// RequiredPerShare is the minimum contribution on a single Geschäftsanteil.
func (r FoundationRules) RequiredPerShare(s domain.Shareholder) domain.Cents {
	if s.Kind == domain.ContributionInKind {
		// Vollständig zu bewirken, § 7 Abs. 3 GmbHG.
		return s.ShareCapital
	}
	return quotaOf(s.ShareCapital, r.PaidInPerShareQuota)
}

// quotaOf takes a fraction of an amount and rounds up to the next full cent.
//
// Aufgerundet, nicht kaufmännisch: die Viertelregel ist eine Untergrenze. Ein
// halber Cent zu wenig ist zu wenig.
func quotaOf(amount domain.Cents, quota float64) domain.Cents {
	if quota >= 1.0 {
		return amount
	}
	if quota <= 0 || amount <= 0 {
		return 0
	}
	// Ganzzahlig gerechnet, damit kein Fließkommarest die Grenze verschiebt.
	num := int64(quota * 1_000_000)
	v := (int64(amount)*num + 999_999) / 1_000_000
	return domain.Cents(v)
}

// -------------------------------------------------------------
// Umsatzsteuer-Voranmeldung im Gründungsfall
// -------------------------------------------------------------

// VatMonthlyThreshold ist die Grenze, über der monatlich anzumelden ist
// (§ 18 Abs. 2 Satz 2 UStG): mehr als 9.000 € Steuer.
const VatMonthlyThreshold = domain.Cents(900_000)

// VatPeriodAdvice ist die Empfehlung für den Voranmeldungszeitraum.
type VatPeriodAdvice struct {
	// Period ist leer, wenn die Empfehlung an einer Steuer hängt, die nicht
	// vorliegt.
	Period domain.VatPeriodType `json:"period"`
	Reason string               `json:"reason"`
}

// RecommendedVatPeriod ist der Voranmeldungszeitraum im Jahr year einer
// Gesellschaft, die ihre Tätigkeit im Jahr foundingYear aufgenommen hat. ok ist
// falsch, wenn year weder das Gründungsjahr noch das Folgejahr ist; dann gilt
// die Steuer des Vorjahres.
//
// § 18 Abs. 2 Satz 4 UStG verlangt in beiden Jahren die monatliche
// Voranmeldung. Für die Besteuerungszeiträume 2021 bis 2026 tritt nach Satz 6
// an ihre Stelle die Grenze des Satzes 2: im Gründungsjahr nach der
// voraussichtlichen Steuer dieses Jahres, im Folgejahr nach der tatsächlichen
// Steuer des Gründungsjahres, in eine Jahressteuer umgerechnet. annualTax ist
// diese Steuer, nil heißt: sie liegt nicht vor. Eine Befreiung nach Satz 3 gibt
// es in beiden Jahren nicht (UStAE 18.7 Abs. 5 Satz 3).
//
// Die Regel steht hier und nicht als Satz in der Oberfläche, weil sie ein
// Stichjahr hat: ein fest getippter Hinweis wäre seit 2021 falsch gewesen und
// wäre es ab 2027 wieder.
func RecommendedVatPeriod(foundingYear, year int, annualTax *domain.Cents) (VatPeriodAdvice, bool) {
	followUp := year == foundingYear+1
	if year != foundingYear && !followUp {
		return VatPeriodAdvice{}, false
	}
	if year < 2021 || year > 2026 {
		return VatPeriodAdvice{Period: domain.VatPeriodMonth, Reason: fmt.Sprintf(
			"Im Jahr der Gründung und im folgenden Kalenderjahr ist die Voranmeldung monatlich "+
				"abzugeben (§ 18 Abs. 2 Satz 4 UStG); das gilt für %d.", year)}, true
	}
	basis := fmt.Sprintf("die voraussichtliche Steuer %d", year)
	if followUp {
		basis = fmt.Sprintf("die Steuer %d, umgerechnet auf ein volles Jahr", foundingYear)
	}
	if annualTax == nil {
		return VatPeriodAdvice{Reason: fmt.Sprintf(
			"Für %d entscheidet %s: über %s € monatlich, sonst vierteljährlich (§ 18 Abs. 2 Satz 6 "+
				"UStG). Die Pflicht zur monatlichen Abgabe für Gründungen ist bis 2026 ausgesetzt.",
			year, basis, VatMonthlyThreshold)}, true
	}
	if *annualTax > VatMonthlyThreshold {
		return VatPeriodAdvice{Period: domain.VatPeriodMonth, Reason: fmt.Sprintf(
			"Für %d entscheidet %s. Sie beträgt %s € und damit mehr als %s €: monatlich "+
				"(§ 18 Abs. 2 Sätze 2 und 6 UStG).", year, basis, *annualTax, VatMonthlyThreshold)}, true
	}
	return VatPeriodAdvice{Period: domain.VatPeriodQuarter, Reason: fmt.Sprintf(
		"Für %d entscheidet %s. Sie beträgt %s € und damit nicht mehr als %s €: vierteljährlich "+
			"(§ 18 Abs. 2 Sätze 1, 2 und 6 UStG). Eine Befreiung von der Abgabe gibt es im Jahr der "+
			"Gründung und im Folgejahr nicht.", year, basis, *annualTax, VatMonthlyThreshold)}, true
}

// addMonths shifts an ISO date by whole months, clamping to the end of the
// target month. Ohne das Kappen würde aus dem 31. Januar plus einem Monat der
// 3. März, weil time.AddDate überläuft.
func addMonths(iso string, months int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	year, month, day := t.Date()
	target := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := target.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}
