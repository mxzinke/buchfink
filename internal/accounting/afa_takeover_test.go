package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein im Juli 2024 angeschaffter Pkw (12.000 €, 48 Monate) kommt zum 01.01.2026
// mit 4.500 € aufgelaufener AfA aus dem Vorsystem. Buchfink schreibt den
// Restbuchwert von 7.500 € über die 30 Restmonate ab: 3.000 € je Jahr, im
// letzten Jahr den Rest.
func TestTakeoverContinuesLinearScheduleFromCarriedBookValue(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2024-07-10",
		Cost:                 1_200_000,
		UsefulLifeMonths:     48,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		TakeoverYear:         2026,
		TakeoverDepreciation: 450_000,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("erwartet die Jahre 2026 bis 2028, bekommen %d Zeilen: %+v", len(rows), rows)
	}
	if rows[0].FiscalYear != 2026 || rows[0].OpeningBookValue != 750_000 || rows[0].Amount != 300_000 {
		t.Errorf("Umstiegsjahr: %d, Buchwert %s €, AfA %s € — erwartet 2026, 7.500,00 € und 3.000,00 €",
			rows[0].FiscalYear, rows[0].OpeningBookValue, rows[0].Amount)
	}
	if rows[2].FiscalYear != 2028 || rows[2].Amount != 150_000 || rows[2].ClosingBookValue != 0 {
		t.Errorf("letztes Jahr: %d mit %s €, Buchwert %s € — erwartet 2028 mit 1.500,00 € und null",
			rows[2].FiscalYear, rows[2].Amount, rows[2].ClosingBookValue)
	}
	if got := sumAmounts(rows); got != 750_000 {
		t.Errorf("Summe %s € — erwartet den übernommenen Restbuchwert 7.500,00 €", got)
	}
}

// Hat das Vorsystem weniger abgeschrieben als der Plan, verteilt sich der
// höhere Restbuchwert auf die Restnutzungsdauer; der Plan endet trotzdem mit
// der Nutzungsdauer bei null.
func TestTakeoverSpreadsDeviatingBookValueOverRemainingLife(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2024-01-05",
		Cost:                 1_000_000,
		UsefulLifeMonths:     60,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		TakeoverYear:         2026,
		TakeoverDepreciation: 280_000, // der Plan ergäbe 4.000,00 €
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 3 || rows[0].Amount != 240_000 || rows[1].Amount != 240_000 || rows[2].Amount != 240_000 {
		t.Fatalf("erwartet drei Jahre mit je 2.400,00 € (7.200,00 € auf 36 Restmonate): %+v", rows)
	}
	if rows[2].ClosingBookValue != 0 {
		t.Errorf("Restbuchwert %s € nach dem letzten Jahr", rows[2].ClosingBookValue)
	}
}

// Die degressive AfA rechnet vom übernommenen Restbuchwert weiter und wechselt
// zur linearen, sobald diese höher ausfällt.
func TestTakeoverContinuesDegressiveFromCarriedBookValue(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2025-07-01",
		Cost:                 1_000_000,
		UsefulLifeMonths:     60,
		Method:               domain.DepreciationDegressive,
		FiscalYearStartMonth: 1,
		TakeoverYear:         2026,
		TakeoverDepreciation: 150_000,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) == 0 || rows[0].FiscalYear != 2026 || rows[0].OpeningBookValue != 850_000 {
		t.Fatalf("Umstiegsjahr: %+v", rows)
	}
	if rows[0].Method != domain.DepreciationDegressive || rows[0].Amount != 255_000 {
		t.Errorf("2026: %s mit %s € — erwartet degressiv 30 %% von 8.500,00 € = 2.550,00 €",
			rows[0].Method, rows[0].Amount)
	}
	if last := rows[len(rows)-1]; last.ClosingBookValue != 0 || sumAmounts(rows) != 850_000 {
		t.Errorf("der Plan endet mit %s € Buchwert und %s € Summe", last.ClosingBookValue, sumAmounts(rows))
	}
}

// Ein Sammelposten aus 2024 mit zwei aufgelösten Fünfteln läuft ab dem
// Umstieg 2026 mit je einem Fünftel der Anschaffungskosten weiter.
func TestTakeoverContinuesPool(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2024-12-31",
		Cost:                 500_000,
		Method:               domain.DepreciationPool,
		PoolYear:             2024,
		FiscalYearStartMonth: 1,
		TakeoverYear:         2026,
		TakeoverDepreciation: 200_000,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 3 || rows[0].FiscalYear != 2026 || rows[2].FiscalYear != 2028 {
		t.Fatalf("erwartet die Jahre 2026 bis 2028: %+v", rows)
	}
	for _, r := range rows {
		if r.Amount != 100_000 {
			t.Errorf("%d: %s € — erwartet 1.000,00 €", r.FiscalYear, r.Amount)
		}
	}
}

// Ein Gebäude behält den festen Satz auf die Anschaffungskosten; der
// übernommene Buchwert bestimmt nur, wann er aufgebraucht ist.
func TestTakeoverKeepsBuildingRateOnCost(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2020-01-01",
		Cost:                 30_000_000,
		Method:               domain.DepreciationBuildingLinear,
		BuildingPermille:     30,
		FiscalYearStartMonth: 1,
		TakeoverYear:         2026,
		TakeoverDepreciation: 5_400_000,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) == 0 || rows[0].FiscalYear != 2026 || rows[0].OpeningBookValue != 24_600_000 ||
		rows[0].Amount != 900_000 {
		t.Fatalf("Umstiegsjahr: %+v", rows[:min(len(rows), 1)])
	}
	if got := sumAmounts(rows); got != 24_600_000 {
		t.Errorf("Summe %s € — erwartet den übernommenen Buchwert", got)
	}
}

// Ist die Nutzungsdauer beim Umstieg abgelaufen, schreibt Buchfink nichts mehr
// ab: ein Erinnerungswert bleibt stehen.
func TestTakeoverAfterEndOfUsefulLifeHasNoSchedule(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2019-03-01",
		Cost:                 300_000,
		UsefulLifeMonths:     36,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		TakeoverYear:         2026,
		TakeoverDepreciation: 299_900,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("erwartet keinen Plan, bekommen %+v", rows)
	}
}
