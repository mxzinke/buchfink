package domain

import (
	"strings"
	"testing"
)

// § 13b UStG ist ein Katalog und kein Fall. Wer den Steuerfall wählt, muss im
// Erklärtext lesen können, welcher Absatz gemeint ist und was damit *nicht*
// geprüft ist: die 5.000-Euro-Grenze der Nummern 10 und 11 und die
// Bauleistungen der Nummer 4 samt dem Steuerabzug nach § 48 EStG.
func TestReverseChargeHintNamesItsLimits(t *testing.T) {
	limits := []string{"5.000", "Nr. 10 und 11", "Mobilfunkgeräte", "§ 48"}
	cases := []struct {
		direction Direction
		treatment TaxTreatment
		required  []string
	}{
		{DirectionIncoming, TaxTreatmentReverseCharge, append([]string{"§ 13b Abs. 2 Nr. 1", "Kennzahl 84"}, limits...)},
		{DirectionIncoming, TaxTreatmentReverseChargeEU, []string{"§ 13b Abs. 1", "§ 3a Abs. 2", "Kennzahl 46"}},
		{DirectionOutgoing, TaxTreatmentReverseChargeSupply, append([]string{"Kennzahl 21", "Kennzahl 60"}, limits...)},
	}
	for _, tc := range cases {
		var hint string
		for _, info := range TaxTreatments(tc.direction) {
			if info.Treatment == tc.treatment {
				hint = info.Hint
			}
		}
		if hint == "" {
			t.Fatalf("%s: kein Erklärtext zum Steuerfall %s", tc.direction, tc.treatment)
		}
		for _, want := range tc.required {
			if !strings.Contains(hint, want) {
				t.Errorf("%s: der Erklärtext nennt %q nicht: %s", tc.treatment, want, hint)
			}
		}
	}
}
