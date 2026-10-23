package ebilanz

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Der Auszug der HGB-/E-Bilanz-Taxonomie liegt je Fassung als JSON unter
// taxonomy/ und entsteht mit scripts/gen_ebilanz_taxonomy.py aus dem amtlichen
// Paket (Lizenz: taxonomy/NOTICE).
//
//go:embed taxonomy/de-*.json
var taxonomyFiles embed.FS

// Rollen der Presentation- und Calculation-Linkbase, die der Auszug führt.
const (
	RoleBalanceSheet      = "balanceSheet"
	RoleIncomeStatement   = "incomeStatement"
	RoleIncomeMicroBilG   = "incomeStatementMicroBilG"
	RoleAppropriation     = "appropriationProfits"
	RoleTransfers         = "transfersCommercialCodeToTax"
	RoleTaxAssets         = "transfersTaxAssets"
	RoleDetailInformation = "detailedInformation"
)

// Kennzeichen der Finanzverwaltung (Referenzteil fiscalRequirement).
const (
	ReqMandatory    = "M" // Mussfeld
	ReqSumMandatory = "S" // Summenmussfeld
	ReqCalcRequired = "R" // rechnerisch notwendig, soweit vorhanden
)

// Werte von notPermittedFor.
const (
	notPermittedTax        = "steuerlich"
	notPermittedCommercial = "handelsrechtlich"
)

// Taxonomy ist eine Fassung der Taxonomie im Umfang des Auszugs.
type Taxonomy struct {
	Version      string            `json:"version"`
	Release      string            `json:"release"`
	Copyright    string            `json:"copyright"`
	Notice       string            `json:"notice"`
	Source       string            `json:"source"`
	Namespaces   map[string]string `json:"namespaces"`
	EntrySchemas EntrySchemas      `json:"entrySchemas"`
	Elements     []*Element        `json:"elements"`

	byName   map[string]*Element
	calcKids map[string][]Arc
	presKids map[string][]Arc
}

// EntrySchemas sind die Einstiegsschemata, auf die die Instanz verweist
// (Leitfaden B.3): GCD und genau eines der beiden GAAP-Schemata.
type EntrySchemas struct {
	GCD       string `json:"gcd"`
	Fiscal    string `json:"fiscal"`
	MicroBilG string `json:"microbilg"`
}

// Element ist eine Position der Taxonomie.
type Element struct {
	Name     string `json:"name"`
	NS       string `json:"ns"`
	Kind     string `json:"kind"`
	Type     string `json:"type"`
	Period   string `json:"period"`
	Balance  string `json:"balance"`
	Abstract bool   `json:"abstract"`
	// Req ist das Kennzeichen der Finanzverwaltung (ReqMandatory …).
	Req string `json:"req"`
	// Forms nennt die zulässigen Rechtsformarten (EU, KSt, PG); leer heißt alle.
	Forms              []string    `json:"forms"`
	GKVOrUKV           string      `json:"gkvUkv"`
	NotPermittedFor    []string    `json:"notPermittedFor"`
	ValidSince         string      `json:"validSince"`
	ValidThrough       string      `json:"validThrough"`
	FiscalValidSince   string      `json:"fiscalValidSince"`
	FiscalValidThrough string      `json:"fiscalValidThrough"`
	Label              string      `json:"label"`
	Shells             []string    `json:"shells"`
	Pres               []Edge      `json:"pres"`
	Calc               []Edge      `json:"calc"`
	Tuple              *TupleModel `json:"tuple"`
	ChoiceHead         string      `json:"choiceHead"`
	Alternatives       []string    `json:"alternatives"`
}

// Edge ist eine Kante zur Oberposition: [Rolle, Oberposition, Reihenfolge
// bzw. Rechengewicht].
type Edge struct {
	Role   string
	Parent string
	Value  float64
}

// UnmarshalJSON liest die Kante aus ihrer Kurzform.
func (e *Edge) UnmarshalJSON(b []byte) error {
	var raw [3]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &e.Role); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[1], &e.Parent); err != nil {
		return err
	}
	return json.Unmarshal(raw[2], &e.Value)
}

// TupleModel ist der Aufbau eines Tupels nach dem XSD.
type TupleModel struct {
	Model    string       `json:"model"`
	Children []TupleChild `json:"children"`
}

// TupleChild ist ein Bestandteil eines Tupels; Max -1 heißt unbeschränkt.
type TupleChild struct {
	Ref string `json:"ref"`
	Min int    `json:"min"`
	Max int    `json:"max"`
}

// Arc ist eine Unterposition mit ihrem Rechengewicht.
type Arc struct {
	Child  string
	Weight int
	Order  float64
}

var (
	taxonomyMu    sync.Mutex
	taxonomyCache = map[string]*Taxonomy{}
)

// Versions nennt die eingebetteten Fassungen, älteste zuerst.
func Versions() []string {
	return []string{"6.9", "6.10"}
}

// LoadTaxonomy liest eine eingebettete Fassung.
func LoadTaxonomy(version string) (*Taxonomy, error) {
	taxonomyMu.Lock()
	defer taxonomyMu.Unlock()
	if t, ok := taxonomyCache[version]; ok {
		return t, nil
	}
	raw, err := taxonomyFiles.ReadFile("taxonomy/de-" + version + ".json")
	if err != nil {
		return nil, fmt.Errorf("die Taxonomie %s ist nicht eingebunden", version)
	}
	var t Taxonomy
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, fmt.Errorf("die Taxonomie %s ließ sich nicht lesen: %w", version, err)
	}
	if err := t.index(); err != nil {
		return nil, fmt.Errorf("die Taxonomie %s ist unvollständig: %w", version, err)
	}
	taxonomyCache[version] = &t
	return &t, nil
}

