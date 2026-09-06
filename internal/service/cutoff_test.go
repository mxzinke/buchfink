package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Stichtagsauswertungen.
//
// Ein Prüfer fragt nach dem Stand an einem Tag, nicht nach dem Stand eines
// Geschäftsjahres. Eine Liste, die das ganze Jahr summiert und danach Zeilen
// ausblendet, zeigt Salden, die es zum Stichtag nie gab — deshalb liegt die
// Grenze in der Abfrage und nicht hinter ihr.

// bookOn legt eine Buchung an einem bestimmten Tag an.
func (e *testEnv) bookOn(t *testing.T, date, debit, credit string, amount domain.Cents) *domain.JournalEntry {
	t.Helper()
	entry := simpleEntry(debit, credit, amount)
	entry.BookingDate, entry.DocumentDate = date, date
	entry.ServiceDateFrom, entry.ServiceDateTo = date, date
	posted, err := e.journal.Post(context.Background(), entry)
	if err != nil {
		t.Fatalf("Buchung am %s: %v", date, err)
	}
	return posted
}

// accountBalance sucht ein Konto in der Summen- und Saldenliste.
func accountBalance(t *testing.T, susa *domain.SuSaOverview, number string) domain.Account {
	t.Helper()
	for _, class := range susa.Classes {
		for _, account := range class.Accounts {
			if account.Number == number {
				return account
			}
		}
	}
	t.Fatalf("das Konto %s steht nicht in der Summen- und Saldenliste", number)
	return domain.Account{}
}

// Die Summen- und Saldenliste zum 30.06. darf die Buchungen des zweiten
// Halbjahres nicht enthalten.
func TestSuSaAtCutoffExcludesLaterBookings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	env.bookOn(t, "2026-03-15", "6815", "1800", 10000)
	env.bookOn(t, "2026-09-15", "6815", "1800", 70000)

	half, err := env.accounting.GetSuSaOverviewAt(ctx, "2026-06-30")
	if err != nil {
		t.Fatalf("SuSa zum Stichtag: %v", err)
	}
	if half.Cutoff != "2026-06-30" {
		t.Errorf("die Liste nennt ihren Stichtag nicht: %q", half.Cutoff)
	}
	if got := accountBalance(t, half, "6815").DebitSum; got != 10000 {
		t.Errorf("Soll auf 6815 zum 30.06. = %s €, erwartet 100,00", got)
	}

	full, err := env.accounting.GetSuSaOverview(ctx)
	if err != nil {
		t.Fatalf("SuSa für das Jahr: %v", err)
	}
	if got := accountBalance(t, full, "6815").DebitSum; got != 80000 {
		t.Errorf("Soll auf 6815 im ganzen Jahr = %s €, erwartet 800,00", got)
	}
	if !half.IsBalanced || !full.IsBalanced {
		t.Error("beide Listen müssen ausgeglichen sein — sonst summiert der Stichtag halbe Buchungen")
	}
}

// Das Kontoblatt eines Zeitraums geht über den Jahreswechsel: „das Bankkonto von
// Oktober bis März" ist eine Frage, die ein Geschäftsjahr nicht beantwortet.
func TestAccountLedgerRangeCrossesTheYearBoundary(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	env.journal.SetFiscalYear(2025)
	env.bookOn(t, "2025-11-10", "6815", "1800", 5000)
	env.journal.SetFiscalYear(2026)
	env.bookOn(t, "2026-02-10", "6815", "1800", 3000)
	env.bookOn(t, "2026-08-10", "6815", "1800", 9000)

	ledger, err := env.accounting.GetAccountLedgerRange(ctx, "1800", "2025-10-01", "2026-03-31")
	if err != nil {
		t.Fatalf("Kontoblatt für den Zeitraum: %v", err)
	}
	if ledger.RowCount != 2 {
		t.Fatalf("erwartet 2 Zeilen im Fenster, erhalten %d", ledger.RowCount)
	}
	if ledger.From != "2025-10-01" || ledger.To != "2026-03-31" {
		t.Errorf("das Kontoblatt nennt seinen Zeitraum nicht: %q bis %q", ledger.From, ledger.To)
	}
	for _, row := range ledger.Rows {
		if row.BookingDate < "2025-10-01" || row.BookingDate > "2026-03-31" {
			t.Errorf("die Buchung vom %s liegt außerhalb des Fensters", row.BookingDate)
		}
	}

	// Und das Geschäftsjahr allein sieht nur seinen eigenen Teil.
	year, err := env.accounting.GetAccountLedger(ctx, "1800")
	if err != nil {
		t.Fatalf("Kontoblatt des Jahres: %v", err)
	}
	if year.RowCount != 2 {
		t.Errorf("das Kontoblatt 2026 hat %d Zeilen, erwartet 2", year.RowCount)
	}
}

