package accounting

import "testing"

// Der Jahresanteil wird mit den Monaten gewichtet, die in den
// Berichtigungszeitraum fallen. Ohne diese Gewichtung wäre das Anfangs- oder
// das Schlussjahr eines mitten im Jahr begonnenen Zeitraums mit einem vollen
// Fünftel berichtigt worden — beim Pkw des Lehrbuchs 608 statt 304 Euro.
func TestInputTaxCorrectionWeighsPartialYears(t *testing.T) {
	params, err := TaxParametersFor("2031-12-31")
	if err != nil {
		t.Fatalf("Steuerparameter: %v", err)
	}
	req := InputTaxCorrectionRequest{
		InputTaxAmount:   760_000,
		OriginalPermille: 1000,
		CurrentPermille:  600,
		PeriodYears:      5,
	}

	full, err := AssessInputTaxCorrection(req, params)
	if err != nil {
		t.Fatalf("volles Jahr: %v", err)
	}
	if full.Amount != -60_800 {
		t.Errorf("volles Jahr: %s € — erwartet -608,00 €", full.Amount)
	}

	req.MonthsInYear = 6
	half, err := AssessInputTaxCorrection(req, params)
	if err != nil {
		t.Fatalf("halbes Jahr: %v", err)
	}
	if half.Amount != -30_400 {
		t.Errorf("sechs Monate: %s € — erwartet -304,00 € (7.600/5 × 6/12 × 40 %%)", half.Amount)
	}
	if !half.Required {
		t.Error("auch die anteilige Berichtigung ist vorzunehmen")
	}

	// Ohne Angabe der Monate bleibt es beim vollen Jahr: der Aufrufer, der sie
	// nicht kennt, meint den Regelfall.
	req.MonthsInYear = 0
	if again, err := AssessInputTaxCorrection(req, params); err != nil || again.Amount != -60_800 {
		t.Errorf("ohne Monatsangabe %s € (%v) — erwartet das volle Jahr", again.Amount, err)
	}
}

// Die Bagatellgrenze des § 44 Abs. 2 UStDV misst am gewichteten Betrag: ein
// Anfangsjahr mit wenigen Monaten kann darunter bleiben, wo das volle Jahr
// darüber läge.
func TestInputTaxCorrectionMinorLimitUsesTheWeightedAmount(t *testing.T) {
	params, err := TaxParametersFor("2026-12-31")
	if err != nil {
		t.Fatalf("Steuerparameter: %v", err)
	}
	req := InputTaxCorrectionRequest{
		InputTaxAmount:   1_500_000,
		OriginalPermille: 1000,
		CurrentPermille:  950,
		PeriodYears:      10,
	}

	// Volles Jahr: 1.500.000 / 10 × 5 % = 75,00 € — unter 1.000 € und unter
	// zehn Prozentpunkten, also keine Berichtigung.
	full, err := AssessInputTaxCorrection(req, params)
	if err != nil {
		t.Fatalf("volles Jahr: %v", err)
	}
	if full.Required {
		t.Errorf("bei 75,00 € und 5 Punkten entfällt die Berichtigung: %s", full.Reason)
	}

	// Ein Monat: derselbe Grund, nur noch kleiner — und der Text nennt die
	// Monate, damit niemand den Betrag für ein volles Jahr hält.
	req.MonthsInYear = 1
	one, err := AssessInputTaxCorrection(req, params)
	if err != nil {
		t.Fatalf("ein Monat: %v", err)
	}
	if one.Required {
		t.Errorf("auch anteilig bleibt es unter der Grenze: %s", one.Reason)
	}
}
