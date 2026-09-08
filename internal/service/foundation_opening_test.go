package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
)

// openingEnv verdrahtet die Gründung mit Gliederung und Dokumentenablage, wie es
// die Anwendung tut.
func openingEnv(t *testing.T, env *testEnv) (*FoundationService, *DocumentService) {
	t.Helper()
	documents := NewDocumentService(
		repository.NewDocumentRepository(env.db),
		receiptstore.New(env.dataDir),
		repository.NewAuditRepository(env.db),
	)
	// Der Sitz ist Pflichtangabe im Kopf (§ 264 Abs. 1a Nr. 2 HGB). Die
	// Testumgebung setzt ihn nicht — hier steht er, damit die Prüfung auf
	// fehlende Angaben nicht jede Aufstellung beanstandet.
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.Seat = "München"
	if err := settings.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Sitz setzen: %v", err)
	}

	svc := env.foundations(t)
	svc.SetStatementSource(env.statements(t))
	svc.SetDocumentStore(documents)
	return svc, documents
}

// Die Eröffnungsbilanz steht auf den Tag der Beurkundung (§ 242 Abs. 1 HGB) und
// zeigt, was an diesem Tag in den Büchern steht: die Zeichnung des
// Stammkapitals und die geleistete Einlage.
func TestOpeningBalanceIsDrawnOnTheNotarizationDay(t *testing.T) {
	env := newTestEnv(t)
	svc, _ := openingEnv(t, env)
	ctx := context.Background()

	env.saveFoundation(t, svc, gmbhFoundation()) // Beurkundung 15.01.2026
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}
	// Die Notarrechnung kommt danach und gehört nicht in die Eröffnungsbilanz.
	env.book(t, "2026-01-20", "Notarkosten Gründung", "6825", "1800", 300_000)

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	if sheet.AsOf != "2026-01-15" {
		t.Errorf("Stichtag %q, erwartet den Beurkundungstag 2026-01-15", sheet.AsOf)
	}
	if !sheet.Balances {
		t.Errorf("die Eröffnungsbilanz geht nicht auf: Aktiva %s €, Passiva %s €", sheet.Assets, sheet.Equity)
	}
	// Bank 12.500 plus ausstehende Einlage 12.500 gegen Stammkapital 25.000.
	if sheet.Assets != 2_500_000 {
		t.Errorf("Aktiva %s €, erwartet 25.000,00", sheet.Assets)
	}
	if len(sheet.Findings) != 0 {
		t.Errorf("unerwartete Befunde: %v", sheet.Findings)
	}

	// Der Kopf trägt die Firma mit dem Zusatz und die Rechtsgrundlage.
	if !strings.Contains(sheet.Header.FirmName, "i. G.") {
		t.Errorf("Firma %q, erwartet den Zusatz i. G.", sheet.Header.FirmName)
	}
	if sheet.Header.Reference != "§ 242 Abs. 1 HGB" {
		t.Errorf("Rechtsgrundlage %q", sheet.Header.Reference)
	}
	if sheet.Header.ClosingDate != "2026-01-15" {
		t.Errorf("Stichtag im Kopf %q, erwartet den Beurkundungstag", sheet.Header.ClosingDate)
	}
}

// Ohne Gründungsbuchung sagt die Aufstellung, was fehlt, statt eine leere
// Bilanz auszuweisen.
func TestOpeningBalanceNamesTheMissingPosting(t *testing.T) {
	env := newTestEnv(t)
	svc, _ := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	if len(sheet.Findings) == 0 {
		t.Fatal("ohne jede Buchung muss die Aufstellung einen Befund nennen")
	}
	if !strings.Contains(strings.Join(sheet.Findings, " "), "Zeichnung des Stammkapitals") {
		t.Errorf("der Befund nennt nicht, was zu tun ist: %v", sheet.Findings)
	}
}

// Registergericht und -nummer fehlen vor der Eintragung notwendigerweise. Nach
// ihnen zu fragen hieße, etwas zu verlangen, das es noch nicht geben kann.
func TestOpeningBalanceDoesNotAskForTheRegisterBeforeRegistration(t *testing.T) {
	env := newTestEnv(t)
	svc, _ := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	for _, finding := range sheet.Findings {
		if strings.Contains(finding, "Register") {
			t.Errorf("vor der Eintragung darf das Register kein Mangel sein: %q", finding)
		}
	}
}

// Eine Bilanz, die nicht aufgeht, wird nicht abgelegt.
func TestFileOpeningBalanceRefusesAnUnbalancedSheet(t *testing.T) {
	env := newTestEnv(t)
	svc, _ := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())
	// Eine einseitige Buchung lässt sich nicht erzeugen; stattdessen wird gegen
	// ein Erfolgskonto gebucht: das steht in keiner der beiden Bilanzseiten.
	env.book(t, "2026-01-15", "Aufwand am Gründungstag", "6825", "1800", 100_000)

	if _, err := svc.FileOpeningBalance(ctx); err == nil {
		t.Fatal("eine Bilanz, die nicht aufgeht, darf nicht abgelegt werden")
	}
}
