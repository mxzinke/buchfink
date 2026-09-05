package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Lehrbuchbeispiel: eine Versicherungsprämie von 1.200 € für die zwölf
// Monate ab dem 1. Dezember. Zum 31.12. ist ein Zwölftel verbraucht, elf sind
// abzugrenzen (§ 250 Abs. 1 HGB).
func TestAccrualShareSpreadsByTwelfths(t *testing.T) {
	share, err := AccrualShare(
		120_000, "2026-12-01", "2027-11-30", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Anteil rechnen: %v", err)
	}
	if share != 110_000 {
		t.Errorf("abzugrenzen %s € — erwartet 1.100,00 €", share)
	}
}

// Taggenau gerechnet ist derselbe Zeitraum 334 von 365 Tagen: die Zwölftel sind
// eine Vereinfachung, und wer sie nicht will, bekommt die Tage.
func TestAccrualShareSpreadsByDays(t *testing.T) {
	share, err := AccrualShare(
		120_000, "2026-12-01", "2027-11-30", "2026-12-31", domain.AccrualDaily, 1)
	if err != nil {
		t.Fatalf("Anteil rechnen: %v", err)
	}
	want := domain.MulRound(120_000, 334, 365)
	if share != want {
		t.Errorf("taggenau abzugrenzen %s € — erwartet %s €", share, want)
	}
	if share == 110_000 {
		t.Error("taggenau und monatsgenau dürfen nicht denselben Betrag ergeben")
	}
}

// Ein Zeitraum, der ganz im Geschäftsjahr liegt, wird nicht abgegrenzt — sonst
// verschöbe die Abgrenzung Aufwand, der bereits am richtigen Platz steht.
func TestAccrualShareIsZeroInsideTheYear(t *testing.T) {
	share, err := AccrualShare(
		120_000, "2026-01-01", "2026-12-31", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Anteil rechnen: %v", err)
	}
	if share != 0 {
		t.Errorf("abzugrenzen %s € — erwartet null", share)
	}
}

// Der Auflösungsplan verteilt den abgegrenzten Betrag auf die Jahre nach dem
// Stichtag, und er geht auf: die Summe der Auflösungen ist der abgegrenzte
// Betrag, keinen Cent mehr und keinen weniger.
func TestAccrualReleasePlanCoversTheWholeAmount(t *testing.T) {
	plan, err := AccrualReleasePlan(
		110_000, "2026-12-01", "2027-11-30", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Auflösungsplan: %v", err)
	}
	if len(plan) != 1 {
		t.Fatalf("erwartet eine Auflösung im Jahr 2027, bekommen %d", len(plan))
	}
	if plan[0].FiscalYear != 2027 || plan[0].Amount != 110_000 {
		t.Errorf("Auflösung %d über %s € — erwartet 2027 über 1.100,00 €",
			plan[0].FiscalYear, plan[0].Amount)
	}
	if plan[0].Date != "2027-01-01" {
		t.Errorf("Auflösung am %s — erwartet den ersten Tag des Folgejahres", plan[0].Date)
	}
}

// Ein Disagio läuft über die Laufzeit des Darlehens und damit über mehrere
// Jahre (§ 250 Abs. 3 HGB). Der Plan verteilt es auf sie alle, und die
// Rundungsdifferenz landet im letzten Jahr — ein Rest auf 1900, der zu nichts
// mehr gehört, fiele erst Jahre später auf.
func TestAccrualReleasePlanSpreadsOverSeveralYears(t *testing.T) {
	plan, err := AccrualReleasePlan(
		100_000, "2026-07-01", "2029-06-30", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Auflösungsplan: %v", err)
	}
	if len(plan) != 3 {
		t.Fatalf("erwartet drei Jahre, bekommen %d", len(plan))
	}
	var total domain.Cents
	years := []int{2027, 2028, 2029}
	for i, release := range plan {
		if release.FiscalYear != years[i] {
			t.Errorf("Position %d gehört zu %d — erwartet %d", i, release.FiscalYear, years[i])
		}
		total += release.Amount
	}
	if total != 100_000 {
		t.Errorf("Summe der Auflösungen %s € — erwartet 1.000,00 €", total)
	}
	// 30 Monate nach dem Stichtag: 12 + 12 + 6.
	if plan[2].Amount >= plan[0].Amount {
		t.Errorf("das halbe letzte Jahr trägt %s € und damit nicht weniger als ein volles (%s €)",
			plan[2].Amount, plan[0].Amount)
	}
}

// Ein Zeitraum, der vollständig vor dem Stichtag liegt, hat nichts aufzulösen —
// und ein Plan über nichts ist ein Fehler und keine leere Liste.
func TestAccrualReleasePlanRefusesAnExpiredPeriod(t *testing.T) {
	if _, err := AccrualReleasePlan(
		100_000, "2025-01-01", "2025-12-31", "2026-12-31", domain.AccrualMonthly, 1); err == nil {
		t.Error("für einen abgelaufenen Zeitraum darf kein Auflösungsplan entstehen")
	}
}

