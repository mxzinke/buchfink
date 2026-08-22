// SPDX-License-Identifier: EUPL-1.2

package domain

// VatFigure is the net base and the tax of one VAT rate.
type VatFigure struct {
	Rate TaxRate `json:"rate"`
	Net  Cents   `json:"net"`
	Tax  Cents   `json:"tax"`
}

// VatSummary aggregates the VAT figures of a period.
//
// It is an orientation figure derived from the journal, not an
// Umsatzsteuer-Voranmeldung: the official return needs its own form, its own
// validation and its own submission path.
type VatSummary struct {
	FiscalYear int    `json:"fiscalYear"`
	PeriodFrom string `json:"periodFrom"`
	PeriodTo   string `json:"periodTo"`

	// Steuerpflichtige Inlandsumsätze je Steuersatz.
	TaxableRevenue []VatFigure `json:"taxableRevenue"`

	// Steuerfreie und nicht im Inland steuerbare Umsätze.
	ExemptRevenue        Cents `json:"exemptRevenue"`
	IntraCommunitySupply Cents `json:"intraCommunitySupply"`
	Export               Cents `json:"export"`
	ReverseChargeSupply  Cents `json:"reverseChargeSupply"`

	// Geschuldete Steuer.
	OutputTax                     Cents `json:"outputTax"`
	ReverseChargeTax              Cents `json:"reverseChargeTax"`
	ReverseChargeBase             Cents `json:"reverseChargeBase"`
	IntraCommunityAcquisitionTax  Cents `json:"intraCommunityAcquisitionTax"`
	IntraCommunityAcquisitionBase Cents `json:"intraCommunityAcquisitionBase"`
	TotalOwedTax                  Cents `json:"totalOwedTax"`

	// Abziehbare Vorsteuer und daraus die Zahllast.
	InputTax Cents `json:"inputTax"`
	Payable  Cents `json:"payable"`
}
