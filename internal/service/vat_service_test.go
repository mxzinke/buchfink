// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func (e *testEnv) vat() *VatService {
	return NewVatService(e.journalRepo, e.fiscalYear)
}

// Die Zahllast folgt aus den Steuerzeilen des Journals, nicht aus einer
// Auswertung von Kontonummern im Frontend.
func TestVatSummaryFromJournal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde", "DE", "")
	vendor := env.vendor(t, "Lieferant", "DE", "")

	// Ausgangsrechnung 2.000,00 netto, 19 % → 380,00 Umsatzsteuer.
	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-02-10",
		ServiceDateFrom: "2026-02-10", ServiceDateTo: "2026-02-10",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items:        []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 200000, TaxRate: domain.TaxRateStandard}},
	}
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	// Eingangsbeleg 1.000,00 netto, 19 % → 190,00 Vorsteuer.
	if _, err := env.posting.PostIncomingReceipt(ctx,
		receipt(vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
		t.Fatalf("Eingangsbeleg: %v", err)
	}

	summary, err := env.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}

	if len(summary.TaxableRevenue) != 1 {
		t.Fatalf("erwartet eine Steuersatzgruppe, erhalten %d", len(summary.TaxableRevenue))
	}
	group := summary.TaxableRevenue[0]
	if group.Rate != domain.TaxRateStandard || group.Net != 200000 || group.Tax != 38000 {
		t.Errorf("19 %%-Gruppe: %s netto / %s Steuer, erwartet 2.000,00 / 380,00", group.Net, group.Tax)
	}
	if summary.InputTax != 19000 {
		t.Errorf("Vorsteuer = %s, erwartet 190,00", summary.InputTax)
	}
	if summary.Payable != 19000 {
		t.Errorf("Zahllast = %s, erwartet 190,00 (380,00 − 190,00)", summary.Payable)
	}
}

// Reverse Charge erhöht die geschuldete Steuer und die Vorsteuer um denselben
// Betrag — die Zahllast bleibt unberührt, die Kennzahlen nicht.
func TestVatSummaryCountsReverseChargeOnBothSides(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "SaaS Ireland Ltd.", "IE", "IE6388047V")

	if _, err := env.posting.PostIncomingReceipt(ctx,
		receipt(vendor.ID, "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge)); err != nil {
		t.Fatalf("§ 13b-Beleg: %v", err)
	}

	summary, err := env.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}

	if summary.ReverseChargeTax != 19000 {
		t.Errorf("geschuldete § 13b-Steuer = %s, erwartet 190,00", summary.ReverseChargeTax)
	}
	if summary.ReverseChargeBase != 100000 {
		t.Errorf("Bemessungsgrundlage § 13b = %s, erwartet 1.000,00", summary.ReverseChargeBase)
	}
	if summary.InputTax != 19000 {
		t.Errorf("Vorsteuer = %s, erwartet 190,00", summary.InputTax)
	}
	if summary.Payable != 0 {
		t.Errorf("Zahllast = %s, erwartet 0,00 — § 13b ist zahlungsneutral, aber meldepflichtig", summary.Payable)
	}
}

// Eine Generalumkehr rechnet sich in der Auswertung von selbst heraus, weil ihre
// Beträge negativ auf derselben Seite stehen.
func TestVatSummaryNetsOutReversals(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-02-10",
		ServiceDateFrom: "2026-02-10", ServiceDateTo: "2026-02-10",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items:        []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 200000, TaxRate: domain.TaxRateStandard}},
	}
	invoices := env.invoices(t)
	if err := invoices.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}
	if err := invoices.Cancel(ctx, inv.ID, "Leistung nicht erbracht"); err != nil {
		t.Fatalf("Storno: %v", err)
	}

	summary, err := env.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}
	if summary.OutputTax != 0 {
		t.Errorf("nach dem Storno muss die Umsatzsteuer null sein, ist aber %s", summary.OutputTax)
	}
	for _, g := range summary.TaxableRevenue {
		if g.Net != 0 {
			t.Errorf("nach dem Storno muss die Bemessungsgrundlage null sein, ist aber %s", g.Net)
		}
	}
}

// Der Zeitraumfilter grenzt nach dem Buchungsdatum ab.
func TestVatSummaryRespectsPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	first := receipt(vendor.ID, "buerobedarf", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	first.BookingDate = "2026-01-15"
	first.DocumentDate = "2026-01-15"
	if _, err := env.posting.PostIncomingReceipt(ctx, first); err != nil {
		t.Fatalf("Januar-Beleg: %v", err)
	}

	second := receipt(vendor.ID, "buerobedarf", 50000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	second.BookingDate = "2026-04-15"
	second.DocumentDate = "2026-04-15"
	if _, err := env.posting.PostIncomingReceipt(ctx, second); err != nil {
		t.Fatalf("April-Beleg: %v", err)
	}

	q1, err := env.vat().Summary(ctx, "2026-01-01", "2026-03-31")
	if err != nil {
		t.Fatalf("Q1: %v", err)
	}
	if q1.InputTax != 19000 {
		t.Errorf("Vorsteuer Q1 = %s, erwartet 190,00", q1.InputTax)
	}

	year, _ := env.vat().Summary(ctx, "", "")
	if year.InputTax != 28500 {
		t.Errorf("Vorsteuer Gesamtjahr = %s, erwartet 285,00", year.InputTax)
	}
}
