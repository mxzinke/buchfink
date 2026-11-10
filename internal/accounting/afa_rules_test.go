package accounting

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Abschreibungsregeln aus der Ressource
// -------------------------------------------------------------------------

// Die Regeln werden aus afa_rules.json geladen und nicht aus Go-Literalen. Der
// Test liest dieselbe Datei und vergleicht ein Fenster — driftete die Ressource
// von der Rechnung ab, fiele es hier auf und nicht im Abschluss.
func TestAfARulesComeFromTheEmbeddedResource(t *testing.T) {
	raw, err := os.ReadFile("afa_rules.json")
	if err != nil {
		t.Fatalf("afa_rules.json: %v", err)
	}
	var fromFile AfARules
	if err := json.Unmarshal(raw, &fromFile); err != nil {
		t.Fatalf("afa_rules.json ist nicht lesbar: %v", err)
	}

	loaded := AfARuleSet()
	if loaded.Version != fromFile.Version {
		t.Errorf("Version %q geladen, %q in der Datei", loaded.Version, fromFile.Version)
	}
	if len(loaded.DegressiveWindows) != len(fromFile.DegressiveWindows) {
		t.Fatalf("%d Fenster geladen, %d in der Datei",
			len(loaded.DegressiveWindows), len(fromFile.DegressiveWindows))
	}

	// Das Fenster, das im Jahr 2026 offen ist: § 7 Abs. 2 Sätze 1 und 2 EStG mit
	// dem Dreifachen des linearen Satzes, höchstens 30 %.
	window, ok := DegressiveWindowFor("2026-05-01")
	if !ok {
		t.Fatal("für eine Anschaffung im Mai 2026 ist die degressive Abschreibung offen")
	}
	if window.FactorPermille != 3000 || window.MaxPermille != 300 {
		t.Errorf("Faktor %d, Deckel %d — erwartet 3000 und 300",
			window.FactorPermille, window.MaxPermille)
	}
	found := false
	for _, w := range fromFile.DegressiveWindows {
		if w.From == window.From && w.Until == window.Until {
			found = true
			if w.FactorPermille != window.FactorPermille || w.MaxPermille != window.MaxPermille {
				t.Errorf("die Datei nennt Faktor %d und Deckel %d, gerechnet wird mit %d und %d",
					w.FactorPermille, w.MaxPermille, window.FactorPermille, window.MaxPermille)
			}
		}
	}
	if !found {
		t.Errorf("das Fenster %s bis %s steht nicht in der Datei", window.From, window.Until)
	}

	// Und die Wertgrenzen: ab 2018 endet der Sofortabzug bei 800 €.
	params, err := AfAParametersFor("2026-01-01")
	if err != nil {
		t.Fatalf("Wertgrenzen: %v", err)
	}
	if params.GWGImmediateLimit != 80000 {
		t.Errorf("GWG-Grenze %s € — erwartet 800,00 €", params.GWGImmediateLimit)
	}
}

// Die Staffel des § 7 Abs. 2a EStG: 75, 10, 5, 5, 3 und 2 % der
// Anschaffungskosten, ohne Zeitanteil im Anschaffungsjahr, und am Ende genau
// null.
func TestElectricVehicleSchedule(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-11-15",
		Cost:                 10_000_000, // 100.000 €
		Method:               domain.DepreciationElectricVehicle,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := []domain.Cents{7_500_000, 1_000_000, 500_000, 500_000, 300_000, 200_000}
	if len(rows) != len(want) {
		t.Fatalf("%d Jahre — erwartet %d", len(rows), len(want))
	}
	var total domain.Cents
	for i, row := range rows {
		if row.Amount != want[i] {
			t.Errorf("Jahr %d: %s € — erwartet %s €", row.FiscalYear, row.Amount, want[i])
		}
		if row.Months != 12 {
			t.Errorf("Jahr %d: %d Monate — die Staffel kürzt im Anschaffungsjahr nicht zeitanteilig",
				row.FiscalYear, row.Months)
		}
		total += row.Amount
	}
	if total != 10_000_000 {
		t.Errorf("Summe %s € — erwartet die vollen Anschaffungskosten", total)
	}
	if rows[len(rows)-1].ClosingBookValue != 0 {
		t.Errorf("Restbuchwert am Ende %s € — erwartet null", rows[len(rows)-1].ClosingBookValue)
	}
	if rows[0].FiscalYear != 2026 {
		t.Errorf("erstes Jahr %d — erwartet 2026", rows[0].FiscalYear)
	}
}

// Außerhalb des Fensters gibt es die Staffel nicht.
func TestElectricVehicleScheduleOutsideItsWindow(t *testing.T) {
	if _, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2024-03-01",
		Cost:                 5_000_000,
		Method:               domain.DepreciationElectricVehicle,
		FiscalYearStartMonth: 1,
	}); err == nil {
		t.Error("für eine Anschaffung im März 2024 gibt es § 7 Abs. 2a EStG noch nicht")
	}
}

