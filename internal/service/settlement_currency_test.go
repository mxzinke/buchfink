package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// usdPayable bucht eine offene Rechnung über 84 USD, zum Kurs der Rechnung
// mit 73,65 €.
func (e *testEnv) usdPayable(t *testing.T, vendor *domain.Contact) *domain.JournalEntry {
	t.Helper()
	entry, err := e.journal.Post(context.Background(), &domain.JournalEntry{
		BookingDate: "2026-07-15", DocumentDate: "2026-07-15", ServiceDateFrom: "2026-07-15", ServiceDateTo: "2026-07-15",
		Description: "Codeforge CF-2026-07-88231", Source: domain.EntrySourceManual, DocumentNumber: "CF-2026-07-88231",
		TaxTreatment: domain.TaxTreatmentNotTaxable, ContactID: &vendor.ID, Currency: "USD",
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6495", Amount: 7_365, ForeignAmount: 8_400},
			{Side: domain.SideCredit, Account: vendor.LedgerAccount, ContactID: &vendor.ID, Amount: 7_365, ForeignAmount: 8_400},
		},
	})
	if err != nil {
		t.Fatalf("USD-Rechnung: %v", err)
	}
	return entry
}

// Zahlt die GmbH einen Fremdwährungsposten vollständig, ist der Unterschied
// zum gebuchten Euro-Betrag Kursdifferenz; Buchfink erkennt das an der
// Währung, ohne dass sie jemand nennt. Gewinn und Verlust gehen auf die
// Konten der Währungsumrechnung.
func TestForeignItemSettlesWithItsCurrencyDifference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	vendor := env.vendor(t, "Codeforge Inc.", "US", "")

	for _, c := range []struct {
		name string
		paid domain.Cents
		want []bookedLine
	}{
		{"Kursgewinn", 7_223, []bookedLine{
			{domain.SideDebit, vendor.LedgerAccount, 7_365},
			{domain.SideCredit, "4840", 142},
			{domain.SideCredit, domain.AccountBank, 7_223},
		}},
		{"Kursverlust", 7_500, []bookedLine{
			{domain.SideDebit, vendor.LedgerAccount, 7_365},
			{domain.SideDebit, "6880", 135},
			{domain.SideCredit, domain.AccountBank, 7_500},
		}},
	} {
		item := env.usdPayable(t, vendor)
		id := env.bankLine(t, "2026-08-05", -c.paid, "Codeforge Inc.", "CF-2026-07-88231")
		entry, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id,
			Lines: []BankSplitLine{{OpenItemEntryID: item.ID, SettledAmount: 7_365}}})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		assertLines(t, entry, c.want)
	}

	// Ein Drittel weniger ist kein Kurs mehr, sondern eine Teilzahlung.
	item := env.usdPayable(t, vendor)
	id := env.bankLine(t, "2026-08-06", -5_000, "Codeforge Inc.", "CF-2026-07-88231")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id,
		Lines: []BankSplitLine{{OpenItemEntryID: item.ID, SettledAmount: 7_365}}}); err == nil {
		t.Error("eine Abweichung über der Toleranz darf nicht als Kursdifferenz gelten")
	}

	// Ein Posten in Euro bekommt keine Kursdifferenz.
	euro := env.openPayable(t, vendor.ID, 6_000, domain.TaxRateStandard)
	other := env.bankLine(t, "2026-08-07", -7_100, "Codeforge Inc.", "Rechnung")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: other,
		Lines: []BankSplitLine{{OpenItemEntryID: euro.ID, SettledAmount: 7_140}}}); err == nil {
		t.Error("ein Euro-Posten darf keine angenommene Kursdifferenz bekommen")
	}
}

// Der Bankabgleich schlägt einen Fremdwährungsposten auch ohne genauen Betrag
// vor, wenn der Umsatz im Rahmen des Kurses liegt.
func TestForeignItemIsSuggestedWithinTheRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Codeforge Inc.", "US", "")
	item := env.usdPayable(t, vendor)
	id := env.bankLine(t, "2026-08-05", -7_223, "Codeforge Inc.", "CF-2026-07-88231")

	suggestions, err := env.banks(t).Suggest(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions.Suggestions) == 0 || len(suggestions.Suggestions[0].Items) != 1 ||
		suggestions.Suggestions[0].Items[0].EntryID != item.ID {
		t.Fatalf("erwartet die USD-Rechnung als ersten Vorschlag, erhalten %+v", suggestions.Suggestions)
	}
}
