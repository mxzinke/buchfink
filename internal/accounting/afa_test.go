package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func sumAmounts(rows []AfAYear) domain.Cents {
	var total domain.Cents
	for _, r := range rows {
		total += r.Amount
	}
	return total
}

// Eine im September angeschaffte Anlage wird im ersten Jahr mit vier Zwölfteln
// abgeschrieben (§ 7 Abs. 1 Satz 4 EStG) — und läuft dafür ein Jahr länger.
func TestLinearScheduleIsProRataInTheFirstYear(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-09-15",
		Cost:                 1_200_000, // 12.000,00 €
		UsefulLifeMonths:     48,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("erwartet 5 Jahre (2026 bis 2030), bekommen %d", len(rows))
	}
	if rows[0].FiscalYear != 2026 || rows[0].Months != 4 || rows[0].Amount != 100_000 {
		t.Errorf("erstes Jahr: %d, %d Monate, %s € — erwartet 2026, 4 Monate, 1.000,00 €",
			rows[0].FiscalYear, rows[0].Months, rows[0].Amount)
	}
	if rows[1].Amount != 300_000 {
		t.Errorf("volles Jahr: %s € — erwartet 3.000,00 €", rows[1].Amount)
	}
	last := rows[len(rows)-1]
	if last.FiscalYear != 2030 || last.ClosingBookValue != 0 {
		t.Errorf("letztes Jahr %d endet mit Buchwert %s € — erwartet 2030 und null", last.FiscalYear, last.ClosingBookValue)
	}
	if got := sumAmounts(rows); got != 1_200_000 {
		t.Errorf("Summe der AfA %s € — erwartet die vollen Anschaffungskosten", got)
	}
}

// Ein abweichendes Geschäftsjahr verschiebt die Jahresgrenzen, nicht die
// Monatsrechnung.
func TestLinearScheduleFollowsDeviatingFiscalYear(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-09-15",
		Cost:                 1_200_000,
		UsefulLifeMonths:     48,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 7, // Geschäftsjahr Juli bis Juni
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// Anschaffung im September 2026 liegt im Geschäftsjahr 2026 (Juli 2026 bis
	// Juni 2027); davon bleiben zehn Monate.
	if rows[0].FiscalYear != 2026 || rows[0].Months != 10 {
		t.Errorf("erstes Jahr: %d mit %d Monaten — erwartet 2026 mit 10", rows[0].FiscalYear, rows[0].Months)
	}
	if got := sumAmounts(rows); got != 1_200_000 {
		t.Errorf("Summe der AfA %s € — erwartet die vollen Anschaffungskosten", got)
	}
}

// Die degressive AfA ist auf das Dreifache des linearen Satzes und auf 30 %
// gedeckelt und geht über, sobald die lineare Restwertabschreibung höher ist
// (§ 7 Abs. 2 und 3 EStG).
func TestDegressiveScheduleCapsAndSwitchesToLinear(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2025-07-01",
		Cost:                 1_000_000, // 10.000,00 €
		UsefulLifeMonths:     60,
		Method:               domain.DepreciationDegressive,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if rows[0].Amount != 150_000 {
		t.Errorf("2025: %s € — erwartet 1.500,00 € (30 %% von 10.000 €, zeitanteilig sechs Monate)", rows[0].Amount)
	}
	if rows[1].Amount != 255_000 {
		t.Errorf("2026: %s € — erwartet 2.550,00 € (30 %% des Restbuchwerts)", rows[1].Amount)
	}

	var switched bool
	for _, r := range rows {
		if switched && r.Method == domain.DepreciationDegressive {
			t.Errorf("nach dem Übergang zur linearen AfA folgt in %d wieder eine degressive Zeile", r.FiscalYear)
		}
		if r.Method == domain.DepreciationLinear && r.RateLabel != "Restwert" {
			switched = true
		}
	}
	if !switched {
		t.Error("der Übergang zur linearen Abschreibung (§ 7 Abs. 3 EStG) kommt im Plan nicht vor")
	}
	if got := sumAmounts(rows); got != 1_000_000 {
		t.Errorf("Summe der AfA %s € — erwartet die vollen Anschaffungskosten", got)
	}
}

// Außerhalb des gesetzlichen Zeitfensters gibt es die degressive AfA nicht. Ein
// Plan, der trotzdem gerechnet würde, wäre still falsch.
func TestDegressiveOutsideWindowIsRefused(t *testing.T) {
	_, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2024-03-01",
		Cost:                 1_000_000,
		UsefulLifeMonths:     60,
		Method:               domain.DepreciationDegressive,
		FiscalYearStartMonth: 1,
	})
	if err == nil {
		t.Fatal("eine Anschaffung von 2024 darf keinen degressiven Plan bekommen")
	}
}

