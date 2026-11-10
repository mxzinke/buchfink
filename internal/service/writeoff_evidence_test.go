package service

import (
	"context"
	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"testing"
	"time"
)

func writeOffEvidence(t *testing.T, env *testEnv) uint {
	return filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindOther, DocumentDate: "2026-03-01", Subject: "Nachweis Uneinbringlichkeit: Schreiben des Insolvenzverwalters",
		Files: []NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "insolvenz.pdf", Content: []byte("%PDF-1.4 Insolvenz mangels Masse\n")}},
	}).ID
}

func TestWriteOffAllowanceAndPreview(t *testing.T) {
	for _, amount := range []domain.Cents{119000, 23800} {
		t.Run(amount.String(), func(t *testing.T) {
			env := newTestEnv(t)
			ctx := context.Background()
			customer := env.customer(t, "Insolventer Kunde GmbH", "DE", "")
			invoice := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
			if err := env.invoicesWired(t).Issue(ctx, invoice); err != nil {
				t.Fatal(err)
			}
			payments := env.payments(t)
			m := env.closingModules(t)
			m.bookings.SetOpenItemSource(payments.OpenItemSource())
			if _, err := m.bookings.BookReceivableAllowance(ctx, ReceivableAllowanceRequest{FiscalYear: 2026, Specific: []accounting.SpecificAllowance{{EntryID: *invoice.JournalEntryID, Percent: 50, Reason: "Insolvenzrisiko"}}}); err != nil {
				t.Fatal(err)
			}
			req := WriteOffRequest{OpenItemEntryID: *invoice.JournalEntryID, Amount: amount, Date: "2027-03-01", Reason: "Mitteilung Insolvenzverwalter: Ausfall", ReceiptID: writeOffEvidence(t, env)}
			before, err := payments.OpenItems(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if before[0].SpecificAllowance != 50000 {
				t.Fatalf("EWB = %v", before)
			}
			early := req
			early.Date = "2026-06-01"
			if _, err := payments.PreviewWriteOff(ctx, early); err == nil {
				t.Fatal("Ausbuchung vor späterer EWB erlaubt")
			}
			preview, err := payments.PreviewWriteOff(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if preview.ID != 0 {
				t.Fatal("Vorschau hat geschrieben")
			}
			entry, err := payments.WriteOffOpenItem(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			used := domain.Cents(50000)
			if amount == 23800 {
				used = 20000
			}
			if got := lineAmount(t, entry, accounting.AccountEWBShortTerm, domain.SideDebit); got != used {
				t.Fatalf("EWB-Auflösung %v statt %v", got, used)
			}
			if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideDebit); got != amount-domain.TaxRateStandard.NetFromGross(amount) {
				t.Fatalf("Steuer = %v", got)
			}
			items, err := payments.OpenItems(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if amount == 119000 && len(items) != 0 {
				t.Fatalf("Restposten: %v", items)
			}
			if amount == 23800 && (len(items) != 1 || items[0].OpenAmount != 95200 || items[0].SpecificAllowance != 30000) {
				t.Fatalf("Teilrest: %v", items)
			}
			env.journal.SetClock(func() time.Time { return time.Date(2027, 3, 2, 12, 0, 0, 0, time.UTC) })
			if _, err := env.journal.Reverse(ctx, entry.ID, "Ausfall irrtümlich angenommen"); err != nil {
				t.Fatal(err)
			}
			items, err = payments.OpenItems(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 1 || items[0].OpenAmount != 119000 || items[0].SpecificAllowance != 50000 {
				t.Fatalf("Storno: %v", items)
			}
		})
	}
}

func TestWriteOffInsolvencyTenantThreeAndLockedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kleinkram Werbeagentur GmbH", "DE", "")
	invoice := env.domesticInvoice(t, customer.ID, "2026-06-30", 352000)
	req := WriteOffRequest{OpenItemEntryID: *invoice.JournalEntryID, Date: "2026-09-01", Reason: "Insolvenzeröffnung; Schreiben vom 04.09.2026", ReceiptID: writeOffEvidence(t, env)}
	payments := env.payments(t)
	entry, err := payments.WriteOffOpenItem(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if lineAmount(t, entry, accounting.AccountForderungsverluste19, domain.SideDebit) != 352000 || lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideDebit) != 66880 {
		t.Fatalf("Ausfall: %+v", entry.Lines)
	}
	q3 := env.vatDraft(t, "2026-Q3")
	if q3.Tax("81") != -66880 {
		t.Fatalf("Steuer III/2026 = %v", q3.Tax("81"))
	}
	invoice = env.domesticInvoice(t, customer.ID, "2026-07-01", 100000)
	req.OpenItemEntryID = *invoice.JournalEntryID
	repo := repository.NewFestschreibungRepository(env.db)
	env.journal.SetFestschreibungRepo(repo)
	if err := repo.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: "month", PeriodLabel: "September 2026", CutoffDate: "2026-09-30", ChainHead: domain.GenesisHash}); err != nil {
		t.Fatal(err)
	}
	if _, err := payments.PreviewWriteOff(ctx, req); err == nil {
		t.Fatal("Vorschau in festgeschriebener Periode erlaubt")
	}
	if _, err := payments.WriteOffOpenItem(ctx, req); err == nil {
		t.Fatal("Ausbuchung in festgeschriebener Periode erlaubt")
	}
}

func TestWriteOffRefusesUnassignedAllowance(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde mit Altbestand", "DE", "")
	invoice := env.domesticInvoice(t, customer.ID, "2026-03-01", 100000)
	env.post(t, "2026-03-31", accounting.AccountEWBExpense, accounting.AccountEWBShortTerm, 50000)
	payments := env.payments(t)
	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].AllowanceUnassigned {
		t.Fatalf("Altbestand wird verschwiegen: %+v", items)
	}
	req := WriteOffRequest{OpenItemEntryID: *invoice.JournalEntryID, ReceiptID: writeOffEvidence(t, env), Date: "2026-04-01", Reason: "Insolvenz"}
	if _, err := payments.WriteOffOpenItem(ctx, req); err == nil {
		t.Fatal("nicht zugeordnete EWB darf nicht übergangen werden")
	}
}
