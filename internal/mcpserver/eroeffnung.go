package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type eroeffnungKontoIn struct {
	Konto  Konto         `json:"konto"`
	Seite  Buchungsseite `json:"seite"`
	Betrag Betrag        `json:"betrag"`
}

type eroeffnungPostenIn struct {
	KontaktID       uint   `json:"kontakt_id" jsonschema:"Kunde: Forderung; Lieferant: Verbindlichkeit"`
	Betrag          Betrag `json:"betrag" jsonschema:"offen, brutto"`
	Rechnungsnummer string `json:"rechnungsnummer"`
	Rechnungsdatum  Datum  `json:"rechnungsdatum"`
	FaelligAm       Datum  `json:"faellig_am,omitempty"`
	Satz            Satz   `json:"satz,omitempty" jsonschema:"wie im Vorsystem erklärt, 0 = nicht steuerbar; ohne Angabe kein Skonto"`
}

type eroeffnungAnlageIn struct {
	Bezeichnung         string     `json:"bezeichnung"`
	Konto               Konto      `json:"konto"`
	AngeschafftAm       Datum      `json:"angeschafft_am"`
	Anschaffungskosten  Betrag     `json:"anschaffungskosten" jsonschema:"historisch"`
	AfaBisher           Betrag     `json:"afa_bisher,omitempty" jsonschema:"bis zum Umstieg"`
	Methode             AfaMethode `json:"methode,omitempty"`
	NutzungsdauerMonate int        `json:"nutzungsdauer_monate,omitempty" jsonschema:"ab Anschaffung"`
	NutzungsdauerGrund  string     `json:"nutzungsdauer_grund,omitempty"`
	GebaeudeStichtag    Datum      `json:"gebaeude_stichtag,omitempty"`
	Inventarnummer      string     `json:"inventarnummer,omitempty"`
}

type eroeffnungIn struct {
	WriteArgs
	NurVorschau  bool                 `json:"nur_vorschau,omitempty"`
	BelegID      uint                 `json:"beleg_id" jsonschema:"Schlussbilanz oder Summen- und Saldenliste"`
	Vorsystem    string               `json:"vorsystem,omitempty"`
	Konten       []eroeffnungKontoIn  `json:"konten" jsonschema:"Bilanzkonten; 1200 und 3300 als Summe der offenen Posten"`
	OffenePosten []eroeffnungPostenIn `json:"offene_posten,omitempty" jsonschema:"je Rechnung"`
	Anlagen      []eroeffnungAnlageIn `json:"anlagen,omitempty" jsonschema:"je Anlagegut; Buchwerte je Konto wie in konten"`
}

func (in eroeffnungIn) preview() bool { return in.NurVorschau }

type eroeffnungAnlageOut struct {
	AnlageID           uint   `json:"anlage_id,omitempty"`
	Inventarnummer     string `json:"inventarnummer,omitempty"`
	Bezeichnung        string `json:"bezeichnung"`
	Konto              string `json:"konto"`
	Anschaffungskosten Betrag `json:"anschaffungskosten"`
	AfaBisher          Betrag `json:"afa_bisher"`
	Buchwert           Betrag `json:"buchwert"`
	AfaErstesJahr      Betrag `json:"afa_erstes_jahr" jsonschema:"planmäßige AfA im ersten Geschäftsjahr in Buchfink"`
}

type eroeffnungOut struct {
	Wiederholung
	Vorschau    bool                  `json:"vorschau"`
	Stichtag    string                `json:"stichtag"`
	Aktiva      Betrag                `json:"aktiva"`
	Passiva     Betrag                `json:"passiva"`
	Buchungen   []buchungKurz         `json:"buchungen"`
	Anlagen     []eroeffnungAnlageOut `json:"anlagen"`
	Differenzen []string              `json:"differenzen" jsonschema:"stehen der Buchung entgegen"`
}

func (out eroeffnungOut) Ref() string { return "eroeffnung:" + out.Stichtag }

