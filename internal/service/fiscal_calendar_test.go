package service

import (
	"context"
	"errors"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

type calendarSettings struct {
	domain.SettingsRepository
	value *domain.CompanySettings
	err   error
}

func (r calendarSettings) GetCompanySettings(context.Context) (*domain.CompanySettings, error) {
	return r.value, r.err
}

type calendarYears struct {
	domain.FiscalYearRepository
	value *domain.FiscalYear
	err   error
}

func (r calendarYears) FindByYear(context.Context, int) (*domain.FiscalYear, error) {
	return r.value, r.err
}

func TestFiscalCalendarStartMonth(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo domain.SettingsRepository
		want int
	}{
		{"ohne Repository", nil, 1},
		{"ohne Einstellungen", calendarSettings{}, 1},
		{"Lesefehler", calendarSettings{err: errors.New("lesen")}, 1},
		{"Wert trotz Fehler", calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: 7}, err: errors.New("lesen")}, 1},
		{"Null", calendarSettings{value: &domain.CompanySettings{}}, 1},
		{"negativ", calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: -1}}, 1},
		{"Januar", calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: 1}}, 1},
		{"Juli", calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: 7}}, 7},
		{"Dezember", calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: 12}}, 12},
		{"über zwölf", calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: 13}}, 13},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for name, read := range map[string]func(context.Context) int{
				"Kalender":    (fiscalCalendar{settingsRepo: tc.repo}).fiscalYearStartMonth,
				"Journal":     NewJournalService(nil, nil, nil, nil, tc.repo, nil).fiscalYearStartMonth,
				"Accrual":     NewAccrualService(nil, nil, nil, tc.repo, nil, nil).fiscalYearStartMonth,
				"Asset":       NewAssetService(nil, nil, NewJournalService(nil, nil, nil, nil, nil, nil), nil, nil, tc.repo, nil).fiscalYearStartMonth,
				"Invoice":     NewInvoiceService(nil, nil, tc.repo, nil, nil, nil).fiscalYearStartMonth,
				"Provision":   NewProvisionService(nil, nil, nil, nil, tc.repo, nil, nil).fiscalYearStartMonth,
				"TaxRegister": NewTaxRegisterService(nil, nil, nil, tc.repo, nil).fiscalYearStartMonth,
			} {
				if got := read(context.Background()); got != tc.want {
					t.Errorf("%s: %d, erwartet %d", name, got, tc.want)
				}
			}
			want := tc.want
			if want > 12 {
				want = 1
			}
			for name, read := range map[string]func(context.Context) int{
				"Kalender begrenzt": (fiscalCalendar{settingsRepo: tc.repo, limitMonth: true}).fiscalYearStartMonth,
				"Closing":           NewClosingService(nil, nil, nil, nil, nil, tc.repo, nil, nil, nil).fiscalYearStartMonth,
				"InputTax":          NewInputTaxService(nil, nil, nil, tc.repo, nil, nil).fiscalYearStartMonth,
			} {
				if got := read(context.Background()); got != want {
					t.Errorf("%s: %d, erwartet %d", name, got, want)
				}
			}
		})
	}
}

func TestFiscalCalendarBounds(t *testing.T) {
	for _, tc := range []struct {
		name       string
		month      int
		years      domain.FiscalYearRepository
		start, end string
	}{
		{"Kalenderjahr", 1, nil, "2024-01-01", "2024-12-31"},
		{"abweichendes Jahr", 7, nil, "2024-07-01", "2025-06-30"},
		{"Schaltjahr", 3, nil, "2024-03-01", "2025-02-28"},
		{"fehlendes Jahr", 7, calendarYears{}, "2024-07-01", "2025-06-30"},
		{"Lesefehler", 7, calendarYears{err: errors.New("lesen")}, "2024-07-01", "2025-06-30"},
		{"Rumpfjahr", 1, calendarYears{value: &domain.FiscalYear{StartDate: "2024-04-17", EndDate: "2024-09-30"}}, "2024-04-17", "2024-09-30"},
		{"angelegtes abweichendes Jahr", 1, calendarYears{value: &domain.FiscalYear{StartDate: "2024-07-01", EndDate: "2025-06-30"}}, "2024-07-01", "2025-06-30"},
		{"leerer Beginn", 7, calendarYears{value: &domain.FiscalYear{EndDate: "2024-12-31"}}, "2024-07-01", "2024-12-31"},
		{"leeres Ende", 7, calendarYears{value: &domain.FiscalYear{StartDate: "2024-08-13"}}, "2024-08-13", "2025-06-30"},
		{"leere Grenzen", 7, calendarYears{value: &domain.FiscalYear{}}, "2024-07-01", "2025-06-30"},
		{"ungültiger Monat", 13, nil, "2024-13-01", "2024-12-31"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := calendarSettings{value: &domain.CompanySettings{FiscalYearStartMonth: tc.month}}
			calendar := fiscalCalendar{settingsRepo: settings, fiscalYearRepo: tc.years}
			journal := NewJournalService(nil, nil, nil, nil, settings, nil)
			journal.SetFiscalYearRepo(tc.years)
			for name, reader := range map[string]interface {
				fiscalYearStart(context.Context, int) string
				fiscalYearEnd(context.Context, int) string
			}{"Kalender": calendar, "Journal": journal.periods} {
				if got := reader.fiscalYearStart(context.Background(), 2024); got != tc.start {
					t.Errorf("%s Beginn: %s, erwartet %s", name, got, tc.start)
				}
				if got := reader.fiscalYearEnd(context.Background(), 2024); got != tc.end {
					t.Errorf("%s Ende: %s, erwartet %s", name, got, tc.end)
				}
			}
		})
	}
}
