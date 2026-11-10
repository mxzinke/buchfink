package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// Die Gutschrift eines Lieferanten wird mit getauschten Seiten gebucht:
// Guthaben beim Kreditor, Aufwand und Vorsteuer gemindert (§ 17 Abs. 1 UStG).
// Die Vorsteuer des Monats sinkt um die Steuer der Gutschrift.
func TestCreditNoteReducesExpenseAndInputTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Isarhost GmbH", "DE", "")

	invoice := env.receipt(t, vendor.ID, "software", 8_900, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	invoice.Settlement = SettlementOpen
	env.setReceivedAt(t, invoice.ReceiptID, "2026-03-10")
	if _, err := env.posting.PostIncomingReceipt(ctx, invoice); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}
	credit := env.receipt(t, vendor.ID, "software", 2_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	credit.Settlement = SettlementOpen
	credit.CreditNoteFor = "IH-2026-08-1421"
	env.setReceivedAt(t, credit.ReceiptID, "2026-03-10")
	entry, err := env.posting.PostIncomingReceipt(ctx, credit)
	if err != nil {
		t.Fatalf("Gutschrift: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideCredit, "6495", 2_000},
		{domain.SideCredit, domain.AccountVorsteuer19, 380},
		{domain.SideDebit, vendor.LedgerAccount, 2_380},
	})

	ret := env.vatDraft(t, "2026-Q1")
	if ret.Tax("66") != 1_691-380 {
		t.Errorf("Kz 66 = %s, erwartet 16,91 − 3,80 = 13,11", ret.Tax("66"))
	}
	items, err := env.payments(t).OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var credits int
	for _, item := range items {
		if item.IsCredit() {
			credits++
		}
	}
	if credits != 1 {
		t.Errorf("erwartet ein Guthaben beim Lieferanten, offene Posten %+v", items)
	}

	gift := env.receipt(t, vendor.ID, "software", 1_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	gift.CreditNoteFor = "X"
	gift.Settlement = SettlementShareholderExpense
	if _, err := env.posting.PreviewIncomingReceipt(ctx, gift); err == nil {
		t.Error("eine Gutschrift als Auslage darf nicht durchgehen")
	}
}

// Ein Rechnungsdatensatz vom Typ Gutschrift wird nicht als Rechnung gebucht,
// und der Vorschlag aus ihm ist eine Gutschrift.
func TestCreditNoteXMLIsNotBookedAsAnInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	einvoices := env.einvoices()
	env.posting.SetDocumentKinds(einvoices)
	vendor := env.vendor(t, "Isarhost GmbH", "DE", "")

	inv := &domain.Invoice{FiscalYear: 2026, ContactID: 1, InvoiceNumber: "IH-GS-2026-0042", Date: "2026-03-21",
		ServiceDateFrom: "2026-03-21", ServiceDateTo: "2026-03-21", ContactName: "Buchfink-Mandant",
		TaxTreatment: domain.TaxTreatmentDomestic, Currency: "EUR",
		Items: []domain.InvoiceItem{{Position: 1, Description: "Gutschrift Serverausfall", QuantityMilli: 1000,
			Unit: "C62", UnitPrice: 2_000, TaxRate: domain.TaxRateStandard}}}
	inv.Recalculate()
	xml, err := invoice.GenerateZUGFeRDXML(inv, supplierSettings("Isarhost GmbH", "DE811111111"),
		&domain.Contact{Name: "Buchfink-Mandant", Street: "Hauptstraße 1", PostalCode: "80331", City: "München",
			CountryCode: "DE", VatID: "DE111111111"})
	if err != nil {
		t.Fatal(err)
	}
	xml = replaceTypeCode(xml, "381")
	filed, err := env.receipts.File(ctx, FileReceiptRequest{DocumentDate: "2026-03-21", IssuerName: "Isarhost GmbH",
		GrossAmount: 2_380, Direction: domain.DirectionIncoming, Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "gs.xml", Content: []byte(xml)},
			{Role: domain.ReceiptRoleStructured, FileName: "gs.xml", Content: []byte(xml)},
		}})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := einvoices.Propose(ctx, filed.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal.Request.CreditNoteFor == "" {
		t.Errorf("der Vorschlag muss die Gutschrift kennzeichnen: %+v", proposal.Request)
	}

	req := env.receipt(t, vendor.ID, "software", 2_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.ReceiptID = filed.ID
	if _, err := env.posting.PreviewIncomingReceipt(ctx, req); err == nil {
		t.Error("eine Gutschrift laut Datensatz darf nicht als Rechnung gebucht werden")
	}
	req.CreditNoteFor = proposal.Request.CreditNoteFor
	if _, err := env.posting.PreviewIncomingReceipt(ctx, req); err != nil {
		t.Errorf("als Gutschrift: %v", err)
	}
}

// replaceTypeCode setzt den Rechnungstyp (BT-3) eines CII-Dokuments.
func replaceTypeCode(xml, code string) string {
	return regexp.MustCompile(`<ram:TypeCode>\d+</ram:TypeCode>`).ReplaceAllString(xml, "<ram:TypeCode>"+code+"</ram:TypeCode>")
}
