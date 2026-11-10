package service

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestAdvanceBankPartialSplitAndFinal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	payments := env.paymentsWithBank(t)
	payments.SetInvoiceService(svc)
	customer := env.customer(t, "Abschlag GmbH", "DE", "")
	group := env.group(t, svc, customer.ID, 20000)
	inv, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{GroupID: group.ID, Date: "2026-03-01", Net: 10000})
	if err != nil {
		t.Fatal(err)
	}
	items, err := payments.OpenItems(ctx)
	if err != nil || len(items) != 1 || items[0].AdvanceInvoiceID != inv.ID {
		t.Fatalf("Posten: %+v, %v", items, err)
	}
	id := env.bankCredit(t, "2026-04-15", 5950, "advance-partial")
	request := PaymentRequest{BankTxID: &id, Allocations: []AllocationRequest{{AdvanceInvoiceID: inv.ID, SettledAmount: 5950}}}
	entry, err := payments.Settle(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, entry, []bookedLine{{domain.SideDebit, domain.AccountBank, 5950}, {domain.SideCredit, accounting.AccountErhalteneAnzahlungen19, 5000}, {domain.SideCredit, domain.AccountUmsatzsteuer19, 950}})
	if _, err := payments.Settle(ctx, request); err == nil {
		t.Fatal("derselbe Umsatz wurde zweimal gebucht")
	}
	for _, tc := range []struct {
		period string
		tax    domain.Cents
	}{{"2026-03", 0}, {"2026-04", 950}} {
		ret, err := env.vatReturns(t).Draft(ctx, tc.period)
		if err != nil {
			t.Fatal(err)
		}
		if ret.Tax("81") != tc.tax {
			t.Fatalf("Voranmeldung %s: %v", tc.period, ret.Tax("81"))
		}
	}

	items, err = payments.OpenItems(ctx)
	if err != nil || len(items) != 1 || items[0].OpenAmount != 5950 {
		t.Fatalf("Rest: %+v, %v", items, err)
	}
	id = env.bankCredit(t, "2026-05-15", 6150, "advance-split")
	split := BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{{AdvanceInvoiceID: inv.ID, SettledAmount: 5950}, {Account: domain.AccountSonstigeZinsertraege, Amount: 200}}}
	entry, err = payments.BookBankSplit(ctx, split)
	if err != nil {
		t.Fatal(err)
	}
	if got := lineAmount(t, entry, domain.AccountBank, domain.SideDebit); got != 6150 {
		t.Fatalf("Bank: %v", got)
	}
	if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideCredit); got != 950 {
		t.Fatalf("Steuer: %v", got)
	}
	if _, err := payments.BookBankSplit(ctx, split); err == nil {
		t.Fatal("Aufteilung doppelt")
	}
	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{GroupID: group.ID, Date: "2026-06-30", ServiceDateFrom: "2026-06-01", ServiceDateTo: "2026-06-30", Items: []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 20000, TaxRate: domain.TaxRateStandard}}})
	if err != nil {
		t.Fatal(err)
	}
	if final.PrepaidAmount != 11900 {
		t.Fatalf("Verrechnung: %v", final.PrepaidAmount)
	}
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != 0 {
		t.Fatalf("Anzahlungen: %v", got)
	}
}

func TestAdvanceRefundBankAndAtomicSplit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	payments := env.paymentsWithBank(t)
	payments.SetInvoiceService(svc)
	env.journal.AddEntryEffects(svc.AdvancePaymentEffect())
	customer := env.customer(t, "Rückzahlung GmbH", "DE", "")
	group := env.group(t, svc, customer.ID, 20000)
	inv, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{GroupID: group.ID, Date: "2026-03-01", Net: 10000})
	if err != nil {
		t.Fatal(err)
	}
	id := env.bankCredit(t, "2026-04-15", 11900, "advance-full")
	if _, err := payments.Settle(ctx, PaymentRequest{BankTxID: &id, Allocations: []AllocationRequest{{AdvanceInvoiceID: inv.ID, SettledAmount: 11900}}}); err != nil {
		t.Fatal(err)
	}
	refundID := env.bankCredit(t, "2026-05-15", -11900, "advance-refund")
	bad := BankSplitRequest{BankTxID: refundID, Lines: []BankSplitLine{{AdvanceInvoiceID: inv.ID, SettledAmount: 11900, Text: "Auftrag entfällt"}, {Account: domain.AccountSonstigeZinsertraege, Amount: 100}}}
	if _, err := payments.BookBankSplit(ctx, bad); err == nil {
		t.Fatal("falsche Summe angenommen")
	}
	refunds, err := payments.RefundItems(ctx)
	if err != nil || len(refunds) != 1 || refunds[0].OpenAmount != 11900 {
		t.Fatalf("Rückzahlungskandidat: %+v %v", refunds, err)
	}
	bad.Lines = bad.Lines[:1]
	bad.Lines[0].Text = ""
	if _, err := payments.BookBankSplit(ctx, bad); err == nil {
		t.Fatal("Rückzahlung ohne Grund")
	}
	bad.Lines[0].Text = "Auftrag entfällt"
	entry, err := payments.BookBankSplit(ctx, bad)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, entry, []bookedLine{{domain.SideCredit, domain.AccountBank, 11900}, {domain.SideDebit, accounting.AccountErhalteneAnzahlungen19, 10000}, {domain.SideDebit, domain.AccountUmsatzsteuer19, 1900}})
	if _, err := payments.BookBankSplit(ctx, bad); err == nil {
		t.Fatal("Rückzahlung doppelt")
	}
	received, err := env.journalRepo.FindActiveByBankTx(ctx, id)
	if err != nil || len(received) != 1 {
		t.Fatalf("Vereinnahmung: %+v %v", received, err)
	}
	if _, err := env.journal.Reverse(ctx, received[0].ID, "falsche Zuordnung"); err == nil {
		t.Fatal("Vereinnahmung trotz nachfolgender Rückzahlung storniert")
	}

	items, err := payments.OpenItems(ctx)
	if err != nil || len(items) != 1 || items[0].OpenAmount != 11900 {
		t.Fatalf("wieder offen: %+v %v", items, err)
	}
	if _, err := svc.CancelWithDocument(ctx, inv.ID, "Auftrag entfällt"); err != nil {
		t.Fatal(err)
	}
}

