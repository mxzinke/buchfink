package accounting

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// TaxParameters are the statutory values a booking depends on that the
// legislator changes over time.
//
// They are keyed by the period they apply to rather than being editable master
// data. The distinction matters: these are not choices the user makes. Making
// them editable would invite a wrong entry to produce a quietly wrong booking,
// while a fixed constant in the code would silently misbook a reworked prior
// year. A dated table does neither, and PostingRuleVersion on every entry says
// which set produced it.
type TaxParameters struct {
	// ValidFrom is the first day this set applies to, as ISO date.
	ValidFrom string

	// EntertainmentDeductiblePermille is the deductible share of entertainment
	// expenses, in permille (700 = 70 %). § 4 Abs. 5 Satz 1 Nr. 2 EStG. The input
	// tax is unaffected — § 15 Abs. 1a Satz 2 UStG takes entertainment out of the
	// deduction ban.
	EntertainmentDeductiblePermille int64

	// SmallAmountInvoiceLimit is the gross ceiling below which a Kleinbetrags-
	// rechnung is enough (§ 33 UStDV). Relevant for the e-invoice obligation: a
	// small-amount invoice may always be a sonstige Rechnung.
	SmallAmountInvoiceLimit domain.Cents

	// CorporateTaxPermille ist der Körperschaftsteuersatz in Promille
	// (150 = 15 %, § 23 Abs. 1 KStG).
	CorporateTaxPermille int64
	// SolidarityPermille ist der Solidaritätszuschlag auf die
	// Körperschaftsteuer (55 = 5,5 %, § 4 SolZG).
	SolidarityPermille int64
	// TradeTaxBasePermille ist die Steuermesszahl der Gewerbesteuer
	// (35 = 3,5 %, § 11 Abs. 2 GewStG). Der Hebesatz kommt von der Gemeinde und
	// steht deshalb in den Einstellungen, nicht hier.
	TradeTaxBasePermille int64
	// WithholdingTaxPermille ist der Kapitalertragsteuersatz auf eine
	// Ausschüttung (250 = 25 %, § 43a Abs. 1 Satz 1 Nr. 1 EStG).
	WithholdingTaxPermille int64
}

// taxParameterSets are ordered by ValidFrom, oldest first.
//
// Historical values are kept, not overwritten: a booking dated into a closed
// period has to be reproducible with the values that applied then.
var taxParameterSets = []TaxParameters{
	{
		ValidFrom:                       "2007-01-01",
		EntertainmentDeductiblePermille: 700,
		// § 33 UStDV: 150 € until 2016, 250 € from 2017.
		SmallAmountInvoiceLimit: 15000,
		// Bis 2007 betrug der Körperschaftsteuersatz 25 % (§ 23 Abs. 1 KStG
		// a. F.); die Unternehmensteuerreform 2008 hat ihn auf 15 % gesenkt und
		// zugleich die Gewerbesteuer-Messzahl von 5 % auf 3,5 % vereinheitlicht.
		CorporateTaxPermille:   250,
		SolidarityPermille:     55,
		TradeTaxBasePermille:   50,
		WithholdingTaxPermille: 200,
	},
	{
		ValidFrom:                       "2008-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         15000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		// Die Abgeltungsteuer von 25 % gilt seit dem 1. Januar 2009
		// (§ 43a Abs. 1 Satz 1 Nr. 1 EStG i. d. F. des UntStRefG 2008).
		WithholdingTaxPermille: 200,
	},
	{
		ValidFrom:                       "2009-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         15000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		WithholdingTaxPermille:          250,
	},
	{
		ValidFrom:                       "2017-01-01",
		EntertainmentDeductiblePermille: 700,
		SmallAmountInvoiceLimit:         25000,
		CorporateTaxPermille:            150,
		SolidarityPermille:              55,
		TradeTaxBasePermille:            35,
		WithholdingTaxPermille:          250,
	},
}

// TaxParametersFor returns the values that applied on a date.
func TaxParametersFor(date string) (TaxParameters, error) {
	if date == "" {
		return TaxParameters{}, fmt.Errorf("ohne Datum lassen sich die steuerlichen Werte nicht bestimmen")
	}
	idx := sort.Search(len(taxParameterSets), func(i int) bool {
		return taxParameterSets[i].ValidFrom > date
	})
	if idx == 0 {
		return TaxParameters{}, fmt.Errorf("für den %s sind keine steuerlichen Werte hinterlegt", date)
	}
	return taxParameterSets[idx-1], nil
}

// EInvoiceTransition says whether a sonstige Rechnung — paper or a plain PDF —
// was still permitted for a transaction on a given date.
//
// § 27 Abs. 38 UStG knows three cases, and the middle one is the reason this
// returns a status rather than a boolean: from 2027 the transitional rule only
// covers issuers whose total turnover in the previous year did not exceed
// 800.000 €. That figure belongs to the supplier, and Buchfink cannot know it.
// Claiming certainty either way would be worse than naming the condition.
type EInvoiceTransition string

const (
	// EInvoiceTransitionAllowed: a sonstige Rechnung is still permitted
	// (§ 27 Abs. 38 Nr. 1 UStG, transactions before 01.01.2027).
	EInvoiceTransitionAllowed EInvoiceTransition = "allowed"
	// EInvoiceTransitionConditional: permitted only if the issuer's total
	// turnover in the previous year was at most 800.000 €
	// (§ 27 Abs. 38 Nr. 2 UStG, transactions in 2027).
	EInvoiceTransitionConditional EInvoiceTransition = "conditional"
	// EInvoiceTransitionExpired: no transitional rule applies any more.
	EInvoiceTransitionExpired EInvoiceTransition = "expired"
)

// EInvoiceTransitionFor classifies a document date against § 27 Abs. 38 UStG.
func EInvoiceTransitionFor(documentDate string) EInvoiceTransition {
	switch {
	case documentDate < "2027-01-01":
		return EInvoiceTransitionAllowed
	case documentDate < "2028-01-01":
		return EInvoiceTransitionConditional
	default:
		return EInvoiceTransitionExpired
	}
}
