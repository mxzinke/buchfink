package service

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/goldentest"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestGoldenPeriodPostingLocks(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, date, commit         string
		source                     domain.EntrySource
		closing, adopted, reversal bool
	}{
		{name: "offen", date: "2026-03-15"},
		{name: "monat_gesperrt", date: "2026-03-15", commit: "month"},
		{name: "jahr_gesperrt", date: "2026-12-31", commit: "year"},
		{name: "abschluss_nach_monat", date: "2026-12-31", commit: "month", source: domain.EntrySourceClosing},
		{name: "afa_nach_monat", date: "2026-12-31", commit: "month", source: domain.EntrySourceDepreciation},
		{name: "manueller_abschluss", date: "2026-12-31", commit: "month", closing: true},
		{name: "abschluss_falscher_tag", date: "2026-12-30", closing: true},
		{name: "abschluss_nach_jahr", date: "2026-12-31", commit: "year", closing: true},
		{name: "festgestellt", date: "2026-03-15", adopted: true},
		{name: "feststellung_vor_festschreibung", date: "2026-12-31", adopted: true, commit: "year", closing: true},
		{name: "generalumkehr_offen", date: "2026-03-15", reversal: true},
		{name: "generalumkehr_gesperrt", date: "2026-03-15", commit: "month", reversal: true},
		{name: "generalumkehr_festgestellt", date: "2026-03-15", adopted: true, reversal: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			closing := env.closing(t)
			fy, err := closing.YearOf(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			var original *domain.JournalEntry
			if tc.reversal {
				e := datedEntry("2026-01-01", "1800", "9000", 10000)
				e.Source = domain.EntrySourceOpening
				original, err = env.journal.Post(ctx, e)
				if err != nil {
					t.Fatal(err)
				}
			}
			if tc.commit != "" {
				if err := repository.NewFestschreibungRepository(env.db).Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: tc.commit, PeriodLabel: tc.commit, CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash}); err != nil {
					t.Fatal(err)
				}
			}
			if tc.adopted {
				fy.Status, fy.PreparedOn, fy.AdoptedOn = domain.FiscalYearAdopted, "2027-03-01", "2027-06-01"
				if err := repository.NewFiscalYearRepository(env.db).Save(ctx, fy); err != nil {
					t.Fatal(err)
				}
			}
			e := datedEntry(tc.date, "6815", "1800", 10000)
			e.ClosingPeriod = tc.closing
			if tc.source != "" {
				e.Source = tc.source
			}
			var posted *domain.JournalEntry
			if tc.reversal {
				posted, err = env.journal.ReverseOn(ctx, original.ID, "Korrektur", tc.date)
			} else {
				posted, err = env.journal.Post(ctx, e)
			}
			result := map[string]any{"error": ""}
			if err != nil {
				result["error"] = err.Error()
			}
			if posted != nil {
				result["year"], result["closing"], result["kind"] = posted.FiscalYear, posted.ClosingPeriod, posted.Kind
				lines := []map[string]any{}
				for _, l := range posted.Lines {
					lines = append(lines, map[string]any{"account": l.Account, "side": l.Side, "amount": l.Amount})
				}
				result["lines"] = lines
			}
			entries, err := env.journalRepo.FindAll(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			result["entryCount"] = len(entries)
			goldentest.Assert(t, "period_post_"+tc.name, result)
		})
	}
}

func TestGoldenPeriodAdoptionAndReopen(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name                       string
		disclose, empty, unadopted bool
	}{
		{name: "festgestellt"}, {name: "offengelegt", disclose: true}, {name: "grund_fehlt", empty: true}, {name: "nicht_festgestellt", unadopted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			closing := env.closing(t)
			commitments := repository.NewFestschreibungRepository(env.db)
			if err := commitments.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: "year", PeriodLabel: "2026", CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash}); err != nil {
				t.Fatal(err)
			}
			if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", ""); err != nil {
				t.Fatal(err)
			}
			if !tc.unadopted {
				if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-01", " Beschluss 1 "); err != nil {
					t.Fatal(err)
				}
			}
			if tc.disclose {
				if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearDisclosed, "2027-07-01", " Auftrag 2 "); err != nil {
					t.Fatal(err)
				}
			}
			reason := " Korrektur des Abschlusses "
			if tc.empty {
				reason = "  "
			}
			_, reopenErr := closing.ReopenFiscalYear(ctx, 2026, reason)
			fy, err := closing.PeriodOf(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			fy.CreatedAt = time.Time{}
			result := map[string]any{"year": fy, "error": ""}
			if reopenErr != nil {
				result["error"] = reopenErr.Error()
			}
			records, err := commitments.FindByFiscalYear(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			result["commitmentCount"] = len(records)
			_, postErr := env.journal.Post(ctx, datedEntry("2026-12-31", "6815", "1800", 10000))
			result["postingError"] = ""
			if postErr != nil {
				result["postingError"] = postErr.Error()
			}
			logs, err := repository.NewAuditRepository(env.db).FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "FISCAL_YEAR"})
			if err != nil {
				t.Fatal(err)
			}
			details := []string{}
			for _, log := range logs {
				details = append(details, log.Details)
			}
			result["audit"] = details
			goldentest.Assert(t, "period_reopen_"+tc.name, result)
		})
	}
}
