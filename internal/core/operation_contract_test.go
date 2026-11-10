package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/goldentest"
	"github.com/buchfink/buchfink/internal/ops"
)

// Die Quelltextprüfung läuft auch ohne die GTK-Abhängigkeiten der Bridge.
func operationFunctions(t *testing.T, dir string) map[string]*ast.FuncDecl {
	t.Helper()
	pkgs, err := parser.ParseDir(token.NewFileSet(), dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*ast.FuncDecl{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				name := fn.Name.Name
				if fn.Recv != nil {
					recv := fn.Recv.List[0].Type
					if ptr, ok := recv.(*ast.StarExpr); ok {
						recv = ptr.X
					}
					name = calledName(recv) + "." + name
				}
				out[pkg.Name+"."+name] = fn
			}
		}
	}
	return out
}

func calledName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.IndexExpr:
		return calledName(expr.X)
	}
	return ""
}

func humanGuardSites(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, dir := range []string{"core", "service", "wailsbridge"} {
		for name, fn := range operationFunctions(t, filepath.Join("..", dir)) {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					guard := calledName(call.Fun)
					if guard == "AppOnly" || guard == "requireHuman" || guard == "requireApp" {
						out[name] = append(out[name], guard)
					}
				}
				return true
			})
			slices.Sort(out[name])
		}
	}
	return out
}

func TestGoldenHumanGuardSites(t *testing.T) {
	goldentest.Assert(t, "human_guard_sites", humanGuardSites(t))
}

type bridgeClassification struct {
	ExaminerAllowed bool
	Guarded         bool
	AppOnly         bool
}

func bridgeClassifications(t *testing.T) map[string]bridgeClassification {
	t.Helper()
	allowed := ops.BridgeExaminerAllowed()
	out := map[string]bridgeClassification{}
	for name, fn := range operationFunctions(t, "../wailsbridge") {
		if !strings.HasPrefix(name, "wailsbridge.BuchfinkBridge.") || !fn.Name.IsExported() {
			continue
		}
		entry := bridgeClassification{ExaminerAllowed: allowed[fn.Name.Name]}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !slices.Contains([]string{"write", "change", "prepared", "operate"}, calledName(call.Fun)) {
				return true
			}
			exempt := false
			for _, arg := range call.Args {
				if option, ok := arg.(*ast.CallExpr); ok {
					switch calledName(option.Fun) {
					case "AppOnly":
						entry.AppOnly = true
					case "ExaminerExempt":
						exempt = true
					}
				}
			}
			entry.Guarded = entry.Guarded || !exempt
			return true
		})
		out[fn.Name.Name] = entry
	}
	if len(out) < 100 || len(allowed) < 100 {
		t.Fatalf("unvollständige Bridge-Klassifikation: %d Methoden, %d Ausnahmen", len(out), len(allowed))
	}
	return out
}

func TestGoldenBridgeClassification(t *testing.T) {
	goldentest.Assert(t, "bridge_classification", bridgeClassifications(t))
}

func TestAppOnlyCatalogCoversGuardSites(t *testing.T) {
	actual := humanGuardSites(t)
	covered := map[string]string{}
	for _, op := range ops.All() {
		if op.Tier != ops.AppOnly || op.Enforcement != ops.Channel {
			continue
		}
		for _, site := range op.Guards {
			if previous := covered[site]; previous != "" {
				t.Errorf("%s gehört zu %s und %s", site, previous, op.Name)
			}
			covered[site] = op.Name
			if len(actual[site]) == 0 {
				t.Errorf("%s: Schutzstelle %s fehlt", op.Name, site)
			}
		}
	}
	for site := range actual {
		if strings.HasPrefix(site, "wailsbridge.") {
			continue
		}
		if covered[site] == "" {
			t.Errorf("Schutzstelle %s fehlt im Operationskatalog", site)
		}
	}
}

