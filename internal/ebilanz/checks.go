package ebilanz

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Severity unterscheidet Befunde, die den Export verhindern, von Hinweisen.
type Severity string

const (
	SeverityBlocking Severity = "blocking"
	SeverityWarning  Severity = "warning"
)

// Finding ist ein Befund der Prüfung vor dem Export.
type Finding struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Element  string   `json:"element,omitempty"`
	Accounts []string `json:"accounts,omitempty"`
}

func (g *generator) block(code, message, element string, accounts ...string) {
	g.findings = append(g.findings, Finding{Severity: SeverityBlocking, Code: code, Message: message, Element: element, Accounts: accounts})
}

func (g *generator) warn(code, message, element string) {
	g.findings = append(g.findings, Finding{Severity: SeverityWarning, Code: code, Message: message, Element: element})
}

// check prüft den übermittelten und den steuerlichen Datensatz.
func (g *generator) check(roles []string) {
	for _, p := range g.placements {
		if p.Finding != "" {
			g.block("konto", p.Finding, "", p.Account)
		}
	}
	for _, name := range g.hb.conflicts {
		g.block("doppelt", fmt.Sprintf("Die Position %s trägt Konten und zugleich Unterpositionen mit Konten.", name),
			name, g.accountsOf(name)...)
	}

	// Rechenregeln auf den Cent (Leitfaden B.14.11), bei einer Handelsbilanz
	// auch für den übergeleiteten Datensatz.
	sets := []*dataset{g.hb}
	if g.commercial {
		sets = append(sets, g.st)
	}
	for i, ds := range sets {
		which := "Instanz"
		if i == 1 {
			which = "übergeleiteter Steuerbilanz"
		}
		for _, role := range roles {
			for _, gap := range ds.inconsistencies(role) {
				g.block("rechenregel", fmt.Sprintf("Rechenregel in der %s verletzt: %s hat %s €, die Unterpositionen ergeben %s €.",
					which, gap.Element, gap.Reported, gap.Computed), gap.Element)
			}
		}
	}

	// Summe Aktiva = Summe Passiva (Leitfaden B.14.3.2.1, B.14.3.2.2).
	for _, ds := range sets {
		if !ds.has("bs.ass") {
			g.block("bilanzsumme", "Die Bilanzsumme der Aktivseite fehlt.", "bs.ass")
			break
		}
		if a, p := ds.value("bs.ass"), ds.value("bs.eqLiab"); a != p {
			g.block("bilanz", fmt.Sprintf("Summe Aktiva %s € und Summe Passiva %s € weichen voneinander ab.", a, p), "bs.eqLiab")
		}
	}

	if g.in.Kind == KindAnnual {
		g.checkNetIncome()
	}
	g.checkMandatory(roles)
	g.checkPermitted()
	g.checkSumMandatory(roles)
	g.checkStatement()
	g.checkMasterData()
	g.checkMasterDataComplete()
	sort.SliceStable(g.findings, func(i, j int) bool {
		return g.findings[i].Severity == SeverityBlocking && g.findings[j].Severity != SeverityBlocking
	})
}

// checkNetIncome: Jahresüberschuss der Bilanz bzw. Ergebnisverwendung gleich
// dem der GuV (Leitfaden B.14.3.2.3.4) und der BVV-Gewinn gleich dem
// Jahresüberschuss (B.14.3.2.12).
func (g *generator) checkNetIncome() {
	for _, ds := range []*dataset{g.hb, g.st} {
		netIncome := ds.value(g.incomeRoot)
		diff := ds.value(taxBalanceTransfer)
		if g.appropriated {
			if a, b := ds.value("bs.eqLiab.equity.profitLoss"), ds.value("incomeUse.gainLoss"); a != b {
				g.block("bilanzgewinn", fmt.Sprintf("Bilanzgewinn der Bilanz (%s €) und der Ergebnisverwendung (%s €) weichen ab.", a, b),
					"bs.eqLiab.equity.profitLoss")
			}
			if a := ds.value("incomeUse.gainLoss.netIncome") + diff; a != netIncome {
				g.block("jahresueberschuss", fmt.Sprintf("Jahresüberschuss der Ergebnisverwendung (%s €) und der GuV (%s €) weichen ab.", a, netIncome),
					"incomeUse.gainLoss.netIncome")
			}
		} else if a := ds.value("bs.eqLiab.equity.netIncome") + diff; a != netIncome {
			g.block("jahresueberschuss", fmt.Sprintf("Jahresüberschuss der Bilanz (%s €) und der GuV (%s €) weichen ab.", a, netIncome),
				"bs.eqLiab.equity.netIncome")
		}
		if !g.commercial {
			break
		}
	}
	if g.hb.has(bvvProfit) {
		if a, b := g.hb.value(bvvProfit), g.st.value(g.incomeRoot); a != b {
			g.block("bvv", fmt.Sprintf("Gewinn nach Betriebsvermögensvergleich (%s €) und Jahresüberschuss (%s €) weichen ab.", a, b), bvvProfit)
		}
		equity := g.st.value(equityTotal) - g.st.value(equityDeficit)
		if a := g.hb.value(bvvCurrent); a != equity {
			g.block("bvv", fmt.Sprintf("Betriebsvermögen zum Ende des Jahres (%s €) und Eigenkapital (%s €) weichen ab.", a, equity), bvvCurrent)
		}
	}
}

