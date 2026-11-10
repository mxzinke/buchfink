package ebilanz

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kind ist die Bilanzart (genInfo.report.id.statementType).
type Kind string

const (
	// KindAnnual ist der Jahresabschluss.
	KindAnnual Kind = "Jahresabschluss"
	// KindOpening ist die Eröffnungsbilanz (§ 242 Abs. 1 HGB): Bilanz ohne
	// GuV auf den Tag, an dem das Handelsgewerbe beginnt.
	KindOpening Kind = "Eröffnungsbilanz"
)

// Input ist alles, was in die E-Bilanz eingeht.
type Input struct {
	Settings *domain.CompanySettings
	// Statement ist die Gliederung desselben Stichtags. Aus ihr folgen der
	// Ausweis des Bilanzgewinns (§ 268 Abs. 1 HGB) und die Abgleichswerte.
	Statement *domain.Statement
	// Accounts sind die Konten mit den Salden am Stichtag.
	Accounts []domain.Account
	Kind     Kind
	// FiscalYearBegin und FiscalYearEnd sind Beginn und Ende des
	// Wirtschaftsjahres (JJJJ-MM-TT). ClosingDate ist der Bilanzstichtag;
	// leer heißt Ende des Wirtschaftsjahres, bei der Eröffnungsbilanz ist er
	// der erste Tag.
	FiscalYearBegin, FiscalYearEnd, ClosingDate string
	// PriorFiscalYearBegin und PriorFiscalYearEnd sind leer, wenn es kein
	// Vorjahr gibt.
	PriorFiscalYearBegin, PriorFiscalYearEnd string
	// Version ist die gewünschte Taxonomie; leer heißt Regelfassung.
	Version string
	// MicroBilG wählt die verkürzte GuV der Kleinstkapitalgesellschaft.
	MicroBilG bool
	SizeClass domain.SizeClassKind
	// Adjustments sind die Abweichungen der Steuerbilanz. Ohne sie geht die
	// Bilanz als Einheitsbilanz, mit ihnen als Handelsbilanz mit
	// Überleitungsrechnung.
	Adjustments []Adjustment
	// PriorEquity und PriorPaidIn sind steuerliches Eigenkapital und
	// eingezahltes Kapital (gezeichnetes Kapital und Kapitalrücklage) am
	// Ende des Vorjahres; nil heißt unbekannt, null im Gründungsjahr.
	PriorEquity *domain.Cents
	PriorPaidIn *domain.Cents
}

// Result ist die erzeugte E-Bilanz mit ihrem Prüfbericht. Instance ist leer,
// solange blockierende Befunde bestehen.
type Result struct {
	Version  VersionChoice
	Instance []byte
	Report   *MappingReport
}

// generator hält den Zustand einer Erzeugung.
type generator struct {
	in            Input
	tax           *Taxonomy
	version       VersionChoice
	b             *instanceBuilder
	placer        placement
	begin, end    string
	closing       string
	appropriated  bool
	commercial    bool
	microBilG     bool
	incomeRole    string
	incomeRoot    string
	taxNumber     string
	placements    []Placement
	hb, st        *dataset
	transfers     []transfer
	findings      []Finding
	tupleChildren map[string]bool
}

// Generate prüft die Eingaben gegen die Taxonomie und erzeugt die Instanz.
// Ein Fehler heißt, dass sich gar nicht erst prüfen ließ; Befunde stehen im
// Bericht.
func Generate(in Input) (*Result, error) {
	g, err := newGenerator(in)
	if err != nil {
		return nil, err
	}
	g.compute()
	roles := g.roles()
	g.check(roles)
	res := &Result{Version: g.version, Report: g.report()}
	if !res.Report.CanExport {
		return res, nil
	}
	instance, err := g.instance(roles)
	if err != nil {
		return nil, err
	}
	res.Instance = instance
	return res, nil
}

func newGenerator(in Input) (*generator, error) {
	if in.Settings == nil {
		return nil, fmt.Errorf("die Unternehmensdaten fehlen")
	}
	if in.Kind == "" {
		in.Kind = KindAnnual
	}
	if len(in.FiscalYearBegin) != 10 || len(in.FiscalYearEnd) != 10 {
		return nil, fmt.Errorf("Beginn und Ende des Wirtschaftsjahres fehlen (erwartet JJJJ-MM-TT)")
	}
	g := &generator{in: in, begin: in.FiscalYearBegin, end: in.FiscalYearEnd, closing: in.ClosingDate}
	switch {
	case in.Kind == KindOpening && len(g.closing) != 10:
		return nil, fmt.Errorf("der Stichtag der Eröffnungsbilanz fehlt (erwartet JJJJ-MM-TT)")
	case in.Kind == KindOpening:
		// Leitfaden B.14.3.1.6: Beginn des Wirtschaftsjahres gleich Stichtag.
		g.begin = g.closing
	case g.closing == "":
		g.closing = g.end
	}
	choice, err := ChooseVersion(g.begin, g.end, in.Version)
	if err != nil {
		return nil, err
	}
	tax, err := LoadTaxonomy(choice.Version)
	if err != nil {
		return nil, err
	}
	g.version, g.tax = choice, tax
	g.appropriated = in.Kind == KindAnnual && in.Statement != nil && in.Statement.Line("passiva.A.bilanzgewinn") != nil
	g.microBilG = in.MicroBilG && in.Kind == KindAnnual
	g.commercial = in.Kind == KindAnnual && hasAdjustments(in.Adjustments)
	g.incomeRole, g.incomeRoot = RoleIncomeStatement, "is.netIncome"
	if g.microBilG {
		g.incomeRole, g.incomeRoot = RoleIncomeMicroBilG, "ismi.netIncome"
	}
	g.taxNumber = digits(in.Settings.TaxNumberElster)
	g.placer = placement{tax: tax, appropriated: g.appropriated, microBilG: g.microBilG, begin: g.begin, closing: g.closing}
	g.tupleChildren = map[string]bool{}
	for _, el := range tax.Elements {
		if el.Tuple != nil {
			for _, c := range el.Tuple.Children {
				g.tupleChildren[c.Ref] = true
			}
		}
	}
	g.b = &instanceBuilder{tax: tax, microBilG: g.microBilG, identifier: g.taxNumber,
		begin: g.begin, end: g.end, closing: g.closing}
	return g, nil
}

