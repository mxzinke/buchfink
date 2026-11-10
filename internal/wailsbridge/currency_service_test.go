package wailsbridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Die Umsatzsteuer-Umrechnungskurse liest der Mensch in der App aus einer
// CSV-Datei ein; danach zeigt die App die eingelesenen Monate. Im Prüfermodus
// nimmt Buchfink keine Kurse an.
func TestVatExchangeRatesAreImportedInTheApp(t *testing.T) {
	b := testBridge(t)
	path := filepath.Join(t.TempDir(), "ust-kurse.csv")
	csv := "Monat;Währung;Kurs\n2026-08;USD;1,1612\n2026-08;CHF;0,9351\n2026-09;USD;1,1702\n2026-13;USD;1,2\n"
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := b.ImportVatExchangeRates(path)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 3 || result.Skipped != 1 || len(result.Problems) != 1 ||
		!strings.Contains(result.Problems[0], "Zeile 5") {
		t.Errorf("Import: %+v", result)
	}
	rates, err := b.GetVatExchangeRates("2026-08", "2026-09")
	if err != nil || len(rates) != 3 {
		t.Fatalf("eingelesene Kurse: %+v (%v)", rates, err)
	}
	if rates[0].Month != "2026-08" || rates[0].Currency != "CHF" || rates[0].RateMicros != 935_100 {
		t.Errorf("erster Kurs: %+v", rates[0])
	}

	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2026"); err != nil {
		t.Fatalf("Prüfermodus: %v", err)
	}
	if _, err := b.ImportVatExchangeRates(path); err == nil {
		t.Error("im Prüfermodus darf die App keine Kurse einlesen")
	}
}
