package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Rechnungsart entscheidet über Buchung, Dokument und Verbund — und sie
// kommt über die Bridge aus der Oberfläche.
//
// Der gewöhnliche Weg „Rechnung ausstellen" erzwingt die Invarianten der
// anderen Arten nicht: eine Abschlagsrechnung, die hier entstünde, hätte keinen
// offenen Posten und keinen Verbund, und eine Schlussrechnung wiese die
// Anzahlungen im Dokument aus, während die volle Forderung gebucht würde
// (§ 14c Abs. 1 UStG). Deshalb wird die Art geprüft, bevor eine Nummer fällt.

// issueOtherKind stellt eine gewöhnliche Rechnung mit fremder Art zu.
func (e *testEnv) issueOtherKind(t *testing.T, customerID uint, kind domain.InvoiceKind) error {
	t.Helper()
	inv := e.simpleInvoice(customerID, "2026-03-01", 100000)
	inv.Kind = kind
	return e.invoicesWired(t).Issue(context.Background(), inv)
}

func TestIssueRejectsInvoiceKindsOfOtherWays(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	for _, kind := range []domain.InvoiceKind{
		domain.InvoiceKindAdvance,
		domain.InvoiceKindFinal,
		domain.InvoiceKindCorrection,
		domain.InvoiceKindCancellation,
	} {
		err := env.issueOtherKind(t, customer.ID, kind)
		if err == nil {
			t.Errorf("%q ließ sich über das gewöhnliche Ausstellen erzeugen", kind)
			continue
		}
		// Die Meldung nennt den Weg, über den die Art entsteht — sonst bleibt
		// nur ein Nein.
		if !strings.Contains(err.Error(), "lässt sich hier nicht ausstellen") {
			t.Errorf("die Meldung zu %q erklärt den Weg nicht: %v", kind, err)
		}
	}

	// Und keine dieser Zurückweisungen hat eine Nummer gekostet.
	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatalf("Nummernkreis lesen: %v", err)
	}
	if next != 1 {
		t.Errorf("der Zähler steht auf %d, erwartet 1 — eine zurückgewiesene Art hat eine Nummer verbraucht", next)
	}
	invoices, err := env.invoiceRepoOf(t).FindAll(ctx, 2026)
	if err != nil {
		t.Fatalf("Rechnungen lesen: %v", err)
	}
	if len(invoices) != 0 {
		t.Errorf("es steht %d Rechnung(en) in der Datenbank, erwartet keine", len(invoices))
	}
}

// Die Schlussrechnung ist der teuerste der Fälle: mit PrepaidAmount trüge das
// Dokument BT-113 und einen geminderten Zahlbetrag, gebucht würde die volle
// Forderung ohne Auflösung der Anzahlungskonten.
func TestIssueRejectsFinalInvoiceWithPrepaidAmount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	inv.Kind = domain.InvoiceKindFinal
	inv.PrepaidAmount = 50000
	inv.PrecedingRefs = []domain.InvoiceReference{{Number: "RE-2026-0001", Date: "2026-02-01"}}

	if err := env.invoicesWired(t).Issue(ctx, inv); err == nil {
		t.Fatal("eine Schlussrechnung mit abgesetzten Anzahlungen ging über das gewöhnliche Ausstellen durch")
	}
	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatalf("Nummernkreis lesen: %v", err)
	}
	if next != 1 {
		t.Errorf("der Zähler steht auf %d, erwartet 1", next)
	}
}

// Die Angaben der anderen Arten werden verworfen und nicht mitgespeichert: eine
// gewöhnliche Rechnung trägt keinen Verbund, keine abgesetzte Anzahlung und
// keinen Bezug auf eine Rechnung, die niemand storniert hat.
func TestIssueDropsFieldsOfOtherInvoiceKinds(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	inv.GroupID = &group.ID
	inv.PrepaidAmount = 50000
	inv.PrecedingRefs = []domain.InvoiceReference{{Number: "RE-2026-9999", Date: "2026-02-01"}}
	inv.CorrectsInvoiceNumber = "RE-2026-8888"
	inv.CorrectsInvoiceDate = "2026-02-01"

	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	stored, err := env.invoiceRepoOf(t).FindByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("Rechnung laden: %v", err)
	}
	if stored.GroupID != nil {
		t.Error("eine gewöhnliche Rechnung gehört zu keinem Rechnungsverbund")
	}
	if stored.PrepaidAmount != 0 {
		t.Errorf("abgesetzte Anzahlungen = %s, erwartet null", stored.PrepaidAmount)
	}
	if len(stored.PrecedingRefs) != 0 {
		t.Errorf("Bezüge auf vorausgegangene Rechnungen: %+v, erwartet keine", stored.PrecedingRefs)
	}
	if stored.CorrectsInvoiceNumber != "" || stored.CorrectsInvoiceDate != "" {
		t.Errorf("der Bezug auf eine berichtigte Rechnung bleibt dem Berichtigen vorbehalten: %q vom %q",
			stored.CorrectsInvoiceNumber, stored.CorrectsInvoiceDate)
	}
}

// Die berichtigte Schlussrechnung geht weiterhin durch: sie trägt die Art
// „Rechnungskorrektur" und den Weg der Schlussrechnung, weil sie dieselben
// Anzahlungen absetzen muss wie die stornierte.
func TestCorrectedFinalInvoiceStillDeductsTheAdvances(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	advance, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advance.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}

	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-05-01",
		ServiceDateFrom: "2026-05-01", ServiceDateTo: "2026-05-01",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, Unit: "C62",
			UnitPrice: 1000000, TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}

	replacement := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-06-01",
		ServiceDateFrom: "2026-05-01", ServiceDateTo: "2026-05-01",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, Unit: "C62",
			UnitPrice: 900000, TaxRate: domain.TaxRateStandard,
		}},
	}
	corrected, err := svc.CorrectInvoice(ctx, final.ID, "Menge falsch abgerechnet", replacement)
	if err != nil {
		t.Fatalf("Schlussrechnung berichtigen: %v", err)
	}
	if corrected.Kind != domain.InvoiceKindCorrection {
		t.Errorf("Dokumentart = %q, erwartet %q (Typcode 384)", corrected.Kind, domain.InvoiceKindCorrection)
	}
	if corrected.PrepaidAmount != advance.GrossAmount {
		t.Errorf("abgesetzte Anzahlungen = %s, erwartet %s", corrected.PrepaidAmount, advance.GrossAmount)
	}
	if len(corrected.PrecedingRefs) != 1 || corrected.PrecedingRefs[0].Number != advance.InvoiceNumber {
		t.Errorf("der Bezug auf die Abschlagsrechnung fehlt: %+v", corrected.PrecedingRefs)
	}
}
