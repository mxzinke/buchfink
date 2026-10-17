package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestOpeningBalanceRejectsMissingCapitalOrContribution(t *testing.T) {
	e := newTestEnv(t)
	svc := openingEnv(t, e)
	ctx := context.Background()
	e.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.OpeningBalancePDF(ctx); err == nil {
		t.Fatal("empty opening balance accepted")
	}
	// A complete capital subscription is balanced, but the declared payment is missing.
	e.post(t, "2026-01-15", "1298", "2900", 2500000)
	if _, err := svc.OpeningBalancePDF(ctx); err == nil {
		t.Fatal("declared but unbooked contribution accepted")
	}
	e.post(t, "2026-01-15", "1800", "1298", 1250000)
	if _, err := svc.OpeningBalancePDF(ctx); err != nil {
		t.Fatal(err)
	}
	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.AsOf != "2026-01-15" {
		t.Fatalf("balance-sheet date %q, want the notarization day", sheet.AsOf)
	}
}

func TestClosingRequiresMicroDisclosuresAndShowsThemBelowBalance(t *testing.T) {
	e := newTestEnv(t)
	m := e.closingModules(t)
	ctx := context.Background()
	statements := e.statements(t)
	statements.SetNotesSources(NotesSources{Texts: m.appropriation})
	m.closing.SetStatementSource(statements)
	if _, err := m.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", ""); err == nil {
		t.Fatal("closing with empty disclosures accepted")
	}
	for _, section := range []domain.NotesSection{domain.NotesSectionContingent, domain.NotesSectionBoardLoans, domain.NotesSectionAdditional} {
		if _, err := m.appropriation.SaveNotesText(ctx, 2026, section, "Es bestehen keine entsprechenden Sachverhalte."); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := statements.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if fs.SizeClass.Obligations.NotesRequired || len(fs.Notes.Missing) != 0 || len(fs.Notes.BelowBalance) != 3 {
		t.Fatalf("micro disclosures: %+v", fs.Notes)
	}
	csv, err := statements.ExportCSV(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv, "angaben_unter_der_bilanz") {
		t.Fatal("CSV loses placement of substitute disclosures")
	}
	if _, err := m.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", ""); err != nil {
		t.Fatal(err)
	}
}
