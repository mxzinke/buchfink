package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Prüfungen der Welle 8: keine Buchung ohne Beleg, der Eigenbeleg als der
// zweite Weg dorthin, die überschreibbare Aufbewahrungsfrist, die
// Fehlerklassen der Klärungsliste und der Voranmeldungszeitraum aus der
// Vorjahressteuer.

// withSelfIssued rüstet den Belegdienst für den Eigenbeleg aus: Setzer,
// Unternehmensangaben und Nummernkreis.
func (e *testEnv) withSelfIssued(t *testing.T) *stubRenderer {
	t.Helper()
	renderer := &stubRenderer{}
	e.receipts.SetRenderer(renderer)
	e.receipts.SetSettingsSource(repository.NewSettingsRepository(e.db))
	e.receipts.SetNumberRepo(e.numberRepo)
	return renderer
}

// Der Eigenbeleg entsteht mit PDF und Kopfdaten (BEL-01 K1/K2, GOB-05 K1).
func TestSelfIssuedReceiptCarriesItsPDFAndHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	renderer := env.withSelfIssued(t)

	receipt, err := env.receipts.CreateSelfIssued(ctx, SelfIssuedReceiptRequest{
		DocumentDate: "2026-03-01", GrossAmount: 500,
		Reason: "Parkgebühr am Automaten, Quittung nicht ausgedruckt",
	})
	if err != nil {
		t.Fatalf("Eigenbeleg: %v", err)
	}
	if renderer.documents != 1 {
		t.Fatalf("%d gesetzte Dokumente — der Eigenbeleg ist ein PDF", renderer.documents)
	}
	if receipt.Kind != domain.ReceiptKindSelfIssued {
		t.Errorf("Belegart %q — erwartet den Eigenbeleg", receipt.Kind)
	}
	if receipt.IssuerName != "Pfennig Ventures GmbH" {
		t.Errorf("Aussteller %q — der Eigenbeleg stammt vom eigenen Unternehmen", receipt.IssuerName)
	}
	if receipt.DocumentDate != "2026-03-01" || receipt.GrossAmount != 500 {
		t.Errorf("Kopfdaten unvollständig: %s / %s", receipt.DocumentDate, receipt.GrossAmount)
	}
	if err := receipt.ValidateHeader(); err != nil {
		t.Errorf("der Eigenbeleg muss ohne Nacharbeit buchbar sein: %v", err)
	}
	// Die Aufbewahrungsklasse ist die des Buchungsbelegs; ein Eigenbeleg ist
	// einer (§ 147 Abs. 1 Nr. 4 AO).
	if receipt.RetentionClass != domain.RetentionClassVouchers {
		t.Errorf("Aufbewahrungsklasse %q — erwartet die des Buchungsbelegs", receipt.RetentionClass)
	}
	// Der Satz, der den Eigenbeleg zum Eigenbeleg macht, steht im Dokument.
	if !strings.Contains(renderer.lastText, "kein Fremdbeleg") {
		t.Error("dem Dokument fehlt der Hinweis, dass kein Fremdbeleg vorliegt")
	}
	if !strings.Contains(renderer.lastText, "Parkgebühr am Automaten") {
		t.Error("dem Dokument fehlt der Grund")
	}
	// Die Originaldatei ist das erzeugte PDF, nicht abgeleitet: es gibt keine
	// Datei, aus der es entstanden wäre.
	original, ok := receipt.FileByRole(domain.ReceiptRoleOriginal)
	if !ok {
		t.Fatal("dem Eigenbeleg fehlt die Originaldatei")
	}
	if original.Derived {
		t.Error("das PDF des Eigenbelegs ist das Original und nicht abgeleitet")
	}
}

// Der Eigenbeleg ohne Grund entsteht nicht: der Grund ist sein ganzer Inhalt.
func TestSelfIssuedReceiptNeedsItsReason(t *testing.T) {
	env := newTestEnv(t)
	env.withSelfIssued(t)

	_, err := env.receipts.CreateSelfIssued(context.Background(), SelfIssuedReceiptRequest{
		DocumentDate: "2026-03-01", GrossAmount: 500,
	})
	if err == nil {
		t.Fatal("ein Eigenbeleg ohne Grund belegt nichts")
	}
	if !strings.Contains(err.Error(), "Grund") {
		t.Errorf("die Meldung nennt nicht, was fehlt: %v", err)
	}
}
