package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Ausgangsrechnung wird beim Ausstellen zum Beleg: das hybride PDF als
// empfangene Form, das XML als strukturierter Teil. Damit gilt für die
// Ausgangsseite dasselbe Modell wie für die Eingangsseite.
func TestIssuedInvoiceBecomesASealedReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-08-22",
		ServiceDateFrom: "2026-08-01", ServiceDateTo: "2026-08-31",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Beratungsleistung", QuantityMilli: 1000,
			UnitPrice: 200000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := env.invoicesWithDocuments(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if inv.ReceiptID == nil {
		t.Fatal("die ausgestellte Rechnung muss auf ihren Beleg zeigen")
	}
	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}

	if receipt.Direction != domain.DirectionOutgoing {
		t.Errorf("Richtung = %q, erwartet %q", receipt.Direction, domain.DirectionOutgoing)
	}
	// Der Beleg trägt die Rechnungsnummer — zwei Nummern für dasselbe Dokument
	// wären eine zu viel.
	if receipt.ReceiptNumber != inv.InvoiceNumber {
		t.Errorf("Belegnummer = %q, erwartet die Rechnungsnummer %q", receipt.ReceiptNumber, inv.InvoiceNumber)
	}
	if receipt.Status != domain.ReceiptStatusSealed {
		t.Errorf("der Beleg muss mit der Buchung versiegelt sein, ist aber %q", receipt.Status)
	}
	if receipt.JournalEntryID == nil || *receipt.JournalEntryID != *inv.JournalEntryID {
		t.Error("der Beleg muss auf die Buchung der Rechnung zeigen")
	}

	original, ok := receipt.FileByRole(domain.ReceiptRoleOriginal)
	if !ok || original.MimeType != "application/pdf" {
		t.Fatalf("die empfangene Form muss das PDF sein, ist aber %+v", original)
	}
	if original.Derived {
		t.Error("das PDF ist die erzeugte Rechnung selbst, nicht aus etwas anderem abgeleitet")
	}
	structured, ok := receipt.FileByRole(domain.ReceiptRoleStructured)
	if !ok {
		t.Fatal("der strukturierte Teil fehlt — an ihm hängt der Vorsteuerabzug des Empfängers")
	}
	if !structured.Derived {
		t.Error("das XML stammt aus demselben Dokument und ist damit abgeleitet")
	}
	if !receipt.IsDisplayable() {
		t.Error("die eigene Rechnung muss ansehbar sein")
	}

	// Und das abgelegte PDF ist wirklich eines, mit dem XML darin.
	content, err := env.receipts.Content(ctx, receipt.ID, original.ID)
	if err != nil {
		t.Fatalf("PDF lesen: %v", err)
	}
	if !bytes.HasPrefix(content.Data, []byte("%PDF-")) {
		t.Error("die abgelegte Datei ist kein PDF")
	}
	for _, marker := range []string{"/EmbeddedFiles", "/AFRelationship", "factur-x.xml"} {
		if !bytes.Contains(content.Data, []byte(marker)) {
			t.Errorf("im abgelegten PDF fehlt %q", marker)
		}
	}
	if !content.Intact {
		t.Error("die frisch abgelegte Datei muss zu ihrer Prüfsumme passen")
	}
}

// Ohne Beleg-Pipeline bleibt das Ausstellen möglich — die Buchung ist der Kern,
// das Dokument kommt aus einer Schicht darüber.
func TestIssuingWorksWithoutTheDocumentPipeline(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-08-22",
		ServiceDateFrom: "2026-08-01", ServiceDateTo: "2026-08-31",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Beratung", QuantityMilli: 1000,
			UnitPrice: 200000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.JournalEntryID == nil {
		t.Error("die Rechnung muss gebucht sein")
	}
	if inv.ReceiptID != nil {
		t.Error("ohne Pipeline darf kein Beleg entstehen")
	}
}

// Scheitert die Buchung, darf der eben abgelegte Ausgangsbeleg nicht als offener
// Beleg zurückbleiben. Er trägt die Rechnungsnummer in einem eindeutigen Index:
// jeder weitere Versuch verbrauchte sonst die nächste Nummer, und die vergebene
// wäre für immer belegt, ohne dass je eine Rechnung dieses Namens existierte.
func TestFailedPostingLeavesNoOpenOutgoingReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	// Istversteuerung lässt die Buchung scheitern, nachdem der Beleg abgelegt ist.
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen: %v", err)
	}
	cfg.TaxationType = "IST"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-05-04",
		ServiceDateFrom: "2026-05-04", ServiceDateTo: "2026-05-04",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Beratung", QuantityMilli: 1000,
			UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := env.invoicesWithDocuments(t).Issue(ctx, inv); err == nil {
		t.Fatal("die Rechnung wurde trotz Istversteuerung ausgestellt")
	}

	receipts, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belegliste: %v", err)
	}
	for _, r := range receipts {
		if r.Direction != domain.DirectionOutgoing {
			continue
		}
		if r.Status == domain.ReceiptStatusFiled {
			t.Errorf("Beleg %s ist nach der gescheiterten Buchung noch offen", r.ReceiptNumber)
		}
		if r.DiscardReason == "" {
			t.Errorf("Beleg %s wurde ohne Begründung verworfen", r.ReceiptNumber)
		}
	}
}
