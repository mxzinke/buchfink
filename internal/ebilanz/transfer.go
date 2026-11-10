package ebilanz

import (
	"sort"
	"strings"
	"sync"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Adjustment ist eine Abweichung des steuerlichen vom handelsrechtlichen
// Wertansatz eines Bestandskontos, die die Überleitungsrechnung zeigt
// (§ 5b Abs. 1 Satz 2 EStG, § 60 Abs. 2 EStDV).
type Adjustment struct {
	// Reason nennt Anlass und Vorschrift, etwa „Sonderabschreibung § 7g Abs. 5 EStG".
	Reason string
	// Account ist das Bestandskonto, dessen Wertansatz abweicht.
	Account string
	// IncomeAccount ist das Erfolgskonto, über das die Abweichung im
	// Geschäftsjahr wirkt; leer heißt: Sammelposten für Gewinnänderungen.
	IncomeAccount string
	// Current und Prior sind steuerlich minus handelsrechtlich als Sollsaldo
	// am Stichtag und am Vorjahresstichtag: positiv, wo das steuerliche
	// Vermögen höher ist.
	Current domain.Cents
	Prior   domain.Cents
}

// Positionen des steuerlichen Ausgleichspostens (Leitfaden B.14.3.2.3.4).
const (
	taxBalance         = "bs.eqLiab.equity.netIncome.taxBalanceGenerally"
	taxBalanceFinalPre = taxBalance + ".finalPrev"
	taxBalanceTransfer = taxBalance + ".transferDiffTaxAccounts"
	collectiveProfit   = "is.netIncome.collItemChangeProfitHbst"
	collectiveMicro    = "ismi.netIncome.collItemChangeProfitHbst"
)

// change ist die Wertänderung einer Position aus der Überleitung.
type change struct{ actual, previous domain.Cents }

// transfer ist ein Überleitungstupel (hbst.transfer).
type transfer struct {
	reason  string
	balance map[string]change
	income  map[string]domain.Cents
}

// buildTransfers leitet aus den Abweichungen je ein Tupel ab. Jede Änderung
// steht an der Position selbst und an allen rechnerischen Oberpositionen, weil
// die Überleitung nur die genannte Position ändert (Leitfaden B.15.1).
func (g *generator) buildTransfers() []transfer {
	var out []transfer
	for _, adj := range g.in.Adjustments {
		if adj.Current == 0 && adj.Prior == 0 {
			continue
		}
		t := transfer{reason: adj.Reason, balance: map[string]change{}, income: map[string]domain.Cents{}}
		element := g.elementOf(adj.Account, adj.Current)
		if element == "" {
			g.block("ueberleitung-konto", "Für das Konto "+adj.Account+" der Überleitung ("+adj.Reason+
				") gibt es keine Position der Bilanz.", "", adj.Account)
			continue
		}
		actual, previous := adj.Current-adj.Prior, adj.Prior
		g.addChain(t.balance, RoleBalanceSheet, element, g.directed(element, actual), g.directed(element, previous))
		// Das Eigenkapital ändert sich um dieselben Beträge; den Unterschied
		// trägt der steuerliche Ausgleichsposten (Leitfaden B.14.3.2.3.4).
		g.addChain(t.balance, RoleBalanceSheet, taxBalanceTransfer, actual, 0)
		g.addChain(t.balance, RoleBalanceSheet, taxBalanceFinalPre, 0, previous)
		if actual != 0 {
			income := g.collective()
			if adj.IncomeAccount != "" {
				if el := g.elementOf(adj.IncomeAccount, -actual); el != "" {
					income = el
				}
			}
			chain := map[string]change{}
			g.addChain(chain, g.incomeRole, income, g.directed(income, -actual), 0)
			for name, c := range chain {
				t.income[name] += c.actual
			}
		}
		for name, c := range t.balance {
			if c.actual == 0 && c.previous == 0 {
				delete(t.balance, name)
			}
		}
		for name, v := range t.income {
			if v == 0 {
				delete(t.income, name)
			}
		}
		out = append(out, t)
	}
	return out
}

// addChain trägt eine Änderung an einer Position und, mit den Rechengewichten,
// an allen ihren Oberpositionen ein.
func (g *generator) addChain(into map[string]change, role, name string, actual, previous domain.Cents) {
	for cur, guard := name, 0; cur != "" && guard < 64; guard++ {
		c := into[cur]
		c.actual += actual
		c.previous += previous
		into[cur] = c
		el, ok := g.tax.Element(cur)
		if !ok {
			return
		}
		parent, weight, ok := el.CalcParent(role)
		if !ok {
			return
		}
		actual *= domain.Cents(weight)
		previous *= domain.Cents(weight)
		cur = parent
	}
}

func (g *generator) collective() string {
	if g.microBilG {
		return collectiveMicro
	}
	return collectiveProfit
}

func (g *generator) directed(name string, debit domain.Cents) domain.Cents {
	return placement{tax: g.tax}.directed(name, debit)
}

// elementOf ist die Position eines Kontos: die seines Saldos, sonst die, auf
// die es mit einem Saldo in Richtung der Abweichung ginge.
func (g *generator) elementOf(account string, debit domain.Cents) string {
	for _, p := range g.placements {
		if p.Account == account && p.Element != "" {
			return p.Element
		}
	}
	for _, acc := range g.in.Accounts {
		if acc.Number == account {
			return g.placer.one(withSign(acc, debit)).Element
		}
	}
	if acc, ok := catalogAccount(account); ok {
		return g.placer.one(withSign(acc, debit)).Element
	}
	return ""
}

var (
	catalogOnce sync.Once
	catalog     map[string]domain.Account
)

// catalogAccount sucht ein Konto im SKR04-Katalog, für Konten der Überleitung
// ohne Saldo in den Büchern.
func catalogAccount(number string) (domain.Account, bool) {
	catalogOnce.Do(func() {
		catalog = map[string]domain.Account{}
		for _, acc := range accounting.DefaultSKR04Accounts() {
			catalog[acc.Number] = acc
		}
	})
	acc, ok := catalog[number]
	return acc, ok
}

func withSign(acc domain.Account, debit domain.Cents) domain.Account {
	if debit == 0 {
		debit = 1
	}
	if debit > 0 {
		acc.DebitSum, acc.CreditSum = debit, 0
	} else {
		acc.DebitSum, acc.CreditSum = 0, -debit
	}
	return acc
}

// applyTransfers ergibt den steuerlichen Datensatz: die gemeldeten Werte plus
// die Änderungen der Überleitung (Leitfaden B.15.1.1 bis B.15.1.5).
func (g *generator) applyTransfers(hb *dataset) *dataset {
	st := newDataset(g.tax)
	for name, v := range hb.values {
		st.values[name] = v
	}
	for name := range hb.nils {
		st.nils[name] = true
	}
	for _, t := range g.transfers {
		for name, c := range t.balance {
			st.values[name] += c.actual + c.previous
			delete(st.nils, name)
		}
		for name, v := range t.income {
			st.values[name] += v
			delete(st.nils, name)
		}
	}
	return st
}

// transferNodes baut die Tupel hbst.transfer.
func (g *generator) transferNodes() []*node {
	b := g.b
	var out []*node
	for _, t := range g.transfers {
		n := b.tuple("hbst.transfer", b.tuple("hbst.transfer.kind",
			b.item("hbst.transfer.kind.changeValue", "Wertansatz")))
		names := make([]string, 0, len(t.balance))
		for name := range t.balance {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, side := range []string{"bs.ass", "bs.eqLiab"} {
			tuple := "hbst.transfer.bsAss"
			if side == "bs.eqLiab" {
				tuple = "hbst.transfer.bsEqLiab"
			}
			for _, name := range names {
				if name != side && !strings.HasPrefix(name, side+".") {
					continue
				}
				c := t.balance[name]
				entry := b.tuple(tuple, b.item(tuple+".name", b.tax.QName(name)))
				if c.actual != 0 {
					entry.children = append(entry.children, b.money(tuple+".changeValueActualPeriod", c.actual))
				}
				if c.previous != 0 || c.actual == 0 {
					entry.children = append(entry.children, b.money(tuple+".changeValuePreviousPeriod", c.previous))
				}
				n.children = append(n.children, entry)
			}
		}
		incomes := make([]string, 0, len(t.income))
		for name := range t.income {
			incomes = append(incomes, name)
		}
		sort.Strings(incomes)
		for _, name := range incomes {
			n.children = append(n.children, b.tuple("hbst.transfer.isChangeNetIncome",
				b.item("hbst.transfer.isChangeNetIncome.name", b.tax.QName(name)),
				b.money("hbst.transfer.isChangeNetIncome.changeValueActualPeriod", t.income[name])))
		}
		out = append(out, n)
	}
	return out
}
