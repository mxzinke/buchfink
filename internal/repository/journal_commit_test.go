package repository

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// --- Festschreibungszeitpunkt je Buchung ----------------------------------

func TestMarkCommittedStampsOnlyEntriesUpToTheCutoff(t *testing.T) {
	db := welle6DB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	march := welle6Entry(t, repo, "2026-03-15")
	june := welle6Entry(t, repo, "2026-06-15")

	stampedAt := time.Date(2026, 4, 10, 8, 30, 0, 0, time.UTC)
	count, err := repo.MarkCommitted(ctx, 2026, "2026-03-31", 42, stampedAt)
	if err != nil {
		t.Fatalf("die Festschreibung der Buchungen ist fehlgeschlagen: %v", err)
	}
	if count != 1 {
		t.Errorf("%d gestempelte Buchungen, erwartet 1", count)
	}

	committed, _ := repo.FindByID(ctx, march.ID)
	if committed.CommittedAt == nil {
		t.Fatal("die Buchung vor dem Stichtag muss einen Festschreibungszeitpunkt haben")
	}
	if !committed.CommittedAt.UTC().Equal(stampedAt) {
		t.Errorf("Festschreibungszeitpunkt %s, erwartet %s", committed.CommittedAt.UTC(), stampedAt)
	}
	if committed.FestschreibungID == nil || *committed.FestschreibungID != 42 {
		t.Errorf("die Buchung muss auf die Festschreibung verweisen, hat %v", committed.FestschreibungID)
	}

	open, _ := repo.FindByID(ctx, june.ID)
	if open.CommittedAt != nil {
		t.Error("eine Buchung nach dem Stichtag darf nicht festgeschrieben sein")
	}
}

func TestMarkCommittedKeepsTheEarlierStamp(t *testing.T) {
	db := welle6DB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	entry := welle6Entry(t, repo, "2026-01-15")

	first := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	second := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	if _, err := repo.MarkCommitted(ctx, 2026, "2026-01-31", 1, first); err != nil {
		t.Fatalf("die erste Festschreibung ist fehlgeschlagen: %v", err)
	}
	count, err := repo.MarkCommitted(ctx, 2026, "2026-06-30", 2, second)
	if err != nil {
		t.Fatalf("die zweite Festschreibung ist fehlgeschlagen: %v", err)
	}
	// Die zweite Festschreibung reicht weiter, darf aber den Zeitpunkt der
	// ersten nicht überschreiben — sonst wäre der Abstand Erfassung →
	// Festschreibung nicht mehr auswertbar.
	if count != 0 {
		t.Errorf("%d neu gestempelte Buchungen, erwartet 0", count)
	}
	stored, _ := repo.FindByID(ctx, entry.ID)
	if !stored.CommittedAt.UTC().Equal(first) {
		t.Errorf("Festschreibungszeitpunkt %s, erwartet den der ersten Festschreibung %s",
			stored.CommittedAt.UTC(), first)
	}
	if *stored.FestschreibungID != 1 {
		t.Errorf("die Buchung verweist auf Festschreibung %d, erwartet 1", *stored.FestschreibungID)
	}
}

func TestMarkCommittedLeavesTheHashChainIntact(t *testing.T) {
	db := welle6DB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	welle6Entry(t, repo, "2026-02-01")
	welle6Entry(t, repo, "2026-02-02")

	if _, err := repo.MarkCommitted(ctx, 2026, "2026-12-31", 1, time.Now().UTC()); err != nil {
		t.Fatalf("die Festschreibung ist fehlgeschlagen: %v", err)
	}

	entries, _ := repo.FindAll(ctx, 2026)
	result := accounting.NewHashChain().VerifyChain(entries)
	if !result.IsValid {
		// Der Festschreibungszeitpunkt steht außerhalb der kanonischen Form.
		// Stünde er darin, bräche jede Festschreibung die Kette des Jahres.
		t.Fatalf("die Festschreibung darf die Hash-Kette nicht berühren: %s", result.Message)
	}
}

func welle6Entry(t *testing.T, repo domain.JournalRepository, bookingDate string) *domain.JournalEntry {
	t.Helper()
	entry := &domain.JournalEntry{
		FiscalYear:      2026,
		BookingDate:     bookingDate,
		DocumentDate:    bookingDate,
		ServiceDateFrom: bookingDate,
		ServiceDateTo:   bookingDate,
		Description:     "Testbuchung " + bookingDate,
		Source:          domain.EntrySourceManual,
		Kind:            domain.EntryKindNormal,
		Currency:        "EUR",
		// Der Umrechnungskurs steht ausdrücklich da: die Spalte hat einen
		// Vorgabewert in der Datenbank, und ein nicht belegtes Feld käme mit
		// 1.000.000 zurück, während es mit 0 gehasht wurde — die Kette wäre
		// nach dem Lesen gebrochen, ohne dass jemand etwas geändert hätte.
		ExchangeRateMicros: 1_000_000,
		AppVersion:         "test",
		Actor:              "test@rechner",
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	if err := repo.Append(context.Background(), entry, accounting.NewHashChain().CalculateHash); err != nil {
		t.Fatalf("Buchung konnte nicht angehängt werden: %v", err)
	}
	return entry
}
