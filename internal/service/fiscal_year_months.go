package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// fiscalYearMonths enthält auch den angefangenen ersten Monat eines Rumpfjahres.
func fiscalYearMonths(ctx context.Context, years MonthFiscalYearSource, year int) []accounting.VatPeriod {
	if years != nil {
		fy, err := years.FindByYear(ctx, year)
		if err == nil && fy != nil && len(fy.StartDate) == 10 && len(fy.EndDate) == 10 {
			out := make([]accounting.VatPeriod, 0, 12)
			for calendarYear := yearOfDate(fy.StartDate); calendarYear <= yearOfDate(fy.EndDate); calendarYear++ {
				for _, month := range accounting.VatPeriodsOfYear(calendarYear, domain.VatPeriodMonth) {
					if month.To >= fy.StartDate && month.From <= fy.EndDate {
						out = append(out, month)
					}
				}
			}
			return out
		}
	}
	return accounting.VatPeriodsOfYear(year, domain.VatPeriodMonth)
}