// Die offene-Posten-Liste zum Stichtag ignoriert eine Zahlung, die erst danach
// gebucht wurde. Sonst wäre die Angabe über den Bilanzstichtag in Wahrheit eine
// über heute — und sie fiele umso kleiner aus, je später jemand hinsieht.
func TestOpenItemsAtCutoffIgnoreLaterPayments(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)

	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	// Die Zahlung fällt in den Juli, der Stichtag ist der 30.06.
	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: "1800",
		PaymentDate:    "2026-07-05",
		Description:    "Überweisung",
		Allocations: []AllocationRequest{{
			OpenItemEntryID: invoice.ID,
			SettledAmount:   119000,
			DifferenceKind:  domain.DifferenceNone,
		}},
	}); err != nil {
		t.Fatalf("Zahlung buchen: %v", err)
	}

	atCutoff, err := payments.OpenItemsAt(ctx, "2026-06-30")
	if err != nil {
		t.Fatalf("offene Posten zum Stichtag: %v", err)
	}
	if len(atCutoff) != 1 {
		t.Fatalf("zum 30.06. war die Rechnung offen — erwartet 1 Posten, erhalten %d", len(atCutoff))
	}
	if atCutoff[0].OpenAmount != 119000 {
		t.Errorf("offener Betrag zum Stichtag = %s €, erwartet 1.190,00", atCutoff[0].OpenAmount)
	}

	today, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten heute: %v", err)
	}
	if len(today) != 0 {
		t.Errorf("nach der Zahlung ist der Posten ausgeglichen — erwartet 0, erhalten %d", len(today))
	}
}

// Die Einzelpostenliste einer Sammelzahlung. Ohne sie steht im Journal eine
// Zeile über den Gesamtbetrag, und wogegen sie lief, ergibt sich nicht mehr.
func TestPaymentAllocationsListEverySettledItem(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)

	vendor := env.vendor(t, "Lieferant", "DE", "")
	first := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)
	second := env.openPayable(t, vendor.ID, 50000, domain.TaxRateStandard)

	payment, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: "1800",
		PaymentDate:    "2026-04-01",
		Description:    "Sammelüberweisung",
		Allocations: []AllocationRequest{
			{OpenItemEntryID: first.ID, SettledAmount: 119000, DifferenceKind: domain.DifferenceNone},
			{OpenItemEntryID: second.ID, SettledAmount: 59500, DifferenceKind: domain.DifferenceNone},
		},
	})
	if err != nil {
		t.Fatalf("Sammelzahlung: %v", err)
	}

	details, err := payments.Allocations(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Einzelposten: %v", err)
	}
	if len(details) != 2 {
		t.Fatalf("erwartet 2 Einzelposten, erhalten %d", len(details))
	}

	byEntry := map[uint]domain.PaymentAllocationDetail{}
	for _, d := range details {
		byEntry[d.OpenItemEntryID] = d
	}
	for _, want := range []struct {
		entry  *domain.JournalEntry
		amount domain.Cents
	}{{first, 119000}, {second, 59500}} {
		got, ok := byEntry[want.entry.ID]
		if !ok {
			t.Errorf("die Rechnung %s fehlt in der Einzelpostenliste", want.entry.EntryNumber)
			continue
		}
		if got.SettledAmount != want.amount {
			t.Errorf("%s: Ausgleichsbetrag %s €, erwartet %s €",
				want.entry.EntryNumber, got.SettledAmount, want.amount)
		}
		if got.OpenItemEntryNumber != want.entry.EntryNumber {
			t.Errorf("die Einzelposition nennt die Buchungsnummer %q statt %q",
				got.OpenItemEntryNumber, want.entry.EntryNumber)
		}
		if got.ContactName != vendor.Name {
			t.Errorf("die Einzelposition nennt den Partner %q statt %q", got.ContactName, vendor.Name)
		}
	}
}
