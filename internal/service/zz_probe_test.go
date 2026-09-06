package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestProbeNilLists(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	closing := m.closing

	state, err := closing.ClosingStateFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Abschlussstand: %v", err)
	}
	assertNoNilSlices(t, "Abschlussstand", state)

	cf, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortrag: %v", err)
	}
	assertNoNilSlices(t, "Saldenvortrag", cf)

	years, err := closing.FiscalYears(ctx)
	if err != nil {
		t.Fatalf("Geschäftsjahre: %v", err)
	}
	assertNoNilSlices(t, "Geschäftsjahre", years)

	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Abschlussschritte: %v", err)
	}
	assertNoNilSlices(t, "Abschlussschritte", steps)

	provisions, err := m.provisions.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungen: %v", err)
	}
	assertNoNilSlices(t, "Rückstellungen", provisions)

	mirror, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungsspiegel: %v", err)
	}
	assertNoNilSlices(t, "Rückstellungsspiegel", mirror)

	preview, err := m.provisions.Preview(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Rückbau", Amount: 100000, ExpectedOn: "2027-06-30",
		Reason: "Mietvertrag",
	})
	if err != nil {
		t.Fatalf("Rückstellungsvorschau: %v", err)
	}
	assertNoNilSlices(t, "Rückstellungsvorschau", preview)

	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Abgrenzungsvorschlag: %v", err)
	}
	assertNoNilSlices(t, "Abgrenzungsvorschlag", proposal)

	accruals, err := m.accruals.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Abgrenzungen: %v", err)
	}
	assertNoNilSlices(t, "Abgrenzungen", accruals)

	reg, err := m.register.Register(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	assertNoNilSlices(t, "Verzeichnis", reg)

	rec, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitung: %v", err)
	}
	assertNoNilSlices(t, "Überleitung", rec)

	app, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{})
	if err != nil {
		t.Fatalf("Ergebnisverwendung: %v", err)
	}
	assertNoNilSlices(t, "Ergebnisverwendung", app)

	periods, err := env.vatReturns(t).Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Voranmeldungszeiträume: %v", err)
	}
	assertNoNilSlices(t, "Voranmeldungszeiträume", periods)

	zmPeriods, err := env.zmReturns(t).Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Meldezeiträume: %v", err)
	}
	assertNoNilSlices(t, "Meldezeiträume", zmPeriods)

	deadlines, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Fristen: %v", err)
	}
	assertNoNilSlices(t, "Fristen", deadlines)

	proposals, err := env.dunning(t, nil).Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	assertNoNilSlices(t, "Mahnvorschläge", proposals)

	size, err := env.statements(t).SizeClassFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Größenklasse: %v", err)
	}
	assertNoNilSlices(t, "Größenklasse", size)

	tasks, err := NewTaskService(repository.NewSettingsRepository(env.db), 2026).
		Tasks(ctx, TaskOptions{Today: "2026-04-17"})
	if err != nil {
		t.Fatalf("Aufgaben: %v", err)
	}
	assertNoNilSlices(t, "Aufgaben", tasks)
}
