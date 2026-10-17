package wailsbridge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Herausgabe einer einzelnen Belegdatei steht als Zugriff im Protokoll
// (QUE-02 K2).
//
// Die Verfahrensdokumentation sagt es zu (Abschnitt 6.3), und der Weg über die
// gefilterte Journalmenge tut es längst. Der Dateiweg war die Lücke: er gab ein
// Dokument mit personenbezogenen Daten heraus, ohne eine Spur zu hinterlassen.
func TestReleasingASingleReceiptFileIsLogged(t *testing.T) {
	b := testBridge(t)
	receipt := fileTestReceipt(t, b)

	var content *service.FileContent
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		content, err = s.Receipts.Content(ctx, receipt.ID, receipt.Files[0].ID)
		return err
	})
	target := filepath.Join(t.TempDir(), "rechnung.pdf")
	if err := b.writeReceiptFileTo(receipt.ID, content, target); err != nil {
		t.Fatalf("Belegdatei herausgeben: %v", err)
	}

	entries := auditEntries(t, b, domain.AuditFilter{Access: true})
	found := false
	for _, e := range entries {
		if e.EntityType == "RECEIPT" && strings.Contains(e.Details, "rechnung.pdf") {
			found = true
		}
	}
	if !found {
		t.Errorf("die Herausgabe der Belegdatei fehlt im Protokoll: %+v", entries)
	}
}

// Ohne Protokolleintrag wird nichts herausgegeben: die Datei liegt erst nach
// dem Eintrag unter ihrem Namen.
func TestNoFileWithoutItsAccessLog(t *testing.T) {
	b := testBridge(t)
	receipt := fileTestReceipt(t, b)
	var content *service.FileContent
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		content, err = s.Receipts.Content(ctx, receipt.ID, receipt.Files[0].ID)
		return err
	})
	db, err := repository.InitTenantDB(b.host.ActiveTenant().DataDir)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`CREATE TRIGGER kein_protokoll BEFORE INSERT ON audit_log_entries
		BEGIN SELECT RAISE(ABORT, 'Protokoll nicht verfügbar'); END`).Error
	_ = repository.CloseDB(db)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := b.writeReceiptFileTo(receipt.ID, content, filepath.Join(dir, "rechnung.pdf")); err == nil {
		t.Fatal("herausgegeben ohne Protokolleintrag")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("im Zielordner liegt %s", entries[0].Name())
	}
}
