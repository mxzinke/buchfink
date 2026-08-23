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
func TestEntryReadsSurviveTheBindVariableLimit(t *testing.T) {
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

	repo := NewJournalRepository(db)
	ctx := context.Background()

	// Alle Wege, die ganze Buchungen mit ihren Zeilen lesen — darunter der, über
	// den die Integritätsprüfung die Hash-Kette nachrechnet.
	for name, read := range map[string]func() ([]domain.JournalEntry, error){
		"FindAll": func() ([]domain.JournalEntry, error) { return repo.FindAll(ctx, 2026) },
		"FindByBookingDateRange": func() ([]domain.JournalEntry, error) {
			return repo.FindByBookingDateRange(ctx, 2026, "2026-01-01", "2026-12-31")
		},
		"FindOpenItemCandidates": func() ([]domain.JournalEntry, error) { return repo.FindOpenItemCandidates(ctx, 2026) },
		"FindByAccount":          func() ([]domain.JournalEntry, error) { return repo.FindByAccount(ctx, "70001", 2026) },
	} {
		got, err := read()
		if err != nil {
			t.Errorf("%s über der Bindvariablengrenze: %v", name, err)
			continue
		}
		if len(got) != n {
			t.Errorf("%s las %d Buchungen, erwartet %d", name, len(got), n)
			continue
		}
		// Und mit ihren Zeilen, nicht nur als Rümpfe.
		if len(got[0].Lines) != 2 {
			t.Errorf("%s lieferte eine Buchung ohne ihre Zeilen (%d)", name, len(got[0].Lines))
		}
	}
}

// FindByAccount und FindByContact liefern chronologisch, nicht in der
// Reihenfolge, in der die Datenbank die Zeilen zurückgibt. Beide lesen jetzt
// nach Kennung gestapelt und sortieren danach — dass das Ergebnis dasselbe
// bleibt, hält dieser Test fest.
func TestAccountAndContactReadsStayChronological(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	contact := uint(7)
	// Bewusst in umgekehrter zeitlicher Reihenfolge angelegt.
	for i, date := range []string{"2026-09-01", "2026-03-01", "2026-06-01"} {
		entry := domain.JournalEntry{
			FiscalYear: 2026, EntryNumber: fmt.Sprintf("2026-%06d", i),
			BookingDate: date, DocumentDate: date, Description: date,
			Source: domain.EntrySourceReceipt, Currency: "EUR", ContactID: &contact,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "6815", Amount: 100},
				{Side: domain.SideCredit, Account: "70001", Amount: 100},
			},
		}
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("Buchung %s: %v", date, err)
		}
	}

	repo := NewJournalRepository(db)
	ctx := context.Background()
	want := []string{"2026-03-01", "2026-06-01", "2026-09-01"}

	for name, read := range map[string]func() ([]domain.JournalEntry, error){
		"FindByAccount": func() ([]domain.JournalEntry, error) { return repo.FindByAccount(ctx, "70001", 2026) },
		"FindByContact": func() ([]domain.JournalEntry, error) { return repo.FindByContact(ctx, contact, 2026) },
	} {
		got, err := read()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s las %d Buchungen, erwartet %d", name, len(got), len(want))
		}
		for i := range want {
			if got[i].BookingDate != want[i] {
				t.Errorf("%s: Position %d ist %s, erwartet %s", name, i+1, got[i].BookingDate, want[i])
			}
		}
	}
}
