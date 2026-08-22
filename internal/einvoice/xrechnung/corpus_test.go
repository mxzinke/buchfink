package xrechnung

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// corpusDir returns the KoSIT test documents, or skips.
//
// Sie liegen nicht im Repository, werden aber bei Bedarf geholt — siehe
// `task test:xrechnung`. Das Artefakt steht unter Apache-2.0.
func corpusDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("XRECHNUNG_TEST_INSTANCES"))
	if dir == "" {
		t.Skip("XRECHNUNG_TEST_INSTANCES ist nicht gesetzt — `task test:xrechnung` holt die Dateien")
	}
	return dir
}

// KoSIT sagt für seine Testinstanzen, ob sie gültig sind. Damit lässt sich das
// eigene Urteil gegen ein fremdes stellen — und zwar in beide Richtungen, was
// eine Sammlung nur gültiger Dateien nicht kann.
//
// kositVerdicts sind die Zusicherungen aus assertions.xml der
// Validator-Konfiguration.
var kositVerdicts = map[string]bool{
	"ubl001.xml": true,
	"ubl002.xml": false,
	"ubl003.xml": false,
	"ubl004.xml": false,
	"ubl005.xml": false,
	"ubl006.xml": false,
	"ubl007.xml": false,
	"ubl008.xml": false,
	"cii001.xml": false,
}

func TestKositVerdictsAreMatched(t *testing.T) {
	dir := corpusDir(t)
	for name, wantValid := range kositVerdicts {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Skipf("%s nicht vorhanden: %v", name, err)
			}
			inv, err := einvoice.ParseAny(data)
			if err != nil {
				if wantValid {
					t.Fatalf("KoSIT hält die Datei für gültig, sie ist aber nicht lesbar: %v", err)
				}
				return
			}

			result := einvoice.ValidateWith(inv, Ruleset())
			gotValid := result.ErrorCount() == 0
			if gotValid == wantValid {
				for _, f := range result.Findings {
					t.Logf("%s %s %s: %s", f.Severity, f.Rule, f.Where, f.Message)
				}
				return
			}
			if wantValid {
				for _, f := range result.Findings {
					if f.Severity == einvoice.SeverityFatal {
						t.Errorf("KoSIT hält die Datei für gültig, Buchfink meldet %s %s: %s",
							f.Rule, f.Where, f.Message)
					}
				}
				return
			}
			// Buchfink hält eine Datei für gültig, die KoSIT verwirft. Das ist
			// keine falsche Meldung, sondern eine fehlende: der Mangel liegt in
			// einer Regel, die hier nicht geprüft wird.
			t.Errorf("KoSIT verwirft die Datei, Buchfink findet keinen Fehler — vermutlich eine ungeprüfte Regel")
		})
	}
}

// Die Regeldokumente von KoSIT sind echte XRechnungen. Alle müssen lesbar sein
// und die Prüfung überstehen, ohne dass etwas abstürzt — das ist die
// Robustheitsprobe an Belegen, die nicht von uns stammen.
func TestKositRuleDocumentsAreReadable(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv("XRECHNUNG_RULE_INSTANCES"))
	if dir == "" {
		t.Skip("XRECHNUNG_RULE_INSTANCES ist nicht gesetzt")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("keine Regeldokumente in %s (%v)", dir, err)
	}

	triggered := map[string]int{}
	unreadable := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s lesen: %v", filepath.Base(path), err)
		}
		inv, err := einvoice.ParseAny(data)
		if err != nil {
			unreadable++
			t.Errorf("%s ist nicht lesbar: %v", filepath.Base(path), err)
			continue
		}
		for _, f := range einvoice.ValidateWith(inv, Ruleset()).Findings {
			if strings.HasPrefix(f.Rule, "BR-DE") || strings.HasPrefix(f.Rule, "BR-TMP") {
				triggered[f.Rule]++
			}
		}
	}

	rules := make([]string, 0, len(triggered))
	for r := range triggered {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	t.Logf("%d Regeldokumente gelesen, %d nicht lesbar", len(files), unreadable)
	t.Logf("davon lösen aus: %s", strings.Join(rules, " "))
}
