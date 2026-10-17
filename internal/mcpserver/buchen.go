package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- Eingangsbeleg buchen -----------------------------------------------------

type positionIn struct {
	Buchungsgruppe Buchungsgruppe `json:"buchungsgruppe"`
	Netto          Betrag         `json:"netto"`
	Satz           Satz           `json:"satz"`
	Text           string         `json:"text,omitempty"`
}

type belegBuchenIn struct {
	WriteArgs
	BelegID       uint         `json:"beleg_id"`
	KontaktID     uint         `json:"kontakt_id" jsonschema:"Lieferant"`
	Belegdatum    Datum        `json:"belegdatum"`
	Buchungsdatum Datum        `json:"buchungsdatum,omitempty" jsonschema:"Vorgabe Belegdatum"`
	LeistungVon   Datum        `json:"leistung_von"`
	LeistungBis   Datum        `json:"leistung_bis,omitempty" jsonschema:"Vorgabe leistung_von"`
	Steuerfall    Steuerfall   `json:"steuerfall"`
	Positionen    []positionIn `json:"positionen" jsonschema:"Netto je Steuersatz und Buchungsgruppe, 1 bis 20"`
	Bezahlung     string       `json:"bezahlung" jsonschema:"offen (Verbindlichkeit) oder kreditkarte (Firmenkreditkarte 3610)"`
	Beschreibung  string       `json:"beschreibung,omitempty"`
	NurVorschau   bool         `json:"nur_vorschau,omitempty" jsonschema:"rechnen und einstufen, nicht buchen"`
}

type geschenkIn struct {
	KontaktID uint   `json:"kontakt_id,omitempty"`
	Name      string `json:"name,omitempty" jsonschema:"Empfänger, wenn er kein Kontakt ist"`
	Anlass    string `json:"anlass"`
}

type positionRueckfrageIn struct {
	Buchungsgruppe          Buchungsgruppe `json:"buchungsgruppe,omitempty"`
	Konto                   Konto          `json:"konto,omitempty" jsonschema:"statt einer Gruppe direkt ein Aufwands- oder Bestandskonto"`
	Netto                   Betrag         `json:"netto" jsonschema:"bei Fremdwährung in der Fremdwährung"`
	Satz                    Satz           `json:"satz"`
	Text                    string         `json:"text,omitempty"`
	VorsteueranteilPromille int            `json:"vorsteueranteil_promille,omitempty" jsonschema:"abziehbarer Anteil der Vorsteuer in Promille bei gemischter Nutzung; 0 heißt voll"`
	VorsteueranteilGrund    string         `json:"vorsteueranteil_grund,omitempty"`
	Geschenk                *geschenkIn    `json:"geschenk,omitempty" jsonschema:"Empfänger, wo die Gruppe ihn verlangt (§ 4 Abs. 7 EStG)"`
}

type bewirtungIn struct {
	Ort        string `json:"ort"`
	Tag        Datum  `json:"tag"`
	Teilnehmer string `json:"teilnehmer"`
	Anlass     string `json:"anlass"`
}

type belegBuchenRueckfrageIn struct {
	WriteArgs
	BelegID                 uint                   `json:"beleg_id"`
	KontaktID               uint                   `json:"kontakt_id" jsonschema:"Lieferant (Aussteller)"`
	Belegdatum              Datum                  `json:"belegdatum"`
	Buchungsdatum           Datum                  `json:"buchungsdatum,omitempty"`
	LeistungVon             Datum                  `json:"leistung_von"`
	LeistungBis             Datum                  `json:"leistung_bis,omitempty"`
	Steuerfall              Steuerfall             `json:"steuerfall"`
	Positionen              []positionRueckfrageIn `json:"positionen"`
	Bezahlung               string                 `json:"bezahlung" jsonschema:"offen, bezahlt (mit zahlungskonto), kreditkarte oder auslage (ein Gesellschafter hat privat bezahlt)"`
	Zahlungskonto           Konto                  `json:"zahlungskonto,omitempty" jsonschema:"Bank oder Kasse bei bezahlt"`
	GesellschafterKontaktID uint                   `json:"gesellschafter_kontakt_id,omitempty" jsonschema:"bei auslage: wem erstattet wird"`
	Waehrung                string                 `json:"waehrung,omitempty" jsonschema:"ISO-Code, Vorgabe EUR"`
	Fremdbetrag             Betrag                 `json:"fremdbetrag,omitempty" jsonschema:"Endsumme in der Fremdwährung als Kontrolle"`
	Bewirtung               *bewirtungIn           `json:"bewirtung,omitempty"`
	RueckstellungID         uint                   `json:"rueckstellung_id,omitempty"`
	AnzahlungFuer           string                 `json:"anzahlung_fuer,omitempty" jsonschema:"inventory, tangible oder intangible: der Beleg ist eine geleistete Anzahlung"`
	VerrechneteAnzahlungen  []uint                 `json:"verrechnete_anzahlungen,omitempty"`
	Leistungsnachweis       string                 `json:"leistungsnachweis,omitempty"`
	LeistungsnachweisAm     Datum                  `json:"leistungsnachweis_am,omitempty"`
	UebersteuerungGrund     string                 `json:"uebersteuerung_grund,omitempty" jsonschema:"Begründung, mit der ein blockierender Befund der Rechnungsprüfung übergangen wird"`
	Beschreibung            string                 `json:"beschreibung,omitempty"`
	NurVorschau             bool                   `json:"nur_vorschau,omitempty"`
}

