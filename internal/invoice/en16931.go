package invoice

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// EN16931RulesetID and EN16931RulesetVersion identify the rule set a result came
// from. They are stored with every validation so a later re-run is comparable —
// a verdict without the rules that produced it is not evidence.
const (
	EN16931RulesetID      = "buchfink-en16931"
	EN16931RulesetVersion = "2026.1"
)

// ValidationCoverage says how far a check went.
//
// There is deliberately no value meaning "fully EN 16931 validated". The
// reference implementation is a Schematron rule set executed with XSLT 2.0, and
// no Go engine runs it; the rules below are a hand-written subset. Claiming
// completeness would be the one failure mode worse than the gap itself, because
// somebody would rely on it.
type ValidationCoverage string

const (
	// CoveragePartial means: the rules listed in ValidationRules ran, and nothing
	// beyond them.
	CoveragePartial ValidationCoverage = "partial"
)

// ValidationSeverity separates what makes a document unusable from what is worth
// knowing.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
)

// ValidationFinding is one violated rule.
type ValidationFinding struct {
	// Rule is the EN 16931 business rule identifier, e.g. "BR-06". Naming it
	// lets a user look the rule up rather than take Buchfink's word for it.
	Rule     string             `json:"rule"`
	Severity ValidationSeverity `json:"severity"`
	Message  string             `json:"message"`
}

// ValidationResult is the outcome of checking a structured invoice.
type ValidationResult struct {
	Ruleset  string              `json:"ruleset"`
	Version  string              `json:"version"`
	Format   string              `json:"format"`
	Profile  string              `json:"profile"`
	Coverage ValidationCoverage  `json:"coverage"`
	Findings []ValidationFinding `json:"findings,omitempty"`
}

// Valid reports whether the document passed every rule that was checked.
func (r ValidationResult) Valid() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return false
		}
	}
	return true
}

// ErrorCount returns the number of hard violations.
func (r ValidationResult) ErrorCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			n++
		}
	}
	return n
}

// ValidationRules documents exactly which rules ValidateEN16931 implements.
//
// It is part of the result rather than a comment, because "validated" without
// the list of what was checked tells the reader nothing they can act on.
func ValidationRules() []string {
	return []string{
		"BR-01", "BR-02", "BR-03", "BR-04", "BR-05", "BR-06", "BR-07",
		"BR-08", "BR-09", "BR-10", "BR-11", "BR-12", "BR-13", "BR-14", "BR-15", "BR-16",
		"BR-21", "BR-22", "BR-24", "BR-25", "BR-26",
		"BR-45", "BR-46", "BR-47", "BR-48",
		"BR-CO-10", "BR-CO-13", "BR-CO-15", "BR-CO-16", "BR-CO-17",
		"BR-CL-03", "BR-CL-10", "BR-CL-11", "BR-CL-17",
		"BR-AE-05", "BR-AE-08", "BR-AE-10",
		"BR-E-05", "BR-E-08", "BR-E-10",
		"BR-K-05", "BR-K-08", "BR-K-10",
		"BR-Z-05", "BR-Z-08",
		"BR-O-05", "BR-O-08",
	}
}

// ValidateEN16931 checks a Cross Industry Invoice against the rules listed in
// ValidationRules.
//
// The coverage is a subset by construction — see ValidationCoverage. It is
// weighted towards the rules whose violation would produce a wrong booking:
// missing mandatory content, and totals that do not add up.
func ValidateEN16931(doc *CIIInvoice) ValidationResult {
	result := ValidationResult{
		Ruleset:  EN16931RulesetID,
		Version:  EN16931RulesetVersion,
		Format:   string(FormatCII),
		Profile:  doc.Profile(),
		Coverage: CoveragePartial,
	}
	v := &validator{result: &result}

	v.checkHeader(doc)
	v.checkParties(doc)
	v.checkLines(doc)
	v.checkTaxBreakdown(doc)
	v.checkTotals(doc)

	return result
}

type validator struct {
	result *ValidationResult
}

