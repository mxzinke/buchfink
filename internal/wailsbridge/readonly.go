package wailsbridge

import (
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ops"
)

// Der Prüfermodus.
//
// Während einer Außenprüfung wird der Datenbestand eingefroren: der Prüfer
// bekommt Zugriff auf die Daten des Prüfungszeitraums (§ 147 Abs. 6 AO), und
// was er sieht, darf sich unter seinen Händen nicht ändern. Buchfink kennt
// keine Benutzerkonten und kann den Zugriff deshalb nicht nach Personen
// trennen; die Sperre gilt für die ganze Anwendung und ist befristet.
//
// Die Prüfung sitzt im Kern (Tenant.Write): jeder Schreibvorgang geht dort
// durch, gleich ob er aus der Oberfläche oder von einem Agenten kommt.

// readOnlyAllowed wird gegen die tatsächlichen Schreibwege der Bridge geprüft.
var readOnlyAllowed = ops.BridgeExaminerAllowed()

// EnableReadOnly schaltet den Prüfermodus bis zu einem Tag ein. Datum und
// Grund sind Pflicht.
func (b *BuchfinkBridge) EnableReadOnly(until, reason string) (domain.AppConfig, error) {
	return b.host.EnableExaminer(uiContext(), until, reason)
}

// DisableReadOnly beendet den Prüfermodus. Der Grund ist Pflicht und steht im
// Änderungsprotokoll.
func (b *BuchfinkBridge) DisableReadOnly(reason string) (domain.AppConfig, error) {
	return b.host.DisableExaminer(uiContext(), reason)
}
