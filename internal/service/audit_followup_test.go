package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestOpeningBalanceRejectsMissingCapitalOrContributionAndUsesActualFilingDate(t *testing.T) {
	e := newTestEnv(t)
	svc, docs := openingEnv(t, e)
	ctx := context.Background()
	e.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.FileOpeningBalance(ctx); err == nil {
		t.Fatal("empty opening balance filed")
	}
	stored, err := docs.List(ctx)
	if err != nil || len(stored) != 0 {
		t.Fatalf("rejected balance created document: %v", err)
	}
	// A complete capital subscription is balanced, but the declared payment is missing.
	e.post(t, "2026-01-15", "1298", "2900", 2500000)
	if _, err := svc.FileOpeningBalance(ctx); err == nil {
		t.Fatal("declared but unbooked contribution accepted")
	}
	e.post(t, "2026-01-15", "1800", "1298", 1250000)
	doc, err := svc.FileOpeningBalance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.AsOf != "2026-01-15" || sheet.FiledOn != time.Now().In(time.Local).Format("2006-01-02") || doc.DocumentDate != "2026-01-15" {
		t.Fatalf("filing date confused with balance-sheet date: %+v", sheet)
	}
}

func TestExportIncludesCompanyDocumentsAndRejectsCorruption(t *testing.T) {
	e := newTestEnv(t)
	_, docs := openingEnv(t, e)
	ctx := context.Background()
	content := []byte("%PDF-1.7\nOpening balance evidence fixture\n%%EOF")
	doc, err := docs.Attach(ctx, DocumentRequest{Kind: domain.DocEroeffnungsbilanz, Title: "Eröffnungsbilanz", DocumentDate: "2026-01-15", DutyKey: DutyKeyEroeffnungsbilanz, FileName: "eroeffnung.pdf", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	e.receipts.SetCompanyDocuments(repository.NewDocumentRepository(e.db))
	dir := filepath.Join(t.TempDir(), "archive")
	result, err := e.exports(t).ExportArchive(ctx, 2026, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.DocumentFiles != 1 {
		t.Fatalf("company document missing: %+v", result)
	}
	rows := readCSV(t, filepath.Join(dir, "unternehmensdokumente.csv"))
	if len(rows) != 2 {
		t.Fatalf("company document metadata missing: %+v", rows)
	}
	path := rows[1][columnIndex(t, rows[0], "Pfad_im_Export")]
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil || string(data) != string(content) {
		t.Fatalf("wrong exported evidence: %v", err)
	}
	if validator, err := exec.LookPath("xmllint"); err == nil {
		cmd := exec.Command(validator, "--noout", "--dtdvalid", "validierung/gdpdu-deterministisch.dtd", "index.xml")
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("GDPdU validation failed: %v\n%s", err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(e.dataDir, doc.StoredPath), []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	check, err := e.receipts.VerifyReceiptFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(check.Issues) == 0 {
		t.Fatal("integrity check ignored company document corruption")
	}
	if _, err := e.exports(t).ExportArchive(ctx, 2026, filepath.Join(t.TempDir(), "bad")); err == nil {
		t.Fatal("corrupt company document exported successfully")
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
