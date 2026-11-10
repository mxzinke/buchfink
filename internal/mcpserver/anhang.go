package mcpserver

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

type anhangTexte struct {
	Organkredite          *string `json:"organkredite,omitempty"`
	WeitereAngaben        *string `json:"weitere_angaben,omitempty"`
	Methoden              *string `json:"methoden,omitempty"`
	Organbezuege          *string `json:"organbezuege,omitempty"`
	Nachtragsbericht      *string `json:"nachtragsbericht,omitempty"`
	Verpflichtungen       *string `json:"verpflichtungen,omitempty"`
	Haftungsverhaeltnisse *string `json:"haftungsverhaeltnisse,omitempty"`
	Beteiligungen         *string `json:"beteiligungen,omitempty"`
	Ergebnisverwendung    *string `json:"ergebnisverwendung,omitempty"`
}

type anhangIn struct {
	WriteArgs
	Jahr  int          `json:"jahr" jsonschema:"Geschäftsjahr"`
	Texte *anhangTexte `json:"texte,omitempty" jsonschema:"nur geänderte Abschnitte; leerer Text löscht den Inhalt"`
}

func (in anhangIn) readOnly() bool { return in.Texte == nil }

type anhangAbschnitt struct {
	Abschnitt string `json:"abschnitt"`
	Titel     string `json:"titel"`
	Hinweis   string `json:"hinweis"`
	Norm      string `json:"norm"`
	Pflicht   bool   `json:"pflicht"`
	Text      string `json:"text"`
}
type anhangOut struct {
	Wiederholung
	Jahr       int               `json:"jahr"`
	Gesperrt   bool              `json:"gesperrt"`
	Abschnitte []anhangAbschnitt `json:"abschnitte"`
}

func (o anhangOut) Ref() string { return fmt.Sprintf("anhang:%d", o.Jahr) }

func registerAnhang(tb *Toolbox) {
	AddWrite(tb, Spec{Name: "anhang_text", Title: "Anhangtexte", Idempotent: true,
		Description: "Liest Anhangabschnitte mit Pflichtkennzeichen und schreibt Entwürfe je Geschäftsjahr. Ohne texte nur lesen. Vor Schreiben Wortlaut vorlegen und Ja einholen. Der Mensch gibt die Texte beim Aufstellen in der App frei; danach erst nach Rücknahme änderbar.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in anhangIn) (anhangOut, error) {
		texts := anhangTexte{}
		if in.Texte != nil {
			texts = *in.Texte
		}
		sections := []struct {
			key     string
			section domain.NotesSection
			text    *string
		}{
			{"organkredite", domain.NotesSectionBoardLoans, texts.Organkredite},
			{"weitere_angaben", domain.NotesSectionAdditional, texts.WeitereAngaben},
			{"methoden", domain.NotesSectionMethods, texts.Methoden},
			{"organbezuege", domain.NotesSectionBoard, texts.Organbezuege},
			{"nachtragsbericht", domain.NotesSectionSubsequent, texts.Nachtragsbericht},
			{"verpflichtungen", domain.NotesSectionCommitments, texts.Verpflichtungen},
			{"haftungsverhaeltnisse", domain.NotesSectionContingent, texts.Haftungsverhaeltnisse},
			{"beteiligungen", domain.NotesSectionInvestments, texts.Beteiligungen},
			{"ergebnisverwendung", domain.NotesSectionAppropriation, texts.Ergebnisverwendung},
		}
		for _, section := range sections {
			if section.text != nil {
				if _, err := s.Appropriation.SaveNotesText(ctx, in.Jahr, section.section, *section.text); err != nil {
					return anhangOut{}, err
				}
			}
		}
		editor, err := s.Appropriation.NotesEditor(ctx, in.Jahr)
		if err != nil {
			return anhangOut{}, err
		}
		out := anhangOut{Jahr: in.Jahr, Gesperrt: editor.Locked, Abschnitte: []anhangAbschnitt{}}
		for _, text := range editor.Texts {
			for _, section := range sections {
				if text.Section == section.section {
					out.Abschnitte = append(out.Abschnitte, anhangAbschnitt{section.key, text.Label, text.Hint, text.Basis, text.Required, text.Text})
				}
			}
		}
		return out, nil
	})
}