func (v *validator) fail(rule, format string, args ...any) {
	v.result.Findings = append(v.result.Findings, ValidationFinding{
		Rule: rule, Severity: SeverityError, Message: fmt.Sprintf(format, args...),
	})
}

func (v *validator) warn(rule, format string, args ...any) {
	v.result.Findings = append(v.result.Findings, ValidationFinding{
		Rule: rule, Severity: SeverityWarning, Message: fmt.Sprintf(format, args...),
	})
}

func (v *validator) require(rule, value, label string) bool {
	if strings.TrimSpace(value) == "" {
		v.fail(rule, "%s fehlt", label)
		return false
	}
	return true
}

func (v *validator) checkHeader(doc *CIIInvoice) {
	v.require("BR-01", doc.Profile(), "Die Kennung der Spezifikation (BT-24)")
	v.require("BR-02", doc.Document.ID, "Die Rechnungsnummer (BT-1)")
	v.require("BR-03", doc.IssueDate(), "Das Rechnungsdatum (BT-2)")
	v.require("BR-04", doc.Document.TypeCode, "Der Rechnungstyp (BT-3)")

	currency := doc.Transaction.Settlement.Currency
	if v.require("BR-05", currency, "Der Währungscode (BT-5)") && !isCurrencyCode(currency) {
		v.fail("BR-CL-03", "Der Währungscode %q steht nicht in ISO 4217", currency)
	}
}

func (v *validator) checkParties(doc *CIIInvoice) {
	seller := doc.Transaction.Agreement.Seller
	buyer := doc.Transaction.Agreement.Buyer

	v.require("BR-06", seller.Name, "Der Name des Verkäufers (BT-27)")
	v.require("BR-07", buyer.Name, "Der Name des Erwerbers (BT-44)")

	if seller.Address.LineOne == "" && seller.Address.CityName == "" && seller.Address.PostCode == "" {
		v.fail("BR-08", "Die Anschrift des Verkäufers (BG-5) fehlt")
	}
	if v.require("BR-09", seller.Address.CountryID, "Das Länderkennzeichen des Verkäufers (BT-40)") &&
		!isCountryCode(seller.Address.CountryID) {
		v.fail("BR-CL-10", "Das Länderkennzeichen %q des Verkäufers steht nicht in ISO 3166-1", seller.Address.CountryID)
	}

	if buyer.Address.LineOne == "" && buyer.Address.CityName == "" && buyer.Address.PostCode == "" {
		v.fail("BR-10", "Die Anschrift des Erwerbers (BG-8) fehlt")
	}
	if v.require("BR-11", buyer.Address.CountryID, "Das Länderkennzeichen des Erwerbers (BT-55)") &&
		!isCountryCode(buyer.Address.CountryID) {
		v.fail("BR-CL-11", "Das Länderkennzeichen %q des Erwerbers steht nicht in ISO 3166-1", buyer.Address.CountryID)
	}
}

func (v *validator) checkLines(doc *CIIInvoice) {
	if len(doc.Transaction.Lines) == 0 {
		v.fail("BR-16", "Die Rechnung enthält keine Rechnungsposition (BG-25)")
		return
	}
	for i, line := range doc.Transaction.Lines {
		n := i + 1
		v.require("BR-21", line.Document.LineID, fmt.Sprintf("Position %d: die Positionsnummer (BT-126)", n))
		v.require("BR-22", line.Delivery.BilledQuantity.Value, fmt.Sprintf("Position %d: die Menge (BT-129)", n))
		v.require("BR-25", line.Product.Name, fmt.Sprintf("Position %d: die Artikelbezeichnung (BT-153)", n))
		v.require("BR-26", line.Agreement.NetPrice.ChargeAmount, fmt.Sprintf("Position %d: der Nettopreis (BT-146)", n))

		if v.require("BR-24", line.Settlement.Summation.LineTotal, fmt.Sprintf("Position %d: der Nettobetrag (BT-131)", n)) {
			if _, err := domain.ParseCents(line.Settlement.Summation.LineTotal); err != nil {
				v.fail("BR-24", "Position %d: der Nettobetrag %q ist unlesbar",
					n, line.Settlement.Summation.LineTotal)
			}
		}
	}
}

