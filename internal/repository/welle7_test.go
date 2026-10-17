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

// Die Nachweisgrenze übersteht das Speichern; ein leeres Feld bleibt die
// Voreinstellung und schaltet nichts stumm ab.
func TestSettingsKeepCheckThreshold(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	// Die Datenbank im Arbeitsspeicher lebt, solange ihre Verbindung offen ist.
	t.Cleanup(func() { _ = CloseDB(db) })
	ctx := context.Background()
	repo := NewSettingsRepository(db)

	defaults, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if defaults.InvoiceCheckThreshold != 100_000 {
		t.Errorf("Nachweisgrenze = %s €, erwartet 1.000,00", defaults.InvoiceCheckThreshold)
	}

	if err := repo.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
	}); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}

	saved, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	// Das leere Feld der Grenze wird zur Voreinstellung und nicht zu null.
	if saved.InvoiceCheckThreshold != 100_000 {
		t.Errorf("Nachweisgrenze = %s €, erwartet die Voreinstellung 1.000,00",
			saved.InvoiceCheckThreshold)
	}
}

// Ein Formular, das die Nachweisgrenze nicht kennt, ändert sie nicht.
//
// Ein Dialog, der nur die Anschrift ändert, schickt für die übrigen Felder den
// Nullwert — und würde eine gepflegte Grenze auf die Voreinstellung
// zurücksetzen, ohne dass jemand das wollte.
func TestSettingsDoNotResetMaintainedValues(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	// Die Datenbank im Arbeitsspeicher lebt, solange ihre Verbindung offen ist.
	t.Cleanup(func() { _ = CloseDB(db) })
	ctx := context.Background()
	repo := NewSettingsRepository(db)

	if err := repo.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
		InvoiceCheckThreshold: 500_000,
	}); err != nil {
		t.Fatalf("Einstellungen pflegen: %v", err)
	}

	// Ein zweiter Aufruf ohne die Grenze.
	if err := repo.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026, Street: "Hauptstraße 1",
	}); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}

	saved, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if saved.InvoiceCheckThreshold != 500_000 {
		t.Errorf("Nachweisgrenze = %s €, erwartet die gepflegten 5.000,00", saved.InvoiceCheckThreshold)
	}
	if saved.StreetLine() != "Hauptstraße 1" {
		t.Errorf("die geänderte Anschrift fehlt: %q", saved.StreetLine())
	}
}
