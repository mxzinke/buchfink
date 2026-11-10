package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Belegkopfdaten und Originaldatei -------------------------------------

// filePDFReceipt legt einen Beleg mit einer ansehbaren Originaldatei ab.
func filePDFReceipt(t *testing.T, env *testEnv, req FileReceiptRequest) *domain.Receipt {
	t.Helper()
	if req.Direction == "" {
		req.Direction = domain.DirectionIncoming
	}
	if len(req.Files) == 0 {
		req.Files = []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf",
			Content: []byte("%PDF-1.4 Rechnung\n"),
		}}
	}
	receipt, err := env.receipts.File(context.Background(), req)
	if err != nil {
		t.Fatalf("Beleg konnte nicht abgelegt werden: %v", err)
	}
	return receipt
}

func TestReceiptGetsRetentionClassOnFiling(t *testing.T) {
	env := newTestEnv(t)

	invoice := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Lieferant GmbH", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	})
	if invoice.RetentionClass != domain.RetentionClassVouchers {
		t.Errorf("eine Rechnung ist ein Buchungsbeleg, bekommen Klasse %q", invoice.RetentionClass)
	}
	if invoice.RetentionUntil != "2034-12-31" {
		t.Errorf("Fristende %q, erwartet 2034-12-31 (acht Jahre ab Schluss 2026)", invoice.RetentionUntil)
	}

	letter := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindLetter, DocumentDate: "2026-03-01",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "vertrag.pdf",
			Content: []byte("%PDF-1.4 Vertrag\n"),
		}},
	})
	if letter.RetentionClass != domain.RetentionClassLetters {
		t.Errorf("ein Handelsbrief ist keine Rechnung, bekommen Klasse %q", letter.RetentionClass)
	}
	if letter.RetentionUntil != "2032-12-31" {
		t.Errorf("Fristende %q, erwartet 2032-12-31 (sechs Jahre ab Schluss 2026)", letter.RetentionUntil)
	}
}

// Eine verlängerte Aufbewahrungsfrist bleibt verlängert, auch wenn danach die
// Kopfdaten erfasst werden (ARC-01 K2, Entscheidung 6).
func TestSaveHeaderKeepsAnExtendedRetention(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten: %v", err)
	}
	extended, err := env.receipts.OverrideRetention(ctx, receipt.ID,
		domain.RetentionClassBooks, "Beleg gehört zu einem laufenden Einspruchsverfahren")
	if err != nil {
		t.Fatalf("Verlängerung: %v", err)
	}
	if extended.RetentionUntil != "2036-12-31" {
		t.Fatalf("Frist %s — zehn Jahre auf 2026 enden am 31.12.2036", extended.RetentionUntil)
	}

	// Dieselben Kopfdaten noch einmal, mit einem anderen Belegdatum: die
	// Neuberechnung aus Belegart und Belegdatum ergäbe wieder die kürzere
	// Belegfrist, und die Verlängerung wäre stillschweigend zurückgenommen.
	after, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-04-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("Kopfdaten nach der Verlängerung: %v", err)
	}
	if after.RetentionUntil != "2036-12-31" {
		t.Errorf("Frist %s — die Verlängerung auf 2036-12-31 darf nicht zurückfallen", after.RetentionUntil)
	}
	if after.RetentionClass != domain.RetentionClassBooks {
		t.Errorf("Klasse %s — die überschriebene Klasse bleibt", after.RetentionClass)
	}
	if after.RetentionOverrideReason == "" {
		t.Error("der Grund der Verlängerung bleibt am Beleg")
	}
}

// Die Aufbewahrungsfrist lässt sich nur verlängern, und die Verlängerung steht
// mit Vorher/Nachher im Änderungsprotokoll (ARC-01 K2).
func TestRetentionOverrideOnlyExtendsAndIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	receipt, err := env.receipts.CreateSelfIssued(ctx, SelfIssuedReceiptRequest{
		DocumentDate: "2026-03-01", GrossAmount: 500, Reason: "Parkgebühr",
	})
	if err != nil {
		t.Fatalf("Eigenbeleg: %v", err)
	}
	if receipt.RetentionUntil != "2034-12-31" {
		t.Fatalf("Frist bis %s — acht Jahre ab dem Schluss von 2026", receipt.RetentionUntil)
	}

	// Nach unten: abgewiesen. Die gesetzliche Frist ist die Untergrenze.
	if _, err := env.receipts.OverrideRetention(
		ctx, receipt.ID, domain.RetentionClassLetters, "kürzer reicht"); err == nil {
		t.Error("eine kürzere Frist wäre ein Verstoß gegen § 257 HGB und § 147 AO")
	}

	// Ohne Grund: abgewiesen.
	if _, err := env.receipts.OverrideRetention(
		ctx, receipt.ID, domain.RetentionClassBooks, "  "); err == nil {
		t.Error("zur Verlängerung gehört ihr Grund")
	}

	updated, err := env.receipts.OverrideRetention(
		ctx, receipt.ID, domain.RetentionClassBooks,
		"Beleg gehört zum Rechtsstreit 4 O 112/26")
	if err != nil {
		t.Fatalf("Verlängerung: %v", err)
	}
	if updated.RetentionUntil != "2036-12-31" {
		t.Errorf("Frist bis %s — erwartet zehn Jahre ab dem Schluss von 2026",
			updated.RetentionUntil)
	}
	if updated.RetentionOverrideReason == "" || updated.RetentionOverrideAt == "" {
		t.Error("die Verlängerung steht ohne Grund und ohne Zeitpunkt am Beleg")
	}

	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Details, "Aufbewahrungsfrist") &&
			strings.Contains(e.Details, "2034-12-31") &&
			strings.Contains(e.Details, "2036-12-31") {
			found = true
		}
	}
	if !found {
		t.Error("das Protokoll nennt nicht beide Fristen — vorher und nachher")
	}
}
