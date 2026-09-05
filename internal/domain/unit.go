package domain

import "strings"

// Die Mengeneinheit einer Rechnungsposition (BT-130).
//
// EN 16931 verlangt an jeder Position einen Schlüssel aus UN/ECE Rec. 20 —
// „Stunde" ist dort kein zulässiger Wert, „HUR" ist es. Buchfink hat bisher
// fest C62 (Stück) geschrieben, egal was in der Position stand: eine
// Beratungsrechnung über 12 Stunden kam beim Empfänger als 12 Stück an, und die
// Preisprüfung seines Systems verglich Äpfel mit Birnen.
//
// Die Tabelle ist bewusst kurz. Rec. 20 kennt tausende Schlüssel; angeboten
// wird, was ein Dienstleister und ein kleiner Händler brauchen. Ein Schlüssel,
// den niemand versteht, ist schlechter als eine Auswahl, aus der man den
// richtigen findet.

// UnitCode is one unit of measure with its German label.
type UnitCode struct {
	Code  string `json:"code"`  // UN/ECE Rec. 20, z. B. "HUR"
	Label string `json:"label"` // Klartext für die Oberfläche und das PDF
}

// UnitCodeDefault is what a position without a stated unit is billed in.
const UnitCodeDefault = "C62"

// unitCodes are the units Buchfink offers, in the order the UI shows them.
var unitCodes = []UnitCode{
	{"C62", "Stück"},
	{"HUR", "Stunde"},
	{"DAY", "Tag"},
	{"MON", "Monat"},
	{"KGM", "Kilogramm"},
	{"MTR", "Meter"},
	{"LTR", "Liter"},
	{"E48", "Leistungseinheit"},
	{"XPP", "Pauschale"},
}

// UnitCodes lists the offered units of measure.
func UnitCodes() []UnitCode {
	out := make([]UnitCode, len(unitCodes))
	copy(out, unitCodes)
	return out
}

// ResolveUnitCode turns what stands in a position into a Rec. 20 key.
//
// Both spellings are accepted: the key itself and the German label. The label
// path is not convenience — Rechnungen aus der Zeit vor dieser Welle tragen
// „Stück" oder „Stunde" im Feld, und ohne die Rückübersetzung würde jede alte
// Rechnung beim erneuten Erzeugen ihres Dokuments zu Stück.
func ResolveUnitCode(unit string) (string, bool) {
	trimmed := strings.TrimSpace(unit)
	if trimmed == "" {
		return UnitCodeDefault, true
	}
	upper := strings.ToUpper(trimmed)
	for _, u := range unitCodes {
		if u.Code == upper {
			return u.Code, true
		}
		if strings.EqualFold(u.Label, trimmed) {
			return u.Code, true
		}
	}
	// Zwei Schreibweisen, die vor dieser Welle im Feld standen und keinen
	// eigenen Eintrag rechtfertigen.
	switch strings.ToLower(trimmed) {
	case "stk", "stk.", "st.":
		return "C62", true
	case "h", "std", "std.", "stunden":
		return "HUR", true
	}
	return "", false
}

// UnitLabel is the German text for a Rec. 20 key, falling back to the key
// itself so an unknown unit still prints something the reader can check.
func UnitLabel(code string) string {
	upper := strings.ToUpper(strings.TrimSpace(code))
	for _, u := range unitCodes {
		if u.Code == upper {
			return u.Label
		}
	}
	if label, ok := ResolveUnitCode(code); ok {
		for _, u := range unitCodes {
			if u.Code == label {
				return u.Label
			}
		}
	}
	return code
}
