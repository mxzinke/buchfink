package accounting

import (
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Schätzung der Ertragsteuern zum Bilanzstichtag.
//
// Sie ist keine Steuererklärung. Gerechnet wird der Regelweg einer GmbH: das
// Ergebnis vor Ertragsteuern, zuzüglich der nicht abziehbaren Aufwendungen,
// abzüglich der steuerfreien Beteiligungserträge (zuzüglich 5 % davon), der
// Spendenabzug in den Höchstbeträgen, der Verlustvortrag mit
// Mindestbesteuerung, darauf Körperschaftsteuer, Solidaritätszuschlag und
// Gewerbesteuer. Hinzurechnungen und Kürzungen der Gewerbesteuer (§§ 8, 9
// GewStG), verdeckte Gewinnausschüttungen und Verlustvorträge sind Angaben, die
// nicht in den Büchern stehen; sie kommen als Eingabe.

// TaxProvisionInput sind die Größen, aus denen der Vorschlag entsteht.
type TaxProvisionInput struct {
	// ProfitBeforeTax ist das Ergebnis vor Ertragsteuern aus der GuV.
	ProfitBeforeTax domain.Cents `json:"profitBeforeTax"`
	// NonDeductible sind die nicht abziehbaren Aufwendungen, die dem Ergebnis
	// wieder hinzuzurechnen sind (§ 4 Abs. 5 EStG, § 10 KStG, § 8b Abs. 3
	// Satz 3 KStG).
	NonDeductible domain.Cents `json:"nonDeductible"`
	// ExemptDividends sind Bezüge nach § 8b Abs. 1 KStG, ExemptGains Gewinne
	// nach § 8b Abs. 2 KStG; beide bleiben außer Ansatz, 5 % davon gelten als
	// nicht abziehbar (§ 8b Abs. 3 Satz 1, Abs. 5 Satz 1 KStG).
	ExemptDividends domain.Cents `json:"exemptDividends"`
	ExemptGains     domain.Cents `json:"exemptGains"`
	// Donations sind die gebuchten Zuwendungen, die in den Höchstbeträgen des
	// § 9 Abs. 1 Satz 1 Nr. 2 KStG abziehbar sind; TurnoverAndWages ist die
	// Summe aus Umsätzen und Löhnen für den Höchstbetrag von 4 ‰.
	Donations        domain.Cents `json:"donations"`
	TurnoverAndWages domain.Cents `json:"turnoverAndWages"`
	// LossCarryforwardCorporate ist der verbleibende Verlustvortrag zur
	// Körperschaftsteuer laut letztem Feststellungsbescheid (§ 10d Abs. 4
	// EStG), LossCarryforwardTrade der vortragsfähige Gewerbeverlust
	// (§ 10a Satz 6 GewStG).
	LossCarryforwardCorporate domain.Cents `json:"lossCarryforwardCorporate"`
	LossCarryforwardTrade     domain.Cents `json:"lossCarryforwardTrade"`
	// TradeAdditions und TradeDeductions sind die Hinzurechnungen nach § 8
	// GewStG und die Kürzungen nach § 9 GewStG.
	TradeAdditions  domain.Cents `json:"tradeAdditions"`
	TradeDeductions domain.Cents `json:"tradeDeductions"`
	// TradeTaxRatePercent ist der Hebesatz der Gemeinde in Prozent (400 = 400 %).
	TradeTaxRatePercent int64 `json:"tradeTaxRatePercent"`
	// PrepaidCorporate, PrepaidSolidarity und PrepaidTrade sind die
	// Vorauszahlungen für dieses Jahr (Konten 7600, 7608, 7610).
	PrepaidCorporate  domain.Cents `json:"prepaidCorporate"`
	PrepaidSolidarity domain.Cents `json:"prepaidSolidarity"`
	PrepaidTrade      domain.Cents `json:"prepaidTrade"`
	// CreditableWithholding und CreditableSolidarity sind die einbehaltene
	// Kapitalertragsteuer und der Solidaritätszuschlag darauf (Konten 7630,
	// 7633). Sie werden angerechnet (§ 36 Abs. 2 Nr. 2 EStG).
	CreditableWithholding domain.Cents `json:"creditableWithholding"`
	CreditableSolidarity  domain.Cents `json:"creditableSolidarity"`
	// Date ist der Bilanzstichtag; sein Kalenderjahr ist der
	// Veranlagungszeitraum (§ 7 Abs. 4 Satz 2 KStG) und wählt die Werte aus der
	// datierten Tabelle.
	Date string `json:"date"`
}

// TaxProvisionResult ist die aufgeschlüsselte Rechnung.
type TaxProvisionResult struct {
	// ExemptIncome sind die steuerfreien Beteiligungserträge, ExemptAddBack
	// die 5 % davon, die als nicht abziehbar gelten.
	ExemptIncome  domain.Cents `json:"exemptIncome"`
	ExemptAddBack domain.Cents `json:"exemptAddBack"`
	// IncomeBeforeDonations ist das Einkommen vor Spenden- und Verlustabzug
	// (§ 9 Abs. 2 Satz 1 KStG).
	IncomeBeforeDonations domain.Cents `json:"incomeBeforeDonations"`
	// DonationsDeducted ist der abziehbare Teil der Zuwendungen,
	// DonationsCarried der Rest, der vorgetragen wird (§ 9 Abs. 1 Satz 1 Nr. 2
	// Satz 9 KStG).
	DonationsDeducted domain.Cents `json:"donationsDeducted"`
	DonationsCarried  domain.Cents `json:"donationsCarried"`
	// IncomeBeforeLoss ist der Gesamtbetrag vor dem Verlustabzug.
	IncomeBeforeLoss domain.Cents `json:"incomeBeforeLoss"`
	// LossDeducted ist der abgezogene Verlustvortrag, LossRemaining der
	// Vortrag danach, um einen Verlust dieses Jahres erhöht.
	LossDeducted  domain.Cents `json:"lossDeducted"`
	LossRemaining domain.Cents `json:"lossRemaining"`
	// TaxableIncome ist das zu versteuernde Einkommen; negativ bei einem
	// Verlust.
	TaxableIncome domain.Cents `json:"taxableIncome"`
	CorporateTax  domain.Cents `json:"corporateTax"`
	Solidarity    domain.Cents `json:"solidarity"`

	// TradeIncomeBeforeLoss ist der Gewerbeertrag vor dem Abzug der
	// Fehlbeträge, TradeLossDeducted und TradeLossRemaining wie bei der
	// Körperschaftsteuer.
	TradeIncomeBeforeLoss domain.Cents `json:"tradeIncomeBeforeLoss"`
	TradeLossDeducted     domain.Cents `json:"tradeLossDeducted"`
	TradeLossRemaining    domain.Cents `json:"tradeLossRemaining"`
	// TradeIncome ist der auf volle 100 Euro abgerundete Gewerbeertrag
	// (§ 11 Abs. 1 Satz 3 GewStG), TradeBase der Steuermessbetrag.
	TradeIncome domain.Cents `json:"tradeIncome"`
	TradeBase   domain.Cents `json:"tradeBase"`
	TradeTax    domain.Cents `json:"tradeTax"`

	// CorporateDue und SolidarityDue sind Körperschaftsteuer und
	// Solidaritätszuschlag nach Vorauszahlungen und Anrechnung; negativ bei
	// einer Überzahlung.
	CorporateDue  domain.Cents `json:"corporateDue"`
	SolidarityDue domain.Cents `json:"solidarityDue"`
	// IncomeProvision ist die Rückstellung für Körperschaftsteuer und
	// Solidaritätszuschlag, TradeProvision die für die Gewerbesteuer.
	// IncomeRefund und TradeRefund sind die Erstattungsansprüche, wo die
	// Vorauszahlungen und die anrechenbaren Beträge die Steuer übersteigen.
	// Je Steuerart ist nur eines von beiden größer als null.
	IncomeProvision domain.Cents `json:"incomeProvision"`
	TradeProvision  domain.Cents `json:"tradeProvision"`
	IncomeRefund    domain.Cents `json:"incomeRefund"`
	TradeRefund     domain.Cents `json:"tradeRefund"`

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
				"Der Hebesatz ist eine Angabe der Gemeinde (§ 16 Abs. 1 GewStG); er steht im " +
				"Gewerbesteuerbescheid und wird in den Abschluss-Einstellungen hinterlegt")
	}
	if in.TradeTaxRatePercent < params.TradeTaxMinRatePercent {
		return nil, fmt.Errorf(
			"der Hebesatz von %d %% liegt unter dem gesetzlichen Mindesthebesatz von %d %% "+
				"(§ 16 Abs. 4 Satz 2 GewStG); bitte den Hebesatz aus dem Gewerbesteuerbescheid angeben",
			in.TradeTaxRatePercent, params.TradeTaxMinRatePercent)
	}
	for _, v := range []struct {
		label string
		value domain.Cents
	}{
		{"Verlustvortrag zur Körperschaftsteuer", in.LossCarryforwardCorporate},
		{"Gewerbeverlust", in.LossCarryforwardTrade},
		{"Hinzurechnungen nach § 8 GewStG", in.TradeAdditions},
		{"Kürzungen nach § 9 GewStG", in.TradeDeductions},
		{"steuerfreie Bezüge", in.ExemptDividends},
		{"steuerfreie Veräußerungsgewinne", in.ExemptGains},
		{"Zuwendungen", in.Donations},
	} {
		if v.value < 0 {
			return nil, fmt.Errorf("%s kann nicht negativ sein (%s €)", v.label, v.value)
		}
	}

	result := &TaxProvisionResult{
		RatesUsed: fmt.Sprintf(
			"KSt %s %%, SolZ %s %%, Messzahl %s %%, Hebesatz %d %%",
			taxRateText(params.CorporateTaxPermille), taxRateText(params.SolidarityPermille),
			taxRateText(params.TradeTaxBasePermille), in.TradeTaxRatePercent),
	}

	// Einkommen vor Spenden und Verlustabzug.
	exempt := in.ExemptDividends + in.ExemptGains
	result.ExemptIncome = exempt
	result.ExemptAddBack = domain.MulRound(exempt, params.ExemptIncomeNonDeductiblePermille, 1000)
	result.IncomeBeforeDonations = in.ProfitBeforeTax + in.NonDeductible + in.Donations -
		exempt + result.ExemptAddBack

	// § 9 Abs. 1 Satz 1 Nr. 2 KStG: abziehbar bis zum höheren der beiden
	// Höchstbeträge, der Rest wird vorgetragen. Die Schätzung lässt aus
	// Zuwendungen keinen Verlust entstehen.
	if in.Donations > 0 {
		limit := maxCents(
			truncPermille(maxCents(result.IncomeBeforeDonations, 0), params.DonationIncomePermille),
			truncPermille(maxCents(in.TurnoverAndWages, 0), params.DonationTurnoverPermille))
		limit = minCents(limit, maxCents(result.IncomeBeforeDonations, 0))
		result.DonationsDeducted = minCents(in.Donations, limit)
		result.DonationsCarried = in.Donations - result.DonationsDeducted
	}
	result.IncomeBeforeLoss = result.IncomeBeforeDonations - result.DonationsDeducted

	result.LossDeducted = lossOffset(result.IncomeBeforeLoss, in.LossCarryforwardCorporate,
		params.LossDeductionBase, params.LossDeductionExcessPermille)
	result.TaxableIncome = result.IncomeBeforeLoss - result.LossDeducted
	result.LossRemaining = in.LossCarryforwardCorporate - result.LossDeducted
	if result.IncomeBeforeLoss < 0 {
		result.LossRemaining += -result.IncomeBeforeLoss
	}

	if result.TaxableIncome > 0 {
		result.CorporateTax = domain.MulRound(result.TaxableIncome, params.CorporateTaxPermille, 1000)
		// § 4 Satz 3 SolZG: Bruchteile eines Cents bleiben außer Ansatz.
		result.Solidarity = truncPermille(result.CorporateTax, params.SolidarityPermille)
	}

	// Gewerbeertrag: der Gewinn nach den Vorschriften des KStG (§ 7 Satz 1
	// GewStG), vermehrt und vermindert um §§ 8, 9 GewStG, gekürzt um die
	// Fehlbeträge (§ 10a GewStG).
	result.TradeIncomeBeforeLoss = result.IncomeBeforeLoss + in.TradeAdditions - in.TradeDeductions
	result.TradeLossDeducted = lossOffset(result.TradeIncomeBeforeLoss, in.LossCarryforwardTrade,
		params.LossDeductionBase, params.TradeLossExcessPermille)
	result.TradeLossRemaining = in.LossCarryforwardTrade - result.TradeLossDeducted
	if result.TradeIncomeBeforeLoss < 0 {
		result.TradeLossRemaining += -result.TradeIncomeBeforeLoss
	}
	if trade := result.TradeIncomeBeforeLoss - result.TradeLossDeducted; trade > 0 {
		// § 11 Abs. 1 Satz 3 GewStG: auf volle 100 Euro nach unten abrunden.
		result.TradeIncome = trade - trade%10000
		result.TradeBase = domain.MulRound(result.TradeIncome, params.TradeTaxBasePermille, 1000)
		// Kaufmännisch gerundet: bei einem Hebesatz, der kein Vielfaches von
		// 100 ist, und einem Messbetrag mit ungeraden Cents schnitte die
		// Ganzzahldivision einen Cent ab.
		result.TradeTax = domain.MulRound(result.TradeBase, in.TradeTaxRatePercent, 100)
	}

	result.CorporateDue = result.CorporateTax - in.PrepaidCorporate - in.CreditableWithholding
	result.SolidarityDue = result.Solidarity - in.PrepaidSolidarity - in.CreditableSolidarity
	if income := result.CorporateDue + result.SolidarityDue; income >= 0 {
		result.IncomeProvision = income
	} else {
		result.IncomeRefund = -income
	}
	if trade := result.TradeTax - in.PrepaidTrade; trade >= 0 {
		result.TradeProvision = trade
	} else {
		result.TradeRefund = -trade
	}
	return result, nil
}

