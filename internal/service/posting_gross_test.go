package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Fahrausweis nennt nur den Bruttobetrag; die Vorsteuer wird daraus
// herausgerechnet (§ 35 Abs. 2 UStDV). Aus dem Nettobetrag gerechnet wären es
// 18,31 € statt 18,30 €.
func TestGrossPositionTakesTheTaxFromTheGross(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Südbahn Fernverkehr AG", "DE", "")

	req := env.receipt(t, vendor.ID, "reisekosten", 0, domain.TaxRateReduced, domain.TaxTreatmentDomestic)
	req.Positions[0].Gross = 27_980
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchung nach der Vorschau: %v", err)
	}
	if got := debitOn(entry, domain.AccountVorsteuer7); got != 1_830 {
		t.Errorf("Vorsteuer %s €, erwartet 18,30 €", got)
	}
	if preview.Gross != 27_980 || preview.Tax != 1_830 {
		t.Errorf("Vorschau brutto %s, Steuer %s; erwartet 279,80 und 18,30", preview.Gross, preview.Tax)
	}
	var credit domain.Cents
	for _, l := range entry.Lines {
		if l.Side == domain.SideCredit {
			credit += l.Amount
		}
	}
	if credit != 27_980 {
		t.Errorf("Gegenbuchung %s €, erwartet der Fahrpreis 279,80 €", credit)
	}
}

// Eine Kleinbetragsrechnung mit zwei Sätzen: die Steuer wird je Satz
// herausgerechnet. Netto und brutto in einem Beleg gibt es nicht.
func TestGrossPositionsPerRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Hafenkante Hotel GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "reisekosten", 0, domain.TaxRateReduced, domain.TaxTreatmentDomestic)
	req.Positions = []ReceiptPosition{
		{PostingGroup: "reisekosten", Gross: 13_803, TaxRate: domain.TaxRateReduced},
		{PostingGroup: "reisekosten", Gross: 2_368, TaxRate: domain.TaxRateStandard},
	}
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if debitOn(entry, domain.AccountVorsteuer7) != 903 || debitOn(entry, domain.AccountVorsteuer19) != 378 {
		t.Errorf("Vorsteuer 7 %% %s, 19 %% %s; erwartet 9,03 und 3,78",
			debitOn(entry, domain.AccountVorsteuer7), debitOn(entry, domain.AccountVorsteuer19))
	}

	mixed := env.receipt(t, vendor.ID, "reisekosten", 1_000, domain.TaxRateReduced, domain.TaxTreatmentDomestic)
	mixed.Positions = append(mixed.Positions, ReceiptPosition{PostingGroup: "reisekosten", Gross: 2_368, TaxRate: domain.TaxRateStandard})
	if _, err := env.posting.PreviewIncomingReceipt(ctx, mixed); err == nil {
		t.Error("netto und brutto gemischt darf nicht durchgehen")
	}
	foreign := env.receipt(t, vendor.ID, "reisekosten", 0, domain.TaxRateNone, domain.TaxTreatmentReverseCharge)
	foreign.Positions[0].Gross = 1_000
	if _, err := env.posting.PreviewIncomingReceipt(ctx, foreign); err == nil {
		t.Error("ohne ausgewiesene Steuer gibt es keinen Bruttobetrag")
	}
}
