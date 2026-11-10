package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein erhaltenes Skonto mindert den Aufwand, auf dem die Rechnung steht, und
// erscheint nicht als Minderung des Materialaufwands (5736). Bei einer
// Bewirtung trifft es beide Teile im Verhältnis 70/30.
func TestReceivedSkontoReducesTheExpense(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Druckerei", "DE", "")

	pay := func(entry *domain.JournalEntry, gross, skonto domain.Cents) *domain.JournalEntry {
		t.Helper()
		paid, err := payments.Settle(ctx, PaymentRequest{
			PaymentAccount: domain.AccountBank, PaymentDate: "2026-03-20",
			Allocations: []AllocationRequest{{OpenItemEntryID: entry.ID, SettledAmount: gross,
				DifferenceKind: domain.DifferenceSkonto, DifferenceAmount: skonto}},
		})
		if err != nil {
			t.Fatalf("Zahlung mit Skonto: %v", err)
		}
		return paid
	}

	req := env.receipt(t, vendor.ID, "werbung", 100_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Settlement = SettlementOpen
	flyer, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Rechnung: %v", err)
	}
	assertLines(t, pay(flyer, 119_000, 2_380), []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 119_000},
		{domain.SideCredit, "6600", 2_000},
		{domain.SideCredit, domain.AccountVorsteuer19, 380},
		{domain.SideCredit, domain.AccountBank, 116_620},
	})

	meal := env.receipt(t, vendor.ID, "bewirtung", 10_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	meal.Settlement = SettlementOpen
	meal.Entertainment = &domain.EntertainmentDetail{Place: "Kutterhus, Hamburg", Day: "2026-03-10",
		Participants: "Henrik Sørensen, Line Madsen", Occasion: "Projektanbahnung"}
	dinner, err := env.posting.PostIncomingReceipt(ctx, meal)
	if err != nil {
		t.Fatalf("Bewirtung: %v", err)
	}
	assertLines(t, pay(dinner, 11_900, 238), []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 11_900},
		{domain.SideCredit, "6640", 140},
		{domain.SideCredit, "6644", 60},
		{domain.SideCredit, domain.AccountVorsteuer19, 38},
		{domain.SideCredit, domain.AccountBank, 11_662},
	})
}
