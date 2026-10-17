package ebilanz

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// bookableAccounts sind die bebuchbaren SKR04-Konten einer Kapitalgesellschaft:
// Klassen 0 bis 7 ohne reservierte Konten und ohne die Konten der
// Personengesellschaft.
func bookableAccounts(t *testing.T) []domain.Account {
	t.Helper()
	var out []domain.Account
	for _, acc := range accounting.DefaultSKR04Accounts() {
		if acc.IsReserved || acc.Kontenklasse > 7 || acc.StatementType == "Statistisch" {
			continue
		}
		if accounting.IsPartnershipAccount(accountNumber(acc)) {
			continue
		}
		out = append(out, acc)
	}
	if len(out) < 1000 {
		t.Fatalf("nur %d bebuchbare Konten im Katalog", len(out))
	}
	return out
}

func withBalance(acc domain.Account, debit domain.Cents) domain.Account {
	if debit >= 0 {
		acc.DebitSum, acc.CreditSum = debit, 0
	} else {
		acc.DebitSum, acc.CreditSum = 0, -debit
	}
	return acc
}

// withoutPosition sind die Konten der Ergebnisverwendung ohne feste
// Bedeutung: für sie hat die Taxonomie keine Position, und der Export nennt
// sie als Befund.
var withoutPosition = map[string]bool{"7788": true, "7789": true, "7800-7899": true}

// fiscalYearsOf nennt Wirtschaftsjahre, die eine Fassung annimmt.
func fiscalYearsOf(version string) [][2]string {
	switch version {
	case "6.9":
		return [][2]string{{"2025-01-01", "2025-12-31"}, {"2026-01-01", "2026-12-31"}}
	default:
		return [][2]string{{"2026-01-01", "2026-12-31"}, {"2027-01-01", "2027-12-31"}}
	}
}

// Jedes bebuchbare Konto findet in jeder eingebundenen Fassung eine Position,
// die die Taxonomie für eine GmbH zulässt, die in der Rolle seiner
// Gliederungsposition liegt und keine Auffangposition der Gliederung ist.
func TestEveryBookableAccountHasAValidPosition(t *testing.T) {
	accounts := bookableAccounts(t)
	for _, version := range Versions() {
		tax, err := LoadTaxonomy(version)
		if err != nil {
			t.Fatal(err)
		}
		for _, fy := range fiscalYearsOf(version) {
			var problems []string
			used := map[string]map[string][]string{} // Rolle → Position → Konten
			for _, appropriated := range []bool{false, true} {
				p := placement{tax: tax, appropriated: appropriated, begin: fy[0], closing: fy[1]}
				for _, base := range accounts {
					for _, sign := range []domain.Cents{100, -100} {
						acc := withBalance(base, sign)
						key, _ := accounting.StatementKeyForAccount(acc)
						if key == "passiva.A.VI" && !appropriated {
							continue
						}
						pl := p.one(acc)
						where := fmt.Sprintf("%s %s %s (%s, Saldo %s)", version, fy[1][:4], acc.Number, key, sign)
						if withoutPosition[acc.Number] {
							if pl.Finding == "" {
								problems = append(problems, where+": ohne Befund auf "+pl.Element)
							}
							continue
						}
						if pl.Finding != "" {
							problems = append(problems, where+": "+pl.Finding)
							continue
						}
						if pl.Fallback {
							problems = append(problems, where+": nur Auffangposition "+pl.Element)
						}
						problems = append(problems, checkPosition(tax, pl, fy)...)
						if used[pl.role] == nil {
							used[pl.role] = map[string][]string{}
						}
						used[pl.role][pl.Element] = append(used[pl.role][pl.Element], acc.Number)
					}
				}
			}
			problems = append(problems, antichainProblems(tax, used)...)
			reportProblems(t, problems)
		}
	}
}

func checkPosition(tax *Taxonomy, pl Placement, fy [2]string) []string {
	var out []string
	where := fmt.Sprintf("%s %s → %s", tax.Version, pl.Account, pl.Element)
	el, ok := tax.Element(pl.Element)
	switch {
	case !ok:
		return []string{where + ": unbekannt"}
	case !el.Monetary() || el.Abstract:
		out = append(out, where+": kein Betrag")
	case !el.AllowedForCorporations():
		out = append(out, where+": für Kapitalgesellschaften unzulässig")
	case !el.ValidAt(fy[0], fy[1]):
		out = append(out, where+": im Wirtschaftsjahr nicht gültig")
	case el.Balance == "":
		out = append(out, where+": ohne Saldorichtung")
	}
	if !el.InRole(pl.role) {
		out = append(out, where+": nicht in der Rolle "+pl.role)
	}
	if el.Req == ReqSumMandatory && !tax.within(RoleBalanceSheet, pl.Element, []string{"bs.eqLiab.equity"}) {
		out = append(out, where+": Summenmussfeld ohne Unterposition (Leitfaden B.14.14)")
	}
	return out
}

