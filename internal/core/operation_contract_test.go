package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/goldentest"
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
	file, err := parser.ParseFile(token.NewFileSet(), "../wailsbridge/readonly.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		v, ok := node.(*ast.ValueSpec)
		if !ok || v.Names[0].Name != "readOnlyAllowed" {
			return true
		}
		for _, elem := range v.Values[0].(*ast.CompositeLit).Elts {
			key := elem.(*ast.KeyValueExpr).Key.(*ast.BasicLit).Value
			name, err := strconv.Unquote(key)
			if err != nil {
				t.Fatal(err)
			}
			allowed[name] = true
		}
		return false
	})
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
