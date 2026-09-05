package wailsbridge

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Methodenliste der Maske führt jede Methode, die der Rechenkern kennt.
//
// Sie ist die einzige Quelle der Auswahl im Anlagenformular. Fehlt eine Methode
// darin, gibt es sie für den Anwender nicht — die Staffel des § 7 Abs. 2a EStG
// und die festen Gebäudesätze des § 7 Abs. 4 EStG waren genau so eine Weile im
// Programm vorhanden und unerreichbar, und ein E-Fahrzeug lief still linear.
func TestAssetRulesOfferEveryMethodTheCoreKnows(t *testing.T) {
	b := &BuchfinkBridge{currentYear: 2026}
	rules, err := b.GetAssetRules()
	if err != nil {
		t.Fatalf("Regeln lesen: %v", err)
	}

	offered := map[domain.DepreciationMethod][]domain.AssetClass{}
	for _, m := range rules.Methods {
		if m.Label == "" || m.Hint == "" {
			t.Errorf("Methode %q ohne Bezeichnung oder Hinweis", m.Method)
		}
		offered[m.Method] = m.Classes
	}

	for _, method := range []domain.DepreciationMethod{
		domain.DepreciationLinear,
		domain.DepreciationDegressive,
		domain.DepreciationElectricVehicle,
		domain.DepreciationBuildingLinear,
		domain.DepreciationPool,
		domain.DepreciationImmediate,
		domain.DepreciationNone,
	} {
		if _, ok := offered[method]; !ok {
			t.Errorf("die Methode %q steht in keiner Auswahl", method)
		}
	}

	// Beide neuen Methoden gehören zum Sachanlagevermögen: die Staffel gilt für
	// Fahrzeuge, die festen Sätze für Gebäude.
	for _, method := range []domain.DepreciationMethod{
		domain.DepreciationElectricVehicle, domain.DepreciationBuildingLinear,
	} {
		classes := offered[method]
		if len(classes) != 1 || classes[0] != domain.AssetClassTangible {
			t.Errorf("%q ist für %v angeboten, gehört aber allein zum Sachanlagevermögen",
				method, classes)
		}
	}
}
