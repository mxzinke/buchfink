package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestBankReversalKeepsMatchUntilLastHolder(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()
	e.paymentsWithBank(t)
	id := e.bankLine(t, "2026-03-01", -1500, "Hausbank", "Kontoführung")
	first := simpleEntry("6855", domain.AccountBank, 1500)
	first.BankTxID = &id
	if _, err := e.journal.Post(ctx, first); err != nil {
		t.Fatal(err)
	}
	second, err := e.journal.Post(ctx, simpleEntry("6855", domain.AccountBank, 1500))
	if err != nil {
		t.Fatal(err)
	}
	// Ein Bestand mit zwei Haltern bleibt bis zum letzten Storno zugeordnet.
	if err := e.db.Model(&domain.JournalEntry{}).Where("id = ?", second.ID).Update("bank_tx_id", id).Error; err != nil {
		t.Fatal(err)
	}
	for i, entry := range []*domain.JournalEntry{first, second} {
		if _, err := e.journal.Reverse(ctx, entry.ID, "Doppelte Zuordnung"); err != nil {
			t.Fatal(err)
		}
		want := domain.MatchStatusMatched
		if i == 1 {
			want = domain.MatchStatusUnmatched
		}
		e.assertMatch(t, id, want)
	}
}

func TestReceiptSealDependsOnBookingPath(t *testing.T) {
	for _, source := range []domain.EntrySource{
		domain.EntrySourceManual, domain.EntrySourceReceipt, domain.EntrySourceAdvance,
		domain.EntrySourceClosing, domain.EntrySourceOpening, domain.EntrySourcePayment,
	} {
		t.Run(string(source), func(t *testing.T) {
			e := newTestEnv(t)
			ctx := context.Background()
			r := e.fileIncoming(t, "beleg.pdf")
			entry := simpleEntry("6815", domain.AccountBank, 10000)
			entry.Source, entry.ReceiptID = source, &r.ID
			posted, err := e.journal.Post(ctx, entry)
			if err != nil {
				t.Fatal(err)
			}
			assertReceiptSeal(t, e, r.ID, nil)
			if _, err := e.journal.Reverse(ctx, posted.ID, "Korrektur"); err != nil {
				t.Fatal(err)
			}
			assertReceiptSeal(t, e, r.ID, nil)
		})
	}
	for _, path := range []string{"eingang", "handbuchung", "abschluss"} {
		t.Run(path, func(t *testing.T) {
			e := newTestEnv(t)
			ctx := context.Background()
			v := e.vendor(t, "Lieferant", "DE", "")
			req := e.receipt(t, v.ID, "buerobedarf", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
			var posted *domain.JournalEntry
			var err error
			switch path {
			case "eingang":
				posted, err = e.posting.PostIncomingReceipt(ctx, req)
			case "handbuchung":
				posted, err = e.posting.PostManualEntry(ctx, ManualEntryRequest{ReceiptID: req.ReceiptID, Entry: *simpleEntry("6815", domain.AccountBank, 10000)})
			case "abschluss":
				r, getErr := e.receipts.Get(ctx, req.ReceiptID)
				if getErr != nil {
					t.Fatal(getErr)
				}
				entry := simpleEntry("6815", domain.AccountBank, 10000)
				entry.Source = domain.EntrySourceClosing
				attachVoucher(entry, r)
				posted, err = atomically(ctx, e.journal, func(ctx context.Context) (*domain.JournalEntry, error) {
					return postWithVoucher(ctx, e.journal, e.receipts, entry, r)
				})
			}
			if err != nil {
				t.Fatal(err)
			}
			assertReceiptSeal(t, e, req.ReceiptID, &posted.ID)
			if _, err := e.journal.Reverse(ctx, posted.ID, "Korrektur"); err != nil {
				t.Fatal(err)
			}
			assertReceiptSeal(t, e, req.ReceiptID, &posted.ID)
		})
	}
}

func assertReceiptSeal(t *testing.T, e *testEnv, receiptID uint, entryID *uint) {
	t.Helper()
	r, err := e.receipts.Get(context.Background(), receiptID)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.ReceiptStatusFiled
	if entryID != nil {
		want = domain.ReceiptStatusSealed
	}
	if r.Status != want || !reflect.DeepEqual(r.JournalEntryID, entryID) {
		t.Fatalf("Belegsiegel: Status %s, Buchung %v; erwartet %s, %v", r.Status, r.JournalEntryID, want, entryID)
	}
}

func TestIncomingAdvanceReversalKeepsRegisterRecord(t *testing.T) {
	e, req := incomingGoldenFixture(t, "anzahlung")
	ctx := context.Background()
	entry, err := e.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	before, err := e.posting.OpenVendorAdvances(ctx, req.ContactID)
	if err != nil || len(before) != 1 {
		t.Fatalf("Anzahlung: %v, %v", before, err)
	}
	if _, err := e.journal.Reverse(ctx, entry.ID, "Korrektur"); err != nil {
		t.Fatal(err)
	}
	after, err := e.posting.OpenVendorAdvances(ctx, req.ContactID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("Anzahlungskartei nach Storno: %v, %v; erwartet %v", after, err, before)
	}
}

func TestIncomingProvisionReversalKeepsConsumption(t *testing.T) {
	e, req := incomingGoldenFixture(t, "rueckstellungsverbrauch")
	ctx := context.Background()
	entry, err := e.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewProvisionRepository(e.db)
	before, err := repo.FindByID(ctx, req.ProvisionID)
	if err != nil || before.Balance() != 0 {
		t.Fatalf("Rückstellung: %v, %v", before, err)
	}
	if _, err := e.journal.Reverse(ctx, entry.ID, "Korrektur"); err != nil {
		t.Fatal(err)
	}
	after, err := repo.FindByID(ctx, req.ProvisionID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("Rückstellungskartei nach Storno: %v, %v; erwartet %v", after, err, before)
	}
}

func TestOpeningBalanceKeepsOneReceiptSealAfterReversal(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()
	r := fileClosingStatement(t, e)
	customer := e.customer(t, "Kunde", "DE", "")
	booked, err := e.journal.BookOpeningBalance(ctx, openingRequest(t, e, customer.ID, r.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(booked.Entries) < 2 {
		t.Fatalf("erwartet mehrere Vortragsbuchungen: %+v", booked.Entries)
	}
	assertReceiptSeal(t, e, r.ID, &booked.Entries[0].ID)
	for _, entry := range booked.Entries {
		if _, err := e.journal.ReverseOn(ctx, entry.ID, "Korrekturvortrag", "2026-01-01"); err != nil {
			t.Fatal(err)
		}
	}
	assertReceiptSeal(t, e, r.ID, &booked.Entries[0].ID)
}