// checkMandatory: jedes Mussfeld der angekündigten Berichtsteile steht in der
// Instanz, mit Wert oder NIL (Leitfaden B.14.2.2).
func (g *generator) checkMandatory(roles []string) {
	for _, el := range g.tax.Elements {
		if el.NS != "de-gaap-ci" || el.Req != ReqMandatory || !el.Monetary() || g.tupleChildren[el.Name] {
			continue
		}
		if !el.AllowedForCorporations() || !el.ValidAt(g.begin, g.closing) || !mandatoryAt(el, g.begin, g.closing) {
			continue
		}
		for _, role := range roles {
			if el.InRole(role) && !g.hb.reported(el.Name) {
				g.block("mussfeld", "Mussfeld fehlt: "+el.Label+".", el.Name)
				break
			}
		}
	}
}

// checkPermitted: Positionen, die die Taxonomie für den Bilanzierungsstandard
// ausschließt, dürfen keinen Wert tragen (Leitfaden B.15.2, B.16.2); für die
// Rechtsform unzulässige Positionen gar nicht vorkommen (B.4.2); jede Position
// muss im Wirtschaftsjahr gelten (B.14.4.1).
func (g *generator) checkPermitted() {
	for _, name := range g.hb.sortedValues() {
		el, ok := g.tax.Element(name)
		if !ok {
			g.block("unbekannt", "Position "+name+" kennt die Taxonomie "+g.tax.Version+" nicht.", name)
			continue
		}
		if !el.AllowedForCorporations() {
			g.block("rechtsform", "Position für Kapitalgesellschaften unzulässig: "+el.Label+".", name, g.accountsOf(name)...)
		}
		if !el.ValidAt(g.begin, g.closing) {
			g.block("gueltigkeit", "Position im Wirtschaftsjahr nicht gültig: "+el.Label+".", name, g.accountsOf(name)...)
		}
		switch {
		case !g.commercial && el.NotPermittedInTaxBalance():
			g.block("steuerlich-unzulaessig", "In der Steuer- bzw. Einheitsbilanz unzulässig (Leitfaden B.16.2): "+el.Label+
				". Handels- und Steuerbilanz weichen hier ab; dafür braucht es eine Überleitungsrechnung.", name, g.accountsOf(name)...)
		case g.commercial && el.NotPermittedInCommercialBalance():
			g.block("handelsrechtlich-unzulaessig", "In der Handelsbilanz unzulässig: "+el.Label+".", name, g.accountsOf(name)...)
		}
	}
	if !g.commercial {
		return
	}
	// In der übergeleiteten Steuerbilanz müssen steuerlich unzulässige
	// Bilanzpositionen auf null stehen (Leitfaden B.15.2).
	for _, name := range g.st.sortedValues() {
		el, ok := g.tax.Element(name)
		if !ok || !el.NotPermittedInTaxBalance() || !el.InRole(RoleBalanceSheet) || g.st.value(name) == 0 {
			continue
		}
		g.block("steuerlich-unzulaessig", fmt.Sprintf("Nach der Überleitung steht %s noch mit %s €; steuerlich ist die Position "+
			"unzulässig (Leitfaden B.15.2).", el.Label, g.st.value(name)), name, g.accountsOf(name)...)
	}
}

// checkSumMandatory: ein werthaltiges Summenmussfeld ungleich 0 braucht eine
// werthaltige Unterposition; ausgenommen das Eigenkapital (Leitfaden B.14.14).
func (g *generator) checkSumMandatory(roles []string) {
	for _, name := range g.hb.sortedValues() {
		el, ok := g.tax.Element(name)
		if !ok || el.Req != ReqSumMandatory || g.hb.value(name) == 0 {
			continue
		}
		if g.tax.within(RoleBalanceSheet, name, []string{equityTotal}) && !strings.HasPrefix(name, taxBalance) {
			continue
		}
		for _, role := range roles {
			if !el.InRole(role) || len(g.tax.CalcChildren(role, name)) == 0 {
				continue
			}
			if g.hb.calcSumHas(role, name) {
				continue
			}
			g.block("summenmussfeld", "Summenmussfeld ohne werthaltige Unterposition: "+el.Label+".", name)
		}
	}
}

