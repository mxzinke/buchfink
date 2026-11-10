package accounting

import (
	"testing"
)

// -------------------------------------------------------------------------
// Vorsteuerberichtigung nach § 15a UStG
// -------------------------------------------------------------------------

func params2026(t *testing.T) TaxParameters {
	t.Helper()
	p, err := TaxParametersFor("2026-12-31")
	if err != nil {
		t.Fatalf("steuerliche Werte: %v", err)
	}
	return p
}

// Der Lehrbuchfall: ein Pkw für 47.600 € brutto, 7.600 € Vorsteuer, zunächst zu
// 100 % für abziehbare Umsätze verwendet, im dritten Jahr nur noch zu 60 %.
// Ein Fünftel der Vorsteuer entfällt auf das Jahr, und davon sind 40 % zurück-
// zuzahlen: 7.600 / 5 × 40 % = 608 €.
func TestInputTaxCorrectionForACarWithChangedUse(t *testing.T) {
	got, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   760_000,
		OriginalPermille: 1000,
		CurrentPermille:  600,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, params2026(t))
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if !got.Required {
		t.Fatalf("bei einer Änderung von 100 auf 60 %% ist zu berichtigen: %s", got.Reason)
	}
	if got.Amount != -60_800 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet -608,00 € (7.600 / 5 × 40 %%)", got.Amount)
	}
	if got.Account != InputTaxCorrectionRepayableMovable {
		t.Errorf("Konto %s — die zurückzuzahlende Vorsteuer eines beweglichen Wirtschaftsguts "+
			"gehört auf %s", got.Account, InputTaxCorrectionRepayableMovable)
	}
	// Unter 6.000 €: nach § 44 Abs. 3 UStDV erst bei der Steuerberechnung für das
	// Kalenderjahr.
	if !got.DeferToAnnual {
		t.Error("ein Betrag unter 6.000 € wird erst zum Jahresende berichtigt (§ 44 Abs. 3 UStDV)")
	}
}

// Die Gegenrichtung: steigt die abziehbare Verwendung, wird nachträglich
// Vorsteuer abziehbar, und der Betrag geht auf das andere Konto.
func TestInputTaxCorrectionCanFavourTheTaxpayer(t *testing.T) {
	got, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   760_000,
		OriginalPermille: 600,
		CurrentPermille:  1000,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, params2026(t))
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if got.Amount != 60_800 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet +608,00 €", got.Amount)
	}
	if got.Account != InputTaxCorrectionDeductibleMovable {
		t.Errorf("Konto %s — erwartet %s", got.Account, InputTaxCorrectionDeductibleMovable)
	}
}

// § 44 Abs. 1 UStDV: bis 1.000 € Vorsteuer je Wirtschaftsgut entfällt die
// Berichtigung ganz. § 44 Abs. 2 UStDV: unter zehn Prozentpunkten Änderung
// *und* bis 1.000 € Betrag entfällt sie für das Jahr.
func TestInputTaxCorrectionMinorLimits(t *testing.T) {
	p := params2026(t)

	small, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   90_000, // 900 €
		OriginalPermille: 1000,
		CurrentPermille:  500,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, p)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if small.Required || small.Amount != 0 {
		t.Errorf("bei 900 € Vorsteuer entfällt die Berichtigung nach § 44 Abs. 1 UStDV; "+
			"erhalten: %s € (%s)", small.Amount, small.Reason)
	}

	// Acht Prozentpunkte auf eine Vorsteuer, deren Jahresanteil klein genug ist:
	// 12.500 / 5 × 8 % = 200 €.
	minor, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   1_250_000,
		OriginalPermille: 1000,
		CurrentPermille:  920,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, p)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if minor.Required {
		t.Errorf("acht Prozentpunkte und 200 € bleiben unter beiden Grenzen des § 44 Abs. 2 UStDV; "+
			"erhalten: %s € (%s)", minor.Amount, minor.Reason)
	}

	// Dieselben acht Prozentpunkte, aber ein Betrag über 1.000 €: dann wird
	// berichtigt, weil § 44 Abs. 2 UStDV beide Bedingungen zusammen verlangt.
	big, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   10_000_000,
		OriginalPermille: 1000,
		CurrentPermille:  920,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, p)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if !big.Required {
		t.Errorf("eine kleine Änderung mit großem Betrag wird berichtigt: %s", big.Reason)
	}
	if big.Amount != -160_000 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet -1.600,00 €", big.Amount)
	}
}

// Grundstücke und Gebäude haben einen Zeitraum von zehn Jahren und eigene
// Konten.
func TestInputTaxCorrectionPeriodForBuildings(t *testing.T) {
	if got := CorrectionPeriodYears(true); got != 10 {
		t.Errorf("Berichtigungszeitraum für ein Grundstück: %d Jahre — erwartet zehn", got)
	}
	if got := CorrectionPeriodYears(false); got != 5 {
		t.Errorf("Berichtigungszeitraum für ein bewegliches Wirtschaftsgut: %d Jahre — erwartet fünf", got)
	}
	got, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   19_000_00,
		OriginalPermille: 1000,
		CurrentPermille:  0,
		PeriodYears:      CorrectionPeriodImmovableYears,
		Immovable:        true,
	}, params2026(t))
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if got.Account != InputTaxCorrectionRepayableImmovable {
		t.Errorf("Konto %s — ein unbewegliches Wirtschaftsgut gehört auf %s",
			got.Account, InputTaxCorrectionRepayableImmovable)
	}
	if got.Amount != -19_000_00/10 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet ein Zehntel der Vorsteuer", got.Amount)
	}
}
