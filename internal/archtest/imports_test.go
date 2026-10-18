package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const internalImport = "github.com/buchfink/buchfink/internal/"

var storageImportExceptions = map[string]string{
	"service/backup_service.go:gorm.io/gorm":       "BackupService verwaltet Datenbankhandles für VACUUM INTO und die Wiederherstellungsprüfung.",
	"mcpserver/oauth/store.go:gorm.io/gorm":        "Store ist die eigene Speicherschicht für OAuth-Zulassungen und Tokens.",
	"mcpserver/oauth/store.go:gorm.io/gorm/logger": "Store konfiguriert das Datenbankprotokoll beim Öffnen.",
	"mcpserver/oauth/server.go:gorm.io/gorm":       "issueTokens schreibt Tokens innerhalb der Store-Transaktion für Ausgabe und Rotation.",
}

func inspectStorageImports(fset *token.FileSet, file *ast.File, path string, exceptions map[string]string) []string {
	var problems []string
	for _, imp := range file.Imports {
		imported, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if !strings.HasPrefix(imported, "gorm.io/") {
			continue
		}
		key := path + ":" + imported
		if reason := exceptions[key]; reason != "" {
			delete(exceptions, key)
			continue
		}
		problems = append(problems, fmt.Sprintf("%s: verbotener Speicherimport %s; nicht gefunden über domain.ErrNotFound prüfen, Datenbankzugriffe im Repository ausführen", fset.Position(imp.Pos()), imported))
	}
	return problems
}

