package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

type fiscalCalendar struct {
	settingsRepo domain.SettingsRepository
	years        domain.FiscalYearRepository
	// Abschluss und Vorsteuerberichtigung begrenzen den Beginnmonat auf 1–12.
	limitMonth bool
}

func (c fiscalCalendar) fiscalYearStartMonth(ctx context.Context) int {
	if c.settingsRepo == nil {
		return 1
	}
	settings, err := c.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil || settings.FiscalYearStartMonth <= 0 {
		return 1
	}
	if c.limitMonth && settings.FiscalYearStartMonth > 12 {
		return 1
	}
	return settings.FiscalYearStartMonth
}

func (c fiscalCalendar) fiscalYearStart(ctx context.Context, year int) string {
	if c.years != nil {
		if fy, err := c.years.FindByYear(ctx, year); err == nil && fy != nil && fy.StartDate != "" {
			return fy.StartDate
		}
	}
	return fmt.Sprintf("%04d-%02d-01", year, c.fiscalYearStartMonth(ctx))
}

func (c fiscalCalendar) fiscalYearEnd(ctx context.Context, year int) string {
	if c.years != nil {
		if fy, err := c.years.FindByYear(ctx, year); err == nil && fy != nil && fy.EndDate != "" {
			return fy.EndDate
		}
	}
	return fiscalYearEndDate(year, c.fiscalYearStartMonth(ctx))
}