// Der Sammelposten wird mit je einem Fünftel aufgelöst, ohne Zeitanteil im Jahr
// der Bildung (§ 6 Abs. 2a Satz 2 EStG).
func TestPoolDissolvesInFifthsWithoutProRata(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-11-20",
		Cost:                 500_000, // 5.000,00 €
		Method:               domain.DepreciationPool,
		PoolYear:             2026,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("erwartet fünf Jahre, bekommen %d", len(rows))
	}
	if rows[0].FiscalYear != 2026 || rows[0].Amount != 100_000 {
		t.Errorf("Jahr der Bildung: %d mit %s € — erwartet 2026 mit 1.000,00 €", rows[0].FiscalYear, rows[0].Amount)
	}
	if rows[4].FiscalYear != 2030 {
		t.Errorf("letztes Jahr %d — erwartet 2030", rows[4].FiscalYear)
	}
	if got := sumAmounts(rows); got != 500_000 {
		t.Errorf("Summe %s € — erwartet den vollen Sammelposten", got)
	}
}

// Der Sofortabzug ist mit dem Anschaffungsjahr erledigt.
func TestImmediateWriteOffIsOneYear(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-04-02",
		Cost:                 79_000,
		Method:               domain.DepreciationImmediate,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 1 || rows[0].Amount != 79_000 || rows[0].ClosingBookValue != 0 {
		t.Fatalf("erwartet eine Zeile über den vollen Betrag, bekommen %+v", rows)
	}
}

// Grund und Boden, Beteiligungen, Wertpapiere: kein Plan, keine Zeilen.
func TestNoMethodProducesNoSchedule(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-01-02",
		Cost:                 10_000_000,
		Method:               domain.DepreciationNone,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("erwartet keinen Plan, bekommen %d Zeilen", len(rows))
	}
}

// Bis einschließlich des Abgangsmonats wird abgeschrieben, danach nicht mehr —
// was bleibt, ist der Restbuchwert, den die Abgangsbuchung ausbucht.
func TestScheduleStopsInTheMonthOfDisposal(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-01-01",
		Cost:                 600_000, // 6.000,00 €, 60 Monate → 100,00 € im Monat
		UsefulLifeMonths:     60,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		DisposalDate:         "2028-06-20",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	last := rows[len(rows)-1]
	if last.FiscalYear != 2028 || last.Months != 6 {
		t.Fatalf("letztes Jahr %d mit %d Monaten — erwartet 2028 mit 6", last.FiscalYear, last.Months)
	}
	if got := sumAmounts(rows); got != 300_000 {
		t.Errorf("Summe %s € — erwartet 3.000,00 € für 30 Monate", got)
	}
	if last.ClosingBookValue != 300_000 {
		t.Errorf("Restbuchwert %s € — erwartet 3.000,00 €", last.ClosingBookValue)
	}
}

// Nach einer außerplanmäßigen Abschreibung verteilt sich der geminderte Wert auf
// die Restnutzungsdauer; die Summe bleibt bei den Anschaffungskosten.
func TestImpairmentReducesTheFollowingPlannedAmounts(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-01-01",
		Cost:                 1_000_000,
		UsefulLifeMonths:     60,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		ImpairmentsByYear:    map[int]domain.Cents{2026: 200_000},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if rows[0].ClosingBookValue != 600_000 {
		t.Fatalf("Buchwert Ende 2026 %s € — erwartet 6.000,00 € (10.000 − 2.000 planmäßig − 2.000 außerplanmäßig)",
			rows[0].ClosingBookValue)
	}
	if rows[1].Amount != 150_000 {
		t.Errorf("2027: %s € — erwartet 1.500,00 € (6.000 € auf 48 Restmonate)", rows[1].Amount)
	}
	if got := sumAmounts(rows) + 200_000; got != 1_000_000 {
		t.Errorf("planmäßige AfA und außerplanmäßige Abschreibung ergeben %s € — erwartet 10.000,00 €", got)
	}
}

// Die Wertgrenzen entscheiden über die Behandlung, die selbständige Nutzbarkeit
// entscheidet vor ihnen.
func TestClassifyAcquisition(t *testing.T) {
	advice, err := ClassifyAcquisition(30_000, "2026-05-01", false)
	if err != nil {
		t.Fatalf("Einordnung: %v", err)
	}
	if advice.Recommended != AcquisitionActivate || len(advice.Allowed) != 1 {
		t.Errorf("ein nicht selbständig nutzbares Gut für 300 € muss aktiviert werden, bekommen %+v", advice)
	}

	advice, err = ClassifyAcquisition(30_000, "2026-05-01", true)
	if err != nil {
		t.Fatalf("Einordnung: %v", err)
	}
	if advice.Recommended != AcquisitionImmediate {
		t.Errorf("300 € selbständig nutzbar: erwartet Sofortabzug, bekommen %q", advice.Recommended)
	}
	if advice.PoolNote == "" {
		t.Error("wo der Sammelposten offensteht, gehört der Hinweis auf das einheitliche Wahlrecht dazu")
	}

	advice, err = ClassifyAcquisition(90_000, "2026-05-01", true)
	if err != nil {
		t.Fatalf("Einordnung: %v", err)
	}
	if advice.Recommended != AcquisitionPool {
		t.Errorf("900 € netto: erwartet den Sammelposten, bekommen %q", advice.Recommended)
	}

	advice, err = ClassifyAcquisition(150_000, "2026-05-01", true)
	if err != nil {
		t.Fatalf("Einordnung: %v", err)
	}
	if advice.Recommended != AcquisitionActivate || len(advice.Allowed) != 1 {
		t.Errorf("1.500 € netto: erwartet die Aktivierung als einzigen Weg, bekommen %+v", advice)
	}
}
