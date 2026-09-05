package accounting

import (
	"strings"
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

// Ein 2021 angeschafftes Wirtschaftsgut schreibt nach dem Satz ab, der damals
// galt: 2,5 % des linearen Satzes, höchstens 25 % (§ 7 Abs. 2 EStG in der
// Fassung des Zweiten Corona-Steuerhilfegesetzes). Mit dem heutigen Satz
// gerechnet wäre sein Plan um ein Fünftel zu hoch.
func TestDegressiveFollowsTheWindowOfItsOwnAcquisition(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2021-01-01",
		Cost:                 1_000_000, // 10.000,00 €
		UsefulLifeMonths:     120,       // linear 10 %, das Zweieinhalbfache sind 25 %
		Method:               domain.DepreciationDegressive,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if rows[0].Amount != 250_000 {
		t.Errorf("2021: %s € — erwartet 2.500,00 € (25 %% von 10.000 €)", rows[0].Amount)
	}
	if rows[0].RateLabel != "25 %" {
		t.Errorf("Satz %q — erwartet 25 %%", rows[0].RateLabel)
	}
	if got := sumAmounts(rows); got != 1_000_000 {
		t.Errorf("Summe der AfA %s € — erwartet die vollen Anschaffungskosten", got)
	}
}

// Das Fenster des Wachstumschancengesetzes erlaubte nur das Zweifache und
// höchstens 20 %, und es begann erst am 01.04.2024. Der Faktor ist kein
// ganzzahliges Vielfaches über alle Fassungen hinweg — 2,5 muss 2,5 bleiben.
func TestDegressiveWindowsCarryTheirOwnCeilings(t *testing.T) {
	cases := []struct {
		date      string
		life      int
		wantFirst domain.Cents
		wantLabel string
	}{
		// Das Fenster von 2024 öffnete erst im April; der erste Betrag ist
		// deshalb zeitanteilig für neun Monate (§ 7 Abs. 1 Satz 4 EStG).
		{date: "2024-04-01", life: 60, wantFirst: 150_000, wantLabel: "20 %"},    // gedeckelt
		{date: "2024-04-01", life: 240, wantFirst: 75_000, wantLabel: "10 %"},    // 2 × 5 %
		{date: "2009-01-01", life: 240, wantFirst: 125_000, wantLabel: "12,5 %"}, // 2,5 × 5 %
	}
	for _, c := range cases {
		rows, err := BuildAfASchedule(AfAPlan{
			AcquisitionDate:      c.date,
			Cost:                 1_000_000,
			UsefulLifeMonths:     c.life,
			Method:               domain.DepreciationDegressive,
			FiscalYearStartMonth: 1,
		})
		if err != nil {
			t.Fatalf("%s: %v", c.date, err)
		}
		if rows[0].Amount != c.wantFirst || rows[0].RateLabel != c.wantLabel {
			t.Errorf("%s (%d Monate): %s € zu %q — erwartet %s € zu %q",
				c.date, c.life, rows[0].Amount, rows[0].RateLabel, c.wantFirst, c.wantLabel)
		}
	}
}

// Zwischen den Fenstern liegen Lücken, und dort gibt es die degressive AfA
// nicht. Die Fehlermeldung nennt die offenen Zeiträume, sonst rät der Nutzer.
func TestDegressiveGapsStayClosed(t *testing.T) {
	for _, date := range []string{"2023-06-01", "2024-03-31", "2025-06-30"} {
		_, err := BuildAfASchedule(AfAPlan{
			AcquisitionDate:      date,
			Cost:                 1_000_000,
			UsefulLifeMonths:     60,
			Method:               domain.DepreciationDegressive,
			FiscalYearStartMonth: 1,
		})
		if err == nil {
			t.Errorf("für den %s darf kein degressiver Plan entstehen", date)
			continue
		}
		if !strings.Contains(err.Error(), "01.07.2025 bis 31.12.2027") {
			t.Errorf("die Meldung zum %s nennt die offenen Zeiträume nicht: %v", date, err)
		}
	}
}

// Die Wertgrenzen des § 6 Abs. 2 EStG haben sich geändert. Ein Altbestand muss
// mit den Grenzen erklärbar bleiben, die bei seiner Anschaffung galten.
func TestValueLimitsFollowTheirOwnYear(t *testing.T) {
	old, err := AfAParametersFor("2015-05-01")
	if err != nil {
		t.Fatalf("Wertgrenzen 2015: %v", err)
	}
	if old.GWGImmediateLimit != 41000 || old.PoolLowerLimit != 15000 {
		t.Errorf("2015: Sofortabzug bis %s €, Sammelposten ab %s € — erwartet 410,00 € und 150,00 €",
			old.GWGImmediateLimit, old.PoolLowerLimit)
	}
	now, err := AfAParametersFor("2026-05-01")
	if err != nil {
		t.Fatalf("Wertgrenzen 2026: %v", err)
	}
	if now.GWGImmediateLimit != 80000 || now.PoolLowerLimit != 25000 {
		t.Errorf("2026: Sofortabzug bis %s €, Sammelposten ab %s € — erwartet 800,00 € und 250,00 €",
			now.GWGImmediateLimit, now.PoolLowerLimit)
	}
}

// Die Sonderabschreibung nach § 7g Abs. 5 EStG tritt neben die planmäßige AfA,
// sie ersetzt sie nicht (§ 7a Abs. 4 EStG) — und sie wirkt allein steuerlich.
// Der handelsrechtliche Plan läuft unverändert bis auf null (§ 253 HGB, § 254
// HGB a. F. ist mit dem BilMoG entfallen); in der steuerlichen Rechnung
// verteilt § 7a Abs. 9 EStG nach dem Begünstigungszeitraum den verbliebenen
// Restwert auf die Restnutzungsdauer.
func TestSpecialDepreciationRunsBesideThePlanAndThenSpreadsTheResidual(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-01-01",
		Cost:                 10_000_000, // 100.000,00 €
		UsefulLifeMonths:     120,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		SpecialPermille:      400,
		SpecialYears:         5,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) != 10 {
		t.Fatalf("erwartet 10 Jahre, bekommen %d", len(rows))
	}

	// Handelsrechtlich: zehn Jahre lang unverändert 10 % der Anschaffungskosten.
	for i := range rows {
		if rows[i].Amount != 1_000_000 {
			t.Errorf("%d: planmäßige AfA %s € — erwartet 10.000,00 €", rows[i].FiscalYear, rows[i].Amount)
		}
	}
	if last := rows[len(rows)-1]; last.FiscalYear != 2035 || last.ClosingBookValue != 0 {
		t.Errorf("letztes Jahr %d endet mit einem Buchwert von %s € — erwartet 2035 und null",
			last.FiscalYear, last.ClosingBookValue)
	}

	// Begünstigungszeitraum: dazu ein Fünftel der Sonderabschreibung, nur
	// steuerlich.
	for i := 0; i < 5; i++ {
		if rows[i].SpecialAmount != 800_000 {
			t.Errorf("%d: Sonderabschreibung %s € — erwartet 8.000,00 €", rows[i].FiscalYear, rows[i].SpecialAmount)
		}
		if rows[i].TaxAmount != 1_000_000 {
			t.Errorf("%d: steuerliche AfA %s € — erwartet 10.000,00 €", rows[i].FiscalYear, rows[i].TaxAmount)
		}
	}
	if rows[4].ClosingBookValue != 5_000_000 {
		t.Errorf("handelsrechtlicher Buchwert nach fünf Jahren %s € — erwartet 50.000,00 €",
			rows[4].ClosingBookValue)
	}
	if rows[4].TaxClosingBookValue != 1_000_000 {
		t.Errorf("steuerlicher Restwert nach dem Begünstigungszeitraum %s € — erwartet 10.000,00 €",
			rows[4].TaxClosingBookValue)
	}

	// Danach: steuerlich der Restwert auf die verbliebenen fünf Jahre
	// (§ 7a Abs. 9 EStG), handelsrechtlich unverändert.
	for i := 5; i < 10; i++ {
		if rows[i].SpecialAmount != 0 {
			t.Errorf("%d trägt noch eine Sonderabschreibung von %s €", rows[i].FiscalYear, rows[i].SpecialAmount)
		}
		if rows[i].TaxAmount != 200_000 {
			t.Errorf("%d: steuerliche Restwertabschreibung %s € — erwartet 2.000,00 €",
				rows[i].FiscalYear, rows[i].TaxAmount)
		}
	}
	if last := rows[len(rows)-1]; last.TaxClosingBookValue != 0 {
		t.Errorf("steuerlicher Restwert am Ende %s € — erwartet null", last.TaxClosingBookValue)
	}

	// Beide Rechnungen kommen auf die vollen Anschaffungskosten — die
	// handelsrechtliche über die planmäßige AfA allein, die steuerliche mit der
	// Sonderabschreibung.
	var commercial, tax domain.Cents
	for _, r := range rows {
		commercial += r.Amount
		tax += r.TaxAmount + r.SpecialAmount
	}
	if commercial != 10_000_000 {
		t.Errorf("Summe der planmäßigen AfA %s € — erwartet die vollen Anschaffungskosten", commercial)
	}
	if tax != 10_000_000 {
		t.Errorf("Summe der steuerlichen Abschreibungen %s € — erwartet die vollen Anschaffungskosten", tax)
	}
}

