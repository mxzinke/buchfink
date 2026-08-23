package einvoice

import (
	"fmt"
	"strings"
)

// The VAT category rules of EN 16931 — nine families of ten rules each, plus
// the extras of the intra-community and the not-subject families.
//
// The families have the same shape. What differs is the rate a category admits,
// what the tax amount has to be, whether an exemption reason is required or
// forbidden, and which VAT identifiers the document has to carry. Writing that
// as a table rather than ninety near-identical branches is what keeps the nine
// from drifting apart — and the drift is not hypothetical: the rule set was
// once carrying an invented family name because one category had been written
// out by hand.
//
// Note the naming. The category *codes* come from UNTDID 5305 ("K" for an
// intra-community supply, "L" for IGIC, "M" for IPSI); the rule *families* are
// named differently ("BR-IC", "BR-AF", "BR-AG"). A rule identifier derived from
// the category code would look exactly like a citation somebody can look up,
// and would not exist.

// The UNTDID 5305 category codes EN 16931 admits.
const (
	categoryStandard       = "S"  // Regelsteuersatz
	categoryZeroRated      = "Z"  // Nullsteuersatz
	categoryExempt         = "E"  // steuerbefreit
	categoryReverseCharge  = "AE" // Steuerschuldnerschaft des Leistungsempfängers
	categoryIntraCommunity = "K"  // steuerfreie innergemeinschaftliche Lieferung
	categoryExport         = "G"  // steuerfreie Ausfuhrlieferung
	categoryNotSubject     = "O"  // nicht steuerbar
	categoryIGIC           = "L"  // Kanarische Inseln
	categoryIPSI           = "M"  // Ceuta und Melilla
	categorySplitPayment   = "B"  // italienisches Split Payment
)

// rateConstraint says what a category admits as a rate.
type rateConstraint uint8

const (
	rateAny rateConstraint = iota
	ratePositive
	rateZero
	rateNonNegative
	rateAbsent
)

// sellerIdentification says which of the seller's tax identifiers satisfies a
// category. The distinction is not pedantry: for a cross-border supply the
// national Steuernummer is worthless to the recipient's tax office, so the
// standard does not accept it there.
type sellerIdentification uint8

const (
	sellerVATOrTaxNumber sellerIdentification = iota // BT-31, BT-32 oder BT-63
	sellerVATOnly                                    // BT-31 oder BT-63
	sellerVATForbidden                               // weder BT-31 noch BT-63
)

// buyerIdentification says what the buyer has to carry.
type buyerIdentification uint8

const (
	buyerNothing           buyerIdentification = iota
	buyerVATOrRegistration                     // BT-48 oder BT-47
	buyerVATOnly                               // BT-48
	buyerVATForbidden                          // kein BT-48
)

// taxConstraint says what the tax amount of a breakdown group has to be.
type taxConstraint uint8

const (
	taxFromRate  taxConstraint = iota // Bemessungsgrundlage mal Satz
	taxZero                           // genau null
	taxUnchecked                      // die Norm prüft es nicht
)

type categorySpec struct {
	code   string // UNTDID 5305
	family string // Regelfamilie der Norm

	lineRate rateConstraint // -05
	acRate   rateConstraint // -06, -07

	seller sellerIdentification // -02, -03, -04
	buyer  buyerIdentification

	// baseExact says whether the -08 comparison is exact. The standard is
	// exact for S, L, M and O and allows one currency unit elsewhere.
	baseExact bool
	tax       taxConstraint // -09

	reasonRequired  bool // -10
	reasonForbidden bool // -10
}

