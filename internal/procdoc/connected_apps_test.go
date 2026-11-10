package procdoc

import (
	"strings"
	"testing"
	"time"
)

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
