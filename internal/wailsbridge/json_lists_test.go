package wailsbridge

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Keine Bridge-Methode gibt eine Liste als `nil` zurück.
//
// Der Grund steht in json_lists.go: ein nicht belegter Slice wird über die
// Brücke zu `null`, und die Masken lesen die Antworten ohne Umweg — `rows.map`,
// `paths.length`. Betroffen ist der Randfall, den niemand von Hand ausprobiert,
// nicht der Regelfall: der noch nicht eingerichtete Dienst, der aktive
// Prüfermodus, der Fehler beim Lesen. Genau dort steht in Go am schnellsten ein
// `return nil, err`.
//
// Der Test prüft den Quelltext als Syntaxbaum und nicht den Lauf: die
// Randfälle zur Laufzeit herbeizuführen hieße, die halbe Anwendung
// nachzubauen — und eine neu hinzugefügte Methode fiele trotzdem durch das
// Netz.
func TestNoBridgeMethodReturnsANilList(t *testing.T) {
	methods := bridgeListMethods(t)
	if len(methods) < 50 {
		t.Fatalf("nur %d listenliefernde Bridge-Methoden gefunden — der Test liest den Quelltext offenbar nicht mehr richtig",
			len(methods))
	}
	for _, m := range methods {
		for _, line := range m.nilReturns {
			t.Errorf("%s.%s gibt in Zeile %d eine Liste als nil zurück — erwartet eine leere Liste (siehe emptyList in json_lists.go)",
				m.file, m.name, line)
		}
	}
}

// bridgeListMethod ist eine exportierte Bridge-Methode, deren erstes Ergebnis
// eine Liste ist, mit den Zeilen, in denen sie dafür nil zurückgibt.
type bridgeListMethod struct {
	file       string
	name       string
	nilReturns []int
}

// bridgeListMethods liest die Methoden des Pakets als Syntaxbaum.
func bridgeListMethods(t *testing.T) []bridgeListMethod {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("Paketordner lesen: %v", err)
	}

	out := make([]bridgeListMethod, 0, 64)
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name == nil || !fn.Name.IsExported() {
					continue
				}
				if !isBridgeReceiver(fn.Recv) || !returnsListFirst(fn) {
					continue
				}
				m := bridgeListMethod{file: filepath.Base(name), name: fn.Name.Name}
				for _, pos := range nilFirstResults(fn) {
					m.nilReturns = append(m.nilReturns, fset.Position(pos).Line)
				}
				out = append(out, m)
			}
		}
	}
	return out
}

// returnsListFirst meldet, ob das erste Ergebnis ein Slice ist. Ein Byte-Slice
// zählt nicht mit: er geht als Zeichenkette über die Brücke und nicht als Liste.
func returnsListFirst(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}
	array, ok := fn.Type.Results.List[0].Type.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	ident, ok := array.Elt.(*ast.Ident)
	return !ok || ident.Name != "byte"
}

// nilFirstResults sammelt die Stellen, an denen die Methode nil als erstes
// Ergebnis zurückgibt.
//
// Rümpfe von Funktionsliteralen bleiben außen vor: deren return gehört zur
// inneren Funktion — etwa der Klammer einer Transaktion — und nicht zur
// Antwort der Bridge.
func nilFirstResults(fn *ast.FuncDecl) []token.Pos {
	var out []token.Pos
	var walk func(n ast.Node) bool
	walk = func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			return true
		}
		if ident, ok := ret.Results[0].(*ast.Ident); ok && ident.Name == "nil" {
			out = append(out, ret.Pos())
		}
		return true
	}
	ast.Inspect(fn.Body, walk)
	return out
}
