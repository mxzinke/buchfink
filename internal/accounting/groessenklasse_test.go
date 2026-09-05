package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func assess(t *testing.T, year int, closing, start string, c domain.SizeCriteria) domain.SizeAssessment {
	t.Helper()
	a, err := AssessSize(year, closing, start, c)
	if err != nil {
		t.Fatalf("Größenklasse zum %s: %v", closing, err)
	}
	return a
}

// Die Schwellen sind datiert: das Gesetz zur Umsetzung der
// Schwellenwertrichtlinie hat sie für Geschäftsjahre angehoben, die nach dem
// 31.12.2023 beginnen. Dieselben Zahlen ergeben davor und danach eine andere
// Klasse.
func TestThresholdsDependOnTheStartOfTheFiscalYear(t *testing.T) {
	// Werte, die genau zwischen alter und neuer Kleinstschwelle liegen.
	criteria := domain.SizeCriteria{BalanceSheetTotal: 40_000_000, Revenue: 80_000_000, Employees: 8}

	old := assess(t, 2023, "2023-12-31", "2023-01-01", criteria)
	if old.Class != domain.SizeSmall {
		t.Errorf("für das Geschäftsjahr 2023 ergibt sich %s, erwartet die kleine Kapitalgesellschaft",
			old.Class.Label())
	}
	fresh := assess(t, 2024, "2024-12-31", "2024-01-01", criteria)
	if fresh.Class != domain.SizeMicro {
		t.Errorf("für das Geschäftsjahr 2024 ergibt sich %s, erwartet die Kleinstkapitalgesellschaft",
			fresh.Class.Label())
	}
	if !strings.Contains(fresh.Thresholds.Reference, "EGHGB") {
		t.Errorf("die angewandten Schwellen nennen ihre Fundstelle nicht: %q", fresh.Thresholds.Reference)
	}
}

// § 267 Abs. 1 HGB verlangt zwei der drei Merkmale. Eines allein genügt nicht,
// und das Ergebnis muss sagen, welche zwei es waren.
func TestTwoOfThreeCriteriaDecide(t *testing.T) {
	// Bilanzsumme und Umsatz klein, aber 600 Arbeitnehmer: zwei Merkmale
	// sprechen für klein, also ist die Gesellschaft klein.
	small := assess(t, 2026, "2026-12-31", "2026-01-01", domain.SizeCriteria{
		BalanceSheetTotal: 500_000_000, Revenue: 1_000_000_000, Employees: 600,
	})
	if small.Class != domain.SizeSmall {
		t.Errorf("zwei von drei Merkmalen sprechen für klein, die Klasse lautet aber %s", small.Class.Label())
	}
	if len(small.Met) != 2 {
		t.Errorf("die Begründung nennt %d Merkmale, erwartet zwei: %v", len(small.Met), small.Met)
	}

	// Nur ein Merkmal unter der Schwelle der mittelgroßen: große Gesellschaft.
	large := assess(t, 2026, "2026-12-31", "2026-01-01", domain.SizeCriteria{
		BalanceSheetTotal: 9_000_000_000, Revenue: 9_000_000_000, Employees: 100,
	})
	if large.Class != domain.SizeLarge {
		t.Errorf("nur ein Merkmal bleibt unter den Schwellen, die Klasse lautet aber %s", large.Class.Label())
	}
}

// Die Zweijahresregel des § 267 Abs. 4 Satz 1 HGB: die Klasse wechselt erst am
// zweiten übereinstimmenden Stichtag.
func TestClassChangesOnlyAtTheSecondMatchingClosingDate(t *testing.T) {
	smallCriteria := domain.SizeCriteria{BalanceSheetTotal: 100_000_000, Revenue: 200_000_000, Employees: 20}
	mediumCriteria := domain.SizeCriteria{BalanceSheetTotal: 2_000_000_000, Revenue: 4_000_000_000, Employees: 200}

	prior := assess(t, 2025, "2025-12-31", "2025-01-01", smallCriteria)
	current := assess(t, 2026, "2026-12-31", "2026-01-01", mediumCriteria)

	first := ClassifySize([]domain.SizeAssessment{prior, current}, false)
	if first.Class != domain.SizeSmall {
		t.Errorf("nach einem einzelnen abweichenden Stichtag gilt %s, erwartet die kleine Gesellschaft",
			first.Class.Label())
	}
	if !strings.Contains(first.Reason, "§ 267 Abs. 4 Satz 1 HGB") {
		t.Errorf("die Begründung nennt die Zweijahresregel nicht: %q", first.Reason)
	}

	next := assess(t, 2027, "2027-12-31", "2027-01-01", mediumCriteria)
	second := ClassifySize([]domain.SizeAssessment{prior, current, next}, false)
	if second.Class != domain.SizeMedium {
		t.Errorf("nach zwei übereinstimmenden Stichtagen gilt %s, erwartet die mittelgroße Gesellschaft",
			second.Class.Label())
	}
}