// antichainProblems: keine Position mit Konten darf rechnerisch über einer
// anderen Position mit Konten stehen — sonst zählte die Oberposition ihre
// eigenen Konten und die der Unterposition, und die Rechenregel bräche.
func antichainProblems(tax *Taxonomy, used map[string]map[string][]string) []string {
	var out []string
	for role, elements := range used {
		for name, accounts := range elements {
			el, _ := tax.Element(name)
			for cur, guard := el, 0; cur != nil && guard < 64; guard++ {
				parent, _, ok := cur.CalcParent(role)
				if !ok {
					break
				}
				if other, clash := elements[parent]; clash {
					out = append(out, fmt.Sprintf("%s %s: %s (%s) liegt unter %s (%s)",
						tax.Version, role, name, strings.Join(accounts[:1], ","), parent, strings.Join(other[:1], ",")))
				}
				cur, _ = tax.Element(parent)
			}
		}
	}
	return out
}

func reportProblems(t *testing.T, problems []string) {
	t.Helper()
	if len(problems) == 0 {
		return
	}
	sort.Strings(problems)
	seen := map[string]bool{}
	var unique []string
	for _, p := range problems {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	if len(unique) > 80 {
		unique = append(unique[:80], fmt.Sprintf("… und %d weitere", len(unique)-80))
	}
	t.Errorf("%d Befunde:\n%s", len(seen), strings.Join(unique, "\n"))
}

// Die Auffangpositionen der Gliederung gelten in jeder Fassung und liegen in
// der Rolle ihrer Gliederungsposition.
func TestStatementDefaultsAreValid(t *testing.T) {
	for _, version := range Versions() {
		tax, err := LoadTaxonomy(version)
		if err != nil {
			t.Fatal(err)
		}
		var problems []string
		for _, fy := range fiscalYearsOf(version) {
			p := placement{tax: tax, begin: fy[0], closing: fy[1]}
			for key, def := range statementDefaults {
				role := RoleBalanceSheet
				if strings.HasPrefix(key, "guv.") {
					role = RoleIncomeStatement
				}
				for _, name := range []string{def.Element, def.Alt} {
					if name == "" {
						continue
					}
					el := p.resolve(name)
					if el == "" {
						problems = append(problems, fmt.Sprintf("%s %s: %s unbekannt oder ungültig", version, key, name))
						continue
					}
					if !tax.within(role, el, statementAnchors[key]) {
						problems = append(problems, fmt.Sprintf("%s %s: %s außerhalb der Gliederungsposition", version, key, el))
					}
					problems = append(problems, checkPosition(tax, Placement{Account: key, Element: el, role: role}, fy)...)
				}
			}
		}
		reportProblems(t, problems)
	}
}

// Jede Gliederungsposition, die Konten trägt, hat Teilbäume und eine
// Auffangposition.
func TestEveryStatementLineHasAnchors(t *testing.T) {
	for _, line := range accounting.StatementLines() {
		switch {
		case line.IsSubtotal, line.Section == domain.SectionStatistical,
			strings.HasSuffix(line.Key, ".X"), line.Key == "passiva.A.V", line.Key == "passiva.A.VI",
			line.Key == "aktiva.0":
			continue
		}
		children := 0
		for _, other := range accounting.StatementLines() {
			if strings.HasPrefix(other.Key, line.Key+".") {
				children++
			}
		}
		if children > 0 {
			continue
		}
		if len(statementAnchors[line.Key]) == 0 {
			t.Errorf("%s (%s) hat keine Teilbäume der Taxonomie", line.Key, line.Label)
		}
		if _, ok := statementDefaults[line.Key]; !ok {
			t.Errorf("%s (%s) hat keine Auffangposition", line.Key, line.Label)
		}
	}
}

// In der GuV nach MicroBilG (§ 275 Abs. 5 HGB) geht jedes Erfolgskonto auf
// eine Position, die rechnerisch in den Jahresüberschuss der Kurzform eingeht.
func TestEveryIncomeAccountFitsTheMicroBilGStatement(t *testing.T) {
	accounts := bookableAccounts(t)
	for _, version := range Versions() {
		tax, err := LoadTaxonomy(version)
		if err != nil {
			t.Fatal(err)
		}
		fy := fiscalYearsOf(version)[0]
		p := placement{tax: tax, microBilG: true, begin: fy[0], closing: fy[1]}
		var problems []string
		for _, base := range accounts {
			for _, sign := range []domain.Cents{100, -100} {
				acc := withBalance(base, sign)
				if key, _ := accounting.StatementKeyForAccount(acc); !strings.HasPrefix(key, "guv.") {
					continue
				}
				pl := p.one(acc)
				if pl.Finding != "" {
					problems = append(problems, fmt.Sprintf("%s %s: %s", version, acc.Number, pl.Finding))
					continue
				}
				if !tax.within(RoleIncomeMicroBilG, pl.Element, []string{"ismi.netIncome"}) {
					problems = append(problems, fmt.Sprintf("%s %s: %s nicht unter ismi.netIncome", version, acc.Number, pl.Element))
				}
			}
		}
		reportProblems(t, problems)
	}
}
