package core

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/timestamp"
)

func TestPeriodPreCommitChecks(t *testing.T) {
	h := newHostWith(t, Network{Timestamp: func(context.Context, string, string) (*timestamp.Result, error) {
		t.Error("Netz innerhalb einer Prüfung")
		return nil, ErrOffline
	}})
	newTenant(t, h, "Perioden GmbH")
	tenant := openTenant(t, h)
	if got, want := tenant.svc.Periods.PreCommitCheckNames(), []string{"legal_reserve", "committable"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Prüfreihenfolge: %v, erwartet %v", got, want)
	}
	calls := []string{}
	stop := errors.New("Prüfung abgebrochen")
	tenant.svc.Periods.AddPreCommitChecks(
		service.PreCommitCheck{Name: "first", Check: func(ctx context.Context, fy domain.FiscalYear, r service.PeriodCommitRequest) error {
			calls = append(calls, "first")
			if fy.Year != 2026 || fy.EndDate != "2026-12-31" || r.CutoffDate != "2026-03-31" {
				t.Fatalf("Prüfungszeitraum: %+v, %+v", fy, r)
			}
			runs, err := tenant.svc.Checks.Runs(ctx, 2026)
			if err != nil {
				return err
			}
			if len(runs) != 1 {
				t.Fatalf("Prüflauf fehlt innerhalb der Transaktion: %+v", runs)
			}
			if _, err := tenant.network.Timestamp(ctx, "tsa", domain.GenesisHash); !errors.Is(err, ErrNetworkInWrite) {
				t.Fatalf("Netz in Prüftransaktion: %v", err)
			}
			return tenant.repos.settings.Set(ctx, "period_check_marker", "in derselben Transaktion")
		}},
		service.PreCommitCheck{Name: "stop", Check: func(ctx context.Context, _ domain.FiscalYear, _ service.PeriodCommitRequest) error {
			calls = append(calls, "stop")
			value, err := tenant.repos.settings.Get(ctx, "period_check_marker")
			if err != nil {
				return err
			}
			if value != "in derselben Transaktion" {
				t.Fatalf("Prüfung sieht Vorgänger nicht: %q", value)
			}
			return stop
		}},
		service.PreCommitCheck{Name: "unreachable", Check: func(context.Context, domain.FiscalYear, service.PeriodCommitRequest) error {
			t.Fatal("Prüfung nach Abbruch")
			return nil
		}},
	)
	_, err := tenant.CommitPeriod(uiCtx, CommitRequest{PeriodType: "month", PeriodLabel: "März", CutoffDate: "2026-03-31", OverrideReason: "Testlauf"})
	if !errors.Is(err, stop) || !reflect.DeepEqual(calls, []string{"first", "stop"}) {
		t.Fatalf("Abbruch und Reihenfolge: %v, %v", calls, err)
	}
	marker, err := tenant.repos.settings.Get(uiCtx, "period_check_marker")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Prüfungsänderung nach Rollback: %q, %v", marker, err)
	}
	if marker != "" {
		t.Fatalf("Prüfungsänderung nicht zurückgerollt: %q", marker)
	}
	runs, err := tenant.svc.Checks.Runs(uiCtx, 2026)
	if err != nil || len(runs) != 0 {
		t.Fatalf("Prüflauf nach Abbruch: %+v, %v", runs, err)
	}
	records, err := tenant.repos.festschreibung.FindByFiscalYear(uiCtx, 2026)
	if err != nil || len(records) != 0 {
		t.Fatalf("Festschreibungen nach Abbruch: %+v, %v", records, err)
	}
}

func TestPeriodPreCommitRegistrationRejectsInvalidChecks(t *testing.T) {
	valid := service.PreCommitCheck{Name: "one", Check: func(context.Context, domain.FiscalYear, service.PeriodCommitRequest) error { return nil }}
	for _, tc := range []struct {
		name   string
		checks []service.PreCommitCheck
	}{
		{"ohne_name", []service.PreCommitCheck{{Check: valid.Check}}},
		{"ohne_funktion", []service.PreCommitCheck{{Name: "one"}}},
		{"doppelt", []service.PreCommitCheck{valid, valid}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("ungültige Prüfung registriert")
				}
			}()
			periods := &service.PeriodService{}
			periods.AddPreCommitChecks(tc.checks...)
		})
	}
}
