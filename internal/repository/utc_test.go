package repository

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

func TestUTCWritesOrderTheRepeatedHour(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	defer CloseDB(db)
	repo := NewCheckRunRepository(db)
	early, _ := time.Parse(time.RFC3339, "2026-10-25T02:45:00+02:00")
	late, _ := time.Parse(time.RFC3339, "2026-10-25T02:15:00+01:00")
	for _, at := range []time.Time{late, early} {
		row := &domain.CheckRun{FiscalYear: 2026, CreatedAt: at}
		if err := repo.Create(context.Background(), row); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := repo.Latest(context.Background(), 2026)
	if err != nil || latest == nil || !latest.CreatedAt.Equal(late) {
		t.Fatalf("neuester Prüflauf: %+v, %v", latest, err)
	}
	if _, offset := latest.CreatedAt.Zone(); offset != 0 {
		t.Fatalf("Zonenversatz: %d", offset)
	}
}

func TestUTCWritesPreserveHashesAndPointers(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	defer CloseDB(db)
	at, _ := time.Parse(time.RFC3339Nano, "2026-10-25T02:15:00.123456789+01:00")
	journal := domain.JournalEntry{CreatedAt: at, CommittedAt: &at, Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1000000}
	hash := accounting.NewHashChain().CalculateHash(&journal, domain.GenesisHash)
	if err := db.Create(&journal).Error; err != nil {
		t.Fatal(err)
	}
	var got domain.JournalEntry
	if err := db.First(&got, journal.ID).Error; err != nil {
		t.Fatal(err)
	}
	if accounting.NewHashChain().CalculateHash(&got, domain.GenesisHash) != hash {
		t.Fatal("Journalhash verändert")
	}
	if got.CommittedAt == nil || !got.CommittedAt.Equal(at) {
		t.Fatal("Festschreibungszeitpunkt verändert")
	}
	audit := domain.AuditLogEntry{Timestamp: at}
	auditHash := accounting.NewAuditChain().CalculateHash(&audit, domain.GenesisHash)
	if err := db.Create(&audit).Error; err != nil {
		t.Fatal(err)
	}
	var loaded domain.AuditLogEntry
	if err := db.First(&loaded, audit.ID).Error; err != nil {
		t.Fatal(err)
	}
	if accounting.NewAuditChain().CalculateHash(&loaded, domain.GenesisHash) != auditHash {
		t.Fatal("Audithash verändert")
	}
	if err := db.Model(&journal).Updates(map[string]any{"committed_at": at}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&got, journal.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, offset := got.CommittedAt.Zone(); offset != 0 {
		t.Fatalf("Map-Update mit Zonenversatz %d", offset)
	}
}

func TestAuditDateFilterUsesGermanMidnight(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	defer CloseDB(db)
	for _, iso := range []string{"2026-06-30T21:59:59Z", "2026-06-30T22:00:00Z", "2026-07-01T21:59:59Z", "2026-07-01T22:00:00Z"} {
		at, _ := time.Parse(time.RFC3339, iso)
		if err := db.Create(&domain.AuditLogEntry{Timestamp: at}).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows, err := NewAuditRepository(db).FindFiltered(context.Background(), 0, domain.AuditFilter{From: "2026-07-01", To: "2026-07-01"})
	if err != nil || len(rows) != 2 || rows[0].ID != 3 || rows[1].ID != 2 {
		t.Fatalf("deutscher 1. Juli: %+v, %v", rows, err)
	}
}
