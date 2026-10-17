package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die einzeilige Anschrift wird in Straße, PLZ und Ort zerlegt.
func TestBackfillContactAddresses(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	// Die Datenbank im Arbeitsspeicher lebt, solange ihre Verbindung offen ist.
	t.Cleanup(func() { _ = CloseDB(db) })

	contact := &domain.Contact{
		Name:    "Kunde GmbH",
		Type:    domain.ContactTypeCustomer,
		Address: "Kundenweg 2\n10115 Berlin",
	}
	if err := db.Create(contact).Error; err != nil {
		t.Fatalf("Kontakt anlegen: %v", err)
	}
	if err := BackfillContactAddresses(db); err != nil {
		t.Fatalf("Adressmigration: %v", err)
	}

	repo := NewContactRepository(db)
	migrated, err := repo.FindByID(context.Background(), contact.ID)
	if err != nil {
		t.Fatalf("Kontakt lesen: %v", err)
	}
	if migrated.Street != "Kundenweg 2" || migrated.PostalCode != "10115" || migrated.City != "Berlin" {
		t.Errorf("Anschrift = %q / %q / %q", migrated.Street, migrated.PostalCode, migrated.City)
	}
}
