package repository

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/security"
)

func TestAddedEncryptionMigratesLegacyFieldsAndRemovesActivePlaintext(t *testing.T) {
	SetActiveVault(nil)
	t.Cleanup(func() { SetActiveVault(nil) })
	dir := t.TempDir()
	db, err := InitTenantDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	contact := domain.Contact{Name: "MIGRATION_PERSON_8E781", Type: domain.ContactTypeCustomer}
	bank := domain.BankTransaction{CounterpartyName: "MIGRATION_BANK_8E781", Amount: 123}
	item := domain.InvoiceItem{Description: "MIGRATION_POSITION_8E781"}
	for _, record := range []any{&contact, &bank, &item} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("UPDATE accounts SET name = ?, description = ? WHERE number = ?", "MIGRATION_ACCOUNT_8E781", "MIGRATION_DESCRIPTION_8E781", "1800").Error; err != nil {
		t.Fatal(err)
	}
	audit := NewAuditRepository(db)
	if err := audit.Log(context.Background(), domain.AuditActionCreate, "CONTACT", "1", "MIGRATION_AUDIT_8E781"); err != nil {
		t.Fatal(err)
	}
	beforeAudit, err := audit.FindAllAscending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the previous schema with its existing unencrypted fields.
	if err := db.Exec("DELETE FROM schema_migrations").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.SchemaMigration{ToVersion: 8, Result: domain.SchemaMigrationResultOK}).Error; err != nil {
		t.Fatal(err)
	}
	if err := CloseDB(db); err != nil {
		t.Fatal(err)
	}
	_, vault, err := security.NewKeyfile("migration-example-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	SetActiveVault(vault)
	db, err = InitTenantDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	var read domain.Contact
	if err := db.First(&read, contact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if read.Name != contact.Name || !read.CreatedAt.Equal(contact.CreatedAt) || !read.UpdatedAt.Equal(contact.UpdatedAt) {
		t.Fatalf("migration altered content or timestamps: %+v", read)
	}
	afterAudit, err := NewAuditRepository(db).FindAllAscending(context.Background())
	if err != nil || len(afterAudit) != len(beforeAudit) {
		t.Fatalf("migrated audit: %+v, %v", afterAudit, err)
	}
	for i := range beforeAudit {
		if afterAudit[i] != beforeAudit[i] {
			t.Fatalf("migration altered audit entry: %+v != %+v", afterAudit[i], beforeAudit[i])
		}
	}
	if result := accounting.NewAuditChain().Verify(afterAudit); !result.IsValid {
		t.Fatalf("audit chain broken by encryption: %+v", result)
	}
	var stored string
	if err := db.Raw("SELECT name FROM contacts WHERE id = ?", contact.ID).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, addedEncryptionPrefix) {
		t.Fatal("contact remains plaintext")
	}
	if _, err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var again string
	db.Raw("SELECT name FROM contacts WHERE id = ?", contact.ID).Scan(&again)
	if stored != again {
		t.Fatal("reopening encrypted the field twice")
	}
	if err := CloseDB(db); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		data, err := os.ReadFile(filepath.Join(dir, "buchfink.sqlite"+suffix))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("MIGRATION_")) {
			t.Errorf("legacy plaintext remains in active SQLite file %s", suffix)
		}
	}
}

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
