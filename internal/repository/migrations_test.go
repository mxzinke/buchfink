package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Eine neue Datenbank entsteht vollständig in der aktuellen Schemaversion, und
// das Protokoll nennt ihre Tabellen.
func TestFreshDatabaseIsCreatedInTheCurrentSchema(t *testing.T) {
	db, err := rawInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })

	record, err := ApplyMigrations(context.Background(), db)
	if err != nil {
		t.Fatalf("Schema anlegen: %v", err)
	}
	if record == nil || record.FromVersion != 0 || record.ToVersion != SchemaVersion {
		t.Fatalf("Protokolleintrag %+v, erwartet 0 → %d", record, SchemaVersion)
	}
	for _, table := range []string{
		"journal_entries", "journal_lines", "audit_log_entries", "fiscal_years",
		"receipts", "setting_items", "schema_migrations",
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("die Tabelle %s fehlt", table)
		}
		if !strings.Contains(record.Tables, table) {
			t.Errorf("das Protokoll nennt die Tabelle %s nicht: %q", table, record.Tables)
		}
	}
}

// Eine Datei aus einer neueren Programmfassung wird nicht geöffnet: ihr Schema
// kennt dieser Code nicht.
func TestNewerSchemaIsRefused(t *testing.T) {
	db, err := rawInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	ctx := context.Background()

	if _, err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("Schema anlegen: %v", err)
	}
	if err := db.Create(&domain.SchemaMigration{
		ToVersion: SchemaVersion + 1, Result: domain.SchemaMigrationResultOK,
	}).Error; err != nil {
		t.Fatalf("Version eintragen: %v", err)
	}
	if _, err := ApplyMigrations(ctx, db); err == nil {
		t.Error("eine Datei mit neuerem Schema darf nicht geöffnet werden")
	}
}
