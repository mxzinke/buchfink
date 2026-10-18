package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