var settlements = map[string]service.SettlementKind{
	"offen": service.SettlementOpen, "bezahlt": service.SettlementPaid, "kreditkarte": service.SettlementCreditCard,
	"auslage": service.SettlementShareholderExpense,
}

func (in belegBuchenIn) request() (service.ReceiptRequest, error) {
	req, err := receiptHead(in.BelegID, in.KontaktID, in.Belegdatum, in.Buchungsdatum, in.LeistungVon, in.LeistungBis,
		in.Steuerfall, in.Beschreibung)
	if err != nil {
		return req, err
	}
	switch in.Bezahlung {
	case "offen", "kreditkarte":
		req.Settlement = settlements[in.Bezahlung]
	default:
		return req, inputError("bezahlung", "direkt nur offen oder kreditkarte, nicht %q; sonst beleg_buchen_mit_rueckfrage", in.Bezahlung)
	}
	if len(in.Positionen) == 0 || len(in.Positionen) > 20 {
		return req, inputError("positionen", "1 bis 20 Positionen, nicht %d", len(in.Positionen))
	}
	for i, p := range in.Positionen {
		net, err := p.Netto.required(fmt.Sprintf("positionen[%d].netto", i))
		if err != nil {
			return req, err
		}
		req.Positions = append(req.Positions, service.ReceiptPosition{
			PostingGroup: string(p.Buchungsgruppe), Net: net, TaxRate: p.Satz.rate(), Text: p.Text})
	}
	return req, nil
}

func (in belegBuchenRueckfrageIn) request() (service.ReceiptRequest, error) {
	req, err := receiptHead(in.BelegID, in.KontaktID, in.Belegdatum, in.Buchungsdatum, in.LeistungVon, in.LeistungBis,
		in.Steuerfall, in.Beschreibung)
	if err != nil {
		return req, err
	}
	kind, ok := settlements[in.Bezahlung]
	if !ok {
		return req, inputError("bezahlung", "offen, bezahlt, kreditkarte oder auslage, nicht %q", in.Bezahlung)
	}
	req.Settlement, req.PaymentAccount = kind, string(in.Zahlungskonto)
	if kind == service.SettlementShareholderExpense {
		if in.GesellschafterKontaktID == 0 {
			return req, inputError("gesellschafter_kontakt_id", "bei einer Auslage fehlt, wem erstattet wird")
		}
		req.CreditorContactID = in.GesellschafterKontaktID
	}
	if len(in.Positionen) == 0 || len(in.Positionen) > 20 {
		return req, inputError("positionen", "1 bis 20 Positionen, nicht %d", len(in.Positionen))
	}
	for i, p := range in.Positionen {
		net, err := p.Netto.required(fmt.Sprintf("positionen[%d].netto", i))
		if err != nil {
			return req, err
		}
		pos := service.ReceiptPosition{PostingGroup: string(p.Buchungsgruppe), Account: string(p.Konto), Net: net,
			TaxRate: p.Satz.rate(), Text: p.Text, InputTaxShare: p.VorsteueranteilPromille,
			InputTaxShareReason: p.VorsteueranteilGrund}
		if p.Geschenk != nil {
			pos.Gift = &service.GiftInput{ContactID: p.Geschenk.KontaktID, Name: p.Geschenk.Name, Occasion: p.Geschenk.Anlass}
		}
		req.Positions = append(req.Positions, pos)
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(in.Waehrung))
	if req.ForeignAmount, err = in.Fremdbetrag.cents("fremdbetrag"); err != nil {
		return req, err
	}
	if in.Bewirtung != nil {
		day, err := in.Bewirtung.Tag.check("bewirtung.tag")
		if err != nil {
			return req, err
		}
		req.Entertainment = &domain.EntertainmentDetail{Place: in.Bewirtung.Ort, Day: day,
			Participants: in.Bewirtung.Teilnehmer, Occasion: in.Bewirtung.Anlass}
	}
	req.ProvisionID = in.RueckstellungID
	req.AdvanceTarget = accounting.AdvanceTarget(in.AnzahlungFuer)
	req.SettledAdvanceIDs = in.VerrechneteAnzahlungen
	req.ServiceProof = in.Leistungsnachweis
	if req.ServiceProofAt, err = in.LeistungsnachweisAm.check("leistungsnachweis_am"); err != nil {
		return req, err
	}
	req.OverrideReason = in.UebersteuerungGrund
	return req, nil
}

