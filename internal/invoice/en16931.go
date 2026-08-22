// ACHTUNG: Dieser Prüfer ist abgelöst.
//
// Die EN-16931-Prüfung liegt in `internal/einvoice`. Sie läuft auf einem
// semantischen Modell statt auf CII-Structs, deckt alle 223 Geschäftsregeln ab
// statt 170, liest neben CII auch UBL, und XRechnung und ZUGFeRD sitzen als
// Schichten darüber.
//
// Was hier steht, hängt nur noch am Buchungspfad (`internal/service`), der
// weiterhin die CIIInvoice-Struktur verwendet. Das Umhängen ist der zweite
// Schritt und für sich zu machen — bis dahin gilt: **neue Regeln kommen ins
// Modul, nicht hierher.** Zwei Prüfer im Baum sind genau die Stelle, an der
// jemand den falschen bearbeitet.

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
	EN16931RulesetVersion = "2026.2"
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
	v := &validator{result: &result, seen: map[string]bool{}}

	v.checkHeader(doc)
	v.checkParties(doc)
	v.checkLines(doc)
	v.checkAllowancesCharges(doc)
	v.checkTaxBreakdown(doc)
	v.checkTotals(doc)

	return result
}

type validator struct {
	result *ValidationResult
	// seen suppresses repetitions of the same finding. Several rules are
	// triggered per line but state one document-wide fact ("the seller VAT
	// identifier is missing"); reporting it once per line would bury the other
	// findings without adding anything.
	seen map[string]bool
}

func (v *validator) add(rule string, severity ValidationSeverity, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	key := rule + "\x00" + message
	if v.seen[key] {
		return
	}
	v.seen[key] = true
	v.result.Findings = append(v.result.Findings, ValidationFinding{
		Rule: rule, Severity: severity, Message: message,
	})
}

func (v *validator) fail(rule, format string, args ...any) {
	v.add(rule, SeverityError, format, args...)
}

func (v *validator) warn(rule, format string, args ...any) {
	v.add(rule, SeverityWarning, format, args...)
}

func (v *validator) require(rule, value, label string) bool {
	if strings.TrimSpace(value) == "" {
		v.fail(rule, "%s fehlt", label)
		return false
	}
	return true
}

// decimals implements the BR-DEC family: every monetary amount of an invoice is
// limited to two decimal places.
//
// It is worth checking explicitly rather than letting the amount parser stumble
// over it. A third decimal place means the document states an amount that no
// account can hold, and every sum rule downstream would silently skip the value
// instead of reporting it.
func (v *validator) decimals(rule, raw, label string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return
	}
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return
	}
	if n := len(s) - dot - 1; n > 2 {
		v.fail(rule, "%s hat %d Nachkommastellen, erlaubt sind höchstens zwei", label, n)
	}
}

func (v *validator) checkHeader(doc *CIIInvoice) {
	v.require("BR-01", doc.Profile(), "Die Kennung der Spezifikation (BT-24)")
	v.require("BR-02", doc.Document.ID, "Die Rechnungsnummer (BT-1)")
	v.require("BR-03", doc.IssueDate(), "Das Rechnungsdatum (BT-2)")
	v.require("BR-04", doc.Document.TypeCode, "Der Rechnungstyp (BT-3)")

	currency := doc.Transaction.Settlement.Currency
	if v.require("BR-05", currency, "Der Währungscode (BT-5)") && !isCurrencyCode(currency) {
		v.fail("BR-CL-04", "Der Währungscode %q steht nicht in ISO 4217", currency)
	}

	for _, total := range doc.Transaction.Settlement.Summation.TaxTotals {
		if code := strings.TrimSpace(total.CurrencyID); code != "" && !isCurrencyCode(code) {
			v.fail("BR-CL-03", "Die Währung %q am Gesamtbetrag der Umsatzsteuer steht nicht in ISO 4217", code)
		}
	}
	if code := strings.TrimSpace(doc.Transaction.Settlement.TaxCurrency); code != "" && !isCurrencyCode(code) {
		v.fail("BR-CL-05", "Die Abrechnungswährung %q (BT-6) steht nicht in ISO 4217", code)
	}

	// BR-CO-19: ein Abrechnungszeitraum, der da ist, muss auch einen Anfang oder
	// ein Ende haben. Deshalb ist Period ein Zeiger — sonst wäre "kein Zeitraum"
	// von "ein leerer Zeitraum" nicht zu unterscheiden.
	if p := doc.Transaction.Delivery.Period; p != nil &&
		strings.TrimSpace(p.Start.Value) == "" && strings.TrimSpace(p.End.Value) == "" {
		v.fail("BR-CO-19", "Der angegebene Abrechnungszeitraum (BG-14) nennt weder Beginn (BT-73) noch Ende (BT-74)")
	}
}

