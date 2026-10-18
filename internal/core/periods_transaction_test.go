package core

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/timestamp"
)

func TestPeriodCommitAuditFailureRollsBack(t *testing.T) {
	for _, direct := range []bool{false, true} {
		name := "Kern"
		if direct {
			name = "Periodendienst"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			h := newHostWith(t, Network{Timestamp: func(context.Context, string, string) (*timestamp.Result, error) { calls++; return nil, ErrOffline }})
			newTenant(t, h, "Perioden GmbH")
			tenant := openTenant(t, h)
			if _, err := bookOn(uiCtx, tenant, "2026-03-01"); err != nil {
				t.Fatal(err)
			}
			if err := tenant.db.Exec(`CREATE TRIGGER kein_festschreibungsprotokoll BEFORE INSERT ON audit_log_entries
    WHEN NEW.entity_type = 'FESTSCHREIBUNG'
    BEGIN SELECT RAISE(ABORT, 'Festschreibungsprotokoll nicht verfügbar'); END`).Error; err != nil {
				t.Fatal(err)
			}
			req := CommitRequest{PeriodType: "month", PeriodLabel: "März", CutoffDate: "2026-03-31", OverrideReason: "Testbeleg"}
			var err error
			if direct {
				result, commitErr := tenant.svc.Periods.Commit(uiCtx, req)
				err = commitErr
				if result.Commitment != nil || result.Blocked != nil {
					t.Fatalf("zurückgerolltes Ergebnis: %+v", result)
				}
			} else {
				rec, commitErr := tenant.CommitPeriod(uiCtx, req)
				err = commitErr
				if rec != nil {
					t.Fatalf("zurückgerollte Festschreibung: %+v", rec)
				}
			}
			if err == nil || !strings.Contains(err.Error(), "Festschreibungsprotokoll nicht verfügbar") {
				t.Fatalf("Protokollfehler: %v", err)
			}
			if calls != 0 {
				t.Fatal("Zeitstempel für zurückgerollte Festschreibung angefordert")
			}
			entries, err := tenant.repos.journal.FindAll(uiCtx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].CommittedAt != nil || entries[0].FestschreibungID != nil {
				t.Fatalf("Buchungsstempel nach Rollback: %+v", entries)
			}
			records, err := tenant.repos.festschreibung.FindByFiscalYear(uiCtx, 2026)
			if err != nil || len(records) != 0 {
				t.Fatalf("Festschreibungen nach Rollback: %+v, %v", records, err)
			}
			runs, err := tenant.svc.Checks.Runs(uiCtx, 2026)
			if err != nil || len(runs) != 0 {
				t.Fatalf("Prüfläufe nach Rollback: %+v, %v", runs, err)
			}
			logs, err := tenant.svc.Repos.Audit.FindFiltered(uiCtx, 0, domain.AuditFilter{EntityType: "FESTSCHREIBUNG"})
			if err != nil || len(logs) != 0 {
				t.Fatalf("Protokoll nach Rollback: %+v, %v", logs, err)
			}
		})
	}
}
