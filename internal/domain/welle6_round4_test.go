package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// „Der besondere Besteuerungsverfahren … sind" war ein Kongruenzfehler in einem
// Text, der im Hilfefenster der Umsatzsteuer-Einstellungen und in der Tabelle
// „Grenzen des Funktionsumfangs" steht.
func TestTaxCaseHintsAreGrammatical(t *testing.T) {
	joined := strings.Join(TaxCaseHints(), " ")
	if strings.Contains(joined, "Der besondere Besteuerungsverfahren") {
		t.Errorf("der Hinweis zu OSS und IOSS ist grammatisch falsch: %q", joined)
	}
	if !strings.Contains(joined, "Die besonderen Besteuerungsverfahren OSS und IOSS") {
		t.Errorf("der Hinweis zu OSS und IOSS fehlt oder lautet anders: %q", joined)
	}
}

// RetentionUntil ist der letzte Aufbewahrungstag, nicht das Löschdatum. Wer das
// Feld als „frühestes Löschdatum" anzeigt, löscht einen Tag zu früh — deshalb
// liefert der Beleg beide Tage aus.
func TestReceiptCarriesTheEarliestDeletionNextToTheRetentionEnd(t *testing.T) {
	if got := EarliestDeletionAfter("2033-12-31"); got != "2034-01-01" {
		t.Errorf("frühestes Löschdatum %q, erwartet 2034-01-01", got)
	}
	if got := EarliestDeletionAfter(""); got != "" {
		t.Errorf("ohne Aufbewahrungsende darf kein Datum erfunden werden, geliefert %q", got)
	}

	receipt := Receipt{
		ReceiptNumber: "ER-2025-0001", FiscalYear: 2025,
		RetentionClass: RetentionClassVouchers, RetentionUntil: "2033-12-31",
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("Beleg als JSON: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	if decoded["retentionUntil"] != "2033-12-31" {
		t.Errorf("retentionUntil ist %v, erwartet 2033-12-31", decoded["retentionUntil"])
	}
	if decoded["earliestDeletion"] != "2034-01-01" {
		t.Errorf("earliestDeletion ist %v, erwartet 2034-01-01", decoded["earliestDeletion"])
	}
	// Der Rest des Belegs muss unverändert mitgehen — die Marshal-Methode darf
	// keine Felder verschlucken.
	if decoded["receiptNumber"] != "ER-2025-0001" {
		t.Errorf("die Belegnummer fehlt in der Ausgabe: %s", raw)
	}

}
