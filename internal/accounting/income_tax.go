package accounting

import "github.com/buchfink/buchfink/internal/domain"

// Die Rechenschritte der Körperschaft- und Gewerbesteuer, die
// Steuerrückstellung und Steuererklärungen gemeinsam nutzen. Rundung und
// Höchstbeträge stehen nur hier; gerechnet wird mit den Werten aus
// TaxParametersFor.

// LossDeduction ist der Verlustabzug mit Mindestbesteuerung: bis zum Sockel
// voll, darüber bis zum Anteil des übersteigenden Betrags (§ 10d Abs. 2 Satz 1
// EStG, § 10a Satz 1 und 2 GewStG). „Bis zu" heißt: der Anteil wird
// abgeschnitten, nicht gerundet.
func LossDeduction(income, carryforward, base domain.Cents, excessPermille int64) domain.Cents {
	if income <= 0 || carryforward <= 0 {
		return 0
	}
	limit := min(income, base)
	if income > base {
		limit += truncPermille(income-base, excessPermille)
	}
	return min(limit, carryforward)
}

// DonationCap ist der Höchstbetrag der abziehbaren Zuwendungen: der höhere aus
// 20 % des Einkommens und 4 ‰ der Summe der Umsätze, Löhne und Gehälter
// (§ 9 Abs. 1 Satz 1 Nr. 2 KStG; für die Gewerbesteuer § 9 Nr. 5 GewStG mit dem
// um § 8 Nr. 9 erhöhten Gewinn). Beide Beträge werden abgeschnitten.
func DonationCap(income, turnoverAndWages domain.Cents, p TaxParameters) (limit, byIncome, byTurnover domain.Cents) {
	if income > 0 {
		byIncome = truncPermille(income, p.DonationIncomePermille)
	}
	if turnoverAndWages > 0 {
		byTurnover = truncPermille(turnoverAndWages, p.DonationTurnoverPermille)
	}
	return max(byIncome, byTurnover), byIncome, byTurnover
}

// ExemptAddBack sind die 5 % der nach § 8b Abs. 1 oder 2 KStG außer Ansatz
// bleibenden Beträge, die als nicht abziehbare Ausgaben gelten (§ 8b Abs. 3
// Satz 1, Abs. 5 Satz 1 KStG).
func ExemptAddBack(exempt domain.Cents, p TaxParameters) domain.Cents {
	return domain.MulRound(exempt, p.ExemptIncomeNonDeductiblePermille, 1000)
}

// CorporateTaxOn ist die Körperschaftsteuer auf das zu versteuernde Einkommen
// (§ 23 Abs. 1 KStG), zugunsten auf volle Euro (§ 31 Abs. 1 Satz 2 KStG).
func CorporateTaxOn(taxable domain.Cents, p TaxParameters) domain.Cents {
	if taxable <= 0 {
		return 0
	}
	tax := truncPermille(taxable, p.CorporateTaxPermille)
	return tax - tax%100
}

// SolidarityOn ist der Solidaritätszuschlag auf die Körperschaftsteuer oder die
// Kapitalertragsteuer (§ 3 Abs. 1 Nr. 1 und 5 SolZG); Bruchteile eines Cents
// bleiben außer Ansatz (§ 4 Satz 1 und 3 SolZG).
func SolidarityOn(tax domain.Cents, p TaxParameters) domain.Cents {
	if tax <= 0 {
		return 0
	}
	return truncPermille(tax, p.SolidarityPermille)
}

// WithholdingTaxPayable ist die Kapitalertragsteuer, die zu einem Zeitpunkt
// abzuführen ist: die einbehaltene, auf volle Euro abgerundet (§ 44 Abs. 1
// Satz 6 EStG). Einbehalten wird sie je Gläubiger auf den Cent.
func WithholdingTaxPayable(withheld domain.Cents) domain.Cents {
	if withheld <= 0 {
		return 0
	}
	return withheld - withheld%100
}

// CreditableWithholding ist die anrechenbare Kapitalertragsteuer, auf volle
// Euro aufgerundet (§ 36 Abs. 3 EStG über § 31 Abs. 1 Satz 1 KStG).
func CreditableWithholding(withheld domain.Cents) domain.Cents {
	if withheld <= 0 || withheld%100 == 0 {
		return withheld
	}
	return withheld - withheld%100 + 100
}

// TradeTaxResult ist die Rechnung vom Gewerbeertrag zur Gewerbesteuer.
type TradeTaxResult struct {
	// Rounded ist der auf volle 100 Euro abgerundete Gewerbeertrag (§ 11
	// Abs. 1 Satz 3 GewStG), Base der Steuermessbetrag, Tax die Gewerbesteuer.
	Rounded domain.Cents
	Base    domain.Cents
	Tax     domain.Cents
}

// TradeTaxOn rechnet die Gewerbesteuer auf den Gewerbeertrag nach
// Verlustabzug: Messzahl (§ 11 Abs. 2 GewStG) und Hebesatz (§ 16 GewStG). Bei
// einem Hebesatz, der kein Vielfaches von 100 ist, wird der halbe Cent
// kaufmännisch gerundet.
func TradeTaxOn(tradeIncome domain.Cents, ratePercent int64, p TaxParameters) TradeTaxResult {
	if tradeIncome <= 0 {
		return TradeTaxResult{}
	}
	var r TradeTaxResult
	r.Rounded = tradeIncome - tradeIncome%10000
	r.Base = domain.MulRound(r.Rounded, p.TradeTaxBasePermille, 1000)
	r.Tax = domain.MulRound(r.Base, ratePercent, 100)
	return r
}
