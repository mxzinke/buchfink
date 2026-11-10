package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestNotesWorkflowThroughAppropriation(t *testing.T) {
	e := newTestEnv(t)
	m := e.closingModules(t)
	ctx := context.Background()
	statements := e.statements(t)
	statements.SetNotesSources(NotesSources{Texts: m.appropriation})
	m.closing.SetStatementSource(statements)
	e.post(t, "2026-06-01", domain.AccountBank, "4830", 400_000)
	if _, err := m.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", ""); err == nil {
		t.Fatal("fehlende Texte akzeptiert")
	}
	editor, err := m.appropriation.NotesEditor(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range editor.Texts {
		if section.Required {
			if _, err := m.appropriation.SaveNotesText(ctx, 2026, section.Section, "Keine entsprechenden Sachverhalte."); err != nil {
				t.Fatal(err)
			}
		}
	}
	m.adopt(t, e, 2026, "2027-05-20")
	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatal(err)
	}
	decision := e.fileIncoming(t, "beschluss.pdf")
	result, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Ausschüttung 1.000 Euro", ReceiptID: decision.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JournalEntryID == nil {
		t.Fatal("Ergebnisverwendung ohne Buchung")
	}
	entry := entryByID(t, e, *result.JournalEntryID)
	var debit, withholding domain.Cents
	for _, line := range entry.Lines {
		if line.Side == domain.SideCredit && line.Account == "3760" {
			withholding += line.Amount
		}
		if line.Side == domain.SideDebit {
			debit += line.Amount
		}
	}
	if debit != 100_000 || withholding != 26_375 {
		t.Fatalf("Ausschüttungsbuchung: %+v", entry.Lines)
	}
	for _, status := range []domain.FiscalYearStatus{domain.FiscalYearAdopted, domain.FiscalYearPrepared} {
		if _, err := m.appropriation.SaveNotesText(ctx, 2026, domain.NotesSectionAdditional, "Änderung"); err == nil {
			t.Fatalf("Änderung bei %s akzeptiert", status)
		}
		if _, err := m.closing.ReopenFiscalYear(ctx, 2026, "Anhang berichtigen"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.appropriation.SaveNotesText(ctx, 2026, domain.NotesSectionAdditional, "Berichtigte Angaben"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-05-21", ""); err != nil {
		t.Fatal(err)
	}
	fs, err := statements.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs.Notes.Missing) != 0 {
		t.Fatal(fs.Notes.Missing)
	}
	if !strings.Contains(statementTypst(fs), "Berichtigte Angaben") {
		t.Fatal("PDF-Vorlage verliert Anhangtext")
	}
	csv, err := statements.ExportCSV(ctx, 2026, domain.DepthFull)
	if err != nil || !strings.Contains(csv, "Berichtigte Angaben") {
		t.Fatalf("Anhangexport: %v", err)
	}
}
