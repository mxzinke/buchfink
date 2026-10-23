package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// fiscalCalendar liest Beginn und Ende eines Geschäftsjahres aus den
// angelegten Jahren und dem Beginnmonat der Stammdaten.
type fiscalCalendar struct {
	settingsRepo   domain.SettingsRepository
	fiscalYearRepo domain.FiscalYearRepository
	// Abschluss, Vorsteuerberichtigung und Geschenke begrenzen den Monat auf 1–12.
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

// fiscalYearStart ist der erste Tag eines Geschäftsjahres: der Beginn des
// angelegten Jahres, sonst der nach dem Beginnmonat.
func (c fiscalCalendar) fiscalYearStart(ctx context.Context, year int) string {
	if c.fiscalYearRepo != nil {
		if fy, err := c.fiscalYearRepo.FindByYear(ctx, year); err == nil && fy != nil && fy.StartDate != "" {
			return fy.StartDate
		}
	}
	return fmt.Sprintf("%04d-%02d-01", year, c.fiscalYearStartMonth(ctx))
}

// fiscalYearEnd ist der Bilanzstichtag eines Geschäftsjahres: das Ende des
// angelegten Jahres, ein Rumpfjahr eingeschlossen, sonst das Ende nach dem
// Beginnmonat.
func (c fiscalCalendar) fiscalYearEnd(ctx context.Context, year int) string {
	if c.fiscalYearRepo != nil {
		if fy, err := c.fiscalYearRepo.FindByYear(ctx, year); err == nil && fy != nil && fy.EndDate != "" {
			return fy.EndDate
		}
	}
	return fiscalYearEndDate(year, c.fiscalYearStartMonth(ctx))
}

func fiscalYearEndDate(fiscalYear, startMonth int) string {
	if startMonth <= 1 || startMonth > 12 {
		return fmt.Sprintf("%d-12-31", fiscalYear)
	}
	start := time.Date(fiscalYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	return start.AddDate(1, 0, -1).Format("2006-01-02")
}
