package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- anlagen ----------------------------------------------------------------

type anlagenIn struct {
	Geschaeftsjahr int          `json:"geschaeftsjahr,omitempty"`
	Klasse         AnlageKlasse `json:"klasse,omitempty" jsonschema:"ohne Angabe alle"`
	Seite
}

type anlage struct {
	ID            uint   `json:"id"`
	Inventarnr    string `json:"inventarnummer"`
	Bezeichnung   string `json:"bezeichnung"`
	Klasse        string `json:"klasse"`
	Konto         string `json:"konto"`
	Angeschafft   string `json:"angeschafft"`
	AK            Betrag `json:"ak" jsonschema:"Anschaffungs- oder Herstellungskosten zum Stichtag"`
	Buchwert      Betrag `json:"buchwert"`
	AfAJahr       Betrag `json:"afa_jahr" jsonschema:"im Geschäftsjahr gebuchte Abschreibung"`
	AfAFaellig    Betrag `json:"afa_faellig" jsonschema:"im Geschäftsjahr noch zu buchen"`
	Methode       string `json:"methode"`
	Nutzungsdauer int    `json:"nutzungsdauer_monate,omitempty"`
	Status        string `json:"status" jsonschema:"im_bestand, abgeschrieben, abgegangen, ohne_zugangsbuchung oder afa_faellig"`
}

type kandidat struct {
	BuchungID uint   `json:"buchung_id"`
	Nummer    string `json:"nummer"`
	Datum     string `json:"datum"`
	Text      string `json:"text"`
	Konto     string `json:"konto"`
	Kontoname string `json:"kontoname"`
	Betrag    Betrag `json:"betrag"`
}

type anlagenOut struct {
	Anlagen    []anlage   `json:"anlagen"`
	AfAFaellig Betrag     `json:"afa_faellig_gesamt"`
	Kandidaten []kandidat `json:"aktivierungskandidaten" jsonschema:"Buchungen auf Anlagekonten ohne Anlagegut; mit abschlussbuchung art=anlage_aktivieren aufnehmen"`
	Seitenende
}

// --- abschluss_status -------------------------------------------------------

type abschlussStatusIn struct {
	Geschaeftsjahr int `json:"geschaeftsjahr,omitempty"`
}

type naechsteAktion struct {
	Werkzeug  string            `json:"werkzeug" jsonschema:"Werkzeug oder nur_app"`
	Parameter map[string]string `json:"parameter,omitempty"`
}

type abschlussSchritt struct {
	Schluessel     string         `json:"schluessel"`
	Nr             int            `json:"nr"`
	Titel          string         `json:"titel"`
	Stand          string         `json:"stand" jsonschema:"offen, erledigt oder uebersprungen"`
	Grund          string         `json:"grund,omitempty"`
	Detail         string         `json:"detail,omitempty"`
	Hinweis        string         `json:"hinweis"`
	Automatisch    bool           `json:"automatisch" jsonschema:"Buchfink erkennt die Erledigung aus den Daten"`
	NaechsteAktion naechsteAktion `json:"naechste_aktion"`
}

type abschlussStatusOut struct {
	Geschaeftsjahr int                `json:"geschaeftsjahr"`
	Stand          string             `json:"stand" jsonschema:"offen, aufgestellt, festgestellt oder offengelegt"`
	Stichtag       string             `json:"stichtag"`
	Offen          int                `json:"offen"`
	Erledigt       int                `json:"erledigt"`
	Uebersprungen  int                `json:"uebersprungen"`
	Schritte       []abschlussSchritt `json:"schritte"`
	NaechsteAktion *naechsteAktion    `json:"naechste_aktion,omitempty" jsonschema:"des ersten offenen Schritts"`
}

