package einvoice

import (
	"fmt"
	"sort"
	"strings"
)

// RulesetVersion identifies the state of this checker.
//
// It is stored with every result: a verdict without the rules that produced it
// is not evidence, and a re-run months later has to be comparable to the first.
const RulesetVersion = "buchfink-en16931/2026.3"

// Severity separates what makes a document unusable from what is worth knowing.
//
// Which is which comes from the rule set, not from Buchfink: EN 16931 flags
// exactly one of its rules as a warning (BR-51, the card number), and the
// German CIUS marks a good dozen of its own that way. Inventing a different
// severity would mean disagreeing with a specification while claiming to
// implement it.
type Severity string

const (
	SeverityFatal   Severity = "fatal"
	SeverityWarning Severity = "warning"
	// SeverityInfo is a remark that is not a defect at all. EN 16931 has none;
	// the layers above it do.
	SeverityInfo Severity = "information"
)

// Finding is one violated business rule.
type Finding struct {
	// Rule is the EN 16931 identifier, e.g. "BR-CO-15". Naming it lets a reader
	// look the rule up instead of taking Buchfink's word for it.
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	// Where names the part of the document, e.g. "Position 3". Empty when the
	// finding is about the document as a whole.
	Where string `json:"where,omitempty"`
	// Terms are the business terms the rule is about, from the standard's own
	// inventory. A user interface can point at the field instead of making
	// somebody decode a rule number.
	Terms   []string `json:"terms,omitempty"`
	Message string   `json:"message"`
}

// Result is the outcome of checking an invoice.
type Result struct {
	// Rulesets names every rule set that ran, in order. A verdict without it is
	// not evidence: "passed" means nothing until it says passed *what*, and a
	// document that clears EN 16931 can still fail the German CIUS.
	Rulesets []string  `json:"rulesets"`
	Syntax   Syntax    `json:"syntax"`
	Profile  string    `json:"profile"`
	Findings []Finding `json:"findings,omitempty"`
}

// Valid reports whether the document violated no fatal rule.
func (r Result) Valid() bool { return r.ErrorCount() == 0 }

// ErrorCount counts the fatal violations.
func (r Result) ErrorCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityFatal {
			n++
		}
	}
	return n
}