func TestStorageImportBoundaries(t *testing.T) {
	exceptions := map[string]string{}
	for key, reason := range storageImportExceptions {
		exceptions[key] = reason
	}
	fset := token.NewFileSet()
	for _, layer := range []string{"service", "mcpserver"} {
		err := filepath.WalkDir(filepath.Join("..", layer), func(path string, d fs.DirEntry, err error) error {
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
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel("..", path)
			if err != nil {
				return err
			}
			for _, problem := range inspectStorageImports(fset, file, filepath.ToSlash(relative), exceptions) {
				t.Error(problem)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for key := range exceptions {
		t.Errorf("ungenutzte Speicherimport-Ausnahme %s entfernen", key)
	}
}

func TestStorageImportScanner(t *testing.T) {
	for _, tc := range []struct {
		path, imported string
		wantError      bool
	}{
		{"service/settings_service.go", "gorm.io/gorm", true},
		{"service/settings_service.go", "gorm.io/driver/sqlite", true},
		{"mcpserver/errors.go", "gorm.io/gorm", true},
		{"mcpserver/errors.go", "gorm.io/gorm/logger", true},
		{"service/backup_service.go", "gorm.io/gorm", false},
		{"service/backup_service.go", "gorm.io/driver/sqlite", true},
		{"mcpserver/oauth/store.go", "gorm.io/gorm", false},
		{"mcpserver/oauth/store.go", "gorm.io/gorm/logger", false},
		{"mcpserver/oauth/server.go", "gorm.io/gorm", false},
		{"mcpserver/oauth/other.go", "gorm.io/gorm", true},
		{"service/settings_service.go", internalImport + "domain", false},
	} {
		t.Run(tc.path+":"+tc.imported, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, tc.path, "package example; import alias "+strconv.Quote(tc.imported), 0)
			if err != nil {
				t.Fatal(err)
			}
			exceptions := map[string]string{}
			for key, reason := range storageImportExceptions {
				exceptions[key] = reason
			}
			problems := inspectStorageImports(fset, file, tc.path, exceptions)
			if (len(problems) > 0) != tc.wantError {
				t.Fatalf("Befunde = %v, Fehler erwartet: %t", problems, tc.wantError)
			}
		})
	}
}

// Kataloge, reine Datentypen und Konstanten, getrennt nach Adapter.
var catalogSymbols = map[string]map[string]bool{
	"mcpserver":   symbolSet("PostingGroups", "LookupAssetAccount", "SpecificAllowance", "WorkInProgressProject", "AdvanceTarget", "AdvanceTargetInventory", "AdvanceTargetTangible", "AdvanceTargetIntangible", "MatchScoreDueNear"),
	"wailsbridge": symbolSet("DegressiveWindows", "AfARuleSet", "SpecialMaxPermille", "SpecialPeriodYears", "DegressiveWindow", "ElectricVehicleWindow", "AllFundClasses", "FundClass", "RetentionRules", "LoadedRetentionRules", "JournalFilter", "JournalFilterResult", "JournalFilterRow", "PostingGroup", "PostingGroups", "FoundationRules", "FoundationLegalForms", "SizeClass", "AfARules", "ProvisionMirror"),
}

// Bekannte Ausnahmen, werden in Schritt 9 abgebaut. Die Fundstelle und die
// Anzahl begrenzen jede Ausnahme auf die vorhandenen Aufrufe.
var knownLogicExceptions = map[string]int{
	"mcpserver/belege.go:inputTaxPeriodHint:VatPeriodOf":                   0,
	"mcpserver/erklaerung.go:periodMismatch:ParseVatPeriodKey":             0,
	"mcpserver/posten.go:registerPosten:AgeOpenItems":                      0,
	"wailsbridge/asset_service.go:GetAssetRules:AfAParametersFor":          0,
	"wailsbridge/asset_service.go:GetInvestmentRules:PartialExemptionFor":  0,
	"wailsbridge/nachweise_service.go:GetComplianceHints:SystemChangeNote": 0,
	"wailsbridge/nachweise_service.go:GetOpenItemsAging:AgeOpenItems":      0,
}

func symbolSet(names ...string) map[string]bool {
	out := map[string]bool{}
	for _, name := range names {
		out[name] = true
	}
	return out
}

func inspectImports(fset *token.FileSet, file *ast.File, path, adapter string, exceptions map[string]int) []string {
	problems := []string{}
	accountingAlias := ""
	for _, imp := range file.Imports {
		imported, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if (adapter == "accounting" && (imported == internalImport+"repository" || strings.HasPrefix(imported, internalImport+"repository/") || strings.HasPrefix(imported, "gorm.io/"))) ||
			(adapter == "mcpserver" && (imported == internalImport+"repository" || strings.HasPrefix(imported, internalImport+"repository/"))) {
			problems = append(problems, fmt.Sprintf("%s: verbotener direkter Import %s; Zugriff über domain-Schnittstellen bzw. Dienste führen", fset.Position(imp.Pos()), imported))
		}
		if imported != internalImport+"accounting" || adapter == "accounting" {
			continue
		}
		accountingAlias = "accounting"
		if imp.Name != nil {
			accountingAlias = imp.Name.Name
		}
		if accountingAlias == "." || accountingAlias == "_" {
			problems = append(problems, fmt.Sprintf("%s: accounting braucht einen benannten Import für die Selektorprüfung", fset.Position(imp.Pos())))
		}
	}
	if accountingAlias == "" {
		return problems
	}
	for _, decl := range file.Decls {
		scope := "<Paket>"
		if fn, ok := decl.(*ast.FuncDecl); ok {
			scope = fn.Name.Name
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != accountingAlias || ident.Obj != nil {
				return true
			}
			if catalogSymbols[adapter][sel.Sel.Name] {
				return true
			}
			key := path + ":" + scope + ":" + sel.Sel.Name
			if remaining, ok := exceptions[key]; ok && remaining > 0 {
				exceptions[key]--
				return true
			}
			problems = append(problems, fmt.Sprintf("%s: accounting.%s außerhalb der Katalog-Erlaubtliste (%s). Anwendungslogik in den Dienst verschieben; ein Katalogsymbol nach Prüfung der Erlaubtliste hinzufügen. Bekannte Ausnahmen sind auf ihre Fundstellen begrenzt", fset.Position(sel.Pos()), sel.Sel.Name, key))
			return true
		})
	}
	return problems
}

func TestAccountingImportBoundaries(t *testing.T) {
	exceptions := map[string]int{}
	for key, count := range knownLogicExceptions {
		exceptions[key] = count
	}
	fset := token.NewFileSet()
	for _, adapter := range []string{"accounting", "mcpserver", "wailsbridge"} {
		// Auch Dateien mit Plattform-Tags werden nur geparst. Kein GTK-Build und
		// keine Test-Fixtures: deren Repository-Zugriffe richten Testdaten ein.
		err := filepath.WalkDir(filepath.Join("..", adapter), func(path string, d fs.DirEntry, err error) error {
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
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel("..", path)
			if err != nil {
				return err
			}
			for _, problem := range inspectImports(fset, file, filepath.ToSlash(relative), adapter, exceptions) {
				t.Error(problem)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for key, remaining := range exceptions {
		if remaining != 0 {
			t.Errorf("Ausnahme %s hat %d ungenutzte Freigaben; abgebaute Aufrufe aus der Ausnahmeliste entfernen", key, remaining)
		}
	}
}

func TestImportBoundaryScanner(t *testing.T) {
	for _, tc := range []struct {
		name, adapter, source string
		wantError             bool
	}{
		{"repository", "accounting", `package example; import "github.com/buchfink/buchfink/internal/repository"`, true},
		{"gorm_unterpaket", "accounting", `package example; import "gorm.io/driver/sqlite"`, true},
		{"mcp_repository", "mcpserver", `package example; import "github.com/buchfink/buchfink/internal/repository"`, true},
		{"katalog_alias", "wailsbridge", `package example; import cat "github.com/buchfink/buchfink/internal/accounting"; var x = cat.PostingGroups`, false},
		{"logik_alias", "wailsbridge", `package example; import cat "github.com/buchfink/buchfink/internal/accounting"; var x = cat.BuildVatReturn`, true},
		{"punktimport", "mcpserver", `package example; import . "github.com/buchfink/buchfink/internal/accounting"`, true},
		{"fremdes_paket", "wailsbridge", `package example; import accounting "example.org/accounting"; var x = accounting.BuildVatReturn`, false},
		{"lokaler_name", "wailsbridge", `package example; import "github.com/buchfink/buchfink/internal/accounting"; var x = accounting.PostingGroups; func f(){ accounting := struct{BuildVatReturn int}{}; _ = accounting.BuildVatReturn }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "probe.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			problems := inspectImports(fset, file, "probe.go", tc.adapter, map[string]int{})
			if (len(problems) > 0) != tc.wantError {
				t.Fatalf("Befunde = %v, Fehler erwartet: %t", problems, tc.wantError)
			}
		})
	}
}

func TestImportBoundaryExceptionIsLimitedToItsCallSite(t *testing.T) {
	const source = `package example
 import cat "github.com/buchfink/buchfink/internal/accounting"
 func existing() { _ = cat.AgeOpenItems; _ = cat.AgeOpenItems }
 func added() { _ = cat.AgeOpenItems }
 `
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "probe.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	exceptions := map[string]int{"probe.go:existing:AgeOpenItems": 1}
	problems := inspectImports(fset, file, "probe.go", "mcpserver", exceptions)
	if len(problems) != 2 || exceptions["probe.go:existing:AgeOpenItems"] != 0 {
		t.Fatalf("genau der bestehende Aufruf darf freigegeben sein: %v, verbleibend %v", problems, exceptions)
	}
}
