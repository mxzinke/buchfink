package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestIncomingHeaderAndReceivedDateAreAtomic(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "buchen", true: "zurueckrollen"}[fail], func(t *testing.T) {
			env := newTestEnv(t)
			ctx := context.Background()
			vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
			req := env.receipt(t, vendor.ID, "software", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
			before, err := env.receipts.SaveHeader(ctx, req.ReceiptID, domain.ReceiptHeader{Kind: domain.ReceiptKindOther, DocumentDate: req.DocumentDate, Subject: "Server", Currency: "EUR"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.posting.PreviewIncomingReceipt(ctx, req); err != nil {
				t.Fatal(err)
			}
			afterPreview, err := env.receipts.Get(ctx, req.ReceiptID)
			if err != nil || !reflect.DeepEqual(before, afterPreview) {
				t.Fatalf("Vorschau ändert Beleg: %v", err)
			}
			if fail {
				req.BookingDate = "ungueltig"
			}
			entry, err := env.posting.PostIncomingReceiptWithHeader(ctx, req, "2026-04-02")
			after, readErr := env.receipts.Get(ctx, req.ReceiptID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if fail {
				if err == nil || entry != nil {
					t.Fatal("Ungültiges Buchungsdatum wurde gebucht")
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("Fehlgeschlagene Buchung ändert Beleg: vorher %+v, nachher %+v", before, after)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if after.IssuerName != vendor.LegalName() || after.GrossAmount != 11900 || after.TaxAmount != 1900 || after.ReceivedAt != "2026-04-02" || after.Status != domain.ReceiptStatusSealed || entry.ReceiptHash != after.ReceiptHash {
					t.Fatalf("Kopf und Buchung: %+v / %+v", after, entry)
				}
			}
		})
	}
}

func TestIncomingHeaderPreservesExistingAndStructuredValues(t *testing.T) {
	for _, tc := range []struct {
		name     string
		receipt  domain.Receipt
		currency string
		foreign  domain.Cents
		want     *domain.ReceiptHeader
	}{
		{name: "eur", receipt: domain.Receipt{Kind: domain.ReceiptKindOther, Subject: "Server"}, want: &domain.ReceiptHeader{Kind: domain.ReceiptKindOther, Subject: "Server", DocumentDate: "2026-03-31", IssuerName: "Lieferant", GrossAmount: 11900, TaxAmount: 1900}},
		{name: "fremdwaehrung", receipt: domain.Receipt{Kind: domain.ReceiptKindOther, TaxAmount: 7}, currency: "USD", foreign: 20000, want: &domain.ReceiptHeader{Kind: domain.ReceiptKindOther, DocumentDate: "2026-03-31", IssuerName: "Lieferant", GrossAmount: 20000, TaxAmount: 7, Currency: "USD"}},
		{name: "vorhanden", receipt: domain.Receipt{DocumentDate: "2026-01-01", IssuerName: "Original", GrossAmount: 1}},
		{name: "teilweise", receipt: domain.Receipt{DocumentDate: "2026-01-01", GrossAmount: 100, TaxAmount: 3, Currency: "USD"}, want: &domain.ReceiptHeader{DocumentDate: "2026-01-01", IssuerName: "Lieferant", GrossAmount: 100, TaxAmount: 3, Currency: "USD"}},
		{name: "strukturiert", receipt: domain.Receipt{Files: []domain.ReceiptFile{{Role: domain.ReceiptRoleStructured}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := incomingReceiptHeader(&incomingPlan{receipt: &tc.receipt, contact: &domain.Contact{Name: "Lieferant"}, request: ReceiptRequest{DocumentDate: "2026-03-31", Currency: tc.currency, ForeignAmount: tc.foreign}, preview: &PostingPreview{Gross: 11900, Tax: 1900}})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Kopf: %+v, erwartet %+v", got, tc.want)
			}
		})
	}
}
