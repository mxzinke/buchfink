package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Kopfdaten vor dem Schreiben, nicht erst beim Versiegeln ---------------

// Das Beschlussdokument der Ergebnisverwendung bekommt die Buchung erst hinter
// dem Journal-Commit (Seal). Fehlen seine Kopfdaten, muss der Vorgang vorher
// scheitern — sonst stünde die Buchung im Journal, während das Dokument
// unverbunden im Belegspeicher liegt und der Prüflauf es dauerhaft als
// ungebucht meldet.
func TestAppropriationDecisionWithoutHeaderLeavesNoBooking(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}
	m.adopt(t, env, 2026, "2027-05-20")
	before, _ := env.journalRepo.FindAll(ctx, 2027)

	decision := filePDFReceipt(t, env, FileReceiptRequest{
		FiscalYear: 2027, Kind: domain.ReceiptKindOther, DocumentDate: "2027-05-20",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "beschluss.pdf",
			Content: []byte("%PDF-1.4 Beschluss\n"),
		}},
	})

	_, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Beschluss",
		ReceiptID: decision.ID,
	})
	if err == nil {
		t.Fatal("ein Beschlussdokument ohne Kopfdaten darf die Buchung nicht stützen")
	}
	if !strings.Contains(err.Error(), "Beschlussdokument") {
		t.Errorf("die Meldung muss sagen, welches Dokument fehlt: %v", err)
	}

	after, _ := env.journalRepo.FindAll(ctx, 2027)
	if len(after) != len(before) {
		t.Errorf("der abgewiesene Beschluss hat %d Buchung(en) hinterlassen",
			len(after)-len(before))
	}
}