func (v *validator) checkParties(doc *CIIInvoice) {
	seller := doc.Transaction.Agreement.Seller
	buyer := doc.Transaction.Agreement.Buyer

	v.require("BR-06", seller.Name, "Der Name des Verkäufers (BT-27)")
	v.require("BR-07", buyer.Name, "Der Name des Erwerbers (BT-44)")

	// BR-08 und BR-10 fragen nach dem Vorhandensein der Adressgruppe, nicht nach
	// ihrem Inhalt. Eine Anschrift, die nur aus dem Länderkennzeichen besteht,
	// ist gültig — dafür ist BR-09 zuständig, und das ist eine eigene Regel.
	if seller.Address == nil {
		v.fail("BR-08", "Die Anschrift des Verkäufers (BG-5) fehlt")
	}
	if v.require("BR-09", seller.CountryCode(), "Das Länderkennzeichen des Verkäufers (BT-40)") &&
		!isCountryCode(seller.CountryCode()) {
		v.fail("BR-CL-14", "Das Länderkennzeichen %q des Verkäufers steht nicht in ISO 3166-1", seller.CountryCode())
	}

	if buyer.Address == nil {
		v.fail("BR-10", "Die Anschrift des Erwerbers (BG-8) fehlt")
	}
	if v.require("BR-11", buyer.CountryCode(), "Das Länderkennzeichen des Erwerbers (BT-55)") &&
		!isCountryCode(buyer.CountryCode()) {
		v.fail("BR-CL-14", "Das Länderkennzeichen %q des Erwerbers steht nicht in ISO 3166-1", buyer.CountryCode())
	}

	// BR-CO-09: die USt-IdNr. beginnt mit dem Länderkennzeichen des ausstellenden
	// Staates. Das ist die einzige Stelle, an der sich eine vertauschte oder
	// abgeschnittene Nummer noch vor der Buchung zeigt.
	v.checkVatPrefix(seller.VatID(), "des Verkäufers (BT-31)")
	v.checkVatPrefix(buyer.VatID(), "des Erwerbers (BT-48)")

	// BR-CO-26: ohne mindestens eine Kennung ist der Lieferant maschinell nicht
	// zuzuordnen — genau das, was Buchfink für den Kontaktabgleich braucht.
	if strings.TrimSpace(seller.ID) == "" && strings.TrimSpace(seller.GlobalID) == "" &&
		strings.TrimSpace(seller.LegalOrganization.ID) == "" && seller.VatID() == "" {
		v.fail("BR-CO-26", "Der Verkäufer trägt keine Kennung: weder eine Nummer (BT-29), noch eine Registernummer (BT-30), noch eine USt-IdNr. (BT-31)")
	}
}

// checkVatPrefix implements BR-CO-09.
//
// The norm's list is ISO 3166-1 alpha-2 plus two entries that are not in it:
// Greece issues its VAT identifiers with "EL", and "1A" is reserved for Kosovo.
func (v *validator) checkVatPrefix(id, whose string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	if len(id) < 2 {
		v.fail("BR-CO-09", "Die USt-IdNr. %s ist mit %q zu kurz für ein Länderkennzeichen", whose, id)
		return
	}
	prefix := strings.ToUpper(id[:2])
	if prefix == "EL" || prefix == "1A" || isCountryCode(prefix) {
		return
	}
	v.fail("BR-CO-09", "Die USt-IdNr. %s beginnt mit %q, was kein Länderkennzeichen nach ISO 3166-1 ist", whose, prefix)
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
		v.require("BR-CO-04", line.Settlement.Tax.CategoryCode, fmt.Sprintf("Position %d: die Steuerkategorie (BT-151)", n))

		if v.require("BR-24", line.Settlement.Summation.LineTotal, fmt.Sprintf("Position %d: der Nettobetrag (BT-131)", n)) {
			if _, err := domain.ParseCents(line.Settlement.Summation.LineTotal); err != nil {
				v.fail("BR-24", "Position %d: der Nettobetrag %q ist unlesbar",
					n, line.Settlement.Summation.LineTotal)
			}
		}
		v.decimals("BR-DEC-23", line.Settlement.Summation.LineTotal,
			fmt.Sprintf("Position %d: der Nettobetrag (BT-131)", n))
	}
}

