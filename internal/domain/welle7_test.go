package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// Die Aufgabenliste geht als JSON an die Oberfläche. Ihre drei Gruppen und die
// Parameter jedes Ziels müssen belegt sein — `null.map` nimmt im Render den
// ganzen Baum mit, und die Aufgabenliste ist der erste Bildschirm.
func TestTaskListMarshalsEmptyListsNotNull(t *testing.T) {
	list := &TaskList{Today: "2026-03-10"}
	list.EnsureLists()
	raw, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("Aufgabenliste als JSON: %v", err)
	}
	for _, key := range []string{`"overdue":[]`, `"open":[]`, `"upcoming":[]`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("%s fehlt in der Ausgabe:\n%s", key, raw)
		}
	}

	list.Add(Task{Key: "a", Group: TaskGroupOpen, Title: "Ohne Ziel"})
	list.EnsureLists()
	raw, err = json.Marshal(list)
	if err != nil {
		t.Fatalf("Aufgabenliste als JSON: %v", err)
	}
	if !strings.Contains(string(raw), `"params":{}`) {
		t.Errorf("die Parameter des Ziels sind `null`:\n%s", raw)
	}
}

// Die Sortierung: Fristen zuerst, die früheste oben, dann die Aufgaben ohne
// Frist nach Schlüssel.
func TestTaskListSortsDueDatesFirst(t *testing.T) {
	list := &TaskList{}
	list.Add(Task{Key: "zzz", Group: TaskGroupOpen, Title: "ohne Frist"})
	list.Add(Task{Key: "spaet", Group: TaskGroupOpen, DueDate: "2026-05-01"})
	list.Add(Task{Key: "aaa", Group: TaskGroupOpen, Title: "ohne Frist"})
	list.Add(Task{Key: "frueh", Group: TaskGroupOpen, DueDate: "2026-04-01"})
	list.Sort()

	got := []string{list.Open[0].Key, list.Open[1].Key, list.Open[2].Key, list.Open[3].Key}
	want := []string{"frueh", "spaet", "aaa", "zzz"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Reihenfolge = %v, erwartet %v", got, want)
		}
	}
	if list.Total() != 4 {
		t.Errorf("Gesamtzahl = %d, erwartet 4", list.Total())
	}
}

// Ein Zahlungsziel über sechzig Tagen löst den Hinweis auf § 271a BGB aus —
// und darunter keinen.
func TestPaymentTermNoticeOnlyAboveSixtyDays(t *testing.T) {
	if notice := PaymentTermNotice(60); notice != "" {
		t.Errorf("sechzig Tage sind unauffällig: %q", notice)
	}
	notice := PaymentTermNotice(90)
	if notice == "" {
		t.Fatal("neunzig Tage müssen den Hinweis auslösen")
	}
	if !strings.Contains(notice, "271a") {
		t.Errorf("der Hinweis nennt die Norm nicht: %q", notice)
	}
	if got := (PaymentTerms{DueDays: 90}).LongTermNotice(); got != notice {
		t.Errorf("die Bedingungen liefern einen anderen Hinweis: %q", got)
	}
}
