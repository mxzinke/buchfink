package accounting

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Größenklassen der §§ 267 und 267a HGB.
//
// An ihnen hängt fast jede Erleichterung des Jahresabschlusses: wie tief zu
// gliedern ist, ob ein Anhang und ein Lagebericht zu erstellen sind, ob geprüft
// werden muss, in welcher Frist aufzustellen ist und was offenzulegen ist. Sie
// gehören deshalb nicht in die Stammdaten, wo jemand sie „einstellen" könnte,
// sondern werden aus den drei Merkmalen berechnet.
//
// Die Schwellen sind datiert wie die steuerlichen Werte in tax_params.go: das
// Gesetz zur Umsetzung der Schwellenwertrichtlinie hat sie für Geschäftsjahre
// angehoben, die nach dem 31.12.2023 beginnen. Ein zurückliegendes Jahr muss
// mit den Werten beurteilbar bleiben, die damals galten.

// SizeClass ist das Ergebnis der Einordnung. Der Typ liegt in domain, damit
// FinancialStatement ihn tragen kann, ohne dass domain dieses Paket importiert.
type SizeClass = domain.SizeClass

// SizeThresholds ist ein datierter Satz Schwellenwerte.
type SizeThresholds struct {
	// ValidFrom ist der erste Geschäftsjahresbeginn, für den dieser Satz gilt.
	ValidFrom string
	Reference string
	Micro     domain.SizeCriteria
	Small     domain.SizeCriteria
	Medium    domain.SizeCriteria
}

// sizeThresholdSets sind nach ValidFrom geordnet, ältester Satz zuerst.
//
// Maßgeblich ist der Beginn des Geschäftsjahres, nicht der Stichtag: Artikel 79
// Abs. 1 EGHGB stellt die angehobenen Werte auf Geschäftsjahre ab, die nach dem
// 31. Dezember 2023 beginnen.
var sizeThresholdSets = []SizeThresholds{
	{
		ValidFrom: "0001-01-01",
		Reference: "§§ 267, 267a HGB in der Fassung vor dem Gesetz zur Umsetzung der Schwellenwertrichtlinie",
		Micro:     domain.SizeCriteria{BalanceSheetTotal: 35_000_000, Revenue: 70_000_000, Employees: 10},
		Small:     domain.SizeCriteria{BalanceSheetTotal: 600_000_000, Revenue: 1_200_000_000, Employees: 50},
		Medium:    domain.SizeCriteria{BalanceSheetTotal: 2_000_000_000, Revenue: 4_000_000_000, Employees: 250},
	},
	{
		ValidFrom: "2024-01-01",
		Reference: "§§ 267, 267a HGB, Art. 79 Abs. 1 EGHGB (Geschäftsjahre, die nach dem 31.12.2023 beginnen)",
		Micro:     domain.SizeCriteria{BalanceSheetTotal: 45_000_000, Revenue: 90_000_000, Employees: 10},
		Small:     domain.SizeCriteria{BalanceSheetTotal: 750_000_000, Revenue: 1_500_000_000, Employees: 50},
		Medium:    domain.SizeCriteria{BalanceSheetTotal: 2_500_000_000, Revenue: 5_000_000_000, Employees: 250},
	},
}

// SizeThresholdsFor liefert die Schwellen, die für ein Geschäftsjahr mit diesem
// Beginn gelten.
func SizeThresholdsFor(fiscalYearStart string) (SizeThresholds, error) {
	if fiscalYearStart == "" {
		return SizeThresholds{}, fmt.Errorf("ohne den Beginn des Geschäftsjahres lassen sich die Schwellenwerte nicht bestimmen")
	}
	idx := sort.Search(len(sizeThresholdSets), func(i int) bool {
		return sizeThresholdSets[i].ValidFrom > fiscalYearStart
	})
	if idx == 0 {
		return SizeThresholds{}, fmt.Errorf("für den %s sind keine Schwellenwerte hinterlegt", fiscalYearStart)
	}
	return sizeThresholdSets[idx-1], nil
}

func (t SizeThresholds) set() domain.SizeThresholdSet {
	return domain.SizeThresholdSet{
		ValidFrom: t.ValidFrom, Reference: t.Reference,
		Micro: t.Micro, Small: t.Small, Medium: t.Medium,
	}
}

