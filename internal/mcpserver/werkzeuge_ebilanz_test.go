package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
)

// Die E-Bilanz geht auch als Summen- und Saldenliste in den Exportordner,
// mit denselben Konten wie die Instanz; blockierende Befunde verhindern sie.
func TestEBilanzAsSuSaCSV(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	a.fail("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026, "format": "susa_csv"}), CodeRule)
	h.capital()

	var out struct {
		Pfad   string `json:"pfad"`
		Format string `json:"format"`
		SHA256 string `json:"sha256"`
	}
	a.ok("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026, "format": "susa_csv"}), &out)
	if out.Pfad != filepath.Join(h.tenant.DataDir, "Export", "ebilanz-2026-susa.csv") || out.Format != "susa_csv" ||
		len(out.SHA256) != 64 {
		t.Fatalf("SuSa: %+v", out)
	}
	data, err := os.ReadFile(out.Pfad)
	if err != nil {
		t.Fatal(err)
	}
	want := "\xEF\xBB\xBFKonto;Bezeichnung;Saldo;Taxonomie\r\n"
	if !strings.HasPrefix(string(data), want) || !strings.Contains(string(data), "\r\n1800;") ||
		!strings.Contains(string(data), ";25000,00;bs.ass.currAss.cashEquiv.bank\r\n") ||
		!strings.Contains(string(data), ";-25000,00;bs.eqLiab.equity.subscribed") {
		t.Errorf("Liste:\n%s", data)
	}

	a.fail("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026, "format": "susa_csv", "umschlag": true}), CodeInvalidInput)
	a.rejected("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026, "format": "pdf"}))
	var xbrl struct {
		Format string `json:"format"`
	}
	a.ok("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026}), &xbrl)
	if xbrl.Format != "xbrl" {
		t.Errorf("Vorgabe: %+v", xbrl)
	}
}

// Die Eröffnungsbilanz geht über dasselbe Werkzeug: Bilanz ohne GuV auf den
// Gründungstag, nur im Geschäftsjahr der Gründung und erst mit den
// Gründungsbuchungen. fristen führt die Pflicht unter Art und Anker.
func TestEBilanzOfTheOpeningBalance(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	h.ui(func(ctx context.Context, s *core.Services) error {
		cfg, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		cfg.FoundedOn = "2026-02-03"
		return s.Settings.UpdateCompanySettings(ctx, cfg)
	})
	a := h.agent("")
	opening := map[string]any{"jahr": 2026, "bilanzart": "eroeffnungsbilanz"}
	if e := a.fail("ebilanz_exportieren", a.args(opening), CodeRule); !strings.Contains(e.Nachricht, "keine Buchung") {
		t.Errorf("ohne Gründungsbuchungen: %+v", e)
	}
	h.capital()
	if e := a.fail("ebilanz_exportieren", a.args(map[string]any{"jahr": 2025, "bilanzart": "eroeffnungsbilanz"}),
		CodeRule); !strings.Contains(e.Nachricht, "Geschäftsjahr 2026") {
		t.Errorf("falsches Jahr: %+v", e)
	}
	a.fail("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026, "bilanzart": "eroeffnungsbilanz",
		"kleinstkapitalgesellschaft": true}), CodeInvalidInput)
	a.rejected("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026, "bilanzart": "schlussbilanz"}))

	var out struct {
		Pfad   string `json:"pfad"`
		Format string `json:"format"`
	}
	a.ok("ebilanz_exportieren", a.args(opening), &out)
	if out.Format != "xbrl" || !strings.HasPrefix(filepath.Base(out.Pfad), "e-bilanz-eroeffnung-2026-") {
		t.Fatalf("Eröffnungsbilanz: %+v", out)
	}
	data, err := os.ReadFile(out.Pfad)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Eröffnungsbilanz", "2026-02-03"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("die Instanz nennt %q nicht", want)
		}
	}
	if strings.Contains(string(data), "is.netIncome") {
		t.Error("Eröffnungsbilanz mit GuV")
	}

	for _, art := range []string{"e-bilanz-eroeffnung", "ebilanz-eroeffnung"} {
		var due struct {
			Pflichten []struct {
				Schluessel string `json:"schluessel"`
				Anker      string `json:"anker"`
				Faellig    string `json:"faellig"`
			} `json:"pflichten"`
		}
		a.ok("fristen", map[string]any{"art": []any{art}, "geschaeftsjahr": 2026}, &due)
		if len(due.Pflichten) != 1 || due.Pflichten[0].Schluessel != "e-bilanz-eroeffnung.gruendung" ||
			due.Pflichten[0].Anker != "ebilanz-eroeffnung" || due.Pflichten[0].Faellig != "2027-08-02" {
			t.Errorf("fristen art=%s: %+v", art, due)
		}
	}
	var one struct {
		Pflichten []struct {
			Schluessel string `json:"schluessel"`
		} `json:"pflichten"`
	}
	a.ok("fristen", map[string]any{"schluessel": "ebilanz-eroeffnung.gruendung"}, &one)
	if len(one.Pflichten) != 1 || one.Pflichten[0].Schluessel != "e-bilanz-eroeffnung.gruendung" {
		t.Errorf("Schlüssel mit Anker: %+v", one)
	}
}
