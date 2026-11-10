package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// Die App zeigt eine reine XRechnung aus ihrem Datensatz: Parteien,
// Positionen, Steuer und Summen, wie die XML sie nennt.
func TestEInvoiceViewShowsTheXML(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	inv := &domain.Invoice{
		FiscalYear: 2026, ContactID: 1, InvoiceNumber: "MF-2026-0912-77812",
		Date: "2026-09-15", ServiceDateFrom: "2026-08-15", ServiceDateTo: "2026-09-14",
		DueDate: "2026-09-29", ContactName: "Kranich Software GmbH",
		TaxTreatment: domain.TaxTreatmentDomestic, Currency: "EUR",
		Items: []domain.InvoiceItem{{Position: 1, Description: "Glasfaser 1000", QuantityMilli: 1000,
			Unit: "MON", UnitPrice: 5_990, TaxRate: domain.TaxRateStandard}},
	}
	inv.Recalculate()
	xml, err := invoice.GenerateZUGFeRDXML(inv, supplierSettings("MUC Fiber GmbH", "DE811111111"),
		&domain.Contact{Name: "Kranich Software GmbH", Street: "Schwanthalerstraße 64", PostalCode: "80336",
			City: "München", CountryCode: "DE", VatID: "DE283746151"})
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	filed, err := env.receipts.File(ctx, FileReceiptRequest{
		DocumentDate: "2026-09-15", IssuerName: "MUC Fiber GmbH", GrossAmount: 7_128, Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "mf.xml", Content: []byte(xml)},
			{Role: domain.ReceiptRoleStructured, FileName: "mf.xml", Content: []byte(xml)},
		},
	})
	if err != nil {
		t.Fatalf("ablegen: %v", err)
	}

	view, err := env.einvoices().View(ctx, filed.ID)
	if err != nil {
		t.Fatalf("Ansicht: %v", err)
	}
	if view.Number != "MF-2026-0912-77812" || view.Seller.Name != "MUC Fiber GmbH" || view.TypeLabel != "Rechnung" {
		t.Errorf("Kopf: %+v", view)
	}
	if len(view.Lines) != 1 || view.Lines[0].Name == "" || view.Lines[0].Unit != "Monat" {
		t.Errorf("Positionen: %+v", view.Lines)
	}
	if view.Totals.DuePayable != "71.28" || len(view.VAT) != 1 || view.VAT[0].TaxAmount != "11.38" {
		t.Errorf("Summen %+v, Steuer %+v", view.Totals, view.VAT)
	}
	if !view.Intact || view.PeriodFrom != "2026-08-15" {
		t.Errorf("Prüfsumme %v, Zeitraum %s", view.Intact, view.PeriodFrom)
	}
}