// lossOffset ist der Verlustabzug mit Mindestbesteuerung: bis zum Sockel voll,
// darüber bis zum Anteil des übersteigenden Betrags (§ 10d Abs. 2 Satz 1 EStG,
// § 10a Satz 1 und 2 GewStG).
func lossOffset(income, carryforward, base domain.Cents, excessPermille int64) domain.Cents {
	if income <= 0 || carryforward <= 0 {
		return 0
	}
	limit := minCents(income, base)
	if income > base {
		limit += truncPermille(income-base, excessPermille)
	}
	return minCents(limit, carryforward)
}

// SplitIncomeTax verteilt einen Betrag für Körperschaftsteuer und
// Solidaritätszuschlag auf die beiden Konten. Haben die Teile verschiedene
// Vorzeichen, nimmt der eine den anderen auf; die Summe bleibt der Betrag.
func SplitIncomeTax(corporate, solidarity domain.Cents) (domain.Cents, domain.Cents) {
	total := corporate + solidarity
	switch {
	case total >= 0 && corporate < 0:
		return 0, total
	case total >= 0 && solidarity < 0:
		return total, 0
	case total < 0 && corporate > 0:
		return 0, total
	case total < 0 && solidarity > 0:
		return total, 0
	}
	return corporate, solidarity
}

func truncPermille(amount domain.Cents, permille int64) domain.Cents {
	return domain.Cents(int64(amount) * permille / 1000)
}

func minCents(a, b domain.Cents) domain.Cents {
	if a < b {
		return a
	}
	return b
}

func maxCents(a, b domain.Cents) domain.Cents {
	if a > b {
		return a
	}
	return b
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