// Im Anschaffungsjahr wird die planmäßige AfA zeitanteilig gekürzt, die
// Sonderabschreibung nicht: § 7 Abs. 1 Satz 4 EStG betrifft die Absetzung für
// Abnutzung, nicht die Sonderabschreibung.
func TestSpecialDepreciationIsNotProRata(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-12-01",
		Cost:                 10_000_000,
		UsefulLifeMonths:     120,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		SpecialPermille:      400,
		SpecialYears:         1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if rows[0].Months != 1 || rows[0].Amount != 83_333 {
		t.Errorf("planmäßige AfA im Anschaffungsjahr: %s € über %d Monate — erwartet 833,33 € über einen",
			rows[0].Amount, rows[0].Months)
	}
	if rows[0].SpecialAmount != 4_000_000 {
		t.Errorf("Sonderabschreibung im Anschaffungsjahr %s € — erwartet die vollen 40.000,00 €",
			rows[0].SpecialAmount)
	}
}

// Zwei Grenzen, an denen ein Plan abgewiesen wird, statt still falsch zu rechnen.
func TestSpecialDepreciationLimits(t *testing.T) {
	base := AfAPlan{
		AcquisitionDate:      "2026-01-01",
		Cost:                 10_000_000,
		UsefulLifeMonths:     120,
		Method:               domain.DepreciationLinear,
		FiscalYearStartMonth: 1,
		SpecialYears:         5,
	}

	over := base
	over.SpecialPermille = 500
	if _, err := BuildAfASchedule(over); err == nil {
		t.Error("mehr als 40 % Sonderabschreibung muss abgewiesen werden (§ 7g Abs. 5 EStG)")
	}

	// § 7g Abs. 5 EStG lässt die Sonderabschreibung „neben den Absetzungen für
	// Abnutzung nach § 7 Absatz 1 oder Absatz 2" zu — die degressive AfA ist
	// also kein Ausschlussgrund. Der Sammelposten kennt dagegen keine AfA, neben
	// die etwas treten könnte.
	withPool := base
	withPool.SpecialPermille = 400
	withPool.Method = domain.DepreciationPool
	if _, err := BuildAfASchedule(withPool); err == nil {
		t.Error("Sonderabschreibung neben dem Sammelposten muss abgewiesen werden")
	}
}

