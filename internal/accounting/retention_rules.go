package accounting

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Aufbewahrungsfristen als Ressource (ARC-01 K4).
//
// Sie standen als Konstanten in retention.go. Das war für den Code richtig und
// für den Anwender falsch: welche Frist Buchfink anwendet und woraus sie folgt,
// war nur im Quelltext nachzulesen, und der Gesetzgeber ändert sie — das Vierte
// Bürokratieentlastungsgesetz hat die Belegfrist zum 1.1.2025 von zehn auf acht
// Jahre verkürzt. Als eingebettete Datei ist die Tabelle sichtbar, versioniert
// und in den Einstellungen anzeigbar, und die Tests prüfen gegen dieselbe
// Datei, aus der gerechnet wird.
//
// Angezeigt und nicht bearbeitet: eine Frist ist keine Wahl des Anwenders. Wer
// sie hochsetzen will, tut das am einzelnen Beleg (siehe
// ReceiptService.OverrideRetention) — dort steht dann auch der Grund.

//go:embed retention_rules.json
var retentionRulesJSON []byte

// RetentionRules ist die eingebettete Fristentabelle.
type RetentionRules struct {
	Version string `json:"version"`
	// ValidFrom ist der Tag, ab dem diese Fassung der Tabelle gilt (ARC-01 K4).
	//
	// Wie in afa_rules.json: eine Fristentabelle ohne ihren Gültigkeitsbeginn
	// ist eine Zahlensammlung ohne Rechtsstand — die Anzeige „Quelle:
	// Gesetzesstand" könnte dann nicht sagen, welcher Stand gemeint ist. Der
	// 1.1.2025 ist der Tag, an dem das Vierte Bürokratieentlastungsgesetz die
	// Belegfrist auf acht Jahre verkürzt hat.
	ValidFrom string                `json:"validFrom"`
	Source    string                `json:"source"`
	Note      string                `json:"note"`
	Classes   []RetentionClassRules `json:"classes"`
}

// RetentionClassRules ist die Frist einer Aufbewahrungsklasse.
type RetentionClassRules struct {
	Class      domain.RetentionClass `json:"class"`
	Label      string                `json:"label"`
	Years      int                   `json:"years"`
	LegalBasis string                `json:"legalBasis"`
	// Kinds sind die Objektarten, die in diese Klasse fallen.
	Kinds []domain.RetentionKind `json:"kinds"`
	// Previous ist die vorige Frist, soweit eine Änderung noch nachwirkt.
	Previous *RetentionPreviousTerm `json:"previous,omitempty"`
}

// RetentionPreviousTerm ist eine abgelöste Frist mit dem Tag ihrer Ablösung.
//
// Sie bleibt in der Tabelle, weil eine Fristverkürzung nicht rückwirkt, wo die
// alte Frist am Stichtag schon abgelaufen war: für einen Beleg aus 2013 endete
// die Zehnjahresfrist am 31.12.2023, und eine nachträglich auf acht Jahre
// verkürzte Frist wäre dort die falsche Auskunft über einen längst beendeten
// Vorgang.
type RetentionPreviousTerm struct {
	Years        int    `json:"years"`
	ReplacedFrom string `json:"replacedFrom"`
	Note         string `json:"note"`
}

var retentionRules RetentionRules

func init() {
	if err := json.Unmarshal(retentionRulesJSON, &retentionRules); err != nil {
		// Die Datei ist eingebettet: ein Fehler hier ist ein Fehler im Bau und
		// keiner, den ein Anwender beheben könnte.
		panic(fmt.Sprintf("retention_rules.json ist unlesbar: %v", err))
	}
}

// LoadedRetentionRules liefert die Fristentabelle für Anzeige und Prüfung.
func LoadedRetentionRules() RetentionRules { return retentionRules }

// rulesForClass liefert die Regel einer Klasse.
func rulesForClass(class domain.RetentionClass) (RetentionClassRules, bool) {
	for _, c := range retentionRules.Classes {
		if c.Class == class {
			return c, true
		}
	}
	return RetentionClassRules{}, false
}

// classForKind ordnet eine Objektart ihrer Klasse zu.
func classForKind(kind domain.RetentionKind) (RetentionClassRules, bool) {
	for _, c := range retentionRules.Classes {
		for _, k := range c.Kinds {
			if k == kind {
				return c, true
			}
		}
	}
	return RetentionClassRules{}, false
}

// yearOf liest das Jahr aus einem ISO-Datum. Ein unlesbares Datum ergibt 0 und
// damit eine Regel, die nie greift — geraten wird an einer Frist nicht.
func yearOf(iso string) int {
	if len(iso) < 4 {
		return 0
	}
	year, err := strconv.Atoi(iso[:4])
	if err != nil {
		return 0
	}
	return year
}
