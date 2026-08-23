package einvoice

import (
	"os"
	"path/filepath"
	"testing"
)

func officialUBLFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "en16931", "ubl-examples", "*.[xX][mM][lL]"))
	if err != nil || len(files) == 0 {
		t.Fatalf("keine UBL-Beispiele unter testdata (%v)", err)
	}
	return files
}

// Die offiziellen UBL-Beispiele müssen lesbar sein und dieselbe Prüfung
// bestehen wie die CII-Beispiele. Genau das ist der Sinn eines semantischen
// Modells: eine Rechnung wird nach ihrem Inhalt beurteilt, nicht nach ihrer
// Schreibweise.
func TestOfficialUBLExamplesParseAndValidate(t *testing.T) {
	for _, path := range officialUBLFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Datei lesen: %v", err)
			}
			syntax, err := DetectSyntax(data)
			if err != nil {
				t.Fatalf("Syntax erkennen: %v", err)
			}
			if syntax != SyntaxUBL {
				t.Fatalf("Syntax = %q, erwartet ubl", syntax)
			}

			inv, err := Parse(data)
			if err != nil {
				t.Fatalf("lesen: %v", err)
			}
			if inv.Number == "" || !inv.IssueDate.Present() || inv.Currency == "" {
				t.Errorf("Pflichtangaben fehlen: Nummer %q, Datum %q, Währung %q",
					inv.Number, inv.IssueDate, inv.Currency)
			}
			if inv.Seller.Name == "" || inv.Buyer.Name == "" {
				t.Errorf("Beteiligte fehlen: %q / %q", inv.Seller.Name, inv.Buyer.Name)
			}
			if len(inv.Lines) == 0 {
				t.Error("keine Position gelesen")
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