func TestBridgeClassificationMatchesOperations(t *testing.T) {
	methods := bridgeClassifications(t)
	functions := operationFunctions(t, "../wailsbridge")
	for name, method := range methods {
		if method.Guarded == method.ExaminerAllowed {
			t.Errorf("%s: Prüfermodus-Ausnahme passt nicht zum Schreibweg", name)
		}
		op, reserved := ops.ForBridge(name)
		if method.AppOnly && (!reserved || op.Tier != ops.AppOnly) {
			t.Errorf("%s: AppOnly-Kennzeichnung ohne Katalogeintrag", name)
		}
	}
	for name := range ops.BridgeExaminerAllowed() {
		if _, ok := methods[name]; !ok {
			t.Errorf("Prüfermodus-Ausnahme ohne Bridge-Methode: %s", name)
		}
	}
	for _, op := range ops.All() {
		for _, name := range op.Bridge {
			method, ok := methods[name]
			if !ok {
				t.Errorf("%s: Bridge-Methode %s fehlt", op.Name, name)
				continue
			}
			if method.AppOnly {
				continue
			}
			delegates := false
			ast.Inspect(functions["wailsbridge.BuchfinkBridge."+name].Body, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					for _, guard := range op.Guards {
						delegates = delegates || strings.HasSuffix(guard, "."+calledName(call.Fun))
					}
				}
				return true
			})
			if !delegates {
				t.Errorf("%s: weder core.AppOnly noch Delegation an eine Schutzstelle von %s", name, op.Name)
			}
		}
	}
}

// Jeder Verweis auf eine private Verwaltungs-API muss innerhalb der Kette
// Bridge → Server → OAuth bleiben. Auch Methodenzeiger (HTTP-Handler oder
// Werkzeug-Callbacks) zählen; geprüft wird bis zur Änderung im OAuth-Store.
func TestAppOnlyAPIsHaveOnlyBridgeCallers(t *testing.T) {
	allowed := map[string]map[string]bool{}
	for _, op := range ops.All() {
		if op.Enforcement != ops.Structural {
			continue
		}
		if op.Tier != ops.AppOnly || len(op.Guards) == 0 || len(op.Bridge) == 0 {
			t.Fatalf("unvollständige strukturelle Grenze: %+v", op)
		}
		callers := make([]string, 0, len(op.Bridge))
		for _, method := range op.Bridge {
			callers = append(callers, "wailsbridge.BuchfinkBridge."+method)
			allowed[method] = map[string]bool{}
		}
		for _, site := range op.Guards {
			name := site[strings.LastIndex(site, ".")+1:]
			if allowed[name] == nil {
				allowed[name] = map[string]bool{}
			}
			for _, caller := range callers {
				allowed[name][caller] = true
			}
			callers = []string{site}
		}
	}
	seen := map[string]bool{}
	for _, dir := range []string{"../wailsbridge", "../mcpserver", "../mcpserver/oauth"} {
		for caller, fn := range operationFunctions(t, dir) {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				sel, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if callers, protected := allowed[sel.Sel.Name]; protected {
					seen[sel.Sel.Name] = true
					if !callers[caller] {
						t.Errorf("%s erreicht die private App-Verwaltung %s", caller, sel.Sel.Name)
					}
				}
				return true
			})
		}
		pkgs, err := parser.ParseDir(token.NewFileSet(), dir, func(info os.FileInfo) bool {
			return !strings.HasSuffix(info.Name(), "_test.go")
		}, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, decl := range file.Decls {
					if _, function := decl.(*ast.FuncDecl); function {
						continue
					}
					ast.Inspect(decl, func(node ast.Node) bool {
						if sel, ok := node.(*ast.SelectorExpr); ok && allowed[sel.Sel.Name] != nil {
							t.Errorf("%s: globaler Verweis auf private App-Verwaltung %s", dir, sel.Sel.Name)
						}
						return true
					})
				}
			}
		}
	}
	for name, callers := range allowed {
		if len(callers) > 0 && !seen[name] {
			t.Errorf("private API %s wird nicht mehr erreicht; Katalog prüfen", name)
		}
	}
}
