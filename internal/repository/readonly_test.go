package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

func openWithReader(t *testing.T) (*gormPair, context.Context) {
	t.Helper()
	dir := t.TempDir()
	db, err := InitTenantDB(dir)
	if err != nil {
		t.Fatalf("Mandantendatenbank: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	reader, err := OpenTenantReader(dir)
	if err != nil {
		t.Fatalf("Leser: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(reader) })
	return &gormPair{db: db, reader: reader}, context.Background()
}

// Ein Lesevorgang, der schreiben will, scheitert an der Verbindung selbst.
func TestReadOnlyTransactionRefusesWrites(t *testing.T) {
	p, ctx := openWithReader(t)
	settings := NewSettingsRepository(p.db)

	err := RunReadOnly(ctx, p.reader, func(ctx context.Context) error {
		if !IsReadOnly(ctx) || !InTx(ctx) {
			t.Error("der Kontext kennt die lesende Transaktion nicht")
		}
		if _, err := settings.Get(ctx, "company_name"); err != nil {
			t.Errorf("Lesen in der lesenden Transaktion: %v", err)
		}
		return settings.Set(ctx, "company_name", "Geändert GmbH")
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "readonly database") {
		t.Fatalf("Schreiben in der lesenden Transaktion muss scheitern, Fehler: %v", err)
	}
	if name, _ := settings.Get(ctx, "company_name"); name == "Geändert GmbH" {
		t.Error("der Schreibversuch ist in der Datei angekommen")
	}
}

// Der Zugriff wird auch beim Lesen protokolliert (QUE-02): der Eintrag geht
// über die Verbindung des Protokolls und steht danach in der Kette.
func TestAuditAppendEscapesTheReadOnlyTransaction(t *testing.T) {
	p, ctx := openWithReader(t)
	audit := NewAuditRepository(p.db)

	err := RunReadOnly(ctx, p.reader, func(ctx context.Context) error {
		return audit.Log(ctx, domain.AuditActionExport, "JOURNAL", "2026", "Journal herausgegeben")
	})
	if err != nil {
		t.Fatalf("Protokolleintrag aus einem Lesevorgang: %v", err)
	}
	entries, err := audit.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "JOURNAL"})
	if err != nil || len(entries) != 1 {
		t.Fatalf("Eintrag nicht gefunden: %v (%d)", err, len(entries))
	}
}

type gormPair struct{ db, reader *gorm.DB }