var stepActions = map[domain.ClosingStepKey]naechsteAktion{
	domain.ClosingStepDepreciation:        {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "afa_lauf"}},
	domain.ClosingStepWriteUp:             {Werkzeug: "nur_app"},
	domain.ClosingStepCurrencyValuation:   {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "fremdwaehrung"}},
	domain.ClosingStepReceivableAllowance: {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "wertberichtigung"}},
	domain.ClosingStepWorkInProgress:      {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "unfertige_leistungen"}},
	domain.ClosingStepAccruals:            {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "abgrenzung"}},
	domain.ClosingStepProvisions:          {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "rueckstellung"}},
	domain.ClosingStepVatSettlement:       {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "ust_verrechnung"}},
	domain.ClosingStepInputTaxCorrection:  {Werkzeug: "nur_app"},
	domain.ClosingStepTaxProvision:        {Werkzeug: "abschlussbuchung", Parameter: map[string]string{"art": "steuerrueckstellung"}},
	domain.ClosingStepCheckRun:            {Werkzeug: "auswertung", Parameter: map[string]string{"art": "pruefbericht"}},
	domain.ClosingStepStatement:           {Werkzeug: "auswertung", Parameter: map[string]string{"art": "bilanz"}},
	domain.ClosingStepAdoption:            {Werkzeug: "nur_app"},
	domain.ClosingStepDisclosure:          {Werkzeug: "ebilanz_exportieren"},
	domain.ClosingStepAppropriation:       {Werkzeug: "ergebnisverwendung_buchen", Parameter: map[string]string{"nur_vorschau": "true"}},
}

var stepState = map[domain.ClosingStepState]string{
	domain.ClosingStepOpen: "offen", domain.ClosingStepDone: "erledigt", domain.ClosingStepSkipped: "uebersprungen",
}

// --- abschlussbuchung -------------------------------------------------------

type abgrenzungIn struct {
	Art            AbgrenzungArt `json:"art"`
	Text           string        `json:"text"`
	Gesamtbetrag   Betrag        `json:"gesamtbetrag"`
	Von            Datum         `json:"von" jsonschema:"Beginn des Leistungszeitraums"`
	Bis            Datum         `json:"bis" jsonschema:"Ende des Leistungszeitraums"`
	Konto          Konto         `json:"konto" jsonschema:"Aufwands- oder Ertragskonto der ursprünglichen Buchung"`
	QuellBuchungID uint          `json:"quell_buchung_id,omitempty"`
	Abzugrenzen    Betrag        `json:"abzugrenzen,omitempty" jsonschema:"abweichender Betrag; ohne Angabe zeitanteilig"`
}

type rueckstellungIn struct {
	Art           RueckstellungArt `json:"art"`
	Text          string           `json:"text"`
	Betrag        Betrag           `json:"betrag" jsonschema:"Erfüllungsbetrag"`
	ErwartetAm    Datum            `json:"erwartet_am" jsonschema:"voraussichtliche Inanspruchnahme"`
	Grund         string           `json:"grund" jsonschema:"Rechtsgrund und Schätzgrundlage"`
	Aufwandskonto Konto            `json:"aufwandskonto,omitempty"`
}

type steuerrueckstellungIn struct {
	Hebesatz            int64  `json:"hebesatz,omitempty" jsonschema:"Hebesatz der Gemeinde in Prozent; ohne Angabe aus den Abschluss-Einstellungen"`
	VerlustvortragKSt   Betrag `json:"verlustvortrag_kst,omitempty"`
	VerlustvortragGewSt Betrag `json:"verlustvortrag_gewst,omitempty"`
	Korrekturen         Betrag `json:"korrekturen,omitempty" jsonschema:"Zurechnungen außerhalb der Bücher, z. B. verdeckte Gewinnausschüttung; negativ für Abzüge"`
	Hinzurechnungen     Betrag `json:"hinzurechnungen,omitempty" jsonschema:"§ 8 GewStG"`
	Kuerzungen          Betrag `json:"kuerzungen,omitempty" jsonschema:"§ 9 GewStG"`
	KStBetrag           Betrag `json:"kst_betrag,omitempty" jsonschema:"abweichender Betrag der KSt-Rückstellung (mit SolZ)"`
	GewStBetrag         Betrag `json:"gewst_betrag,omitempty" jsonschema:"abweichender Betrag der GewSt-Rückstellung"`
	Grund               string `json:"grund,omitempty"`
}

type einzelwertberichtigungIn struct {
	BuchungID uint   `json:"buchung_id" jsonschema:"offene Forderung"`
	Prozent   int64  `json:"prozent" jsonschema:"erwarteter Ausfall in Prozent des Nettobetrags"`
	Grund     string `json:"grund"`
}

