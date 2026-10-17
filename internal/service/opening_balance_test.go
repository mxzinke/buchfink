package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Jede Eröffnungsbuchung ist für sich ausgeglichen; aufgehen muss die
// Eröffnungsbilanz als Ganzes. Fehlt eine Passivseite, darf sie nicht gebucht
// werden, auch wenn der Ausgleich auf 9000 die einzelne Buchung schließt.
func TestOpeningBalanceMustBalanceAcrossCarryForwardAccounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := fileClosingStatement(t, env)
	req := OpeningBalanceRequest{
		FiscalYear: 2026, ReceiptID: &closing.ID, LegacySystem: "Vorgängerprogramm",
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 500000},
			{Account: "2900", Side: domain.SideCredit, Amount: 500000},
		},
		Receivables: []OpenOpeningItem{{
			ContactID: customer.ID, Amount: 119000, DocumentNumber: "RE-1", DocumentDate: "2025-12-01",
		}},
	}

	preview, err := env.journal.PreviewOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Balanced {
		t.Error("die Forderung von 1.190 € steht keiner Passivseite gegenüber; die Vorschau darf nicht ausgeglichen sein")
	}
	if preview.DebitTotal != 619000 || preview.CreditTotal != 500000 {
		t.Errorf("Aktiva/Passiva %s/%s, erwartet 6.190,00/5.000,00", preview.DebitTotal, preview.CreditTotal)
	}
	if _, err := env.journal.BookOpeningBalance(ctx, req); err == nil || !strings.Contains(err.Error(), "geht nicht auf") {
		t.Fatalf("eine Eröffnungsbilanz, die nicht aufgeht, darf nicht gebucht werden; Fehler: %v", err)
	}
	if entries, _ := env.journalRepo.FindAll(ctx, 2026); len(entries) != 0 {
		t.Errorf("%d Buchungen stehen nach der Abweisung im Journal", len(entries))
	}

	req.Accounts = append(req.Accounts, OpeningBalanceLine{
		Account: domain.AccountSaldenvortraegeSachkonten, Side: domain.SideCredit, Amount: 119000})
	if _, err := env.journal.PreviewOpeningBalance(ctx, req); err == nil {
		t.Error("ein Vortragskonto in den Beständen würde jede Differenz verdecken")
	}

	req.Accounts = req.Accounts[:2]
	req.Accounts[1].Amount = 619000
	booked, err := env.journal.BookOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die ausgeglichene Eröffnungsbilanz muss buchbar sein: %v", err)
	}
	if !booked.Balanced || carryForwardBalance(booked.Entries) != 0 {
		t.Errorf("Vortragskonten nach der Buchung: %s", carryForwardBalance(booked.Entries))
	}
}