func roleKey(role, parent string) string { return role + "\x00" + parent }

func (t *Taxonomy) index() error {
	t.byName = make(map[string]*Element, len(t.Elements))
	for _, e := range t.Elements {
		if _, dup := t.byName[e.Name]; dup {
			return fmt.Errorf("Element %s doppelt", e.Name)
		}
		t.byName[e.Name] = e
	}
	t.calcKids = map[string][]Arc{}
	t.presKids = map[string][]Arc{}
	for _, e := range t.Elements {
		for _, c := range e.Calc {
			if _, ok := t.byName[c.Parent]; !ok {
				return fmt.Errorf("Oberposition %s von %s fehlt", c.Parent, e.Name)
			}
			k := roleKey(c.Role, c.Parent)
			t.calcKids[k] = append(t.calcKids[k], Arc{Child: e.Name, Weight: int(c.Value)})
		}
		for _, p := range e.Pres {
			k := roleKey(p.Role, p.Parent)
			t.presKids[k] = append(t.presKids[k], Arc{Child: e.Name, Order: p.Value})
		}
	}
	for _, m := range []map[string][]Arc{t.calcKids, t.presKids} {
		for k := range m {
			arcs := m[k]
			sort.SliceStable(arcs, func(i, j int) bool {
				if arcs[i].Order != arcs[j].Order {
					return arcs[i].Order < arcs[j].Order
				}
				return arcs[i].Child < arcs[j].Child
			})
		}
	}
	return nil
}

// Element liefert eine Position.
func (t *Taxonomy) Element(name string) (*Element, bool) {
	e, ok := t.byName[name]
	return e, ok
}

// QName ist der Name mit dem Präfix des Namensraums, wie die Instanz ihn
// schreibt.
func (t *Taxonomy) QName(name string) string {
	if e, ok := t.byName[name]; ok {
		return e.NS + ":" + name
	}
	return "de-gaap-ci:" + name
}

// CalcChildren sind die rechnerisch verknüpften Unterpositionen in einer Rolle.
func (t *Taxonomy) CalcChildren(role, parent string) []Arc {
	return t.calcKids[roleKey(role, parent)]
}

// PresChildren sind die Unterpositionen der Darstellung in einer Rolle.
func (t *Taxonomy) PresChildren(role, parent string) []Arc {
	return t.presKids[roleKey(role, parent)]
}

// CalcParent ist die rechnerische Oberposition in einer Rolle.
func (e *Element) CalcParent(role string) (string, int, bool) {
	for _, c := range e.Calc {
		if c.Role == role {
			return c.Parent, int(c.Value), true
		}
	}
	return "", 0, false
}

// PresParent ist die Oberposition der Darstellung in einer Rolle.
func (e *Element) PresParent(role string) (string, bool) {
	for _, p := range e.Pres {
		if p.Role == role {
			return p.Parent, true
		}
	}
	return "", false
}

// InRole meldet, ob die Position zur Darstellung einer Rolle gehört.
func (e *Element) InRole(role string) bool {
	_, ok := e.PresParent(role)
	return ok
}

// AllowedForCorporations meldet, ob eine Kapitalgesellschaft die Position
// übermitteln darf (Leitfaden B.4.2).
func (e *Element) AllowedForCorporations() bool {
	if len(e.Forms) == 0 {
		return true
	}
	for _, f := range e.Forms {
		if f == "KSt" {
			return true
		}
	}
	return false
}

// NotPermittedInTaxBalance meldet „notPermittedFor = steuerlich": die Position
// darf in einer Steuer- oder Einheitsbilanz nicht stehen (Leitfaden A.11).
func (e *Element) NotPermittedInTaxBalance() bool {
	return contains(e.NotPermittedFor, notPermittedTax)
}

// NotPermittedInCommercialBalance meldet „notPermittedFor = handelsrechtlich".
func (e *Element) NotPermittedInCommercialBalance() bool {
	return contains(e.NotPermittedFor, notPermittedCommercial)
}

// ValidAt meldet, ob die Position zu Beginn und Stichtag berichtet werden darf
// (Leitfaden B.14.4.1).
func (e *Element) ValidAt(fiscalYearBegin, closingDate string) bool {
	if e.ValidSince != "" && fiscalYearBegin < e.ValidSince {
		return false
	}
	if e.ValidThrough != "" && closingDate > e.ValidThrough {
		return false
	}
	return true
}

// MandatoryAt meldet, ob das Kennzeichen „Mussfeld" zu Beginn und Stichtag
// gilt (Leitfaden B.14.4.2).
func (e *Element) MandatoryAt(fiscalYearBegin, closingDate string) bool {
	if e.Req != ReqMandatory {
		return false
	}
	if e.FiscalValidSince != "" && fiscalYearBegin < e.FiscalValidSince {
		return false
	}
	if e.FiscalValidThrough != "" && closingDate > e.FiscalValidThrough {
		return false
	}
	return true
}

// Monetary meldet einen Geldbetrag.
func (e *Element) Monetary() bool { return e.Type == "monetary" }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
