package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Rechenbeispiel: Ergebnis vor Steuern 100.000 €, nicht abziehbare
// Betriebsausgaben 1.000 €, Hebesatz 400 %.
//
//	Bemessungsgrundlage  101.000,00 €
//	Körperschaftsteuer    15 %      15.150,00 €
//	Solidaritätszuschlag  5,5 %        833,25 €
//	Gewerbeertrag                  101.000,00 € (auf volle 100 € abgerundet)
//	Messbetrag            3,5 %      3.535,00 €
//	Gewerbesteuer         400 %     14.140,00 €
func TestTaxProvisionFollowsTheStatutoryRates(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, NonDeductible: 100_000,
		TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Steuerrückstellung rechnen: %v", err)
	}
	if result.TaxableIncome != 10_100_000 {
		t.Errorf("Bemessungsgrundlage %s € — erwartet 101.000,00 €", result.TaxableIncome)
	}
	if result.CorporateTax != 1_515_000 {
		t.Errorf("Körperschaftsteuer %s € — erwartet 15.150,00 €", result.CorporateTax)
	}
	if result.Solidarity != 83_325 {
		t.Errorf("Solidaritätszuschlag %s € — erwartet 833,25 €", result.Solidarity)
	}
	if result.TradeBase != 353_500 {
		t.Errorf("Steuermessbetrag %s € — erwartet 3.535,00 €", result.TradeBase)
	}
	if result.TradeTax != 1_414_000 {
		t.Errorf("Gewerbesteuer %s € — erwartet 14.140,00 €", result.TradeTax)
	}
	if result.IncomeProvision != result.CorporateTax+result.Solidarity {
		t.Errorf("Rückstellung %s € — ohne Vorauszahlungen ist sie die volle Steuer", result.IncomeProvision)
	}
}

// § 11 Abs. 1 Satz 3 GewStG rundet den Gewerbeertrag auf volle 100 Euro ab. Ohne
// die Abrundung wäre der Messbetrag um Cent-Beträge zu hoch.
func TestTradeIncomeIsRoundedDownToFullHundreds(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_009_900, TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Steuerrückstellung rechnen: %v", err)
	}
	if result.TradeIncome != 10_000_000 {
		t.Errorf("Gewerbeertrag %s € — erwartet 100.000,00 € nach Abrundung", result.TradeIncome)
	}
}

// Vorauszahlungen mindern die Rückstellung, nicht die Steuer. Wo sie sie
// übersteigen, entsteht eine Forderung und keine negative Rückstellung.
func TestPrepaymentsReduceTheProvision(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, NonDeductible: 100_000, TradeTaxRatePercent: 400,
		PrepaidCorporate: 1_000_000, PrepaidTrade: 2_000_000, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Steuerrückstellung rechnen: %v", err)
	}
	if result.CorporateTax != 1_515_000 {
		t.Errorf("die Steuer selbst darf sich nicht ändern: %s €", result.CorporateTax)
	}
	if want := domain.Cents(1_515_000 + 83_325 - 1_000_000); result.IncomeProvision != want {
		t.Errorf("Rückstellung %s € — erwartet %s €", result.IncomeProvision, want)
	}
	if result.TradeProvision != 0 {
		t.Errorf("Gewerbesteuerrückstellung %s € — die Vorauszahlung übersteigt die Steuer",
			result.TradeProvision)
	}
	if want := domain.Cents(2_000_000 - 1_414_000); result.TradeRefund != want {
		t.Errorf("Erstattungsanspruch %s € — erwartet %s €", result.TradeRefund, want)
	}
}

// Ein Verlust erzeugt keine Steuer.
func TestALossCarriesNoTax(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: -5_000_000, TradeTaxRatePercent: 400, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Steuerrückstellung rechnen: %v", err)
	}
	if result.CorporateTax != 0 || result.TradeTax != 0 || result.IncomeProvision != 0 {
		t.Errorf("aus einem Verlust darf keine Steuer entstehen: %+v", result)
	}
}

// Ohne Hebesatz lässt sich die Gewerbesteuer nicht rechnen, und geraten wird
// nicht: der Hebesatz ist eine Angabe der Gemeinde.
func TestTaxProvisionNeedsTheMunicipalRate(t *testing.T) {
	if _, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_000_000, Date: "2026-12-31",
	}); err == nil {
		t.Error("ohne Hebesatz darf keine Gewerbesteuer entstehen")
	}
}

// Die Gewerbesteuer wird kaufmännisch gerundet wie jeder andere Rechenschritt.
// Bei einem Hebesatz, der kein Vielfaches von 100 ist, und einem Messbetrag mit
// ungeraden Cents schnitte die Ganzzahldivision den Rest ab.
func TestTradeTaxIsRoundedNotTruncated(t *testing.T) {
	result, err := CalculateTaxProvision(TaxProvisionInput{
		ProfitBeforeTax: 10_015_700, TradeTaxRatePercent: 415, Date: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("Steuerrückstellung rechnen: %v", err)
	}
	// Gewerbeertrag auf volle 100 € abgerundet: 100.100,00 €. Messbetrag
	// 3,5 % davon = 3.503,50 €, davon 415 % = 14.539,53 € (abgeschnitten wären
	// es 14.539,52 €).
	if result.TradeIncome != 10_010_000 || result.TradeBase != 350_350 {
		t.Fatalf("Gewerbeertrag %s €, Messbetrag %s €", result.TradeIncome, result.TradeBase)
	}
	if want := domain.MulRound(result.TradeBase, 415, 100); result.TradeTax != want {
		t.Errorf("Gewerbesteuer %s € — erwartet %s €", result.TradeTax, want)
	}
	if result.TradeTax != 1_453_953 {
		t.Errorf("Gewerbesteuer %s € — erwartet 14.539,53 €", result.TradeTax)
	}
}
