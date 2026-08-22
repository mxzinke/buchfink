package einvoice

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// knownCorpusDeviations are the places where an official example does not meet
// the standard's own code list rules.
//
// The list is short and each entry names why. It exists because the alternative
// — quietly loosening the check until the corpus passes — would trade a real
// test for a comfortable one, and Buchfink would then certify invoices that the
// next recipient's validator rejects.
var knownCorpusDeviations = map[string]string{
	"XRechnung-O.xml/BR-CL-22": "der Befreiungsgrund steht klein geschrieben im Beispiel, " +
		"die Codeliste führt ihn groß; der Referenzprüfer der Norm vergleicht die " +
		"Schreibweise genau und würde die Datei ebenso beanstanden",
}

// Die offiziellen Beispiele sind bekannt gültig. Jeder Fund darin ist ein
// Fehler im Prüfer, nicht in der Rechnung — das ist der einzige Weg, eine
// selbstgeschriebene Prüfung gegen die Wirklichkeit zu stellen.
func TestOfficialExamplesProduceNoErrors(t *testing.T) {
	for _, path := range officialCIIFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Datei lesen: %v", err)
			}
			inv, err := ParseCII(data)
			if err != nil {
				t.Fatalf("XML lesen: %v", err)
			}
			for _, f := range Validate(inv).Findings {
				if f.Severity != SeverityFatal {
					continue
				}
				if reason, known := knownCorpusDeviations[filepath.Base(path)+"/"+f.Rule]; known {
					t.Logf("bekannte Abweichung %s: %s (%s)", f.Rule, f.Message, reason)
					continue
				}
				t.Errorf("%s %s: %s", f.Rule, f.Where, f.Message)
			}
		})
	}
}

// Welche Regeln der Korpus überhaupt berührt, sagt etwas über seine Aussagekraft
// — und darüber, welche Prüfungen von ihm gar nicht bestätigt werden.
func TestCorpusExercisesTheChecks(t *testing.T) {
	triggered := map[string]int{}
	files := officialCIIFiles(t)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Datei lesen: %v", err)
		}
		inv, err := ParseCII(data)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(path), err)
		}
		for _, f := range Validate(inv).Findings {
			triggered[f.Rule]++
		}
	}
	if len(triggered) == 0 {
		t.Log("keine einzige Regel schlägt im Korpus an — er ist vollständig normkonform")
		return
	}
	rules := make([]string, 0, len(triggered))
	for r := range triggered {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	for _, r := range rules {
		t.Logf("  %s: %dx (Hinweis)", r, triggered[r])
	}
}
