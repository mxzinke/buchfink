package domain

import (
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

// --- Hinweise -------------------------------------------------------------

func TestTaxCaseHintsNameTheGaps(t *testing.T) {
	hints := TaxCaseHints()
	if len(hints) == 0 {
		t.Fatal("die Grenzen des Funktionsumfangs müssen benannt sein")
	}
	joined := strings.Join(hints, " ")
	for _, needle := range []string{"OSS", "IOSS", "§ 19 UStG"} {
		if !strings.Contains(joined, needle) {
			t.Errorf("die Hinweise nennen %q nicht", needle)
		}
	}
}
