package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Konten der Anzahlungen und der Forderungsausbuchung.
//
// Sie stehen hier und nicht in den Buchungsgruppen, weil sie kein Anwender
// wählt: welches Konto eine vereinnahmte Anzahlung trägt, folgt aus dem
// Steuersatz, und welches ein uneinbringlicher Posten trägt, folgt aus dem
// Steuersatz der ursprünglichen Rechnung. Eine Auswahl anzubieten hieße, den
// Fehler anzubieten.

// Erhaltene Anzahlungen (SKR04, Passiva D.3).
const (
	// AccountErhalteneAnzahlungen19 trägt die vereinnahmte, versteuerte
	// Anzahlung zum Regelsteuersatz.
	AccountErhalteneAnzahlungen19 = "3272"
	// AccountErhalteneAnzahlungen7 dasselbe zum ermäßigten Satz.
	AccountErhalteneAnzahlungen7 = "3260"
	// AccountErhalteneAnzahlungen trägt eine Anzahlung ohne Steuerausweis.
	AccountErhalteneAnzahlungen = "3250"
)

// Geleistete Anzahlungen (Aktiva). Sie richten sich nach der Bilanzposition,
// für die angezahlt wurde — eine Anzahlung auf eine Maschine gehört ins
// Anlagevermögen, eine auf Handelsware ins Umlaufvermögen. Aus dem Betrag lässt
// sich das nicht ableiten, deshalb ist es eine Angabe des Anwenders.
const (
	AccountGeleisteteAnzahlungenVorraete    = "1180"
	AccountGeleisteteAnzahlungenImmateriell = "0170"
	AccountGeleisteteAnzahlungenAnlagen     = "0700"
)

// AdvanceAccountFor liefert das Konto der erhaltenen Anzahlung zu einem
// Steuersatz.
//
// Der Steuersatz steht im Kontonamen, weil die Anzahlung mit der Vereinnahmung
// versteuert ist (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG) und die
// Schlussrechnung genau diesen Betrag wieder auflösen muss. Ein Sammelkonto
// ließe am Jahresende offen, welcher Teil des Saldos welche Steuer trägt.
func AdvanceAccountFor(rate domain.TaxRate) (string, error) {
	switch rate {
	case domain.TaxRateStandard:
		return AccountErhalteneAnzahlungen19, nil
	case domain.TaxRateReduced:
		return AccountErhalteneAnzahlungen7, nil
	case domain.TaxRateNone:
		return AccountErhalteneAnzahlungen, nil
	}
	return "", fmt.Errorf("für den Steuersatz %s ist kein Anzahlungskonto hinterlegt", rate.Label())
}

// Forderungsverluste (SKR04, sonstige betriebliche Aufwendungen).
const (
	AccountForderungsverluste   = "6930" // ohne Steuerausweis
	AccountForderungsverluste7  = "6931"
	AccountForderungsverluste19 = "6936"
)

// WriteOffAccountFor liefert das Aufwandskonto einer Forderungsausbuchung.
//
// Getrennt nach Steuersatz, weil § 17 Abs. 2 Nr. 1 UStG mit der Uneinbringlich-
// keit auch die Steuer berichtigt: der Aufwand und seine Steuerkorrektur
// gehören zusammen, und ein Sammelkonto verlöre die Zuordnung. Die „übliche
// Höhe" (6930er) und nicht die „unüblich hohe" (6280er): der Ausnahmefall ist
// eine Bewertungsfrage des Abschlusses und keine Voreinstellung.
func WriteOffAccountFor(rate domain.TaxRate) (string, error) {
	switch rate {
	case domain.TaxRateStandard:
		return AccountForderungsverluste19, nil
	case domain.TaxRateReduced:
		return AccountForderungsverluste7, nil
	case domain.TaxRateNone:
		return AccountForderungsverluste, nil
	}
	return "", fmt.Errorf("für den Steuersatz %s ist kein Konto für Forderungsverluste hinterlegt", rate.Label())
}

// AdvanceTarget sagt, wofür angezahlt wurde.
//
// Aus dem Betrag lässt sich das nicht ableiten, und die Bilanz trennt die
// Fälle: eine Anzahlung auf eine Maschine steht im Anlagevermögen (§ 266
// Abs. 2 A HGB), eine auf Handelsware im Umlaufvermögen. Deshalb ist es eine
// Angabe des Anwenders und keine Rechnung.
type AdvanceTarget string

const (
	// AdvanceTargetInventory ist die Anzahlung auf Vorräte (§ 266 Abs. 2 B I 4
	// HGB).
	AdvanceTargetInventory AdvanceTarget = "inventory"
	// AdvanceTargetTangible ist die Anzahlung auf eine Sachanlage; sie steht
	// zusammen mit den Anlagen im Bau (§ 266 Abs. 2 A II 4 HGB).
	AdvanceTargetTangible AdvanceTarget = "tangible"
	// AdvanceTargetIntangible ist die Anzahlung auf einen immateriellen
	// Vermögensgegenstand (§ 266 Abs. 2 A I 4 HGB).
	AdvanceTargetIntangible AdvanceTarget = "intangible"
)

// VendorAdvanceAccountFor liefert das Konto der geleisteten Anzahlung.
func VendorAdvanceAccountFor(target AdvanceTarget) (string, error) {
	switch target {
	case AdvanceTargetInventory:
		return AccountGeleisteteAnzahlungenVorraete, nil
	case AdvanceTargetTangible:
		return AccountGeleisteteAnzahlungenAnlagen, nil
	case AdvanceTargetIntangible:
		return AccountGeleisteteAnzahlungenImmateriell, nil
	}
	return "", fmt.Errorf(
		"zu einer geleisteten Anzahlung gehört die Angabe, wofür angezahlt wurde "+
			"(Vorräte, Sachanlage, immaterieller Vermögensgegenstand); %q ist keine davon", target)
}

// AllAdvanceTargets listet die Verwendungen in fester Reihenfolge — die
// Oberfläche bietet sie zur Auswahl an.
func AllAdvanceTargets() []AdvanceTarget {
	return []AdvanceTarget{AdvanceTargetInventory, AdvanceTargetTangible, AdvanceTargetIntangible}
}

// AdvanceTargetLabel ist der Klartext zur Verwendung.
func AdvanceTargetLabel(target AdvanceTarget) string {
	switch target {
	case AdvanceTargetInventory:
		return "Vorräte"
	case AdvanceTargetTangible:
		return "Sachanlagen"
	case AdvanceTargetIntangible:
		return "Immaterielle Vermögensgegenstände"
	}
	return string(target)
}
