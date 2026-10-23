package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Vertrag eines Gesellschafterdarlehens belegt den Vertrag, keine Buchung.
// Mit dem Darlehen ist er zugeordnet und gilt nicht mehr als ungebucht; nennt
// eine Berichtigung einen anderen Vertrag, ist der bisherige wieder abgelegt.
func TestLoanContractIsAssignedNotBooked(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	mara := env.person(t, "Mara Beispiel")
	svc := env.shareholders(t)
	contract := env.fileIncoming(t, "darlehensvertrag.pdf")

	req := LoanRequest{ContactID: mara.ID, Direction: domain.LoanToCompany, Principal: 1_500_000,
		InterestRateBP: 400, ContractDate: "2026-06-26", StartDate: "2026-07-01", MaturityDate: "2029-06-30",
		ContractReceiptID: contract.ID}
	view, err := svc.RegisterLoan(ctx, req)
	if err != nil {
		t.Fatalf("Darlehen erfassen: %v", err)
	}
	assigned, err := env.receipts.Get(ctx, contract.ID)
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Status != domain.ReceiptStatusAssigned || assigned.AssignedKind != domain.AssignmentShareholderLoan ||
		assigned.AssignedID != view.Loan.ID || assigned.NeedsBooking() {
		t.Fatalf("Vertrag: Status %s, Zuordnung %s/%d", assigned.Status, assigned.AssignedKind, assigned.AssignedID)
	}
	if _, err := env.receipts.AddFile(ctx, contract.ID, NewFile{Role: domain.ReceiptRoleAttachment,
		FileName: "anlage.pdf", Content: []byte(minimalPDF)}); err == nil {
		t.Error("die Dateien eines zugeordneten Belegs sind eingefroren")
	}

	other := env.fileIncoming(t, "darlehensvertrag-neu.pdf")
	req.ContractReceiptID = other.ID
	if _, err := svc.CorrectLoan(ctx, view.Loan.ID, req, "falscher Vertrag abgelegt"); err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	released, _ := env.receipts.Get(ctx, contract.ID)
	now, _ := env.receipts.Get(ctx, other.ID)
	if released.Status != domain.ReceiptStatusFiled || !released.NeedsBooking() || now.Status != domain.ReceiptStatusAssigned {
		t.Errorf("nach der Berichtigung: alt %s, neu %s", released.Status, now.Status)
	}
}
