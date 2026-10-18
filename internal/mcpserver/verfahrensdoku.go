package mcpserver

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// organisationsTexteIn sind die Abschnitte, die sich ändern sollen; ein
// fehlender bleibt, wie er ist.
type organisationsTexteIn struct {
	Zustaendigkeit *string `json:"zustaendigkeit,omitempty" jsonschema:"wer die Bücher führt und verantwortet"`
	Belegfluss     *string `json:"belegfluss,omitempty" jsonschema:"wie Belege eingehen und geprüft werden"`
	Scannen        *string `json:"scannen,omitempty"`
	Freigabe       *string `json:"freigabe,omitempty" jsonschema:"wer freigibt und festschreibt"`
	Vertretung     *string `json:"vertretung,omitempty"`
	Sicherung      *string `json:"sicherung,omitempty"`
	Sonstiges      *string `json:"sonstiges,omitempty"`
}

type organisationsTexte struct {
	Zustaendigkeit string `json:"zustaendigkeit"`
	Belegfluss     string `json:"belegfluss"`
	Scannen        string `json:"scannen"`
	Freigabe       string `json:"freigabe"`
	Vertretung     string `json:"vertretung"`
	Sicherung      string `json:"sicherung"`
	Sonstiges      string `json:"sonstiges"`
}

type verfahrensdokuIn struct {
	WriteArgs
	Texte *organisationsTexteIn `json:"texte,omitempty" jsonschema:"nur geänderte Abschnitte; leer setzt das Muster zurück"`
}

type verfahrensdokuOut struct {
	Wiederholung
	Texte     organisationsTexte `json:"texte"`
	Muster    []string           `json:"muster" jsonschema:"Abschnitte, in denen noch das unveränderte Muster steht"`
	Geaendert []string           `json:"geaendert" jsonschema:"Abschnitte, die dieser Aufruf geändert hat"`
	Hinweis   string             `json:"hinweis,omitempty"`
}

func (o verfahrensdokuOut) Ref() string { return "verfahrensdoku" }

// readOnly: ohne texte liest das Werkzeug nur, mit dem Recht zu lesen und auch
// im Prüfermodus.
func (in verfahrensdokuIn) readOnly() bool { return in.Texte == nil }

// organisationSection verbindet einen Abschnitt der Eingabe mit seinem Feld im
// Kern.
type organisationSection struct {
	name  string
	in    func(*organisationsTexteIn) *string
	field func(*domain.OrganisationTexts) *string
}

var organisationSections = []organisationSection{
	{"zustaendigkeit", func(t *organisationsTexteIn) *string { return t.Zustaendigkeit },
		func(t *domain.OrganisationTexts) *string { return &t.Responsibilities }},
	{"belegfluss", func(t *organisationsTexteIn) *string { return t.Belegfluss },
		func(t *domain.OrganisationTexts) *string { return &t.ReceiptFlow }},
	{"scannen", func(t *organisationsTexteIn) *string { return t.Scannen },
		func(t *domain.OrganisationTexts) *string { return &t.Scanning }},
	{"freigabe", func(t *organisationsTexteIn) *string { return t.Freigabe },
		func(t *domain.OrganisationTexts) *string { return &t.Approval }},
	{"vertretung", func(t *organisationsTexteIn) *string { return t.Vertretung },
		func(t *domain.OrganisationTexts) *string { return &t.Substitution }},
	{"sicherung", func(t *organisationsTexteIn) *string { return t.Sicherung },
		func(t *domain.OrganisationTexts) *string { return &t.Backup }},
	{"sonstiges", func(t *organisationsTexteIn) *string { return t.Sonstiges },
		func(t *domain.OrganisationTexts) *string { return &t.Notes }},
}

func organisationsTexteOf(t domain.OrganisationTexts) organisationsTexte {
	return organisationsTexte{Zustaendigkeit: t.Responsibilities, Belegfluss: t.ReceiptFlow, Scannen: t.Scanning,
		Freigabe: t.Approval, Vertretung: t.Substitution, Sicherung: t.Backup, Sonstiges: t.Notes}
}

func registerVerfahrensdoku(tb *Toolbox) {
	AddWrite(tb, Spec{
		Name:       "verfahrensdoku_text",
		Title:      "Verfahrensdokumentation: Organisationstexte",
		Idempotent: true,
		Description: "Liest und schreibt die Organisationstexte der Verfahrensdokumentation: Zuständigkeit, " +
			"Belegfluss, Scannen, Freigabe, Vertretung, Sicherung, Sonstiges. Ohne texte liefert es den Stand und " +
			"welche Abschnitte noch das Muster tragen, ohne zu schreiben; das braucht nur das Recht zu lesen und " +
			"geht auch im Prüfermodus. Vor dem Schreiben dem Menschen den neuen " +
			"Wortlaut jedes Abschnitts vorlegen und sein Ja einholen; die Texte beschreiben, wie sein Unternehmen " +
			"arbeitet.",
	}, func(ctx context.Context, c *Call, s *core.Services, in verfahrensdokuIn) (verfahrensdokuOut, error) {
		current := s.ProcDoc.OrganisationTexts(ctx)
		out := verfahrensdokuOut{Muster: []string{}, Geaendert: []string{}}
		if in.Texte != nil {
			next := current
			defaults := domain.DefaultOrganisationTexts()
			for _, sec := range organisationSections {
				text := sec.in(in.Texte)
				if text == nil {
					continue
				}
				value := *text
				if value == "" {
					value = *sec.field(&defaults)
				}
				if value != *sec.field(&current) {
					*sec.field(&next) = *text
					out.Geaendert = append(out.Geaendert, sec.name)
				}
			}
			if len(out.Geaendert) > 0 {
				if err := s.ProcDoc.SaveOrganisationTexts(ctx, next); err != nil {
					return verfahrensdokuOut{}, err
				}
				current = s.ProcDoc.OrganisationTexts(ctx)
				out.Hinweis = "Die nächste Fassung der Verfahrensdokumentation erzeugt der Mensch in der App; " +
					"sie übernimmt die neuen Texte."
			}
		}
		out.Texte = organisationsTexteOf(current)
		defaults := domain.DefaultOrganisationTexts()
		for _, sec := range organisationSections {
			if pattern := *sec.field(&defaults); pattern != "" && *sec.field(&current) == pattern {
				out.Muster = append(out.Muster, sec.name)
			}
		}
		return out, nil
	})
}
