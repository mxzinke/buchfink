package wailsbridge

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// newBridge baut eine Bridge ohne Wails-Anwendung und ohne offenen Mandanten,
// mit eigener Konfiguration im Testordner.
func newBridge(t *testing.T, opts ...func(*core.HostOptions)) *BuchfinkBridge {
	t.Helper()
	base := t.TempDir()
	o := core.HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base,
		Network: core.OfflineNetwork(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	host, err := core.NewHost(context.Background(), o)
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(context.Background()) })
	return &BuchfinkBridge{host: host}
}

// testBridge ist eine Bridge mit einem frisch angelegten Mandanten für 2026.
func testBridge(t *testing.T, opts ...func(*core.HostOptions)) *BuchfinkBridge {
	t.Helper()
	b := newBridge(t, opts...)
	if _, err := b.createTenant("Pfennig Ventures GmbH", filepath.Join(t.TempDir(), "mandant"),
		domain.CompanySettings{CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026}); err != nil {
		t.Fatalf("Mandant anlegen: %v", err)
	}
	return b
}

// testWrite schreibt im offenen Mandanten, wie es ein Bridge-Aufruf tut.
func testWrite(t *testing.T, b *BuchfinkBridge, fn func(ctx context.Context, s *core.Services) error) {
	t.Helper()
	if err := b.change(fn); err != nil {
		t.Fatalf("Schreibvorgang: %v", err)
	}
}

// testRead liest im offenen Mandanten.
func testRead(t *testing.T, b *BuchfinkBridge, fn func(ctx context.Context, s *core.Services) error) {
	t.Helper()
	if _, err := read(b, func(ctx context.Context, s *core.Services) (struct{}, error) {
		return struct{}{}, fn(ctx, s)
	}); err != nil {
		t.Fatalf("Lesevorgang: %v", err)
	}
}

// fileTestReceipt legt einen Eingangsbeleg über denselben Weg wie im Betrieb ab.
func fileTestReceipt(t *testing.T, b *BuchfinkBridge) *domain.Receipt {
	t.Helper()
	var receipt *domain.Receipt
	testWrite(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		receipt, err = s.Receipts.File(ctx, service.FileReceiptRequest{
			Direction: domain.DirectionIncoming, FiscalYear: 2026,
			Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
			IssuerName: "Büromarkt GmbH", GrossAmount: 10000,
			Files: []service.NewFile{{
				Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf",
				Content: []byte("%PDF-1.4 Testbeleg " + t.Name()),
			}},
		})
		return err
	})
	return receipt
}

// manualEntryRequest legt einen Beleg an und liefert die Handbuchung darauf.
// Es gibt keine Buchung ohne Beleg.
// postManualEntry bucht einen Buchungssatz über denselben Schreibweg wie jede
// Bridge-Methode. Die Oberfläche bucht nicht frei; die Tests brauchen Buchungen.
func postManualEntry(b *BuchfinkBridge, req service.ManualEntryRequest) (*domain.JournalEntry, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Posting.PostManualEntry(ctx, req)
	})
}

func manualEntryRequest(t *testing.T, b *BuchfinkBridge) service.ManualEntryRequest {
	t.Helper()
	return service.ManualEntryRequest{Entry: manualEntry(), ReceiptID: fileTestReceipt(t, b).ID}
}

func manualEntry() domain.JournalEntry {
	return domain.JournalEntry{
		BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		Description: "Testbuchung", Source: domain.EntrySourceManual,
		// Die Anschaffung geringwertiger Wirtschaftsgüter gegen Bank ist hier
		// kein steuerbarer Umsatz.
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
}

// auditEntries liest das Protokoll mit einem Filter.
func auditEntries(t *testing.T, b *BuchfinkBridge, filter domain.AuditFilter) []domain.AuditLogEntry {
	t.Helper()
	var out []domain.AuditLogEntry
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		var err error
		out, err = s.Repos.Audit.FindFiltered(ctx, 0, filter)
		return err
	})
	return out
}
