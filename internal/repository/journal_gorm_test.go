package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// GORM lädt die Zeilen einer Buchung über eine IN-Liste mit einer Bindvariablen
// je Elternzeile, und SQLite bricht bei 32.766 davon ab. Weil die offenen Posten
// über alle Jahre gelesen werden, ist diese Zahl erreichbar — und dann käme
// nicht eine langsame Liste zurück, sondern ein Fehler, der die Liste der
// offenen Posten und damit jede Zahlung stilllegt.
func TestOpenItemCandidatesSurviveTheBindVariableLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("legt 33.000 Buchungen an, zu langsam für -short")
	}
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}

	const n = 33000 // knapp über SQLITE_MAX_VARIABLE_NUMBER
	entries := make([]domain.JournalEntry, 0, n)
	for i := 0; i < n; i++ {
		entries = append(entries, domain.JournalEntry{
			FiscalYear: 2026, EntryNumber: fmt.Sprintf("2026-%06d", i),
			BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
			Description: "Massentest", Source: domain.EntrySourceReceipt, Currency: "EUR",
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "6815", Amount: 100},
				{Side: domain.SideCredit, Account: "70001", Amount: 100},
			},
		})
	}
	if err := db.CreateInBatches(&entries, 200).Error; err != nil {
		t.Fatalf("Buchungen anlegen: %v", err)
	}

	got, err := NewJournalRepository(db).FindOpenItemCandidates(context.Background(), 2026)
	if err != nil {
		t.Fatalf("offene Posten über der Bindvariablengrenze: %v", err)
	}
	if len(got) != n {
		t.Fatalf("%d Buchungen gelesen, erwartet %d", len(got), n)
	}
	// Und mit ihren Zeilen, nicht nur als Rümpfe.
	if len(got[0].Lines) != 2 {
		t.Errorf("die Buchung kam ohne ihre Zeilen zurück (%d)", len(got[0].Lines))
	}
}
