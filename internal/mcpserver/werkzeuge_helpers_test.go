package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// agent ist ein verbundener Client mit den Ausgabeschemas der Werkzeuge.
// Jede erfolgreiche Antwort wird gegen ihr Schema geprüft, wie Claude Code es
// tut.
type agent struct {
	t       *testing.T
	h       *harness
	cs      *mcp.ClientSession
	schemas map[string]*jsonschema.Resolved
}

func (h *harness) agent(scope string) *agent {
	h.t.Helper()
	cs := h.mustSession(h.login("Claude Code", scope).access)
	listed, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		h.t.Fatal(err)
	}
	a := &agent{t: h.t, h: h, cs: cs, schemas: map[string]*jsonschema.Resolved{}}
	for _, tool := range listed.Tools {
		raw, _ := json.Marshal(tool.OutputSchema)
		var s jsonschema.Schema
		if err := json.Unmarshal(raw, &s); err != nil {
			h.t.Fatalf("%s: %v", tool.Name, err)
		}
		resolved, err := s.Resolve(nil)
		if err != nil {
			h.t.Fatalf("%s: %v", tool.Name, err)
		}
		a.schemas[tool.Name] = resolved
	}
	return a
}

// args ergänzt die Eingabe eines schreibenden Werkzeugs um mandant_id.
func (a *agent) args(in map[string]any) map[string]any {
	out := map[string]any{"mandant_id": a.h.tenant.ID}
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ok ruft ein Werkzeug, erwartet Erfolg, prüft das Schema und liest die
// Antwort in out (darf nil sein).
func (a *agent) ok(tool string, in map[string]any, out any) map[string]any {
	a.t.Helper()
	res := callTool(a.t, a.cs, tool, in)
	if res.IsError {
		a.t.Fatalf("%s scheitert: %s", tool, resultText(res))
	}
	var instance any
	raw, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(raw, &instance)
	if err := a.schemas[tool].Validate(instance); err != nil {
		a.t.Fatalf("%s: Ausgabe passt nicht zum Schema: %v\n%s", tool, err, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			a.t.Fatalf("%s: %v", tool, err)
		}
	}
	m, _ := instance.(map[string]any)
	return m
}

// fail ruft ein Werkzeug und erwartet den Fehler code.
func (a *agent) fail(tool string, in map[string]any, code string) ToolError {
	a.t.Helper()
	e := toolErrorOf(a.t, callTool(a.t, a.cs, tool, in))
	if e.Code != code {
		a.t.Fatalf("%s: Fehler %s (%s), erwartet %s", tool, e.Code, e.Nachricht, code)
	}
	return e
}

// ui führt einen Vorgang in der App aus.
func (h *harness) ui(fn func(ctx context.Context, s *core.Services) error) {
	h.t.Helper()
	if err := h.host.Write(uiCtx, fn); err != nil {
		h.t.Fatal(err)
	}
}

// read liest im geöffneten Mandanten.
func (h *harness) read(fn func(ctx context.Context, s *core.Services) error) {
	h.t.Helper()
	if err := h.host.Read(uiCtx, fn); err != nil {
		h.t.Fatal(err)
	}
}

// buyer ist der Mandant als Rechnungsempfänger.
func buyer() *domain.Contact {
	return &domain.Contact{Name: "Pfennig Ventures GmbH", Street: "Hauptstraße 1", PostalCode: "80331", City: "München",
		CountryCode: "DE", VatID: "DE123456789"}
}

// completeCompany setzt vollständige Unternehmensdaten, wie sie Rechnungen
// und die Prüfung der Pflichtangaben brauchen.
func (h *harness) completeCompany() {
	h.ui(func(ctx context.Context, s *core.Services) error {
		cfg, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		cfg.CompanyName, cfg.LegalForm = "Pfennig Ventures GmbH", domain.LegalFormGmbH
		cfg.ManagingDirectors, cfg.Seat = "Mara Beispiel", "München"
		cfg.RegisterCourt, cfg.RegisterNumber = "Amtsgericht München", "HRB 123456"
		cfg.Street, cfg.PostalCode, cfg.City, cfg.CountryCode = "Hauptstraße 1", "80331", "München", "DE"
		cfg.TaxNumber, cfg.VatID, cfg.FederalState = "143/815/08151", "DE123456789", "BY"
		cfg.FiscalYear, cfg.FiscalYearStartMonth, cfg.Currency, cfg.SKR = 2026, 1, "EUR", "SKR04"
		cfg.VatPeriod, cfg.TaxationType = "quarter", "SOLL"
		return s.Settings.UpdateCompanySettings(ctx, cfg)
	})
}

var supplier = &domain.CompanySettings{CompanyName: "Netzwerk GmbH", Street: "Lieferantenweg 3", PostalCode: "20095",
	City: "Hamburg", CountryCode: "DE", VatID: "DE555666777", TaxNumber: "22/333/44444"}

// vendor legt den Lieferanten der E-Rechnungen in der App an.
func (h *harness) vendor() uint {
	var id uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		c := domain.Contact{Type: domain.ContactTypeVendor, Name: supplier.CompanyName, Street: supplier.Street,
			PostalCode: supplier.PostalCode, City: supplier.City, CountryCode: "DE", VatID: supplier.VatID,
			TaxID: supplier.TaxNumber}
		err := s.Contacts.SaveContact(ctx, &c)
		id = c.ID
		return err
	})
	return id
}

// eInvoice schreibt eine E-Rechnung (CII) des Lieferanten in den Eingang.
func (h *harness) eInvoice(number, date string, net domain.Cents) string {
	h.t.Helper()
	inv := &domain.Invoice{FiscalYear: 2026, ContactID: 1, InvoiceNumber: number, Date: date,
		ServiceDateFrom: date, ServiceDateTo: date, DueDate: date, ContactName: "Pfennig Ventures GmbH",
		TaxTreatment: domain.TaxTreatmentDomestic, Currency: "EUR",
		Items: []domain.InvoiceItem{{Position: 1, Description: "Wartung Server", QuantityMilli: 1000, Unit: "C62",
			UnitPrice: net, TaxRate: domain.TaxRateStandard}}}
	inv.Recalculate()
	xml, err := invoice.GenerateZUGFeRDXML(inv, supplier, buyer())
	if err != nil {
		h.t.Fatalf("E-Rechnung: %v", err)
	}
	return h.inboxFile(number+".xml", []byte(xml))
}

// inboxFile legt eine Datei in den Belegeingang.
func (h *harness) inboxFile(name string, data []byte) string {
	h.t.Helper()
	dir := filepath.Join(h.tenant.DataDir, policy.InboxDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		h.t.Fatal(err)
	}
	return path
}

// warmRenderer übersetzt das Satzprogramm vorab in der App, damit die
// Zeitgrenzen der Werkzeuge nicht die einmalige Übersetzung messen.
func (h *harness) warmRenderer() {
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Receipts.CreateSelfIssued(ctx, service.SelfIssuedReceiptRequest{DocumentDate: "2026-01-02",
			GrossAmount: 100, Reason: "Vorlauf des Satzprogramms"})
		return err
	})
}

func pdf(text string) []byte { return []byte("%PDF-1.4\n% " + text + "\n%%EOF\n") }

func num(v any) uint {
	switch n := v.(type) {
	case float64:
		return uint(n)
	case json.Number:
		i, _ := n.Int64()
		return uint(i)
	}
	panic(fmt.Sprintf("keine Zahl: %v", v))
}
