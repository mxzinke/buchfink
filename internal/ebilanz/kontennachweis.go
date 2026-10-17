package ebilanz

import (
	"sort"
	"strings"
)

// Kontennachweis (§ 5b Abs. 1 EStG, Leitfaden B.12): je Konto ein Tupel mit
// Position, Kontonummer, Bezeichnung und Saldo in Richtung der Position.

const (
	ksTuple       = "detailedInformation.accountBalances"
	ksPosition    = "detailedInformation.accountbalances.positionName"
	ksNumber      = "detailedInformation.accountbalances.accountNumber"
	ksDescription = "detailedInformation.accountbalances.accountDescription"
	ksAmount      = "detailedInformation.accountbalances.amount"
	ksNotPossible = "detailedInformation.accountbalances.notPossible"
)

// accountTuples baut die Tupel des Kontennachweises für die Konten mit
// Position. Ab Fassung 6.10 bekommt zusätzlich jede werthaltige rechnerisch
// verknüpfte Position ohne werthaltige Unterposition und ohne Konto ein
// Tupel ohne Konteninformationen (Leitfaden 6.10 B.12.1) — das betrifft die
// abgeleiteten Positionen wie den Jahresüberschuss der Bilanz.
func (g *generator) accountTuples(roles []string) []*node {
	b := g.b
	_, withFlag := b.tax.Element(ksNotPossible)
	var out []*node
	covered := map[string]bool{}
	placements := append([]Placement(nil), g.placements...)
	sort.SliceStable(placements, func(i, j int) bool { return placements[i].Account < placements[j].Account })
	for _, p := range placements {
		if p.Element == "" || p.Finding != "" || !g.hb.has(p.Element) {
			continue
		}
		description := strings.TrimSpace(p.Name)
		if description == "" {
			description = p.Account
		}
		children := []*node{
			b.item(ksPosition, b.tax.QName(p.Element)),
			b.item(ksNumber, p.Account),
			b.item(ksDescription, description),
			b.money(ksAmount, p.Value),
		}
		if withFlag {
			children = append(children, b.item(ksNotPossible, "false"))
		}
		out = append(out, b.tuple(ksTuple, children...))
		covered[p.Element] = true
	}
	if !withFlag {
		return out
	}
	for _, name := range g.hb.sortedValues() {
		if covered[name] || !g.calcLeaf(name, roles) {
			continue
		}
		out = append(out, b.tuple(ksTuple,
			b.item(ksPosition, b.tax.QName(name)),
			b.nilItem(ksNumber),
			b.nilItem(ksDescription),
			b.money(ksAmount, g.hb.value(name)),
			b.item(ksNotPossible, "true"),
		))
	}
	return out
}

// calcLeaf meldet eine werthaltige, rechnerisch verknüpfte Position einer der
// Rollen, unter der keine Unterposition werthaltig ist.
func (g *generator) calcLeaf(name string, roles []string) bool {
	el, ok := g.b.tax.Element(name)
	if !ok || el.NS != "de-gaap-ci" {
		return false
	}
	for _, role := range roles {
		if !el.InRole(role) {
			continue
		}
		_, _, linked := el.CalcParent(role)
		children := g.b.tax.CalcChildren(role, name)
		if !linked && len(children) == 0 {
			continue
		}
		valuedChild := false
		for _, arc := range children {
			if g.hb.has(arc.Child) {
				valuedChild = true
				break
			}
		}
		if !valuedChild {
			return true
		}
	}
	return false
}

func (d *dataset) sortedValues() []string {
	out := make([]string, 0, len(d.values))
	for name := range d.values {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
