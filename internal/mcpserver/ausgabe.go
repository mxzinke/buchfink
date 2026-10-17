package mcpserver

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Gemeinsame Ausgaben: eine Buchung kurz und eine Vorschau.

type buchungsZeile struct {
	Konto            string `json:"konto"`
	Kontoname        string `json:"kontoname,omitempty"`
	Seite            string `json:"seite" jsonschema:"S (Soll) oder H (Haben)"`
	Betrag           Betrag `json:"betrag"`
	Steuerschluessel string `json:"steuerschluessel,omitempty"`
	KontaktID        *uint  `json:"kontakt_id,omitempty"`
	Text             string `json:"text,omitempty"`
}

type buchungKurz struct {
	ID        uint            `json:"id"`
	Nummer    string          `json:"nummer"`
	Datum     string          `json:"datum"`
	Text      string          `json:"text"`
	BelegNr   string          `json:"beleg_nr,omitempty"`
	Abschluss bool            `json:"abschlussperiode,omitempty"`
	Zeilen    []buchungsZeile `json:"zeilen"`
}

func zeilenOf(lines []domain.JournalLine) []buchungsZeile {
	out := make([]buchungsZeile, 0, len(lines))
	for _, l := range lines {
		out = append(out, buchungsZeile{Konto: l.Account, Kontoname: l.AccountName, Seite: string(l.Side),
			Betrag: betrag(l.Amount), Steuerschluessel: l.TaxKey, KontaktID: l.ContactID, Text: l.Text})
	}
	return out
}

func buchungKurzOf(e *domain.JournalEntry) buchungKurz {
	if e == nil {
		return buchungKurz{Zeilen: []buchungsZeile{}}
	}
	return buchungKurz{ID: e.ID, Nummer: e.EntryNumber, Datum: e.BookingDate, Text: e.Description,
		BelegNr: e.DocumentNumber, Abschluss: e.ClosingPeriod, Zeilen: zeilenOf(e.Lines)}
}

type hinweisZeile struct {
	Code  string `json:"code"`
	Titel string `json:"titel"`
	Text  string `json:"text"`
}

type vorschau struct {
	Zeilen   []buchungsZeile `json:"zeilen"`
	Netto    Betrag          `json:"netto"`
	Steuer   Betrag          `json:"steuer"`
	Brutto   Betrag          `json:"brutto"`
	Hinweise []hinweisZeile  `json:"hinweise"`
	Befunde  []hinweisZeile  `json:"befunde" jsonschema:"blockieren die Buchung; mit uebersteuerung_grund begründet übergehbar"`
}

func vorschauOf(p *service.PostingPreview) vorschau {
	v := vorschau{Zeilen: zeilenOf(p.Lines), Netto: betrag(p.Net), Steuer: betrag(p.Tax), Brutto: betrag(p.Gross),
		Hinweise: []hinweisZeile{}, Befunde: []hinweisZeile{}}
	for _, w := range p.Warnings {
		v.Hinweise = append(v.Hinweise, hinweisZeile{Code: w.Code, Titel: w.Title, Text: w.Detail})
	}
	for _, f := range p.IssuingFindings {
		v.Hinweise = append(v.Hinweise, hinweisZeile{Code: "pflichtangabe", Titel: "Pflichtangabe", Text: f})
	}
	for _, f := range p.InputTaxFindings {
		v.Befunde = append(v.Befunde, hinweisZeile{Code: f.Code, Titel: f.Title, Text: f.Detail})
	}
	return v
}

// buchungOut ist die Ausgabe der buchenden Werkzeuge.
type buchungOut struct {
	Wiederholung
	Buchung    *buchungKurz `json:"buchung,omitempty"`
	Vorschau   *vorschau    `json:"vorschau,omitempty" jsonschema:"nur mit nur_vorschau: so würde gebucht"`
	Einstufung string       `json:"einstufung,omitempty" jsonschema:"direkt oder rueckfrage"`
}

func (o buchungOut) Ref() string {
	if o.Buchung == nil {
		return "vorschau"
	}
	return fmt.Sprintf("buchung:%d", o.Buchung.ID)
}

func booked(e *domain.JournalEntry, level string) buchungOut {
	b := buchungKurzOf(e)
	return buchungOut{Buchung: &b, Einstufung: level}
}