func hasAdjustments(list []Adjustment) bool {
	for _, a := range list {
		if a.Current != 0 || a.Prior != 0 {
			return true
		}
	}
	return false
}

// compute ordnet die Konten zu und rechnet beide Datensätze.
func (g *generator) compute() {
	g.placements = g.placer.place(g.in.Accounts)
	g.hb = newDataset(g.tax)
	for i := range g.placements {
		p := &g.placements[i]
		if p.Element == "" || p.Finding != "" {
			continue
		}
		if g.in.Kind == KindOpening && p.role != RoleBalanceSheet {
			p.Finding = "Die Eröffnungsbilanz hat keine Gewinn- und Verlustrechnung; das Konto darf am Stichtag keinen Saldo tragen."
			continue
		}
		g.hb.add(p.Element, p.Value)
	}

	if g.in.Kind == KindAnnual {
		if _, ok := g.hb.rollUp(g.incomeRole, g.incomeRoot); !ok {
			g.hb.set(g.incomeRoot, 0)
			g.hb.rollUp(g.incomeRole, g.incomeRoot)
		}
		netIncome := g.hb.value(g.incomeRoot)
		if g.appropriated {
			g.hb.set("incomeUse.gainLoss.netIncome", netIncome)
			profit, _ := g.hb.rollUp(RoleAppropriation, "incomeUse.gainLoss")
			g.hb.set("bs.eqLiab.equity.profitLoss", profit)
		} else {
			g.hb.set("bs.eqLiab.equity.netIncome", netIncome)
		}
	}
	g.hb.rollUp(RoleBalanceSheet, "bs.ass")
	g.hb.rollUp(RoleBalanceSheet, "bs.eqLiab")

	if g.commercial {
		g.transfers = g.buildTransfers()
	}
	g.st = g.applyTransfers(g.hb)
	if g.in.Kind == KindAnnual {
		g.assetComparison(g.st, g.st.value(g.incomeRoot))
	}
	for _, role := range g.roles() {
		g.hb.mandatory(role, g.begin, g.closing, g.tupleChildren)
	}
	for _, role := range g.roles() {
		g.hb.closePresentation(role)
	}
	g.st = g.applyTransfers(g.hb)
}

// roles sind die Rollen der angekündigten Berichtsteile mit Werten.
func (g *generator) roles() []string {
	roles := []string{RoleBalanceSheet}
	if g.in.Kind == KindAnnual {
		roles = append(roles, g.incomeRole, RoleTaxAssets)
		if g.appropriated {
			roles = append(roles, RoleAppropriation)
		}
	}
	return roles
}

func (g *generator) announced() reportElements {
	annual := g.in.Kind == KindAnnual
	return reportElements{
		balance:         true,
		income:          annual && !g.microBilG,
		incomeMicroBilG: annual && g.microBilG,
		appropriation:   g.appropriated,
		accounts:        true,
		assetComparison: annual,
		transfer:        g.commercial,
	}
}

// instance schreibt die Instanz.
func (g *generator) instance(roles []string) ([]byte, error) {
	b := g.b
	b.gcd = g.gcdNodes(g.announced())
	for _, name := range g.hb.sortedValues() {
		if el, ok := g.tax.Element(name); ok && el.NS == "de-gaap-ci" {
			b.facts = append(b.facts, b.money(name, g.hb.value(name)))
			b.usesInstant = b.usesInstant || el.Period == "instant"
		}
	}
	nils := make([]string, 0, len(g.hb.nils))
	for name := range g.hb.nils {
		nils = append(nils, name)
	}
	sortStrings(nils)
	for _, name := range nils {
		if el, ok := g.tax.Element(name); ok {
			b.facts = append(b.facts, b.nilItem(name))
			b.usesInstant = b.usesInstant || el.Period == "instant"
		}
	}
	b.tuples = append(b.tuples, g.accountTuples([]string{RoleBalanceSheet, g.incomeRole})...)
	b.tuples = append(b.tuples, g.transferNodes()...)
	return b.write()
}

func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}
