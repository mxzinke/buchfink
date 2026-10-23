package service

import (
	"context"
	"errors"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestPeriodsState(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	closing := env.closing(t)
	periods := closing.periods
	years := repository.NewFiscalYearRepository(env.db)
	if err := years.Save(ctx, domain.NewFiscalYear(2026, "2026-03-15", "2026-06-30")); err != nil {
		t.Fatal(err)
	}
	commitments := repository.NewFestschreibungRepository(env.db)
	if err := commitments.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: "month", CutoffDate: "2026-03-31"}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		date          string
		open, closing bool
	}{
		{"2026-03-15", false, false}, {"2026-04-01", true, false}, {"2026-06-30", true, true},
	} {
		state, err := periods.State(ctx, tc.date)
		if err != nil {
			t.Fatal(err)
		}
		if state.FiscalYear != 2026 || state.EndDate != "2026-06-30" || state.Open != tc.open || state.ClosingPeriod != tc.closing || state.ClosingOpen != tc.closing || state.CommittedThrough != "2026-03-31" || state.AdoptedYear != nil {
			t.Fatalf("%s: %+v", tc.date, state)
		}
	}
	if err := commitments.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: "month", CutoffDate: "2026-06-30"}); err != nil {
		t.Fatal(err)
	}
	state, err := periods.State(ctx, "2026-06-30")
	if err != nil || state.Open || !state.ClosingOpen {
		t.Fatalf("Abschlussperiode: %+v, %v", state, err)
	}
	if err := commitments.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: "year", CutoffDate: "2026-06-30"}); err != nil {
		t.Fatal(err)
	}
	state, err = periods.State(ctx, "2026-06-30")
	if err != nil || state.Open || state.ClosingOpen {
		t.Fatalf("Jahresfestschreibung: %+v, %v", state, err)
	}
	fy, err := periods.FiscalYear(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if !fy.IsShort || fy.StartDate != "2026-03-15" || fy.EndDate != "2026-06-30" {
		t.Fatalf("Rumpfjahr: %+v", fy)
	}
	fy.Status, fy.PreparedOn, fy.AdoptedOn = domain.FiscalYearAdopted, "2026-07-01", "2026-08-01"
	if err := years.Save(ctx, fy); err != nil {
		t.Fatal(err)
	}
	env.setSetting(t, "fiscal_year_start_month", "7")
	state, err = periods.State(ctx, "2026-03-15")
	if err != nil || state.FiscalYear != 2025 || state.Open || state.AdoptedYear == nil || state.AdoptedYear.Year != 2026 {
		t.Fatalf("Feststellung nach geändertem Beginnmonat: %+v, %v", state, err)
	}
}

func TestPeriodsFiscalYearDerivesWithoutWriting(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	closing := env.closing(t)
	env.setSettings(t, func(c *domain.CompanySettings) { c.FiscalYearStartMonth = 7; c.FoundedOn = "2026-09-17" })
	fy, err := closing.periods.FiscalYear(ctx, 2026)
	if err != nil || fy.StartDate != "2026-09-17" || fy.EndDate != "2027-06-30" || !fy.IsShort {
		t.Fatalf("abgeleitetes Rumpfjahr: %+v, %v", fy, err)
	}
	stored, err := repository.NewFiscalYearRepository(env.db).FindByYear(ctx, 2026)
	if err != nil || stored != nil {
		t.Fatalf("lesende Auskunft hat ein Jahr angelegt: %+v, %v", stored, err)
	}
}

type failingPeriodCommitments struct {
	domain.FestschreibungRepository
	err error
}

func (r failingPeriodCommitments) FindCutoffOnOrAfter(context.Context, string) ([]domain.Festschreibung, error) {
	return nil, r.err
}

func TestPeriodsReadFailureKeepsAdoptionPriority(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	closing := env.closing(t)
	fy, err := closing.YearOf(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	fy.Status, fy.PreparedOn, fy.AdoptedOn = domain.FiscalYearAdopted, "2027-03-01", "2027-06-01"
	if err := repository.NewFiscalYearRepository(env.db).Save(ctx, fy); err != nil {
		t.Fatal(err)
	}
	unavailable := errors.New("Festschreibungen nicht lesbar")
	env.journal.SetFestschreibungRepo(failingPeriodCommitments{err: unavailable})
	_, err = env.journal.Post(ctx, datedEntry("2026-03-15", "6815", "1800", 10000))
	if err == nil || errors.Is(err, unavailable) {
		t.Fatalf("Feststellung hat keinen Vorrang: %v", err)
	}
	fy.Status = domain.FiscalYearPrepared
	fy.AdoptedOn = ""
	if err := repository.NewFiscalYearRepository(env.db).Save(ctx, fy); err != nil {
		t.Fatal(err)
	}
	_, err = env.journal.Post(ctx, datedEntry("2026-03-15", "6815", "1800", 10000))
	if !errors.Is(err, unavailable) {
		t.Fatalf("Lesefehler öffnet die Periode: %v", err)
	}
}
