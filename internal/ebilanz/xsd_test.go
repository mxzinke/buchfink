package ebilanz

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// TestInstancesAreSchemaValid prüft die Instanzen der Beispiele gegen das
// XML-Schema der amtlichen Taxonomie.
//
// Die Schemata liegen nicht im Repository; eingebettet ist nur der Auszug.
// Die Prüfung läuft, wenn EBILANZ_XSD_6_9 bzw. EBILANZ_XSD_6_10 auf ein Schema
// zeigen, das xbrl-instance-2003-12-31.xsd sowie die Schemata de-gaap-ci und
// de-gcd der Fassung importiert (lokale Kopien des Taxonomiepakets).
// Geprüft wird mit dem Kommando aus EBILANZ_XSD_VALIDATOR (Aufruf mit Schema
// und Instanz, Erfolg bei Rückgabewert 0), sonst mit xmllint. xmllint braucht
// für das vollständige Schema von de-gaap-ci sehr lange; EBILANZ_XSD_TIMEOUT
// (Go-Dauer, Vorgabe 30m) begrenzt jeden Aufruf.
func TestInstancesAreSchemaValid(t *testing.T) {
	validator := strings.Fields(os.Getenv("EBILANZ_XSD_VALIDATOR"))
	if len(validator) == 0 {
		path, err := exec.LookPath("xmllint")
		if err != nil {
			t.Skip("weder EBILANZ_XSD_VALIDATOR noch xmllint vorhanden")
		}
		validator = []string{path, "--noout", "--nonet", "--schema"}
	}
	timeout := 30 * time.Minute
	if v := os.Getenv("EBILANZ_XSD_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	scenarios := map[string]func(t *testing.T) Input{
		"Jahresabschluss": func(t *testing.T) Input { return exampleInput(t, exampleBalances, "2025") },
		"Fassung610": func(t *testing.T) Input {
			in := exampleInput(t, exampleBalances, "2026")
			in.Version = "6.10"
			return in
		},
		"Ergebnisverwendung": func(t *testing.T) Input {
			return exampleInput(t, withChanges(exampleBalances, map[string]domain.Cents{"2970": -400000, "7790": 500000, "1800": -100000}), "2025")
		},
		"MicroBilG": func(t *testing.T) Input {
			in := exampleInput(t, withChanges(exampleBalances, map[string]domain.Cents{"4800": -200000, "1110-1139": 200000}), "2025")
			in.MicroBilG = true
			return in
		},
		"Eroeffnungsbilanz": func(t *testing.T) Input {
			in := exampleInput(t, map[string]domain.Cents{"1800": 2500000, "2900": -2500000}, "2025")
			in.Kind, in.ClosingDate, in.FiscalYearBegin, in.Statement = KindOpening, "2025-03-15", "2025-03-15", nil
			return in
		},
		"OhneGesellschafterliste": func(t *testing.T) Input {
			in := exampleInput(t, exampleBalances, "2025")
			in.Settings.Shareholders, in.Settings.HouseNumber = nil, ""
			return in
		},
		"Ueberleitung": func(t *testing.T) Input {
			in := exampleInput(t, withChanges(exampleBalances, map[string]domain.Cents{"3092": -200000, "6300": 200000}), "2025")
			in.Adjustments = []Adjustment{{Reason: "§ 5 Abs. 4a EStG", Account: "3092", IncomeAccount: "6300", Current: 200000}}
			return in
		},
	}
	ran := 0
	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			res := generateOK(t, scenario(t))
			schema := os.Getenv("EBILANZ_XSD_" + strings.ReplaceAll(res.Version.Version, ".", "_"))
			if schema == "" {
				t.Skipf("EBILANZ_XSD_%s nicht gesetzt", strings.ReplaceAll(res.Version.Version, ".", "_"))
			}
			ran++
			path := filepath.Join(t.TempDir(), "instanz.xml")
			if err := os.WriteFile(path, res.Instance, 0o644); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			args := append(append([]string{}, validator[1:]...), schema, path)
			out, err := exec.CommandContext(ctx, validator[0], args...).CombinedOutput()
			if err != nil {
				t.Fatalf("Instanz nicht schemagültig: %v\n%s", err, out)
			}
		})
	}
	if ran == 0 {
		t.Skip("keine Schemata angegeben (EBILANZ_XSD_6_9, EBILANZ_XSD_6_10)")
	}
}
