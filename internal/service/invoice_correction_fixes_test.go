package service

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Storno und Berichtigung: der Zeitraum der Steuerkorrektur, das Dokument und
// der Vermerk über den Versand.

// Die Steuerkorrektur des Stornos fällt in den Zeitraum der Berichtigung.
//
// § 17 Abs. 1 Satz 8 UStG und Abschn. 14c.1 Abs. 5 UStAE führen beide auf den
// Tag, an dem die Berichtigung gegenüber dem Empfänger erklärt wird — also auf
// den Tag des Stornodokuments. Landete sie im Ursprungszeitraum, verlangte jedes
// Storno eine berichtigte Voranmeldung für einen längst übermittelten Monat.
func TestCancellationCorrectsTheVatInThePeriodOfCorrection(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if _, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht"); err != nil {
		t.Fatalf("stornieren: %v", err)
	}

	reversal, err := env.journalRepo.FindReversalOf(ctx, *original.JournalEntryID)
	if err != nil || reversal == nil {
		t.Fatalf("die Generalumkehr fehlt: %v", err)
	}
	today := time.Now().Format("2006-01-02")
	if reversal.BookingDate != today {
		t.Fatalf("die Generalumkehr trägt das Buchungsdatum %q, erwartet den heutigen Tag %q",
			reversal.BookingDate, today)
	}
	for _, line := range reversal.Lines {
		got := accounting.VatPeriodFor(reversal, line, "")
		if got != today {
			t.Errorf("die Zeile auf %s wird dem Zeitraum %q zugeordnet, erwartet den Korrekturtag %q",
				line.Account, got, today)
		}
	}
	// Gegenprobe: die Ursprungsbuchung bleibt in ihrem Leistungszeitraum.
	entry, err := env.journalRepo.FindByID(ctx, *original.JournalEntryID)
	if err != nil {
		t.Fatalf("Ursprungsbuchung laden: %v", err)
	}
	if got := accounting.VatPeriodFor(entry, entry.Lines[0], ""); got != "2026-03-01" {
		t.Errorf("die Ursprungsbuchung gehört in den März 2026, ergibt aber %q", got)
	}
}

// Auf einem Dokument mit negativen Beträgen steht keine Zahlungsbedingung.
//
// „Zahlbar innerhalb von 14 Tagen, bei Zahlung bis … 2 % Skonto" forderte eine
// Zahlung, die niemand leisten soll — und stünde so auch in BT-20.
func TestStornoCarriesNoPaymentTerms(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	original.Terms = domain.PaymentTerms{DueDays: 14, DiscountPermille: 20, DiscountDays: 7}
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	storno, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}
	if storno.Terms.Stated() {
		t.Errorf("die Stornorechnung trägt Zahlungsbedingungen: %q", storno.Terms.Note(storno.Date))
	}
	// Gegenprobe: die Ursprungsrechnung behält ihre Bedingungen.
	if !original.Terms.Stated() {
		t.Error("die Ursprungsrechnung darf ihre Zahlungsbedingungen nicht verlieren")
	}
}

// Der Versandvermerk ist der Nachweis des Zugangs; ein Datum, das keines ist,
// wäre keiner.
func TestMarkSentRejectsSomethingThatIsNoDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if _, err := svc.MarkSent(ctx, inv.ID, "gestern", domain.InvoiceSentViaEmail, ""); err == nil {
		t.Error("„gestern\" wurde als Versanddatum übernommen")
	}
	if _, err := svc.MarkSent(ctx, inv.ID, "01.03.2026", domain.InvoiceSentViaEmail, ""); err == nil {
		t.Error("ein Datum in deutscher Schreibweise gehört zurückgewiesen, nicht gespeichert")
	}

	sent, err := svc.MarkSent(ctx, inv.ID, "2026-03-02", domain.InvoiceSentViaEmail, "an die Buchhaltung")
	if err != nil {
		t.Fatalf("Versand vermerken: %v", err)
	}
	if sent.SentAt != "2026-03-02" {
		t.Errorf("Versanddatum = %q, erwartet 2026-03-02", sent.SentAt)
	}
}

// Ein offener Rechnungsverbund überlebt den Jahreswechsel.
//
// Die Abschläge fallen ins eine Jahr, die Schlussrechnung ins nächste. Wäre
// allein das Geschäftsjahr maßgeblich, verschwände der Verbund am 1. Januar aus
// der Ansicht — und mit ihm der einzige Weg, die vereinnahmten Anzahlungen
// abzusetzen (§ 14 Abs. 5 Satz 2 UStG).
func TestOpenInvoiceGroupStaysVisibleAfterTheYearChange(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	next, err := svc.GetInvoiceGroups(ctx, group.FiscalYear+1)
	if err != nil {
		t.Fatalf("Verbünde des Folgejahres: %v", err)
	}
	if len(next) != 1 || next[0].ID != group.ID {
		t.Fatalf("der offene Verbund fehlt im Folgejahr: %+v", next)
	}

	// Der abgeschlossene Verbund dagegen bleibt in seinem Jahr: er ist kein
	// offener Vorgang mehr, und die Ansicht des Folgejahres soll ihn nicht
	// mitschleppen.
	group.Closed = true
	if err := env.invoiceGroupRepoOf(t).Save(ctx, group); err != nil {
		t.Fatalf("Verbund abschließen: %v", err)
	}
	after, err := svc.GetInvoiceGroups(ctx, group.FiscalYear+1)
	if err != nil {
		t.Fatalf("Verbünde des Folgejahres: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("der abgeschlossene Verbund erscheint im Folgejahr: %+v", after)
	}
	same, err := svc.GetInvoiceGroups(ctx, group.FiscalYear)
	if err != nil {
		t.Fatalf("Verbünde des eigenen Jahres: %v", err)
	}
	if len(same) != 1 {
		t.Errorf("der Verbund fehlt in seinem eigenen Jahr: %+v", same)
	}
}
