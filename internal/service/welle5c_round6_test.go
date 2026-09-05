package service

import (
	"context"
	"strings"
	"testing"
)

// Der Berichtigungszeitraum des § 15a UStG läuft ab der erstmaligen Verwendung
// und nicht ab dem Jahresbeginn. Bei einem Zugang mitten im Jahr reicht er
// deshalb ins sechste Kalenderjahr, und Anfangs- wie Schlussjahr gehen nur mit
// ihren Monaten ein (§ 45 UStDV).
func TestInputTaxCorrectionPeriodRunsFromFirstUse(t *testing.T) {
	// Pkw, 47.600 € brutto, 7.600 € Vorsteuer, Zugang zur Jahresmitte. Der
	// Zeitraum endet am 30.06.2031; auf 2031 entfallen sechs Monate.
	t.Run("Schlussjahr anteilig", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.inputTax(t)

		correction, err := svc.Register(ctx, RegisterInputTaxRequest{
			Label: "Pkw AN-2026-0002", Account: "0520",
			AcquisitionDate: "2026-07-01",
			NetAmount:       4_000_000, InputTaxAmount: 760_000,
		})
		if err != nil {
			t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
		}
		if correction.PeriodEnd != "2031-06-30" {
			t.Errorf("Ende des Zeitraums %s — erwartet 2031-06-30", correction.PeriodEnd)
		}
		if correction.LastFiscalYear != 2031 {
			t.Errorf("letztes Berichtigungsjahr %d — erwartet 2031: der Zeitraum reicht ins "+
				"sechste Kalenderjahr", correction.LastFiscalYear)
		}

		view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
			CorrectionID: correction.ID, FiscalYear: 2031, Permille: 600,
			Reason: "Pkw ab 2031 zu 40 % für steuerfreie Vermietung eingesetzt",
		})
		if err != nil {
			t.Fatalf("Verwendungsanteil 2031: %v", err)
		}
		if len(view.Rows) != 1 {
			t.Fatalf("%d Zeilen im Verzeichnis — erwartet eine", len(view.Rows))
		}
		if !view.Rows[0].InPeriod {
			t.Fatalf("2031 liegt im Zeitraum: %s", view.Rows[0].Assessment.Reason)
		}
		if view.Rows[0].MonthsInYear != 6 {
			t.Errorf("%d Monate in 2031 — erwartet sechs (Januar bis Juni)",
				view.Rows[0].MonthsInYear)
		}
		// 7.600 / 5 × 6/12 × 40 % = 304,00 €.
		if view.TotalAmount != -30_400 {
			t.Errorf("Berichtigung 2031: %s € — erwartet -304,00 €", view.TotalAmount)
		}

		// Erst das Jahr danach liegt außerhalb, und der Grund nennt den Zeitraum
		// mit seinen Daten.
		after, err := svc.Year(ctx, 2032)
		if err != nil {
			t.Fatalf("Verzeichnis 2032: %v", err)
		}
		if after.Rows[0].InPeriod {
			t.Error("2032 liegt nach dem Berichtigungszeitraum")
		}
		if !strings.Contains(after.Rows[0].Assessment.Reason, "30.06.2031") {
			t.Errorf("der Grund muss das Ende des Zeitraums nennen: %s",
				after.Rows[0].Assessment.Reason)
		}
	})

	// Zugang am 20.12.: der Zeitraum endet am 19.12.2031, und weil das nach dem
	// 15. liegt, zählt der Dezember 2031 nach § 45 UStDV voll. 2031 geht
	// deshalb mit zwölf Monaten ein — und war vorher gar nicht berichtigbar.
	t.Run("Zugang im Dezember", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.inputTax(t)

		correction, err := svc.Register(ctx, RegisterInputTaxRequest{
			Label: "Pkw AN-2026-0003", Account: "0520",
			AcquisitionDate: "2026-12-20",
			NetAmount:       4_000_000, InputTaxAmount: 760_000,
		})
		if err != nil {
			t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
		}
		if correction.PeriodEnd != "2031-12-31" {
			t.Errorf("Ende des Zeitraums %s — erwartet 2031-12-31", correction.PeriodEnd)
		}

		view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
			CorrectionID: correction.ID, FiscalYear: 2031, Permille: 600,
		})
		if err != nil {
			t.Fatalf("Verwendungsanteil 2031: %v", err)
		}
		if view.Rows[0].MonthsInYear != 12 {
			t.Errorf("%d Monate in 2031 — erwartet zwölf", view.Rows[0].MonthsInYear)
		}
		if view.TotalAmount != -60_800 {
			t.Errorf("Berichtigung 2031: %s € — erwartet -608,00 €", view.TotalAmount)
		}

		// Das Zugangsjahr trägt nur den Dezember.
		first, err := svc.Year(ctx, 2026)
		if err != nil {
			t.Fatalf("Verzeichnis 2026: %v", err)
		}
		if first.Rows[0].MonthsInYear != 1 {
			t.Errorf("%d Monate in 2026 — erwartet einen (Dezember)", first.Rows[0].MonthsInYear)
		}
	})

	// Weicht die Verwendung schon im Jahr der erstmaligen Verwendung von der
	// beim Abzug beabsichtigten ab, ist auch das ein Fall des § 15a UStG
	// (UStAE 15a.2 Abs. 2) — anteilig für die Monate der Verwendung.
	t.Run("Änderung im Zugangsjahr", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.inputTax(t)

		correction, err := svc.Register(ctx, RegisterInputTaxRequest{
			Label: "Pkw AN-2026-0004", Account: "0520",
			AcquisitionDate: "2026-07-01",
			NetAmount:       4_000_000, InputTaxAmount: 760_000,
		})
		if err != nil {
			t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
		}

		view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
			CorrectionID: correction.ID, FiscalYear: 2026, Permille: 600,
			Reason: "Pkw von Beginn an zu 40 % für steuerfreie Vermietung eingesetzt",
		})
		if err != nil {
			t.Fatalf("Verwendungsanteil im Zugangsjahr: %v", err)
		}
		if !view.Rows[0].InPeriod {
			t.Fatalf("das Zugangsjahr gehört in den Zeitraum: %s", view.Rows[0].Assessment.Reason)
		}
		if view.TotalAmount != -30_400 {
			t.Errorf("Berichtigung 2026: %s € — erwartet -304,00 € (sechs Monate)", view.TotalAmount)
		}

		booked, err := svc.BookYear(ctx, 2026)
		if err != nil {
			t.Fatalf("Buchung im Zugangsjahr: %v", err)
		}
		if !booked.Rows[0].Booked {
			t.Error("die Berichtigung des Zugangsjahres muss buchbar sein")
		}
	})
}
