package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die leere Zinstabelle ist der Zustand jeder frischen Installation: die Sätze
// der Deutschen Bundesbank werden gepflegt, nicht ausgeliefert. `max(month)`
// liefert über eine leere Menge NULL, und ein NULL in einen string zu scannen
// bricht ab — mit dem Fehler stünde die Rückstellungsvorschau still, obwohl
// „kein Satz hinterlegt" ein Befund und kein Fehler ist.
func TestFindLatestUpToOnEmptyTableReturnsNothing(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	repo := NewDiscountRateRepository(db)

	rates, err := repo.FindLatestUpTo(context.Background(), "2026-12")
	if err != nil {
		t.Fatalf("leere Zinstabelle lesen: %v", err)
	}
	if rates == nil {
		t.Error("die Sätze kommen als nil zurück — an der Oberfläche wäre das `null`")
	}
	if len(rates) != 0 {
		t.Errorf("erwartet keine Sätze, bekommen %d", len(rates))
	}
}

// Und mit gepflegten Sätzen: gesucht wird der jüngste Monat, der nicht nach dem
// Stichtag liegt.
func TestFindLatestUpToPicksTheYoungestMonthBeforeTheCutoff(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	repo := NewDiscountRateRepository(db)
	ctx := context.Background()

	if err := repo.Save(ctx, []domain.DiscountRate{
		{Month: "2026-09", Years: 3, RateMicros: 15_000, Average: 7},
		{Month: "2026-11", Years: 3, RateMicros: 16_000, Average: 7},
		{Month: "2027-02", Years: 3, RateMicros: 17_000, Average: 7},
	}); err != nil {
		t.Fatalf("Sätze pflegen: %v", err)
	}

	rates, err := repo.FindLatestUpTo(ctx, "2026-12")
	if err != nil {
		t.Fatalf("Sätze lesen: %v", err)
	}
	if len(rates) != 1 || rates[0].Month != "2026-11" {
		t.Fatalf("erwartet die Sätze vom November 2026, bekommen %+v", rates)
	}

	// Vor dem ersten gepflegten Monat gibt es nichts — auch das ist kein Fehler.
	older, err := repo.FindLatestUpTo(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Sätze vor der ersten Veröffentlichung: %v", err)
	}
	if len(older) != 0 {
		t.Errorf("erwartet keine Sätze, bekommen %+v", older)
	}
}
