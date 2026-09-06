package domain

import (
	"strings"
	"testing"
)

// § 13b UStG ist ein Katalog und kein Fall. Buchfink rechnet den des Absatzes 2
// Nr. 1 — die Leistung eines im Ausland ansässigen Unternehmers, ohne
// Betragsgrenze. Wer den Steuerfall wählt, muss im Erklärtext lesen können, was
// damit *nicht* geprüft ist: die 5.000-Euro-Grenze der Nummern 10 und 11 und die
// Bauleistungen der Nummer 4 samt dem Steuerabzug nach § 48 EStG.
//
// Ohne diesen Hinweis liest sich „Reverse Charge" wie eine Zusage, den ganzen
// § 13b abzudecken — und der Anwender wählte den Steuerfall bei einem Kauf von
// Mobiltelefonen über 4.000 €, bei dem er gar nicht greift.
func TestReverseChargeHintNamesItsLimits(t *testing.T) {
	required := []string{
		"§ 13b Abs. 2 Nr. 1",
		"5.000",
		"Nr. 10 und 11",
		"Mobilfunkgeräte",
		"§ 48",
	}
	cases := []struct {
		direction Direction
		treatment TaxTreatment
	}{
		{DirectionIncoming, TaxTreatmentReverseCharge},
		{DirectionOutgoing, TaxTreatmentReverseChargeSupply},
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
		for _, want := range required {
			if !strings.Contains(hint, want) {
				t.Errorf("%s: der Erklärtext nennt %q nicht: %s", tc.treatment, want, hint)
			}
		}
	}
}