// checkAllowancesCharges implements the rules on Nachlässe and Zuschläge —
// BR-31 to BR-44 and their decimal limits.
//
// Reading them is not optional decoration: without them the taxable amount of a
// breakdown group cannot be reconstructed, and every sum rule would have to be
// downgraded to a guess.
func (v *validator) checkAllowancesCharges(doc *CIIInvoice) {
	for i, a := range doc.DocumentAllowances() {
		where := fmt.Sprintf("Nachlass %d auf Dokumentebene", i+1)
		v.require("BR-31", a.ActualAmount, where+": der Betrag (BT-92)")
		v.require("BR-32", a.CategoryTax.CategoryCode, where+": die Steuerkategorie (BT-95)")
		if !a.HasReason() {
			v.fail("BR-33", "%s: es fehlt der Grund (BT-97) oder der Grundschlüssel (BT-98)", where)
		}
		v.decimals("BR-DEC-01", a.ActualAmount, where+": der Betrag (BT-92)")
		v.decimals("BR-DEC-02", a.BasisAmount, where+": der Grundbetrag (BT-93)")
	}

	for i, a := range doc.DocumentCharges() {
		where := fmt.Sprintf("Zuschlag %d auf Dokumentebene", i+1)
		v.require("BR-36", a.ActualAmount, where+": der Betrag (BT-99)")
		v.require("BR-37", a.CategoryTax.CategoryCode, where+": die Steuerkategorie (BT-102)")
		if !a.HasReason() {
			v.fail("BR-38", "%s: es fehlt der Grund (BT-104) oder der Grundschlüssel (BT-105)", where)
		}
		v.decimals("BR-DEC-05", a.ActualAmount, where+": der Betrag (BT-99)")
		v.decimals("BR-DEC-06", a.BasisAmount, where+": der Grundbetrag (BT-100)")
	}

	for i, line := range doc.Transaction.Lines {
		for j, a := range line.Settlement.AllowancesCharges {
			if a.IsCharge() {
				where := fmt.Sprintf("Position %d, Zuschlag %d", i+1, j+1)
				v.require("BR-43", a.ActualAmount, where+": der Betrag (BT-141)")
				if !a.HasReason() {
					v.fail("BR-44", "%s: es fehlt der Grund (BT-144) oder der Grundschlüssel (BT-145)", where)
				}
				v.decimals("BR-DEC-27", a.ActualAmount, where+": der Betrag (BT-141)")
				v.decimals("BR-DEC-28", a.BasisAmount, where+": der Grundbetrag (BT-142)")
				continue
			}
			where := fmt.Sprintf("Position %d, Nachlass %d", i+1, j+1)
			v.require("BR-41", a.ActualAmount, where+": der Betrag (BT-136)")
			if !a.HasReason() {
				v.fail("BR-42", "%s: es fehlt der Grund (BT-139) oder der Grundschlüssel (BT-140)", where)
			}
			v.decimals("BR-DEC-24", a.ActualAmount, where+": der Betrag (BT-136)")
			v.decimals("BR-DEC-25", a.BasisAmount, where+": der Grundbetrag (BT-137)")
		}
	}
}

func (v *validator) checkTaxBreakdown(doc *CIIInvoice) {
	taxes := doc.Transaction.Settlement.Taxes
	if len(taxes) == 0 {
		v.fail("BR-CO-18", "Die Rechnung enthält keine Aufschlüsselung der Umsatzsteuer (BG-23)")
		return
	}

	for i, tax := range taxes {
		n := i + 1
		category := strings.ToUpper(strings.TrimSpace(tax.CategoryCode))

		v.require("BR-45", tax.BasisAmount, fmt.Sprintf("Steuergruppe %d: die Bemessungsgrundlage (BT-116)", n))
		v.require("BR-46", tax.CalculatedAmount, fmt.Sprintf("Steuergruppe %d: der Steuerbetrag (BT-117)", n))
		v.decimals("BR-DEC-19", tax.BasisAmount, fmt.Sprintf("Steuergruppe %d: die Bemessungsgrundlage (BT-116)", n))
		v.decimals("BR-DEC-20", tax.CalculatedAmount, fmt.Sprintf("Steuergruppe %d: der Steuerbetrag (BT-117)", n))

		if !v.require("BR-47", category, fmt.Sprintf("Steuergruppe %d: der Steuerkategoriecode (BT-118)", n)) {
			continue
		}
		if !isVATCategoryCode(category) {
			v.fail("BR-CL-18", "Steuergruppe %d: der Kategoriecode %q steht nicht in UNTDID 5305", n, category)
			continue
		}

		// BR-48: außer bei "nicht steuerbar" ist der Steuersatz anzugeben.
		if category != "O" {
			v.require("BR-48", tax.RatePercent, fmt.Sprintf("Steuergruppe %d: der Steuersatz (BT-119)", n))
		}
	}

	v.checkCategoryRules(doc)
}

// rateConstraint says what a VAT category permits as a rate.
type rateConstraint uint8

