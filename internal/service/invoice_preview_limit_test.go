package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Vorschau nennt die Grenze der Kleinbetragsrechnung, und zwar die des
// Rechnungsdatums (§ 33 UStDV: 150 Euro bis 2016, seither 250 Euro).
//
// Ohne sie bietet der Rechnungsdialog die Kleinbetragsrechnung unabhängig vom
// Betrag an und lässt den Anwender bis zur Fehlermeldung des Ausstellens
// laufen. Die Grenze im Frontend nachzubauen hieße, eine datierte Zahl ein
// zweites Mal zu führen — deshalb reist sie mit der Vorschau, in der auch der
// Bruttobetrag steht, gegen den sie zu vergleichen ist.
func TestPreviewReportsDatedSmallAmountLimit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	now := env.simpleInvoice(customer.ID, "2026-03-01", 10000)
	preview, err := svc.Preview(ctx, now)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.SmallAmountLimit != 25000 {
		t.Errorf("Grenze 2026 = %s, erwartet 250,00", preview.SmallAmountLimit)
	}
	// Die Vorschau liefert den Bruttobetrag, gegen den die Maske vergleicht.
	if preview.Gross != 11900 {
		t.Errorf("brutto = %s, erwartet 119,00", preview.Gross)
	}

	old := env.simpleInvoice(customer.ID, "2016-03-01", 10000)
	preview, err = svc.Preview(ctx, old)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.SmallAmountLimit != 15000 {
		t.Errorf("Grenze 2016 = %s, erwartet 150,00", preview.SmallAmountLimit)
	}
}

// Auch der Barverkauf bekommt die Grenze: er ist der Fall, in dem die
// Kleinbetragsrechnung überhaupt erst ohne Empfänger auskommt, und die Maske
// muss dort dieselbe Sperre ziehen können.
func TestCashSalePreviewReportsSmallAmountLimit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)

	cash := &domain.Invoice{
		Date: "2026-03-01", ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		TaxTreatment: domain.TaxTreatmentDomestic,
		SmallAmount:  true,
		Items: []domain.InvoiceItem{{
			Description: "Ladenverkauf", QuantityMilli: 1000, Unit: "C62",
			UnitPrice: 5000, TaxRate: domain.TaxRateStandard,
		}},
	}
	preview, err := svc.Preview(ctx, cash)
	if err != nil {
		t.Fatalf("Vorschau des Barverkaufs: %v", err)
	}
	if preview.SmallAmountLimit != 25000 {
		t.Errorf("Grenze = %s, erwartet 250,00", preview.SmallAmountLimit)
	}
}
