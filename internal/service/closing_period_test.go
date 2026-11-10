package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Abschlussperiode („13. Monat"): nach der Festschreibung des letzten
// Monats nimmt das Jahr noch Abschlussbuchungen zum Bilanzstichtag auf, bis es
// als Ganzes festgeschrieben ist — gewöhnliche Buchungen nicht.
func TestClosingPeriodStaysOpenUntilTheYearIsCommitted(t *testing.T) {
	env := newTestEnv(t)
	env.closing(t) // verdrahtet Geschäftsjahre und Festschreibungen am Journal
	ctx := context.Background()
	commitments := repository.NewFestschreibungRepository(env.db)
	commit := func(periodType string) {
		t.Helper()
		if err := commitments.Create(ctx, &domain.Festschreibung{
			FiscalYear: 2026, PeriodType: periodType, PeriodLabel: periodType,
			CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	entry := func(date string, source domain.EntrySource, closingPeriod bool) *domain.JournalEntry {
		e := simpleEntry("6815", "1800", 10000)
		e.BookingDate, e.DocumentDate, e.ServiceDateFrom, e.ServiceDateTo = date, date, date, date
		e.Source = source
		e.ClosingPeriod = closingPeriod
		return e
	}

	// Vor jeder Festschreibung gehört eine Abschlussbuchung zum Stichtag schon
	// zur Abschlussperiode.
	early, err := env.journal.Post(ctx, entry("2026-12-31", domain.EntrySourceClosing, false))
	if err != nil {
		t.Fatal(err)
	}
	if !early.ClosingPeriod {
		t.Error("eine Abschlussbuchung zum Bilanzstichtag gehört zur Abschlussperiode")
	}
	if _, err := env.journal.Post(ctx, entry("2026-12-30", domain.EntrySourceManual, true)); err == nil ||
		!strings.Contains(err.Error(), "Bilanzstichtag") {
		t.Errorf("eine Buchung der Abschlussperiode vor dem Stichtag: %v", err)
	}

	commit("month")
	_, err = env.journal.Post(ctx, entry("2026-12-31", domain.EntrySourceManual, false))
	if err == nil {
		t.Fatal("eine gewöhnliche Buchung in den festgeschriebenen Dezember")
	}
	if !strings.Contains(err.Error(), "Abschlussperiode") {
		t.Errorf("die Meldung nennt den Weg über die Abschlussperiode nicht: %v", err)
	}
	closing, err := env.journal.Post(ctx, entry("2026-12-31", domain.EntrySourceClosing, false))
	if err != nil {
		t.Fatalf("Abschlussbuchung nach der Festschreibung des Dezembers: %v", err)
	}
	if !closing.ClosingPeriod {
		t.Error("die Abschlussbuchung trägt das Kennzeichen der Abschlussperiode nicht")
	}
	if _, err := env.journal.Post(ctx, entry("2026-12-31", domain.EntrySourceManual, true)); err != nil {
		t.Errorf("Handbuchung der Abschlussperiode nach der Festschreibung des Dezembers: %v", err)
	}

	commit(domain.PeriodTypeYear)
	if _, err := env.journal.Post(ctx, entry("2026-12-31", domain.EntrySourceClosing, false)); err == nil {
		t.Error("nach der Jahresfestschreibung nimmt auch die Abschlussperiode nichts mehr auf")
	}
}

// Die Abschlussperiode richtet sich nach dem Bilanzstichtag des Jahres, auch
// beim abweichenden Geschäftsjahr.
func TestClosingPeriodFollowsAShiftedYearEnd(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.FiscalYearStartMonth = 7 })
	e := simpleEntry("6815", "1800", 10000)
	e.BookingDate, e.DocumentDate, e.ServiceDateFrom, e.ServiceDateTo = "2026-06-30", "2026-06-30", "2026-06-30", "2026-06-30"
	e.Source = domain.EntrySourceClosing
	posted, err := env.journal.Post(ctx, e)
	if err != nil {
		t.Fatal(err)
	}
	if posted.FiscalYear != 2025 || !posted.ClosingPeriod {
		t.Errorf("Abschlussbuchung zum 30.06.2026: Jahr %d, Abschlussperiode %v", posted.FiscalYear, posted.ClosingPeriod)
	}
}