// Die Sonderabschreibung ist auch neben der degressiven AfA zulässig
// (§ 7g Abs. 5 EStG). Handelsrechtlich läuft der degressive Plan unverändert
// weiter; steuerlich zehrt die Sonderabschreibung den Restwert früh auf, und
// mehr als die Anschaffungskosten wird auch steuerlich nicht abgeschrieben.
func TestSpecialDepreciationAlongsideDegressive(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-01-01",
		Cost:                 10_000_000, // 100.000,00 €
		UsefulLifeMonths:     120,        // linear 10 %, degressiv gedeckelt auf 30 %
		Method:               domain.DepreciationDegressive,
		FiscalYearStartMonth: 1,
		SpecialPermille:      400,
		SpecialYears:         1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// Erstes Jahr: 30 % von 100.000 € degressiv, dazu die volle
	// Sonderabschreibung von 40 % — nur steuerlich.
	if rows[0].Amount != 3_000_000 || rows[0].SpecialAmount != 4_000_000 {
		t.Fatalf("2026: %s € degressiv, %s € Sonderabschreibung — erwartet 30.000,00 € und 40.000,00 €",
			rows[0].Amount, rows[0].SpecialAmount)
	}
	if rows[0].ClosingBookValue != 7_000_000 {
		t.Errorf("handelsrechtlicher Buchwert Ende 2026 %s € — erwartet 70.000,00 €", rows[0].ClosingBookValue)
	}
	if rows[0].TaxClosingBookValue != 3_000_000 {
		t.Errorf("steuerlicher Restwert Ende 2026 %s € — erwartet 30.000,00 €", rows[0].TaxClosingBookValue)
	}
	// Zweites Jahr: 30 % vom handelsrechtlichen Restbuchwert.
	if rows[1].Amount != 2_100_000 {
		t.Errorf("2027: %s € — erwartet 21.000,00 € (30 %% von 70.000,00 €)", rows[1].Amount)
	}

	// Der handelsrechtliche Plan endet auf null, und die Summe der planmäßigen
	// AfA sind die vollen Anschaffungskosten.
	var commercial, tax domain.Cents
	for _, r := range rows {
		commercial += r.Amount
		tax += r.TaxAmount + r.SpecialAmount
	}
	if commercial != 10_000_000 {
		t.Errorf("Summe der planmäßigen AfA %s € — erwartet die vollen Anschaffungskosten", commercial)
	}
	if last := rows[len(rows)-1]; last.ClosingBookValue != 0 {
		t.Errorf("Buchwert am Ende der Nutzungsdauer %s € — erwartet null", last.ClosingBookValue)
	}
	// Steuerlich ist der Wertansatz mit der Sonderabschreibung früher
	// aufgezehrt; abgeschrieben wird auch dort nicht mehr als die
	// Anschaffungskosten.
	if tax != 10_000_000 {
		t.Errorf("Summe der steuerlichen Abschreibungen %s € — erwartet die vollen Anschaffungskosten", tax)
	}
}