func (v *validator) checkTaxBreakdown(doc *CIIInvoice) {
	taxes := doc.Transaction.Settlement.Taxes
	if len(taxes) == 0 {
		v.fail("BR-45", "Die Rechnung enthält keine Aufschlüsselung der Umsatzsteuer (BG-23)")
		return
	}

	for i, tax := range taxes {
		n := i + 1
		category := strings.ToUpper(strings.TrimSpace(tax.CategoryCode))

		v.require("BR-45", tax.BasisAmount, fmt.Sprintf("Steuergruppe %d: die Bemessungsgrundlage (BT-116)", n))
		v.require("BR-46", tax.CalculatedAmount, fmt.Sprintf("Steuergruppe %d: der Steuerbetrag (BT-117)", n))

		if !v.require("BR-47", category, fmt.Sprintf("Steuergruppe %d: der Steuerkategoriecode (BT-118)", n)) {
			continue
		}
		if !isVATCategoryCode(category) {
			v.fail("BR-CL-17", "Steuergruppe %d: der Kategoriecode %q steht nicht in UNTDID 5305", n, category)
			continue
		}

		// BR-48: außer bei "nicht steuerbar" ist der Steuersatz anzugeben.
		if category != "O" {
			v.require("BR-48", tax.RatePercent, fmt.Sprintf("Steuergruppe %d: der Steuersatz (BT-119)", n))
		}

		v.checkCategoryRules(n, category, tax)
	}
}

// checkCategoryRules implements the per-category rules of EN 16931 that decide
// whether a document says what it claims.
//
// The pattern is the same for every exempt or shifted category: the rate has to
// be zero, and a reason has to be given. A document that carries "reverse
// charge" and a rate of 19 % is internally contradictory, and booking from it
// would produce tax twice.
func (v *validator) checkCategoryRules(n int, category string, tax ciiTradeTax) {
	rateRules := map[string][2]string{
		"AE": {"BR-AE-05", "BR-AE-08"},
		"E":  {"BR-E-05", "BR-E-08"},
		"K":  {"BR-K-05", "BR-K-08"},
		"Z":  {"BR-Z-05", "BR-Z-08"},
		"O":  {"BR-O-05", "BR-O-08"},
	}
	rules, applies := rateRules[category]
	if !applies {
		return
	}

	if rate, err := domain.ParseCents(tax.RatePercent); err == nil && rate != 0 {
		v.fail(rules[0], "Steuergruppe %d: bei Kategorie %q muss der Steuersatz null sein, angegeben sind %s %%",
			n, category, tax.RatePercent)
	}
	if amount, err := domain.ParseCents(tax.CalculatedAmount); err == nil && amount != 0 {
		v.fail(rules[1], "Steuergruppe %d: bei Kategorie %q muss der Steuerbetrag null sein, angegeben sind %s",
			n, category, tax.CalculatedAmount)
	}

	// BR-AE-10, BR-E-10, BR-K-10: ein Befreiungsgrund ist anzugeben. Bei "O"
	// verlangt die Norm ihn nicht.
	switch category {
	case "AE", "E", "K":
		if strings.TrimSpace(tax.ExemptionReason) == "" {
			v.fail("BR-"+category+"-10",
				"Steuergruppe %d: bei Kategorie %q ist der Grund für die Befreiung oder Umkehr der Steuerschuld anzugeben (BT-120)",
				n, category)
		}
	}
}