// AssessSize beurteilt einen einzelnen Abschlussstichtag.
//
// § 267 Abs. 1 HGB verlangt, dass mindestens zwei der drei Merkmale die
// Schwellen nicht überschreiten. Welche zwei es sind, gehört ins Ergebnis: eine
// Klasse ohne Begründung ließe sich nicht nachprüfen, und die Zweijahresregel
// des Absatzes 4 setzt genau auf dieser Beurteilung auf.
func AssessSize(year int, closingDate, fiscalYearStart string, criteria domain.SizeCriteria) (domain.SizeAssessment, error) {
	thresholds, err := SizeThresholdsFor(fiscalYearStart)
	if err != nil {
		return domain.SizeAssessment{}, err
	}

	assessment := domain.SizeAssessment{
		Year: year, ClosingDate: closingDate, Criteria: criteria,
		Thresholds: thresholds.set(),
	}

	for _, candidate := range []struct {
		class  domain.SizeClassKind
		limits domain.SizeCriteria
	}{
		{domain.SizeMicro, thresholds.Micro},
		{domain.SizeSmall, thresholds.Small},
		{domain.SizeMedium, thresholds.Medium},
	} {
		met := metCriteria(criteria, candidate.limits)
		if len(met) >= 2 {
			assessment.Class = candidate.class
			assessment.Met = met
			return assessment, nil
		}
	}
	assessment.Class = domain.SizeLarge
	// Die große Gesellschaft erfüllt kein Merkmal — die Liste ist leer und
	// nicht `null`, weil die Ansicht über sie läuft.
	assessment.Met = []string{}
	return assessment, nil
}

// metCriteria benennt die Merkmale, die die Schwelle nicht überschreiten.
func metCriteria(criteria, limits domain.SizeCriteria) []string {
	met := make([]string, 0, 3)
	if criteria.BalanceSheetTotal <= limits.BalanceSheetTotal {
		met = append(met, fmt.Sprintf("Bilanzsumme %s € (höchstens %s €)",
			criteria.BalanceSheetTotal, limits.BalanceSheetTotal))
	}
	if criteria.Revenue <= limits.Revenue {
		met = append(met, fmt.Sprintf("Umsatzerlöse %s € (höchstens %s €)",
			criteria.Revenue, limits.Revenue))
	}
	if criteria.Employees <= limits.Employees {
		met = append(met, fmt.Sprintf("%d Arbeitnehmer im Jahresdurchschnitt (höchstens %d)",
			criteria.Employees, limits.Employees))
	}
	return met
}

// ClassifySize wendet die Zweijahresregel des § 267 Abs. 4 HGB an.
//
// Die Rechtsfolgen treten erst ein, wenn zwei aufeinanderfolgende
// Abschlussstichtage dieselbe neue Klasse ergeben. Ohne diese Regel würde ein
// einzelnes gutes Jahr die Prüfungspflicht auslösen und ein einzelnes schlechtes
// sie wieder nehmen — Satz 1 verhindert genau das. Satz 2 nimmt den ersten
// Abschluss nach der Neugründung aus: dort gilt der erste Stichtag.
//
// Stimmen die beiden letzten Stichtage nicht überein, bleibt es bei der
// *wirksamen* Klasse des Vorjahres — nicht bei dessen Beurteilung. Der
// Unterschied ist keine Feinheit: ergibt der erste Stichtag klein, der zweite
// mittelgroß und der dritte wieder klein, so war die Gesellschaft nie an zwei
// aufeinanderfolgenden Stichtagen mittelgroß; sie bleibt durchgehend klein. Der
// Vergleich mit der bloßen Vorjahresbeurteilung machte sie im dritten Jahr
// mittelgroß und löste die Prüfungspflicht aus, die Satz 1 gerade verhindert.
//
// history sind die Beurteilungen aufeinanderfolgender Stichtage, der älteste
// zuerst; das letzte Element ist der Stichtag, der einzuordnen ist. Die Kette
// darf am jüngsten übereinstimmenden Paar enden: ältere Stichtage ändern die
// wirksame Klasse dann nicht mehr.
func ClassifySize(history []domain.SizeAssessment, isFirstYear bool) domain.SizeClass {
	if len(history) == 0 {
		return domain.SizeClass{}
	}
	current := history[len(history)-1]
	result := domain.SizeClass{
		Year: current.Year, ClosingDate: current.ClosingDate,
		Criteria: current.Criteria, Current: current,
		IsFirstYear: isFirstYear, History: history,
	}
	var prior *domain.SizeAssessment
	if len(history) > 1 && !isFirstYear {
		previous := history[len(history)-2]
		prior = &previous
		result.Prior = prior
	}

	switch {
	case isFirstYear || prior == nil:
		result.Class = current.Class
		result.Reason = fmt.Sprintf(
			"%s nach dem Abschlussstichtag %s. %s",
			current.Class.Label(), germanDate(current.ClosingDate), reasonOf(current))
		if isFirstYear {
			result.Reason += " Beim ersten Abschluss nach der Gründung entscheidet dieser Stichtag allein (§ 267 Abs. 4 Satz 2 HGB)."
		} else {
			result.Reason += " Ein Vorjahresstichtag liegt nicht vor, deshalb entscheidet dieser Stichtag allein."
		}
	case current.Class == prior.Class:
		result.Class = current.Class
		result.Reason = fmt.Sprintf(
			"%s an beiden Stichtagen %s und %s. %s (§ 267 Abs. 4 Satz 1 HGB)",
			current.Class.Label(), germanDate(prior.ClosingDate), germanDate(current.ClosingDate),
			reasonOf(current))
	default:
		// Die Klasse wechselt erst am zweiten übereinstimmenden Stichtag; bis
		// dahin bleibt die wirksame Klasse der Vorjahre maßgeblich.
		effective, at, matched := effectiveClass(history)
		result.Class = effective
		origin := fmt.Sprintf("die seit dem Stichtag %s gilt", germanDate(history[at].ClosingDate))
		if matched {
			origin = fmt.Sprintf("die seit den Stichtagen %s und %s gilt",
				germanDate(history[at-1].ClosingDate), germanDate(history[at].ClosingDate))
		}
		result.Reason = fmt.Sprintf(
			"Der Stichtag %s ergibt %s, der Stichtag %s ergab %s. Nach § 267 Abs. 4 Satz 1 HGB "+
				"treten die Rechtsfolgen erst ein, wenn zwei aufeinanderfolgende Stichtage dieselbe "+
				"Klasse ergeben; es bleibt deshalb bei der Einordnung als %s, %s.",
			germanDate(current.ClosingDate), current.Class.Label(),
			germanDate(prior.ClosingDate), prior.Class.Label(),
			effective.Label(), origin)
	}

	result.Obligations = ObligationsFor(result.Class)
	return result
}

