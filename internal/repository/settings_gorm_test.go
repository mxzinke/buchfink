package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

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
