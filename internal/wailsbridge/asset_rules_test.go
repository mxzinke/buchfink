package wailsbridge

import (
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Methodenliste der Maske führt jede Methode, die der Rechenkern kennt.
//
// Sie ist die einzige Quelle der Auswahl im Anlagenformular. Fehlt eine Methode
// darin, gibt es sie für den Anwender nicht — die Staffel des § 7 Abs. 2a EStG
// und die festen Gebäudesätze des § 7 Abs. 4 EStG waren genau so eine Weile im
// Programm vorhanden und unerreichbar, und ein E-Fahrzeug lief still linear.
func TestAssetRulesOfferEveryMethodTheCoreKnows(t *testing.T) {
	b := newBridge(t)
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

// Die Investmentregeln nennen je Fondsart den Satz der Teilfreistellung, wie
// er aus der Anlegerstellung der Rechtsform folgt, und woher die
// Anlegerstellung kommt. Eine GmbH unterliegt dem KStG: 80 % für Aktienfonds,
// die Hälfte für Mischfonds, 60 % für Immobilienfonds unabhängig vom Anleger.
func TestInvestmentRulesFollowTheLegalForm(t *testing.T) {
	b := testBridge(t)
	rules, err := b.GetInvestmentRules()
	if err != nil {
		t.Fatalf("Regeln lesen: %v", err)
	}
	if len(rules.FundClasses) != len(accounting.AllFundClasses()) || len(rules.InvestorTypes) == 0 {
		t.Fatalf("Auswahl unvollständig: %d Fondsarten, %d Anlegerstellungen",
			len(rules.FundClasses), len(rules.InvestorTypes))
	}
	if rules.InvestorType != domain.InvestorCorporate || rules.InvestorReason == "" {
		t.Errorf("die GmbH ist Körperschaft, mit Begründung: %q, %q", rules.InvestorType, rules.InvestorReason)
	}

	byClass := map[accounting.FundClass]exemptionInfo{}
	for _, e := range rules.Exemptions {
		byClass[e.Class] = e
	}
	for _, class := range accounting.AllFundClasses() {
		if _, ok := byClass[class]; ok != class.IsFund() {
			t.Errorf("%q: in der Liste %v, Investmentanteil %v", class, ok, class.IsFund())
		}
	}
	for class, want := range map[accounting.FundClass]int{
		accounting.FundEquity: 800, accounting.FundMixed: 400, accounting.FundRealEstate: 600, accounting.FundOther: 0,
	} {
		e := byClass[class]
		if e.Problem != "" || e.Permille != want || e.Source == "" {
			t.Errorf("%q: erwartet %d Promille mit Quelle, erhalten %+v", class, want, e)
		}
	}
}
