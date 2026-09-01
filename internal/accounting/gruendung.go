package accounting

import (
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Was eine Kapitalgesellschaft aufbringen muss, bevor sie angemeldet werden darf
// — und welche Pflichten aus ihrer Gründung folgen.
//
// Der Katalog steht hier als Tabelle und nicht als Kette von if-Abfragen an den
// Aufrufstellen, aus demselben Grund wie der Rechtsformkatalog in
// domain/legalform.go: eine Regel, die an drei Stellen ausformuliert ist, ist
// drei Regeln, sobald sich eine ändert.
//
// Nur Kapitalgesellschaften. Bei einer Personengesellschaft gibt es weder eine
// Vorgesellschaft noch eine Unterbilanzhaftung — der Gründungsweg gilt dort
// nicht, und eine Rechtsform, die hier fehlt, bekommt ihn deshalb gar nicht erst
// angeboten.

// FoundationRules is what one legal form demands before the Anmeldung.
type FoundationRules struct {
	// LegalForm ist die Schreibweise aus domain.LegalFormCatalog.
	LegalForm string `json:"legalForm"`
	// MinShareCapital ist das gesetzliche Mindestkapital.
	MinShareCapital domain.Cents `json:"minShareCapital"`
	// PaidInPerShareQuota ist der Anteil jeder einzelnen Geldeinlage, der vor der
	// Anmeldung eingezahlt sein muss: ein Viertel bei GmbH und AG, alles bei der
	// UG.
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
	{
		LegalForm:           "AG",
		MinShareCapital:     5_000_000,
		PaidInPerShareQuota: 0.25,
		Reference:           "§ 36a Abs. 1 AktG",
		Note: "Der eingeforderte Betrag muss bei Bareinlagen mindestens ein Viertel des " +
			"geringsten Ausgabebetrags umfassen (§ 36a Abs. 1 AktG); das Grundkapital " +
			"beträgt mindestens 50.000 € (§ 7 AktG). Sacheinlagen sind vollständig zu " +
			"leisten (§ 36a Abs. 2 AktG).",
	},
}

// FoundationRulesFor returns the rules of a Rechtsform.
//
// Der zweite Rückgabewert ist die Antwort auf die Frage, ob der Gründungsweg für
// diese Rechtsform überhaupt gilt. Er ist die einzige Stelle, an der das
// entschieden wird — die Oberfläche fragt hier, statt Rechtsformnamen zu
// vergleichen.
func FoundationRulesFor(legalForm string) (FoundationRules, bool) {
	for _, r := range foundationRules {
		if r.LegalForm == legalForm {
			return r, true
		}
	}
	return FoundationRules{}, false
}

// FoundationLegalForms returns the legal forms the Gründungsweg covers.
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
		// Vollständig zu bewirken, § 7 Abs. 3 GmbHG bzw. § 36a Abs. 2 AktG.
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

// RecommendedVatPeriod is the Voranmeldungszeitraum a company founded in the
// given year starts with.
//
// § 18 Abs. 2 Satz 4 UStG verlangt im Gründungsjahr und im folgenden Jahr die
// monatliche Voranmeldung. Satz 6 hat das für die Besteuerungszeiträume 2021 bis
// 2026 ausgesetzt; dort gilt der Regelfall des Satzes 2, also das Kalender-
// vierteljahr. Ab dem Besteuerungszeitraum 2027 lebt die monatliche Pflicht
// wieder auf.
//
// Die Regel steht hier und nicht als Satz in der Oberfläche, weil sie ein
// Stichjahr hat: ein fest getippter Hinweis wäre seit 2021 falsch gewesen und
// wäre es ab 2027 wieder.
func RecommendedVatPeriod(foundingYear int) string {
	if foundingYear >= 2021 && foundingYear <= 2026 {
		return "quarter"
	}
	return "month"
}

// VatPeriodReason explains the recommendation in one sentence.
func VatPeriodReason(foundingYear int) string {
	if foundingYear >= 2021 && foundingYear <= 2026 {
		return fmt.Sprintf(
			"Für eine Gründung im Jahr %d gilt das Kalendervierteljahr: § 18 Abs. 2 Satz 6 UStG "+
				"setzt die monatliche Abgabepflicht des Satzes 4 für die Besteuerungszeiträume "+
				"2021 bis 2026 aus. Ab 2027 gilt sie wieder.",
			foundingYear,
		)
	}
	return fmt.Sprintf(
		"Für eine Gründung im Jahr %d ist die Voranmeldung monatlich abzugeben, im Gründungsjahr "+
			"und im folgenden Kalenderjahr (§ 18 Abs. 2 Satz 4 UStG).",
		foundingYear,
	)
}

// -------------------------------------------------------------
// Pflichten aus der Gründung
// -------------------------------------------------------------

// Die Schlüssel der Gründungspflichten. Sie stehen in der Datenbank und dürfen
// sich deshalb nicht mehr ändern.
const (
	DutyHandelsregister     = "handelsregister"
	DutyFragebogen          = "fragebogen"
	DutyGewerbeanmeldung    = "gewerbeanmeldung"
	DutyTransparenzregister = "transparenzregister"
	DutyEroeffnungsbilanz   = "eroeffnungsbilanz"
	DutyRuecklage           = "ruecklage"
	DutyOffenlegung         = "offenlegung"
)

// FoundationDuties returns the obligations arising from this founding, with
// their due dates and whether they are done.
//
// Die Fristen hängen an zwei Daten: der Beurkundung und der Eintragung. Was erst
// nach der Eintragung zu tun ist, erscheint erst dann — eine Frist, die auf ein
// noch nicht eingetretenes Ereignis zeigt, ist keine.
func FoundationDuties(f *domain.Foundation, rules FoundationRules, done map[string]string) []domain.FoundationDuty {
	if f == nil || f.NotarizedOn == "" {
		return nil
	}

	duties := []domain.FoundationDuty{
		{
			Key:       DutyHandelsregister,
			Title:     "Anmeldung zum Handelsregister",
			Deadline:  "sobald die Mindesteinlage geleistet ist",
			Reference: "§§ 7, 8 GmbHG",
			Description: "Die Anmeldung nimmt der Notar vor. Sie darf erst erfolgen, wenn die " +
				"Mindesteinlage auf dem Geschäftskonto steht — " + rules.Reference + ".",
		},
		{
			Key:       DutyFragebogen,
			Title:     "Fragebogen zur steuerlichen Erfassung",
			DueDate:   addMonths(f.NotarizedOn, 1),
			Deadline:  "innerhalb eines Monats",
			Reference: "§ 138 Abs. 1b und Abs. 4 AO",
			Description: "Elektronisch über Mein ELSTER an das Finanzamt. Daraus folgt die " +
				"Steuernummer, ohne die keine Rechnung mit Steuerausweis möglich ist.",
		},
		{
			Key:       DutyGewerbeanmeldung,
			Title:     "Gewerbeanmeldung bei der Gemeinde",
			DueDate:   addMonths(f.NotarizedOn, 1),
			Deadline:  "innerhalb eines Monats",
			Reference: "§ 14 GewO, § 138 Abs. 1 AO",
			Description: "Die Gemeinde unterrichtet das Finanzamt von sich aus; die Anzeige " +
				"ersetzt den Fragebogen aber nicht.",
		},
		{
			Key:       DutyEroeffnungsbilanz,
			Title:     "Eröffnungsbilanz aufstellen",
			DueDate:   addMonths(f.NotarizedOn, 6),
			Deadline:  "im ordnungsmäßigen Geschäftsgang",
			Reference: "§ 242 Abs. 1 HGB",
			Description: "Aufzustellen zu Beginn des Handelsgewerbes, also auf den Tag der " +
				"Beurkundung. Das Gesetz nennt keine Tagesfrist; der angezeigte Termin ist " +
				"der Richtwert einer kleinen Kapitalgesellschaft nach § 264 Abs. 1 Satz 4 HGB.",
		},
	}

	if rules.LegalReserve {
		duties = append(duties, domain.FoundationDuty{
			Key:       DutyRuecklage,
			Title:     "Gesetzliche Rücklage einstellen",
			DueDate:   fiscalYearEndAfter(f.NotarizedOn),
			Deadline:  "mit dem Jahresabschluss",
			Reference: "§ 5a Abs. 3 GmbHG",
			Description: "Ein Viertel des um einen Verlustvortrag aus dem Vorjahr geminderten " +
				"Jahresüberschusses gehört in die gesetzliche Rücklage (Konto 2930), bis das " +
				"Stammkapital 25.000 € erreicht. Buchfink führt die Pflicht, bucht sie aber " +
				"nicht: sie hängt am Jahresabschluss.",
		})
	}

	if f.IsRegistered() {
		duties = append(duties,
			domain.FoundationDuty{
				Key:       DutyTransparenzregister,
				Title:     "Wirtschaftlich Berechtigte melden",
				Deadline:  "unverzüglich nach der Eintragung",
				Reference: "§ 20 Abs. 1 GwG",
				Description: "Die Mitteilung an das Transparenzregister ist seit 2022 für jede " +
					"Gesellschaft Pflicht; die frühere Mitteilungsfiktion gibt es nicht mehr.",
			},
			domain.FoundationDuty{
				Key:       DutyOffenlegung,
				Title:     "Ersten Jahresabschluss offenlegen",
				DueDate:   addMonths(fiscalYearEndAfter(f.NotarizedOn), 12),
				Deadline:  "zwölf Monate nach dem Abschlussstichtag",
				Reference: "§ 325 Abs. 1a HGB",
				Description: "Übermittlung an das Unternehmensregister. Kleinstkapital" +
					"gesellschaften können stattdessen die Bilanz hinterlegen (§ 326 Abs. 2 HGB).",
			},
		)
	}

	for i := range duties {
		if day, ok := done[duties[i].Key]; ok {
			duties[i].DoneOn = day
			duties[i].IsDone = true
		}
	}
	return duties
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

// fiscalYearEndAfter is the 31 December of the founding year — the end of the
// Rumpfgeschäftsjahr.
func fiscalYearEndAfter(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d-12-31", t.Year())
}
