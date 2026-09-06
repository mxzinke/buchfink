package wailsbridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 7 antwortet auch dann, wenn ihre Dienste fehlen.
//
// Der Fall ist nicht künstlich: zwischen dem Start und dem Öffnen eines
// Mandanten ist genau das der Zustand, und die Aufgabenliste ist der erste
// Bildschirm. Käme dort `null` an, nähme `null.map` im Render den ganzen Baum
// mit — und zwar bevor die Anwenderin etwas tun konnte.
func TestWelle7BridgeAnswersWithEmptyListsWithoutServices(t *testing.T) {
	b := testBridge(t)

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
	proposals, err := b.GetDunningProposals()
	if err != nil || proposals == nil {
		t.Errorf("Mahnvorschläge: %v (%v)", proposals, err)
	}
	notices, err := b.GetDunningNotices(0)
	if err != nil || notices == nil {
		t.Errorf("Mahnverlauf: %v (%v)", notices, err)
	}
	rates, err := b.GetBaseRates()
	if err != nil || rates == nil {
		t.Errorf("Basiszinssätze: %v (%v)", rates, err)
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

// Im Prüfermodus bleiben die lesenden Wege offen und die schreibenden zu.
func TestWelle7BridgeRespectsTheReadOnlyMode(t *testing.T) {
	b := testBridge(t)
	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2022 bis 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}

	if _, err := b.GetTasks(); err != nil {
		t.Errorf("die Aufgabenliste muss im Prüfermodus lesbar bleiben: %v", err)
	}
	if err := b.DeleteBankRule(1); err == nil || !strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("das Löschen einer Bankregel muss abgewiesen werden: %v", err)
	}
	if _, err := b.SaveBaseRate("2026-07-01", 152); err == nil ||
		!strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("das Nachtragen eines Basiszinssatzes muss abgewiesen werden: %v", err)
	}
	if _, err := b.SaveServiceProof(1, "geprüft", "2026-03-20"); err == nil ||
		!strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("der Leistungsnachweis muss abgewiesen werden: %v", err)
	}
	if _, err := b.CreateDunningNotices(service.DunningRunRequest{ContactIDs: []uint{1}}); err == nil ||
		!strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("das Erzeugen eines Mahnschreibens muss abgewiesen werden: %v", err)
	}
}
