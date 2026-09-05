package accounting

import (
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Schätzung der Ertragsteuern zum Bilanzstichtag.
//
// Sie ist eine Schätzung und keine Steuererklärung. Was Buchfink rechnen kann,
// ist die Regel: Körperschaftsteuer auf das handelsrechtliche Ergebnis zuzüglich
// der nicht abziehbaren Betriebsausgaben, der Solidaritätszuschlag darauf und
// die Gewerbesteuer nach Messzahl und Hebesatz. Was Buchfink nicht kennt, sind
// Verlustvorträge (§ 10d EStG, § 10a GewStG), Hinzurechnungen und Kürzungen
// (§§ 8 und 9 GewStG) und verdeckte Gewinnausschüttungen. Deshalb ist das
// Ergebnis ein Vorschlag, der geändert werden darf, und der Erklärtext sagt das.

// TaxProvisionInput sind die Größen, aus denen der Vorschlag entsteht.
type TaxProvisionInput struct {
	// ProfitBeforeTax ist das Ergebnis vor Steuern aus der GuV.
	ProfitBeforeTax domain.Cents `json:"profitBeforeTax"`
	// NonDeductible sind die nicht abziehbaren Betriebsausgaben, die dem
	// Ergebnis wieder hinzuzurechnen sind (§ 4 Abs. 5 EStG, § 10 KStG).
	NonDeductible domain.Cents `json:"nonDeductible"`
	// TradeTaxRatePercent ist der Hebesatz der Gemeinde in Prozent (400 = 400 %).
	TradeTaxRatePercent int64 `json:"tradeTaxRatePercent"`
	// PrepaidCorporate und PrepaidTrade sind die bereits gebuchten
	// Vorauszahlungen. Sie mindern die Rückstellung, nicht die Steuer.
	PrepaidCorporate domain.Cents `json:"prepaidCorporate"`
	PrepaidTrade     domain.Cents `json:"prepaidTrade"`
	// Date ist der Bilanzstichtag; er wählt den Satz aus der datierten Tabelle.
	Date string `json:"date"`
}

// TaxProvisionResult ist die aufgeschlüsselte Rechnung.
type TaxProvisionResult struct {
	TaxableIncome domain.Cents `json:"taxableIncome"`
	CorporateTax  domain.Cents `json:"corporateTax"`
	Solidarity    domain.Cents `json:"solidarity"`
	// TradeIncome ist der auf volle 100 Euro abgerundete Gewerbeertrag
	// (§ 11 Abs. 1 Satz 3 GewStG), TradeBase der Steuermessbetrag.
	TradeIncome domain.Cents `json:"tradeIncome"`
	TradeBase   domain.Cents `json:"tradeBase"`
	TradeTax    domain.Cents `json:"tradeTax"`

	// IncomeProvision ist die Rückstellung für Körperschaftsteuer und
	// Solidaritätszuschlag, TradeProvision die für die Gewerbesteuer — jeweils
	// nach Abzug der Vorauszahlungen und nie negativ: eine Überzahlung ist eine
	// Forderung und keine Rückstellung.
	IncomeProvision domain.Cents `json:"incomeProvision"`
	TradeProvision  domain.Cents `json:"tradeProvision"`
	// IncomeRefund und TradeRefund sind die Überzahlungen, die stattdessen
	// entstehen. Buchfink bucht sie nicht, sondern weist sie aus.
	IncomeRefund domain.Cents `json:"incomeRefund"`
	TradeRefund  domain.Cents `json:"tradeRefund"`

	RatesUsed string `json:"ratesUsed"`
}

// CalculateTaxProvision rechnet den Vorschlag für die Steuerrückstellung.
func CalculateTaxProvision(in TaxProvisionInput) (*TaxProvisionResult, error) {
	params, err := TaxParametersFor(in.Date)
	if err != nil {
		return nil, err
	}
	if in.TradeTaxRatePercent <= 0 {
		return nil, fmt.Errorf(
			"ohne Gewerbesteuer-Hebesatz lässt sich die Gewerbesteuer nicht rechnen. " +
				"Der Hebesatz steht in den Einstellungen und ist eine Angabe der Gemeinde")
	}

	base := in.ProfitBeforeTax + in.NonDeductible
	result := &TaxProvisionResult{
		TaxableIncome: base,
		RatesUsed: fmt.Sprintf(
			"KSt %s %%, SolZ %s %%, Messzahl %s %%, Hebesatz %d %%",
			taxRateText(params.CorporateTaxPermille), taxRateText(params.SolidarityPermille),
			taxRateText(params.TradeTaxBasePermille), in.TradeTaxRatePercent),
	}
	if base <= 0 {
		// Ein Verlust erzeugt keine Steuer. Er erzeugt auch keine Erstattung —
		// die entstünde erst aus einem Verlustrücktrag, und der ist ein Antrag.
		result.IncomeRefund = in.PrepaidCorporate
		result.TradeRefund = in.PrepaidTrade
		return result, nil
	}

	result.CorporateTax = domain.MulRound(base, params.CorporateTaxPermille, 1000)
	result.Solidarity = domain.MulRound(result.CorporateTax, params.SolidarityPermille, 1000)

	// § 11 Abs. 1 Satz 3 GewStG: der Gewerbeertrag wird auf volle 100 Euro nach
	// unten abgerundet.
	result.TradeIncome = base - base%10000
	result.TradeBase = domain.MulRound(result.TradeIncome, params.TradeTaxBasePermille, 1000)
	// Kaufmännisch gerundet wie jeder andere Rechenschritt: bei einem Hebesatz,
	// der kein Vielfaches von 100 ist (415 % kommt vor), und einem Messbetrag
	// mit ungeraden Cents schnitte die Ganzzahldivision den Rest ab und ergäbe
	// eine um bis zu einen Cent zu niedrige Steuer.
	result.TradeTax = domain.MulRound(result.TradeBase, in.TradeTaxRatePercent, 100)

	income := result.CorporateTax + result.Solidarity - in.PrepaidCorporate
	if income >= 0 {
		result.IncomeProvision = income
	} else {
		result.IncomeRefund = -income
	}
	trade := result.TradeTax - in.PrepaidTrade
	if trade >= 0 {
		result.TradeProvision = trade
	} else {
		result.TradeRefund = -trade
	}
	return result, nil
}

// taxRateText schreibt einen Promillewert als Prozentzahl in deutscher
// Schreibweise: 150 ‰ wird zu „15", 55 ‰ zu „5,5".
func taxRateText(permille int64) string {
	whole := permille / 10
	frac := permille % 10
	if frac == 0 {
		return strconv.FormatInt(whole, 10)
	}
	return fmt.Sprintf("%d,%d", whole, frac)
}