// Beim monatlichen Takt entsteht je Monat eine Auflösung am Monatsersten. Die
// Jahressummen bleiben dieselben; verteilt wird nur, wann der Aufwand innerhalb
// des Jahres ankommt.
func TestAccrualReleasePlanMonthlySpreadsOverTheMonths(t *testing.T) {
	plan, err := AccrualReleasePlanFor(
		110_000, "2026-12-01", "2027-11-30", "2026-12-31",
		domain.AccrualMonthly, 1, domain.AccrualReleaseMonthly)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan) != 11 {
		t.Fatalf("erwartet elf Monate, bekommen %d", len(plan))
	}
	if plan[0].Date != "2027-01-01" || plan[10].Date != "2027-11-01" {
		t.Errorf("erster Termin %s, letzter %s — erwartet 2027-01-01 und 2027-11-01",
			plan[0].Date, plan[10].Date)
	}
	var total domain.Cents
	for _, release := range plan {
		if release.FiscalYear != 2027 {
			t.Errorf("Auflösung im Geschäftsjahr %d — erwartet 2027", release.FiscalYear)
		}
		total += release.Amount
	}
	if total != 110_000 {
		t.Errorf("der Plan deckt %s € — erwartet 1.100,00 €", total)
	}
}

// Auch taggenau gerechnet bleibt der monatliche Takt monatlich: die Methode
// entscheidet über den Anteil, der Takt über die Zahl der Buchungen.
func TestAccrualReleasePlanMonthlyStaysMonthlyWhenCountedByDays(t *testing.T) {
	plan, err := AccrualReleasePlanFor(
		100_000, "2026-12-01", "2027-02-28", "2026-12-31",
		domain.AccrualDaily, 1, domain.AccrualReleaseMonthly)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan) != 2 {
		t.Fatalf("erwartet zwei Monate, bekommen %d: %+v", len(plan), plan)
	}
	if plan[0].Date != "2027-01-01" || plan[1].Date != "2027-02-01" {
		t.Errorf("Termine %s und %s — erwartet die beiden Monatsersten", plan[0].Date, plan[1].Date)
	}
	// 31 von 59 Tagen entfallen auf den Januar.
	if plan[0].Amount != 52_542 {
		t.Errorf("Januar %s € — erwartet 31 von 59 Tagen aus 1.000,00 €", plan[0].Amount)
	}
	if plan[0].Amount+plan[1].Amount != 100_000 {
		t.Errorf("der Plan deckt %s € — erwartet 1.000,00 €", plan[0].Amount+plan[1].Amount)
	}
}

// Ein Zwölfmonatsvertrag bleibt ein Zwölftelvertrag, auch wenn er mitten im
// Monat beginnt. Zählten Anfangs- und Endmonat je voll, ergäbe der Vertrag vom
// 15.12.2026 bis zum 14.12.2027 dreizehn Einheiten und damit 12/13 statt 11/12
// als Abgrenzung.
func TestAccrualShareCountsTheStartMonthOnlyOnce(t *testing.T) {
	share, err := AccrualShare(
		120_000, "2026-12-15", "2027-12-14", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Anteil rechnen: %v", err)
	}
	if share != 110_000 {
		t.Errorf("abzugrenzen %s € — erwartet 1.100,00 € (elf von zwölf Monaten)", share)
	}
}

// Ragt der Vertrag dagegen über den Zwölfmonatszeitraum hinaus, zählt der
// angefangene Monat am Ende: vom 15.12.2026 bis zum 31.12.2027 sind es
// dreizehn Einheiten.
func TestAccrualShareCountsAnOverhangingLastMonth(t *testing.T) {
	share, err := AccrualShare(
		130_000, "2026-12-15", "2027-12-31", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Anteil rechnen: %v", err)
	}
	if want := domain.MulRound(130_000, 12, 13); share != want {
		t.Errorf("abzugrenzen %s € — erwartet %s € (zwölf von dreizehn Monaten)", share, want)
	}
}

// Der Auflösungsplan eines unterjährig beginnenden Vertrags geht auf und
// verteilt auf dieselben zwölf Einheiten.
func TestAccrualReleasePlanForAMidMonthContract(t *testing.T) {
	plan, err := AccrualReleasePlan(
		110_000, "2026-12-15", "2027-12-14", "2026-12-31", domain.AccrualMonthly, 1)
	if err != nil {
		t.Fatalf("Auflösungsplan: %v", err)
	}
	var total domain.Cents
	for _, release := range plan {
		total += release.Amount
		if release.FiscalYear != 2027 {
			t.Errorf("Auflösung im Geschäftsjahr %d — erwartet 2027", release.FiscalYear)
		}
	}
	if total != 110_000 {
		t.Errorf("der Plan löst %s € auf — erwartet die abgegrenzten 1.100,00 €", total)
	}
}
