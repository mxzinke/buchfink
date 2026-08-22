package einvoice

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Jede Regel, die Buchfink zu prüfen behauptet, muss es in EN 16931 geben. Ein
// erfundener Regelname wäre die schlimmste Form von Scheingenauigkeit: er sähe
// aus wie eine Fundstelle, die man nachschlagen kann.
func TestClaimedRulesExistInTheStandard(t *testing.T) {
	for _, rule := range RulesChecked() {
		if _, ok := Rule(rule); !ok {
			t.Errorf("Buchfink führt %q als geprüfte Regel, die Norm kennt sie nicht", rule)
		}
	}
}

// Der Prüfumfang wird gemessen, nicht behauptet.
func TestCoverageIsMeasured(t *testing.T) {
	total := len(RulesInStandard())
	checked := len(RulesChecked())
	unchecked := RulesUnchecked()

	if total != checked+len(unchecked) {
		t.Fatalf("die Rechnung geht nicht auf: %d Regeln, %d geprüft, %d ungeprüft",
			total, checked, len(unchecked))
	}
	t.Logf("EN-16931-Abdeckung: %d von %d Geschäftsregeln (%d %%)", checked, total, checked*100/total)
	if len(unchecked) > 0 {
		t.Logf("ungeprüft: %s", strings.Join(unchecked, " "))
	}

	if checked < total {
		t.Errorf("%d Regeln der Norm werden nicht geprüft", len(unchecked))
	}
}

// Eine behauptete Regel ohne Prüfung ist schlimmer als eine ungeprüfte Regel:
// die eine steht in der Abdeckungsliste und beruhigt, die andere nicht.
//
// Der Test liest den Quelltext der Prüfung und vergleicht die dort gemeldeten
// Regelkennungen mit denen, die die Abdeckungsliste zusagt. Beispieldateien
// können das nicht leisten — ein gültiges Dokument löst keine Regel aus und
// zeigt deshalb weder eine falsche Kennung noch eine leere Zusage.
func TestEveryClaimedRuleIsActuallyChecked(t *testing.T) {
	reported := map[string]bool{}
	for _, file := range []string{"validate_model.go", "validate_categories.go", "validate_codes.go"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%s lesen: %v", file, err)
		}
		text := string(source)
		for _, m := range regexp.MustCompile(`"(BR-[A-Z]*-?\d+)"`).FindAllStringSubmatch(text, -1) {
			reported[m[1]] = true
		}
		// Die Kategorie-Familien werden zur Laufzeit zusammengesetzt.
		for _, m := range regexp.MustCompile(`"BR-"\s*\+\s*spec\.family\s*\+\s*"-(\d+)"`).FindAllStringSubmatch(text, -1) {
			for _, spec := range categorySpecs {
				reported["BR-"+spec.family+"-"+m[1]] = true
			}
		}
	}
	// BR-CO-05 bis BR-CO-08 sind auch im Referenzprüfer der Norm `true()`.
	// Sie stehen bewusst in der Abdeckungsliste und melden nichts.
	for _, rule := range []string{"BR-CO-05", "BR-CO-06", "BR-CO-07", "BR-CO-08"} {
		reported[rule] = true
	}

	for _, rule := range RulesChecked() {
		if !reported[rule] {
			t.Errorf("%s steht in der Abdeckungsliste, wird aber nirgends gemeldet", rule)
		}
	}
	claimed := map[string]bool{}
	for _, rule := range RulesChecked() {
		claimed[rule] = true
	}
	for rule := range reported {
		if !claimed[rule] {
			t.Errorf("%s wird gemeldet, steht aber nicht in der Abdeckungsliste", rule)
		}
	}
}

// Die Schweregrade kommen aus der Norm, nicht aus einer eigenen Einschätzung.
func TestSeverityFollowsTheStandard(t *testing.T) {
	var warnings []string
	for rule, info := range en16931Rules {
		if info.Severity != SeverityFatal {
			warnings = append(warnings, rule)
		}
	}
	sort.Strings(warnings)
	if len(warnings) != 1 || warnings[0] != "BR-51" {
		t.Errorf("die Norm führt %v als Hinweis, erwartet wurde allein BR-51", warnings)
	}
}

// Jede Regel nennt die Geschäftsbegriffe, um die es geht. Ohne sie müsste ein
// Nutzer aus einer Regelnummer erraten, welches Feld gemeint ist.
func TestRulesNameTheirBusinessTerms(t *testing.T) {
	// Die Codelisten-Regeln nennen in der Norm keine Begriffe — ihr Text sagt
	// nur, welche Liste gilt. Alle übrigen tun es.
	missing := 0
	for rule, info := range en16931Rules {
		if strings.HasPrefix(rule, "BR-CL-") {
			continue
		}
		if len(info.Terms) == 0 {
			t.Errorf("%s nennt keinen Geschäftsbegriff", rule)
			missing++
		}
	}
	if missing == 0 {
		t.Logf("alle %d Regeln außerhalb der Codelisten sind an Geschäftsbegriffe geknüpft",
			len(en16931Rules)-countPrefix(RulesInStandard(), "BR-CL-"))
	}
}

func countPrefix(values []string, prefix string) int {
	n := 0
	for _, v := range values {
		if strings.HasPrefix(v, prefix) {
			n++
		}
	}
	return n
}