// effectiveClass löst die Kette des § 267 Abs. 4 Satz 1 HGB auf.
//
// Wirksam ist die Klasse des jüngsten Stichtagspaares, das übereinstimmt; gibt
// es keines, bleibt es bei der Beurteilung des ältesten bekannten Stichtags.
// Die zweite Rückgabe ist dessen Platz in der Kette, die dritte sagt, ob ein
// übereinstimmendes Paar die Klasse trägt.
func effectiveClass(history []domain.SizeAssessment) (domain.SizeClassKind, int, bool) {
	// Das jüngste Paar zählt: es überschreibt jede ältere Einordnung. Deshalb
	// von hinten, und nicht von vorn.
	for i := len(history) - 1; i > 0; i-- {
		if history[i].Class == history[i-1].Class {
			return history[i].Class, i, true
		}
	}
	return history[0].Class, 0, false
}

func reasonOf(a domain.SizeAssessment) string {
	if len(a.Met) == 0 {
		return "Keine zwei der drei Größenmerkmale bleiben unter den Schwellen der mittelgroßen Gesellschaft."
	}
	return "Maßgebend sind: " + strings.Join(a.Met, "; ") + "."
}

// ObligationsFor leitet die Folgen der Größenklasse ab.
func ObligationsFor(class domain.SizeClassKind) domain.SizeObligations {
	o := domain.SizeObligations{
		// Die Aufstellungsfrist des § 264 Abs. 1 Satz 3 HGB: drei Monate, für
		// kleine Gesellschaften sechs (Satz 4).
		PreparationMonths:    3,
		PreparationReference: "§ 264 Abs. 1 Satz 3 HGB",
		DisclosureMonths:     12,
		DisclosureReference:  "§ 325 Abs. 1a Satz 1 HGB",
	}

	switch class {
	case domain.SizeMicro:
		o.Depth = domain.DepthLetters
		o.DepthReference = "§ 266 Abs. 1 Satz 4 HGB"
		o.NotesRequired = false
		o.NotesReference = "§ 264 Abs. 1 Satz 5 HGB: kein Anhang, wenn die Angaben unter der Bilanz gemacht werden"
		o.PreparationMonths = 6
		o.PreparationReference = "§ 264 Abs. 1 Satz 4 HGB"
		o.DisclosureScope = "Bilanz; die Offenlegung kann durch Hinterlegung ersetzt werden"
		o.DisclosureScopeReference = "§ 326 Abs. 2 HGB"
	case domain.SizeSmall:
		o.Depth = domain.DepthShort
		o.DepthReference = "§ 266 Abs. 1 Satz 3 HGB"
		o.NotesRequired = true
		o.NotesReference = "§ 264 Abs. 1 Satz 1 HGB"
		o.PreparationMonths = 6
		o.PreparationReference = "§ 264 Abs. 1 Satz 4 HGB"
		o.DisclosureScope = "Bilanz und Anhang ohne die Angaben zur Gewinn- und Verlustrechnung"
		o.DisclosureScopeReference = "§ 326 Abs. 1 HGB"
	default:
		o.Depth = domain.DepthFull
		o.DepthReference = "§ 266 Abs. 2 und 3 HGB"
		o.NotesRequired = true
		o.NotesReference = "§ 264 Abs. 1 Satz 1 HGB"
		o.ManagementReport = true
		o.ManagementReportReference = "§ 264 Abs. 1 Satz 1 HGB, § 289 HGB"
		o.AuditRequired = true
		o.AuditReference = "§ 316 Abs. 1 HGB"
		o.DisclosureScope = "Jahresabschluss, Lagebericht und Bestätigungsvermerk"
		o.DisclosureScopeReference = "§ 325 Abs. 1 HGB"
	}
	return o
}

