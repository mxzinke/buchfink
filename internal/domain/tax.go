package domain

import "fmt"

// Direction distinguishes an incoming document (Eingangsbeleg, Aufwand oder
// Zugang) from an outgoing one (Ausgangsrechnung, Ertrag).
type Direction string

const (
	DirectionIncoming Direction = "incoming"
	DirectionOutgoing Direction = "outgoing"
)

// TaxRate is a VAT rate in basis points, so 1900 means 19,00 %. An integer rate
// keeps the tax computation exact and reproducible; a float rate would reopen
// the rounding problem the Cents type exists to close.
type TaxRate int

const (
	TaxRateNone     TaxRate = 0
	TaxRateReduced  TaxRate = 700  // § 12 Abs. 2 UStG
	TaxRateStandard TaxRate = 1900 // § 12 Abs. 1 UStG
)

// ValidTaxRates lists the rates the system accepts for current transactions.
func ValidTaxRates() []TaxRate {
	return []TaxRate{TaxRateNone, TaxRateReduced, TaxRateStandard}
}

// Label renders the rate for the UI, e.g. "19 %".
func (r TaxRate) Label() string {
	if r%100 == 0 {
		return fmt.Sprintf("%d %%", int(r)/100)
	}
	return fmt.Sprintf("%.2f %%", float64(r)/100)
}

// Tax computes the VAT amount from a net base, rounded commercially. This is the
// only place a tax amount is derived, and it rounds once per rate group — not
// per line item, which would accumulate a difference against the invoice total.
func (r TaxRate) Tax(net Cents) Cents {
	if r == 0 {
		return 0
	}
	return MulRound(net, int64(r), 10000)
}

// NetFromGross splits a gross amount into its net part for this rate.
func (r TaxRate) NetFromGross(gross Cents) Cents {
	if r == 0 {
		return gross
	}
	return MulRound(gross, 10000, 10000+int64(r))
}

// TaxTreatment is the Steuerfall of a transaction. It is an input to the
// booking, not something derivable from the expense category: the same service
// bought domestically, from an EU supplier or under § 13b UStG produces three
// different sets of accounts.
type TaxTreatment string

const (
	// TaxTreatmentDomestic is a normal domestic taxable transaction.
	TaxTreatmentDomestic TaxTreatment = "domestic"
	// TaxTreatmentReverseCharge shifts the tax liability to the recipient
	// (§ 13b UStG). The recipient books output tax and, where deductible, input
	// tax of the same amount at once.
	TaxTreatmentReverseCharge TaxTreatment = "reverse_charge"
	// TaxTreatmentIntraCommunityAcquisition is an innergemeinschaftlicher Erwerb
	// (§ 1a UStG): acquisition tax and matching input tax in the same booking.
	TaxTreatmentIntraCommunityAcquisition TaxTreatment = "intra_community_acquisition"
	// TaxTreatmentIntraCommunitySupply is a tax-exempt innergemeinschaftliche
	// Lieferung (§ 4 Nr. 1 Buchst. b i. V. m. § 6a UStG).
	TaxTreatmentIntraCommunitySupply TaxTreatment = "intra_community_supply"
	// TaxTreatmentExport is a tax-exempt Ausfuhrlieferung into a third country
	// (§ 4 Nr. 1 Buchst. a i. V. m. § 6 UStG).
	TaxTreatmentExport TaxTreatment = "export"
	// TaxTreatmentReverseChargeSupply is an outgoing service where the recipient
	// owes the tax under § 13b UStG.
	TaxTreatmentReverseChargeSupply TaxTreatment = "reverse_charge_supply"
	// TaxTreatmentExempt covers the remaining exemptions of § 4 UStG.
	TaxTreatmentExempt TaxTreatment = "exempt"
	// TaxTreatmentNotTaxable is outside the scope of German VAT (nicht steuerbar),
	// e.g. genuine damages or a transfer between own accounts.
	TaxTreatmentNotTaxable TaxTreatment = "not_taxable"
)

