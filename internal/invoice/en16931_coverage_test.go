package invoice

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Jede Regel, die Buchfink zu prüfen behauptet, muss es in EN 16931 geben. Ein
// erfundener Regelname wäre die schlimmste Form von Scheingenauigkeit: er sähe
// aus wie eine Fundstelle, die man nachschlagen kann.
func TestClaimedRulesExistInTheStandard(t *testing.T) {
	known := map[string]bool{}
	for _, r := range EN16931AllRules() {
		known[r] = true
	}
	for _, r := range ValidationRules() {
		if !known[r] {
			t.Errorf("Buchfink führt %q als geprüfte Regel, die Norm kennt sie nicht", r)
		}
	}

	seen := map[string]bool{}
	for _, r := range ValidationRules() {
		if seen[r] {
			t.Errorf("die Regel %q steht doppelt in der Liste", r)
		}
		seen[r] = true
	}
}

// Der Prüfumfang wird gemessen, nicht behauptet. Der Test schreibt die Zahl ins
// Protokoll und hält eine Untergrenze fest, damit ein Rückschritt auffällt.
func TestCoverageIsMeasured(t *testing.T) {
	total := EN16931RuleCount()
	checked := len(ValidationRules())
	unchecked := EN16931UncheckedRules()

	if total != checked+len(unchecked) {
		t.Fatalf("die Rechnung geht nicht auf: %d Regeln, %d geprüft, %d ungeprüft",
			total, checked, len(unchecked))
	}

	percent := checked * 100 / total
	t.Logf("EN-16931-Abdeckung: %d von %d Geschäftsregeln (%d %%)", checked, total, percent)

	byFamily := map[string]int{}
	for _, r := range unchecked {
		byFamily[ruleFamily(r)]++
	}
	families := make([]string, 0, len(byFamily))
	for f := range byFamily {
		families = append(families, f)
	}
	sort.Strings(families)
	for _, f := range families {
		t.Logf("  ungeprüft in %-8s %d", f, byFamily[f])
	}

	// Eine Untergrenze, kein Ziel: sie fängt einen Rückschritt ab, ohne so zu
	// tun, als wäre die erreichte Zahl genug. Ungeprüft bleiben vor allem die
	// Codelisten zu Feldern, die Buchfink gar nicht liest (Zahlungsmittel,
	// Anlagen, Kennungsschemata), und die Regeln zu Rechnungsbezügen.
	const minimum = 165
	if checked < minimum {
		t.Errorf("die Abdeckung ist auf %d Regeln gefallen, erwartet mindestens %d", checked, minimum)
	}
}

func ruleFamily(rule string) string {
	parts := strings.Split(rule, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:2], "-")
	}
	return "BR-nn"
}

// corpusDir liefert das Verzeichnis mit den offiziellen CII-Beispielen.
//
// Die Dateien werden nicht mitgeliefert: sie stehen unter der EUPL, Buchfink
// unter EUPL-1.2. Wer sie prüfen will, legt sich das Validierungsartefakt daneben und
// setzt EN16931_CII_EXAMPLES — `task test:en16931` nimmt einem das ab.
func corpusDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("EN16931_CII_EXAMPLES")
	if dir == "" {
		t.Skip("EN16931_CII_EXAMPLES ist nicht gesetzt — siehe `task test:en16931`")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("EN16931_CII_EXAMPLES zeigt auf %s, das nicht lesbar ist: %v", dir, err)
	}
	return dir
}

// Die offiziellen Beispiele des Validierungsartefakts sind gültige Rechnungen.
// Meldet Buchfink an ihnen einen Fehler, ist die eigene Regel falsch — und genau
// dafür ist der Korpus da: er prüft die Prüfung.
//
// Zwei Befunde kamen so zustande. BR-08 und BR-10 fragen nach dem Vorhandensein
// der Adressgruppe, nicht nach ihrem Inhalt; und BR-CO-17 lässt eine Toleranz von
// einer Währungseinheit zu, ohne die eine ungarische Beispielrechnung als
// fehlerhaft galt.
func TestOfficialExamplesProduceNoErrors(t *testing.T) {
	dir := corpusDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Beispielverzeichnis lesen: %v", err)
	}

	var checked int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".xml") {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("lesen: %v", err)
			}
			doc, err := ParseCII(data)
			if err != nil {
				t.Fatalf("die Datei ist eine gültige CII-Rechnung, wurde aber nicht gelesen: %v", err)
			}
			result := ValidateEN16931(doc)
			for _, f := range result.Findings {
				if f.Severity == SeverityError {
					t.Errorf("%s: %s", f.Rule, f.Message)
				}
			}
		})
		checked++
	}

	if checked == 0 {
		t.Fatalf("in %s liegt keine einzige XML-Datei", dir)
	}
	t.Logf("%d offizielle Beispiele geprüft", checked)
}

// Eine behauptete Regel ohne Prüfung ist schlimmer als eine ungeprüfte Regel:
// die eine steht in der Abdeckungsliste und beruhigt, die andere nicht.
//
// Der Test liest die Prüfung im Quelltext und vergleicht die dort gemeldeten
// Regelkennungen mit denen, die ValidationRules zusagt. Er hat einen echten
// Fehler gefunden: BR-AG-05 stand in der Liste, die Kategorie "IPSI" hatte aber
// gar keine Satzbedingung hinterlegt, also prüfte niemand etwas.
func TestEveryClaimedRuleIsActuallyChecked(t *testing.T) {
	source, err := os.ReadFile("en16931.go")
	if err != nil {
		t.Fatalf("Quelltext der Prüfung lesen: %v", err)
	}

	reported := map[string]bool{}
	for _, m := range regexp.MustCompile(`"(BR-[A-Z]*-?\d+)"`).FindAllStringSubmatch(string(source), -1) {
		reported[m[1]] = true
	}
	// Die Kategorie-Familien werden zur Laufzeit zusammengesetzt, deshalb steht
	// im Quelltext nur die Endung.
	for _, m := range regexp.MustCompile(`"BR-"\s*\+\s*spec\.family\s*\+\s*"-(\d+)"`).FindAllStringSubmatch(string(source), -1) {
		for _, spec := range categorySpecs {
			reported["BR-"+spec.family+"-"+m[1]] = true
		}
	}

	for _, rule := range ValidationRules() {
		if !reported[rule] {
			t.Errorf("%s steht in der Abdeckungsliste, wird aber nirgends gemeldet", rule)
		}
	}

	claimed := map[string]bool{}
	for _, rule := range ValidationRules() {
		claimed[rule] = true
	}
	for rule := range reported {
		if !claimed[rule] {
			t.Errorf("%s wird gemeldet, steht aber nicht in der Abdeckungsliste", rule)
		}
	}
}
