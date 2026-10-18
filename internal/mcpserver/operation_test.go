package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/ops"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegistrationRejectsAppOnlyAndUnknownOperations(t *testing.T) {
	names := []string{"unbekannter_vorgang"}
	for _, op := range ops.All() {
		if op.Tier == ops.AppOnly {
			names = append(names, op.Name)
		}
	}
	for _, name := range names {
		for wrapper, register := range map[string]func(*Toolbox, Spec){
			"Info": func(tb *Toolbox, s Spec) {
				AddInfo(tb, s, func(context.Context, *Call, struct{}) (struct{}, error) { return struct{}{}, nil })
			},
			"Read": func(tb *Toolbox, s Spec) {
				AddRead(tb, s, func(context.Context, *Call, *core.Services, struct{}) (struct{}, error) { return struct{}{}, nil })
			},
			"Write": func(tb *Toolbox, s Spec) {
				AddWrite(tb, s, func(context.Context, *Call, *core.Services, WriteArgs) (struct{}, error) { return struct{}{}, nil })
			},
			"Flow": func(tb *Toolbox, s Spec) {
				AddFlow(tb, s, func(context.Context, *Call, *core.Tenant, TenantArgs) (struct{}, error) { return struct{}{}, nil })
			},
		} {
			t.Run(name+"/"+wrapper, func(t *testing.T) {
				tb := &Toolbox{srv: mcp.NewServer(&mcp.Implementation{Name: "Test", Version: "test"}, nil)}
				defer func() {
					got := recover()
					if got == nil || !strings.Contains(fmt.Sprint(got), "keine zulässige MCP-Stufe") {
						t.Errorf("Registrierung muss am Operationskatalog scheitern: %v", got)
					}
					if len(tb.Specs()) != 0 {
						t.Error("abgewiesener Vorgang wurde registriert")
					}
				}()
				register(tb, Spec{Name: "alias", Operation: name, Title: "Test", Description: "Test"})
			})
		}
	}
}

func TestRegistrationCannotOverrideCatalogTier(t *testing.T) {
	for _, tier := range []Tier{TierRead, TierDirect} {
		t.Run(string(tier), func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil || !strings.Contains(fmt.Sprint(got), "widerspricht dem Katalog") {
					t.Errorf("abweichende Stufe akzeptiert: %v", got)
				}
			}()
			catalogSpec(Spec{Name: "buchung_erfassen", Tier: tier}, TierRead, TierDirect, TierConfirm)
		})
	}
}
