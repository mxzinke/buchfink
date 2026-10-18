package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Der Graph folgt den tatsächlich gespeicherten Abhängigkeiten, auch durch
// Schnittstellen, eingebettete Module und Listen. Repositories und reine
// Funktionen sind Blätter; registrierte Funktionsrückrufe prüft die Verdrahtung.
func TestServiceDependencyGraph(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Graph GmbH")
	graph := serviceDependencies(reflect.ValueOf(openTenant(t, h).svc).Elem())
	var edges []string
	for from, targets := range graph {
		for to := range targets {
			edges = append(edges, from+" -> "+to)
		}
	}
	slices.Sort(edges)
	t.Logf("Dienstabhängigkeiten (%d):\n%s", len(edges), strings.Join(edges, "\n"))
	cycles := dependencyCycles(graph)
	t.Logf("Zyklische Komponenten (%d):\n%s", len(cycles), strings.Join(cycles, "\n"))
	// Jede entfernte Komponente verkleinert diese Ausgangsliste.
	want := []string{
		"AppropriationService, ClosingBookingService, ClosingService, ObligationService, PeriodService, StatementService",
	}
	if !slices.Equal(cycles, want) {
		t.Fatalf("zyklische Komponenten: %q; erwartet %q", cycles, want)
	}
}

type dependencyGraph map[string]map[string]bool

func serviceDependencies(root reflect.Value) dependencyGraph {
	graph := dependencyGraph{}
	seen := map[uintptr]bool{}
	var visit func(reflect.Value, string)
	visit = func(v reflect.Value, owner string) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface {
			if !v.IsNil() {
				visit(v.Elem(), owner)
			}
			return
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i), owner)
			}
			return
		}
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return
			}
			typ := v.Type().Elem()
			if typ.PkgPath() != "github.com/buchfink/buchfink/internal/service" {
				return
			}
			name := typ.Name()
			if graph[name] == nil {
				graph[name] = map[string]bool{}
			}
			if owner != "" {
				graph[owner][name] = true
			}
			if seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
			visit(v.Elem(), name)
			return
		}
		if v.Kind() != reflect.Struct {
			return
		}
		pkg := v.Type().PkgPath()
		if pkg != "github.com/buchfink/buchfink/internal/service" && pkg != "github.com/buchfink/buchfink/internal/core" {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			visit(v.Field(i), owner)
		}
	}
	visit(root, "")
	return graph
}

// Eine Komponente enthält alle gegenseitig erreichbaren Dienste. Dadurch
// bleibt die Ausgabe auch bei mehreren verschachtelten Kreisen lesbar.
func dependencyCycles(graph dependencyGraph) []string {
	reachable := func(from, to string) bool {
		seen := map[string]bool{}
		var search func(string) bool
		search = func(at string) bool {
			if seen[at] {
				return false
			}
			seen[at] = true
			for next := range graph[at] {
				if next == to || search(next) {
					return true
				}
			}
			return false
		}
		return search(from)
	}
	var nodes []string
	for node := range graph {
		nodes = append(nodes, node)
	}
	slices.Sort(nodes)
	seen := map[string]bool{}
	var cycles []string
	for _, node := range nodes {
		if seen[node] || !reachable(node, node) {
			continue
		}
		var component []string
		for _, other := range nodes {
			if reachable(node, other) && reachable(other, node) {
				component = append(component, other)
				seen[other] = true
			}
		}
		cycles = append(cycles, strings.Join(component, ", "))
	}
	return cycles
}

func TestServiceSetterInventory(t *testing.T) {
	var setters, calls []string
	files, err := filepath.Glob("../service/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !strings.HasPrefix(fn.Name.Name, "Set") {
				continue
			}
			recv := fn.Recv.List[0].Type
			if p, ok := recv.(*ast.StarExpr); ok {
				recv = p.X
			}
			if id, ok := recv.(*ast.Ident); ok {
				setters = append(setters, id.Name+"."+fn.Name.Name)
			}
		}
	}
	source, err := os.ReadFile("wire.go")
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "wire.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !strings.HasPrefix(method.Sel.Name, "Set") {
			return true
		}
		receiver, ok := method.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		calls = append(calls, receiver.Sel.Name+"."+method.Sel.Name)
		return true
	})
	slices.Sort(setters)
	slices.Sort(calls)
	t.Logf("Set*-Methoden (%d):\n%s", len(setters), strings.Join(setters, "\n"))
	t.Logf("Setter-Aufrufe in wire.go (%d):\n%s", len(calls), strings.Join(calls, "\n"))
}
