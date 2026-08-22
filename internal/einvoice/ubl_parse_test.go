package einvoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ublExamplesDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("EN16931_UBL_EXAMPLES"))
	if dir == "" {
		t.Skip("EN16931_UBL_EXAMPLES ist nicht gesetzt — `task test:en16931` holt die Beispiele")
	}
	return dir
}

// Die offiziellen UBL-Beispiele müssen lesbar sein und dieselbe Prüfung
// bestehen wie die CII-Beispiele. Genau das ist der Sinn eines semantischen
// Modells: eine Rechnung wird nach ihrem Inhalt beurteilt, nicht nach ihrer
// Schreibweise.
func TestOfficialUBLExamplesParseAndValidate(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(ublExamplesDir(t), "*.[xX][mM][lL]"))
	if err != nil || len(files) == 0 {
		t.Fatalf("keine UBL-Beispiele in %s (%v)", ublExamplesDir(t), err)
	}

	for _, path := range files {
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
