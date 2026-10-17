package wailsbridge

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
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
