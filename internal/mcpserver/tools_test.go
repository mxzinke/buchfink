package mcpserver

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Weist ein Dienst einen Vorgang dem Menschen in der App zu, erfährt der
// Agent das als nur_in_app mit Ausweg, nicht als fachliche Regel.
func TestAppOnlyRefusalOfAServiceIsMarked(t *testing.T) {
	channel, err := actor.Agent("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	ctx := actor.WithChannel(context.Background(), channel)
	err = (&service.SettingsService{}).UpdateCompanySettings(ctx, &domain.CompanySettings{})
	te := toolError(err)
	if te.Code != CodeAppOnly || te.Werkzeug != "nur_app" || te.NaechsterSchritt == "" ||
		!strings.Contains(te.Nachricht, "Unternehmensdaten") {
		t.Errorf("Abweisung des Dienstes: %+v", te)
	}
}

// Werkzeuge arbeiten über die Dienste. Die Repositories darunter kennen die
// Vorbehalte der App nicht (requireHuman); ein Werkzeug, das sie anfasst,
// schriebe daran vorbei. Nur das Protokoll der Zulassungen schreibt der
// Server selbst, unter dem Erfassungsweg ui.
func TestToolsDoNotTouchRepositories(t *testing.T) {
	allowed := map[string]bool{"logInTenant": true}
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && allowed[fn.Name.Name] {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Repos" {
					t.Errorf("%s: Zugriff auf Repos; Werkzeuge gehen über die Dienste", fset.Position(sel.Pos()))
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
