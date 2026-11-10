package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// fileSmall legt einen Papierbeleg mit den Kopfdaten eines Kleinbetrags ab.
func (e *testEnv) fileSmall(t *testing.T, kind domain.ReceiptKind, issuer string, gross domain.Cents, currency string) *domain.Receipt {
	t.Helper()
	receipt, err := e.receipts.File(context.Background(), FileReceiptRequest{
		DocumentDate: "2026-03-05", IssuerName: issuer, GrossAmount: gross, Currency: currency,
		Direction: domain.DirectionIncoming, Kind: kind,
		Files: []NewFile{{Role: domain.ReceiptRoleOriginal, Path: e.writeTempFile(t, "quittung.pdf", minimalPDF)}},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	return receipt
}

func smallRequest(vendorID, receiptID uint, group string, net domain.Cents, rate domain.TaxRate) ReceiptRequest {
	return ReceiptRequest{
		ContactID: vendorID, ReceiptID: receiptID,
		BookingDate: "2026-03-05", DocumentDate: "2026-03-05",
		ServiceDateFrom: "2026-03-05", ServiceDateTo: "2026-03-05",
		Description: "Quittung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:  []ReceiptPosition{{PostingGroup: group, Net: net, TaxRate: rate}},
		Settlement: SettlementPaid, PaymentAccount: domain.AccountBank,
	}
}

// Die Tankquittung über 59,50 € ist eine Kleinbetragsrechnung (§ 33 UStDV): die
// Steuernummer der Tankstelle gehört nicht zu ihren Pflichtangaben. Der
// Vorsteuerabzug läuft ohne Übersteuerung.
func TestSmallAmountReceiptNeedsNoTaxNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	station := env.incompleteVendor(t, "Tankstelle Nord")
	receipt := env.fileSmall(t, domain.ReceiptKindInvoice, "Tankstelle Nord", 5950, "")

	req := smallRequest(station.ID, receipt.ID, "fahrzeugkosten", 5000, domain.TaxRateStandard)
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(preview.InputTaxFindings) != 0 {
		t.Errorf("Kleinbetragsrechnung: keine Befunde erwartet, erhalten %+v", preview.InputTaxFindings)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("die Kleinbetragsrechnung muss ohne Grund buchbar sein: %v", err)
	}
}

// Über 250 € ist dieselbe Quittung eine gewöhnliche Rechnung, und die fehlende
// Steuernummer hält den Vorsteuerabzug an (§ 14 Abs. 4 Nr. 2 UStG). Es zählt
// der gebuchte Betrag, nicht nur der Kopf.
func TestAboveTheLimitTheTaxNumberIsRequired(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	station := env.incompleteVendor(t, "Tankstelle Nord")
	receipt := env.fileSmall(t, domain.ReceiptKindInvoice, "Tankstelle Nord", 5950, "")

	req := smallRequest(station.ID, receipt.ID, "fahrzeugkosten", 30000, domain.TaxRateStandard)
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(preview.InputTaxFindings) != 1 || preview.InputTaxFindings[0].Code != findingIssuerTaxNumber {
		t.Errorf("357 € brutto: erwartet der Befund zur Steuernummer, erhalten %+v", preview.InputTaxFindings)
	}
}

// Die Grenze von 250 € gilt einschließlich: genau 250 € sind noch ein
// Kleinbetrag, ein Cent mehr nicht.
func TestSmallAmountLimitIsInclusive(t *testing.T) {
	at := &domain.Receipt{Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-05", GrossAmount: 25000}
	if got := reliefOf(at, 25000); got != reliefSmallAmount {
		t.Errorf("250,00 €: Erleichterung %q, erwartet %q", got, reliefSmallAmount)
	}
	above := &domain.Receipt{Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-05", GrossAmount: 25001}
	if got := reliefOf(above, 0); got != reliefNone {
		t.Errorf("250,01 €: Erleichterung %q, erwartet keine", got)
	}
	old := &domain.Receipt{Kind: domain.ReceiptKindInvoice, DocumentDate: "2016-12-30", GrossAmount: 20000}
	if got := reliefOf(old, 0); got != reliefNone {
		t.Errorf("2016 lag die Grenze bei 150 €: Erleichterung %q, erwartet keine", got)
	}
	foreign := &domain.Receipt{Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-05", GrossAmount: 5000, Currency: "USD"}
	if got := reliefOf(foreign, 0); got != reliefNone {
		t.Errorf("Fremdwährung: Erleichterung %q, erwartet keine", got)
	}
}

// Ein Fahrausweis gilt ohne Betragsgrenze als Rechnung (§ 34 UStDV): auch die
// Bahnfahrkarte über 300 € braucht keine Steuernummer.
func TestTicketNeedsNoTaxNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	rail := env.incompleteVendor(t, "Bahn Fernverkehr AG")
	receipt := env.fileSmall(t, domain.ReceiptKindTicket, "Bahn Fernverkehr AG", 35700, "")

	req := smallRequest(rail.ID, receipt.ID, "reisekosten", 30000, domain.TaxRateStandard)
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(preview.InputTaxFindings) != 0 {
		t.Errorf("Fahrausweis: keine Befunde erwartet, erhalten %+v", preview.InputTaxFindings)
	}
}

// Die Klärungsliste nennt für die Parkquittung keine blockierenden Befunde zu
// Steuernummer und eigener Anschrift, sagt aber, welche Erleichterung gilt.
// Name und Anschrift des Ausstellers bleiben verlangt.
func TestClarificationListAppliesTheSmallAmountRelief(t *testing.T) {
	receipt := &domain.Receipt{
		ReceiptNumber: "B-1", Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindInvoice,
		DocumentDate: "2026-03-05", IssuerName: "Parkhaus Mitte", GrossAmount: 1200, TaxAmount: 192,
	}
	operator := &domain.Contact{Name: "Parkhaus Mitte", Street: "Am Markt 1", PostalCode: "20095", City: "Hamburg"}
	company := &domain.CompanySettings{CompanyName: "Muster GmbH"}

	findings := ClassifyReceiptFindings(receipt, ReceiptCheckContext{Contact: operator, Company: company})
	if findings.Blocking != 0 {
		t.Errorf("Parkquittung: %d blockierende Befunde, erwartet keine: %+v", findings.Blocking, findings.Groups)
	}
	rules := map[string]bool{}
	for _, g := range findings.Groups {
		for _, f := range g.Findings {
			rules[f.Rule] = true
		}
	}
	if !rules["content_relief_small_amount"] {
		t.Errorf("die Liste sollte die Kleinbetragsregel nennen: %v", rules)
	}
	if rules["format_no_structured_part"] {
		t.Error("der Hinweis auf das Ende der sonstigen Rechnung gilt für Kleinbeträge nicht (§ 33 Satz 4 UStDV)")
	}

	operator.Street = ""
	findings = ClassifyReceiptFindings(receipt, ReceiptCheckContext{Contact: operator, Company: company})
	if findings.Blocking != 1 {
		t.Errorf("ohne Anschrift des Ausstellers: %d blockierende Befunde, erwartet 1 (§ 33 Satz 1 Nr. 1 UStDV)", findings.Blocking)
	}
}