func TestAdvanceHistoricalPaymentsAndReversal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	payments := env.paymentsWithBank(t)
	payments.SetInvoiceService(svc)
	c := env.customer(t, "Historie GmbH", "DE", "")
	g := env.group(t, svc, c.ID, 20000)
	inv, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{GroupID: g.ID, Date: "2026-03-01", Net: 10000})
	if err != nil {
		t.Fatal(err)
	}
	id := env.bankCredit(t, "2026-04-15", 11900, "advance-history")
	entry, err := payments.Settle(ctx, PaymentRequest{BankTxID: &id, Allocations: []AllocationRequest{{AdvanceInvoiceID: inv.ID, SettledAmount: 11900}}})
	if err != nil {
		t.Fatal(err)
	}
	env.journal.SetClock(func() time.Time { return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC) })
	if _, err := env.journal.Reverse(ctx, entry.ID, "falsche Zuordnung"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		date  string
		count int
	}{{"2026-03-31", 1}, {"2026-04-30", 0}, {"2026-05-31", 1}} {
		items, err := payments.OpenItemsAt(ctx, tc.date)
		if err != nil || len(items) != tc.count {
			t.Fatalf("%s: %+v %v", tc.date, items, err)
		}
	}
	items, err := payments.OpenItems(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("Storno: %+v %v", items, err)
	}
	if _, err := payments.Settle(ctx, PaymentRequest{BankTxID: &id, Allocations: []AllocationRequest{{AdvanceInvoiceID: inv.ID, SettledAmount: 11900}}}); err != nil {
		t.Fatalf("erneute Zuordnung nach Storno: %v", err)
	}

}

func TestAdvancePartialFinalAndLockedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	payments := env.paymentsWithBank(t)
	payments.SetInvoiceService(svc)
	c := env.customer(t, "Teilbetrag GmbH", "DE", "")
	g := env.group(t, svc, c.ID, 20000)
	inv, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{GroupID: g.ID, Date: "2026-03-01", Net: 10000})
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewFestschreibungRepository(env.db)
	env.journal.SetFestschreibungRepo(repo)
	if err := repo.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: "month", PeriodLabel: "März", CutoffDate: "2026-03-31", ChainHead: domain.GenesisHash}); err != nil {
		t.Fatal(err)
	}
	id := env.bankCredit(t, "2026-03-15", 5950, "advance-locked")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{{AdvanceInvoiceID: inv.ID, SettledAmount: 5950}}}); err == nil {
		t.Fatal("Festschreibung umgangen")
	}
	env.assertMatch(t, id, domain.MatchStatusUnmatched)
	items, err := payments.OpenItems(ctx)
	if err != nil || len(items) != 1 || items[0].OpenAmount != 11900 {
		t.Fatalf("fehlgeschlagene Zahlung verändert Abschlag: %+v %v", items, err)
	}
	id = env.bankCredit(t, "2026-04-15", 5950, "advance-final-partial")
	if _, err := payments.BookBankSplit(ctx, BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{{AdvanceInvoiceID: inv.ID, SettledAmount: 5950}}}); err != nil {
		t.Fatal(err)
	}
	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{GroupID: g.ID, Date: "2026-06-30", ServiceDateFrom: "2026-06-01", ServiceDateTo: "2026-06-30", Items: []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 20000, TaxRate: domain.TaxRateStandard}}})
	if err != nil {
		t.Fatal(err)
	}
	if final.PrepaidAmount != 5950 {
		t.Fatalf("verrechnet: %v", final.PrepaidAmount)
	}
	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil || groups[0].Advances[0].GrossAmount != 11900 {
		t.Fatalf("Rechnungsbetrag verändert: %+v %v", groups, err)
	}
	open, err := svc.OpenAdvanceItems(ctx)
	if err != nil || len(open) != 0 {
		t.Fatalf("Abschlag neben Schlussrechnung offen: %+v %v", open, err)
	}
	if _, err := svc.RefundAdvance(ctx, RefundAdvanceRequest{AdvanceID: inv.ID, RefundDate: "2026-07-01", Reason: "Auftrag entfällt"}); err == nil {
		t.Fatal("verrechneter Abschlag zurückgezahlt")
	}
}
