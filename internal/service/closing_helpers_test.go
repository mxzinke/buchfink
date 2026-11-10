package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// closingEntry sucht die erste Buchung eines Jahres unter einer Belegnummer.
func (e *testEnv) closingEntry(t *testing.T, year int, document string) *domain.JournalEntry {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	for i := range entries {
		if entries[i].DocumentNumber == document && entries[i].Kind != domain.EntryKindReversal {
			return &entries[i]
		}
	}
	t.Fatalf("keine Buchung %s im Journal %d", document, year)
	return nil
}

// standingClosingEntry sucht die Buchung eines Jahres, die noch steht — die
// stornierten bleiben draußen.
func (e *testEnv) standingClosingEntry(t *testing.T, year int, document string) *domain.JournalEntry {
	t.Helper()
	ctx := context.Background()
	entries, err := e.journalRepo.FindAll(ctx, year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	for i := range entries {
		if entries[i].DocumentNumber != document || entries[i].Kind == domain.EntryKindReversal {
			continue
		}
		reversal, err := e.journalRepo.FindReversalOf(ctx, entries[i].ID)
		if err != nil {
			t.Fatalf("Storno zu %s: %v", entries[i].EntryNumber, err)
		}
		if reversal == nil {
			return &entries[i]
		}
	}
	t.Fatalf("keine stehende Buchung %s im Journal %d", document, year)
	return nil
}

// entryByNumberAnyYear sucht eine Buchung über alle Geschäftsjahre. Eine
// Korrektur hat den Tag ihrer Erstellung und liegt deshalb nicht zwingend im
// Jahr der Buchung, die sie korrigiert.
func (e *testEnv) entryByNumberAnyYear(t *testing.T, number string) *domain.JournalEntry {
	t.Helper()
	ctx := context.Background()
	years, err := e.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		t.Fatalf("Geschäftsjahre: %v", err)
	}
	for _, year := range years {
		entries, err := e.journalRepo.FindAll(ctx, year)
		if err != nil {
			t.Fatalf("Journal %d: %v", year, err)
		}
		for i := range entries {
			if entries[i].EntryNumber == number {
				return &entries[i]
			}
		}
	}
	t.Fatalf("die Buchung %s steht in keinem Geschäftsjahr", number)
	return nil
}

// countEntries zählt die Buchungen eines Geschäftsjahres.
func (e *testEnv) countEntries(t *testing.T, year int) int {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	return len(entries)
}