type wertberichtigungIn struct {
	Einzeln          []einzelwertberichtigungIn `json:"einzeln,omitempty"`
	PauschalPromille int64                      `json:"pauschal_promille,omitempty" jsonschema:"Pauschalwertberichtigung auf die übrigen Forderungen in Promille"`
	Grund            string                     `json:"grund"`
}

type projektIn struct {
	Name         string `json:"name"`
	KontaktID    uint   `json:"kontakt_id,omitempty"`
	Einzelkosten Betrag `json:"einzelkosten"`
	Gemeinkosten Betrag `json:"gemeinkosten,omitempty"`
	Auftragswert Betrag `json:"auftragswert,omitempty" jsonschema:"für die verlustfreie Bewertung"`
	Restkosten   Betrag `json:"restkosten,omitempty" jsonschema:"erwartete Kosten bis zur Fertigstellung"`
	Anzahlungen  Betrag `json:"anzahlungen,omitempty" jsonschema:"erhaltene Anzahlungen netto"`
	Notiz        string `json:"notiz,omitempty"`
}

type unfertigeIn struct {
	Projekte            []projektIn `json:"projekte"`
	AnzahlungenAbsetzen bool        `json:"anzahlungen_absetzen,omitempty" jsonschema:"erhaltene Anzahlungen offen absetzen (§ 268 Abs. 5 Satz 2 HGB)"`
	Grund               string      `json:"grund"`
}

type aktivierenIn struct {
	BuchungID           uint         `json:"buchung_id" jsonschema:"Buchung aus aktivierungskandidaten (anlagen)"`
	Bezeichnung         string       `json:"bezeichnung"`
	Klasse              AnlageKlasse `json:"klasse,omitempty" jsonschema:"Vorgabe aus dem Anlagekonto"`
	Methode             AfaMethode   `json:"methode,omitempty" jsonschema:"Vorgabe linear, bei nicht abnutzbaren Konten keine"`
	NutzungsdauerMonate int          `json:"nutzungsdauer_monate,omitempty" jsonschema:"nach AfA-Tabelle"`
	NutzungsdauerGrund  string       `json:"nutzungsdauer_grund,omitempty" jsonschema:"bei Abweichung von der AfA-Tabelle"`
	InBetrieb           Datum        `json:"in_betrieb,omitempty"`
}

type abschlussbuchungIn struct {
	KeyedWriteArgs
	Art                 AbschlussArt           `json:"art"`
	Geschaeftsjahr      int                    `json:"geschaeftsjahr,omitempty"`
	NurVorschau         bool                   `json:"nur_vorschau,omitempty" jsonschema:"rechnen, nicht buchen"`
	Buchungsdatum       Datum                  `json:"buchungsdatum,omitempty" jsonschema:"afa_lauf: Vorgabe Bilanzstichtag"`
	AnlageIDs           []uint                 `json:"anlage_ids,omitempty" jsonschema:"afa_lauf: nur diese Anlagen"`
	Abgrenzung          *abgrenzungIn          `json:"abgrenzung,omitempty"`
	Rueckstellung       *rueckstellungIn       `json:"rueckstellung,omitempty"`
	Steuerrueckstellung *steuerrueckstellungIn `json:"steuerrueckstellung,omitempty"`
	Wertberichtigung    *wertberichtigungIn    `json:"wertberichtigung,omitempty"`
	UnfertigeLeistungen *unfertigeIn           `json:"unfertige_leistungen,omitempty"`
	Anlage              *aktivierenIn          `json:"anlage,omitempty"`
}

func (in abschlussbuchungIn) preview() bool { return in.NurVorschau }

type abschlussbuchungOut struct {
	Wiederholung
	Art          string          `json:"art"`
	Vorschau     bool            `json:"vorschau"`
	BuchungIDs   []uint          `json:"buchung_ids"`
	Zeilen       []buchungsZeile `json:"zeilen"`
	Betrag       *Betrag         `json:"betrag,omitempty"`
	Erlaeuterung string          `json:"erlaeuterung,omitempty"`
	Hinweise     []string        `json:"hinweise"`
	AnlageID     *uint           `json:"anlage_id,omitempty"`
}