// Verglichen wird mit der wirksamen Klasse des Vorjahres, nicht mit dessen
// Beurteilung.
//
// klein, mittelgroß, klein: an keinen zwei aufeinanderfolgenden Stichtagen war
// die Gesellschaft mittelgroß, sie bleibt also durchgehend klein. Wer im
// dritten Jahr die Roh-Beurteilung des zweiten heranzieht, findet dort
// „mittelgroß", stellt eine Abweichung fest und schreibt die Prüfungspflicht
// fort, die nie eingetreten ist.
func TestTheTwoYearRuleComparesTheEffectiveClassOfThePriorYear(t *testing.T) {
	smallCriteria := domain.SizeCriteria{BalanceSheetTotal: 100_000_000, Revenue: 200_000_000, Employees: 20}
	mediumCriteria := domain.SizeCriteria{BalanceSheetTotal: 2_000_000_000, Revenue: 4_000_000_000, Employees: 200}

	small2025 := assess(t, 2025, "2025-12-31", "2025-01-01", smallCriteria)
	medium2026 := assess(t, 2026, "2026-12-31", "2026-01-01", mediumCriteria)
	small2027 := assess(t, 2027, "2027-12-31", "2027-01-01", smallCriteria)
	medium2027 := assess(t, 2027, "2027-12-31", "2027-01-01", mediumCriteria)

	back := ClassifySize([]domain.SizeAssessment{small2025, medium2026, small2027}, false)
	if back.Class != domain.SizeSmall {
		t.Errorf("klein/mittelgroß/klein ergibt %s, erwartet die kleine Gesellschaft: "+
			"zwei aufeinanderfolgende Stichtage mit derselben neuen Klasse gab es nie",
			back.Class.Label())
	}
	if back.Obligations.AuditRequired {
		t.Error("die kleine Gesellschaft ist nach § 316 Abs. 1 HGB nicht prüfungspflichtig")
	}

	up := ClassifySize([]domain.SizeAssessment{small2025, medium2026, medium2027}, false)
	if up.Class != domain.SizeMedium {
		t.Errorf("klein/mittelgroß/mittelgroß ergibt %s, erwartet die mittelgroße Gesellschaft",
			up.Class.Label())
	}
	if !up.Obligations.AuditRequired {
		t.Error("die mittelgroße Gesellschaft ist nach § 316 Abs. 1 HGB prüfungspflichtig")
	}

	// Und die Kette hält auch über mehrere Wechsel: klein, mittel, klein,
	// mittel — kein Paar stimmt überein, es bleibt beim ersten Stichtag.
	medium2028 := assess(t, 2028, "2028-12-31", "2028-01-01", mediumCriteria)
	long := ClassifySize(
		[]domain.SizeAssessment{small2025, medium2026, small2027, medium2028}, false)
	if long.Class != domain.SizeSmall {
		t.Errorf("nach vier wechselnden Stichtagen gilt %s, erwartet die kleine Gesellschaft",
			long.Class.Label())
	}
	if len(long.History) != 4 {
		t.Errorf("das Ergebnis führt %d Beurteilungen mit, erwartet vier", len(long.History))
	}
}

// § 267 Abs. 4 Satz 2 HGB: beim ersten Abschluss nach der Neugründung
// entscheidet dieser Stichtag allein.
func TestFirstYearIsDecidedByItsOwnClosingDate(t *testing.T) {
	current := assess(t, 2026, "2026-12-31", "2026-01-01", domain.SizeCriteria{
		BalanceSheetTotal: 3_000_000_000, Revenue: 6_000_000_000, Employees: 400,
	})
	result := ClassifySize([]domain.SizeAssessment{current}, true)
	if result.Class != domain.SizeLarge {
		t.Errorf("das erste Geschäftsjahr ergibt %s, erwartet die große Gesellschaft", result.Class.Label())
	}
	if !strings.Contains(result.Reason, "§ 267 Abs. 4 Satz 2 HGB") {
		t.Errorf("die Begründung nennt die Neugründungsregel nicht: %q", result.Reason)
	}
	if !result.IsFirstYear {
		t.Error("das Ergebnis kennzeichnet das erste Geschäftsjahr nicht")
	}
}