const (
	rateAny         rateConstraint = iota // keine Einschränkung
	ratePositive                          // größer null
	rateZero                              // genau null
	rateNonNegative                       // null oder größer
	rateAbsent                            // gar kein Satz
)

// categorySpec describes one VAT category family of EN 16931.
//
// The families have the same shape — what differs is the rate constraint, what
// the tax amount has to be, whether an exemption reason is required or
// forbidden, and which VAT identifiers have to be on the document. Writing that
// out as a table rather than nine near-identical branches is what keeps the
// nine of them from drifting apart.
//
// Note the family names: the intra-community rules are BR-IC, not BR-K. The
// category *code* is "K", the rule *family* is "IC" — a distinction worth
// keeping straight, because a wrong rule identifier looks exactly like a
// citation somebody can look up.
type categorySpec struct {
	code   string // UNTDID 5305
	family string // EN 16931 rule family

	// lineRate applies on invoice lines (-05), acRate to allowances and charges
	// on document level (-06, -07). They differ: on the Canary Islands rate a
	// line has to carry a positive rate, a discount on it may be zero.
	lineRate rateConstraint
	acRate   rateConstraint

	// taxZero: der Steuerbetrag der Gruppe muss null sein (-09). Ohne das Flag
	// folgt der Steuerbetrag aus Bemessungsgrundlage mal Satz.
	taxZero bool

	reasonRequired  bool // -10
	reasonForbidden bool // -10

	sellerVatRequired bool // -02, -03, -04
	// sellerVatOnly: nur die USt-IdNr. genügt, nicht die Steuernummer. Bei
	// grenzüberschreitenden Umsätzen ist die nationale Steuernummer für den
	// Empfängerstaat wertlos, deshalb lässt die Norm sie dort nicht zu.
	sellerVatOnly      bool
	buyerVatRequired   bool
	sellerVatForbidden bool
}

var categorySpecs = []categorySpec{
	{code: "S", family: "S", lineRate: ratePositive, acRate: ratePositive,
		reasonForbidden: true, sellerVatRequired: true},
	{code: "Z", family: "Z", lineRate: rateZero, acRate: rateZero, taxZero: true,
		reasonForbidden: true, sellerVatRequired: true},
	{code: "E", family: "E", lineRate: rateZero, acRate: rateZero, taxZero: true,
		reasonRequired: true, sellerVatRequired: true},
	{code: "AE", family: "AE", lineRate: rateZero, acRate: rateZero, taxZero: true,
		reasonRequired: true, sellerVatRequired: true, buyerVatRequired: true},
	{code: "K", family: "IC", lineRate: rateZero, acRate: rateZero, taxZero: true,
		reasonRequired: true, sellerVatRequired: true, sellerVatOnly: true, buyerVatRequired: true},
	{code: "G", family: "G", lineRate: rateZero, acRate: rateZero, taxZero: true,
		reasonRequired: true, sellerVatRequired: true, sellerVatOnly: true},
	{code: "O", family: "O", lineRate: rateAbsent, acRate: rateAbsent, taxZero: true,
		reasonRequired: true, sellerVatForbidden: true},
	{code: "L", family: "AF", lineRate: ratePositive, acRate: rateNonNegative,
		reasonForbidden: true, sellerVatRequired: true},
	{code: "M", family: "AG", lineRate: rateNonNegative, acRate: rateNonNegative,
		reasonForbidden: true, sellerVatRequired: true},
}

func specForCategory(code string) (categorySpec, bool) {
	for _, s := range categorySpecs {
		if s.code == strings.ToUpper(strings.TrimSpace(code)) {
			return s, true
		}
	}
	return categorySpec{}, false
}

