package accounting

import (
	"strings"
	"testing"
)

// Buchfink löscht nicht. Die Antwort verspricht deshalb keine Löschung durch
// die Software, sondern durch das Unternehmen nach Fristablauf, stützt die
// Sperre nicht auf Art. 18 DSGVO, dessen Fälle hier nicht vorliegen, und nennt
// Beschwerde und Rechtsbehelf (Art. 12 Abs. 4 DSGVO).
func TestBlockedContactAnswerPromisesOnlyWhatTheControllerDoes(t *testing.T) {
	answer := BlockedContactAnswer("Meier GmbH")
	for _, needle := range []string{
		"gesperrt", "Sobald die Frist abgelaufen ist, löschen wir sie",
		"Art. 77 DSGVO", "Art. 79 DSGVO",
	} {
		if !strings.Contains(answer, needle) {
			t.Errorf("die Antwort nennt %q nicht:\n%s", needle, answer)
		}
	}
	for _, wrong := range []string{"Art. 18", "werden nach Ablauf der Aufbewahrungsfrist gelöscht", "automatisch"} {
		if strings.Contains(answer, wrong) {
			t.Errorf("die Antwort enthält %q:\n%s", wrong, answer)
		}
	}
	if anonymous := BlockedContactAnswer(""); !strings.HasPrefix(anonymous, "Die zu Ihrer Person gespeicherten Daten") {
		t.Errorf("ohne Namen: %s", anonymous)
	}
}
