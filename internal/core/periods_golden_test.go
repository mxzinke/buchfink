package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

func TestGoldenCommitPeriod(t *testing.T) {
	cases := []struct {
		name, period, cutoff, reason string
		prior                        *CommitRequest
		short, ug                    bool
	}{
		{name: "monat", period: "month", cutoff: "2026-03-31", reason: " Testbelege fehlen "},
		{name: "quartal", period: "quarter", cutoff: "2026-06-30", reason: "Testbelege fehlen"},
		{name: "jahr", period: "year", cutoff: "2026-12-31", reason: "Testbelege fehlen"},
		{name: "jahr_vor_bilanzstichtag", period: "year", cutoff: "2026-12-30", reason: "Testbelege fehlen"},
		{name: "gleicher_stichtag", period: "month", cutoff: "2026-06-30", prior: &CommitRequest{PeriodType: "quarter", PeriodLabel: "Q2", CutoffDate: "2026-06-30", OverrideReason: "Testbelege fehlen"}},
		{name: "frueherer_stichtag", period: "month", cutoff: "2026-03-31", prior: &CommitRequest{PeriodType: "quarter", PeriodLabel: "Q2", CutoffDate: "2026-06-30", OverrideReason: "Testbelege fehlen"}},
		{name: "jahr_bereits_festgeschrieben", period: "month", cutoff: "2026-12-31", prior: &CommitRequest{PeriodType: "year", PeriodLabel: "2026", CutoffDate: "2026-12-31", OverrideReason: "Testbelege fehlen"}},
		{name: "jahr_nach_dezember", period: "year", cutoff: "2026-12-31", reason: "Testbelege fehlen", prior: &CommitRequest{PeriodType: "month", PeriodLabel: "Dezember", CutoffDate: "2026-12-31", OverrideReason: "Testbelege fehlen"}},
		{name: "blockiert", period: "month", cutoff: "2026-03-31"},
		{name: "leere_begruendung", period: "month", cutoff: "2026-03-31", reason: "  "},
		{name: "ug_ruecklage_fehlt", period: "year", cutoff: "2026-12-31", reason: "Testbelege fehlen", ug: true},
		{name: "ug_ruecklage_letzter_monat", period: "month", cutoff: "2026-12-31", reason: "Testbelege fehlen", ug: true},
		{name: "rumpfjahr", period: "year", cutoff: "2026-06-30", reason: "Testbelege fehlen", short: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHost(t)
			newTenant(t, h, "Perioden GmbH")
			tenant := openTenant(t, h)
			if tc.short {
				fy, err := tenant.repos.fiscalYears.FindByYear(uiCtx, 2026)
				if err != nil {
					t.Fatal(err)
				}
				fy.StartDate, fy.EndDate, fy.IsShort = "2026-03-15", "2026-06-30", true
				if err := tenant.repos.fiscalYears.Save(uiCtx, fy); err != nil {
					t.Fatal(err)
				}
			}
			if tc.ug {
				if err := tenant.repos.settings.Set(uiCtx, "legal_form", domain.LegalFormUG); err != nil {
					t.Fatal(err)
				}
			}
			dates := []string{"2026-03-15", "2026-04-15", "2026-12-31"}
			if tc.short {
				dates[2] = "2026-06-30"
			}
			for _, date := range dates {
				if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
					debit, credit := "6815", "1800"
					if tc.ug {
						debit, credit = "1800", "4830"
					}
					_, err := s.Journal.Post(ctx, &domain.JournalEntry{BookingDate: date, Description: "Periodentest", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
						Lines: []domain.JournalLine{{Side: domain.SideDebit, Account: debit, Amount: 10000}, {Side: domain.SideCredit, Account: credit, Amount: 10000}}})
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			if tc.prior != nil {
				if _, err := tenant.CommitPeriod(uiCtx, *tc.prior); err != nil {
					t.Fatal(err)
				}
			}
			before, err := tenant.repos.journal.FindAll(uiCtx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			rec, commitErr := tenant.CommitPeriod(uiCtx, CommitRequest{PeriodType: tc.period, PeriodLabel: tc.name, CutoffDate: tc.cutoff, OverrideReason: tc.reason})
			var blocked *service.CommitBlocked
			result := map[string]any{"error": "", "blocked": errors.As(commitErr, &blocked)}
			if commitErr != nil {
				result["error"] = commitErr.Error()
			}
			records, err := tenant.repos.festschreibung.FindByFiscalYear(uiCtx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			for i := range records {
				r := &records[i]
				if r.ChainHead != before[len(before)-1].EntryHash || r.Actor != actor.Actor() || r.AppVersion != buildinfo.Version || r.CreatedAt.IsZero() {
					t.Fatalf("Festschreibungsstempel: %+v", r)
				}
				if rec != nil && rec.ID == r.ID && !reflect.DeepEqual(rec, r) {
					t.Fatalf("Rückgabe weicht vom Datensatz ab: %+v / %+v", rec, r)
				}
			}
			after, err := tenant.repos.journal.FindAll(uiCtx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			stamps := []map[string]any{}
			for i, e := range after {
				stamps = append(stamps, map[string]any{"date": e.BookingDate, "commitment": e.FestschreibungID, "stamped": e.CommittedAt != nil})
				if e.CommittedAt != nil {
					found := false
					for _, r := range records {
						if e.FestschreibungID != nil && r.ID == *e.FestschreibungID {
							found = e.CommittedAt.Equal(r.CreatedAt)
						}
					}
					if !found {
						t.Fatal("Buchungsstempel stimmt nicht mit der Festschreibung überein")
					}
				}
				e.CommittedAt, e.FestschreibungID = before[i].CommittedAt, before[i].FestschreibungID
				if !reflect.DeepEqual(e, before[i]) {
					t.Fatalf("Festschreibung hat Buchung %d außer ihrem Stempel verändert", e.ID)
				}
			}
			for i := range records {
				records[i].CreatedAt = time.Time{}
				records[i].Actor = "<Bearbeiter>"
				records[i].AppVersion = "<Version>"
				records[i].ChainHead = "<letzter Buchungshash>"
			}
			result["commitments"], result["entries"] = records, stamps
			audit, err := tenant.svc.Repos.Audit.FindFiltered(uiCtx, 0, domain.AuditFilter{EntityType: "FESTSCHREIBUNG"})
			if err != nil {
				t.Fatal(err)
			}
			logs := []map[string]any{}
			for _, a := range audit {
				logs = append(logs, map[string]any{"action": a.Action, "entity": a.EntityType, "id": a.EntityID, "details": a.Details, "channel": a.Channel})
			}
			result["audit"] = logs
			runs, err := tenant.svc.Checks.Runs(uiCtx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			checks := []map[string]any{}
			for _, r := range runs {
				checks = append(checks, map[string]any{"cutoff": r.CutoffDate, "blocking": r.HasBlocking(), "override": r.OverrideReason})
			}
			result["checks"] = checks
			assertPeriodGolden(t, tc.name, result)
		})
	}
}

func assertPeriodGolden(t *testing.T, name string, value any) {
	t.Helper()
	got, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "periods", name+".json")
	if os.Getenv("UPDATE_PERIOD_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("%s: erwartet\n%s\nerhalten\n%s", path, want, got)
	}
}