// checkCategoryRules implements the per-category rules of EN 16931.
//
// They decide whether a document says what it claims. A breakdown carrying
// "reverse charge" together with a rate of 19 % contradicts itself, and booking
// from it would produce the tax twice.
//
// The structure follows the norm's own: the rules ending in -02, -05 are
// triggered by invoice lines, -03 and -06 by document level allowances, -04 and
// -07 by charges, and the rest by the breakdown groups themselves.
func (v *validator) checkCategoryRules(doc *CIIInvoice) {
	taxes := doc.Transaction.Settlement.Taxes

	inBreakdown := map[string]bool{}
	for _, tax := range taxes {
		inBreakdown[strings.ToUpper(strings.TrimSpace(tax.CategoryCode))] = true
	}

	// Aus den Positionen: -01, -02, -05.
	for i, line := range doc.Transaction.Lines {
		code := strings.ToUpper(strings.TrimSpace(line.Settlement.Tax.CategoryCode))
		spec, known := specForCategory(code)
		if !known {
			continue
		}
		v.requireBreakdownGroup(spec, inBreakdown, fmt.Sprintf("Position %d", i+1))
		v.checkIdentifiers("BR-"+spec.family+"-02", spec, doc)
		v.checkRateConstraint("BR-"+spec.family+"-05", spec, spec.lineRate,
			fmt.Sprintf("Position %d", i+1), line.Settlement.Tax.RatePercent)
	}

	// Aus den Nachlässen: -01, -03, -06. Aus den Zuschlägen: -01, -04, -07.
	for i, a := range doc.DocumentAllowances() {
		code := strings.ToUpper(strings.TrimSpace(a.CategoryTax.CategoryCode))
		spec, known := specForCategory(code)
		if !known {
			continue
		}
		where := fmt.Sprintf("Nachlass %d auf Dokumentebene", i+1)
		v.requireBreakdownGroup(spec, inBreakdown, where)
		v.checkIdentifiers("BR-"+spec.family+"-03", spec, doc)
		v.checkRateConstraint("BR-"+spec.family+"-06", spec, spec.acRate, where, a.CategoryTax.RatePercent)
	}
	for i, a := range doc.DocumentCharges() {
		code := strings.ToUpper(strings.TrimSpace(a.CategoryTax.CategoryCode))
		spec, known := specForCategory(code)
		if !known {
			continue
		}
		where := fmt.Sprintf("Zuschlag %d auf Dokumentebene", i+1)
		v.requireBreakdownGroup(spec, inBreakdown, where)
		v.checkIdentifiers("BR-"+spec.family+"-04", spec, doc)
		v.checkRateConstraint("BR-"+spec.family+"-07", spec, spec.acRate, where, a.CategoryTax.RatePercent)
	}

	// Aus der Aufschlüsselung: -08, -09, -10 und die Zusatzregeln.
	for i, tax := range taxes {
		n := i + 1
		code := strings.ToUpper(strings.TrimSpace(tax.CategoryCode))
		spec, known := specForCategory(code)
		if !known {
			continue
		}

		v.checkBreakdownBase("BR-"+spec.family+"-08", doc, n, code, tax)
		v.checkBreakdownTax("BR-"+spec.family+"-09", spec, n, code, tax)

		reason := strings.TrimSpace(tax.ExemptionReason) != "" || strings.TrimSpace(tax.ExemptionReasonCode) != ""
		if spec.reasonRequired && !reason {
			v.fail("BR-"+spec.family+"-10",
				"Steuergruppe %d: bei Kategorie %q ist der Grund für Befreiung oder Umkehr der Steuerschuld anzugeben (BT-120/BT-121)",
				n, code)
		}
		if spec.reasonForbidden && reason {
			v.fail("BR-"+spec.family+"-10",
				"Steuergruppe %d: bei Kategorie %q darf kein Befreiungsgrund angegeben werden — der Umsatz ist steuerpflichtig",
				n, code)
		}

		if spec.family == "IC" {
			period := doc.Transaction.Delivery.Period
			hasPeriod := period != nil &&
				(strings.TrimSpace(period.Start.Value) != "" || strings.TrimSpace(period.End.Value) != "")
			if doc.DeliveryDate() == "" && !hasPeriod {
				v.fail("BR-IC-11", "Bei einer innergemeinschaftlichen Lieferung ist das Lieferdatum oder der Abrechnungszeitraum anzugeben (BT-72)")
			}
			if doc.Transaction.Delivery.ShipTo == nil || doc.Transaction.Delivery.ShipTo.CountryCode() == "" {
				v.fail("BR-IC-12", "Bei einer innergemeinschaftlichen Lieferung ist das Bestimmungsland anzugeben (BT-80)")
			}
		}
		if spec.family == "O" {
			if len(taxes) > 1 {
				v.fail("BR-O-11", "Eine Rechnung mit der Kategorie \"nicht steuerbar\" darf keine weiteren Steuergruppen enthalten")
			}
			for j, line := range doc.Transaction.Lines {
				if strings.ToUpper(strings.TrimSpace(line.Settlement.Tax.CategoryCode)) != "O" {
					v.fail("BR-O-12", "Position %d führt eine andere Steuerkategorie, obwohl die Rechnung als nicht steuerbar ausgewiesen ist", j+1)
					break
				}
			}
			for j, a := range doc.DocumentAllowances() {
				if strings.ToUpper(strings.TrimSpace(a.CategoryTax.CategoryCode)) != "O" {
					v.fail("BR-O-13", "Nachlass %d auf Dokumentebene führt eine andere Steuerkategorie, obwohl die Rechnung als nicht steuerbar ausgewiesen ist", j+1)
					break
				}
			}
			for j, a := range doc.DocumentCharges() {
				if strings.ToUpper(strings.TrimSpace(a.CategoryTax.CategoryCode)) != "O" {
					v.fail("BR-O-14", "Zuschlag %d auf Dokumentebene führt eine andere Steuerkategorie, obwohl die Rechnung als nicht steuerbar ausgewiesen ist", j+1)
					break
				}
			}
		}
	}
}

