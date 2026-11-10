package invoice

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Der Hybridabgleich am PDF und nicht nur am XML (RECH-06 K4).
//
// Der Betrieb übergibt VerifyEmbeddedRecord bei ZUGFeRD das PDF und verlässt
// sich darauf, dass der Leser den Datensatz aus dem Anhang holt (siehe
// invoice_service.renderDocument). Bisher prüfte nur der Weg mit dem XML als
// Eingabe — der Weg PDF → XML → Vergleich war der einzige, den die Tests nicht
// gingen, und er läuft in der Anwendung.
//
// Das PDF entsteht hier ohne Typst: gebraucht wird kein gesetztes Dokument,
// sondern ein PDF mit einem Anhang namens factur-x.xml. Ohne diesen Anhang
// lässt sich der Datensatz nicht zurückgewinnen.
func attachXML(t *testing.T, xml string) []byte {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	var base bytes.Buffer
	page := `{"pages": {"1": {"content": {"box": [` +
		`{"x": 50, "y": 50, "width": 200, "height": 100, "fillcol": "#eeeeee"}]}}}}`
	if err := api.Create(nil, strings.NewReader(page), &base, conf); err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	path := filepath.Join(t.TempDir(), "factur-x.xml")
	if err := os.WriteFile(path, []byte(xml), 0o600); err != nil {
		t.Fatalf("Datensatz schreiben: %v", err)
	}
	var out bytes.Buffer
	if err := api.AddAttachments(bytes.NewReader(base.Bytes()), &out, []string{path}, false, conf); err != nil {
		t.Fatalf("Datensatz einbetten: %v", err)
	}
	return out.Bytes()
}

func TestEmbeddedRecordIsReadBackFromThePDF(t *testing.T) {
	inv := testInvoice()
	xml := renderFor(t, inv, testSeller(), testBuyer(), domain.EInvoiceProfileZUGFeRD)

	pdf := attachXML(t, xml)
	if err := VerifyEmbeddedRecord(pdf, inv); err != nil {
		t.Fatalf("das erzeugte Dokument muss durchgehen: %v", err)
	}

	// Dasselbe PDF mit einem manipulierten Anhang: der Bruttobetrag weicht ab.
	tampered := strings.Replace(xml, ">2142.00<", ">2242.00<", 1)
	if tampered == xml {
		t.Fatal("die Stelle des Bruttobetrags steht nicht im erzeugten Datensatz")
	}
	err := VerifyEmbeddedRecord(attachXML(t, tampered), inv)
	if err == nil {
		t.Fatal("ein abweichender Anhang muss vor der Ablage auffallen")
	}
	if !strings.Contains(err.Error(), "Bruttobetrag") {
		t.Errorf("die Meldung nennt die Abweichung nicht: %v", err)
	}

	// Und ein PDF ohne Anhang ist keine E-Rechnung.
	var plain bytes.Buffer
	page := `{"pages": {"1": {"content": {"box": [` +
		`{"x": 50, "y": 50, "width": 200, "height": 100, "fillcol": "#eeeeee"}]}}}}`
	if err := api.Create(nil, strings.NewReader(page), &plain, model.NewDefaultConfiguration()); err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	if err := VerifyEmbeddedRecord(plain.Bytes(), inv); err == nil {
		t.Error("ein PDF ohne eingebetteten Datensatz darf nicht durchgehen")
	}
}
