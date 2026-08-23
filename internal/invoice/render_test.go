package invoice

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/zugferd"
)

func sampleInvoice() (*domain.Invoice, *domain.CompanySettings, *domain.Contact) {
	inv := &domain.Invoice{
		FiscalYear:      2026,
		ContactID:       1,
		InvoiceNumber:   "RE-2026-0001",
		Date:            "2026-08-22",
		ServiceDateFrom: "2026-08-01", ServiceDateTo: "2026-08-31",
		DueDate:      "2026-09-05",
		ContactName:  "Kunde GmbH",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Currency:     "EUR",
		Items: []domain.InvoiceItem{{
			Position: 1, Description: "Beratungsleistung", QuantityMilli: 1000,
			Unit: "Std", UnitPrice: 200000, TaxRate: domain.TaxRateStandard,
		}},
	}
	inv.Recalculate()

	seller := &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", Street: "Hauptstraße 1", ZipCity: "80331 München",
		TaxNumber: "143/815/08151", VatID: "DE123456789",
		BankName: "Sparkasse", IBAN: "DE02120300000000202051", BIC: "BYLADEM1001",
	}
	buyer := &domain.Contact{Name: "Kunde GmbH", Address: "Kundenweg 2\n10115 Berlin", CountryCode: "DE"}
	return inv, seller, buyer
}

// Die eingebettete Datei muss text/xml sein — so schreibt es die
// ZUGFeRD-Spezifikation vor, und darauf prüfen die Prüfprogramme der Empfänger. Der Wert steht in internal/einvoice/zugferd; die Vorlage darf nicht
// davon abweichen.
func TestTemplateDeclaresTheAttachmentMimeType(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	template := GenerateTypstTemplate(inv, seller, buyer)
	if !strings.Contains(template, `mime-type: "`+zugferd.MimeType+`"`) {
		t.Errorf("die Vorlage deklariert nicht %q als MIME-Typ", zugferd.MimeType)
	}
}

// Die Ausgangsrechnung entsteht als hybrides PDF: PDF/A-3b mit dem
// ZUGFeRD-XML als zugeordneter Datei. Beides in einem Schritt, ohne
// Nachbearbeitung.
func TestRenderProducesAHybridPDFA3(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	inv, seller, buyer := sampleInvoice()

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("ZUGFeRD-XML: %v", err)
	}

	renderer := NewRenderer()
	ctx := context.Background()
	t.Cleanup(func() { _ = renderer.Close(ctx) })

	pdf, err := renderer.RenderInvoicePDF(ctx, inv, seller, buyer, xml)
	if err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("das Ergebnis ist kein PDF")
	}

	// PDF/A-3, Konformitätsstufe B — das ist die Stufe, die eingebettete Dateien
	// überhaupt erlaubt.
	if m := regexp.MustCompile(`pdfaid:part>\s*(\d)`).FindSubmatch(pdf); m == nil || string(m[1]) != "3" {
		t.Error("das PDF weist sich nicht als PDF/A-3 aus")
	}
	if m := regexp.MustCompile(`pdfaid:conformance>\s*(\w)`).FindSubmatch(pdf); m == nil || string(m[1]) != "B" {
		t.Error("das PDF weist sich nicht als Konformitätsstufe B aus")
	}

	// Der Anhang muss als "Alternative" gekennzeichnet sein: PDF und XML sind
	// zwei Darstellungen derselben Rechnung. Alles andere ist für die Profile
	// BASIC und EN 16931 in Deutschland nicht rechtsgültig.
	for _, marker := range []string{"/EmbeddedFiles", "/AFRelationship", "/Alternative", "factur-x.xml"} {
		if !bytes.Contains(pdf, []byte(marker)) {
			t.Errorf("im PDF fehlt %q", marker)
		}
	}
}

