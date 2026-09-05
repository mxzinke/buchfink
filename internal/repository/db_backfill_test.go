package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/security"
)

// Die Adressmigration muss die Verschlüsselung der Felder mitnehmen.
//
// Straße, PLZ und Ort sind als `serializer:encrypted` deklariert. GORM wendet
// den Serializer nur auf dem Struct-Weg an; ein Map-Update schriebe Klartext in
// die Spalten, und beim nächsten Lesen scheiterte die Entschlüsselung. Der
// Kontakt wäre damit beim ersten Öffnen eines Bestandsmandanten verschwunden —
// aus der Kontaktliste und aus dem Rechnungsdialog.
func TestBackfillContactAddressesKeepsFieldsEncrypted(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}

	_, vault, err := security.NewKeyfile("Testkennwort-1234")
	if err != nil {
		t.Fatalf("Schlüsselbund: %v", err)
	}
	SetActiveVault(vault)
	t.Cleanup(func() { SetActiveVault(nil) })

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

	var migrated domain.Contact
	if err := db.First(&migrated, contact.ID).Error; err != nil {
		t.Fatalf("der migrierte Kontakt ist nicht mehr lesbar: %v", err)
	}
	if migrated.Street != "Kundenweg 2" || migrated.PostalCode != "10115" || migrated.City != "Berlin" {
		t.Errorf("Anschrift = %q / %q / %q, erwartet \"Kundenweg 2\" / \"10115\" / \"Berlin\"",
			migrated.Street, migrated.PostalCode, migrated.City)
	}

	// Und in der Spalte steht Geheimtext: sonst hätte der Lesetest oben nur
	// deshalb gestimmt, weil gar nicht verschlüsselt wurde.
	var raw struct {
		Street     string
		PostalCode string
		City       string
	}
	if err := db.Raw("SELECT street, postal_code, city FROM contacts WHERE id = ?", contact.ID).
		Scan(&raw).Error; err != nil {
		t.Fatalf("Rohspalten lesen: %v", err)
	}
	for name, value := range map[string]string{"street": raw.Street, "postal_code": raw.PostalCode, "city": raw.City} {
		if value == "" {
			t.Errorf("die Spalte %s ist leer", name)
		}
	}
	if strings.Contains(raw.Street, "Kundenweg") || raw.PostalCode == "10115" || raw.City == "Berlin" {
		t.Errorf("die Anschrift steht im Klartext in der Datenbank: %+v", raw)
	}
}

// Ohne Schlüsselbund bleibt der Klartextfall unverändert: derselbe Lauf muss
// auch den unverschlüsselten Mandanten migrieren.
func TestBackfillContactAddressesWithoutVault(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	SetActiveVault(nil)

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