var categorySpecs = []categorySpec{
	{code: categoryStandard, family: "S",
		lineRate: ratePositive, acRate: ratePositive,
		seller: sellerVATOrTaxNumber, baseExact: true, tax: taxFromRate, reasonForbidden: true},
	{code: categoryZeroRated, family: "Z",
		lineRate: rateZero, acRate: rateZero,
		seller: sellerVATOrTaxNumber, tax: taxZero, reasonForbidden: true},
	{code: categoryExempt, family: "E",
		lineRate: rateZero, acRate: rateZero,
		seller: sellerVATOrTaxNumber, tax: taxZero, reasonRequired: true},
	{code: categoryReverseCharge, family: "AE",
		lineRate: rateZero, acRate: rateZero,
		seller: sellerVATOrTaxNumber, buyer: buyerVATOrRegistration, tax: taxZero, reasonRequired: true},
	{code: categoryIntraCommunity, family: "IC",
		lineRate: rateZero, acRate: rateZero,
		seller: sellerVATOnly, buyer: buyerVATOnly, tax: taxZero, reasonRequired: true},
	{code: categoryExport, family: "G",
		lineRate: rateZero, acRate: rateZero,
		seller: sellerVATOnly, tax: taxZero, reasonRequired: true},
	{code: categoryNotSubject, family: "O",
		lineRate: rateAbsent, acRate: rateAbsent,
		seller: sellerVATForbidden, buyer: buyerVATForbidden,
		baseExact: true, tax: taxZero, reasonRequired: true},
	// Bei IGIC und IPSI weichen die beiden Syntaxbindungen der Norm
	// voneinander ab: CII verlangt für den Positionssatz "größer null" und
	// führt die Steuerprüfung als `true()`, UBL lässt "null oder größer" zu und
	// prüft die Steuer wie überall. Buchfink folgt UBL. Ein Nullsatz ist auf
	// den Kanaren keine Erfindung — das IGIC kennt den tipo cero —, und eine
	// Steuerprüfung wegzulassen, die die andere Bindung durchführt, hieße die
	// Rechnung nach ihrer Schreibweise unterschiedlich streng zu behandeln.
	{code: categoryIGIC, family: "AF",
		lineRate: rateNonNegative, acRate: rateNonNegative,
		seller: sellerVATOrTaxNumber, baseExact: true, tax: taxFromRate, reasonForbidden: true},
	{code: categoryIPSI, family: "AG",
		lineRate: rateNonNegative, acRate: rateNonNegative,
		seller: sellerVATOrTaxNumber, baseExact: true, tax: taxFromRate, reasonForbidden: true},
}

func specForCategory(code string) (categorySpec, bool) {
	code = normaliseCategory(code)
	for _, s := range categorySpecs {
		if s.code == code {
			return s, true
		}
	}
	return categorySpec{}, false
}

// checkCategories runs the per-category rules.
//
// The structure follows the standard's own: -01 is triggered by lines,
// allowances and charges; -02 by lines, -03 by allowances, -04 by charges; -05
// by lines, -06 by allowances, -07 by charges; and -08 to -10 by the breakdown
// groups themselves.
func (v *validator) checkCategories() {
	inv := v.inv
	inBreakdown := map[string]int{}
	for _, g := range inv.VATBreakdowns() {
		inBreakdown[normaliseCategory(g.CategoryCode)]++
	}

	for i, line := range inv.Lines {
		spec, known := specForCategory(line.VAT.CategoryCode)
		if !known {
			continue
		}
		where := linePos(i)
		v.requireBreakdownGroup(spec, inBreakdown, where)
		v.checkIdentifiers("BR-"+spec.family+"-02", spec)
		v.checkRate("BR-"+spec.family+"-05", spec, spec.lineRate, where, line.VAT.Rate)
	}

	for i, a := range inv.Allowances {
		spec, known := specForCategory(a.VATCategory)
		if !known {
			continue
		}
		where := fmt.Sprintf("Nachlass %d auf Dokumentebene", i+1)
		v.requireBreakdownGroup(spec, inBreakdown, where)
		v.checkIdentifiers("BR-"+spec.family+"-03", spec)
		v.checkRate("BR-"+spec.family+"-06", spec, spec.acRate, where, a.VATRate)
	}
	for i, a := range inv.Charges {
		spec, known := specForCategory(a.VATCategory)
		if !known {
			continue
		}
		where := fmt.Sprintf("Zuschlag %d auf Dokumentebene", i+1)
		v.requireBreakdownGroup(spec, inBreakdown, where)
		v.checkIdentifiers("BR-"+spec.family+"-04", spec)
		v.checkRate("BR-"+spec.family+"-07", spec, spec.acRate, where, a.VATRate)
	}

	groups := inv.VATBreakdowns()
	for i, group := range groups {
		spec, known := specForCategory(group.CategoryCode)
		if !known {
			continue
		}
		where := breakdownPos(i)

		v.checkBreakdownBase("BR-"+spec.family+"-08", spec, where, group, len(groups))
		v.checkBreakdownTax("BR-"+spec.family+"-09", spec, where, group)

		if spec.reasonRequired && !group.HasExemptionReason() {
			v.failAt("BR-"+spec.family+"-10", where,
				"Bei der Steuerkategorie %q ist der Grund für Befreiung oder Umkehr der Steuerschuld anzugeben (BT-120 oder BT-121)",
				spec.code)
		}
		if spec.reasonForbidden && group.HasExemptionReason() {
			v.failAt("BR-"+spec.family+"-10", where,
				"Bei der Steuerkategorie %q darf kein Befreiungsgrund angegeben werden — der Umsatz ist steuerpflichtig",
				spec.code)
		}
	}

	v.checkIntraCommunityExtras(groups)
	v.checkNotSubjectExtras(groups)
	v.checkSplitPayment()
}