// Ohne strukturierten Teil entsteht keine E-Rechnung — daran hängt der
// Vorsteuerabzug des Empfängers.
func TestRenderRefusesWithoutStructuredData(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	renderer := NewRenderer()
	if _, err := renderer.RenderInvoicePDF(context.Background(), inv, seller, buyer, ""); err == nil {
		t.Error("ohne ZUGFeRD-XML darf kein hybrides PDF entstehen")
	}
}

// Der Kundenname darf das Dokument nicht sprengen: Typst-Markup wird neutralisiert.
func TestRenderSurvivesMarkupInMasterData(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	inv, seller, buyer := sampleInvoice()
	buyer.Name = `Müller #[weird] *bold* $x$ \ GmbH`
	buyer.Address = "Straße [1]\n#12345 Ort"
	seller.CompanyName = `Ventures "GmbH" \ & Co.`
	inv.Items[0].Description = "Beratung #1 [Phase *A*]"

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("ZUGFeRD-XML: %v", err)
	}
	renderer := NewRenderer()
	ctx := context.Background()
	t.Cleanup(func() { _ = renderer.Close(ctx) })

	if _, err := renderer.RenderInvoicePDF(ctx, inv, seller, buyer, xml); err != nil {
		t.Fatalf("Stammdaten mit Sonderzeichen dürfen das PDF nicht scheitern lassen: %v", err)
	}
}

func TestTypstDateHandlesMissingAndMalformedDates(t *testing.T) {
	cases := map[string]string{
		"2026-08-22": "datetime(year: 2026, month: 8, day: 22)",
		"":           "auto",
		"kaputt":     "auto",
	}
	for input, want := range cases {
		if got := typstDate(input); got != want {
			t.Errorf("typstDate(%q) = %q, erwartet %q", input, got, want)
		}
	}
}

// Der Rundlauf über das Dokument: erzeugen, das XML wieder herausziehen, lesen.
// Das ist der Weg, den eine empfangene E-Rechnung nimmt.
func TestEmbeddedInvoiceCanBeExtractedAgain(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	inv, seller, buyer := sampleInvoice()

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("ZUGFeRD-XML: %v", err)
	}
	renderer := NewRenderer()
	ctx := context.Background()
	t.Cleanup(func() { _ = renderer.Close(ctx) })

	pdf, err := renderer.RenderInvoicePDF(ctx, inv, seller, buyer, xml)
	if err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}

	embedded, err := ExtractEmbeddedInvoice(pdf)
	if err != nil {
		t.Fatalf("XML aus dem PDF holen: %v", err)
	}
	if embedded.Name != "factur-x.xml" {
		t.Errorf("Dateiname des Anhangs = %q, erwartet factur-x.xml", embedded.Name)
	}
	if string(embedded.Data) != xml {
		t.Error("das herausgeholte XML weicht vom eingebetteten ab")
	}

	doc, err := einvoice.ParseCII(embedded.Data)
	if err != nil {
		t.Fatalf("XML lesen: %v", err)
	}
	if doc.Number != inv.InvoiceNumber {
		t.Errorf("Rechnungsnummer nach dem Rundlauf = %q", doc.Number)
	}
}

// Ein PDF ohne eingebetteten Datensatz ist eine sonstige Rechnung — und die
// Meldung soll das sagen, statt nur "nichts gefunden".
func TestPlainPDFIsReportedAsSonstigeRechnung(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	renderer := NewRenderer()
	ctx := context.Background()
	t.Cleanup(func() { _ = renderer.Close(ctx) })
	if err := renderer.Warm(ctx); err != nil {
		t.Fatalf("Renderer: %v", err)
	}

	pdf, err := renderer.compilePlainForTest(ctx)
	if err != nil {
		t.Fatalf("PDF ohne Anhang: %v", err)
	}
	if _, err := ExtractEmbeddedInvoice(pdf); err == nil {
		t.Fatal("ein PDF ohne Anhang darf keinen Rechnungsdatensatz liefern")
	} else if !strings.Contains(err.Error(), "sonstige Rechnung") {
		t.Errorf("die Meldung soll den Fall benennen, lautet aber: %v", err)
	}
}
