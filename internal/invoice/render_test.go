package invoice

import (
	"bytes"
	"context"
	"regexp"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
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
