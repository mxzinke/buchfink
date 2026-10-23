package accounting

import (
	"testing"
)

// Die Rechnung, an der sich die Verzugsrechnung messen lässt: 10.000 € über 45 Tage
// bei einem Basiszinssatz von 1,27 % und neun Prozentpunkten Zuschlag sind
// 10,27 % im Jahr — 10.000 × 10,27 % × 45/365 = 126,6164… €, kaufmännisch
// 126,62 €.
func TestDefaultInterestIsCalculatedByTheDay(t *testing.T) {
	result, err := DefaultInterest(
		1_000_000, "2025-08-01", "2025-09-15", false, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.Days != 45 {
		t.Errorf("verzinste Tage = %d, erwartet 45", result.Days)
	}
	if result.Amount != 12662 {
		t.Errorf("Zinsen = %s €, erwartet 126,62", result.Amount)
	}
	if len(result.Segments) != 1 {
		t.Fatalf("erwartet einen Abschnitt, erhalten %d", len(result.Segments))
	}
	if result.Segments[0].TotalPoints != 1027 {
		t.Errorf("Zinssatz = %d Hundertstel Prozentpunkte, erwartet 1027 (1,27 %% + 9)",
			result.Segments[0].TotalPoints)
	}
}

// Gegenüber einem Verbraucher sind es fünf Prozentpunkte (§ 288 Abs. 1 BGB):
// 10.000 × 6,27 % × 45/365 = 77,3013… €.
func TestDefaultInterestUsesTheConsumerRate(t *testing.T) {
	result, err := DefaultInterest(
		1_000_000, "2025-08-01", "2025-09-15", true, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.Amount != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30", result.Amount)
	}
}

// Der Basiszinssatz wechselt zum 1. Januar und zum 1. Juli. Ein Verzug, der über
// den Wechsel läuft, wird abschnittsweise gerechnet — sonst trüge der ganze
// Zeitraum den Satz seines ersten Tages.
func TestDefaultInterestSplitsAtTheRateChange(t *testing.T) {
	result, err := DefaultInterest(
		1_000_000, "2026-06-01", "2026-08-01", false, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if len(result.Segments) != 2 {
		t.Fatalf("erwartet zwei Abschnitte über den Wechsel zum 1. Juli, erhalten %d", len(result.Segments))
	}
	first, second := result.Segments[0], result.Segments[1]
	if first.Days != 30 || second.Days != 31 {
		t.Errorf("Tage je Abschnitt = %d und %d, erwartet 30 und 31", first.Days, second.Days)
	}
	if first.TotalPoints != 1027 {
		t.Errorf("erster Abschnitt = %d, erwartet 1027 (1,27 %% + 9)", first.TotalPoints)
	}
	if second.TotalPoints != 1052 {
		t.Errorf("zweiter Abschnitt = %d, erwartet 1052 (1,52 %% + 9)", second.TotalPoints)
	}
	// 10.000 × 10,27 % × 30/365 = 84,4109… → 84,41;
	// 10.000 × 10,52 % × 31/365 = 89,3479… → 89,35.
	if first.Amount != 8441 || second.Amount != 8935 {
		t.Errorf("Beträge je Abschnitt = %s und %s €, erwartet 84,41 und 89,35",
			first.Amount, second.Amount)
	}
	if result.Amount != first.Amount+second.Amount {
		t.Errorf("die Summe %s € entspricht nicht den ausgewiesenen Abschnitten", result.Amount)
	}
}

// Der Verzug beginnt mit dem Ablauf von dreißig Tagen nach Fälligkeit; verzinst
// wird ab dem Tag danach (§ 286 Abs. 3 BGB).
func TestDefaultInterestStartIsThirtyDaysAfterTheDueDate(t *testing.T) {
	start, err := DefaultInterestStart("2026-01-31")
	if err != nil {
		t.Fatalf("Verzugsbeginn: %v", err)
	}
	if start != "2026-03-03" {
		t.Errorf("Verzugsbeginn = %s, erwartet 2026-03-03", start)
	}

	// Und vor diesem Tag läuft kein Zins: wer am dreißigsten Tag zahlt, zahlt
	// pünktlich.
	result, err := DefaultInterest(1_000_000, start, start, false, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.Amount != 0 || result.Days != 0 {
		t.Errorf("am Tag des Verzugsbeginns dürfen keine Zinsen laufen: %s € über %d Tage",
			result.Amount, result.Days)
	}
}

// Nach dem Halbjahr des letzten Eintrags kennt die Tabelle den Satz nicht.
// Die Rechnung setzt den letzten Satz fort und kennzeichnet den Zeitraum.
func TestDefaultInterestContinuesProvisionallyAfterTheTableEnds(t *testing.T) {
	rates := []BaseRatePeriod{
		{ValidFrom: "2026-01-01", BasisPoints: 127},
		{ValidFrom: "2026-07-01", BasisPoints: 152},
	}
	if got := BaseRatesCoveredUntil(rates); got != "2026-12-31" {
		t.Fatalf("die Tabelle deckt bis %s, erwartet 2026-12-31", got)
	}

	result, err := DefaultInterest(1_000_000, "2026-12-01", "2027-02-01", false, rates)
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.UncoveredFrom != "2027-01-01" {
		t.Errorf("ungedeckt ab %q, erwartet 2027-01-01", result.UncoveredFrom)
	}
	if result.Days != 62 {
		t.Errorf("verzinste Tage = %d, erwartet 62 (Dezember und Januar)", result.Days)
	}

	later, err := DefaultInterest(1_000_000, "2027-03-01", "2027-04-01", false, rates)
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if later.Amount != 8935 || later.UncoveredFrom != "2027-03-01" {
		t.Errorf("nach dem Ende der Tabelle: %s € ab %q — erwartet 89,35 Euro vorläufig ab 2027-03-01",
			later.Amount, later.UncoveredFrom)
	}
}

// BaseRateAt setzt eine aufsteigend sortierte Tabelle ohne doppelte Stichtage
// voraus.
func TestBuiltInBaseRatesAreSorted(t *testing.T) {
	rates := DefaultBaseRates()
	for i := 1; i < len(rates); i++ {
		if rates[i].ValidFrom <= rates[i-1].ValidFrom {
			t.Errorf("Stichtag %s folgt auf %s", rates[i].ValidFrom, rates[i-1].ValidFrom)
		}
	}
	rate, err := BaseRateAt(rates, "2026-08-15")
	if err != nil {
		t.Fatalf("Basiszinssatz: %v", err)
	}
	if rate.BasisPoints != 152 {
		t.Errorf("Basiszinssatz = %d, erwartet 152 ab dem 1. Juli 2026", rate.BasisPoints)
	}
}

// Vor dem ersten hinterlegten Stichtag gibt es keinen Satz — und dann sagt die
// Tabelle das, statt null zu liefern.
func TestBaseRateBeforeTheFirstEntryIsAnError(t *testing.T) {
	if _, err := BaseRateAt(DefaultBaseRates(), "2001-01-01"); err == nil {
		t.Error("für einen Tag vor dem ersten Eintrag darf kein Satz zurückkommen")
	}
}
