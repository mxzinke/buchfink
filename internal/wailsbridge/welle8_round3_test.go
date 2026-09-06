package wailsbridge

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

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
	ctx := context.Background()

	receipt, err := b.receiptSvc.File(ctx, service.FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: 2026,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 10000,
		Files: []service.NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf",
			Content: []byte("%PDF-1.4 Testbeleg"),
		}},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}

	content, err := b.receiptSvc.Content(ctx, receipt.ID, receipt.Files[0].ID)
	if err != nil {
		t.Fatalf("Belegdatei lesen: %v", err)
	}
	target := filepath.Join(t.TempDir(), "rechnung.pdf")
	if err := b.writeReceiptFileTo(receipt.ID, content, target); err != nil {
		t.Fatalf("Belegdatei herausgeben: %v", err)
	}

	entries, err := b.auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{Access: true})
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
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
