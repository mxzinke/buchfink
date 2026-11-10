package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestInvoiceAmendmentDifferenceAndTextOnly(t *testing.T) {
	for _, net := range []domain.Cents{100000, 80000, 120000} {
		t.Run(net.String(), func(t *testing.T) {
			env := newTestEnv(t)
			ctx := context.Background()
			svc := env.invoicesWired(t)
			customer := env.customer(t, "Kunde GmbH", "DE", "")
			original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
			if err := svc.Issue(ctx, original); err != nil {
				t.Fatal(err)
			}
			replacement := env.simpleInvoice(customer.ID, "2026-04-15", net)
			replacement.ServiceDateFrom, replacement.ServiceDateTo = original.ServiceDateFrom, original.ServiceDateTo
			replacement.Items[0].Description = "Berichtigte Leistungsbeschreibung"
			preview, err := svc.PreviewInvoiceAmendment(ctx, original.ID, replacement)
			if err != nil {
				t.Fatal(err)
			}
			corrected, err := svc.CorrectInvoice(ctx, original.ID, "Leistungsumfang", replacement)
			if err != nil {
				t.Fatal(err)
			}
			if preview.Net != net-100000 || preview.BookingDate != "2026-04-15" {
				t.Fatalf("Vorschau: %+v", preview)
			}
			if net == 100000 {
				if corrected.JournalEntryID != nil || len(preview.Lines) != 0 {
					t.Fatal("Textänderung hat eine Buchung")
				}
			} else {
				entry, err := env.journalRepo.FindByID(ctx, *corrected.JournalEntryID)
				if err != nil {
					t.Fatal(err)
				}
				debit, credit := domain.SideDebit, domain.SideCredit
				diff := net - 100000
				if diff < 0 {
					diff = -diff
					debit, credit = credit, debit
				}
				assertLines(t, entry, []bookedLine{{debit, customer.LedgerAccount, diff * 119 / 100}, {credit, "4400", diff}, {credit, "3806", diff * 19 / 100}})
				for _, line := range entry.Lines {
					if got := accounting.VatPeriodFor(entry, line, ""); got != "2026-04-15" {
						t.Fatal(got)
					}
				}
			}
			if _, err := svc.CorrectInvoice(ctx, original.ID, "noch einmal", replacement); err == nil {
				t.Fatal("ersetzte Rechnung erneut berichtigt")
			}
			cancellation, err := svc.PreviewInvoiceAmendment(ctx, corrected.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if cancellation.Net != -net || cancellation.Gross != -net*119/100 {
				t.Fatalf("Stornovorschau: %+v", cancellation)
			}
			if _, err := svc.CancelWithDocument(ctx, corrected.ID, "Auftrag entfällt"); err != nil {
				t.Fatal(err)
			}
			items, err := env.payments(t).OpenItems(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 0 {
				t.Fatalf("Posten nach Storno: %+v", items)
			}
		})
	}
}

func TestPaidInvoiceCancellationRetainsRefundableCredit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatal(err)
	}
	payments := env.payments(t)
	if _, err := payments.Settle(ctx, PaymentRequest{PaymentAccount: domain.AccountBank, PaymentDate: "2026-03-20",
		Allocations: []AllocationRequest{{OpenItemEntryID: *original.JournalEntryID, SettledAmount: 119000}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelWithDocument(ctx, original.ID, "Leistung entfällt"); err != nil {
		t.Fatal(err)
	}
	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].OpenAmount != 119000 || !items[0].IsCredit() {
		t.Fatalf("Kundenguthaben: %+v", items)
	}
	if _, err := payments.Settle(ctx, PaymentRequest{PaymentAccount: domain.AccountBank, PaymentDate: svc.today(),
		Allocations: []AllocationRequest{{OpenItemEntryID: *original.JournalEntryID, SettledAmount: 119000}}}); err != nil {
		t.Fatal(err)
	}
	items, err = payments.OpenItems(ctx)
	if err != nil || len(items) != 0 {
		t.Fatalf("nach Erstattung: %+v, %v", items, err)
	}
}

func TestInvoiceAmendmentRespectsCommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatal(err)
	}
	env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))
	env.commitUntil(t, "2026-03-31")
	replacement := env.simpleInvoice(customer.ID, "2026-03-15", 100000)
	if _, err := svc.CorrectInvoice(ctx, original.ID, "Beschreibung", replacement); err == nil {
		t.Fatal("Berichtigung in festgeschriebener Periode")
	}
	replacement.Date = "2026-04-15"
	if _, err := svc.CorrectInvoice(ctx, original.ID, "Beschreibung", replacement); err != nil {
		t.Fatal(err)
	}
}