func receiptHead(receiptID, contactID uint, docDate, bookDate, from, to Datum, treatment Steuerfall, text string) (service.ReceiptRequest, error) {
	req := service.ReceiptRequest{ReceiptID: receiptID, ContactID: contactID, TaxTreatment: domain.TaxTreatment(treatment),
		Description: strings.TrimSpace(text)}
	var err error
	if req.DocumentDate, err = docDate.check("belegdatum"); err != nil {
		return req, err
	}
	if req.DocumentDate == "" {
		return req, inputError("belegdatum", "das Belegdatum fehlt")
	}
	if req.BookingDate, err = bookDate.check("buchungsdatum"); err != nil {
		return req, err
	}
	if req.BookingDate == "" {
		req.BookingDate = req.DocumentDate
	}
	if req.ServiceDateFrom, err = from.check("leistung_von"); err != nil {
		return req, err
	}
	if req.ServiceDateFrom == "" {
		return req, inputError("leistung_von", "der Leistungszeitpunkt fehlt (§ 14 Abs. 4 Nr. 6 UStG)")
	}
	if req.ServiceDateTo, err = to.check("leistung_bis"); err != nil {
		return req, err
	}
	if req.ServiceDateTo == "" {
		req.ServiceDateTo = req.ServiceDateFrom
	}
	return req, nil
}

// prefetchRate holt bei Fremdwährung den Kurs vor der Schreibsperre.
func prefetchRate(ctx context.Context, t *core.Tenant, currency, date string) error {
	if currency == "" || currency == "EUR" {
		return nil
	}
	return t.PrefetchRate(ctx, currency, date)
}

// --- buchung_erfassen -------------------------------------------------------

type zeileIn struct {
	Konto            Konto  `json:"konto"`
	Seite            string `json:"seite" jsonschema:"S (Soll) oder H (Haben)"`
	Betrag           Betrag `json:"betrag"`
	Steuerschluessel string `json:"steuerschluessel,omitempty" jsonschema:"an Steuerzeilen, z. B. VST19"`
	Steuerbasis      Betrag `json:"steuerbasis,omitempty" jsonschema:"Bemessungsgrundlage an Steuerzeilen"`
	KontaktID        uint   `json:"kontakt_id,omitempty" jsonschema:"an Personenkonten"`
	Text             string `json:"text,omitempty"`
}

type eigenbelegIn struct {
	Grund      string `json:"grund" jsonschema:"warum es keinen fremden Beleg gibt"`
	Belegdatum Datum  `json:"belegdatum,omitempty"`
	Brutto     Betrag `json:"brutto"`
	Steuer     Betrag `json:"steuer,omitempty"`
}

