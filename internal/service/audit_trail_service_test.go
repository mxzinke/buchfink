package service

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Prüfpfad -------------------------------------------------------------

// Der Prüfpfad führt vom Beleg über die Buchung und die Zahlung zum Bankumsatz.
func TestAuditTrailLeadsFromReceiptToBankTransaction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100_000, domain.TaxRateStandard)

	txID := env.bankLine(t, "2026-03-20", -119_000, "Lieferant GmbH", "Rechnung")
	if _, err := env.payments(t).Settle(ctx, PaymentRequest{
		BankTxID:       &txID,
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations: []AllocationRequest{
			{OpenItemEntryID: invoice.ID, SettledAmount: 119_000},
		},
	}); err != nil {
		t.Fatalf("Zahlung buchen: %v", err)
	}

	receipt, err := env.receiptRepo.FindByJournalEntry(ctx, invoice.ID)
	if err != nil || receipt == nil {
		t.Fatalf("Beleg zur Buchung: %v", err)
	}

	svc := NewAuditTrailService(
		env.receiptRepo, env.journalRepo, repository.NewPaymentAllocationRepository(env.db),
		repository.NewBankRepository(env.db), repository.NewAuditRepository(env.db))

	trail, err := svc.Trail(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Prüfpfad: %v", err)
	}
	stages := make([]AuditTrailStage, 0, len(trail.Steps))
	for _, step := range trail.Steps {
		stages = append(stages, step.Stage)
	}
	want := []AuditTrailStage{TrailStageReceipt, TrailStageBooking, TrailStagePayment, TrailStageBank}
	if len(stages) != len(want) {
		t.Fatalf("Stufen = %v, erwartet %v", stages, want)
	}
	for i := range want {
		if stages[i] != want[i] {
			t.Errorf("Stufe %d = %q, erwartet %q", i+1, stages[i], want[i])
		}
	}

	// Und als CSV: dieselbe Kette in einer Datei.
	path := filepath.Join(t.TempDir(), "pruefpfad.csv")
	if _, err := svc.ExportCSV(ctx, receipt.ID, path); err != nil {
		t.Fatalf("Prüfpfad als CSV: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("CSV lesen: %v", err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV zerlegen: %v", err)
	}
	if len(rows) != len(want)+1 {
		t.Errorf("CSV hat %d Zeilen, erwartet %d samt Kopfzeile", len(rows), len(want)+1)
	}
}

// Ein Beleg ohne Buchung endet nach der ersten Stufe — und sagt das.
func TestAuditTrailStopsAtAnUnbookedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.fileIncoming(t, "rechnung.pdf")

	svc := NewAuditTrailService(
		env.receiptRepo, env.journalRepo, repository.NewPaymentAllocationRepository(env.db),
		repository.NewBankRepository(env.db), nil)

	trail, err := svc.Trail(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Prüfpfad: %v", err)
	}
	if len(trail.Steps) != 1 || trail.Steps[0].Stage != TrailStageReceipt {
		t.Fatalf("erwartet nur die Belegstufe, erhalten %+v", trail.Steps)
	}
	if !strings.Contains(trail.Note, "noch nicht gebucht") {
		t.Errorf("der Hinweis sagt nicht, wo der Pfad endet: %q", trail.Note)
	}
}
