package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/security"
)

func TestAddedEncryptedFieldsRejectWrongVaultAndTampering(t *testing.T) {
	_, vault, err := security.NewKeyfile("field-encryption-example")
	if err != nil {
		t.Fatal(err)
	}
	SetActiveVault(vault)
	t.Cleanup(func() { SetActiveVault(nil) })
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	defer CloseDB(db)
	c := domain.Contact{Name: "Geschützter Name", Type: domain.ContactTypeCustomer}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	var read domain.Contact
	if err := db.WithContext(WithVault(context.Background(), nil)).First(&read, c.ID).Error; err == nil {
		t.Fatal("encrypted name readable without a vault")
	}
	_, wrong, err := security.NewKeyfile("wrong-example-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(WithVault(context.Background(), wrong)).First(&read, c.ID).Error; err == nil {
		t.Fatal("wrong vault accepted")
	}
	if err := db.Exec("UPDATE contacts SET name = ? WHERE id = ?", addedEncryptionPrefix+"damaged", c.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&read, c.ID).Error; err == nil {
		t.Fatal("damaged envelope accepted")
	}
}