func registerEroeffnung(tb *Toolbox) {
	AddWrite(tb, Spec{
		Name:  "eroeffnung_buchen",
		Title: "Eröffnung buchen",
		Description: "Umstieg aus einem Vorsystem: Eröffnung zum in der App festgelegten Tag aus der abgelegten " +
			"Schlussbilanz buchen. Erst nur_vorschau, nach dem Ja buchen; danach auswertung art=umstieg. " +
			"Nicht für Saldenvortrag oder Gründung.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in eroeffnungIn) (eroeffnungOut, error) {
		out := eroeffnungOut{Vorschau: in.NurVorschau, Buchungen: []buchungKurz{}, Anlagen: []eroeffnungAnlageOut{},
			Differenzen: []string{}}
		req, err := in.request(ctx, s)
		if err != nil {
			return out, err
		}
		var result *service.TakeoverPreview
		if in.NurVorschau {
			result, err = s.Takeover.Preview(ctx, req)
		} else {
			result, err = s.Takeover.Book(ctx, req)
		}
		if errors.Is(err, service.ErrTakeoverNotSet) {
			return out, &ToolError{Code: CodeRule, Nachricht: err.Error(), Werkzeug: "nur_app",
				NaechsterSchritt: "Den Menschen bitten, den Umstieg in Buchfink unter Unternehmen › Stammdaten " +
					"festzulegen."}
		}
		if err != nil {
			return out, err
		}
		out.Stichtag = result.Date
		out.Aktiva, out.Passiva = betrag(result.DebitTotal), betrag(result.CreditTotal)
		out.Differenzen = append(out.Differenzen, result.Differences...)
		for _, entry := range result.Entries {
			out.Buchungen = append(out.Buchungen, buchungKurzOf(entry))
		}
		for _, a := range result.Assets {
			out.Anlagen = append(out.Anlagen, eroeffnungAnlageOut{
				AnlageID: a.AssetID, Inventarnummer: a.InventoryNumber, Bezeichnung: a.Name, Konto: a.Account,
				Anschaffungskosten: betrag(a.AcquisitionCost), AfaBisher: betrag(a.AccumulatedDepreciation),
				Buchwert: betrag(a.BookValue), AfaErstesJahr: betrag(a.FirstYearDepreciation),
			})
		}
		return out, nil
	})
}

// request übersetzt die Eingabe in die Schlussbilanz des Vorsystems. Ein
// offener Posten gehört zur Seite seines Kontakts.
func (in eroeffnungIn) request(ctx context.Context, s core.AgentServices) (service.TakeoverRequest, error) {
	var req service.TakeoverRequest
	if in.BelegID == 0 {
		return req, inputError("beleg_id", "die abgelegte Schlussbilanz des Vorsystems fehlt")
	}
	req.ReceiptID, req.LegacySystem = &in.BelegID, in.Vorsystem
	for i, k := range in.Konten {
		field := fmt.Sprintf("konten[%d]", i)
		amount, err := k.Betrag.required(field + ".betrag")
		if err != nil {
			return req, err
		}
		side := domain.Side(k.Seite)
		if side != domain.SideDebit && side != domain.SideCredit {
			return req, inputError(field+".seite", "S oder H, nicht %q", k.Seite)
		}
		req.Accounts = append(req.Accounts, service.OpeningBalanceLine{Account: string(k.Konto), Side: side, Amount: amount})
	}
	for i, p := range in.OffenePosten {
		field := fmt.Sprintf("offene_posten[%d]", i)
		item := service.OpenOpeningItem{ContactID: p.KontaktID, DocumentNumber: p.Rechnungsnummer}
		contact, err := s.Contacts.Contact(ctx, p.KontaktID)
		if err != nil {
			return req, notFound("kontakt_id", p.KontaktID, err)
		}
		if item.Amount, err = p.Betrag.required(field + ".betrag"); err != nil {
			return req, err
		}
		if item.DocumentDate, err = p.Rechnungsdatum.check(field + ".rechnungsdatum"); err != nil {
			return req, err
		}
		if p.FaelligAm != "" {
			if item.DueDate, err = p.FaelligAm.check(field + ".faellig_am"); err != nil {
				return req, err
			}
		}
		if p.Satz != "" {
			rate := p.Satz.rate()
			item.TaxRate, item.TaxTreatment = &rate, domain.TaxTreatmentDomestic
			if rate == domain.TaxRateNone {
				item.TaxTreatment = domain.TaxTreatmentNotTaxable
			}
		}
		if contact.Type == domain.ContactTypeVendor {
			req.Payables = append(req.Payables, item)
		} else {
			req.Receivables = append(req.Receivables, item)
		}
	}
	for i, a := range in.Anlagen {
		field := fmt.Sprintf("anlagen[%d]", i)
		asset := service.TakeoverAsset{Name: a.Bezeichnung, Account: string(a.Konto),
			UsefulLifeMonths: a.NutzungsdauerMonate, UsefulLifeReason: a.NutzungsdauerGrund, LegacyRef: a.Inventarnummer}
		var err error
		if asset.AcquisitionDate, err = a.AngeschafftAm.check(field + ".angeschafft_am"); err != nil {
			return req, err
		}
		if asset.AcquisitionCost, err = a.Anschaffungskosten.required(field + ".anschaffungskosten"); err != nil {
			return req, err
		}
		if asset.AccumulatedDepreciation, err = a.AfaBisher.cents(field + ".afa_bisher"); err != nil {
			return req, err
		}
		if asset.Method, err = pick(field+".methode", a.Methode, depreciationMethods); err != nil {
			return req, err
		}
		if a.GebaeudeStichtag != "" {
			if asset.BuildingReferenceDate, err = a.GebaeudeStichtag.check(field + ".gebaeude_stichtag"); err != nil {
				return req, err
			}
		}
		req.Assets = append(req.Assets, asset)
	}
	return req, nil
}