type buchungErfassenIn struct {
	WriteArgs
	Buchungsdatum    Datum         `json:"buchungsdatum"`
	Belegdatum       Datum         `json:"belegdatum,omitempty" jsonschema:"Vorgabe Buchungsdatum"`
	LeistungVon      Datum         `json:"leistung_von,omitempty"`
	LeistungBis      Datum         `json:"leistung_bis,omitempty"`
	Beschreibung     string        `json:"beschreibung" jsonschema:"so, dass ein Prüfer ohne Rückfrage versteht, was gebucht wurde"`
	Steuerfall       Steuerfall    `json:"steuerfall" jsonschema:"not_taxable für Umbuchungen und Vorgänge ohne Umsatzsteuer; domestic verlangt eine Steuerzeile"`
	Zeilen           []zeileIn     `json:"zeilen" jsonschema:"Soll und Haben müssen aufgehen"`
	BelegID          uint          `json:"beleg_id,omitempty" jsonschema:"abgelegter Beleg zum Vorgang"`
	Eigenbeleg       *eigenbelegIn `json:"eigenbeleg,omitempty" jsonschema:"statt beleg_id, wenn es keinen fremden Beleg gibt"`
	Abschlussperiode bool          `json:"abschlussperiode,omitempty" jsonschema:"Abschlussbuchung im 13. Monat zum Bilanzstichtag"`
}

func (in buchungErfassenIn) request() (service.ManualEntryRequest, error) {
	var req service.ManualEntryRequest
	e := domain.JournalEntry{Description: strings.TrimSpace(in.Beschreibung), ClosingPeriod: in.Abschlussperiode,
		TaxTreatment: domain.TaxTreatment(in.Steuerfall)}
	var err error
	if e.BookingDate, err = in.Buchungsdatum.check("buchungsdatum"); err != nil || e.BookingDate == "" {
		if err == nil {
			err = inputError("buchungsdatum", "das Buchungsdatum fehlt")
		}
		return req, err
	}
	if e.DocumentDate, err = in.Belegdatum.check("belegdatum"); err != nil {
		return req, err
	}
	if e.DocumentDate == "" {
		e.DocumentDate = e.BookingDate
	}
	if e.ServiceDateFrom, err = in.LeistungVon.check("leistung_von"); err != nil {
		return req, err
	}
	if e.ServiceDateFrom == "" {
		e.ServiceDateFrom = e.DocumentDate
	}
	if e.ServiceDateTo, err = in.LeistungBis.check("leistung_bis"); err != nil {
		return req, err
	}
	if e.ServiceDateTo == "" {
		e.ServiceDateTo = e.ServiceDateFrom
	}
	if e.Description == "" {
		return req, inputError("beschreibung", "die Beschreibung fehlt")
	}
	if len(in.Zeilen) < 2 {
		return req, inputError("zeilen", "eine Buchung hat mindestens zwei Zeilen")
	}
	for i, z := range in.Zeilen {
		field := fmt.Sprintf("zeilen[%d]", i)
		amount, err := z.Betrag.required(field + ".betrag")
		if err != nil {
			return req, err
		}
		base, err := z.Steuerbasis.cents(field + ".steuerbasis")
		if err != nil {
			return req, err
		}
		side := domain.Side(z.Seite)
		if side != domain.SideDebit && side != domain.SideCredit {
			return req, inputError(field+".seite", "S oder H, nicht %q", z.Seite)
		}
		line := domain.JournalLine{Position: i + 1, Side: side, Amount: amount, Account: string(z.Konto),
			TaxKey: z.Steuerschluessel, TaxBase: base, Text: z.Text}
		if z.KontaktID != 0 {
			id := z.KontaktID
			line.ContactID = &id
		}
		e.Lines = append(e.Lines, line)
	}
	req.Entry, req.ReceiptID = e, in.BelegID
	if in.Eigenbeleg != nil {
		gross, err := in.Eigenbeleg.Brutto.required("eigenbeleg.brutto")
		if err != nil {
			return req, err
		}
		tax, err := in.Eigenbeleg.Steuer.cents("eigenbeleg.steuer")
		if err != nil {
			return req, err
		}
		date, err := in.Eigenbeleg.Belegdatum.check("eigenbeleg.belegdatum")
		if err != nil {
			return req, err
		}
		if date == "" {
			date = e.DocumentDate
		}
		req.SelfIssued = &service.SelfIssuedReceiptRequest{DocumentDate: date, GrossAmount: gross, TaxAmount: tax,
			Reason: in.Eigenbeleg.Grund}
	}
	return req, nil
}

// --- buchung_stornieren -----------------------------------------------------