// requireBreakdownGroup implements the -01 rule: whatever category a line, an
// allowance or a charge uses, the breakdown has to carry a group for it.
// Otherwise the turnover is missing from the Voranmeldung.
func (v *validator) requireBreakdownGroup(spec categorySpec, inBreakdown map[string]bool, where string) {
	if inBreakdown[spec.code] {
		return
	}
	v.fail("BR-"+spec.family+"-01",
		"%s führt die Steuerkategorie %q, die Aufschlüsselung enthält dafür aber keine Gruppe",
		where, spec.code)
}

// checkIdentifiers implements the -02, -03 and -04 rules: which VAT identifiers
// a document has to carry for a given category.
func (v *validator) checkIdentifiers(rule string, spec categorySpec, doc *CIIInvoice) {
	seller := doc.Transaction.Agreement.Seller
	buyer := doc.Transaction.Agreement.Buyer

	if spec.sellerVatRequired {
		identified := seller.VatID() != ""
		if !spec.sellerVatOnly {
			identified = identified || seller.TaxNumber() != ""
		}
		if !identified {
			if spec.sellerVatOnly {
				v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. des Verkäufers (BT-31) — die nationale Steuernummer genügt nicht", spec.code)
			} else {
				v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. oder Steuernummer des Verkäufers (BT-31/BT-32)", spec.code)
			}
		}
	}
	if spec.buyerVatRequired && buyer.VatID() == "" {
		v.fail(rule, "Bei der Steuerkategorie %q braucht die Rechnung die USt-IdNr. des Erwerbers (BT-48)", spec.code)
	}
	// Verboten ist allein die USt-IdNr. (schemeID "VA"). Die Steuernummer
	// (schemeID "FC") darf ein nicht steuerbarer Umsatz sehr wohl tragen —
	// beides zusammenzuwerfen meldete korrekte Rechnungen als falsch.
	if spec.sellerVatForbidden && (seller.VatID() != "" || buyer.VatID() != "") {
		v.fail(rule, "Bei der Steuerkategorie %q darf die Rechnung keine USt-IdNr. tragen — der Umsatz ist nicht steuerbar", spec.code)
	}
}

// checkRateConstraint applies the -05, -06 and -07 rules of a category family.
func (v *validator) checkRateConstraint(rule string, spec categorySpec, want rateConstraint, where, rawRate string) {
	trimmed := strings.TrimSpace(rawRate)

	if want == rateAbsent {
		if trimmed != "" {
			if rate, err := domain.ParseCents(trimmed); err == nil && rate != 0 {
				v.fail(rule, "%s: bei Kategorie %q darf kein Steuersatz angegeben werden, angegeben sind %s %%",
					where, spec.code, trimmed)
			}
		}
		return
	}
	if trimmed == "" {
		return
	}
	rate, err := domain.ParseCents(trimmed)
	if err != nil {
		return
	}
	switch {
	case want == ratePositive && rate <= 0:
		v.fail(rule, "%s: bei Kategorie %q muss der Steuersatz größer null sein, angegeben sind %s %%",
			where, spec.code, trimmed)
	case want == rateZero && rate != 0:
		v.fail(rule, "%s: bei Kategorie %q muss der Steuersatz null sein, angegeben sind %s %%",
			where, spec.code, trimmed)
	case want == rateNonNegative && rate < 0:
		v.fail(rule, "%s: bei Kategorie %q darf der Steuersatz nicht negativ sein, angegeben sind %s %%",
			where, spec.code, trimmed)
	}
}

