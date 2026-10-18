package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/bankeffects"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// paymentsWithBank ist der Zahlungsdienst mit dem Bankbezug des Journals, wie
// ihn der Kern verdrahtet: jeder Umsatz steht in genau einer gültigen Buchung.
func (e *testEnv) paymentsWithBank(t *testing.T) *PaymentService {
	t.Helper()
	e.journal.AddEntryEffects(bankeffects.EntryEffect(repository.NewBankRepository(e.db), e.journalRepo))
	return e.payments(t)
}

func (e *testEnv) assertMatch(t *testing.T, id uint, want domain.MatchStatus) {
	t.Helper()
	if got := e.bankTx(t, id).MatchStatus; got != want {
		t.Errorf("Bankumsatz %d steht auf %q, erwartet %q", id, got, want)
	}
}

// Zinsgutschrift einer inländischen Bank: 100,00 € Zinsen, 25 %
// Kapitalertragsteuer 25,00 €, Solidaritätszuschlag 5,5 % = 1,375 €, also
// 1,37 € (§ 4 Satz 3 SolZG). Gutschrift 73,63 €.
func TestSplitInterestCreditWithWithholdingTax(t *testing.T) {
	env := newTestEnv(t)
	payments := env.paymentsWithBank(t)
	id := env.bankLine(t, "2026-09-30", 7363, "Hausbank", "Zinsen Q3 2026")

	entry, err := payments.BookBankSplit(context.Background(), BankSplitRequest{
		BankTxID: id,
		Lines: []BankSplitLine{
			{Account: domain.AccountSonstigeZinsertraege, Amount: 10000, Text: "Zinsen Q3 2026"},
			{Account: "7630", Amount: -2500, Text: "Kapitalertragsteuer"},
			{Account: "7633", Amount: -137, Text: "Solidaritätszuschlag"},
		},
	})
	if err != nil {
		t.Fatalf("Aufteilung: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, domain.AccountBank, 7363},
		{domain.SideCredit, domain.AccountSonstigeZinsertraege, 10000},
		{domain.SideDebit, "7630", 2500},
		{domain.SideDebit, "7633", 137},
	})
	if entry.BankTxID == nil || *entry.BankTxID != id {
		t.Errorf("die Buchung verweist nicht auf den Bankumsatz")
	}
	env.assertMatch(t, id, domain.MatchStatusMatched)
}

// Darlehensrate der Bank: Tilgung und Zinsen in einer Abbuchung.
func TestSplitLoanInstallment(t *testing.T) {
	env := newTestEnv(t)
	id := env.bankLine(t, "2026-05-30", -125000, "Hausbank", "Rate Darlehen 4711")
	entry, err := env.paymentsWithBank(t).BookBankSplit(context.Background(), BankSplitRequest{
		BankTxID: id,
		Lines: []BankSplitLine{
			{Account: "3150", Amount: 100000, Text: "Tilgung"},
			{Account: "7320", Amount: 25000, Text: "Zinsen"},
		},
	})
	if err != nil {
		t.Fatalf("Aufteilung: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideCredit, domain.AccountBank, 125000},
		{domain.SideDebit, "3150", 100000},
		{domain.SideDebit, "7320", 25000},
	})
}

// Steuerlastschrift mit mehreren Steuerarten.
func TestSplitTaxDebit(t *testing.T) {
	env := newTestEnv(t)
	id := env.bankLine(t, "2026-09-10", -430000, "Finanzamt", "USt 08/2026, KSt/SolZ III/2026")
	entry, err := env.paymentsWithBank(t).BookBankSplit(context.Background(), BankSplitRequest{
		BankTxID: id,
		Lines: []BankSplitLine{
			{Account: domain.AccountUmsatzsteuerVorauszahlungen, Amount: 250000, Text: "USt 08/2026"},
			{Account: domain.AccountKoerperschaftsteuer, Amount: 170616, Text: "KSt III/2026"},
			{Account: domain.AccountSolidaritaetszuschlag, Amount: 9384, Text: "SolZ III/2026"},
		},
	})
	if err != nil {
		t.Fatalf("Aufteilung: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideCredit, domain.AccountBank, 430000},
		{domain.SideDebit, domain.AccountUmsatzsteuerVorauszahlungen, 250000},
		{domain.SideDebit, domain.AccountKoerperschaftsteuer, 170616},
		{domain.SideDebit, domain.AccountSolidaritaetszuschlag, 9384},
	})
}

