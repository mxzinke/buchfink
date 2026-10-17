package procdoc

import (
	"strings"
	"testing"
	"time"
)

// Das Verzeichnis von Verarbeitungstätigkeiten nach Art. 30 DSGVO ist Teil der
// Verfahrensdokumentation (QUE-02 K1, K2, K4).
//
// Geprüft wird an dem Text, den die Vorlage erzeugt: die Angaben, die Art. 30
// Abs. 1 DSGVO verlangt — Zwecke, Kategorien, Empfänger, Löschfristen — und die
// Maßnahmen nach Art. 32 DSGVO, allen voran der Schutz der Daten am Rechner
// und die Protokollierung der Lesezugriffe.
func TestProcDocCarriesTheArticle30Register(t *testing.T) {
	markdown, err := Render(Input{
		CompanyName: "Pfennig Ventures GmbH", LegalForm: "GmbH", FiscalYear: 2026,
		Street: "Hauptstraße 1", PostalLine: "80331 München",
		AppVersion: "1.0.0", RuleVersion: "2026.1", Version: "2026-09-05-1",
		CreatedAt: time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), Actor: "anna@rechner",
		TSAName: "Bundesdruckerei D-TRUST",
	})
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}

	if !strings.Contains(markdown, HeadingPrivacy) {
		t.Fatal("der Abschnitt nach Art. 30 DSGVO fehlt")
	}
	for _, want := range []string{
		// Art. 30 Abs. 1: Verantwortlicher, Zwecke, Kategorien, Empfänger,
		// Fristen.
		"Verantwortlicher", "Art. 4 Nr. 7 DSGVO",
		"Finanzbuchhaltung", "Rechnungsstellung", "Forderungsverfolgung",
		"Kategorien betroffener Personen", "Empfänger", "Löschfrist",
		"Finanzverwaltung", "Steuerberater", "Bundeszentralamt für Steuern",
		"Bundesdruckerei D-TRUST",
		// Die Fristen kommen aus den Aufbewahrungsklassen.
		"Abschnitt 3.8", "Art. 17 Abs. 3 Buchst. b",
		// Art. 32: die Maßnahmen, und zwar die, die es tatsächlich gibt.
		"Art. 32 DSGVO", "Festplattenverschlüsselung",
		"Zugriff", "Hash-Kette",
	} {
		if !strings.Contains(markdown, want) {
			t.Errorf("dem Verzeichnis fehlt %q", want)
		}
	}
	// Der Verantwortliche steht mit seiner Anschrift da und nicht als Platzhalter.
	if !strings.Contains(markdown, "Pfennig Ventures GmbH,\n80331 München") &&
		!strings.Contains(markdown, "Hauptstraße 1") {
		t.Error("die Anschrift des Verantwortlichen fehlt")
	}

	// Und der Satz übersetzt den Abschnitt mit: er ist Teil desselben Dokuments.
	template := Typst(markdown, "Verfahrensdokumentation", time.Now())
	if !strings.Contains(template, "Verzeichnis von Verarbeitungst") {
		t.Error("der gesetzte Text trägt den Abschnitt nicht")
	}
}

// Die Betriebsform nennt den lokalen Server für verbundene Apps, die
// Zustimmung in der App, die Erfassungswege und die Vorbehalte der App; das
// Verzeichnis nennt die Ablage der verbundenen Apps.
func TestProcDocDescribesConnectedApps(t *testing.T) {
	markdown, err := Render(Input{CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
		CreatedAt: time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}
	for _, want := range []string{
		"127.0.0.1", "zustimmt", "`ui`", "`system`", "`agent:<kurzname>`",
		"Ergebnisverwendung", "Gesellschafterdarlehen", "verbindungen.sqlite", "Verbundene Apps (KI-Agenten)",
	} {
		if !strings.Contains(markdown, want) {
			t.Errorf("der Verfahrensdokumentation fehlt %q", want)
		}
	}
	if strings.Contains(markdown, "keine Serverkomponente") {
		t.Error("die Verfahrensdokumentation verneint den lokalen Server")
	}
}