// checkBreakdownBase applies the -08 rule: the taxable amount of a breakdown
// group equals the lines of the same category, plus the charges, minus the
// allowances that carry it.
func (v *validator) checkBreakdownBase(rule string, doc *CIIInvoice, n int, code string, tax ciiTradeTax) {
	declared, err := domain.ParseCents(tax.BasisAmount)
	if err != nil {
		return
	}

	// Führt die Aufschlüsselung dieselbe Kategorie mehrfach, trennt sie allein
	// der Steuersatz — dann muss auch danach zugeordnet werden. Bei nur einer
	// Gruppe je Kategorie ist das unnötig und würde an Belegen scheitern, die
	// den Satz auf der Position weglassen.
	groups := 0
	for _, other := range doc.Transaction.Settlement.Taxes {
		if strings.ToUpper(strings.TrimSpace(other.CategoryCode)) == code {
			groups++
		}
	}
	matchRate := groups > 1

	var sum domain.Cents
	var matched int
	for _, line := range doc.Transaction.Lines {
		if strings.ToUpper(strings.TrimSpace(line.Settlement.Tax.CategoryCode)) != code {
			continue
		}
		if matchRate && !sameRate(line.Settlement.Tax.RatePercent, tax.RatePercent) {
			continue
		}
		amount, err := domain.ParseCents(line.Settlement.Summation.LineTotal)
		if err != nil {
			return
		}
		sum += amount
		matched++
	}
	if matched == 0 {
		return
	}

	for _, a := range doc.DocumentCharges() {
		if strings.ToUpper(strings.TrimSpace(a.CategoryTax.CategoryCode)) != code {
			continue
		}
		if matchRate && !sameRate(a.CategoryTax.RatePercent, tax.RatePercent) {
			continue
		}
		amount, err := domain.ParseCents(a.ActualAmount)
		if err != nil {
			return
		}
		sum += amount
	}
	for _, a := range doc.DocumentAllowances() {
		if strings.ToUpper(strings.TrimSpace(a.CategoryTax.CategoryCode)) != code {
			continue
		}
		if matchRate && !sameRate(a.CategoryTax.RatePercent, tax.RatePercent) {
			continue
		}
		amount, err := domain.ParseCents(a.ActualAmount)
		if err != nil {
			return
		}
		sum -= amount
	}

	if sum != declared {
		v.fail(rule, "Steuergruppe %d: die Bemessungsgrundlage ist %s, Positionen, Zu- und Abschläge der Kategorie %q ergeben aber %s",
			n, declared, code, sum)
	}
}

// checkBreakdownTax applies the -09 rule: what the tax amount of a group has to
// be. For the categories without tax that is zero; for the others it follows
// from the taxable amount and the rate.
func (v *validator) checkBreakdownTax(rule string, spec categorySpec, n int, code string, tax ciiTradeTax) {
	amount, err := domain.ParseCents(tax.CalculatedAmount)
	if err != nil {
		return
	}
	if spec.taxZero {
		if amount != 0 {
			v.fail(rule, "Steuergruppe %d: bei Kategorie %q muss der Steuerbetrag null sein, angegeben sind %s",
				n, code, amount)
		}
		return
	}
	base, errBase := domain.ParseCents(tax.BasisAmount)
	rate, errRate := domain.ParseCents(tax.RatePercent)
	if errBase != nil || errRate != nil {
		return
	}
	expected := domain.MulRound(base.Abs(), int64(rate), 10000)
	if diff := (amount.Abs() - expected).Abs(); diff > vatRoundingTolerance {
		v.fail(rule, "Steuergruppe %d: aus %s zu %s %% folgt bei Kategorie %q ein Steuerbetrag von %s, angegeben ist %s",
			n, base, tax.RatePercent, code, expected, amount)
	}
}

