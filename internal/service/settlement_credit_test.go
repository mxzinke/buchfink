package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// creditNote bucht die Gutschrift eines Lieferanten von Hand: Soll auf dem
// Kreditor, Aufwand und Vorsteuer im Haben.
func (e *testEnv) creditNote(t *testing.T, vendor *domain.Contact, number string, net domain.Cents) *domain.JournalEntry {
	t.Helper()
	tax := domain.TaxRateStandard.Tax(net)
	entry, err := e.journal.Post(context.Background(), &domain.JournalEntry{
		BookingDate: "2026-09-21", DocumentDate: "2026-09-21", ServiceDateFrom: "2026-09-21", ServiceDateTo: "2026-09-21",
		Description: "Gutschrift " + number, Source: domain.EntrySourceManual, DocumentNumber: number,
		TaxTreatment: domain.TaxTreatmentDomestic, ContactID: &vendor.ID,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: vendor.LedgerAccount, ContactID: &vendor.ID, Amount: net + tax},
			{Side: domain.SideCredit, Account: "6495", Amount: net},
			{Side: domain.SideCredit, Account: domain.AccountVorsteuer19, Amount: tax, TaxKey: "VST19", TaxBase: net},
		},
	})
	if err != nil {
		t.Fatalf("Gutschrift: %v", err)
	}
	return entry
}

// Die Gutschrift eines Lieferanten steht als Guthaben in den offenen Posten:
// die Erstattung gleicht sie als Zahlungseingang aus, und sie wird nie
// überfällig.
func TestSupplierCreditIsRefundedByAnIncomingPayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	vendor := env.vendor(t, "Isarhost GmbH", "DE", "")
	credit := env.creditNote(t, vendor, "IH-GS-2026-0042", 2_000)

	items, err := payments.OpenItems(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("offene Posten: %v, %+v", err, items)
	}
	if !items[0].IsCredit() || items[0].SignedOpen() != -2_380 || items[0].IsOverdue("2027-01-01") {
		t.Errorf("Gutschrift als Posten: %+v", items[0])
	}

	id := env.bankLine(t, "2026-09-25", 2_380, "Isarhost GmbH", "Erstattung IH-GS-2026-0042")
	entry, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id,
		Lines: []BankSplitLine{{OpenItemEntryID: credit.ID, SettledAmount: 2_380}}})
	if err != nil {
		t.Fatalf("Erstattung: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideCredit, vendor.LedgerAccount, 2_380},
		{domain.SideDebit, domain.AccountBank, 2_380},
	})
	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("nach der Erstattung bleibt offen: %+v", items)
	}
}

// Der Lieferant zieht die Gutschrift von der nächsten Rechnung ab: eine
// Überweisung gleicht Rechnung und Gutschrift zusammen aus.
func TestCreditIsNettedAgainstAnInvoiceInOnePayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	vendor := env.vendor(t, "Isarhost GmbH", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 8_900, domain.TaxRateStandard)
	credit := env.creditNote(t, vendor, "IH-GS-2026-0042", 2_000)

	id := env.bankLine(t, "2026-10-05", -(10_591 - 2_380), "Isarhost GmbH", "IH-2026-09-1550 abzgl. IH-GS-2026-0042")
	entry, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{
		{OpenItemEntryID: invoice.ID, SettledAmount: 10_591},
		{OpenItemEntryID: credit.ID, SettledAmount: 2_380},
	}})
	if err != nil {
		t.Fatalf("Zahlung mit Verrechnung: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 10_591},
		{domain.SideCredit, vendor.LedgerAccount, 2_380},
		{domain.SideCredit, domain.AccountBank, 8_211},
	})
	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("nach der Zahlung bleibt offen: %+v", items)
	}

	// Eine Gutschrift allein gleicht keinen Zahlungsausgang aus.
	other := env.creditNote(t, vendor, "IH-GS-2026-0043", 1_000)
	out := env.bankLine(t, "2026-10-06", -1_190, "Isarhost GmbH", "falsch")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: out,
		Lines: []BankSplitLine{{OpenItemEntryID: other.ID, SettledAmount: 1_190}}}); err == nil {
		t.Error("eine Gutschrift darf keinen Zahlungsausgang ausgleichen")
	}
}
