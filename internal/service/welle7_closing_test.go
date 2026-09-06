package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Jahresabschluss als geführter Weg (Architektur 6.3) braucht einen
// Fortschritt: „Schritt n von 11". Er wird gerechnet und nicht in der Ansicht
// gezählt — der übersprungene Schritt ist abgeschlossen, aber nicht getan, und
// wer das zweimal entscheidet, entscheidet es zweimal verschieden.
func TestClosingStepsCountTheProgress(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	modules := env.closingModules(t)

	steps, err := modules.steps.Steps(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	if steps.Total != len(steps.Steps) || steps.Total == 0 {
		t.Fatalf("Gesamtzahl = %d bei %d Schritten", steps.Total, len(steps.Steps))
	}
	if steps.OpenCount+steps.DoneCount+steps.SkippedCount != steps.Total {
		t.Errorf("offen %d + erledigt %d + übersprungen %d ≠ %d",
			steps.OpenCount, steps.DoneCount, steps.SkippedCount, steps.Total)
	}
	before := steps.Progress()

	// Ein übergangener Baustein zählt zum Fortschritt: der Weg ist an dieser
	// Stelle zu Ende, auch wenn nichts gebucht wurde.
	var candidate domain.ClosingStepKey
	for _, step := range steps.Steps {
		if step.State == domain.ClosingStepOpen {
			candidate = step.Key
			break
		}
	}
	if candidate == "" {
		t.Skip("kein offener Baustein in diesem Bestand")
	}
	after, err := modules.steps.SkipStep(ctx, env.fiscalYear, candidate, "im ersten Jahr nicht einschlägig")
	if err != nil {
		t.Fatalf("Baustein übergehen: %v", err)
	}
	if after.SkippedCount < 1 {
		t.Errorf("übersprungene Schritte = %d, erwartet mindestens 1", after.SkippedCount)
	}
	if after.Progress() != before+1 {
		t.Errorf("Fortschritt = %d von %d, erwartet %d", after.Progress(), after.Total, before+1)
	}
}

// Der Weg zurück gehört zum geführten Weg (Architektur 6.3): eine übersprungene
// Arbeit lässt sich wieder aufnehmen, solange das Jahr nicht festgeschrieben
// ist. Ohne diesen Weg wäre das Überspringen die einzige Entscheidung, die
// niemand mehr korrigieren kann.
func TestSkippedClosingStepCanBeReopened(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	modules := env.closingModules(t)

	if _, err := modules.steps.SkipStep(ctx, env.fiscalYear, domain.ClosingStepProvisions,
		"im ersten Jahr keine ungewissen Verbindlichkeiten"); err != nil {
		t.Fatalf("Baustein übergehen: %v", err)
	}

	// Ohne Begründung nicht: die Rücknahme tritt an die Stelle eines Grundes,
	// der im Prüfbericht stand.
	if _, err := modules.steps.ReopenStep(ctx, env.fiscalYear, domain.ClosingStepProvisions, "  "); err == nil {
		t.Error("eine Rücknahme ohne Begründung darf nicht durchgehen")
	}

	steps, err := modules.steps.ReopenStep(ctx, env.fiscalYear, domain.ClosingStepProvisions,
		"Die Prozesskosten sind doch zurückzustellen")
	if err != nil {
		t.Fatalf("Baustein wieder aufnehmen: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key != domain.ClosingStepProvisions {
			continue
		}
		if step.State != domain.ClosingStepOpen {
			t.Errorf("der zurückgenommene Baustein steht auf %q, erwartet offen", step.State)
		}
		if step.Reason != "" {
			t.Errorf("der Grund des Überspringens bleibt stehen: %q", step.Reason)
		}
	}
	if steps.SkippedCount != 0 {
		t.Errorf("übersprungene Schritte = %d, erwartet 0", steps.SkippedCount)
	}

	// Was nicht übersprungen ist, ist auch nicht zurückzunehmen.
	if _, err := modules.steps.ReopenStep(ctx, env.fiscalYear, domain.ClosingStepInventory,
		"aus Versehen"); err == nil {
		t.Error("ein nicht übersprungener Baustein darf nicht zurückgenommen werden")
	}

	// Und das Protokoll nennt beide Gründe.
	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 200)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Details, "wieder aufgenommen") &&
			strings.Contains(entry.Details, "Prozesskosten") &&
			strings.Contains(entry.Details, "ungewissen Verbindlichkeiten") {
			found = true
		}
	}
	if !found {
		t.Errorf("das Protokoll nennt die Rücknahme nicht mit beiden Gründen: %+v", entries)
	}
}

// Nach der Jahres-Festschreibung ist der Weg zurück zu: was danach zu ändern
// ist, wird storniert (Architektur 6.3). Der Grund steht in der Schrittliste,
// damit die Oberfläche den gesperrten Knopf beschriften kann.
func TestReopenIsClosedAfterTheYearIsCommitted(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	modules := env.closingModules(t)

	if _, err := modules.steps.SkipStep(ctx, env.fiscalYear, domain.ClosingStepProvisions,
		"im ersten Jahr keine ungewissen Verbindlichkeiten"); err != nil {
		t.Fatalf("Baustein übergehen: %v", err)
	}
	before, err := modules.steps.Steps(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	if !before.Reopenable || before.ReopenBlocker != "" {
		t.Errorf("vor der Festschreibung ist die Rücknahme möglich: %+v", before)
	}

	if err := repository.NewFestschreibungRepository(env.db).Create(ctx, &domain.Festschreibung{
		FiscalYear: env.fiscalYear, PeriodType: "year",
		PeriodLabel: "Geschäftsjahr", CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	after, err := modules.steps.Steps(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	if after.Reopenable || after.ReopenBlocker == "" {
		t.Errorf("nach der Festschreibung ist die Rücknahme gesperrt und sagt warum: %+v", after)
	}
	if _, err := modules.steps.ReopenStep(ctx, env.fiscalYear, domain.ClosingStepProvisions,
		"doch noch"); err == nil {
		t.Error("nach der Festschreibung darf kein Schritt zurückgenommen werden")
	}
}
