package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Belegnummer entsteht nach der eingestellten Systematik (BEL-02 K4).
//
// Geprüft wird an der Vergabe und nicht am Formatierer: das Format entscheidet
// über die Nummer, die in der Datenbank landet, und genau dort war es bisher
// eine Konstante.
func TestReceiptNumberFollowsTheConfiguredFormat(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	settings := NewSettingsRepository(db)
	receipts := NewReceiptRepository(db)

	file := func() []domain.ReceiptFile {
		return []domain.ReceiptFile{{
			Position: 1, Role: domain.ReceiptRoleOriginal, FileName: "beleg.pdf",
			MimeType: "application/pdf", Size: 17,
			SHA256:     "0000000000000000000000000000000000000000000000000000000000000001",
			StoredPath: "2026/eingang/00/0001.pdf",
		}}
	}
	newReceipt := func() *domain.Receipt {
		return &domain.Receipt{
			FiscalYear: 2026, Direction: domain.DirectionIncoming,
			Kind: domain.ReceiptKindInvoice, Status: domain.ReceiptStatusFiled,
			DocumentDate: "2026-03-01", IssuerName: "Büromarkt GmbH", GrossAmount: 11900,
			Files: file(),
		}
	}

	// Ohne Einstellung: die Voreinstellung.
	first := newReceipt()
	if err := receipts.Create(ctx, first, accounting.ReceiptHash); err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	if first.ReceiptNumber != "ER-2026-0001" {
		t.Errorf("Belegnummer %q — erwartet die Voreinstellung", first.ReceiptNumber)
	}

	// Mit eigener Systematik: sie gilt für die nächste Nummer, die vergebenen
	// bleiben stehen.
	if err := settings.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
		ReceiptNumberFormat: "BE-{JAHR}-{NR:5}",
	}); err != nil {
		t.Fatalf("Einstellung speichern: %v", err)
	}
	second := newReceipt()
	second.Files[0].SHA256 = "0000000000000000000000000000000000000000000000000000000000000002"
	if err := receipts.Create(ctx, second, accounting.ReceiptHash); err != nil {
		t.Fatalf("zweiter Beleg: %v", err)
	}
	if second.ReceiptNumber != "BE-2026-00002" {
		t.Errorf("Belegnummer %q — erwartet BE-2026-00002 nach der eigenen Systematik",
			second.ReceiptNumber)
	}
	if stored, err := receipts.FindByID(ctx, first.ID); err != nil {
		t.Fatalf("ersten Beleg lesen: %v", err)
	} else if stored.ReceiptNumber != "ER-2026-0001" {
		t.Errorf("die vergebene Nummer %q hat sich geändert", stored.ReceiptNumber)
	}

	// Ein untaugliches Format wird beim Speichern abgewiesen und nicht
	// stillschweigend ersetzt.
	err = settings.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
		ReceiptNumberFormat: "BE-{JAHR}",
	})
	if err == nil {
		t.Error("ein Format ohne Zähler gäbe jedem Beleg dieselbe Nummer")
	}
}
