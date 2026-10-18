package wailsbridge

import (
	"encoding/json"
	"strings"
	"testing"
)

// Die Bedienung der Welle 7 antwortet auch dann, wenn ihre Dienste fehlen.
//
// Der Fall ist nicht künstlich: zwischen dem Start und dem Öffnen eines
// Mandanten ist genau das der Zustand, und die Aufgabenliste ist der erste
// Bildschirm. Käme dort `null` an, nähme `null.map` im Render den ganzen Baum
// mit — und zwar bevor die Anwenderin etwas tun konnte.
func TestBridgeAnswersWithEmptyListsWithoutServices(t *testing.T) {
	b := newBridge(t)

	tasks, err := b.GetTasks()
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	raw, err := json.Marshal(tasks)
	if err != nil {
		t.Fatalf("Aufgabenliste als JSON: %v", err)
	}
	for _, key := range []string{`"overdue":[]`, `"open":[]`, `"upcoming":[]`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("die Aufgabenliste enthält %s nicht:\n%s", key, raw)
		}
	}

	rules, err := b.GetBankRules()
	if err != nil || rules == nil {
		t.Errorf("Bankregeln: %v (%v)", rules, err)
	}

	suggestions, err := b.SuggestBankMatches(1)
	if err != nil {
		t.Fatalf("Zuordnungsvorschlag: %v", err)
	}
	if suggestions.Suggestions == nil {
		t.Error("die Vorschlagsliste ist nil statt leer")
	}

	state, err := b.GetMonthCloseState("2026-03")
	if err != nil {
		t.Fatalf("Monatsstand: %v", err)
	}
	if state.Steps == nil || state.Findings == nil {
		t.Error("die Listen des Monatsabschlusses sind nil statt leer")
	}
}