// Ein Betriebsgebäude wird mit 3 % linear abgeschrieben, ohne Übergang auf eine
// Restwertabschreibung — nur das Anschaffungsjahr ist zeitanteilig.
func TestBuildingScheduleUsesTheStatutoryRate(t *testing.T) {
	rate, err := BuildingRateFor(false, "2020-06-01")
	if err != nil {
		t.Fatalf("Gebäudesatz: %v", err)
	}
	if rate.Permille != 30 {
		t.Fatalf("Satz %d Promille — erwartet 30 (3 %%)", rate.Permille)
	}

	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-07-01",
		Cost:                 100_000_000, // 1.000.000 €
		Method:               domain.DepreciationBuildingLinear,
		BuildingPermille:     rate.Permille,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) < 34 {
		t.Fatalf("%d Jahre — bei 3 %% läuft der Plan über gut 33 Jahre", len(rows))
	}
	// Im Anschaffungsjahr sechs von zwölf Monaten: 30.000 / 2 = 15.000 €.
	if rows[0].Amount != 1_500_000 || rows[0].Months != 6 {
		t.Errorf("erstes Jahr: %s € über %d Monate — erwartet 15.000,00 € über sechs Monate",
			rows[0].Amount, rows[0].Months)
	}
	if rows[1].Amount != 3_000_000 {
		t.Errorf("zweites Jahr: %s € — erwartet 30.000,00 € (3 %% von 1.000.000 €)", rows[1].Amount)
	}
	// Kein Übergang: jedes volle Jahr hat denselben Betrag, bis der Restwert
	// aufgebraucht ist.
	if rows[10].Amount != 3_000_000 {
		t.Errorf("elftes Jahr: %s € — der feste Satz läuft unverändert weiter", rows[10].Amount)
	}
	var total domain.Cents
	for _, r := range rows {
		total += r.Amount
	}
	if total != 100_000_000 {
		t.Errorf("Summe %s € — erwartet die vollen Anschaffungskosten", total)
	}
	if rows[len(rows)-1].ClosingBookValue != 0 {
		t.Errorf("Restbuchwert am Ende %s € — erwartet null", rows[len(rows)-1].ClosingBookValue)
	}
}

// Der Investitionsabzugsbetrag des § 7g Abs. 1 EStG fehlt in Buchfink nicht, er
// gehört nicht hierher: er wird außerbilanziell abgezogen und in der
// Steuererklärung geltend gemacht. Ohne einen Satz dazu sähe die Anlagenmaske
// aus, als kenne Buchfink den § 7g nur zur Hälfte — die Sonderabschreibung des
// Absatzes 5 rechnet sie ja.
func TestAfARulesExplainTheInvestmentDeduction(t *testing.T) {
	note := AfARuleSet().InvestmentDeductionNote
	for _, want := range []string{"§ 7g Abs. 1", "außerbilanziell", "Steuererklärung", "§ 7g Abs. 2"} {
		if !strings.Contains(note, want) {
			t.Errorf("der Hinweis nennt %q nicht: %s", want, note)
		}
	}
}

// Wohngebäude folgen eigenen Sätzen, und der Stichtag ist die Fertigstellung.
func TestBuildingRatesFollowTheirReferenceDate(t *testing.T) {
	tests := []struct {
		name        string
		residential bool
		reference   string
		permille    int64
	}{
		{"Betriebsgebäude, Bauantrag 1990", false, "1990-05-01", 30},
		{"Betriebsgebäude, Bauantrag 1980", false, "1980-05-01", 20},
		// Nummer 2 gilt für jedes Gebäude, das die Voraussetzungen der Nummer 1
		// nicht erfüllt — auch für ein Betriebsgebäude. Buchstabe c gibt ihm vor
		// 1925 dieselben 2,5 % wie einem Wohngebäude; pauschal 2 % wären für ein
		// sehr altes Betriebsgebäude zu wenig.
		{"Betriebsgebäude, fertiggestellt 1910", false, "1910-05-01", 25},
		{"Wohngebäude, fertiggestellt 2024", true, "2024-03-01", 30},
		{"Wohngebäude, fertiggestellt 2010", true, "2010-03-01", 20},
		{"Wohngebäude, fertiggestellt 1910", true, "1910-03-01", 25},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := BuildingRateFor(tc.residential, tc.reference)
			if err != nil {
				t.Fatalf("Gebäudesatz: %v", err)
			}
			if rate.Permille != tc.permille {
				t.Errorf("Satz %d Promille — erwartet %d (%s)",
					rate.Permille, tc.permille, rate.Source)
			}
		})
	}
	if _, err := BuildingRateFor(false, ""); err == nil {
		t.Error("ohne Stichtag lässt sich kein Gebäudesatz bestimmen")
	}
}
