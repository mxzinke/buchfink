package mcpserver

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
)

// Die Fassade enthält nur konsumierte Methoden. Ein zusätzlicher Vollzugriff
// oder eine ungenutzte Erweiterung muss bei einer Adapteränderung auffallen.
func TestAgentServicesMatchAdapterUse(t *testing.T) {
	facade := reflect.TypeOf(core.AgentServices{})
	used := map[string]map[string]bool{}
	for i := 0; i < facade.NumField(); i++ {
		field := facade.Field(i)
		if field.Type.Kind() != reflect.Interface {
			t.Fatalf("AgentServices.%s muss eine Schnittstelle sein", field.Name)
		}
		used[field.Name] = map[string]bool{}
		for j := 0; j < field.Type.NumMethod(); j++ {
			method := field.Type.Method(j)
			for k := 0; k < method.Type.NumOut(); k++ {
				out := method.Type.Out(k)
				if out.Kind() == reflect.Ptr {
					out = out.Elem()
				}
				if out.PkgPath() == "github.com/buchfink/buchfink/internal/service" && strings.HasSuffix(out.Name(), "Service") {
					t.Errorf("%s.%s gibt einen vollständigen Dienst zurück", field.Name, method.Name)
				}
			}
		}
	}
	fset := token.NewFileSet()
	for _, dir := range []string{".", "../core/policy"} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			imports := map[string]string{}
			for _, imp := range file.Imports {
				pkg, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				name := filepath.Base(pkg)
				if imp.Name != nil {
					name = imp.Name.Name
				}
				if name == "." {
					t.Errorf("%s: Punktimport umgeht die Adapterprüfung", fset.Position(imp.Pos()))
				}
				imports[name] = pkg
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); ok {
					pkg := imports[id.Name]
					if (pkg == "github.com/buchfink/buchfink/internal/core" && (sel.Sel.Name == "Services" || sel.Sel.Name == "Repos")) ||
						(pkg == "github.com/buchfink/buchfink/internal/service" && strings.HasSuffix(sel.Sel.Name, "Service")) {
						t.Errorf("%s: vollständiger Dienstzugang %s.%s", fset.Position(sel.Pos()), id.Name, sel.Sel.Name)
					}
				}
				if sel.Sel.Name == "Repos" {
					t.Errorf("%s: Repository-Zugriff außerhalb der Fassade", fset.Position(sel.Pos()))
				}
				if field, ok := sel.X.(*ast.SelectorExpr); ok {
					if methods, ok := used[field.Sel.Name]; ok {
						methods[sel.Sel.Name] = true
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	for i := 0; i < facade.NumField(); i++ {
		field := facade.Field(i)
		for j := 0; j < field.Type.NumMethod(); j++ {
			method := field.Type.Method(j)
			if !used[field.Name][method.Name] {
				t.Errorf("AgentServices.%s.%s wird vom Adapter und der Policy nicht genutzt", field.Name, method.Name)
			}
			count++
		}
	}
	t.Logf("MCP und Policy: %d Dienstschnittstellen, %d Methoden", facade.NumField(), count)
}
