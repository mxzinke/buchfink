package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kein `null` in den Listen, die an die Oberfläche gehen.
//
// Die Ansicht liest die Listen ohne Umweg — `figures.length`,
// `lateEntries.length`, `findings.length`. Ein nicht belegter Go-Slice wird in
// JSON zu `null`, und `null.length` wirft im Render einen TypeError; ohne
// ErrorBoundary nimmt das den ganzen Baum mit. Betroffen wäre jeweils der
// Regelfall: der Zeitraum ohne Nachtrag, die Meldung ohne Befund, der saubere
// Prüflauf. Die Zusage wird deshalb an der Ausgabe geprüft und nicht an den
// Feldern.
func assertNoNullLists(t *testing.T, label string, v any, keys ...string) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("%s als JSON: %v", label, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("%s zurücklesen: %v", label, err)
	}
	for _, key := range keys {
		value, ok := decoded[key]
		if !ok {
			t.Errorf("%s: das Feld %q fehlt in der Ausgabe — die Oberfläche läse `undefined`", label, key)
			continue
		}
		if value == nil {
			t.Errorf("%s: %q ist `null`, erwartet eine leere Liste:\n%s", label, key, raw)
			continue
		}
		if _, ok := value.([]any); !ok {
			t.Errorf("%s: %q ist keine Liste (%T)", label, key, value)
		}
	}
}

// Die leere Voranmeldung: kein Umsatz, kein Nachtrag.
func TestVatReturnMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	draft, err := env.vatReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	assertNoNullLists(t, "Voranmeldung (Entwurf)", draft, "figures", "lateEntries")

	saved, err := env.vatReturns(t).Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	// Auch aus der Datenbank zurückgelesen: gespeichert wird die Liste als JSON,
	// und eine gespeicherte „null" käme sonst als `null` wieder heraus.
	loaded, err := env.vatReturns(t).List(ctx, 2026)
	if err != nil || len(loaded) == 0 {
		t.Fatalf("gespeicherte Anmeldungen lesen: %v (%d)", err, len(loaded))
	}
	if loaded[0].ID != saved.ID {
		t.Fatalf("erwartet die gespeicherte Anmeldung %d, erhalten %d", saved.ID, loaded[0].ID)
	}
	assertNoNullLists(t, "Voranmeldung (gespeichert)", loaded[0], "figures", "lateEntries")
}

// Die leere Zusammenfassende Meldung: keine Zeile, kein Befund, kein Nachtrag.
func TestZMReturnMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	draft, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	assertNoNullLists(t, "Zusammenfassende Meldung (Entwurf)", draft, "lines", "lateEntries", "findings")

	saved, err := env.zmReturns(t).Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	assertNoNullLists(t, "Zusammenfassende Meldung (gespeichert)", saved, "lines", "lateEntries", "findings")
}

// Der saubere Prüflauf — genau der Fall, in dem der Festschreibungsdialog
// `findings.length` liest.
func TestCheckRunMarshalsEmptyFindingsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	preview, err := env.checks(t).Preview(ctx, CheckRequest{CutoffDate: "2026-01-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(preview.Findings) != 0 {
		t.Fatalf("für diesen Fall wird ein Lauf ohne Befund erwartet: %+v", preview.Findings)
	}
	assertNoNullLists(t, "Prüflauf (Vorschau)", preview, "findings")

	run, err := env.checks(t).Run(ctx, CheckRequest{CutoffDate: "2026-01-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	assertNoNullLists(t, "Prüflauf (gespeichert)", run, "findings")

	stored, err := env.checks(t).Runs(ctx, 2026)
	if err != nil || len(stored) == 0 {
		t.Fatalf("gespeicherte Läufe lesen: %v (%d)", err, len(stored))
	}
	assertNoNullLists(t, "Prüflauf (zurückgelesen)", stored[0], "findings")
}

// Die Fristenliste ist selbst die Liste: leer statt `null`, damit die Ansicht
// darüber laufen kann.
func TestDeadlineListMarshalsEmptyNotNull(t *testing.T) {
	env := newTestEnv(t)
	deadlines, err := env.deadlines(t).Deadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if deadlines == nil {
		t.Fatal("die Fristenliste darf nicht nil sein")
	}
	raw, err := json.Marshal(deadlines)
	if err != nil {
		t.Fatalf("Termine als JSON: %v", err)
	}
	if string(raw) == "null" {
		t.Error("die Fristenliste kommt als `null` in der Oberfläche an")
	}
}

// Die Entitäten sagen die leeren Listen selbst zu — an ihnen hängt die Zusage,
// nicht an den Stellen, die sie erzeugen.
func TestEnsureListsReplacesNilWithEmpty(t *testing.T) {
	var vat domain.VatReturn
	vat.EnsureLists()
	assertNoNullLists(t, "leere Voranmeldung", vat, "figures", "lateEntries")

	var zm domain.ZMReturn
	zm.EnsureLists()
	assertNoNullLists(t, "leere Zusammenfassende Meldung", zm, "lines", "lateEntries", "findings")

	var run domain.CheckRun
	run.EnsureLists()
	assertNoNullLists(t, "leerer Prüflauf", run, "findings")

	// Gegenprobe: ohne die Zusage steht dort `null`.
	raw, err := json.Marshal(domain.VatReturn{})
	if err != nil {
		t.Fatalf("Gegenprobe: %v", err)
	}
	if !strings.Contains(string(raw), `"figures":null`) {
		t.Errorf("die Gegenprobe trägt nicht mehr — der Test prüft dann nichts mehr: %s", raw)
	}
}