// TaxTreatmentInfo describes a Steuerfall for the UI and for validation.
type TaxTreatmentInfo struct {
	Treatment TaxTreatment `json:"treatment"`
	Label     string       `json:"label"`
	Hint      string       `json:"hint"`
	Direction Direction    `json:"direction"`
	// RequiresRate is false for treatments that never carry a rate.
	RequiresRate bool `json:"requiresRate"`
	// RequiresVatID marks treatments that are only valid with a counterparty
	// VAT identification number on file.
	RequiresVatID bool `json:"requiresVatId"`
}

// TaxTreatments returns the Steuerfälle valid for a direction.
func TaxTreatments(dir Direction) []TaxTreatmentInfo {
	switch dir {
	case DirectionIncoming:
		return []TaxTreatmentInfo{
			{TaxTreatmentDomestic, "Inland, steuerpflichtig", "Normalfall: deutscher Lieferant weist Umsatzsteuer aus.", dir, true, false},
			{TaxTreatmentReverseCharge, "§ 13b UStG (Reverse Charge)", "Leistung eines ausländischen Unternehmers oder Bauleistung: Du schuldest die Steuer und ziehst sie zugleich als Vorsteuer.", dir, true, true},
			{TaxTreatmentIntraCommunityAcquisition, "Innergemeinschaftlicher Erwerb", "Warenkauf aus einem anderen EU-Land ohne ausgewiesene Steuer.", dir, true, true},
			{TaxTreatmentExempt, "Steuerfrei", "Umsatz ist nach § 4 UStG von der Steuer befreit.", dir, false, false},
			{TaxTreatmentNotTaxable, "Nicht steuerbar", "Kein Leistungsaustausch, z. B. echter Schadenersatz.", dir, false, false},
		}
	case DirectionOutgoing:
		return []TaxTreatmentInfo{
			{TaxTreatmentDomestic, "Inland, steuerpflichtig", "Normalfall: Rechnung mit ausgewiesener Umsatzsteuer.", dir, true, false},
			{TaxTreatmentIntraCommunitySupply, "Innergemeinschaftliche Lieferung", "Warenlieferung an ein EU-Unternehmen, steuerfrei nach § 4 Nr. 1b UStG.", dir, false, true},
			{TaxTreatmentReverseChargeSupply, "§ 13b UStG (Steuerschuld beim Empfänger)", "Leistung an ein Unternehmen, das die Steuer schuldet.", dir, false, true},
			{TaxTreatmentExport, "Ausfuhr in ein Drittland", "Lieferung außerhalb der EU, steuerfrei nach § 4 Nr. 1a UStG.", dir, false, false},
			{TaxTreatmentExempt, "Steuerfrei", "Umsatz ist nach § 4 UStG von der Steuer befreit.", dir, false, false},
			{TaxTreatmentNotTaxable, "Nicht steuerbar", "Kein Leistungsaustausch, z. B. echter Schadenersatz.", dir, false, false},
		}
	}
	return nil
}

// TaxLeg is one generated tax line of a booking.
type TaxLeg struct {
	Account string `json:"account"` // SKR04 tax account
	Side    Side   `json:"side"`
	Amount  Cents  `json:"amount"`
	Base    Cents  `json:"base"` // Bemessungsgrundlage
	Key     string `json:"key"`  // Steuerschlüssel, e.g. "VST19", "RC19_VST"
}

// TaxResolver turns a Steuerfall into the tax lines and the tax accounts a
// booking needs. The SKR04 implementation lives in the accounting package.
type TaxResolver interface {
	// Resolve returns the tax legs for a net base amount.
	Resolve(dir Direction, treatment TaxTreatment, rate TaxRate, net Cents) ([]TaxLeg, error)
	// IsTaxAccount reports whether an account may only be written by the tax
	// automation. Booking to a tax account by hand desynchronises the UStVA.
	IsTaxAccount(account string) bool
}
