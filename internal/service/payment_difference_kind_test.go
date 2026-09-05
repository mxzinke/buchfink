package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Ausbuchung steht in derselben Auswahl wie Skonto und Bankgebühr, geht
// aber einen anderen Weg.
//
// Sie ist keine Zahlung: es fließt kein Geld, und gebucht wird der
// Forderungsverlust samt Steuerkorrektur (§ 17 Abs. 2 Nr. 1 UStG). Wählt der
// Anwender sie im Zahlungsausgleich, muss die Auskunft ihn dorthin führen und
// nicht behaupten, die Art sei unbekannt.
func TestSettleSendsAWriteOffToItsOwnWay(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWired(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	_, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-04-01",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  *inv.JournalEntryID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceWriteoff,
			DifferenceAmount: 119000,
		}},
	})
	if err == nil {
		t.Fatal("die Ausbuchung darf nicht über den Zahlungsausgleich laufen")
	}
	if strings.Contains(err.Error(), "unbekannte Differenzart") {
		t.Errorf("die Ausbuchung ist bekannt, nur nicht hier buchbar; erhalten: %v", err)
	}
	if !strings.Contains(err.Error(), "Forderung ausbuchen") {
		t.Errorf("die Meldung muss den richtigen Weg nennen, erhalten: %v", err)
	}

	// Und dort geht sie durch.
	entry, err := payments.WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: *inv.JournalEntryID, Date: "2026-04-01",
		Reason: "Insolvenz des Kunden, Quote null",
	})
	if err != nil {
		t.Fatalf("Forderung ausbuchen: %v", err)
	}
	if entry == nil {
		t.Fatal("die Ausbuchung muss eine Buchung erzeugen")
	}
}

// Die Oberfläche muss die Weiche stellen können, bevor sie bucht.
//
// „Ausbuchung" steht in derselben Liste wie die Differenzarten einer Zahlung;
// ohne ein Kennzeichen am Eintrag müsste das Frontend den Sonderfall an seinem
// Namen erkennen.
func TestDifferenceKindsMarkTheWriteOffAsPaymentless(t *testing.T) {
	byKind := make(map[domain.DifferenceKind]domain.DifferenceKindInfo)
	for _, info := range domain.DifferenceKinds() {
		byKind[info.Kind] = info
	}
	if got, ok := byKind[domain.DifferenceWriteoff]; !ok || !got.WithoutPayment {
		t.Errorf("die Ausbuchung muss als zahlungslos gekennzeichnet sein: %+v", got)
	}
	for _, kind := range []domain.DifferenceKind{
		domain.DifferenceNone, domain.DifferenceSkonto, domain.DifferenceBankFee,
		domain.DifferenceRounding, domain.DifferenceCurrency,
	} {
		if byKind[kind].WithoutPayment {
			t.Errorf("die Differenzart %q begleitet eine Zahlung und darf nicht als zahlungslos gelten", kind)
		}
	}
}