type neubuchungIn struct {
	Buchungsdatum Datum      `json:"buchungsdatum"`
	Belegdatum    Datum      `json:"belegdatum,omitempty"`
	Beschreibung  string     `json:"beschreibung"`
	Steuerfall    Steuerfall `json:"steuerfall"`
	Zeilen        []zeileIn  `json:"zeilen"`
}

type stornierenIn struct {
	WriteArgs
	BuchungID  uint          `json:"buchung_id"`
	Grund      string        `json:"grund" jsonschema:"warum storniert wird (GoBD Rz. 58)"`
	Datum      Datum         `json:"datum,omitempty" jsonschema:"Tag des Stornos; Vorgabe heute, muss im offenen Zeitraum liegen"`
	Neubuchung *neubuchungIn `json:"neubuchung,omitempty" jsonschema:"die richtige Buchung, die an die Stelle tritt"`
}

type stornierenOut struct {
	Wiederholung
	Storno     buchungKurz  `json:"storno"`
	Neubuchung *buchungKurz `json:"neubuchung,omitempty"`
	Hinweis    string       `json:"hinweis,omitempty"`
}

func (o stornierenOut) Ref() string { return fmt.Sprintf("buchung:%d", o.Storno.ID) }

// --- lohnjournal_buchen -----------------------------------------------------

type lohnjournalIn struct {
	WriteArgs
	Monat         string `json:"monat" jsonschema:"Abrechnungsmonat JJJJ-MM"`
	BelegID       uint   `json:"beleg_id" jsonschema:"das abgelegte Lohnjournal des Lohnbüros"`
	Buchungsdatum Datum  `json:"buchungsdatum,omitempty" jsonschema:"Vorgabe letzter Tag des Monats"`

	Gehaelter                 Betrag `json:"gehaelter,omitempty" jsonschema:"6020"`
	GfGehaelterGesellschafter Betrag `json:"gf_gehaelter_gesellschafter,omitempty" jsonschema:"6024"`
	GfGehaelter               Betrag `json:"gf_gehaelter,omitempty" jsonschema:"6027"`
	MinijobLoehne             Betrag `json:"minijob_loehne,omitempty" jsonschema:"6035"`

	AnSozialversicherung Betrag `json:"an_sozialversicherung,omitempty"`
	Lohnsteuer           Betrag `json:"lohnsteuer,omitempty"`
	Kirchensteuer        Betrag `json:"kirchensteuer,omitempty"`
	Soli                 Betrag `json:"soli,omitempty"`
	Netto                Betrag `json:"netto"`

	AgSozialversicherung      Betrag `json:"ag_sozialversicherung,omitempty"`
	Umlagen                   Betrag `json:"umlagen,omitempty" jsonschema:"U1, U2, Insolvenzgeldumlage"`
	MinijobAbgaben            Betrag `json:"minijob_abgaben,omitempty"`
	MinijobPauschsteuer       Betrag `json:"minijob_pauschsteuer,omitempty"`
	PauschaleLohnsteuer       Betrag `json:"pauschale_lohnsteuer,omitempty"`
	PauschaleLohnsteuerGf     Betrag `json:"pauschale_lohnsteuer_gf,omitempty"`
	NettoAufVerrechnungskonto bool   `json:"netto_auf_verrechnungskonto,omitempty" jsonschema:"Netto auf 3790 statt 3720"`
	Abschlussperiode          bool   `json:"abschlussperiode,omitempty"`
}

