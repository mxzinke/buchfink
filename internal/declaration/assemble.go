package declaration

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// figure ist der gerechnete Inhalt einer Zeile, bevor der Katalog ihr
// Bezeichnung, Zeile und Prüfvermerk gibt.
type figure struct {
	value  Value
	status LineStatus
	origin Origin
	notes  []string
}

// formState sagt, ob ein Vordruck abzugeben ist.
type formState struct {
	required Requirement
	reason   string
}

// calc sammelt die Zeilen einer Erklärung, während sie gerechnet wird.
type calc struct {
	cat      *Catalog
	inputs   map[string]Input
	sheet    map[string]figure
	forms    map[string]formState
	findings []Finding
	// params sind die gesetzlichen Werte des Jahres (KSt und GewSt).
	params accounting.TaxParameters
}

func newCalc(cat *Catalog, inputs []Input) *calc {
	c := &calc{
		cat: cat, inputs: map[string]Input{},
		sheet: map[string]figure{}, forms: map[string]formState{},
	}
	for _, in := range inputs {
		c.inputs[in.Key] = in
	}
	return c
}

func (c *calc) put(key string, f figure) figure {
	if _, ok := c.cat.Line(key); !ok {
		c.finding(SeverityBlocking, "intern_zeile_unbekannt", key,
			fmt.Sprintf("Die Rechnung setzt die Zeile %s, die der Zeilenkatalog %s nicht kennt.", key, c.cat.Version), "")
	}
	c.sheet[key] = f
	return f
}

// books setzt einen Wert aus den Büchern, über den nichts zu entscheiden ist.
func (c *calc) books(key string, v Value, origin Origin, notes ...string) figure {
	return c.resolve(key, figure{value: v, status: StatusAuto, origin: origin, notes: notes})
}

// propose setzt einen Wert, den die zuständige Stelle bestätigen muss.
func (c *calc) propose(key string, v Value, origin Origin, notes ...string) figure {
	return c.resolve(key, figure{value: v, status: StatusProposed, origin: origin, notes: notes})
}

// open setzt eine Zeile ohne Wert, die eine Eingabe braucht.
func (c *calc) open(key string, notes ...string) figure {
	return c.resolve(key, figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: notes})
}

// none kennzeichnet eine Zeile, die für die Gesellschaft nicht zutrifft. Ein
// Wert darf trotzdem stehen, etwa „nein“ bei einer Frage des Vordrucks.
func (c *calc) none(key string, v Value, notes ...string) figure {
	return c.resolve(key, figure{value: v, status: StatusNotApplicable, origin: Origin{Kind: OriginNone}, notes: notes})
}

// resolve setzt die Zeile; liegt eine Eingabe vor und lässt die Zeile sie zu,
// gilt die Eingabe.
func (c *calc) resolve(key string, fallback figure) figure {
	line, known := c.cat.Line(key)
	in, has := c.inputs[key]
	if !has {
		return c.put(key, fallback)
	}
	if known && !line.Input {
		c.finding(SeverityWarning, "eingabe_ohne_wirkung", key,
			fmt.Sprintf("Zur Zeile %q liegt eine Eingabe vor; die Zeile rechnet aber aus den Büchern, die Eingabe bleibt ohne Wirkung.", line.Label), "")
		return c.put(key, fallback)
	}
	status := StatusInput
	if !in.Confirmed {
		status = StatusProposed
	}
	f := figure{
		value:  in.Value,
		status: status,
		origin: Origin{Kind: OriginInput, InputKey: key, Text: inputOriginText(in)},
	}
	if !fallback.value.Empty() && fallback.value.Canonical() != in.Value.Canonical() {
		f.notes = append(f.notes, fmt.Sprintf("Aus den Büchern ergäbe sich %s.", describe(fallback.value)))
	}
	f.notes = append(f.notes, fallback.notes...)
	return c.put(key, f)
}

func inputOriginText(in Input) string {
	text := "Eingabe"
	switch in.Source {
	case InputFromNotice:
		text += " laut Bescheid"
	case InputFromResolution:
		text += " laut Beschluss"
	case InputFromReceipt:
		text += " laut Beleg"
	case InputFromBooks:
		text += " aus den Büchern bestätigt"
	}
	if in.ReceiptID != nil {
		text += fmt.Sprintf(" (Beleg %d)", *in.ReceiptID)
	}
	if in.Note != "" {
		text += ": " + in.Note
	}
	return text
}

func describe(v Value) string {
	switch {
	case v.Cents != nil:
		return v.Cents.String() + " €"
	case v.Bool != nil:
		if *v.Bool {
			return "ja"
		}
		return "nein"
	case v.Text != nil:
		return "„" + *v.Text + "“"
	case v.Date != nil:
		return *v.Date
	}
	return "kein Wert"
}

// amount ist der Betrag einer schon gerechneten Zeile.
func (c *calc) amount(key string) domain.Cents { return c.sheet[key].value.Amount() }

// flag ist der Ja/Nein-Wert einer gerechneten Zeile.
func (c *calc) flag(key string) (bool, bool) {
	v := c.sheet[key].value
	if v.Bool == nil {
		return false, false
	}
	return *v.Bool, true
}

func (c *calc) finding(sev Severity, code, key, text, legal string) {
	c.findings = append(c.findings, Finding{Severity: sev, Code: code, LineKey: key, Text: text, Legal: legal})
}

func (c *calc) form(id string, req Requirement, reason string) {
	c.forms[id] = formState{required: req, reason: reason}
}