// requireBreakdownGroup implements the -01 rules: whatever category a line, an
// allowance or a charge uses, the breakdown has to carry exactly one group for
// it. Without the group the turnover is missing from the Voranmeldung; with two
// the amounts cannot be attributed.
func (v *validator) requireBreakdownGroup(spec categorySpec, inBreakdown map[string]int, where string) {
	switch count := inBreakdown[spec.code]; {
	case count == 0:
		v.failAt("BR-"+spec.family+"-01", where,
			"Die Steuerkategorie %q wird verwendet, die Aufschlüsselung enthält dafür aber keine Gruppe",
			spec.code)
	case count > 1 && spec.code != categoryStandard && spec.code != categoryIGIC && spec.code != categoryIPSI:
		// Nur die Kategorien mit einem Satz größer null dürfen mehrfach
		// vorkommen — dort trennt sie der Steuersatz. Bei den übrigen ist der
		// Satz immer null, zwei Gruppen wären also nicht unterscheidbar.
		v.failAt("BR-"+spec.family+"-01", where,
			"Die Aufschlüsselung enthält %d Gruppen der Steuerkategorie %q; zulässig ist genau eine",
			count, spec.code)
	}
}

// checkIdentifiers implements the -02, -03 and -04 rules.
func (v *validator) checkIdentifiers(rule string, spec categorySpec) {
	inv := v.inv
	sellerVAT := inv.Seller.VATIdentifier != ""
	if inv.TaxRepresentative != nil && inv.TaxRepresentative.VATIdentifier != "" {
		sellerVAT = true
	}
	sellerTaxNumber := inv.Seller.TaxRegistration != ""

	switch spec.seller {
	case sellerVATOrTaxNumber:
		if !sellerVAT && !sellerTaxNumber {
			v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. oder die Steuernummer des Verkäufers (BT-31, BT-32 oder BT-63)",
				spec.code)
		}
	case sellerVATOnly:
		if !sellerVAT {
			v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. des Verkäufers (BT-31 oder BT-63) — die nationale Steuernummer genügt nicht",
				spec.code)
		}
	case sellerVATForbidden:
		if sellerVAT {
			v.fail(rule, "Bei der Steuerkategorie %q darf die Rechnung keine USt-IdNr. des Verkäufers tragen — der Umsatz ist nicht steuerbar",
				spec.code)
		}
	}

	switch spec.buyer {
	case buyerVATOrRegistration:
		if inv.Buyer.VATIdentifier == "" && !inv.Buyer.LegalRegistration.Present() {
			v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. (BT-48) oder die Registernummer (BT-47) des Erwerbers",
				spec.code)
		}
	case buyerVATOnly:
		if inv.Buyer.VATIdentifier == "" {
			v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. des Erwerbers (BT-48)", spec.code)
		}
	case buyerVATForbidden:
		if inv.Buyer.VATIdentifier != "" {
			v.fail(rule, "Bei der Steuerkategorie %q darf die Rechnung keine USt-IdNr. des Erwerbers tragen — der Umsatz ist nicht steuerbar",
				spec.code)
		}
	}
}

// checkRate implements the -05, -06 and -07 rules.
func (v *validator) checkRate(rule string, spec categorySpec, want rateConstraint, where string, rate Amount) {
	if want == rateAbsent {
		if rate.Present() && !rate.IsZero() {
			v.failAt(rule, where, "Bei der Steuerkategorie %q darf kein Steuersatz angegeben werden, angegeben sind %s %%",
				spec.code, rate)
		}
		return
	}
	if !rate.Present() {
		return
	}
	switch sign := rate.Sign(); {
	case want == ratePositive && sign <= 0:
		v.failAt(rule, where, "Bei der Steuerkategorie %q muss der Steuersatz größer null sein, angegeben sind %s %%",
			spec.code, rate)
	case want == rateZero && sign != 0:
		v.failAt(rule, where, "Bei der Steuerkategorie %q muss der Steuersatz null sein, angegeben sind %s %%",
			spec.code, rate)
	case want == rateNonNegative && sign < 0:
		v.failAt(rule, where, "Bei der Steuerkategorie %q darf der Steuersatz nicht negativ sein, angegeben sind %s %%",
			spec.code, rate)
	}
}