// ByRule returns the findings for one rule.
func (r Result) ByRule(rule string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

// RulesChecked lists every rule this checker implements, sorted.
func RulesChecked() []string {
	out := make([]string, 0, len(checkedRules))
	for rule := range checkedRules {
		out = append(out, rule)
	}
	sort.Strings(out)
	return out
}

// RulesInStandard lists every rule EN 16931 defines for the semantic model and
// its code lists, sorted.
func RulesInStandard() []string {
	out := make([]string, 0, len(en16931Rules))
	for rule := range en16931Rules {
		out = append(out, rule)
	}
	sort.Strings(out)
	return out
}

// RulesUnchecked lists the rules of the standard this checker does not
// implement.
//
// It is exported on purpose. Somebody who needs to know whether one particular
// rule was looked at can ask, instead of inferring it from a percentage.
func RulesUnchecked() []string {
	var out []string
	for _, rule := range RulesInStandard() {
		if !checkedRules[rule] {
			out = append(out, rule)
		}
	}
	return out
}

// Rule returns what the standard says about one rule: which business group it
// hangs off, which terms it is about, and whether it is fatal.
func Rule(id string) (RuleInfo, bool) {
	info, ok := en16931Rules[id]
	return info, ok
}

// Validate checks an invoice against EN 16931.
//
// The rules run on the semantic model, which is how the standard itself defines
// them — its business rules live in an abstract rule set and each syntax only
// supplies the bindings. A CII invoice and a UBL invoice are therefore judged
// by exactly the same code, which is the only way to guarantee they are judged
// alike.
func Validate(inv *Invoice) Result {
	result := Result{
		Rulesets: []string{RulesetVersion},
		Syntax:   inv.Syntax,
		Profile:  inv.SpecificationID,
	}
	v := &validator{inv: inv, seen: map[string]bool{}}

	v.checkDocument()
	v.checkParties()
	v.checkDelivery()
	v.checkPeriods()
	v.checkPaymentInstructions()
	v.checkSupportingDocuments()
	v.checkPrecedingInvoices()
	v.checkAllowancesCharges()
	v.checkLines()
	v.checkVATBreakdown()
	v.checkTotals()
	v.checkCategories()
	v.checkCodeLists()

	result.Findings = v.findings
	return result
}

type validator struct {
	inv      *Invoice
	findings []Finding
	// seen suppresses repetitions. Several rules state one document-wide fact
	// ("the seller VAT identifier is missing") but are triggered per line;
	// reporting them once per line would bury everything else.
	seen map[string]bool
}

func (v *validator) report(rule, where, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	key := rule + "\x00" + where + "\x00" + message
	if v.seen[key] {
		return
	}
	v.seen[key] = true

	severity := SeverityFatal
	var terms []string
	if info, ok := en16931Rules[rule]; ok {
		severity = info.Severity
		terms = info.Terms
	}
	v.findings = append(v.findings, Finding{
		Rule: rule, Severity: severity, Where: where, Terms: terms, Message: message,
	})
}

// fail reports a violation of a document-level rule.
func (v *validator) fail(rule, format string, args ...any) {
	v.report(rule, "", format, args...)
}

// failAt reports a violation located in a specific part of the document.
func (v *validator) failAt(rule, where, format string, args ...any) {
	v.report(rule, where, format, args...)
}

// require reports the rule if the value is blank, and says whether it was there.
func (v *validator) require(rule, value, label string) bool {
	if strings.TrimSpace(value) == "" {
		v.fail(rule, "%s fehlt", label)
		return false
	}
	return true
}

func (v *validator) requireAt(rule, where, value, label string) bool {
	if strings.TrimSpace(value) == "" {
		v.failAt(rule, where, "%s fehlt", label)
		return false
	}
	return true
}

// requireAmount reports the rule if the amount is absent or unreadable.
func (v *validator) requireAmount(rule string, a Amount, label string) (Cents, bool) {
	if !a.Present() {
		v.fail(rule, "%s fehlt", label)
		return 0, false
	}
	cents, err := a.Cents()
	if err != nil {
		v.fail(rule, "%s ist mit %q unlesbar", label, a)
		return 0, false
	}
	return cents, true
}

// decimals implements the BR-DEC family: an amount is limited to two decimal
// places.
//
// Checking the written form rather than a parsed value is the whole point. A
// third decimal place means the document states an amount no account can hold,
// and a parser that rounded it away on the way in would hide exactly that.
func (v *validator) decimals(rule string, a Amount, label string) {
	v.decimalsAt(rule, "", a, label)
}

func (v *validator) decimalsAt(rule, where string, a Amount, label string) {
	// Was gar keine Dezimalzahl ist, wird hier gemeldet und nicht erst dort, wo
	// gerechnet wird. Ein Betrag in Exponentialschreibweise hat nach Decimals()
	// null Nachkommastellen; ohne diese Zeile bliebe er unbeanstandet und fiele
	// anschließend still aus jeder Summenprüfung heraus.
	if a.Present() {
		if _, ok := a.Rat(); !ok {
			v.report(rule, where, "%s ist mit %q keine Dezimalzahl", label, a)
			return
		}
	}
	if n := a.Decimals(); n > 2 {
		v.report(rule, where, "%s hat %d Nachkommastellen, erlaubt sind höchstens zwei", label, n)
	}
}

// inList reports the rule when a stated code is not in a code list. An absent
// code is not a violation — its presence is somebody else's rule.
func (v *validator) inList(rule string, code string, list map[string]struct{}, label string) {
	v.inListAt(rule, "", code, list, label)
}

func (v *validator) inListAt(rule, where, code string, list map[string]struct{}, label string) {
	code = strings.TrimSpace(code)
	if code == "" {
		return
	}
	if _, ok := list[code]; ok {
		return
	}
	// Die Norm vergleicht die Schreibweise genau, und ein Prüfer, der das nicht
	// tut, hätte der Rechnung ein Zeugnis ausgestellt, das sie beim nächsten
	// Empfänger nicht bekommt. Der Fund bleibt deshalb ein Fehler — aber er
	// sagt, wenn allein die Groß- und Kleinschreibung ihn ausgelöst hat. Sonst
	// sucht jemand stundenlang nach einem falschen Schlüssel, der keiner ist.
	if _, ok := list[strings.ToUpper(code)]; ok {
		v.report(rule, where,
			"%s: %q steht nicht in der Codeliste — mit Großschreibung als %q wäre der Schlüssel gültig",
			label, code, strings.ToUpper(code))
		return
	}
	v.report(rule, where, "%s: %q steht nicht in der Codeliste", label, code)
}

func linePos(i int) string      { return fmt.Sprintf("Position %d", i+1) }
func breakdownPos(i int) string { return fmt.Sprintf("Steuergruppe %d", i+1) }

// Ruleset is a layer of rules on top of EN 16931.
//
// XRechnung and ZUGFeRD are not separate formats — they are a CIUS and a set of
// profiles *over* the standard, adding rules and narrowing what a document may
// contain. Modelling them as layers rather than as parallel implementations is
// what keeps them from drifting: they see the same parsed invoice as the core
// rules do, and a document is judged by the core exactly once no matter how
// many layers apply.
type Ruleset interface {
	// ID identifies the rule set, e.g. "xrechnung/3.0".
	ID() string
	// Check returns the findings this layer adds. The core rules have already
	// run; a layer neither repeats nor overrides them.
	Check(inv *Invoice) []Finding
}

// ValidateWith checks an invoice against EN 16931 and then against every layer
// given, in order.
func ValidateWith(inv *Invoice, layers ...Ruleset) Result {
	result := Validate(inv)
	for _, layer := range layers {
		if layer == nil {
			continue
		}
		result.Rulesets = append(result.Rulesets, layer.ID())
		result.Findings = append(result.Findings, layer.Check(inv)...)
	}
	return result
}

// Reporter collects the findings of a rule set layer.
//
// It exists so a layer does not have to restate, at every call site, what
// severity a rule carries and which business terms it is about. Those come from
// the layer's own inventory — the same arrangement the core rules use, and the
// reason a wrong severity or an unknown rule identifier cannot slip in
// unnoticed.
type Reporter struct {
	rules    map[string]RuleInfo
	findings []Finding
	seen     map[string]bool
}

// NewReporter starts collecting for a layer with the given rule inventory.
func NewReporter(rules map[string]RuleInfo) *Reporter {
	return &Reporter{rules: rules, seen: map[string]bool{}}
}

// Report records a violation. The severity and the business terms come from the
// inventory; a rule that is not in it is reported as fatal, which is the safe
// direction — an unknown rule identifier is a defect in the layer, and burying
// it as a remark would hide it.
func (r *Reporter) Report(rule, where, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	key := rule + "\x00" + where + "\x00" + message
	if r.seen[key] {
		return
	}
	r.seen[key] = true

	finding := Finding{Rule: rule, Severity: SeverityFatal, Where: where, Message: message}
	if info, ok := r.rules[rule]; ok {
		finding.Severity = info.Severity
		finding.Terms = info.Terms
	}
	r.findings = append(r.findings, finding)
}

// Require reports the rule when the value is blank, and says whether it was
// there.
func (r *Reporter) Require(rule, where, value, label string) bool {
	if strings.TrimSpace(value) == "" {
		r.Report(rule, where, "%s fehlt", label)
		return false
	}
	return true
}

// Findings returns what was collected.
func (r *Reporter) Findings() []Finding { return r.findings }