// fromLines ist eine Herkunft „Rechnung aus Zeilen“.
func fromLines(formula string, keys ...string) Origin {
	return Origin{Kind: OriginComputation, FromLines: keys, Formula: formula}
}

// assemble baut aus Katalog und Rechnung die Vordrucke, zählt die Stände und
// ergänzt die Befunde zu offenen und vorgeschlagenen Zeilen.
func (c *calc) assemble() []Form {
	lines := map[string]*Line{}
	forms := make([]Form, 0, len(c.cat.Forms))
	for _, cf := range c.cat.Forms {
		state, ok := c.forms[cf.ID]
		if !ok {
			state = formState{required: RequirementMandatory}
		}
		form := Form{
			ID: cf.ID, Title: cf.Title, Required: state.required, Reason: state.reason,
			Source: c.cat.source(cf.Source), Sections: make([]Section, 0, len(cf.Sections)),
		}
		for _, cs := range cf.Sections {
			section := Section{ID: cs.ID, Title: cs.Title, Lines: make([]Line, 0, len(cs.Lines))}
			for _, cl := range cs.Lines {
				f, computed := c.sheet[cl.Key]
				if !computed {
					f = figure{status: StatusOpen, origin: Origin{Kind: OriginNone}}
					c.finding(SeverityBlocking, "intern_zeile_ohne_rechnung", cl.Key,
						fmt.Sprintf("Für die Zeile %q gibt es keine Rechnung.", cl.Label), "")
				}
				rounding := cl.Rounding
				if rounding == "" {
					rounding = RoundingNone
				}
				notes := []string{}
				if cl.Note != "" {
					notes = append(notes, cl.Note)
				}
				notes = append(notes, f.notes...)
				section.Lines = append(section.Lines, Line{
					Key: cl.Key, Row: cl.Row, Code: cl.Code, Label: cl.Label,
					Value: f.value, Unit: cl.Unit, Rounding: rounding, Origin: f.origin,
					Status: f.status, Owner: cl.Owner, InputAllowed: cl.Input, Required: cl.Required,
					Legal: cl.Legal, Check: cl.Check, Notes: notes,
				})
			}
			form.Sections = append(form.Sections, section)
		}
		forms = append(forms, form)
	}
	for i := range forms {
		for j := range forms[i].Sections {
			for k := range forms[i].Sections[j].Lines {
				l := &forms[i].Sections[j].Lines[k]
				lines[l.Key] = l
			}
		}
	}
	markProvisional(lines)

	for i := range forms {
		if forms[i].Required == RequirementNone {
			continue
		}
		for j := range forms[i].Sections {
			for _, l := range forms[i].Sections[j].Lines {
				switch {
				case l.Status == StatusOpen && l.Required:
					c.finding(SeverityBlocking, "zeile_offen", l.Key,
						fmt.Sprintf("%s: %q fehlt (%s).", forms[i].Title, l.Label, ownerText(l.Owner)), "")
				case l.Status == StatusOpen:
					c.finding(SeverityHint, "zeile_ohne_wert", l.Key,
						fmt.Sprintf("%s: %q ist ohne Wert.", forms[i].Title, l.Label), "")
				case l.Status == StatusProposed:
					c.finding(SeverityBlocking, "zeile_vorgeschlagen", l.Key,
						fmt.Sprintf("%s: %q ist vorgeschlagen (%s) und muss bestätigt werden.",
							forms[i].Title, l.Label, describe(l.Value)), "")
				}
			}
		}
	}
	summarize(forms)
	return forms
}

func ownerText(o Responsibility) string {
	switch o {
	case OwnerHuman:
		return "Angabe des Menschen in der App"
	case OwnerAgent:
		return "Angabe mit Rückfrage"
	}
	return "aus den Büchern"
}

// markProvisional kennzeichnet Zeilen, die aus offenen oder vorgeschlagenen
// rechnen, auch über mehrere Stufen.
func markProvisional(lines map[string]*Line) {
	keys := make([]string, 0, len(lines))
	for k := range lines {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for changed := true; changed; {
		changed = false
		for _, k := range keys {
			l := lines[k]
			if l.Provisional || l.Status == StatusInput {
				continue
			}
			for _, from := range l.Origin.FromLines {
				src, ok := lines[from]
				if !ok {
					continue
				}
				if src.Status == StatusOpen || src.Status == StatusProposed || src.Provisional {
					l.Provisional = true
					changed = true
					break
				}
			}
		}
	}
}

func summarize(forms []Form) {
	for i := range forms {
		var fs Summary
		for j := range forms[i].Sections {
			var s Summary
			for _, l := range forms[i].Sections[j].Lines {
				s.Lines++
				if !l.Value.IsZero() {
					s.Filled++
				}
				switch l.Status {
				case StatusOpen:
					s.Open++
				case StatusProposed:
					s.Proposed++
				case StatusInput:
					s.Inputs++
				}
			}
			forms[i].Sections[j].Summary = s
			fs.Lines += s.Lines
			fs.Filled += s.Filled
			fs.Open += s.Open
			fs.Proposed += s.Proposed
			fs.Inputs += s.Inputs
		}
		forms[i].Summary = fs
	}
}

// sortFindings ordnet die Befunde nach Schwere, dann nach Zeile.
func sortFindings(list []Finding) {
	rank := map[Severity]int{SeverityBlocking: 0, SeverityWarning: 1, SeverityHint: 2}
	sort.SliceStable(list, func(i, j int) bool {
		return rank[list[i].Severity] < rank[list[j].Severity]
	})
}
