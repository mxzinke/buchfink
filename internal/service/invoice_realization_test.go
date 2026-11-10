package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Erlös und Forderung entstehen mit der Leistung (§ 252 Abs. 1 Nr. 4 HGB): Eine
// im Oktober geschriebene Rechnung über eine Septemberleistung bucht am Ende
// der Leistung. Ist der September schon festgeschrieben, bucht sie am
// Rechnungsdatum, und die Vorschau sagt das. Die Fälligkeit der Rechnung steht
// an der Buchung und bestimmt den offenen Posten.
func TestOutgoingInvoiceBooksOnServiceDay(t *testing.T) {
	env := newTestEnv(t)
	env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))
	ctx := context.Background()
	customer := env.customer(t, "Fischmarkt GmbH", "DE", "")
	svc := env.invoicesWired(t)

	september := func(date string) *domain.Invoice {
		inv := env.simpleInvoice(customer.ID, date, 544000)
		inv.ServiceDateFrom, inv.ServiceDateTo = "2026-09-01", "2026-09-25"
		inv.Terms.DueDays = 30
		return inv
	}

	preview, err := svc.Preview(ctx, september("2026-10-22"))
	if err != nil {
		t.Fatal(err)
	}
	if preview.BookingDate != "2026-09-25" || preview.DueDate != "2026-11-21" ||
		!hasWarning(preview.Warnings, warningBookedOnServiceDay) {
		t.Fatalf("Vorschau: Buchung %s, fällig %s, %+v", preview.BookingDate, preview.DueDate, preview.Warnings)
	}
	inv := september("2026-10-22")
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatal(err)
	}
	entry, err := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.BookingDate != "2026-09-25" || entry.DocumentDate != "2026-10-22" || entry.DueDate != "2026-11-21" {
		t.Fatalf("Buchung am %s, Beleg %s, fällig %s", entry.BookingDate, entry.DocumentDate, entry.DueDate)
	}

	env.commitUntil(t, "2026-09-30")
	late := september("2026-10-23")
	preview, err = svc.Preview(ctx, late)
	if err != nil {
		t.Fatal(err)
	}
	if preview.BookingDate != "2026-10-23" || !hasWarning(preview.Warnings, warningServiceDayLocked) {
		t.Fatalf("festgeschrieben: Buchung %s, %+v", preview.BookingDate, preview.Warnings)
	}
	if err := svc.Issue(ctx, late); err != nil {
		t.Fatal(err)
	}
	entry, err = env.journalRepo.FindByID(ctx, *late.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.BookingDate != "2026-10-23" {
		t.Fatalf("Buchung trotz Festschreibung am %s", entry.BookingDate)
	}
}

func hasWarning(ws []PostingWarning, code string) bool {
	for _, w := range ws {
		if w.Code == code {
			return true
		}
	}
	return false
}
