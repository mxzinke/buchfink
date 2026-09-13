package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

type testMonthYearSource struct{ year domain.FiscalYear }

func (s testMonthYearSource) FindByYear(context.Context, int) (*domain.FiscalYear, error) {
	return &s.year, nil
}

func TestFiscalYearMonthsRespectBounds(t *testing.T) {
	for _, tc := range []struct {
		name, start, end, first, last string
		count                         int
	}{
		{"Rumpfjahr", "2026-08-15", "2026-12-31", "2026-08", "2026-12", 5},
		{"Abweichendes Jahr", "2026-07-01", "2027-06-30", "2026-07", "2027-06", 12},
		{"Kalenderjahr", "2026-01-01", "2026-12-31", "2026-01", "2026-12", 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := fiscalYearMonths(context.Background(), testMonthYearSource{domain.FiscalYear{Year: 2026, StartDate: tc.start, EndDate: tc.end}}, 2026)
			if len(got) != tc.count || got[0].Key != tc.first || got[len(got)-1].Key != tc.last {
				t.Fatalf("Monate: %+v", got)
			}
		})
	}
}