// --- auswertung art=umstieg --------------------------------------------------

type umstieg struct {
	Stichtag     string             `json:"stichtag" jsonschema:"festgelegter Tag des Umstiegs"`
	Gebucht      bool               `json:"gebucht" jsonschema:"Eröffnungsbuchungen stehen im Journal"`
	Abgestimmt   bool               `json:"abgestimmt" jsonschema:"gebucht und ohne Differenz zur Schlussbilanz"`
	BelegID      uint               `json:"beleg_id,omitempty" jsonschema:"Schlussbilanz des Vorsystems"`
	Konten       []umstiegKonto     `json:"konten" jsonschema:"Salden: Soll positiv, Haben negativ"`
	Nebenbuecher []umstiegNebenbuch `json:"nebenbuecher"`
	Differenzen  []string           `json:"differenzen"`
}

type umstiegKonto struct {
	Konto         string `json:"konto"`
	Name          string `json:"name"`
	Schlussbilanz Betrag `json:"schlussbilanz"`
	Gebucht       Betrag `json:"gebucht"`
	Differenz     Betrag `json:"differenz"`
}

type umstiegNebenbuch struct {
	Art       string `json:"art" jsonschema:"forderungen, verbindlichkeiten oder anlagen"`
	Konto     string `json:"konto"`
	Anzahl    int    `json:"anzahl"`
	Nebenbuch Betrag `json:"nebenbuch"`
	Sachkonto Betrag `json:"sachkonto"`
	Differenz Betrag `json:"differenz"`
}

var subLedgerKinds = map[string]string{
	"receivables": "forderungen", "payables": "verbindlichkeiten", "assets": "anlagen",
}

// umstiegOf baut die Abstimmung der Eröffnung; die Konten kommen seitenweise.
func umstiegOf(rec *service.TakeoverReconciliation, page Seite) (*umstieg, Seitenende, error) {
	out := &umstieg{Stichtag: rec.Date, Gebucht: rec.Booked, Abgestimmt: rec.Reconciled, BelegID: rec.ReceiptID,
		Nebenbuecher: make([]umstiegNebenbuch, 0, len(rec.SubLedgers)), Differenzen: rec.Differences}
	rows := make([]umstiegKonto, 0, len(rec.Accounts))
	for _, a := range rec.Accounts {
		rows = append(rows, umstiegKonto{Konto: a.Account, Name: a.AccountName, Schlussbilanz: betrag(a.Statement),
			Gebucht: betrag(a.Booked), Differenz: betrag(a.Difference)})
	}
	for _, sub := range rec.SubLedgers {
		out.Nebenbuecher = append(out.Nebenbuecher, umstiegNebenbuch{Art: subLedgerKinds[sub.Kind], Konto: sub.Account,
			Anzahl: sub.Count, Nebenbuch: betrag(sub.SubLedger), Sachkonto: betrag(sub.Ledger),
			Differenz: betrag(sub.Difference)})
	}
	var end Seitenende
	var err error
	out.Konten, end, err = Paginate(rows, page)
	return out, end, err
}
