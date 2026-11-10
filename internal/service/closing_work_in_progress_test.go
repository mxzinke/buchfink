package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Projekt läuft über den Stichtag: Herstellungskosten 13.500 €, erhaltene
// Anzahlung 10.000 € netto auf 3272. Aktiviert wird der Bestand, die Anzahlung
// wird von den Vorräten offen abgesetzt; mit dem Saldenvortrag wird beides am
// 01.01. aufgelöst, und die Anzahlung steht wieder auf ihrem Konto, wo die
// Schlussrechnung sie findet.
func TestWorkInProgressIsCapitalisedAndReversedWithTheCarryForward(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	m.closing.AddOpeningReverser(m.bookings.WorkInProgressReverser())
	ctx := context.Background()
	env.post(t, "2026-09-30", domain.AccountBank, accounting.AccountErhalteneAnzahlungen19, 1_000_000)

	req := WorkInProgressRequest{
		FiscalYear: 2026, DeductAdvances: true,
		Projects: []accounting.WorkInProgressProject{{
			Name: "Onlineshop Muster GmbH", DirectCosts: 1_200_000, Overheads: 150_000,
			ContractValue: 3_000_000, CostsToComplete: 1_000_000, AdvancesReceived: 1_000_000,
		}},
	}
	preview, err := m.bookings.PreviewWorkInProgress(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Total != 1_350_000 || len(preview.Lines) != 4 || preview.ReversalDate != "2027-01-01" {
		t.Fatalf("Vorschau: Bestand %s €, %d Zeilen, Auflösung %s", preview.Total, len(preview.Lines), preview.ReversalDate)
	}
	too := req
	too.Projects = []accounting.WorkInProgressProject{{
		Name: "Onlineshop Muster GmbH", DirectCosts: 2_000_000, AdvancesReceived: 1_500_000,
	}}
	if _, err := m.bookings.PreviewWorkInProgress(ctx, too); err == nil {
		t.Error("abgesetzt wurde mehr, als als Anzahlung gebucht ist")
	}

	if _, err := m.bookings.BookWorkInProgress(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := m.bookings.BookWorkInProgress(ctx, req); err == nil {
		t.Error("ein zweiter Lauf hätte den Bestand ein zweites Mal aktiviert")
	}
	year := balances(t, env, 2026)
	if year[accounting.AccountUnfertigeLeistungen] != 1_350_000 ||
		year[accounting.AccountBestandsveraenderungUL] != -1_350_000 ||
		year[accounting.AccountErhalteneAnzahlungen19] != 0 ||
		year[accounting.AccountAnzahlungenVonVorraetenOffen] != -1_000_000 {
		t.Errorf("Bestände 2026: %v", year)
	}
	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps.Steps {
		if step.Key == domain.ClosingStepWorkInProgress && step.State != domain.ClosingStepDone {
			t.Errorf("Schritt unfertige Leistungen: %s", step.State)
		}
	}

	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	again, err := m.bookings.WorkInProgressReverser().ReverseInto(ctx, 2027)
	if err != nil || len(again) != 0 {
		t.Fatalf("zweite Auflösung: %d Buchungen, %v", len(again), err)
	}
	next := balances(t, env, 2027)
	if next[accounting.AccountUnfertigeLeistungen] != 0 || next[accounting.AccountAnzahlungenVonVorraetenOffen] != 0 {
		t.Errorf("nach der Auflösung: 1080 %s €, 1190 %s €",
			next[accounting.AccountUnfertigeLeistungen], next[accounting.AccountAnzahlungenVonVorraetenOffen])
	}
	if next[accounting.AccountErhalteneAnzahlungen19] != -1_000_000 ||
		next[accounting.AccountBestandsveraenderungUL] != 1_350_000 {
		t.Errorf("Anzahlung %s €, Bestandsminderung %s € — ein zweiter Vortrag darf nicht zweimal auflösen",
			next[accounting.AccountErhalteneAnzahlungen19], next[accounting.AccountBestandsveraenderungUL])
	}
}
