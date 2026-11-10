package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// auditCall erkennt die Aufrufe, die ins Änderungsprotokoll schreiben:
// audit, logAudit, logAuditChange, Log, LogChange, logAccess und ihre
// Verwandten.
var auditCall = regexp.MustCompile(`^(?i:log|audit)`)

// Ein Protokolleintrag, dessen Fehler verworfen wird, lässt den Vorgang ohne
// Eintrag stehen. Der Test sucht in Diensten, Kern, Bridge und Repositories
// nach Aufrufen, deren Ergebnis niemand ansieht.
func TestAuditResultsAreChecked(t *testing.T) {
	fset := token.NewFileSet()
	for _, dir := range []string{".", "../core", "../wailsbridge", "../repository"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				var call *ast.CallExpr
				switch st := n.(type) {
				case *ast.ExprStmt:
					call, _ = st.X.(*ast.CallExpr)
				case *ast.AssignStmt:
					if len(st.Rhs) == 1 && allBlank(st.Lhs) {
						call, _ = st.Rhs[0].(*ast.CallExpr)
					}
				}
				if call != nil && auditCall.MatchString(calleeName(call)) {
					t.Errorf("%s: das Ergebnis von %s wird verworfen", fset.Position(call.Pos()), calleeName(call))
				}
				return true
			})
		}
	}
}

func calleeName(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

func allBlank(exprs []ast.Expr) bool {
	for _, e := range exprs {
		if id, ok := e.(*ast.Ident); !ok || id.Name != "_" {
			return false
		}
	}
	return true
}
