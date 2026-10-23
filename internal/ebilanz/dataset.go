package ebilanz

import (
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// dataset ist der GAAP-Teil der Instanz für den aktuellen Zeitraum: je
// Position ein Wert oder NIL.
type dataset struct {
	tax *Taxonomy
	// own sind die Werte, die aus Konten oder Ableitungen kommen; values
	// enthält zusätzlich die errechneten Oberpositionen.
	own    map[string]domain.Cents
	values map[string]domain.Cents
	nils   map[string]bool
	// conflicts sind Positionen mit eigenem Wert und Unterpositionen mit
	// Werten. Die Zuordnung vermeidet sie; tritt einer auf, ist das ein
	// Befund und kein Rechenfall.
	conflicts []string
}

func newDataset(tax *Taxonomy) *dataset {
	return &dataset{tax: tax, own: map[string]domain.Cents{}, values: map[string]domain.Cents{}, nils: map[string]bool{}}
}

func (d *dataset) add(name string, v domain.Cents) { d.own[name] += v }

func (d *dataset) set(name string, v domain.Cents) { d.own[name] = v }

// has meldet einen werthaltigen Wert (auch 0,00).
func (d *dataset) has(name string) bool {
	_, ok := d.values[name]
	return ok
}

func (d *dataset) value(name string) domain.Cents { return d.values[name] }

// reported meldet, ob die Position in der Instanz steht, mit Wert oder NIL.
func (d *dataset) reported(name string) bool { return d.has(name) || d.nils[name] }

// rollUp rechnet eine Rolle von den Blättern bis zur Wurzel hoch: jede
// Oberposition ist die gewichtete Summe ihrer Unterpositionen mit Wert
// (Leitfaden A.13, B.14.11).
func (d *dataset) rollUp(role, root string) (domain.Cents, bool) {
	return d.total(role, root, 0)
}

func (d *dataset) total(role, name string, depth int) (domain.Cents, bool) {
	if depth > 64 {
		return 0, false
	}
	own, hasOwn := d.own[name]
	var sum domain.Cents
	anyChild := false
	for _, arc := range d.tax.CalcChildren(role, name) {
		v, ok := d.total(role, arc.Child, depth+1)
		if !ok {
			continue
		}
		anyChild = true
		sum += domain.Cents(arc.Weight) * v
	}
	switch {
	case hasOwn && anyChild:
		d.conflicts = append(d.conflicts, name)
		d.values[name] = own + sum
		return own + sum, true
	case anyChild:
		d.values[name] = sum
		return sum, true
	case hasOwn:
		d.values[name] = own
		return own, true
	}
	return 0, false
}

// mandatory setzt NIL für jedes Muss- und Summenmussfeld einer Rolle, das ohne
// Wert bleibt (Leitfaden A.10, B.14.2.2, B.14.2.3).
func (d *dataset) mandatory(role, begin, closing string, tupleChildren map[string]bool) {
	for _, el := range d.tax.Elements {
		if el.NS != "de-gaap-ci" || !el.Monetary() || el.Abstract || tupleChildren[el.Name] {
			continue
		}
		if el.Req != ReqMandatory && el.Req != ReqSumMandatory {
			continue
		}
		if !el.InRole(role) || !el.AllowedForCorporations() || !el.ValidAt(begin, closing) {
			continue
		}
		if !mandatoryAt(el, begin, closing) {
			continue
		}
		if !d.has(el.Name) {
			d.nils[el.Name] = true
		}
	}
}

func mandatoryAt(el *Element, begin, closing string) bool {
	if el.FiscalValidSince != "" && begin < el.FiscalValidSince {
		return false
	}
	if el.FiscalValidThrough != "" && closing > el.FiscalValidThrough {
		return false
	}
	return true
}

// closePresentation sorgt dafür, dass zu jeder berichteten Position einer
// Rolle alle Oberpositionen der Darstellung mitkommen: mit Wert, wenn darunter
// etwas werthaltig ist, sonst mit NIL (Leitfaden B.14.10).
func (d *dataset) closePresentation(role string) {
	names := make([]string, 0, len(d.values)+len(d.nils))
	for name := range d.values {
		names = append(names, name)
	}
	for name := range d.nils {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		el, ok := d.tax.Element(name)
		if !ok || !el.InRole(role) {
			continue
		}
		valued := d.has(name)
		for cur, guard := el, 0; guard < 64; guard++ {
			parentName, ok := cur.PresParent(role)
			if !ok {
				break
			}
			parent, ok := d.tax.Element(parentName)
			if !ok {
				break
			}
			if !parent.Abstract && parent.Monetary() {
				switch {
				case valued && !d.has(parentName):
					delete(d.nils, parentName)
					d.values[parentName] = d.calcSum(role, parentName)
				case !d.reported(parentName):
					d.nils[parentName] = true
				}
			}
			cur = parent
		}
	}
}

// calcSum ist die gewichtete Summe der werthaltigen Unterpositionen.
func (d *dataset) calcSum(role, name string) domain.Cents {
	var sum domain.Cents
	for _, arc := range d.tax.CalcChildren(role, name) {
		if v, ok := d.values[arc.Child]; ok {
			sum += domain.Cents(arc.Weight) * v
		}
	}
	return sum
}

// inconsistencies prüft jede Rechenregel einer Rolle auf den Cent: der Wert
// der Oberposition gleich der gewichteten Summe der werthaltigen
// Unterpositionen; NIL zählt nicht (Leitfaden B.14.11).
func (d *dataset) inconsistencies(role string) []calcGap {
	var out []calcGap
	parents := make([]string, 0)
	for name := range d.values {
		parents = append(parents, name)
	}
	sort.Strings(parents)
	for _, name := range parents {
		children := d.tax.CalcChildren(role, name)
		any := false
		var sum domain.Cents
		for _, arc := range children {
			if v, ok := d.values[arc.Child]; ok {
				any = true
				sum += domain.Cents(arc.Weight) * v
			}
		}
		if any && sum != d.values[name] {
			out = append(out, calcGap{Element: name, Reported: d.values[name], Computed: sum})
		}
	}
	return out
}

type calcGap struct {
	Element  string
	Reported domain.Cents
	Computed domain.Cents
}