func (o abschlussbuchungOut) Ref() string {
	return fmt.Sprintf("abschluss:%s:%v", o.Art, o.BuchungIDs)
}

func registerAbschluss(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "anlagen",
		Title: "Anlagen",
		Description: "Anlagenverzeichnis mit AK, Buchwert, gebuchter und noch fälliger AfA des Geschäftsjahres, " +
			"dazu Buchungen auf Anlagekonten ohne Anlagegut (aktivierungskandidaten). Buchen über abschlussbuchung " +
			"(afa_lauf, anlage_aktivieren).",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in anlagenIn) (anlagenOut, error) {
		out := anlagenOut{Anlagen: []anlage{}, Kandidaten: []kandidat{}}
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return out, err
		}
		class, err := pick("klasse", in.Klasse, assetClasses)
		if err != nil {
			return out, err
		}
		assets, err := s.Assets.List(ctx, year, class)
		if err != nil {
			return out, err
		}
		rows := make([]anlage, 0, len(assets))
		var due domain.Cents
		for _, a := range assets {
			due += a.DueAmount
			rows = append(rows, anlage{ID: a.ID, Inventarnr: a.InventoryNumber, Bezeichnung: a.Name,
				Klasse: label(assetClasses, a.Class), Konto: a.Account, Angeschafft: a.AcquisitionDate, AK: betrag(a.Cost),
				Buchwert: betrag(a.BookValue), AfAJahr: betrag(a.YearAmount), AfAFaellig: betrag(a.DueAmount),
				Methode: label(depreciationMethods, a.Method), Nutzungsdauer: a.UsefulLifeMonths,
				Status: label(assetStatus, a.Status)})
		}
		out.AfAFaellig = betrag(due)
		candidates, err := s.Assets.AcquisitionCandidates(ctx, year)
		if err != nil {
			return out, err
		}
		for _, k := range candidates {
			out.Kandidaten = append(out.Kandidaten, kandidat{BuchungID: k.EntryID, Nummer: k.EntryNumber, Datum: k.BookingDate,
				Text: k.Description, Konto: k.Account, Kontoname: k.AccountName, Betrag: betrag(k.Amount)})
		}
		out.Anlagen, out.Seitenende, err = Paginate(rows, in.Seite)
		return out, err
	})

	AddRead(tb, Spec{
		Name:  "abschluss_status",
		Title: "Stand des Jahresabschlusses",
		Description: "Die Schritte des Jahresabschlusses mit Stand (offen, erledigt, uebersprungen), Hinweis und " +
			"nächster Aktion: welches Werkzeug den Schritt vorbereitet oder bucht, oder nur_app (Aufstellen, " +
			"Feststellen, Offenlegen, Wertaufholung, § 15a). Zuerst den ersten offenen Schritt.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in abschlussStatusIn) (abschlussStatusOut, error) {
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return abschlussStatusOut{}, err
		}
		steps, err := s.ClosingSteps.Steps(ctx, year)
		if err != nil {
			return abschlussStatusOut{}, err
		}
		out := abschlussStatusOut{Geschaeftsjahr: year, Stichtag: steps.Cutoff, Offen: steps.OpenCount,
			Erledigt: steps.DoneCount, Uebersprungen: steps.SkippedCount, Schritte: []abschlussSchritt{}}
		if fy, err := s.Closing.YearOf(ctx, year); err == nil && fy != nil {
			out.Stand = yearStand[fy.Status]
		}
		for _, st := range steps.Steps {
			action := stepActions[st.Key]
			if action.Werkzeug == "" {
				action.Werkzeug = "nur_app"
			}
			if action.Parameter != nil {
				params := map[string]string{"geschaeftsjahr": fmt.Sprint(year)}
				for k, v := range action.Parameter {
					params[k] = v
				}
				action.Parameter = params
			}
			step := abschlussSchritt{Schluessel: string(st.Key), Nr: st.Order, Titel: st.Label, Stand: stepState[st.State],
				Grund: st.Reason, Detail: st.Detail, Hinweis: st.Hint, Automatisch: st.Automatic, NaechsteAktion: action}
			if step.Stand == "" {
				step.Stand = "offen"
			}
			if step.Stand == "offen" && out.NaechsteAktion == nil {
				a := action
				out.NaechsteAktion = &a
			}
			out.Schritte = append(out.Schritte, step)
		}
		return out, nil
	})

	AddWritePrepared(tb, Spec{
		Name:  "abschlussbuchung",
		Title: "Abschlussbuchung",
		Description: "Bucht einen Schritt des Jahresabschlusses: afa_lauf, abgrenzung, rueckstellung, ust_verrechnung, " +
			"steuerrueckstellung, ug_ruecklage, wertberichtigung, unfertige_leistungen, fremdwaehrung oder " +
			"anlage_aktivieren, jeweils mit seinen Angaben im gleichnamigen Feld. Erst mit nur_vorschau rechnen, " +
			"dem Menschen Buchungssatz und Erläuterung vorlegen, nach seinem Ja buchen. Welcher Schritt offen ist, " +
			"zeigt abschluss_status.",
	}, func(ctx context.Context, c *Call, t *core.Tenant, in abschlussbuchungIn) error {
		if in.Art != "fremdwaehrung" {
			return nil
		}
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return err
		}
		return t.PrefetchValuationRates(ctx, year)
	}, func(ctx context.Context, c *Call, s core.AgentServices, in abschlussbuchungIn) (abschlussbuchungOut, error) {
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return abschlussbuchungOut{}, err
		}
		out := abschlussbuchungOut{Art: string(in.Art), Vorschau: in.NurVorschau, BuchungIDs: []uint{}, Zeilen: []buchungsZeile{},
			Hinweise: []string{}}
		run, ok := closingRuns[string(in.Art)]
		if !ok {
			return out, inputError("art", "unbekannte Art %q", in.Art)
		}
		if err := run(ctx, s, year, in, &out); err != nil {
			return out, err
		}
		return out, nil
	})
}