func TestInvoiceCancellationRequiresWriteoffReversal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatal(err)
	}
	writtenOff, err := env.payments(t).WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: *original.JournalEntryID, Date: "2026-04-01", Reason: "Uneinbringlich", ReceiptID: writeOffEvidence(t, env),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelWithDocument(ctx, original.ID, "Auftrag entfällt"); err == nil {
		t.Fatal("ausgebuchte Forderung doppelt gemindert")
	}
	if _, err := env.journal.Reverse(ctx, writtenOff.ID, "Rechnung wird storniert"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelWithDocument(ctx, original.ID, "Auftrag entfällt"); err != nil {
		t.Fatal(err)
	}
}

func TestInvoiceAmendmentVATAndZMPeriods(t *testing.T) {
	t.Run("UStVA", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.invoicesWired(t)
		customer := env.customer(t, "Kunde GmbH", "DE", "")
		original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
		if err := svc.Issue(ctx, original); err != nil {
			t.Fatal(err)
		}
		replacement := env.simpleInvoice(customer.ID, "2026-04-15", 80000)
		replacement.ServiceDateFrom, replacement.ServiceDateTo = "2026-03-01", "2026-03-01"
		if _, err := svc.CorrectInvoice(ctx, original.ID, "Minderung", replacement); err != nil {
			t.Fatal(err)
		}
		for _, period := range []struct {
			from, to string
			net, tax domain.Cents
		}{
			{"2026-03-01", "2026-03-31", 100000, 19000}, {"2026-04-01", "2026-04-30", -20000, -3800},
		} {
			summary, err := env.vat().Summary(ctx, 2026, period.from, period.to)
			if err != nil {
				t.Fatal(err)
			}
			if len(summary.TaxableRevenue) != 1 || summary.TaxableRevenue[0].Net != period.net || summary.TaxableRevenue[0].Tax != period.tax {
				t.Fatalf("%s: %+v", period.from, summary)
			}
		}
	})
	t.Run("ZM", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.invoicesWired(t)
		customer := env.customer(t, "Client", "FR", "FR12345678901")
		original := env.euService(t, customer.ID, "2026-03-01", 100000)
		replacement := env.simpleInvoice(customer.ID, "2026-04-15", 80000)
		replacement.ServiceDateFrom, replacement.ServiceDateTo = "2026-03-01", "2026-03-01"
		replacement.TaxTreatment = domain.TaxTreatmentReverseChargeSupply
		replacement.Items[0].TaxRate = domain.TaxRateNone
		if _, err := svc.CorrectInvoice(ctx, original.ID, "Minderung", replacement); err != nil {
			t.Fatal(err)
		}
		for _, period := range []struct {
			key    string
			amount domain.Cents
		}{{"2026-Q1", 100000}, {"2026-Q2", -20000}} {
			draft, err := env.zmReturns(t).Draft(ctx, period.key)
			if err != nil {
				t.Fatal(err)
			}
			if len(draft.Lines) != 1 || draft.Lines[0].Amount != period.amount {
				t.Fatalf("%s: %+v", period.key, draft.Lines)
			}
		}
	})
}
