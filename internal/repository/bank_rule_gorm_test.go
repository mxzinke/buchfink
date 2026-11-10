package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die gelernte Regel wird fortgeschrieben und nicht verdoppelt, und die
// Nachweisgrenze übersteht das Speichern.

func TestBankRuleIsUpdatedInsteadOfDuplicated(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	// Die Datenbank im Arbeitsspeicher lebt, solange ihre Verbindung offen ist.
	t.Cleanup(func() { _ = CloseDB(db) })
	ctx := context.Background()
	repo := NewBankRuleRepository(db)

	first := &domain.BankRule{
		Pattern: "meier hausverwaltung | miete buero", Label: "Hausverwaltung Meier",
		CounterAccount: "6310", MoneyIn: false, LastUsedAt: "2026-03-01",
	}
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("Regel anlegen: %v", err)
	}
	// Dieselbe Sache, anderes Konto: der Anwender hat es sich anders überlegt.
	second := &domain.BankRule{
		Pattern: "meier hausverwaltung | miete buero", Label: "Hausverwaltung Meier",
		CounterAccount: "6315", MoneyIn: false, LastUsedAt: "2026-04-01",
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("Regel fortschreiben: %v", err)
	}

	rules, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Regeln lesen: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("erwartet eine Regel, erhalten %d", len(rules))
	}
	if rules[0].CounterAccount != "6315" {
		t.Errorf("Gegenkonto = %q, erwartet das zuletzt bestätigte 6315", rules[0].CounterAccount)
	}
	if rules[0].Hits != 2 {
		t.Errorf("Bestätigungen = %d, erwartet 2", rules[0].Hits)
	}
	if rules[0].LastUsedAt != "2026-04-01" {
		t.Errorf("zuletzt benutzt = %q, erwartet 2026-04-01", rules[0].LastUsedAt)
	}

	if err := repo.Delete(ctx, rules[0].ID); err != nil {
		t.Fatalf("Regel löschen: %v", err)
	}
	if remaining, err := repo.FindAll(ctx); err != nil || len(remaining) != 0 {
		t.Errorf("nach dem Löschen bleiben %d Regeln (%v)", len(remaining), err)
	}
}
