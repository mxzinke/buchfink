package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// rawInMemoryDB öffnet eine leere Datenbank ohne Schema.
//
// InitInMemoryDB migriert bereits; die Migrationstests brauchen aber genau den
// Zustand davor — eine Datei, die noch keine Schemaversion hat.
func rawInMemoryDB() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func welle6DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	// Die Datenbank im Arbeitsspeicher lebt, solange ihre Verbindung offen ist.
	t.Cleanup(func() { _ = CloseDB(db) })
	return db
}

// --- Änderungsprotokoll ---------------------------------------------------

func TestAuditLogChainsEntriesOnWrite(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Log(ctx, domain.AuditActionUpdate, "CONTACT", "7", "geändert"); err != nil {
			t.Fatalf("Protokolleintrag konnte nicht geschrieben werden: %v", err)
		}
	}

	entries, err := repo.FindAllAscending(ctx)
	if err != nil {
		t.Fatalf("Protokoll konnte nicht gelesen werden: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("%d Einträge, erwartet 3", len(entries))
	}

	if entries[0].PreviousHash != domain.GenesisHash {
		t.Errorf("der erste Eintrag muss am Genesis-Hash hängen, hat %q", entries[0].PreviousHash)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].PreviousHash != entries[i-1].EntryHash {
			t.Errorf("Eintrag %d hängt nicht an seinem Vorgänger", entries[i].ID)
		}
	}

	result := accounting.NewAuditChain().Verify(entries)
	if !result.IsValid {
		t.Errorf("die frisch geschriebene Kette muss gültig sein: %s", result.Message)
	}
}

func TestAuditLogDetectsTamperingInTheDatabase(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	_ = repo.Log(ctx, domain.AuditActionCreate, "JOURNAL", "1", "Buchung 2026-000001 erfasst")
	_ = repo.Log(ctx, domain.AuditActionUpdate, "JOURNAL", "1", "Betrag von 100 € auf 200 € geändert")
	_ = repo.Log(ctx, domain.AuditActionExport, "JOURNAL", "", "Journal exportiert")

	// Der Eingriff, gegen den die Kette schützt: jemand ändert die
	// Protokollzeile an der Datenbank vorbei am Programm.
	if err := db.Exec("UPDATE audit_log_entries SET details = ? WHERE id = 2", "nichts passiert").Error; err != nil {
		t.Fatalf("der Testeingriff ist fehlgeschlagen: %v", err)
	}

	entries, _ := repo.FindAllAscending(ctx)
	result := accounting.NewAuditChain().Verify(entries)
	if result.IsValid {
		t.Fatal("eine an der Datenbank geänderte Protokollzeile muss auffallen")
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != 2 {
		t.Errorf("der Bruch muss an Eintrag 2 gemeldet werden, bekommen %v", result.FirstBrokenID)
	}
}

func TestAuditLogStoresUTCActorAndVersion(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	before := time.Now().UTC().Add(-time.Second)
	if err := repo.Log(ctx, domain.AuditActionUpdate, "SETTINGS", "COMPANY", "geändert"); err != nil {
		t.Fatalf("Protokolleintrag konnte nicht geschrieben werden: %v", err)
	}

	entries, _ := repo.FindAll(ctx, 1)
	if len(entries) != 1 {
		t.Fatalf("%d Einträge, erwartet 1", len(entries))
	}
	entry := entries[0]

	// UTC: eine Ortszeit ohne Zone ist in der Nacht der Zeitumstellung
	// mehrdeutig, und die Reihenfolge des Protokolls richtet sich nach ihr.
	if _, offset := entry.Timestamp.Zone(); offset != 0 {
		t.Errorf("der Zeitstempel steht nicht in UTC (Versatz %d Sekunden)", offset)
	}
	if entry.Timestamp.Before(before) {
		t.Errorf("der Zeitstempel %s liegt vor dem Schreiben", entry.Timestamp)
	}
	if entry.Actor == "" {
		t.Error("jeder Protokolleintrag braucht eine Bearbeiterkennung (UNV-04)")
	}
	if entry.AppVersion == "" {
		t.Error("jeder Protokolleintrag braucht die Programmfassung (UNV-06)")
	}
}

func TestLogChangeStoresOnlyChangedFields(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	before := &domain.Contact{ID: 7, Name: "Meier GmbH", Street: "Hauptstraße 1", LedgerAccount: "10001"}
	after := *before
	after.Street = "Nebenstraße 5"

	if err := repo.LogChange(ctx, domain.AuditActionUpdate, "CONTACT", "7",
		"Anschrift geändert", before, &after); err != nil {
		t.Fatalf("Änderung konnte nicht protokolliert werden: %v", err)
	}

	entries, _ := repo.FindAll(ctx, 1)
	entry := entries[0]
	if !strings.Contains(entry.Before, "Hauptstraße 1") {
		t.Errorf("das Vorher enthält die alte Anschrift nicht: %q", entry.Before)
	}
	if !strings.Contains(entry.After, "Nebenstraße 5") {
		t.Errorf("das Nachher enthält die neue Anschrift nicht: %q", entry.After)
	}
	if strings.Contains(entry.After, "Meier GmbH") {
		t.Errorf("der unveränderte Name gehört nicht ins Protokoll: %q", entry.After)
	}
}

func TestAuditFilterNarrowsTheLog(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	_ = repo.Log(ctx, domain.AuditActionCreate, "CONTACT", "7", "angelegt")
	_ = repo.Log(ctx, domain.AuditActionUpdate, "CONTACT", "8", "geändert")
	_ = repo.Log(ctx, domain.AuditActionExport, "JOURNAL", "", "exportiert")

	byType, err := repo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "CONTACT"})
	if err != nil {
		t.Fatalf("gefiltertes Lesen ist fehlgeschlagen: %v", err)
	}
	if len(byType) != 2 {
		t.Errorf("%d Einträge zur Objektart CONTACT, erwartet 2", len(byType))
	}

	byID, _ := repo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "CONTACT", EntityID: "8"})
	if len(byID) != 1 {
		t.Errorf("%d Einträge zu Kontakt 8, erwartet 1", len(byID))
	}

	byAction, _ := repo.FindFiltered(ctx, 0, domain.AuditFilter{Action: domain.AuditActionExport})
	if len(byAction) != 1 {
		t.Errorf("%d Exporte, erwartet 1", len(byAction))
	}

	today := time.Now().UTC().Format("2006-01-02")
	byDay, _ := repo.FindFiltered(ctx, 0, domain.AuditFilter{From: today, To: today})
	if len(byDay) != 3 {
		t.Errorf("%d Einträge von heute, erwartet 3", len(byDay))
	}
}