// checkTotals verifies the arithmetic of the document.
//
// These are the rules whose violation is the most dangerous, because a document
// can be formally complete and still not add up — and the amounts are what gets
// booked.
func (v *validator) checkTotals(doc *CIIInvoice) {
	sum := doc.Transaction.Settlement.Summation

	lineTotal, okLineTotal := v.amount("BR-12", sum.LineTotal, "Die Summe der Positionsbeträge (BT-106)")
	taxBasis, okBasis := v.amount("BR-13", sum.TaxBasisTotal, "Der Gesamtbetrag ohne Umsatzsteuer (BT-109)")
	grand, okGrand := v.amount("BR-14", sum.GrandTotal, "Der Gesamtbetrag mit Umsatzsteuer (BT-112)")
	due, okDue := v.amount("BR-15", sum.DuePayable, "Der fällige Betrag (BT-115)")

	// BR-CO-10: die Summe der Positionsbeträge muss den Positionen entsprechen.
	if okLineTotal && len(doc.Transaction.Lines) > 0 {
		var lines domain.Cents
		complete := true
		for _, line := range doc.Transaction.Lines {
			amount, err := domain.ParseCents(line.Settlement.Summation.LineTotal)
			if err != nil {
				complete = false
				break
			}
			lines += amount
		}
		if complete && lines != lineTotal {
			v.fail("BR-CO-10", "Die Summe der Positionsbeträge ist %s, die Positionen ergeben aber %s",
				lineTotal, lines)
		}
	}

	// BR-CO-13: ohne Nachlässe und Zuschläge ist der Nettogesamtbetrag die Summe
	// der Positionen. Buchfink liest Nachlässe noch nicht, deshalb nur ein
	// Hinweis — eine Abweichung kann hier legitim sein.
	if okLineTotal && okBasis && lineTotal != taxBasis {
		v.warn("BR-CO-13", "Der Gesamtbetrag ohne Umsatzsteuer (%s) weicht von der Summe der Positionen (%s) ab. Das ist zulässig, wenn die Rechnung Nachlässe oder Zuschläge enthält — Buchfink liest diese noch nicht mit",
			taxBasis, lineTotal)
	}

	// BR-CO-15: Bruttobetrag = Nettobetrag + Steuer.
	var taxTotal domain.Cents
	taxKnown := true
	for _, tax := range doc.Transaction.Settlement.Taxes {
		amount, err := domain.ParseCents(tax.CalculatedAmount)
		if err != nil {
			taxKnown = false
			break
		}
		taxTotal += amount
	}
	if okBasis && okGrand && taxKnown && taxBasis+taxTotal != grand {
		v.fail("BR-CO-15", "Der Gesamtbetrag mit Umsatzsteuer ist %s, netto plus Steuer ergeben aber %s",
			grand, taxBasis+taxTotal)
	}

	// BR-CO-16: der fällige Betrag entspricht dem Bruttobetrag, solange nichts
	// vorausgezahlt wurde. Anzahlungen liest Buchfink noch nicht.
	if okGrand && okDue && due != grand {
		v.warn("BR-CO-16", "Der fällige Betrag (%s) weicht vom Gesamtbetrag (%s) ab. Das ist zulässig, wenn bereits gezahlt wurde — Buchfink liest Anzahlungen noch nicht mit",
			due, grand)
	}

	// BR-CO-17: der Steuerbetrag je Gruppe folgt aus Bemessungsgrundlage und Satz.
	for i, tax := range doc.Transaction.Settlement.Taxes {
		base, errBase := domain.ParseCents(tax.BasisAmount)
		amount, errAmount := domain.ParseCents(tax.CalculatedAmount)
		rate, errRate := domain.ParseCents(tax.RatePercent)
		if errBase != nil || errAmount != nil || errRate != nil {
			continue
		}
		expected := domain.MulRound(base, int64(rate), 10000)
		if expected != amount {
			v.fail("BR-CO-17", "Steuergruppe %d: aus %s zu %s %%%% folgt ein Steuerbetrag von %s, angegeben ist %s",
				i+1, base, tax.RatePercent, expected, amount)
		}
	}
}

func (v *validator) amount(rule, raw, label string) (domain.Cents, bool) {
	if !v.require(rule, raw, label) {
		return 0, false
	}
	value, err := domain.ParseCents(raw)
	if err != nil {
		v.fail(rule, "%s ist mit %q unlesbar", label, raw)
		return 0, false
	}
	return value, true
}
