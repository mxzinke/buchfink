package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"

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

// Der Gründungsweg zählt seinen Fortschritt selbst und nennt den nächsten
// Schritt. Wartende Schritte zählen nicht als offen: zu tun ist an ihnen gerade
// nichts.
func TestFoundationGuideCountsProgressAndNamesTheNextStep(t *testing.T) {
	env := newTestEnv(t)
	svc, _ := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())

	state, err := svc.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	guide := state.Guide
	if guide.Total != len(state.Duties) || guide.Total == 0 {
		t.Fatalf("der Weg zählt %d von %d Schritten", guide.Total, len(state.Duties))
	}
	if guide.Done+guide.Open+guide.Waiting != guide.Total {
		t.Errorf("die Zählung geht nicht auf: %d erledigt, %d offen, %d wartend von %d",
			guide.Done, guide.Open, guide.Waiting, guide.Total)
	}
	// Vor der Eintragung warten Gewerbeanmeldung und Transparenzregister.
	if guide.Waiting < 2 {
		t.Errorf("%d wartende Schritte, erwartet mindestens zwei vor der Eintragung", guide.Waiting)
	}
	// Der erste offene Schritt ist die Anmeldung zum Handelsregister.
	if guide.NextKey != "handelsregister" {
		t.Errorf("nächster Schritt %q, erwartet die Anmeldung zum Handelsregister", guide.NextKey)
	}
	if guide.NextTitle == "" {
		t.Error("der nächste Schritt hat keinen Titel")
	}

	// Erledigt verschiebt den nächsten Schritt.
	if err := svc.CompleteDuty(ctx, "handelsregister", "2026-02-01", ""); err != nil {
		t.Fatalf("Pflicht quittieren: %v", err)
	}
	state, err = svc.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Guide.Done != 1 {
		t.Errorf("%d erledigt, erwartet 1", state.Guide.Done)
	}
	if state.Guide.NextKey == "handelsregister" {
		t.Error("der erledigte Schritt darf nicht mehr der nächste sein")
	}
}

// Jeder Schritt sagt, wo er zu erledigen ist und was zu tun ist. Ohne das wäre
// die Liste eine Aufzählung von Pflichten und keine Anleitung.
func TestFoundationDutiesCarryInstructions(t *testing.T) {
	env := newTestEnv(t)
	svc, _ := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())

	state, err := svc.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	for _, duty := range state.Duties {
		if duty.Order == 0 {
			t.Errorf("%s hat keinen Platz im Weg", duty.Key)
		}
		if strings.TrimSpace(duty.Where) == "" {
			t.Errorf("%s sagt nicht, wo es zu erledigen ist", duty.Key)
		}
		if len(duty.Todo) == 0 {
			t.Errorf("%s nennt keine Handgriffe", duty.Key)
		}
	}
	// Die Reihenfolge ist die des Weges und nicht die der Fälligkeit.
	for i := 1; i < len(state.Duties); i++ {
		if state.Duties[i-1].Order > state.Duties[i].Order {
			t.Errorf("die Schritte stehen nicht in ihrer Reihenfolge: %d vor %d",
				state.Duties[i-1].Order, state.Duties[i].Order)
		}
	}
}

// Ein abgelegter Nachweis hängt an seinem Schritt.
func TestFoundationDutyCarriesItsProof(t *testing.T) {
	env := newTestEnv(t)
	svc, documents := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())

	if _, err := documents.Attach(ctx, DocumentRequest{
		Kind:     domain.DocHandelsregister,
		Title:    "Eintragungsnachricht",
		DutyKey:  "handelsregister",
		FileName: "auszug.pdf",
		Content:  []byte("%PDF-1.4 Auszug"),
	}); err != nil {
		t.Fatalf("Nachweis ablegen: %v", err)
	}

	state, err := svc.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	for _, duty := range state.Duties {
		if duty.Key != "handelsregister" {
			if len(duty.Proof) != 0 {
				t.Errorf("%s trägt einen fremden Nachweis", duty.Key)
			}
			continue
		}
		if len(duty.Proof) != 1 || duty.Proof[0].Title != "Eintragungsnachricht" {
			t.Errorf("der Nachweis hängt nicht am Schritt: %+v", duty.Proof)
		}
	}
}