type closingRun func(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error

var closingRuns = map[string]closingRun{
	"afa_lauf":             runDepreciation,
	"abgrenzung":           runAccrual,
	"rueckstellung":        runProvision,
	"ust_verrechnung":      runVatSettlement,
	"steuerrueckstellung":  runTaxProvision,
	"ug_ruecklage":         runLegalReserve,
	"wertberichtigung":     runAllowance,
	"unfertige_leistungen": runWorkInProgress,
	"fremdwaehrung":        runCurrency,
	"anlage_aktivieren":    runActivate,
}

func addEntry(out *abschlussbuchungOut, e *domain.JournalEntry) {
	if e == nil {
		return
	}
	out.BuchungIDs = append(out.BuchungIDs, e.ID)
	out.Zeilen = append(out.Zeilen, zeilenOf(e.Lines)...)
}

func runDepreciation(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	date, err := in.Buchungsdatum.check("buchungsdatum")
	if err != nil {
		return err
	}
	if in.NurVorschau {
		due, err := s.Assets.PendingDepreciation(ctx, year)
		if err != nil {
			return err
		}
		var total domain.Cents
		for _, d := range due {
			total += d.Due + d.SpecialDue
			out.Hinweise = append(out.Hinweise, fmt.Sprintf("%s %s: %s € auf %s", d.InventoryNumber, d.Name, d.Due+d.SpecialDue, d.ExpenseAccount))
		}
		out.Betrag = optBetrag(total)
		return nil
	}
	res, err := s.Assets.BookDepreciation(ctx, service.BookDepreciationRequest{FiscalYear: year, BookingDate: date, AssetIDs: in.AnlageIDs})
	if err != nil {
		return err
	}
	for i := range res.Entries {
		addEntry(out, &res.Entries[i])
	}
	out.Betrag = optBetrag(res.Total)
	out.Hinweise = append(out.Hinweise, res.Skipped...)
	out.Hinweise = append(out.Hinweise, res.TaxOnly...)
	return nil
}

func runAccrual(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	a := in.Abgrenzung
	if a == nil {
		return inputError("abgrenzung", "die Angaben der Abgrenzung fehlen")
	}
	total, err := a.Gesamtbetrag.required("abgrenzung.gesamtbetrag")
	if err != nil {
		return err
	}
	deferred, err := a.Abzugrenzen.cents("abgrenzung.abzugrenzen")
	if err != nil {
		return err
	}
	kind, err := pick("abgrenzung.art", a.Art, accrualKinds)
	if err != nil {
		return err
	}
	from, err := a.Von.check("abgrenzung.von")
	if err != nil {
		return err
	}
	to, err := a.Bis.check("abgrenzung.bis")
	if err != nil {
		return err
	}
	req := service.AccrualRequest{FiscalYear: year, Kind: kind, SourceEntryID: a.QuellBuchungID,
		Text: a.Text, TotalAmount: total, StartDate: from, EndDate: to, Account: string(a.Konto),
		DeferredAmount: deferred}
	preview, err := s.Accruals.Preview(ctx, req)
	if err != nil {
		return err
	}
	out.Erlaeuterung, out.Hinweise = preview.Explanation, append(out.Hinweise, preview.Warnings...)
	if in.NurVorschau {
		out.Zeilen = zeilenOf(preview.Lines)
		return nil
	}
	accrual, err := s.Accruals.Book(ctx, req)
	if err != nil {
		return err
	}
	if accrual.FormationEntryID != nil {
		out.BuchungIDs = append(out.BuchungIDs, *accrual.FormationEntryID)
	}
	out.Zeilen = zeilenOf(preview.Lines)
	return nil
}

func runProvision(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	p := in.Rueckstellung
	if p == nil {
		return inputError("rueckstellung", "die Angaben der Rückstellung fehlen")
	}
	amount, err := p.Betrag.required("rueckstellung.betrag")
	if err != nil {
		return err
	}
	kind, err := pick("rueckstellung.art", p.Art, provisionKinds)
	if err != nil {
		return err
	}
	expected, err := p.ErwartetAm.check("rueckstellung.erwartet_am")
	if err != nil {
		return err
	}
	req := service.ProvisionRequest{FiscalYear: year, Kind: kind, Text: p.Text, Amount: amount,
		ExpectedOn: expected, Reason: p.Grund, ExpenseAccount: string(p.Aufwandskonto)}
	preview, err := s.Provisions.Preview(ctx, req)
	if err != nil {
		return err
	}
	out.Zeilen, out.Erlaeuterung = zeilenOf(preview.Lines), preview.Explanation
	out.Betrag = optBetrag(preview.Amount)
	out.Hinweise = append(out.Hinweise, preview.Findings...)
	if in.NurVorschau {
		return nil
	}
	provision, err := s.Provisions.BookFormation(ctx, req)
	if err != nil {
		return err
	}
	for _, m := range provision.Movements {
		if m.JournalEntryID != nil {
			out.BuchungIDs = append(out.BuchungIDs, *m.JournalEntryID)
		}
	}
	return nil
}

func runVatSettlement(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	preview, err := s.ClosingBookings.PreviewVatSettlement(ctx, year)
	if err != nil {
		return err
	}
	out.Zeilen, out.Erlaeuterung = zeilenOf(preview.Lines), preview.Explanation
	out.Betrag = optBetrag(preview.Payable - preview.Refund)
	if in.NurVorschau {
		return nil
	}
	entry, err := s.ClosingBookings.BookVatSettlement(ctx, year)
	if err != nil {
		return err
	}
	out.Zeilen = []buchungsZeile{}
	addEntry(out, entry)
	return nil
}

func runTaxProvision(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	opts := service.TaxProvisionOptions{}
	var req service.TaxProvisionRequest
	if p := in.Steuerrueckstellung; p != nil {
		var err error
		opts.TradeTaxRatePercent = p.Hebesatz
		for _, f := range []struct {
			name string
			in   Betrag
			out  *domain.Cents
		}{
			{"verlustvortrag_kst", p.VerlustvortragKSt, &opts.LossCarryforwardCorporate},
			{"verlustvortrag_gewst", p.VerlustvortragGewSt, &opts.LossCarryforwardTrade},
			{"korrekturen", p.Korrekturen, &opts.IncomeAdjustments},
			{"hinzurechnungen", p.Hinzurechnungen, &opts.TradeAdditions},
			{"kuerzungen", p.Kuerzungen, &opts.TradeDeductions},
			{"kst_betrag", p.KStBetrag, &req.IncomeProvision},
			{"gewst_betrag", p.GewStBetrag, &req.TradeProvision},
		} {
			if *f.out, err = f.in.cents("steuerrueckstellung." + f.name); err != nil {
				return err
			}
		}
		req.Reason = p.Grund
	}
	preview, err := s.ClosingBookings.PreviewTaxProvision(ctx, year, opts)
	if err != nil {
		return err
	}
	out.Zeilen, out.Erlaeuterung = zeilenOf(preview.Lines), preview.Explanation
	out.Betrag = optBetrag(preview.IncomeProvision + preview.TradeProvision)
	out.Hinweise = append(out.Hinweise, preview.Notes...)
	if preview.Warning != "" {
		out.Hinweise = append(out.Hinweise, preview.Warning)
	}
	if in.NurVorschau {
		return nil
	}
	req.FiscalYear, req.Options = year, opts
	provisions, err := s.ClosingBookings.BookTaxProvision(ctx, req)
	if err != nil {
		return err
	}
	seen := map[uint]bool{}
	for _, p := range provisions {
		for _, m := range p.Movements {
			if m.JournalEntryID != nil && !seen[*m.JournalEntryID] {
				seen[*m.JournalEntryID] = true
				out.BuchungIDs = append(out.BuchungIDs, *m.JournalEntryID)
			}
		}
	}
	return nil
}

func runLegalReserve(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	state, err := s.Closing.LegalReserve(ctx, year)
	if err != nil {
		return err
	}
	if !state.Applies {
		return &ToolError{Code: CodeRule, Nachricht: "Die gesetzliche Rücklage nach § 5a Abs. 3 GmbHG gilt nur für die UG (haftungsbeschränkt)."}
	}
	out.Betrag = optBetrag(state.Difference)
	out.Erlaeuterung = fmt.Sprintf("Einzustellen sind %s € (ein Viertel des um den Verlustvortrag geminderten Jahresüberschusses), gebucht %s €.",
		state.Required, state.Booked)
	if in.NurVorschau {
		return nil
	}
	entry, err := s.Closing.BookLegalReserve(ctx, year)
	if err != nil {
		return err
	}
	addEntry(out, entry)
	return nil
}

func runAllowance(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	w := in.Wertberichtigung
	if w == nil {
		return inputError("wertberichtigung", "die Angaben der Wertberichtigung fehlen")
	}
	req := service.ReceivableAllowanceRequest{FiscalYear: year, GeneralPermille: w.PauschalPromille, Reason: w.Grund}
	for _, e := range w.Einzeln {
		req.Specific = append(req.Specific, accounting.SpecificAllowance{EntryID: e.BuchungID, Percent: e.Prozent, Reason: e.Grund})
	}
	preview, err := s.ClosingBookings.PreviewReceivableAllowance(ctx, req)
	if err != nil {
		return err
	}
	out.Zeilen, out.Erlaeuterung = zeilenOf(preview.Lines), preview.Explanation
	out.Hinweise = append(out.Hinweise, preview.Notes...)
	if in.NurVorschau {
		return nil
	}
	entry, err := s.ClosingBookings.BookReceivableAllowance(ctx, req)
	if err != nil {
		return err
	}
	out.Zeilen = []buchungsZeile{}
	addEntry(out, entry)
	return nil
}

func runWorkInProgress(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	u := in.UnfertigeLeistungen
	if u == nil || len(u.Projekte) == 0 {
		return inputError("unfertige_leistungen", "die Projekte fehlen")
	}
	req := service.WorkInProgressRequest{FiscalYear: year, DeductAdvances: u.AnzahlungenAbsetzen, Reason: u.Grund}
	for i, p := range u.Projekte {
		field := fmt.Sprintf("unfertige_leistungen.projekte[%d]", i)
		project := accounting.WorkInProgressProject{Name: p.Name, ContactID: p.KontaktID, Note: p.Notiz}
		for _, f := range []struct {
			name string
			in   Betrag
			out  *domain.Cents
		}{
			{"einzelkosten", p.Einzelkosten, &project.DirectCosts}, {"gemeinkosten", p.Gemeinkosten, &project.Overheads},
			{"auftragswert", p.Auftragswert, &project.ContractValue}, {"restkosten", p.Restkosten, &project.CostsToComplete},
			{"anzahlungen", p.Anzahlungen, &project.AdvancesReceived},
		} {
			var err error
			if *f.out, err = f.in.cents(field + "." + f.name); err != nil {
				return err
			}
		}
		req.Projects = append(req.Projects, project)
	}
	preview, err := s.ClosingBookings.PreviewWorkInProgress(ctx, req)
	if err != nil {
		return err
	}
	out.Zeilen, out.Erlaeuterung = zeilenOf(preview.Lines), preview.Explanation
	out.Hinweise = append(out.Hinweise, preview.Notes...)
	if in.NurVorschau {
		return nil
	}
	entry, err := s.ClosingBookings.BookWorkInProgress(ctx, req)
	if err != nil {
		return err
	}
	out.Zeilen = []buchungsZeile{}
	addEntry(out, entry)
	return nil
}

func runCurrency(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	var v *service.ForeignCurrencyValuation
	var err error
	if in.NurVorschau {
		v, err = s.Currency.PreviewCurrencyValuation(ctx, year)
	} else {
		v, err = s.Currency.BookCurrencyValuation(ctx, year)
	}
	if err != nil {
		return err
	}
	out.Betrag = optBetrag(v.TotalGain - v.TotalLoss)
	out.Erlaeuterung = v.Note
	if v.EntryID != 0 {
		out.BuchungIDs = append(out.BuchungIDs, v.EntryID)
		out.Hinweise = append(out.Hinweise, "Bewertung gebucht als "+v.EntryNumber+"; die Auflösung kommt mit dem Saldenvortrag.")
	}
	return nil
}

func runActivate(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	a := in.Anlage
	if a == nil || a.BuchungID == 0 || strings.TrimSpace(a.Bezeichnung) == "" {
		return inputError("anlage", "buchung_id und bezeichnung fehlen")
	}
	candidates, err := s.Assets.AcquisitionCandidates(ctx, year)
	if err != nil {
		return err
	}
	var found *service.AcquisitionCandidate
	for i := range candidates {
		if candidates[i].EntryID == a.BuchungID {
			found = &candidates[i]
		}
	}
	if found == nil {
		return &ToolError{Code: CodeNotFound, Nachricht: fmt.Sprintf("Buchung %d ist kein Aktivierungskandidat im Geschäftsjahr %d", a.BuchungID, year),
			NaechsterSchritt: "Die Kandidaten nennt anlagen.", Werkzeug: "anlagen"}
	}
	out.Betrag = optBetrag(found.Amount)
	out.Erlaeuterung = fmt.Sprintf("%s aus Buchung %s auf Konto %s", a.Bezeichnung, found.EntryNumber, found.Account)
	if in.NurVorschau {
		return nil
	}
	inService, err := a.InBetrieb.check("anlage.in_betrieb")
	if err != nil {
		return err
	}
	// Klasse, Aufwandskonto und ob das Gut abnutzbar ist, sagt das Anlagekonto.
	account, known := accounting.LookupAssetAccount(found.Account)
	class, err := pick("anlage.klasse", a.Klasse, assetClasses)
	if err != nil {
		return err
	}
	if class == "" {
		class = account.Class
	}
	if class == "" {
		class = domain.AssetClassTangible
	}
	method, err := pick("anlage.methode", a.Methode, depreciationMethods)
	if err != nil {
		return err
	}
	if method == "" {
		method = domain.DepreciationLinear
		if known && !account.Depreciable {
			method = domain.DepreciationNone
		}
	}
	entryID := found.EntryID
	asset := &domain.FixedAsset{Name: a.Bezeichnung, Class: class, Account: found.Account, AcquisitionDate: found.BookingDate,
		AcquisitionCost: found.Amount, InServiceDate: inService, Method: method,
		DepreciationAccount: account.DepreciationAccount,
		UsefulLifeMonths:    a.NutzungsdauerMonate, UsefulLifeReason: a.NutzungsdauerGrund, ContactID: found.ContactID,
		AcquisitionEntryID: &entryID, InputTaxAmount: found.InputTaxAmount, InputTaxPermille: found.InputTaxPermille}
	saved, err := s.Assets.Save(ctx, asset)
	if err != nil {
		return err
	}
	out.AnlageID = &saved.ID
	return nil
}
