package declaration

import (
	"encoding/json"
)

// Overview ist die erste Ebene der Anzeige: Ergebnis, Vordrucke mit ihren
// Ständen und was noch zu tun ist. Die Zeilen holt die Oberfläche erst beim
// Aufklappen eines Vordrucks (View).
type Overview struct {
	Kind        Kind           `json:"art"`
	Title       string         `json:"titel"`
	Year        int            `json:"jahr"`
	Status      Status         `json:"status"`
	FiguresHash string         `json:"zahlen_hash"`
	Complete    bool           `json:"abgabereif"`
	Result      []KeyFigure    `json:"ergebnis"`
	Forms       []FormOverview `json:"formulare"`
	Pending     []PendingLine  `json:"zu_klaeren"`
	Blocking    int            `json:"befunde_blockierend"`
	Warnings    int            `json:"befunde_warnung"`
	Hints       int            `json:"befunde_hinweis"`
	Submission  *Submission    `json:"uebermittlung,omitempty"`
}

// KeyFigure ist eine Hauptzahl der Erklärung. Label ist der amtliche Wortlaut
// der Zeile, Short der kurze Name über der Kennzahl.
type KeyFigure struct {
	Key   string `json:"schluessel"`
	Label string `json:"bezeichnung"`
	Short string `json:"kurz"`
	Value Value  `json:"wert"`
}

// FormOverview ist ein Vordruck ohne seine Zeilen.
type FormOverview struct {
	ID       string      `json:"id"`
	Title    string      `json:"titel"`
	Required Requirement `json:"erforderlich"`
	Reason   string      `json:"grund,omitempty"`
	Summary  Summary     `json:"uebersicht"`
}

// PendingLine ist eine Zeile, die eine Entscheidung oder Eingabe braucht.
type PendingLine struct {
	Key    string         `json:"schluessel"`
	Form   string         `json:"formular"`
	Label  string         `json:"bezeichnung"`
	Status LineStatus     `json:"status"`
	Owner  Responsibility `json:"zustaendig"`
	Value  Value          `json:"wert"`
	Notes  []string       `json:"hinweise,omitempty"`
}

// resultKey ist eine Hauptzahl der Übersicht. Der amtliche Wortlaut trägt
// Ausfüllhinweise wie „mit Minuszeichen; bitte in jedem Fall ausfüllen“ und
// passt nicht über eine Kennzahl.
type resultKey struct{ key, short string }

var resultKeys = map[Kind][]resultKey{
	KindVatYear: {
		{"ust.z118.kz816", "Verbleibende USt"},
		{"ust.z119", "Vorauszahlungssoll"},
		{"ust.z120.kz820", "Abschlusszahlung"},
	},
	KindCorporateTax: {
		{"kst.gk.summe", "Einkünfte"},
		{"kst.zve.zve", "Zu versteuerndes Einkommen"},
		{"kst.k.kst", "Körperschaftsteuer"},
		{"kst.k.solz", "Solidaritätszuschlag"},
		{"kst.k.ergebnis", "Abschlusszahlung"},
	},
	KindTradeTax: {
		{"gewst.k.ertrag_gerundet", "Gewerbeertrag"},
		{"gewst.k.messbetrag", "Messbetrag"},
		{"gewst.k.steuer", "Gewerbesteuer"},
		{"gewst.k.ergebnis", "Abschlusszahlung"},
	},
}

// Overview fasst die Erklärung zusammen.
func (d *Declaration) Overview() Overview {
	o := Overview{
		Kind: d.Kind, Title: d.Title, Year: d.Period.Year, Status: d.Status,
		FiguresHash: d.FiguresHash, Submission: d.Submission,
		Result: []KeyFigure{}, Forms: []FormOverview{}, Pending: []PendingLine{},
	}
	for _, r := range resultKeys[d.Kind] {
		if l, ok := d.Line(r.key); ok {
			o.Result = append(o.Result, KeyFigure{Key: r.key, Label: l.Label, Short: r.short, Value: l.Value})
		}
	}
	for _, f := range d.Forms {
		o.Forms = append(o.Forms, FormOverview{ID: f.ID, Title: f.Title, Required: f.Required, Reason: f.Reason, Summary: f.Summary})
		if f.Required == RequirementNone {
			continue
		}
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				if l.Status == StatusProposed || (l.Status == StatusOpen && l.Required) {
					o.Pending = append(o.Pending, PendingLine{Key: l.Key, Form: f.ID, Label: l.Label,
						Status: l.Status, Owner: l.Owner, Value: l.Value, Notes: l.Notes})
				}
			}
		}
	}
	for _, f := range d.Findings {
		switch f.Severity {
		case SeverityBlocking:
			o.Blocking++
		case SeverityWarning:
			o.Warnings++
		default:
			o.Hints++
		}
	}
	o.Complete = o.Blocking == 0
	return o
}

// ViewOptions wählt aus, was die Anzeige zeigt.
type ViewOptions struct {
	// Form zeigt nur diesen Vordruck; leer zeigt alle.
	Form string
	// Section zeigt nur diesen Abschnitt des Vordrucks.
	Section string
	// Filled lässt Zeilen ohne Wert weg, außer offenen.
	Filled bool
	// Pending zeigt nur offene und vorgeschlagene Zeilen.
	Pending bool
	// Applicable lässt Vordrucke und Zeilen weg, die entfallen.
	Applicable bool
}

// View liefert eine Kopie der Erklärung mit den gewählten Zeilen. Zahlen-Hash
// und Befunde bleiben die der ganzen Erklärung.
func (d *Declaration) View(opts ViewOptions) *Declaration {
	out := *d
	out.Forms = []Form{}
	for _, f := range d.Forms {
		if opts.Form != "" && f.ID != opts.Form {
			continue
		}
		if opts.Applicable && f.Required == RequirementNone {
			continue
		}
		form := f
		form.Sections = []Section{}
		for _, s := range f.Sections {
			if opts.Section != "" && s.ID != opts.Section {
				continue
			}
			section := s
			section.Lines = []Line{}
			for _, l := range s.Lines {
				switch {
				case opts.Pending && l.Status != StatusOpen && l.Status != StatusProposed:
					continue
				case opts.Applicable && l.Status == StatusNotApplicable:
					continue
				case opts.Filled && l.Value.IsZero() && l.Status != StatusOpen && l.Status != StatusProposed:
					continue
				}
				section.Lines = append(section.Lines, l)
			}
			if len(section.Lines) > 0 || (!opts.Pending && !opts.Filled) {
				form.Sections = append(form.Sections, section)
			}
		}
		out.Forms = append(out.Forms, form)
	}
	return &out
}

// ExportJSON schreibt die Erklärung im Schema buchfink.erklaerung/1.
func ExportJSON(d *Declaration) ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}
