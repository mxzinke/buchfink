package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Trinkgeld auf der Rechnung der Gaststätte trägt keine Steuer und steht
// in derselben Buchung wie die Bewirtung: ohne Vorsteuer, aber im Aufwand
// 70/30 geteilt. Ein Satz von null
// ohne Grund bleibt abgewiesen.
func TestNotTaxablePositionOnADomesticReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Kutterhus am Hafen GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 10_000, domain.TaxRateReduced, domain.TaxTreatmentDomestic)
	req.Entertainment = &domain.EntertainmentDetail{Place: "Kutterhus, Hamburg", Day: "2026-03-10",
		Participants: "Henrik Sørensen, Line Madsen", Occasion: "Projektanbahnung"}
	req.Positions = append(req.Positions, ReceiptPosition{PostingGroup: "bewirtung", Net: 1_000,
		TaxRate: domain.TaxRateNone, NotTaxableReason: "Trinkgeld"})
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Bewirtung mit Trinkgeld: %v", err)
	}
	if got := debitOn(entry, domain.AccountVorsteuer7); got != 700 {
		t.Errorf("Vorsteuer %s €, erwartet 7,00 € nur auf die Speisen", got)
	}
	if debitOn(entry, "6640") != 7_700 || debitOn(entry, "6644") != 3_300 {
		t.Errorf("Bewirtung 6640 %s, 6644 %s; erwartet 77,00 und 33,00", debitOn(entry, "6640"), debitOn(entry, "6644"))
	}

	bare := env.receipt(t, vendor.ID, "reisekosten", 1_000, domain.TaxRateNone, domain.TaxTreatmentDomestic)
	if _, err := env.posting.PreviewIncomingReceipt(ctx, bare); err == nil {
		t.Error("Satz null ohne Grund auf einem Inlandsbeleg darf nicht durchgehen")
	}
	taxed := env.receipt(t, vendor.ID, "reisekosten", 1_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	taxed.Positions[0].NotTaxableReason = "Kurtaxe"
	if _, err := env.posting.PreviewIncomingReceipt(ctx, taxed); err == nil {
		t.Error("ein nicht steuerbarer Posten mit Steuersatz darf nicht durchgehen")
	}
}
