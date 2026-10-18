package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type offenePostenIn struct {
	Richtung          PostenRichtung `json:"richtung,omitempty"`
	KontaktID         uint           `json:"kontakt_id,omitempty"`
	NurUeberfaellig   bool           `json:"nur_ueberfaellig,omitempty"`
	Stichtag          Datum          `json:"stichtag,omitempty" jsonschema:"Stand zu diesem Tag; Vorgabe heute"`
	MitAltersstruktur bool           `json:"mit_altersstruktur,omitempty" jsonschema:"Summen nach Überfälligkeit und Restlaufzeit"`
	Seite
}

type offenerPosten struct {
	BuchungID        uint    `json:"buchung_id"`
	Nummer           string  `json:"nummer"`
	BelegNr          string  `json:"beleg_nr,omitempty"`
	Richtung         string  `json:"richtung" jsonschema:"forderung oder verbindlichkeit"`
	KontaktID        uint    `json:"kontakt_id"`
	Kontakt          string  `json:"kontakt"`
	Belegdatum       string  `json:"belegdatum,omitempty"`
	Faellig          string  `json:"faellig,omitempty"`
	Betrag           Betrag  `json:"betrag"`
	Offen            Betrag  `json:"offen"`
	TageUeberfaellig int     `json:"tage_ueberfaellig,omitempty"`
	VerzugAb         string  `json:"verzug_ab,omitempty"`
	Verzugszinsen    *Betrag `json:"verzugszinsen,omitempty" jsonschema:"nach § 288 BGB bis zum Stichtag; nicht gebucht"`
	Pauschale        *Betrag `json:"pauschale,omitempty" jsonschema:"40 Euro nach § 288 Abs. 5 BGB bei Unternehmern"`
	LetzteMahnung    string  `json:"letzte_mahnung,omitempty"`
	Mahnstufe        int     `json:"mahnstufe,omitempty"`
	Hinweis          string  `json:"hinweis,omitempty"`
}

type altersband struct {
	Schluessel string `json:"schluessel"`
	Name       string `json:"name"`
	Betrag     Betrag `json:"betrag"`
	Posten     int    `json:"posten"`
}

type altersseite struct {
	Seite        string       `json:"seite"`
	Summe        Betrag       `json:"summe"`
	Posten       int          `json:"posten"`
	Faelligkeit  []altersband `json:"faelligkeit"`
	Restlaufzeit []altersband `json:"restlaufzeit"`
}

type offenePostenOut struct {
	Stichtag       string          `json:"stichtag"`
	Posten         []offenerPosten `json:"posten"`
	SummeOffen     Betrag          `json:"summe_offen"`
	Altersstruktur []altersseite   `json:"altersstruktur"`
	Hinweis        string          `json:"hinweis,omitempty"`
	Seitenende
}

type mahnungIn struct {
	KeyedWriteArgs
	BuchungID uint   `json:"buchung_id" jsonschema:"Buchung der offenen Forderung"`
	Datum     Datum  `json:"datum" jsonschema:"Tag, an dem die Mahnung dem Kunden zugegangen ist"`
	Stufe     int    `json:"stufe,omitempty" jsonschema:"1. Mahnung, 2. Mahnung …; Vorgabe 1"`
	Notiz     string `json:"notiz,omitempty" jsonschema:"Weg und Ablage des Schreibens"`
}

type mahnungOut struct {
	Wiederholung
	ID      uint           `json:"id"`
	Posten  *offenerPosten `json:"posten,omitempty" jsonschema:"der Posten mit Verzugsrechnung nach dem Vermerk"`
	Hinweis string         `json:"hinweis"`
}

func (o mahnungOut) Ref() string { return fmt.Sprintf("mahnung:%d", o.ID) }

func richtungOfItem(item domain.OpenItem) string {
	if item.ContactType == domain.ContactTypeCustomer {
		return "forderung"
	}
	return "verbindlichkeit"
}

func postenOf(item domain.OpenItem, overdue map[uint]service.OverdueItem, cutoff string) offenerPosten {
	p := offenerPosten{BuchungID: item.EntryID, Nummer: item.EntryNumber, BelegNr: item.DocumentNumber,
		Richtung: richtungOfItem(item), KontaktID: item.ContactID, Kontakt: item.ContactName,
		Belegdatum: item.DocumentDate, Faellig: item.DueDate, Betrag: betrag(item.GrossAmount), Offen: betrag(item.OpenAmount)}
	if item.DueDate != "" && item.DueDate < cutoff {
		p.TageUeberfaellig = daysBetween(item.DueDate, cutoff)
	}
	if o, ok := overdue[item.EntryID]; ok {
		p.VerzugAb, p.LetzteMahnung, p.Mahnstufe, p.Hinweis = o.DefaultFrom, o.LastReminder, o.ReminderLevel, o.Note
		if o.Interest != 0 {
			p.Verzugszinsen = optBetrag(o.Interest)
		}
		if o.LumpSum != 0 {
			p.Pauschale = optBetrag(o.LumpSum)
		}
	}
	return p
}

func daysBetween(from, to string) int {
	a, err1 := time.Parse("2006-01-02", from)
	b, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(b.Sub(a).Hours() / 24)
}

