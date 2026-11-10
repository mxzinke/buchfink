package ebilanz

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Stammdaten der Instanz (GCD-Modul, Leitfaden B.14.3.1).

const gcd = "genInfo."

// reportElements sind die angekündigten Berichtsbestandteile (Leitfaden
// B.9.2, B.14.3.1.10).
type reportElements struct {
	balance, income, incomeMicroBilG, appropriation, accounts, assetComparison, transfer bool
}

// gcdNodes baut die Stammdaten.
func (g *generator) gcdNodes(announced reportElements) []*node {
	b := g.b
	in := g.in
	var out []*node

	out = append(out, g.choice("report.id.reportStatus", "report.id.reportStatus.reportStatus.E"))
	out = append(out, g.choice("report.id.revisionStatus", "report.id.revisionStatus.revisionStatus.E"))
	selected := []string{}
	add := func(on bool, name string) {
		if on {
			selected = append(selected, "report.id.reportElement.reportElements."+name)
		}
	}
	add(announced.balance, "B")
	add(announced.income, "GuV")
	add(announced.incomeMicroBilG, "GuVMicroBilG")
	add(announced.appropriation, "EV")
	add(announced.accounts, "KS")
	add(announced.assetComparison, "BVV")
	add(announced.transfer, "STU")
	out = append(out, g.choice("report.id.reportElement", selected...))

	statementType := "report.id.statementType.statementType.E"
	if in.Kind == KindOpening {
		statementType = "report.id.statementType.statementType.EB"
	}
	out = append(out, g.choice("report.id.statementType", statementType))
	// Leitfaden B.14.3.1.11: bei Kapitalgesellschaften eine Ausprägung der
	// steuerlichen Bilanzart für Personengesellschaften mit NIL.
	if _, ok := b.tax.Element(gcd + "report.id.statementType.tax"); ok {
		out = append(out, b.tuple(gcd+"report.id.statementType.tax",
			b.nilItem(gcd+"report.id.statementType.tax.statementTypeTax.GHB")))
	}
	out = append(out, b.item(gcd+"report.id.incomeStatementendswithBalProfit", fmt.Sprint(g.appropriated)))
	standard := "report.id.accountingStandard.accountingStandard.HAOE"
	if g.commercial {
		standard = "report.id.accountingStandard.accountingStandard.HGBM"
	}
	out = append(out, g.choice("report.id.accountingStandard", standard))
	out = append(out, g.choice("report.id.specialAccountingStandard", "report.id.specialAccountingStandard.K"))
	out = append(out, g.choice("report.id.incomeStatementFormat", "report.id.incomeStatementFormat.incomeStatementFormat.GKV"))
	out = append(out, g.choice("report.id.consolidationRange", "report.id.consolidationRange.consolidationRange.EA"))
	out = append(out, g.sequence("report.id.accordingTo", nil))

	out = append(out, b.item(gcd+"report.period.balSheetClosingDate", g.closing))
	out = append(out, g.dateOrNil("report.period.balSheetClosingDatePreviousYear", in.PriorFiscalYearEnd))
	out = append(out, g.dateOrNil("report.period.fiscalPreviousYearBegin", in.PriorFiscalYearBegin))
	out = append(out, g.dateOrNil("report.period.fiscalPreviousYearEnd", in.PriorFiscalYearEnd))
	out = append(out, b.item(gcd+"report.period.fiscalYearBegin", g.begin))
	out = append(out, b.item(gcd+"report.period.fiscalYearEnd", g.end))
	if transmission := gcd + "report.id.reportElement.reportElements.transmissionNotYetPossible"; g.known(transmission) {
		out = append(out, b.nilItem(transmission))
	}

	s := in.Settings
	out = append(out, b.nilItem(gcd+"company.id.business"))
	out = append(out, g.choice("company.id.domesticBranchOfForeignCompany"))
	out = append(out, g.choice("company.id.entityWithTaxablePurposeBusiness"))
	out = append(out, g.choice("company.id.incomeClassification", "company.id.incomeClassification.trade"))
	out = append(out, g.choice("company.id.legalStatus", legalStatusOf(s.LegalForm)))
	out = append(out, g.textOrNil("company.id.location.street", s.Street))
	out = append(out, g.textOrNil("company.id.location.houseNo", s.HouseNumber))
	out = append(out, g.textOrNil("company.id.location.zipCode", s.PostalCode))
	out = append(out, g.textOrNil("company.id.location.city", s.City))
	out = append(out, b.item(gcd+"company.id.location.country", countryName(s.CountryCode)))
	out = append(out, g.textOrNil("company.id.name", s.FirmName()))

	idNo := map[string][]*node{
		"company.id.idNo.type.companyId.ST13": {b.item(gcd+"company.id.idNo.type.companyId.ST13", g.taxNumber)},
		"company.id.idNo.type.companyId.BF4":  {b.item(gcd+"company.id.idNo.type.companyId.BF4", firstFour(g.taxNumber))},
	}
	if vat := strings.TrimSpace(s.VatID); vat != "" {
		idNo["company.id.idNo.type.companyId.UID"] = []*node{b.item(gcd+"company.id.idNo.type.companyId.UID", vat)}
	}
	out = append(out, g.sequence("company.id.idNo", idNo))
	out = append(out, g.shareholders()...)
	if class := sizeClassOf(in.SizeClass); class != "" {
		out = append(out, g.choice("company.id.sizeClass", "company.id.sizeClass.sizeClass."+class))
	}
	return out
}