func (d *dataset) calcSumHas(role, name string) bool {
	for _, arc := range d.tax.CalcChildren(role, name) {
		if d.has(arc.Child) {
			return true
		}
	}
	return false
}

// checkStatement gleicht Jahresergebnis und Eigenkapital mit der Bilanz ab,
// die Buchfink aus denselben Konten aufstellt.
func (g *generator) checkStatement() {
	stmt := g.in.Statement
	if stmt == nil {
		return
	}
	if g.in.Kind == KindAnnual {
		if a, b := g.hb.value(g.incomeRoot), stmt.NetIncome; a != b {
			g.block("abgleich-ergebnis", fmt.Sprintf("Jahresergebnis der E-Bilanz (%s €) und der Gewinn- und Verlustrechnung (%s €) weichen ab.", a, b), g.incomeRoot)
		}
	}
	if line := stmt.Line("passiva.A"); line != nil {
		if a, b := g.hb.value(equityTotal), line.Amount; a != b {
			g.block("abgleich-eigenkapital", fmt.Sprintf("Eigenkapital der E-Bilanz (%s €) und der Bilanz (%s €) weichen ab.", a, b), equityTotal)
		}
	}
}

// checkMasterData prüft die Stammdaten, die die Instanz braucht
// (Leitfaden B.14.3.1).
func (g *generator) checkMasterData() {
	s := g.in.Settings
	if len(g.taxNumber) != 13 {
		g.block("steuernummer", "Die 13-stellige Steuernummer (ELSTER-Format) fehlt in den Stammdaten; sie ist Kennung "+
			"und Pflichtangabe der E-Bilanz.", "genInfo.company.id.idNo.type.companyId.ST13")
	}
	if strings.TrimSpace(s.FirmName()) == "" {
		g.block("stammdaten", "Der Name des Unternehmens fehlt in den Stammdaten.", "genInfo.company.id.name")
	}
	if strings.TrimSpace(s.Street) == "" || strings.TrimSpace(s.PostalCode) == "" || strings.TrimSpace(s.City) == "" {
		g.block("stammdaten", "Die Anschrift (Straße, Postleitzahl, Ort) fehlt in den Stammdaten.", "genInfo.company.id.location.street")
	}
	if s.LegalForm != domain.LegalFormGmbH && s.LegalForm != domain.LegalFormUG {
		g.block("rechtsform", "Die E-Bilanz kennt nur GmbH und UG (haftungsbeschränkt) als Rechtsform.", "genInfo.company.id.legalStatus")
	}
	if g.in.MicroBilG && g.in.SizeClass != domain.SizeMicro {
		g.block("microbilg", "Die GuV nach MicroBilG ist nur für Kleinstkapitalgesellschaften zulässig (§ 267a HGB, § 275 Abs. 5 HGB).",
			"genInfo.report.id.reportElement.reportElements.GuVMicroBilG")
	}
}

// checkMasterDataComplete: jedes GCD-Mussfeld, das für Kapitalgesellschaften
// zulässig ist, steht in den Stammdaten (Leitfaden B.14.2.1).
func (g *generator) checkMasterDataComplete() {
	present := map[string]bool{}
	var walk func(nodes []*node)
	walk = func(nodes []*node) {
		for _, n := range nodes {
			present[strings.TrimPrefix(n.name, "de-gcd:")] = true
			walk(n.children)
		}
	}
	walk(g.gcdNodes(g.announced()))
	for _, el := range g.tax.Elements {
		if el.NS != "de-gcd" || el.Req != ReqMandatory || el.ChoiceHead != "" || g.tupleChildren[el.Name] {
			continue
		}
		if !el.AllowedForCorporations() || !el.ValidAt(g.begin, g.closing) || !mandatoryAt(el, g.begin, g.closing) {
			continue
		}
		if !present[el.Name] {
			g.block("mussfeld", "Mussfeld der Stammdaten fehlt: "+el.Label+".", el.Name)
		}
	}
}

// accountsOf nennt die Konten unter einer Position und ihren Unterpositionen.
func (g *generator) accountsOf(name string) []string {
	var out []string
	for _, p := range g.placements {
		if p.Element == "" {
			continue
		}
		if p.Element == name || g.tax.within(p.role, p.Element, []string{name}) {
			out = append(out, p.Account)
		}
	}
	return out
}
