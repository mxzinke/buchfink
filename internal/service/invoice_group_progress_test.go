package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Fortschritt eines Verbunds kommt gefüllt aus dem Backend.
//
// Die Anzahlungsseite hat Abgerechnet, Vereinnahmt und Offen früher selbst
// summiert. Welche Abschläge mitzählen, ist aber eine fachliche Regel — ein
// stornierter fällt heraus (§ 14 Abs. 5 Satz 2 UStG), vereinnahmt ist erst, was
// bezahlt wurde — und zwei Rechenwege für dieselben Summen laufen auseinander,
// sobald einer von beiden eine Regel dazubekommt. Deshalb steht das Ergebnis im
// JSON, und dieser Test hält fest, dass es beim Lesen tatsächlich dort steht.
func TestInvoiceGroupsCarryTheirProgress(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	// Ein gestellter und ein gestellter, dann stornierter Abschlag.
	open, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	cancelled, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-04-01", Net: 200000,
	})
	if err != nil {
		t.Fatalf("zweite Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.CancelWithDocument(ctx, cancelled.ID, "Auftrag geändert"); err != nil {
		t.Fatalf("Abschlag stornieren: %v", err)
	}

	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("erwartet einen Verbund, erhalten %d", len(groups))
	}
	got := groups[0].Progress

	if got.AgreedNet != 1000000 {
		t.Errorf("vereinbart = %s, erwartet 10.000,00", got.AgreedNet)
	}
	// 4.000 aus dem offenen Abschlag; der stornierte zählt nicht mit.
	if got.BilledNet != 400000 {
		t.Errorf("abgerechnet = %s, erwartet 4.000,00", got.BilledNet)
	}
	if got.ReceivedGross != 0 {
		t.Errorf("vereinnahmt = %s, erwartet 0 — bezahlt wurde noch nichts", got.ReceivedGross)
	}
	if got.OpenNet != 600000 {
		t.Errorf("offen = %s, erwartet 6.000,00", got.OpenNet)
	}
	// Das mitgelieferte Feld und die Rechnung auf dem geladenen Verbund dürfen
	// nicht auseinanderfallen: sonst wäre genau die zweite Wahrheit entstanden,
	// die das Feld beseitigen soll.
	if want := groups[0].ComputeProgress(); got != want {
		t.Errorf("mitgeliefert %+v, gerechnet %+v", got, want)
	}

	// Nach der Vereinnahmung führt derselbe Weg den neuen Stand mit.
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: open.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Abschlag vereinnahmen: %v", err)
	}
	groups, err = svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if groups[0].Progress.ReceivedGross != 476000 {
		t.Errorf("vereinnahmt = %s, erwartet 4.760,00 brutto", groups[0].Progress.ReceivedGross)
	}
}
