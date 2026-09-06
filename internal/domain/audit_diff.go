package domain

import (
	"bytes"
	"encoding/json"
	"sort"
)

// ChangedFields rendert Vorher und Nachher eines geänderten Objekts auf die
// Felder, die sich tatsächlich unterscheiden.
//
// Der ganze Datensatz in beiden Ständen wäre einfacher zu schreiben und im
// Ergebnis unbrauchbar: die eine geänderte Zeile stünde zwischen dreißig
// unveränderten, und wer im Protokoll nachsieht, müsste zwei JSON-Blöcke von
// Hand vergleichen. GoBD Rz. 34 will die Änderung erkennbar haben, nicht das
// Objekt.
//
// before darf nil sein — dann ist das Objekt neu, „vorher" ist leer, und
// „nachher" trägt jedes belegte Feld.
//
// ignore nennt Felder, die nicht ins Protokoll gehören: berechnete Werte, die
// nicht gespeichert werden (ein Kontostand, ein Hinweistext), würden sonst als
// Änderung erscheinen, obwohl niemand etwas geändert hat.
func ChangedFields(before, after any, ignore ...string) (beforeJSON, afterJSON string) {
	beforeMap := toFieldMap(before)
	afterMap := toFieldMap(after)
	for _, name := range ignore {
		delete(beforeMap, name)
		delete(afterMap, name)
	}

	keys := make([]string, 0, len(beforeMap)+len(afterMap))
	seen := map[string]bool{}
	for k := range beforeMap {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range afterMap {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	changedBefore := map[string]json.RawMessage{}
	changedAfter := map[string]json.RawMessage{}
	for _, k := range keys {
		b, hasBefore := beforeMap[k]
		a, hasAfter := afterMap[k]
		if hasBefore && hasAfter && bytes.Equal(b, a) {
			continue
		}
		// Ein Feld, das in beiden Ständen fehlt oder in beiden leer ist, ist
		// keine Änderung. `null` und „nicht vorhanden" werden deshalb gleich
		// behandelt.
		if isEmptyValue(b) && isEmptyValue(a) {
			continue
		}
		if hasBefore {
			changedBefore[k] = b
		}
		if hasAfter {
			changedAfter[k] = a
		}
	}

	if len(changedBefore) == 0 && len(changedAfter) == 0 {
		return "", ""
	}
	return marshalMap(changedBefore), marshalMap(changedAfter)
}

// toFieldMap bringt ein beliebiges Objekt über seine JSON-Form auf eine Karte
// von Feldname zu Rohwert.
//
// Über JSON und nicht über Reflexion: die JSON-Namen sind die, unter denen die
// Oberfläche die Felder kennt, und der Serializer lässt aus, was ohnehin nicht
// gespeichert wird. Ein Objekt, das sich nicht serialisieren lässt, liefert eine
// leere Karte — ein Protokolleintrag darf an seinem Vorher/Nachher nicht
// scheitern.
func toFieldMap(v any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	if v == nil {
		return out
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]json.RawMessage{}
	}
	return out
}

func isEmptyValue(raw json.RawMessage) bool {
	s := string(bytes.TrimSpace(raw))
	switch s {
	case "", "null", `""`, "0", "false", "[]", "{}":
		return true
	}
	return false
}

func marshalMap(m map[string]json.RawMessage) string {
	if len(m) == 0 {
		return ""
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(raw)
}
