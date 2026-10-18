package wailsbridge

import (
	"github.com/buchfink/buchfink/internal/accounting"
)

// Die laufende Buchhaltung in der Oberfläche: Belegverweis in beide Richtungen,
// Aufbewahrung und der Filter, mit dem sich eine Buchung im Journal
// wiederfinden lässt. Gebucht wird hier nicht; das tut der Agent über den Kern.

// --- Aufbewahrung ---------------------------------------------------------

// GetRetentionRules liefert die Fristentabelle mit ihrer Quelle und ihrem
// Gültigkeitsstand — für die Anzeige in den Einstellungen.
func (b *BuchfinkBridge) GetRetentionRules() accounting.RetentionRules {
	return accounting.LoadedRetentionRules()
}