// choice baut ein Auswahltupel: die gewählten Ausprägungen mit Text, die
// übrigen Mussfeld-Ausprägungen mit NIL (Leitfaden A.10.1). Ohne Wahl steht
// die erste zulässige Ausprägung mit NIL.
func (g *generator) choice(tuple string, selected ...string) *node {
	b := g.b
	el, ok := b.tax.Element(gcd + tuple)
	if !ok || el.Tuple == nil || len(el.Tuple.Children) == 0 {
		return &node{name: b.tax.QName(gcd + tuple)}
	}
	head, _ := b.tax.Element(el.Tuple.Children[0].Ref)
	chosen := map[string]bool{}
	for _, s := range selected {
		chosen[gcd+s] = true
	}
	n := b.tuple(gcd + tuple)
	var first string
	for _, alt := range head.Alternatives {
		a, ok := b.tax.Element(alt)
		if !ok || !a.AllowedForCorporations() || !a.ValidAt(g.begin, g.closing) {
			continue
		}
		if first == "" {
			first = alt
		}
		switch {
		case chosen[alt]:
			n.children = append(n.children, b.item(alt, choiceText(a)))
		case a.Req == ReqMandatory && mandatoryAt(a, g.begin, g.closing):
			n.children = append(n.children, b.nilItem(alt))
		}
	}
	if len(n.children) == 0 && first != "" {
		n.children = append(n.children, b.nilItem(first))
	}
	return n
}

// sequence baut ein Tupel nach seinem Modell: gegebene Kinder, sonst NIL für
// Mussfelder; für Kapitalgesellschaften unzulässige Kinder entfallen
// (Leitfaden B.4.2).
func (g *generator) sequence(tuple string, given map[string][]*node) *node {
	b := g.b
	n := b.tuple(gcd + tuple)
	el, ok := b.tax.Element(gcd + tuple)
	if !ok || el.Tuple == nil {
		return n
	}
	for _, child := range el.Tuple.Children {
		short := strings.TrimPrefix(child.Ref, gcd)
		if nodes, ok := given[short]; ok {
			n.children = append(n.children, nodes...)
			continue
		}
		// Leer geht, was Mussfeld ist oder was das Schema im Tupel verlangt.
		c, ok := b.tax.Element(child.Ref)
		if !ok || !c.AllowedForCorporations() || c.Req != ReqMandatory && child.Min < 1 {
			continue
		}
		switch {
		case c.Tuple != nil && c.Tuple.Model == "choice":
			n.children = append(n.children, g.choice(short))
		case c.Tuple != nil:
			n.children = append(n.children, g.sequence(short, nil))
		default:
			n.children = append(n.children, b.nilItem(child.Ref))
		}
	}
	return n
}

// shareholders baut je Gesellschafter ein Tupel mit Name und
// Beteiligungsschlüssel aus dem Anteil am Stammkapital; ohne Liste eines mit
// NIL.
func (g *generator) shareholders() []*node {
	b := g.b
	list := g.in.Settings.Shareholders
	var total domain.Cents
	for _, sh := range list {
		total += sh.ShareCapital
	}
	if len(list) == 0 || total <= 0 {
		return []*node{g.sequence("company.id.shareholder", nil)}
	}
	out := make([]*node, 0, len(list))
	for i, sh := range list {
		num, den := reduce(int64(sh.ShareCapital), int64(total))
		given := map[string][]*node{
			"company.id.shareholder.name": {b.item(gcd+"company.id.shareholder.name", sh.Name)},
			"company.id.shareholder.id":   {b.item(gcd+"company.id.shareholder.id", fmt.Sprint(i+1))},
			"company.id.shareholder.ShareDivideKey": {b.tuple(gcd+"company.id.shareholder.ShareDivideKey",
				b.item(gcd+"company.id.shareholder.ShareDivideKey.numerator", fmt.Sprint(num)),
				b.item(gcd+"company.id.shareholder.ShareDivideKey.denominator", fmt.Sprint(den)))},
		}
		out = append(out, g.sequence("company.id.shareholder", given))
	}
	return out
}

func reduce(num, den int64) (int64, int64) {
	d := new(big.Int).GCD(nil, nil, big.NewInt(num), big.NewInt(den)).Int64()
	if d <= 0 {
		return num, den
	}
	return num / d, den / d
}

func (g *generator) known(name string) bool {
	el, ok := g.b.tax.Element(name)
	return ok && el.ValidAt(g.begin, g.closing)
}

func (g *generator) dateOrNil(name, date string) *node {
	if date == "" {
		return g.b.nilItem(gcd + name)
	}
	return g.b.item(gcd+name, date)
}

func (g *generator) textOrNil(name, value string) *node {
	if v := strings.TrimSpace(value); v != "" {
		return g.b.item(gcd+name, v)
	}
	return g.b.nilItem(gcd + name)
}

// choiceText ist der Klartext einer Ausprägung ohne die Bezeichnung der
// Gruppe („Berichtsbestandteile, Bilanz" → „Bilanz").
func choiceText(el *Element) string {
	label := el.Label
	if i := strings.Index(label, ", "); i >= 0 && strings.HasPrefix(label, "Berichtsbestandteile") {
		return label[i+2:]
	}
	return strings.TrimPrefix(label, "Rechtsform, ")
}

func legalStatusOf(form string) string {
	if form == domain.LegalFormUG {
		return "company.id.legalStatus.legalStatus.UG"
	}
	return "company.id.legalStatus.legalStatus.GMBH"
}

func sizeClassOf(kind domain.SizeClassKind) string {
	switch kind {
	case domain.SizeMicro:
		return "KstK"
	case domain.SizeSmall:
		return "KK"
	case domain.SizeMedium:
		return "MK"
	case domain.SizeLarge:
		return "GK"
	}
	return ""
}

func countryName(code string) string {
	if code == "" || strings.EqualFold(code, domain.DefaultCountryCode) {
		return "Deutschland"
	}
	return code
}

func firstFour(s string) string {
	if len(s) < 4 {
		return s
	}
	return s[:4]
}