func (in lohnjournalIn) request() (service.PayrollRequest, error) {
	req := service.PayrollRequest{Month: in.Monat, ReceiptID: in.BelegID, NetToClearing: in.NettoAufVerrechnungskonto,
		ClosingPeriod: in.Abschlussperiode}
	var err error
	if req.BookingDate, err = in.Buchungsdatum.check("buchungsdatum"); err != nil {
		return req, err
	}
	fields := []struct {
		name string
		in   Betrag
		out  *domain.Cents
	}{
		{"gehaelter", in.Gehaelter, &req.Salaries},
		{"gf_gehaelter_gesellschafter", in.GfGehaelterGesellschafter, &req.ShareholderDirectorSalaries},
		{"gf_gehaelter", in.GfGehaelter, &req.DirectorSalaries},
		{"minijob_loehne", in.MinijobLoehne, &req.MinijobWages},
		{"an_sozialversicherung", in.AnSozialversicherung, &req.EmployeeSocialSecurity},
		{"lohnsteuer", in.Lohnsteuer, &req.WageTax},
		{"kirchensteuer", in.Kirchensteuer, &req.ChurchTax},
		{"soli", in.Soli, &req.SolidaritySurcharge},
		{"netto", in.Netto, &req.NetPay},
		{"ag_sozialversicherung", in.AgSozialversicherung, &req.EmployerSocialSecurity},
		{"umlagen", in.Umlagen, &req.Levies},
		{"minijob_abgaben", in.MinijobAbgaben, &req.MinijobContributions},
		{"minijob_pauschsteuer", in.MinijobPauschsteuer, &req.MinijobFlatTax},
		{"pauschale_lohnsteuer", in.PauschaleLohnsteuer, &req.FlatRateWageTax},
		{"pauschale_lohnsteuer_gf", in.PauschaleLohnsteuerGf, &req.ShareholderDirectorFlatRateTax},
	}
	for _, f := range fields {
		if *f.out, err = f.in.cents(f.name); err != nil {
			return req, err
		}
	}
	return req, nil
}

