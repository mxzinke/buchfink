package einvoice

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Die Regelsuite der Norm: je Regel eine Datei mit Bruchstücken, von denen
// bekannt ist, ob die Regel anschlagen muss oder nicht.
//
// Das schließt die Lücke, die Beispielrechnungen prinzipiell offen lassen. Ein
// gültiges Dokument löst keine Regel aus; es kann deshalb bestätigen, dass eine
// Prüfung nicht zu streng ist, aber niemals, dass sie überhaupt greift. Erst
// hier zeigt sich, ob eine Prüfung tut, was ihr Name behauptet.
//
// Die Dateien sind in UBL geschrieben. Dass sie damit auch die CII-Seite
// absichern, ist der Ertrag des semantischen Modells: geprüft wird derselbe
// Code, nur der Leser davor ist ein anderer.

// unitTestDir returns the artefact's rule test suite, or skips.
func unitTestDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("EN16931_UNIT_TESTS"))
	if dir == "" {
		t.Skip("EN16931_UNIT_TESTS ist nicht gesetzt — `task test:en16931` holt die Dateien")
	}
	return dir
}

// renamedFamilies maps the rule family names of the artefact's test files onto
// the names the current standard uses.
//
// The tests were written when the Canary Islands and Ceuta rules were called
// BR-IG and BR-IP; they are BR-AF and BR-AG today. Running them under the old
// names would report every one of those twenty rules as unknown — which looks
// exactly like a gap in the implementation and is not one.
var renamedFamilies = map[string]string{"BR-IG-": "BR-AF-", "BR-IP-": "BR-AG-"}

func currentRuleName(rule string) string {
	for old, current := range renamedFamilies {
		if strings.HasPrefix(rule, old) {
			return current + strings.TrimPrefix(rule, old)
		}
	}
	return rule
}

type unitTestSet struct {
	Tests []struct {
		Assert struct {
			Description string   `xml:"description"`
			Success     []string `xml:"success"`
			Error       []string `xml:"error"`
		} `xml:"assert"`
	} `xml:"test"`
}

func TestArtefactRuleSuite(t *testing.T) {
	// Beide Sammlungen laufen: die Regeln gelten für Rechnung und Gutschrift
	// gleichermaßen, und dass eine Gutschrift bei uns durch denselben Leser und
	// dieselbe Prüfung geht, ist eine Zusage, die nachzuweisen ist.
	root := unitTestDir(t)
	var files []string
	for _, sub := range []string{"Invoice-unit-UBL", "CreditNote-unit-UBL"} {
		found, err := filepath.Glob(filepath.Join(root, sub, "*.xml"))
		if err != nil {
			t.Fatalf("%s lesen: %v", sub, err)
		}
		files = append(files, found...)
	}
	if len(files) == 0 {
		t.Fatalf("keine Regeldateien unter %s", root)
	}

	checked := map[string]bool{}
	for _, rule := range RulesChecked() {
		checked[rule] = true
	}

	confirmed := map[string]int{}
	var skipped []string

	for _, path := range files {
		name := filepath.Base(filepath.Dir(path)) + "/" + strings.TrimSuffix(filepath.Base(path), ".xml")
		// UBL-CR, UBL-DT und UBL-SR sind Syntaxregeln der UBL-Bindung, keine
		// Geschäftsregeln des Modells. Sie prüfen, welche UBL-Elemente
		// überhaupt vorkommen dürfen — eine Frage, die sich an einem
		// semantischen Modell nicht stellt.
		if strings.Contains(name, "/UBL-") {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s lesen: %v", name, err)
		}

		var set unitTestSet
		if err := xml.Unmarshal(data, &set); err != nil {
			t.Fatalf("%s: Testbeschreibung lesen: %v", name, err)
		}
		fragments, err := extractUBLFragments(data)
		if err != nil {
			t.Fatalf("%s: Bruchstücke lesen: %v", name, err)
		}
		if len(fragments) != len(set.Tests) {
			t.Fatalf("%s: %d Bruchstücke, aber %d Testfälle", name, len(fragments), len(set.Tests))
		}

		for i, testCase := range set.Tests {
			expectations := 0
			for _, rule := range append(append([]string{}, testCase.Assert.Success...), testCase.Assert.Error...) {
				if checked[currentRuleName(strings.TrimSpace(rule))] {
					expectations++
				}
			}
			if expectations == 0 {
				continue
			}

			inv, err := ParseUBL(fragments[i])
			if err != nil {
				skipped = append(skipped, name)
				continue
			}
			result := Validate(inv)

			for _, raw := range testCase.Assert.Success {
				rule := currentRuleName(strings.TrimSpace(raw))
				if !checked[rule] {
					continue
				}
				if findings := result.ByRule(rule); len(findings) > 0 {
					t.Errorf("%s Fall %d: %s hätte nicht anschlagen dürfen — %s",
						name, i+1, rule, findings[0].Message)
				}
				confirmed[rule]++
			}
			for _, raw := range testCase.Assert.Error {
				rule := currentRuleName(strings.TrimSpace(raw))
				if !checked[rule] {
					continue
				}
				if len(result.ByRule(rule)) == 0 {
					t.Errorf("%s Fall %d (%s): %s hätte anschlagen müssen",
						name, i+1, testCase.Assert.Description, rule)
				}
				confirmed[rule]++
			}
		}
	}

	rules := make([]string, 0, len(confirmed))
	for r := range confirmed {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	t.Logf("%d Regeln gegen die Testsuite der Norm bestätigt", len(rules))
	if len(skipped) > 0 {
		t.Logf("%d Bruchstücke waren nicht lesbar", len(skipped))
	}

	var unconfirmed []string
	for _, rule := range RulesChecked() {
		if confirmed[rule] == 0 {
			unconfirmed = append(unconfirmed, rule)
		}
	}
	if len(unconfirmed) > 0 {
		t.Logf("ohne Bestätigung durch die Suite: %s", strings.Join(unconfirmed, " "))
	}
}

// extractUBLFragments pulls the invoice fragments out of a test set, one per
// test case, re-serialised so they can be parsed on their own.
//
// Re-encoding rather than slicing the file text is what keeps the namespaces
// intact: the fragments inherit their prefixes from wherever the test file
// happens to declare them, and a substring would arrive without them.
func extractUBLFragments(data []byte) ([][]byte, error) {
	var out [][]byte
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Space != nsUBLInvoice && start.Name.Space != nsUBLCreditNote {
			continue
		}

		var buf bytes.Buffer
		encoder := xml.NewEncoder(&buf)
		if err := encoder.EncodeToken(start); err != nil {
			return nil, err
		}
		depth := 1
		for depth > 0 {
			inner, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			switch inner.(type) {
			case xml.StartElement:
				depth++
			case xml.EndElement:
				depth--
			}
			if err := encoder.EncodeToken(inner); err != nil {
				return nil, err
			}
		}
		if err := encoder.Flush(); err != nil {
			return nil, err
		}
		out = append(out, buf.Bytes())
	}
}
