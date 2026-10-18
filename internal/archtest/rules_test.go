package archtest

import (
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go/parser"
	"go/token"
)

// importsOf liefert je Produktivdatei eines Pakets die importierten Pfade.
func importsOf(t *testing.T, dir string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join("..", dir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			imported, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(path)] = append(out[filepath.ToSlash(path)], imported)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Der Journaldienst kennt die Steuerregeln nicht; core/wire.go registriert
// sie. Importierte service das Regelpaket, ließe sich eine Regel am Kern
// vorbei fest einbauen, und taxrules dürfte service nie mehr importieren.
func TestLedgerDoesNotImportItsRules(t *testing.T) {
	for path, imports := range importsOf(t, "service") {
		for _, imported := range imports {
			if imported == internalImport+"taxrules" {
				t.Errorf("%s importiert %s; Journalregeln werden in core/wire.go registriert", path, imported)
			}
		}
	}
}

// Eine Regel liest nur über die Sicht, die ihr der Journaldienst gibt. Sie
// braucht dafür die Typen aus domain und die Kataloge aus accounting, aber
// weder Dienste noch Speicher.
func TestTaxRulesDependOnlyOnDomainAndCatalogs(t *testing.T) {
	allowed := map[string]bool{internalImport + "domain": true, internalImport + "accounting": true}
	for path, imports := range importsOf(t, "taxrules") {
		for _, imported := range imports {
			stdlib := !strings.Contains(strings.SplitN(imported, "/", 2)[0], ".")
			if !stdlib && !allowed[imported] {
				t.Errorf("%s importiert %s; erlaubt sind domain, accounting und die Standardbibliothek", path, imported)
			}
		}
	}
}

func TestLedgerDoesNotImportItsEffects(t *testing.T) {
	for path, imports := range importsOf(t, "service") {
		for _, imported := range imports {
			if imported == internalImport+"bankeffects" {
				t.Errorf("%s importiert %s; Buchungseffekte werden in core/wire.go registriert", path, imported)
			}
		}
	}
}

func TestBankEffectsUseOnlyNarrowViews(t *testing.T) {
	allowed := map[string]bool{"context": true, "fmt": true, internalImport + "domain": true}
	for path, imports := range importsOf(t, "bankeffects") {
		for _, imported := range imports {
			if !allowed[imported] {
				t.Errorf("%s importiert %s; der Bankeffekt braucht nur domain, context und fmt", path, imported)
			}
		}
	}
}