// StatementDeadlines sind die beiden Termine, die aus Stichtag und Größenklasse
// folgen: die Aufstellung nach § 264 Abs. 1 HGB und die Offenlegung nach
// § 325 Abs. 1a HGB.
func StatementDeadlines(year int, closingDate string, obligations domain.SizeObligations) []domain.Deadline {
	if closingDate == "" {
		return nil
	}
	return []domain.Deadline{
		{
			Key:   "abschluss.aufstellung",
			Title: fmt.Sprintf("Jahresabschluss %d aufstellen", year),
			// § 264 Abs. 1 Sätze 3 und 4 HGB stellt auf „die ersten drei" (bzw.
			// sechs) „Monate des Geschäftsjahrs" ab, nicht auf den Stichtag plus
			// N Monate. Das ist der Unterschied zwischen dem 31.12. und dem
			// 30.12.: für einen Stichtag am 30.06. beginnt das folgende
			// Geschäftsjahr am 01.07., und dessen erste sechs Monate enden am
			// 31.12. Nur bei Stichtagen am Ende eines 31-tägigen Monats fällt
			// beides zufällig zusammen.
			DueDate:    endOfMonthsAfter(closingDate, obligations.PreparationMonths),
			Period:     fmt.Sprintf("erste %d Monate des folgenden Geschäftsjahres", obligations.PreparationMonths),
			Reference:  obligations.PreparationReference,
			FiscalYear: year,
			Description: "Die gesetzlichen Vertreter haben den Jahresabschluss und, wo er zu erstellen ist, " +
				"den Lagebericht in dieser Frist aufzustellen.",
		},
		{
			Key:   "abschluss.offenlegung",
			Title: fmt.Sprintf("Jahresabschluss %d offenlegen", year),
			// Hier zählt der Stichtag selbst: § 325 Abs. 1a Satz 1 HGB gibt
			// „spätestens ein Jahr nach dem Abschlussstichtag" und nicht die
			// ersten Monate eines Geschäftsjahres.
			DueDate:    addMonths(closingDate, obligations.DisclosureMonths),
			Period:     fmt.Sprintf("%d Monate nach dem Abschlussstichtag", obligations.DisclosureMonths),
			Reference:  obligations.DisclosureReference,
			FiscalYear: year,
			Description: "Übermittlung an das Unternehmensregister. Umfang: " +
				obligations.DisclosureScope + " (" + obligations.DisclosureScopeReference + ").",
		},
	}
}

// endOfMonthsAfter ist das Ende der ersten N Monate des Zeitraums, der auf den
// Stichtag folgt.
//
// Gerechnet wird vom Tag nach dem Stichtag, weil die Frist des § 264 Abs. 1
// HGB an den Monaten des folgenden Geschäftsjahres hängt und nicht am Stichtag:
// 30.06. + sechs Monate ergäbe den 30.12., die ersten sechs Monate eines am
// 01.07. beginnenden Geschäftsjahres enden dagegen am 31.12.
func endOfMonthsAfter(iso string, months int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	start := t.AddDate(0, 0, 1)
	return start.AddDate(0, months, 0).AddDate(0, 0, -1).Format("2006-01-02")
}