// Sammelüberweisung an zwei Lieferanten samt Überweisungsentgelt.
func TestSplitCollectivePaymentWithFee(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	a := env.vendor(t, "Lieferant A", "DE", "")
	b := env.vendor(t, "Lieferant B", "DE", "")
	first := env.openPayable(t, a.ID, 100000, domain.TaxRateStandard) // 1.190,00
	second := env.openPayable(t, b.ID, 50000, domain.TaxRateStandard) // 595,00
	id := env.bankLine(t, "2026-04-15", -178750, "Sammelauftrag", "Sammelüberweisung")

	entry, err := payments.BookBankSplit(ctx, BankSplitRequest{
		BankTxID: id,
		Lines: []BankSplitLine{
			{OpenItemEntryID: first.ID, SettledAmount: 119000},
			{OpenItemEntryID: second.ID, SettledAmount: 59500},
			{Account: domain.AccountNebenkostenGeld, Amount: 250, Text: "Entgelt Sammelauftrag"},
		},
	})
	if err != nil {
		t.Fatalf("Aufteilung: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, a.LedgerAccount, 119000},
		{domain.SideDebit, b.LedgerAccount, 59500},
		{domain.SideDebit, domain.AccountNebenkostenGeld, 250},
		{domain.SideCredit, domain.AccountBank, 178750},
	})
	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("nach der Sammelüberweisung sind noch %d Posten offen", len(items))
	}
	allocations, err := payments.Allocations(ctx, entry.ID)
	if err != nil || len(allocations) != 2 {
		t.Errorf("Zuordnungen: %d, %v", len(allocations), err)
	}
}