func sameRate(a, b string) bool {
	ra, errA := domain.ParseCents(strings.TrimSpace(a))
	rb, errB := domain.ParseCents(strings.TrimSpace(b))
	if errA != nil || errB != nil {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return ra == rb
}

// vatRoundingTolerance is the one currency unit EN 16931 allows when checking a
// tax amount against base times rate.
//
// It is not sloppiness: currencies without decimal places, and systems that
// round per line instead of per group, land a cent beside the exact result often
// enough that an exact comparison would report correct invoices as broken.
const vatRoundingTolerance = domain.Cents(100)

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

	allowanceTotal, okAllowanceTotal := optionalAmount(sum.AllowanceTotal)
	chargeTotal, okChargeTotal := optionalAmount(sum.ChargeTotal)
	prepaid, okPrepaid := optionalAmount(sum.TotalPrepaid)
	rounding, okRounding := optionalAmount(sum.RoundingAmount)

	v.decimals("BR-DEC-09", sum.LineTotal, "Die Summe der Positionsbeträge (BT-106)")
	v.decimals("BR-DEC-10", sum.AllowanceTotal, "Die Summe der Nachlässe (BT-107)")
	v.decimals("BR-DEC-11", sum.ChargeTotal, "Die Summe der Zuschläge (BT-108)")
	v.decimals("BR-DEC-12", sum.TaxBasisTotal, "Der Gesamtbetrag ohne Umsatzsteuer (BT-109)")
	v.decimals("BR-DEC-13", doc.TaxTotal().Value, "Der Gesamtbetrag der Umsatzsteuer (BT-110)")
	v.decimals("BR-DEC-15", doc.TaxTotalInAccountingCurrency().Value,
		"Der Gesamtbetrag der Umsatzsteuer in der Abrechnungswährung (BT-111)")
	v.decimals("BR-DEC-14", sum.GrandTotal, "Der Gesamtbetrag mit Umsatzsteuer (BT-112)")
	v.decimals("BR-DEC-16", sum.TotalPrepaid, "Der bereits gezahlte Betrag (BT-113)")
	v.decimals("BR-DEC-17", sum.RoundingAmount, "Der Rundungsbetrag (BT-114)")
	v.decimals("BR-DEC-18", sum.DuePayable, "Der fällige Betrag (BT-115)")

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

	// BR-CO-11 und BR-CO-12: die ausgewiesenen Summen der Nachlässe und
	// Zuschläge müssen den einzelnen Beträgen entsprechen.
	if okAllowanceTotal {
		if declared, ok := sumAllowanceCharges(doc.DocumentAllowances()); ok && declared != allowanceTotal {
			v.fail("BR-CO-11", "Die Summe der Nachlässe ist mit %s ausgewiesen, die einzelnen Nachlässe ergeben aber %s",
				allowanceTotal, declared)
		}
	}
	if okChargeTotal {
		if declared, ok := sumAllowanceCharges(doc.DocumentCharges()); ok && declared != chargeTotal {
			v.fail("BR-CO-12", "Die Summe der Zuschläge ist mit %s ausgewiesen, die einzelnen Zuschläge ergeben aber %s",
				chargeTotal, declared)
		}
	}

	// BR-CO-13: netto = Positionen - Nachlässe + Zuschläge.
	if okLineTotal && okBasis && okAllowanceTotal && okChargeTotal {
		if expected := lineTotal - allowanceTotal + chargeTotal; expected != taxBasis {
			v.fail("BR-CO-13", "Der Gesamtbetrag ohne Umsatzsteuer ist %s, aus Positionen, Nachlässen und Zuschlägen folgt aber %s",
				taxBasis, expected)
		}
	}

	// BR-CO-14: die ausgewiesene Gesamtsteuer entspricht der Summe der Gruppen.
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
	stated := doc.TaxTotal().Value
	if declared, ok := optionalAmount(stated); ok && taxKnown &&
		strings.TrimSpace(stated) != "" && declared != taxTotal {
		v.fail("BR-CO-14", "Der Gesamtbetrag der Umsatzsteuer ist mit %s ausgewiesen, die Steuergruppen ergeben aber %s",
			declared, taxTotal)
	}

	// BR-CO-15: Bruttobetrag = Nettobetrag + Steuer.
	if okBasis && okGrand && taxKnown && taxBasis+taxTotal != grand {
		v.fail("BR-CO-15", "Der Gesamtbetrag mit Umsatzsteuer ist %s, netto plus Steuer ergeben aber %s",
			grand, taxBasis+taxTotal)
	}

	// BR-CO-16: fälliger Betrag = brutto - bereits gezahlt + Rundung.
	if okGrand && okDue && okPrepaid && okRounding {
		if expected := grand - prepaid + rounding; expected != due {
			v.fail("BR-CO-16", "Der fällige Betrag ist %s, aus Gesamtbetrag, Anzahlung und Rundung folgt aber %s",
				due, expected)
		}
	}

	// BR-CO-17: der Steuerbetrag je Gruppe folgt aus Bemessungsgrundlage und Satz.
	for i, tax := range doc.Transaction.Settlement.Taxes {
		base, errBase := domain.ParseCents(tax.BasisAmount)
		amount, errAmount := domain.ParseCents(tax.CalculatedAmount)
		rate, errRate := domain.ParseCents(tax.RatePercent)
		if errBase != nil || errAmount != nil || errRate != nil {
			continue
		}
		expected := domain.MulRound(base.Abs(), int64(rate), 10000)
		if diff := (amount.Abs() - expected).Abs(); diff > vatRoundingTolerance {
			v.fail("BR-CO-17", "Steuergruppe %d: aus %s zu %s %% folgt ein Steuerbetrag von %s, angegeben ist %s",
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

// optionalAmount reads a field that may legitimately be absent. Absent counts as
// zero — that is how every sum rule of the norm treats it.
func optionalAmount(raw string) (domain.Cents, bool) {
	if strings.TrimSpace(raw) == "" {
		return 0, true
	}
	value, err := domain.ParseCents(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func sumAllowanceCharges(entries []ciiAllowanceCharge) (domain.Cents, bool) {
	var total domain.Cents
	for _, e := range entries {
		amount, err := domain.ParseCents(e.ActualAmount)
		if err != nil {
			return 0, false
		}
		total += amount
	}
	return total, true
}
