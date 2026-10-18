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

// --- Migrationsprotokoll --------------------------------------------------

func TestFirstOpenLogsAMigration(t *testing.T) {
	db, err := rawInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	ctx := context.Background()

	record, err := ApplyMigrations(ctx, db)
	if err != nil {
		t.Fatalf("die erste Migration ist fehlgeschlagen: %v", err)
	}
	if record == nil {
		t.Fatal("die erste Öffnung muss protokolliert werden")
	}
	if record.FromVersion != 0 || record.ToVersion != SchemaVersion {
		t.Errorf("Version %d → %d, erwartet 0 → %d", record.FromVersion, record.ToVersion, SchemaVersion)
	}
	if record.Result != domain.SchemaMigrationResultOK {
		t.Errorf("Ergebnis %q, erwartet %q", record.Result, domain.SchemaMigrationResultOK)
	}
	if record.AppVersion == "" || record.Tables == "" {
		t.Error("zum Protokolleintrag gehören Programmfassung und betroffene Tabellen")
	}
}

func TestSecondOpenWithoutChangeLogsNothing(t *testing.T) {
	db, err := rawInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	ctx := context.Background()

	if _, err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("die erste Migration ist fehlgeschlagen: %v", err)
	}
	second, err := ApplyMigrations(ctx, db)
	if err != nil {
		t.Fatalf("die zweite Öffnung ist fehlgeschlagen: %v", err)
	}
	// Ohne diese Bedingung protokollierte jeder Start eine Migration, und das
	// Protokoll sagte nichts mehr aus.
	if second != nil {
		t.Errorf("eine Öffnung ohne Schemaänderung darf nichts protokollieren, bekommen %+v", second)
	}

	records, err := NewMigrationRepository(db).FindSchemaMigrations(ctx)
	if err != nil {
		t.Fatalf("das Protokoll konnte nicht gelesen werden: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("%d Protokolleinträge, erwartet 1", len(records))
	}
}

func TestSeedAddsMissingAccountsAndNeverDeletes(t *testing.T) {
	db := welle6DB(t)
	ctx := context.Background()

	// Ein Konto, das es im Kontenrahmen nicht gibt, und ein umbenanntes aus dem
	// Kontenrahmen: beides muss den Seed überleben. Vorher warf ein
	// `DELETE FROM accounts` bei weniger als hundert Konten alles weg.
	own := domain.Account{Number: "9999", Name: "Eigenes Konto", Type: domain.AccountTypeStatistical, Category: "Eigene"}
	renamed := domain.Account{Number: "1800", Name: "Geschäftskonto Sparkasse", Type: domain.AccountTypeAsset, Category: "Umlaufvermögen"}
	if err := db.Create(&own).Error; err != nil {
		t.Fatalf("eigenes Konto konnte nicht angelegt werden: %v", err)
	}
	if err := db.Create(&renamed).Error; err != nil {
		t.Fatalf("umbenanntes Konto konnte nicht angelegt werden: %v", err)
	}

	if err := SeedDefaultsIfEmpty(ctx, db, 2026); err != nil {
		t.Fatalf("der Seed ist fehlgeschlagen: %v", err)
	}

	var keptOwn domain.Account
	if err := db.Where("number = ?", "9999").First(&keptOwn).Error; err != nil {
		t.Fatalf("das eigene Konto wurde gelöscht: %v", err)
	}
	var keptRenamed domain.Account
	if err := db.Where("number = ?", "1800").First(&keptRenamed).Error; err != nil {
		t.Fatalf("das umbenannte Konto wurde gelöscht: %v", err)
	}
	if keptRenamed.Name != "Geschäftskonto Sparkasse" {
		t.Errorf("der geänderte Kontoname wurde überschrieben: %q", keptRenamed.Name)
	}

	var count int64
	db.Model(&domain.Account{}).Count(&count)
	if count < 100 {
		t.Errorf("die fehlenden Konten des Kontenrahmens wurden nicht ergänzt: %d Konten", count)
	}

	// Der Lauf steht im Änderungsprotokoll — sonst wäre nicht feststellbar,
	// wann der Kontenplan ergänzt wurde.
	entries, _ := NewAuditRepository(db).FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "ACCOUNT"})
	if len(entries) == 0 {
		t.Error("das Ergänzen des Kontenplans muss protokolliert werden")
	}
}