// checkBreakdownBase implements the -08 rules: the taxable amount of a group
// equals the lines carrying that category, plus the charges, minus the
// allowances.
//
// Where a category admits more than one rate, the attribution runs by rate as
// well. Where it does not — every category whose rate is fixed at zero — it
// runs by category alone, which also survives the documents that leave the rate
// off the line.
func (v *validator) checkBreakdownBase(rule string, spec categorySpec, where string, group VATBreakdown, groupCount int) {
	declared, err := group.TaxableAmount.Cents()
	if err != nil {
		return
	}
	byRate := spec.lineRate == ratePositive || spec.lineRate == rateNonNegative

	var sum Cents
	var matched int
	for _, line := range v.inv.Lines {
		if normaliseCategory(line.VAT.CategoryCode) != spec.code {
			continue
		}
		if byRate && !line.VAT.Rate.Equal(group.Rate) {
			continue
		}
		amount, err := line.NetAmount.Cents()
		if err != nil {
			return
		}
		sum += amount
		matched++
	}
	for _, a := range v.inv.Charges {
		if normaliseCategory(a.VATCategory) != spec.code {
			continue
		}
		if byRate && !a.VATRate.Equal(group.Rate) {
			continue
		}
		amount, err := a.Amount.Cents()
		if err != nil {
			return
		}
		sum += amount
		matched++
	}
	for _, a := range v.inv.Allowances {
		if normaliseCategory(a.VATCategory) != spec.code {
			continue
		}
		if byRate && !a.VATRate.Equal(group.Rate) {
			continue
		}
		amount, err := a.Amount.Cents()
		if err != nil {
			return
		}
		sum -= amount
		matched++
	}
	if matched == 0 {
		return
	}

	tolerance := Cents(0)
	if !spec.baseExact {
		tolerance = vatRoundingTolerance
	}
	if (sum - declared).Abs() > tolerance {
		v.failAt(rule, where,
			"Die Bemessungsgrundlage ist %s, Positionen, Zu- und Abschläge der Steuerkategorie %q ergeben aber %s",
			declared, spec.code, sum)
	}
}

// checkBreakdownTax implements the -09 rules.
func (v *validator) checkBreakdownTax(rule string, spec categorySpec, where string, group VATBreakdown) {
	switch spec.tax {
	case taxZero:
		amount, err := group.TaxAmount.Cents()
		if err != nil {
			return
		}
		if amount != 0 {
			v.failAt(rule, where, "Bei der Steuerkategorie %q muss der Steuerbetrag null sein, angegeben sind %s",
				spec.code, amount)
		}
	case taxFromRate:
		v.checkTaxFollowsFromRate(rule, where, group)
	case taxUnchecked:
		// Keine Kategorie nutzt das derzeit. Der Fall bleibt, weil die Norm
		// selbst ihn kennt: die CII-Bindung führt BR-AF-09 und BR-AG-09 als
		// `true()`, und käme das je zurück, wäre es hier abgebildet statt
		// stillschweigend geprüft.
	}
}

// checkIntraCommunityExtras covers BR-IC-11 and BR-IC-12.
func (v *validator) checkIntraCommunityExtras(groups []VATBreakdown) {
	used := false
	for _, g := range groups {
		if normaliseCategory(g.CategoryCode) == categoryIntraCommunity {
			used = true
		}
	}
	if !used {
		return
	}

	delivered := v.inv.Delivery != nil && v.inv.Delivery.Date.Present()
	if !delivered && !v.inv.Period.Present() {
		v.fail("BR-IC-11", "Bei einer innergemeinschaftlichen Lieferung ist das Lieferdatum (BT-72) oder der Rechnungszeitraum (BG-14) anzugeben")
	}
	if v.inv.Delivery == nil || v.inv.Delivery.Address == nil ||
		strings.TrimSpace(v.inv.Delivery.Address.CountryCode) == "" {
		v.fail("BR-IC-12", "Bei einer innergemeinschaftlichen Lieferung ist das Bestimmungsland anzugeben (BT-80)")
	}
}

