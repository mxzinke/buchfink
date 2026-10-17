package ebilanz

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Placement ist ein Konto mit Saldo und die Position der Taxonomie, unter der
// es in der Instanz und im Kontennachweis steht.
type Placement struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	// Balance ist der Sollsaldo des Kontos.
	Balance domain.Cents `json:"balance"`
	// PositionKey ist die Gliederungsposition, unter der die Bilanz das Konto
	// zeigt.
	PositionKey string `json:"positionKey"`
	Element     string `json:"element"`
	// Value ist der Saldo in Richtung der Position: positiv, wo er die
	// Position erhöht (Leitfaden A.13).
	Value domain.Cents `json:"value"`
	// Fallback heißt: keine Kontenregel, sondern die Auffangposition der
	// Gliederungsposition.
	Fallback bool `json:"fallback"`
	// Finding benennt, warum das Konto keine Position hat.
	Finding string `json:"finding,omitempty"`

	role string
}

// ElementForAccount nennt die Position der Taxonomie, unter der ein Konto mit
// seinem Saldo in die E-Bilanz eines Wirtschaftsjahres geht; leer, wo es
// keine gibt. Für Jahre außerhalb der eingebundenen Fassungen gilt die
// nächstliegende.
func ElementForAccount(acc domain.Account, fiscalYearBegin, fiscalYearEnd string) string {
	versions := Versions()
	version := versions[len(versions)-1]
	if choice, err := ChooseVersion(fiscalYearBegin, fiscalYearEnd, ""); err == nil {
		version = choice.Version
	} else if fiscalYearBegin < versionRules[0].RegularAfter {
		version = versions[0]
	}
	tax, err := LoadTaxonomy(version)
	if err != nil {
		return ""
	}
	p := placement{tax: tax, begin: fiscalYearBegin, closing: fiscalYearEnd}
	if p.begin == "" || p.closing == "" {
		p.begin, p.closing = ruleOf(version).RegularAfter, ruleOf(version).RegularAfter
	}
	pl := p.one(acc)
	if pl.Finding != "" {
		return ""
	}
	return pl.Element
}

// placement fasst, was die Zuordnung vom Jahresabschluss wissen muss.
type placement struct {
	tax *Taxonomy
	// appropriated: die Bilanz zeigt den Bilanzgewinn (§ 268 Abs. 1 HGB).
	appropriated bool
	microBilG    bool
	begin        string
	closing      string
}

// place ordnet die Konten mit Saldo zu.
func (p placement) place(accounts []domain.Account) []Placement {
	out := make([]Placement, 0, len(accounts))
	for _, acc := range accounting.AccountsWithBalance(accounts) {
		out = append(out, p.one(acc))
	}
	return out
}

func accountNumber(acc domain.Account) string {
	if acc.IsRange && acc.RangeStart != "" {
		return acc.RangeStart
	}
	return acc.Number
}

