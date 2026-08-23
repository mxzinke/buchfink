package xrechnung

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// kositDir holds the KoSIT test documents. Herkunft und Lizenz stehen in
// ../testdata/README.md; sie liegen im Repository, damit der Abgleich mit
// einem fremden Urteil ohne Vorbereitung läuft.
func kositDir() string { return filepath.Join("testdata", "kosit") }

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
	dir := kositDir()
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