// checkNotSubjectExtras covers BR-O-11 to BR-O-14: a document that is not
// subject to VAT must not mix in anything that is.
func (v *validator) checkNotSubjectExtras(groups []VATBreakdown) {
	used := false
	for _, g := range groups {
		if normaliseCategory(g.CategoryCode) == categoryNotSubject {
			used = true
		}
	}
	if !used {
		return
	}

	for i, g := range groups {
		if normaliseCategory(g.CategoryCode) != categoryNotSubject {
			v.failAt("BR-O-11", breakdownPos(i),
				"Eine Rechnung mit der Steuerkategorie \"nicht steuerbar\" darf keine weitere Steuergruppe enthalten, hier steht %q",
				g.CategoryCode)
		}
	}
	for i, line := range v.inv.Lines {
		if normaliseCategory(line.VAT.CategoryCode) != categoryNotSubject {
			v.failAt("BR-O-12", linePos(i),
				"Die Position führt die Steuerkategorie %q, obwohl die Rechnung als nicht steuerbar ausgewiesen ist",
				line.VAT.CategoryCode)
		}
	}
	for i, a := range v.inv.Allowances {
		if normaliseCategory(a.VATCategory) != categoryNotSubject {
			v.failAt("BR-O-13", fmt.Sprintf("Nachlass %d auf Dokumentebene", i+1),
				"Der Nachlass führt die Steuerkategorie %q, obwohl die Rechnung als nicht steuerbar ausgewiesen ist",
				a.VATCategory)
		}
	}
	for i, a := range v.inv.Charges {
		if normaliseCategory(a.VATCategory) != categoryNotSubject {
			v.failAt("BR-O-14", fmt.Sprintf("Zuschlag %d auf Dokumentebene", i+1),
				"Der Zuschlag führt die Steuerkategorie %q, obwohl die Rechnung als nicht steuerbar ausgewiesen ist",
				a.VATCategory)
		}
	}
}

// checkSplitPayment covers BR-B-01 and BR-B-02.
//
// Split payment is an Italian arrangement: the public-sector buyer pays the VAT
// straight to the tax office. It has no counterpart in German VAT law, which is
// exactly why the two rules matter — a document using it that is not Italian is
// using it wrongly, and Buchfink would have no way to book it.
func (v *validator) checkSplitPayment() {
	used := false
	standard := false
	for _, code := range v.splitPaymentCategories() {
		switch code {
		case categorySplitPayment:
			used = true
		case categoryStandard:
			standard = true
		}
	}
	if !used {
		return
	}

	for _, country := range v.countryCodes() {
		if !strings.EqualFold(country, "IT") {
			v.fail("BR-B-01", "Split Payment ist eine italienische Regelung, die Rechnung nennt aber das Land %q", country)
			break
		}
	}
	if standard {
		v.fail("BR-B-02", "Split Payment und Regelsteuersatz stehen auf derselben Rechnung; die Norm lässt nur eines von beiden zu")
	}
}

// splitPaymentCategories lists every VAT category the document assigns
// anywhere — breakdown, lines, allowances and charges.
func (v *validator) splitPaymentCategories() []string {
	var out []string
	for _, g := range v.inv.VATBreakdown {
		out = append(out, normaliseCategory(g.CategoryCode))
	}
	for _, l := range v.inv.Lines {
		out = append(out, normaliseCategory(l.VAT.CategoryCode))
	}
	for _, a := range v.inv.Allowances {
		out = append(out, normaliseCategory(a.VATCategory))
	}
	for _, a := range v.inv.Charges {
		out = append(out, normaliseCategory(a.VATCategory))
	}
	return out
}

// countryCodes lists every country code stated anywhere in the document.
func (v *validator) countryCodes() []string {
	var out []string
	add := func(a *Address) {
		if a != nil && strings.TrimSpace(a.CountryCode) != "" {
			out = append(out, strings.TrimSpace(a.CountryCode))
		}
	}
	add(v.inv.Seller.Address)
	add(v.inv.Buyer.Address)
	if v.inv.Payee != nil {
		add(v.inv.Payee.Address)
	}
	if v.inv.TaxRepresentative != nil {
		add(v.inv.TaxRepresentative.Address)
	}
	if v.inv.Delivery != nil {
		add(v.inv.Delivery.Address)
	}
	for _, l := range v.inv.Lines {
		if c := strings.TrimSpace(l.Item.OriginCountryCode); c != "" {
			out = append(out, c)
		}
	}
	return out
}