// Die Folgen der Klasse: Gliederungstiefe, Anhang, Lagebericht, Prüfung und
// Fristen.
func TestObligationsFollowTheClass(t *testing.T) {
	cases := []struct {
		class       domain.SizeClassKind
		depth       domain.StatementDepth
		notes       bool
		report      bool
		audit       bool
		preparation int
	}{
		{domain.SizeMicro, domain.DepthLetters, false, false, false, 6},
		{domain.SizeSmall, domain.DepthShort, true, false, false, 6},
		{domain.SizeMedium, domain.DepthFull, true, true, true, 3},
		{domain.SizeLarge, domain.DepthFull, true, true, true, 3},
	}
	for _, c := range cases {
		o := ObligationsFor(c.class)
		if o.Depth != c.depth {
			t.Errorf("%s: Gliederungstiefe %s, erwartet %s", c.class.Label(), o.Depth, c.depth)
		}
		if o.NotesRequired != c.notes {
			t.Errorf("%s: Anhangpflicht %v, erwartet %v", c.class.Label(), o.NotesRequired, c.notes)
		}
		if o.ManagementReport != c.report {
			t.Errorf("%s: Lagebericht %v, erwartet %v", c.class.Label(), o.ManagementReport, c.report)
		}
		if o.AuditRequired != c.audit {
			t.Errorf("%s: Prüfungspflicht %v, erwartet %v", c.class.Label(), o.AuditRequired, c.audit)
		}
		if o.PreparationMonths != c.preparation {
			t.Errorf("%s: Aufstellungsfrist %d Monate, erwartet %d",
				c.class.Label(), o.PreparationMonths, c.preparation)
		}
		if o.DisclosureMonths != 12 {
			t.Errorf("%s: Offenlegungsfrist %d Monate, erwartet zwölf (§ 325 Abs. 1a HGB)",
				c.class.Label(), o.DisclosureMonths)
		}
		if o.DisclosureScope == "" || o.DisclosureScopeReference == "" {
			t.Errorf("%s: der Offenlegungsumfang ist ohne Angabe oder ohne Fundstelle", c.class.Label())
		}
	}

	if scope := ObligationsFor(domain.SizeMicro).DisclosureScopeReference; scope != "§ 326 Abs. 2 HGB" {
		t.Errorf("die Kleinstkapitalgesellschaft darf hinterlegen (§ 326 Abs. 2 HGB), genannt ist %q", scope)
	}
	if scope := ObligationsFor(domain.SizeSmall).DisclosureScopeReference; scope != "§ 326 Abs. 1 HGB" {
		t.Errorf("die kleine Kapitalgesellschaft legt nach § 326 Abs. 1 HGB offen, genannt ist %q", scope)
	}
}

// Die Fristen laufen ab dem Abschlussstichtag, nicht ab dem Jahresende: ein
// Rumpfgeschäftsjahr endet mitten im Jahr.
//
// Die Aufstellungsfrist ist dabei keine Addition von Monaten auf den Stichtag:
// § 264 Abs. 1 Sätze 3 und 4 HGB nennt „die ersten drei" beziehungsweise „sechs
// Monate des Geschäftsjahrs". Für den Stichtag 30.06.2026 beginnt das folgende
// Geschäftsjahr am 01.07.2026, und dessen erste sechs Monate enden am
// 31.12.2026 — nicht am 30.12.
func TestDeadlinesRunFromTheClosingDate(t *testing.T) {
	deadlines := StatementDeadlines(2026, "2026-06-30", ObligationsFor(domain.SizeSmall))
	if len(deadlines) != 2 {
		t.Fatalf("es entstehen %d Termine, erwartet Aufstellung und Offenlegung", len(deadlines))
	}
	if deadlines[0].DueDate != "2026-12-31" {
		t.Errorf("die Aufstellungsfrist endet am %s, erwartet den 31.12.2026 — das Ende der "+
			"ersten sechs Monate des am 01.07.2026 beginnenden Geschäftsjahres", deadlines[0].DueDate)
	}
	if deadlines[0].Reference != "§ 264 Abs. 1 Satz 4 HGB" {
		t.Errorf("die Aufstellungsfrist nennt %q als Fundstelle", deadlines[0].Reference)
	}
	if deadlines[1].DueDate != "2027-06-30" {
		t.Errorf("die Offenlegungsfrist endet am %s, erwartet den 30.06.2027 (zwölf Monate)", deadlines[1].DueDate)
	}
	if deadlines[1].Reference != "§ 325 Abs. 1a Satz 1 HGB" {
		t.Errorf("die Offenlegungsfrist nennt %q als Fundstelle", deadlines[1].Reference)
	}

	large := StatementDeadlines(2026, "2026-12-31", ObligationsFor(domain.SizeLarge))
	if large[0].DueDate != "2027-03-31" {
		t.Errorf("die Aufstellungsfrist der großen Gesellschaft endet am %s, erwartet den 31.03.2027",
			large[0].DueDate)
	}

	// Ein Rumpfgeschäftsjahr zum 28.02.: die ersten drei Monate des am 01.03.
	// beginnenden Geschäftsjahres enden am 31.05., nicht am 28.05.
	short := StatementDeadlines(2026, "2026-02-28", ObligationsFor(domain.SizeMedium))
	if short[0].DueDate != "2026-05-31" {
		t.Errorf("die Aufstellungsfrist endet am %s, erwartet den 31.05.2026", short[0].DueDate)
	}
	if short[1].DueDate != "2027-02-28" {
		t.Errorf("die Offenlegungsfrist endet am %s, erwartet den 28.02.2027 — ein Jahr nach dem "+
			"Abschlussstichtag (§ 325 Abs. 1a Satz 1 HGB)", short[1].DueDate)
	}
}

// Ohne Stichtag gibt es keine Frist. Ein erfundenes Datum wäre schlimmer als
// keines.
func TestDeadlinesNeedAClosingDate(t *testing.T) {
	if got := StatementDeadlines(2026, "", ObligationsFor(domain.SizeSmall)); got != nil {
		t.Errorf("ohne Stichtag entstehen %d Termine, erwartet keine", len(got))
	}
}
