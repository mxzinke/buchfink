package mcpserver

import (
	"context"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

type buchungenIn struct {
	Geschaeftsjahr int    `json:"geschaeftsjahr,omitempty"`
	Von            Datum  `json:"von,omitempty" jsonschema:"Buchungsdatum ab"`
	Bis            Datum  `json:"bis,omitempty" jsonschema:"Buchungsdatum bis"`
	Konto          Konto  `json:"konto,omitempty" jsonschema:"Buchungen mit einer Zeile auf diesem Konto"`
	KontaktID      uint   `json:"kontakt_id,omitempty"`
	BelegID        uint   `json:"beleg_id,omitempty"`
	Text           string `json:"text,omitempty" jsonschema:"Teil von Buchungstext, Buchungs- oder Belegnummer"`
	BetragMin      Betrag `json:"betrag_min,omitempty" jsonschema:"Summe der Sollseite ab"`
	BetragMax      Betrag `json:"betrag_max,omitempty" jsonschema:"Summe der Sollseite bis"`
	Kanal          string `json:"kanal,omitempty" jsonschema:"Erfassungsweg, z. B. ui oder agent:claude-code; agent findet alle Agenten"`
	Storniert      *bool  `json:"storniert,omitempty" jsonschema:"true nur stornierte, false nur nicht stornierte"`
	Seite
}

type buchungZeile struct {
	ID              uint     `json:"id"`
	Nummer          string   `json:"nummer"`
	Datum           string   `json:"datum"`
	Text            string   `json:"text"`
	BelegNr         string   `json:"beleg_nr,omitempty"`
	BelegID         *uint    `json:"beleg_id,omitempty"`
	KontaktID       *uint    `json:"kontakt_id,omitempty"`
	Quelle          string   `json:"quelle"`
	Art             string   `json:"art" jsonschema:"normal oder storno"`
	Betrag          Betrag   `json:"betrag" jsonschema:"Summe der Sollseite"`
	Konten          []string `json:"konten"`
	Kanal           string   `json:"kanal,omitempty"`
	StorniertDurch  *uint    `json:"storniert_durch,omitempty"`
	Festgeschrieben bool     `json:"festgeschrieben"`
	Abschluss       bool     `json:"abschlussperiode,omitempty"`
}

type buchungenOut struct {
	Buchungen []buchungZeile `json:"buchungen"`
	Seitenende
}

func debitSum(e *domain.JournalEntry) domain.Cents {
	var sum domain.Cents
	for _, l := range e.Lines {
		if l.Side == domain.SideDebit {
			sum += l.Amount
		}
	}
	return sum
}

func buchungZeileOf(e *domain.JournalEntry, reversedBy map[uint]uint) buchungZeile {
	accounts := []string{}
	seen := map[string]bool{}
	for _, l := range e.Lines {
		if !seen[l.Account] {
			seen[l.Account] = true
			accounts = append(accounts, l.Account)
		}
	}
	z := buchungZeile{ID: e.ID, Nummer: e.EntryNumber, Datum: e.BookingDate, Text: e.Description,
		BelegNr: e.DocumentNumber, BelegID: e.ReceiptID, KontaktID: e.ContactID, Quelle: string(e.Source),
		Art: "normal", Betrag: betrag(debitSum(e)), Konten: accounts, Kanal: e.Channel,
		Festgeschrieben: e.CommittedAt != nil, Abschluss: e.ClosingPeriod}
	if e.Kind == domain.EntryKindReversal {
		z.Art = "storno"
	}
	if by, ok := reversedBy[e.ID]; ok {
		z.StorniertDurch = &by
	}
	return z
}

type buchungLesenIn struct {
	BuchungID uint `json:"buchung_id"`
}

type buchungLesenOut struct {
	buchungZeile
	Belegdatum        string          `json:"belegdatum"`
	LeistungVon       string          `json:"leistung_von"`
	LeistungBis       string          `json:"leistung_bis"`
	Valuta            string          `json:"valuta,omitempty"`
	Faellig           string          `json:"faellig,omitempty"`
	Steuerfall        string          `json:"steuerfall,omitempty"`
	StornoVon         *uint           `json:"storno_von,omitempty"`
	StornoGrund       string          `json:"storno_grund,omitempty"`
	KorrigiertBuchung *uint           `json:"korrigiert_buchung,omitempty" jsonschema:"diese Buchung ersetzt die genannte"`
	Neubuchung        *uint           `json:"neubuchung,omitempty" jsonschema:"die Buchung, die diese stornierte ersetzt"`
	BankumsatzID      *uint           `json:"bankumsatz_id,omitempty"`
	Waehrung          string          `json:"waehrung"`
	Bearbeiter        string          `json:"bearbeiter,omitempty"`
	Zeilen            []buchungsZeile `json:"zeilen"`
}

func registerBuchungen(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "buchungen_suchen",
		Title: "Buchungen suchen",
		Description: "Das Journal eines Geschäftsjahres nach Zeitraum, Konto, Kontakt, Beleg, Text, Betrag, " +
			"Erfassungsweg und Storno. Je Buchung die Kopfzeile mit Konten; alle Zeilen liefert buchung_lesen.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in buchungenIn) (buchungenOut, error) {
		out := buchungenOut{Buchungen: []buchungZeile{}}
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return out, err
		}
		from, err := in.Von.check("von")
		if err != nil {
			return out, err
		}
		to, err := in.Bis.check("bis")
		if err != nil {
			return out, err
		}
		minAmount, err := in.BetragMin.cents("betrag_min")
		if err != nil {
			return out, err
		}
		maxAmount, err := in.BetragMax.cents("betrag_max")
		if err != nil {
			return out, err
		}
		entries, err := s.Accounting.GetEntries(ctx, year)
		if err != nil {
			return out, err
		}
		reversedBy := map[uint]uint{}
		for _, e := range entries {
			if e.ReversalOfID != nil {
				reversedBy[*e.ReversalOfID] = e.ID
			}
		}
		text := strings.ToLower(strings.TrimSpace(in.Text))
		rows := []buchungZeile{}
		for i := range entries {
			e := &entries[i]
			_, reversed := reversedBy[e.ID]
			switch {
			case from != "" && e.BookingDate < from,
				to != "" && e.BookingDate > to,
				in.Konto != "" && !hasAccount(e, string(in.Konto)),
				in.KontaktID != 0 && !hasContact(e, in.KontaktID),
				in.BelegID != 0 && (e.ReceiptID == nil || *e.ReceiptID != in.BelegID),
				text != "" && !strings.Contains(strings.ToLower(e.Description+" "+e.EntryNumber+" "+e.DocumentNumber), text),
				in.BetragMin != "" && debitSum(e) < minAmount,
				in.BetragMax != "" && debitSum(e) > maxAmount,
				in.Kanal != "" && !strings.HasPrefix(e.Channel, in.Kanal),
				in.Storniert != nil && *in.Storniert != reversed:
				continue
			}
			rows = append(rows, buchungZeileOf(e, reversedBy))
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Nummer < rows[j].Nummer })
		out.Buchungen, out.Seitenende, err = Paginate(rows, in.Seite)
		return out, err
	})

	AddRead(tb, Spec{
		Name:  "buchung_lesen",
		Title: "Buchung lesen",
		Description: "Eine Buchung mit allen Zeilen (Konto, Seite, Betrag, Steuerschlüssel), Daten, Beleg, " +
			"Erfassungsweg und Storno-Bezügen. Zum Korrigieren buchung_stornieren.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in buchungLesenIn) (buchungLesenOut, error) {
		e, err := s.Accounting.Entry(ctx, in.BuchungID)
		if err != nil {
			return buchungLesenOut{}, err
		}
		reversedBy := map[uint]uint{}
		for _, year := range []int{e.FiscalYear, e.FiscalYear + 1} {
			later, err := s.Accounting.GetEntries(ctx, year)
			if err != nil {
				return buchungLesenOut{}, err
			}
			for _, l := range later {
				if l.ReversalOfID != nil && *l.ReversalOfID == e.ID {
					reversedBy[e.ID] = l.ID
				}
			}
		}
		out := buchungLesenOut{buchungZeile: buchungZeileOf(e, reversedBy), Belegdatum: e.DocumentDate,
			LeistungVon: e.ServiceDateFrom, LeistungBis: e.ServiceDateTo, Valuta: e.ValueDate, Faellig: e.DueDate,
			Steuerfall: string(e.TaxTreatment), StornoVon: e.ReversalOfID, StornoGrund: e.ReversalReason,
			KorrigiertBuchung: e.CorrectsEntryID, BankumsatzID: e.BankTxID, Waehrung: e.Currency, Bearbeiter: e.Actor,
			Zeilen: zeilenOf(e.Lines)}
		if replacement, err := s.Journal.CorrectionOf(ctx, e.ID); err == nil && replacement != nil {
			out.Neubuchung = &replacement.ID
		}
		return out, nil
	})
}

func hasAccount(e *domain.JournalEntry, account string) bool {
	for _, l := range e.Lines {
		if l.Account == account {
			return true
		}
	}
	return false
}

func hasContact(e *domain.JournalEntry, id uint) bool {
	if e.ContactID != nil && *e.ContactID == id {
		return true
	}
	for _, l := range e.Lines {
		if l.ContactID != nil && *l.ContactID == id {
			return true
		}
	}
	return false
}