func registerBuchen(tb *Toolbox) {
	AddWrite(tb, Spec{
		Name:  "beleg_buchen",
		Title: "Beleg buchen (Routine)",
		Tier:  TierDirect, Idempotent: true,
		Description: "Bucht eine Eingangsrechnung ohne Rückfrage, wenn Buchfink sie als Routine einstuft: " +
			"E-Rechnung ohne Befund, Beträge centgenau wie in der E-Rechnung, Inland in Euro, Brutto bis zur " +
			"Direktgrenze, Lieferant mit vom Menschen bestätigter Standardgruppe und genug früheren Buchungen, " +
			"bezahlung offen oder kreditkarte, kein Dublettenverdacht. Sonst kommt freigabe_noetig mit Gründen und " +
			"es ist nichts gebucht; dann dem Menschen vorlegen und beleg_buchen_mit_rueckfrage nutzen. Mit " +
			"nur_vorschau wird gerechnet und eingestuft, nicht gebucht.",
	}, func(ctx context.Context, c *Call, s *core.Services, in belegBuchenIn) (buchungOut, error) {
		req, err := in.request()
		if err != nil {
			return buchungOut{}, err
		}
		verdict, err := policy.ClassifyReceiptRequest(ctx, s, req, today())
		if err != nil {
			return buchungOut{}, err
		}
		if in.NurVorschau {
			preview, err := s.Posting.PreviewIncomingReceipt(ctx, req)
			if err != nil {
				return buchungOut{}, err
			}
			v := vorschauOf(preview)
			return buchungOut{Vorschau: &v, Einstufung: string(verdict.Level)}, nil
		}
		if !verdict.Direct() {
			return buchungOut{}, errApproval(verdict, "beleg_buchen_mit_rueckfrage")
		}
		entry, err := s.Posting.PostIncomingReceipt(ctx, req)
		if err != nil {
			return buchungOut{}, err
		}
		return booked(entry, string(policy.Direct)), nil
	})

	AddWritePrepared(tb, Spec{
		Name:  "beleg_buchen_mit_rueckfrage",
		Title: "Beleg buchen (mit Rückfrage)",
		Tier:  TierConfirm,
		Description: "Bucht einen Eingangsbeleg in allen Fällen, die keine Routine sind: Foto oder PDF, " +
			"Abweichung von der E-Rechnung, Fremdwährung, Bewirtung, Geschenk, Vorsteueranteil, Rückstellung, " +
			"Anzahlung, Auslage eines Gesellschafters (bezahlung=auslage), bar bezahlt, Konto statt Gruppe, " +
			"begründete Übersteuerung eines Befunds. Vorher dem Menschen Buchungssatz und Gründe vorlegen und sein " +
			"Ja einholen; nur_vorschau rechnet ohne zu buchen. Die Standardgruppe des Lieferanten bestätigt das nicht.",
	}, func(ctx context.Context, c *Call, t *core.Tenant, in belegBuchenRueckfrageIn) error {
		if in.NurVorschau {
			return nil
		}
		return prefetchRate(ctx, t, strings.ToUpper(strings.TrimSpace(in.Waehrung)), string(in.Belegdatum))
	}, func(ctx context.Context, c *Call, s *core.Services, in belegBuchenRueckfrageIn) (buchungOut, error) {
		req, err := in.request()
		if err != nil {
			return buchungOut{}, err
		}
		if in.NurVorschau {
			preview, err := s.Posting.PreviewIncomingReceipt(ctx, req)
			if err != nil {
				return buchungOut{}, err
			}
			v := vorschauOf(preview)
			return buchungOut{Vorschau: &v, Einstufung: string(policy.Confirm)}, nil
		}
		entry, err := s.Posting.PostIncomingReceipt(ctx, req)
		if err != nil {
			return buchungOut{}, err
		}
		return booked(entry, string(policy.Confirm)), nil
	})

	AddWrite(tb, Spec{
		Name:  "buchung_erfassen",
		Title: "Buchung erfassen",
		Tier:  TierConfirm,
		Description: "Handbuchung mit Soll- und Habenzeilen zu einem abgelegten Beleg (beleg_id) oder mit Eigenbeleg " +
			"(grund, brutto). Für Umbuchungen, Gesellschafterdarlehen (Zugang, Tilgung), Kreditkartenabrechnung " +
			"ohne Bankumsatz und Abschlussbuchungen im 13. Monat (abschlussperiode). Eingangsrechnungen über " +
			"beleg_buchen, Bankumsätze über bankumsatz_buchen. Vorher im Chat bestätigen lassen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in buchungErfassenIn) (buchungOut, error) {
		req, err := in.request()
		if err != nil {
			return buchungOut{}, err
		}
		entry, err := s.Posting.PostManualEntry(ctx, req)
		if err != nil {
			return buchungOut{}, err
		}
		return booked(entry, string(policy.Confirm)), nil
	})

	AddWrite(tb, Spec{
		Name:  "buchung_stornieren",
		Title: "Buchung stornieren",
		Tier:  TierConfirm,
		Description: "Storniert eine Buchung durch Generalumkehr mit Grund, auf Wunsch zugleich mit der richtigen " +
			"Neubuchung (neubuchung). Die ursprüngliche Buchung bleibt sichtbar. Eine festgeschriebene Buchung " +
			"wird im offenen Zeitraum storniert. Vorher im Chat bestätigen lassen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in stornierenIn) (stornierenOut, error) {
		if strings.TrimSpace(in.Grund) == "" {
			return stornierenOut{}, inputError("grund", "zum Stornieren gehört ein Grund (GoBD Rz. 58)")
		}
		date, err := in.Datum.check("datum")
		if err != nil {
			return stornierenOut{}, err
		}
		if in.Neubuchung == nil {
			reversal, err := s.Journal.ReverseOn(ctx, in.BuchungID, in.Grund, date)
			if err != nil {
				return stornierenOut{}, err
			}
			return stornierenOut{Storno: buchungKurzOf(reversal)}, nil
		}
		manual, err := buchungErfassenIn{Buchungsdatum: in.Neubuchung.Buchungsdatum, Belegdatum: in.Neubuchung.Belegdatum,
			Beschreibung: in.Neubuchung.Beschreibung, Steuerfall: in.Neubuchung.Steuerfall, Zeilen: in.Neubuchung.Zeilen}.request()
		if err != nil {
			return stornierenOut{}, err
		}
		result, err := s.Journal.CorrectEntry(ctx, in.BuchungID, in.Grund, &manual.Entry)
		if err != nil {
			return stornierenOut{}, err
		}
		replacement := buchungKurzOf(result.Replacement)
		return stornierenOut{Storno: buchungKurzOf(result.Reversal), Neubuchung: &replacement, Hinweis: result.Message}, nil
	})

	AddWrite(tb, Spec{
		Name:  "lohnjournal_buchen",
		Title: "Lohnjournal buchen",
		Tier:  TierConfirm,
		Description: "Bucht die Monatssummen aus dem Lohnjournal des Lohnbüros (Beleg vorher mit beleg_ablegen " +
			"ablegen). Buchfink prüft Brutto minus Abzüge gleich Netto. Vorher die Summen dem Menschen vorlegen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in lohnjournalIn) (buchungOut, error) {
		req, err := in.request()
		if err != nil {
			return buchungOut{}, err
		}
		entry, err := s.Payroll.BookPayroll(ctx, req)
		if err != nil {
			return buchungOut{}, err
		}
		return booked(entry, string(policy.Confirm)), nil
	})
}