func overdueByEntry(ctx context.Context, s *core.Services, cutoff string) (map[uint]service.OverdueItem, error) {
	claims, err := s.Overdue.Claims(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	out := map[uint]service.OverdueItem{}
	for _, c := range claims {
		for _, item := range c.Items {
			out[item.EntryID] = item
		}
	}
	return out, nil
}

func bands(buckets []domain.AgingBucket) []altersband {
	out := make([]altersband, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, altersband{Schluessel: string(b.Key), Name: b.Label, Betrag: betrag(b.Amount), Posten: b.Items})
	}
	return out
}

func registerPosten(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "offene_posten",
		Title: "Offene Posten",
		Description: "Offene Forderungen und Verbindlichkeiten zum Stichtag mit Fälligkeit, Tagen überfällig, " +
			"Verzugsbeginn, Verzugszinsen und Pauschale (§§ 286, 288 BGB, nicht gebucht) und letzter Mahnung; " +
			"auf Wunsch die Altersstruktur. Zahlungen ordnet bankumsatz_zuordnen oder bankumsatz_buchen zu; eine " +
			"versandte Mahnung vermerkt mahnung_vermerken.",
	}, func(ctx context.Context, c *Call, s *core.Services, in offenePostenIn) (offenePostenOut, error) {
		out := offenePostenOut{Posten: []offenerPosten{}, Altersstruktur: []altersseite{}}
		cutoff, err := in.Stichtag.check("stichtag")
		if err != nil {
			return out, err
		}
		if cutoff == "" {
			cutoff = today()
		}
		out.Stichtag = cutoff

		items, err := s.Payments.OpenItemsAt(ctx, cutoff)
		if err != nil {
			return out, err
		}
		overdue, err := overdueByEntry(ctx, s, cutoff)
		if err != nil {
			return out, err
		}
		var picked []domain.OpenItem
		rows := []offenerPosten{}
		var sum domain.Cents
		for _, item := range items {
			if item.OpenAmount == 0 {
				continue
			}
			switch {
			case in.Richtung != "" && in.Richtung != "alle" && richtungOfItem(item) != string(in.Richtung),
				in.KontaktID != 0 && item.ContactID != in.KontaktID,
				in.NurUeberfaellig && (item.DueDate == "" || item.DueDate >= cutoff):
				continue
			}
			picked = append(picked, item)
			sum += item.OpenAmount
			rows = append(rows, postenOf(item, overdue, cutoff))
		}
		out.SummeOffen = betrag(sum)
		if in.MitAltersstruktur {
			aging := s.Payments.AgeOpenItems(picked, cutoff)
			for _, side := range aging.Sides {
				out.Altersstruktur = append(out.Altersstruktur, altersseite{Seite: side.Label, Summe: betrag(side.Total),
					Posten: side.Items, Faelligkeit: bands(side.Buckets), Restlaufzeit: maturities(side.Maturities)})
			}
		}
		out.Hinweis = "Verzugszinsen und Pauschale sind gerechnet, nicht gebucht; sie sind erst mit der Zahlung Ertrag."
		out.Posten, out.Seitenende, err = Paginate(rows, in.Seite)
		return out, err
	})

	AddWrite(tb, Spec{
		Name:  "mahnung_vermerken",
		Title: "Mahnung vermerken",
		Description: "Vermerkt, dass eine offene Forderung gemahnt wurde und wann die Mahnung zugegangen ist. Das " +
			"Schreiben entsteht außerhalb von Buchfink. Ab dem Folgetag des Zugangs rechnet offene_posten den Verzug, " +
			"auch gegenüber einem Verbraucher ohne Hinweis in der Rechnung. Vorher im Chat bestätigen lassen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in mahnungIn) (mahnungOut, error) {
		date, err := in.Datum.check("datum")
		if err != nil {
			return mahnungOut{}, err
		}
		if date == "" {
			return mahnungOut{}, inputError("datum", "der Tag des Zugangs fehlt")
		}
		note, err := s.Overdue.RecordReminder(ctx, service.ReminderRequest{EntryID: in.BuchungID, Date: date,
			Level: in.Stufe, Note: in.Notiz})
		if err != nil {
			return mahnungOut{}, err
		}
		out := mahnungOut{ID: note.ID, Hinweis: "Verzugszinsen und Pauschale bucht Buchfink nicht; sie sind mit der Zahlung Ertrag."}
		cutoff := today()
		items, err := s.Payments.OpenItemsAt(ctx, cutoff)
		if err != nil {
			return out, err
		}
		overdue, err := overdueByEntry(ctx, s, cutoff)
		if err != nil {
			return out, err
		}
		for _, item := range items {
			if item.EntryID == in.BuchungID {
				p := postenOf(item, overdue, cutoff)
				out.Posten = &p
			}
		}
		return out, nil
	})
}

func maturities(bands []domain.MaturityBand) []altersband {
	out := make([]altersband, 0, len(bands))
	for _, b := range bands {
		out = append(out, altersband{Schluessel: string(b.Key), Name: b.Label, Betrag: betrag(b.Amount), Posten: b.Items})
	}
	return out
}