func (p placement) one(acc domain.Account) Placement {
	pl := Placement{Account: acc.Number, Name: acc.Name, Balance: acc.DebitSum - acc.CreditSum}
	number := accountNumber(acc)
	if accounting.IsPartnershipAccount(number) {
		pl.Finding = "Konto einer Personengesellschaft; bei einer Kapitalgesellschaft ist es nicht zulässig " +
			"(Leitfaden B.4.2). Saldo auf ein Konto der GmbH umbuchen."
		return pl
	}
	key, known := accounting.StatementKeyForAccount(acc)
	if !known {
		pl.Finding = fmt.Sprintf("Die SKR04-Position %q kennt die Gliederung nicht.", acc.PositionID)
		return pl
	}
	pl.PositionKey = key

	table, anchors, defaults := compiledSKR04, statementAnchors[key], statementDefaults
	role := RoleBalanceSheet
	switch {
	case strings.HasPrefix(key, "guv."):
		role = RoleIncomeStatement
	case key == "passiva.A.IV" && p.appropriated, key == "passiva.A.VI":
		if !p.appropriated {
			pl.Finding = "Konto der Ergebnisverwendung, obwohl die Bilanz keinen Bilanzgewinn ausweist."
			return pl
		}
		table, anchors, defaults = compiledAppropriation, appropriationAnchors, appropriationDefaults
		role = RoleAppropriation
	case key == "statistisch":
		pl.Finding = "Statistisches Konto mit Saldo; es gehört weder in Bilanz noch GuV (Saldenvortrag prüfen)."
		return pl
	}
	pl.role = role
	if len(anchors) == 0 {
		pl.Finding = fmt.Sprintf("Für die Gliederungsposition %q gibt es keine Position der Taxonomie.", accounting.LineLabel(key))
		return pl
	}

	rule, hasRule := table.lookup(number)
	var candidates []string
	if hasRule {
		for _, name := range []string{rule.Element, rule.Alt} {
			el := p.resolve(name)
			if el == "" {
				continue
			}
			if rule.Net && name == rule.Element || p.tax.within(role, el, anchors) {
				candidates = append(candidates, el)
			}
		}
	}
	if len(candidates) == 0 {
		def, ok := defaults[key]
		if !ok && role == RoleAppropriation {
			pl.Finding = "Für dieses Konto der Ergebnisverwendung hat die Taxonomie keine Position " +
				"(Leitfaden B.9.1.1); den Betrag auf ein Konto mit fester Bedeutung (7700 bis 7790) umbuchen."
			return pl
		}
		if !ok {
			pl.Finding = fmt.Sprintf("Für die Gliederungsposition %q gibt es keine Auffangposition der Taxonomie.", accounting.LineLabel(key))
			return pl
		}
		for _, name := range []string{def.Element, def.Alt} {
			if el := p.resolve(name); el != "" {
				candidates = append(candidates, el)
			}
		}
		pl.Fallback = true
	}
	if len(candidates) == 0 {
		pl.Finding = fmt.Sprintf("Die Taxonomie %s kennt keine Position für dieses Konto.", p.tax.Version)
		return pl
	}

	// Zwei Kandidaten derselben Gliederungsposition sind die beiden
	// Saldorichtungen: es gilt der, in dessen Richtung der Saldo positiv ist.
	pl.Element = candidates[0]
	if len(candidates) == 2 && p.directed(candidates[0], pl.Balance) < 0 {
		pl.Element = candidates[1]
	}
	if p.microBilG && role == RoleIncomeStatement {
		if el, ok := p.microBilGElement(pl.Element, pl.Balance); ok {
			pl.Element = el
		} else {
			pl.Finding = "Die Position gehört nicht zur GuV nach MicroBilG."
			pl.Element = ""
			return pl
		}
	}
	pl.Value = p.directed(pl.Element, pl.Balance)
	return pl
}

// resolve wählt aus „a|b" die erste Position, die die Fassung kennt und die
// im Wirtschaftsjahr gilt.
func (p placement) resolve(names string) string {
	if names == "" {
		return ""
	}
	for _, name := range strings.Split(names, "|") {
		if el, ok := p.tax.Element(name); ok && el.ValidAt(p.begin, p.closing) {
			return name
		}
	}
	return ""
}

// directed bringt einen Sollsaldo in die Richtung der Position.
func (p placement) directed(name string, debit domain.Cents) domain.Cents {
	if el, ok := p.tax.Element(name); ok && el.Balance == "credit" {
		return -debit
	}
	return debit
}

// microBilGElement prüft, ob die Position in die GuV nach MicroBilG eingeht,
// und lenkt die Bestandsveränderung um.
func (p placement) microBilGElement(name string, debit domain.Cents) (string, bool) {
	if p.tax.within(RoleIncomeMicroBilG, name, []string{"ismi.netIncome"}) {
		return name, true
	}
	if p.tax.within(RoleIncomeStatement, name, []string{microBilGInventoryParent}) {
		if debit < 0 {
			return microBilGInventoryIncrease, true
		}
		return microBilGInventoryDecrease, true
	}
	return "", false
}

// within meldet, ob eine Position in einem der Teilbäume liegt.
func (t *Taxonomy) within(role, name string, roots []string) bool {
	for cur, guard := name, 0; cur != "" && guard < 64; guard++ {
		for _, root := range roots {
			if cur == root {
				return true
			}
		}
		el, ok := t.Element(cur)
		if !ok {
			return false
		}
		parent, _, ok := el.CalcParent(role)
		if !ok {
			return false
		}
		cur = parent
	}
	return false
}
