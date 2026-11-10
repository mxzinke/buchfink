package wailsbridge

import (
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Der Pflichtenkalender eines Jahres kommt vollständig in die Oberfläche; eine
// Pflicht mit Nachweis durch den Menschen lässt sich dort bestätigen, eine
// andere mit Grund als entfallen kennzeichnen. Pflichten, deren Stand aus den
// Daten folgt, nimmt die Bestätigung nicht an.
func TestObligationsAreConfirmedInTheApp(t *testing.T) {
	b := testBridge(t)
	list, err := b.GetObligations(2026)
	if err != nil {
		t.Fatal(err)
	}
	find := func(kind domain.ObligationKind) *ObligationView {
		for i := range list {
			if list[i].Kind == kind {
				return &list[i]
			}
		}
		return nil
	}
	commit, ebilanz := find(domain.ObligationMonthCommit), find(domain.ObligationEBilanz)
	if commit == nil || ebilanz == nil {
		t.Fatalf("Monatsfestschreibung oder E-Bilanz fehlt im Kalender 2026: %d Pflichten", len(list))
	}
	for _, o := range list {
		if o.FiscalYear != 2026 || o.Hints == nil {
			t.Errorf("%s: Geschäftsjahr %d, Hinweise %v", o.Key, o.FiscalYear, o.Hints)
		}
	}

	today := time.Now().Format(domain.DateLayout)
	if _, err := b.ConfirmObligation(commit.Key, service.ConfirmInput{Date: today}); err == nil {
		t.Errorf("Monatsfestschreibung von Hand bestätigt; ihr Stand folgt aus den Daten")
	}
	if _, err := b.ConfirmObligation(ebilanz.Key, service.ConfirmInput{
		Date: today, Kind: domain.EvidenceTransferTicket}); err == nil {
		t.Errorf("Transferticket ohne Nummer angenommen")
	}
	done, err := b.ConfirmObligation(ebilanz.Key, service.ConfirmInput{
		Date: today, Kind: domain.EvidenceTransferTicket, Reference: "ET-4711"})
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != domain.ObligationDone || done.Proof == nil || done.Proof.Reference != "ET-4711" {
		t.Errorf("bestätigte E-Bilanz: Stand %s, Nachweis %+v", done.Status, done.Proof)
	}

	payroll := ""
	for _, o := range list {
		if o.Evidence != domain.EvidenceFromData && !o.Status.Closed() && o.Key != ebilanz.Key {
			payroll = o.Key
			break
		}
	}
	if payroll == "" {
		t.Fatal("keine weitere Pflicht, die sich von Hand entscheiden ließe")
	}
	if _, err := b.WaiveObligation(payroll, " "); err == nil {
		t.Errorf("Entfallen ohne Grund angenommen")
	}
	waived, err := b.WaiveObligation(payroll, "keine Arbeitnehmer")
	if err != nil {
		t.Fatal(err)
	}
	if waived.Status != domain.ObligationWaived || waived.WaivedReason != "keine Arbeitnehmer" {
		t.Errorf("entfallene Pflicht: %+v", waived)
	}
}

// Eine Jahreserklärung kommt mit Übersicht und Vordrucken. Eine Angabe aus
// der App gilt sofort und lässt sich zurücknehmen; die Übermittlung braucht
// Transferticket und den Zahlen-Hash des übermittelten Stands.
func TestDeclarationFromInputToSubmission(t *testing.T) {
	b := testBridge(t)
	kind := string(declaration.KindCorporateTax)
	view, err := b.GetDeclaration(kind, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if view.Overview.Kind != declaration.KindCorporateTax || len(view.Declaration.Forms) == 0 ||
		len(view.Overview.Forms) != len(view.Declaration.Forms) {
		t.Fatalf("Übersicht %+v mit %d Vordrucken", view.Overview, len(view.Declaration.Forms))
	}
	if _, err := b.GetDeclaration("lohnsteuer", 2026); err == nil {
		t.Errorf("unbekannte Erklärung gerechnet")
	}

	in, err := b.SetDeclarationInput(service.InputRequest{
		Kind: declaration.KindCorporateTax, Year: 2026, Key: "kst.v.z14",
		Value: declaration.Cents(250000), Source: declaration.InputFromNotice, Note: "Bescheid 2025"})
	if err != nil {
		t.Fatal(err)
	}
	if !in.Confirmed || in.Channel != "ui" {
		t.Errorf("Angabe aus der App: %+v", in)
	}
	view, _ = b.GetDeclaration(kind, 2026)
	if l, ok := view.Declaration.Line("kst.v.z14"); !ok || l.Status != declaration.StatusInput || l.Value.Amount() != 250000 {
		t.Errorf("Zeile nach der Angabe: %+v", l)
	}
	if _, err := b.ConfirmDeclarationInput(kind, 2026, "kst.v.z14"); err != nil {
		t.Errorf("eine schon bestätigte Angabe noch einmal bestätigen: %v", err)
	}
	if err := b.DeleteDeclarationInput(kind, 2026, "kst.v.z14"); err != nil {
		t.Fatal(err)
	}
	view, _ = b.GetDeclaration(kind, 2026)
	if l, _ := view.Declaration.Line("kst.v.z14"); l.Status == declaration.StatusInput {
		t.Errorf("zurückgenommene Angabe gilt weiter: %+v", l)
	}

	today := time.Now().Format(domain.DateLayout)
	if _, err := b.SubmitDeclaration(service.SubmissionRequest{Kind: declaration.KindCorporateTax, Year: 2026,
		SubmittedAt: today, FiguresHash: view.Declaration.FiguresHash}); err == nil {
		t.Errorf("Übermittlung ohne Transferticket erfasst")
	}
	reason := ""
	if len(view.Declaration.Blocking()) > 0 {
		reason = "Testmandant ohne Stammdaten"
	}
	if _, err := b.SubmitDeclaration(service.SubmissionRequest{Kind: declaration.KindCorporateTax, Year: 2026,
		SubmittedAt: today, TransferTicket: "TT-2026-1", FiguresHash: view.Declaration.FiguresHash,
		OverrideReason: reason}); err != nil {
		t.Fatal(err)
	}
	view, _ = b.GetDeclaration(kind, 2026)
	if view.Declaration.Submission == nil || view.Declaration.Submission.TransferTicket != "TT-2026-1" {
		t.Errorf("Übermittlung fehlt in der Erklärung: %+v", view.Declaration.Submission)
	}
}

// Die E-Bilanz gibt es nur über den Speichern-Dialog; ohne Anwendung bleibt
// der Pfad leer. Blockierende Befunde melden sich davor, ein unbekanntes
// Format auch.
func TestEBilanzIsSavedThroughTheDialog(t *testing.T) {
	b := testBridge(t)
	for _, format := range []string{"", "xbrl", "susa_csv"} {
		path, err := b.SaveEBilanzAs(2026, format)
		if err == nil && path != "" {
			t.Errorf("%s: ohne Dialog gespeichert unter %s", format, path)
		}
	}
	if _, err := b.SaveEBilanzAs(2026, "pdf"); err == nil {
		t.Error("ein unbekanntes Format wurde angenommen")
	}
}

// Die E-Bilanz der Eröffnungsbilanz steht auf dem Gründungstag aus den
// Unternehmensdaten; ohne ihn entsteht sie nicht.
func TestOpeningEBilanzNeedsTheFoundingDate(t *testing.T) {
	b := testBridge(t)
	if _, err := b.SaveOpeningEBilanzAs(2026); err == nil || !strings.Contains(err.Error(), "Gründungsdatum") {
		t.Errorf("ohne Gründungsdatum: %v", err)
	}
}

// Das PDF der Eröffnungsbilanz entsteht vor dem Speichern-Dialog: was ihr
// fehlt, meldet sich, bevor jemand einen Ort sucht. Im Prüfermodus bleibt es
// zu haben wie die E-Bilanz.
func TestOpeningBalancePDFIsSetBeforeTheDialog(t *testing.T) {
	b := testBridge(t)
	if _, err := b.SaveOpeningBalanceAs(); err == nil || !strings.Contains(err.Error(), "Gründungsdatum") {
		t.Errorf("ohne Gründungsdatum: %v", err)
	}
	if _, err := b.SaveFoundation("2026-01-15", domain.Foundation{
		ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{{Name: "Anna Bauer", ShareCapital: 2_500_000, PaidIn: 1_250_000,
			Kind: domain.ContributionCash}},
	}); err != nil {
		t.Fatalf("Gründung: %v", err)
	}
	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2026"); err != nil {
		t.Fatalf("Prüfermodus: %v", err)
	}
	if _, err := b.SaveOpeningBalanceAs(); err == nil || !strings.Contains(err.Error(), "Zeichnung des Stammkapitals") {
		t.Errorf("ohne Gründungsbuchungen im Prüfermodus: %v", err)
	}
}

// Bestätigen, Entfallen, Übermitteln und das Bestätigen eines Vorschlags sind
// der App vorbehalten.
func TestClosingsDecisionsAreAppOnly(t *testing.T) {
	methods := bridgeMethods(t)
	for _, name := range []string{
		"ConfirmObligation", "WaiveObligation", "ConfirmDeclarationInput", "SubmitDeclaration",
	} {
		if m, ok := methods[name]; !ok || !m.appOnly {
			t.Errorf("%s schreibt ohne core.AppOnly()", name)
		}
	}
}