// Die Zeilen müssen den Umsatz treffen; sonst wird nichts gebucht.
func TestSplitMustMatchTheTransaction(t *testing.T) {
	env := newTestEnv(t)
	id := env.bankLine(t, "2026-09-30", 7363, "Hausbank", "Zinsen")
	_, err := env.paymentsWithBank(t).BookBankSplit(context.Background(), BankSplitRequest{
		BankTxID: id,
		Lines: []BankSplitLine{
			{Account: domain.AccountSonstigeZinsertraege, Amount: 10000},
			{Account: "7630", Amount: -2500},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Differenz -1,37") {
		t.Fatalf("erwartet Abweisung mit Differenz, erhalten %v", err)
	}
	env.assertMatch(t, id, domain.MatchStatusUnmatched)
	if n := env.entryCount(t); n != 0 {
		t.Errorf("nach der Abweisung stehen %d Buchungen im Journal", n)
	}
}

// Umsatzsteuer entsteht über Beleg oder Rechnung, nicht über den Kontoauszug;
// ein offener Posten gehört über seine Zeile ausgeglichen.
func TestSplitRefusesAccountsOfOtherPaths(t *testing.T) {
	env := newTestEnv(t)
	payments := env.paymentsWithBank(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	payable := env.openPayable(t, vendor.ID, 10000, domain.TaxRateStandard)
	id := env.bankLine(t, "2026-04-01", 11900, "Kunde", "Zahlung")
	for name, line := range map[string]BankSplitLine{
		"Erlöskonto mit Umsatzsteuer":      {Account: "4400", Amount: 11900},
		"Personenkonto":                    {Account: vendor.LedgerAccount, Amount: 11900},
		"Verbindlichkeit im Eingang":       {OpenItemEntryID: payable.ID, SettledAmount: 11900},
		"Gesellschafterkonto ohne Kontakt": {Account: domain.AccountVerbindlichkeitenGesellschafter, Amount: 11900},
	} {
		if _, err := payments.BookBankSplit(context.Background(), BankSplitRequest{
			BankTxID: id, Lines: []BankSplitLine{line},
		}); err == nil {
			t.Errorf("%s: die Aufteilung wurde angenommen", name)
		}
	}
	env.assertMatch(t, id, domain.MatchStatusUnmatched)
}

// Ein Umsatz wird genau einmal gebucht, gleich über welchen Weg.
func TestBankTransactionIsBookedOnce(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	bank := env.banking(t)
	id := env.bankLine(t, "2026-03-31", -1500, "Hausbank", "Kontoführung")

	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id,
		Lines: []BankSplitLine{{Account: domain.AccountNebenkostenGeld, Amount: 1500}}}); err != nil {
		t.Fatalf("Aufteilung: %v", err)
	}
	if _, err := bank.BookDirect(ctx, id, domain.AccountNebenkostenGeld, ""); err == nil {
		t.Error("ein zugeordneter Umsatz wurde ein zweites Mal gebucht")
	}

	// Auch die Handbuchung mit Bankbezug kommt nicht ein zweites Mal durch.
	receipt := env.fileIncoming(t, "kontoauszug.pdf")
	_, err := env.posting.PostManualEntry(ctx, ManualEntryRequest{
		ReceiptID: receipt.ID,
		Entry: domain.JournalEntry{
			BookingDate: "2026-03-31", Description: "Kontoführung", TaxTreatment: domain.TaxTreatmentExempt,
			BankTxID: &id,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: domain.AccountNebenkostenGeld, Amount: 1500},
				{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 1500},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "bereits mit Buchung") {
		t.Errorf("erwartet Abweisung der zweiten Buchung, erhalten %v", err)
	}
}

// Die Handbuchung mit Bankbezug setzt den Umsatz auf zugeordnet; heute stand
// er danach weiter offen und lud zur zweiten Buchung ein.
func TestManualEntryWithBankReferenceMatchesTheTransaction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.paymentsWithBank(t)
	id := env.bankLine(t, "2026-03-31", -1500, "Hausbank", "Kontoführung")
	receipt := env.fileIncoming(t, "kontoauszug.pdf")
	manual := func(amount domain.Cents) (*domain.JournalEntry, error) {
		return env.posting.PostManualEntry(ctx, ManualEntryRequest{
			ReceiptID: receipt.ID,
			Entry: domain.JournalEntry{
				BookingDate: "2026-03-31", Description: "Kontoführung", TaxTreatment: domain.TaxTreatmentExempt,
				BankTxID: &id,
				Lines: []domain.JournalLine{
					{Side: domain.SideDebit, Account: domain.AccountNebenkostenGeld, Amount: amount},
					{Side: domain.SideCredit, Account: domain.AccountBank, Amount: amount},
				},
			},
		})
	}
	if _, err := manual(1400); err == nil || !strings.Contains(err.Error(), "lautet über -15,00") {
		t.Errorf("ein abweichender Betrag muss abgewiesen werden, erhalten %v", err)
	}
	env.assertMatch(t, id, domain.MatchStatusUnmatched)
	if _, err := manual(1500); err != nil {
		t.Fatalf("Handbuchung: %v", err)
	}
	env.assertMatch(t, id, domain.MatchStatusMatched)
}

// Die Generalumkehr gibt den Umsatz wieder frei; „stornieren und neu buchen"
// lässt ihn bei der Neubuchung.
func TestReversalReleasesAndCorrectionKeepsTheTransaction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.paymentsWithBank(t)
	bank := env.banking(t)
	id := env.bankLine(t, "2026-03-31", -1500, "Hausbank", "Kontoführung")

	booked, err := bank.BookDirect(ctx, id, domain.AccountSonstigerAufwand, "Kontoführung")
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	result, err := env.journal.CorrectEntry(ctx, booked.ID, "falsches Konto", &domain.JournalEntry{
		BookingDate: "2026-03-31", Description: "Kontoführung", Source: domain.EntrySourcePayment, BankTxID: &id,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountNebenkostenGeld, Amount: 1500},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 1500},
		},
	})
	if err != nil {
		t.Fatalf("Korrektur: %v", err)
	}
	env.assertMatch(t, id, domain.MatchStatusMatched)

	if _, err := env.journal.Reverse(ctx, result.Replacement.ID, "Umsatz neu zuordnen"); err != nil {
		t.Fatalf("Storno: %v", err)
	}
	env.assertMatch(t, id, domain.MatchStatusUnmatched)
	if _, err := bank.BookDirect(ctx, id, domain.AccountNebenkostenGeld, "Kontoführung"); err != nil {
		t.Errorf("nach dem Storno muss der Umsatz wieder buchbar sein: %v", err)
	}
}

// Ein als nicht zu buchen markierter Umsatz nimmt keine Buchung an.
func TestIgnoredTransactionTakesNoBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	id := env.bankLine(t, "2026-03-31", -1500, "Hausbank", "Kontoführung")
	if err := env.banking(t).Ignore(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id,
		Lines: []BankSplitLine{{Account: domain.AccountNebenkostenGeld, Amount: 1500}}}); err == nil {
		t.Error("ein ignorierter Umsatz wurde gebucht")
	}
}
