package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Stammdaten ändert nur der Mensch. Das Werkzeug legt einen Vorschlag ab; die
// App zeigt ihn als Vorschau, und erst das Übernehmen dort schreibt.

type stammdatenIn struct {
	KeyedWriteArgs
	Begruendung string            `json:"begruendung" jsonschema:"Quelle, z. B. Schreiben des Finanzamts vom 02.09.2026"`
	BelegID     uint              `json:"beleg_id,omitempty" jsonschema:"abgelegtes Schreiben"`
	Unternehmen map[string]string `json:"unternehmen,omitempty" jsonschema:"nur was sich ändert, Feld → Wert: firma, geschaeftsfuehrung, aufsichtsrat_vorsitz, strasse, hausnummer, adresszusatz, plz, ort, land (DE), ansprechpartner, telefon, email, notar, urkundennummer, sitz, registergericht, registernummer, eingetragen_am (JJJJ-MM-TT), steuernummer, finanzamt, bundesland (HH), ust_id, ust_zeitraum (monat, quartal, jahr, keiner), besteuerung (SOLL, IST), dauerfrist (ja, nein), sondervorauszahlung, stammkapital (119.00)"`
	Gruendung   *gruendungIn      `json:"gruendung,omitempty" jsonschema:"laut Gesellschaftsvertrag"`
	Bankkonten  []bankkontoIn     `json:"bankkonten,omitempty" jsonschema:"erkannt an der IBAN"`
	// Steuerberatung entscheidet, wer die Jahreserklärungen erstellt; daraus
	// folgen ihre Fristen (§ 149 Abs. 3 AO). Den Namen der Kanzlei nennt die
	// Begründung.
	Steuerberatung *steuerberatungIn `json:"steuerberatung,omitempty"`
}

type steuerberatungIn struct {
	Beauftragt   bool     `json:"beauftragt"`
	AbJahr       int      `json:"ab_jahr,omitempty"`
	Erklaerungen []string `json:"erklaerungen,omitempty" jsonschema:"kst, gewst, ust_jahr, ebilanz; leer alle"`
}

type gruendungIn struct {
	BeurkundetAm         Datum                     `json:"beurkundet_am"`
	Stammkapital         Betrag                    `json:"stammkapital"`
	GruendungsaufwandBis Betrag                    `json:"gruendungsaufwand_bis,omitempty" jsonschema:"laut Satzung"`
	Gesellschafter       []gruendungGesellschafter `json:"gesellschafter"`
}

type gruendungGesellschafter struct {
	Name              string `json:"name"`
	Anteil            Betrag `json:"anteil"`
	Eingezahlt        Betrag `json:"eingezahlt,omitempty"`
	NichtEingefordert Betrag `json:"nicht_eingefordert,omitempty" jsonschema:"erst auf Beschluss fällig"`
	Sacheinlage       bool   `json:"sacheinlage,omitempty"`
}

type bankkontoIn struct {
	IBAN           string `json:"iban"`
	Name           string `json:"name"`
	BIC            string `json:"bic,omitempty"`
	Bank           string `json:"bank,omitempty"`
	Konto          Konto  `json:"konto,omitempty" jsonschema:"Vorgabe: erstes freie von 1800 bis 1850"`
	FuerRechnungen bool   `json:"fuer_rechnungen,omitempty"`
}

type stammdatenOut struct {
	Wiederholung
	VorschlagID uint        `json:"vorschlag_id"`
	Stand       string      `json:"stand"`
	Aenderungen []aenderung `json:"aenderungen" jsonschema:"Vorschau gegen den heutigen Stand"`
	Hinweis     string      `json:"hinweis"`
}

type aenderung struct {
	Bereich string `json:"bereich"`
	Feld    string `json:"feld"`
	Bisher  string `json:"bisher"`
	Neu     string `json:"neu"`
}

func (o stammdatenOut) Ref() string { return fmt.Sprintf("stammdaten_vorschlag:%d", o.VorschlagID) }

func registerStammdaten(tb *Toolbox) {
	AddWrite(tb, Spec{
		Name:  "stammdaten_vorschlagen",
		Title: "Stammdaten vorschlagen",
		Description: "Schlägt Unternehmensdaten, Gründung, Bankkonten oder die steuerliche Beratung vor, etwa aus einem Schreiben von Notar, " +
			"Registergericht, Finanzamt oder Bank. Ändert nichts: der Mensch prüft die Vorschau in der App und " +
			"übernimmt oder lehnt ab; die Entscheidung steht in status.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in stammdatenIn) (stammdatenOut, error) {
		changes, err := in.changes()
		if err != nil {
			return stammdatenOut{}, err
		}
		req := service.MasterDataProposalRequest{Reason: in.Begruendung, Changes: changes}
		if in.BelegID != 0 {
			if _, err := s.Receipts.Get(ctx, in.BelegID); err != nil {
				return stammdatenOut{}, inputError("beleg_id", "%v", err)
			}
			req.ReceiptID = &in.BelegID
		}
		view, err := s.MasterData.Propose(ctx, req)
		if err != nil {
			return stammdatenOut{}, err
		}
		out := stammdatenOut{
			VorschlagID: view.ID, Stand: string(view.Status), Aenderungen: []aenderung{},
			Hinweis: "Vorgeschlagen, nicht übernommen. Den Menschen bitten, den Vorschlag in Buchfink unter " +
				"Unternehmen › Stammdaten zu prüfen; bis dahin gelten die bisherigen Angaben.",
		}
		for _, l := range view.Lines {
			out.Aenderungen = append(out.Aenderungen, aenderung{Bereich: l.Area, Feld: l.Field, Bisher: l.Before, Neu: l.After})
		}
		return out, nil
	})
}

func (in stammdatenIn) changes() (domain.MasterDataChanges, error) {
	var out domain.MasterDataChanges
	if len(in.Unternehmen) > 0 {
		co, err := companyChanges(in.Unternehmen)
		if err != nil {
			return out, err
		}
		out.Company = co
	}
	if g := in.Gruendung; g != nil {
		f, err := g.proposal()
		if err != nil {
			return out, err
		}
		out.Foundation = f
	}
	if a := in.Steuerberatung; a != nil {
		advisor := domain.TaxAdvisor{Engaged: a.Beauftragt, SinceYear: a.AbJahr, Declarations: a.Erklaerungen}
		if err := advisor.Validate(); err != nil {
			return out, inputError("steuerberatung.erklaerungen", "%v", err)
		}
		if out.Company == nil {
			out.Company = &domain.CompanyChanges{}
		}
		out.Company.TaxAdvisor = &advisor
	}
	for _, b := range in.Bankkonten {
		out.BankAccounts = append(out.BankAccounts, domain.BankAccount{
			IBAN: b.IBAN, Name: b.Name, BIC: b.BIC, BankName: b.Bank, LedgerAccount: string(b.Konto),
			IsInvoiceAccount: b.FuerRechnungen,
		})
	}
	return out, nil
}

// companyFields ordnet die Felder von unternehmen den Unternehmensdaten zu.
var companyFields = map[string]func(co *domain.CompanyChanges, field, v string) error{
	"firma":                textField(func(co *domain.CompanyChanges, v *string) { co.CompanyName = v }),
	"geschaeftsfuehrung":   textField(func(co *domain.CompanyChanges, v *string) { co.ManagingDirectors = v }),
	"aufsichtsrat_vorsitz": textField(func(co *domain.CompanyChanges, v *string) { co.SupervisoryBoardChair = v }),
	"strasse":              textField(func(co *domain.CompanyChanges, v *string) { co.Street = v }),
	"hausnummer":           textField(func(co *domain.CompanyChanges, v *string) { co.HouseNumber = v }),
	"adresszusatz":         textField(func(co *domain.CompanyChanges, v *string) { co.AddressAddition = v }),
	"plz":                  textField(func(co *domain.CompanyChanges, v *string) { co.PostalCode = v }),
	"ort":                  textField(func(co *domain.CompanyChanges, v *string) { co.City = v }),
	"land":                 textField(func(co *domain.CompanyChanges, v *string) { co.CountryCode = v }),
	"ansprechpartner":      textField(func(co *domain.CompanyChanges, v *string) { co.ContactName = v }),
	"telefon":              textField(func(co *domain.CompanyChanges, v *string) { co.ContactPhone = v }),
	"email":                textField(func(co *domain.CompanyChanges, v *string) { co.ContactEmail = v }),
	"notar":                textField(func(co *domain.CompanyChanges, v *string) { co.Notary = v }),
	"urkundennummer":       textField(func(co *domain.CompanyChanges, v *string) { co.DeedNumber = v }),
	"sitz":                 textField(func(co *domain.CompanyChanges, v *string) { co.Seat = v }),
	"registergericht":      textField(func(co *domain.CompanyChanges, v *string) { co.RegisterCourt = v }),
	"registernummer":       textField(func(co *domain.CompanyChanges, v *string) { co.RegisterNumber = v }),
	"steuernummer":         textField(func(co *domain.CompanyChanges, v *string) { co.TaxNumber = v }),
	"finanzamt":            textField(func(co *domain.CompanyChanges, v *string) { co.TaxOffice = v }),
	"bundesland":           textField(func(co *domain.CompanyChanges, v *string) { co.FederalState = v }),
	"ust_id":               textField(func(co *domain.CompanyChanges, v *string) { co.VatID = v }),
	"besteuerung":          textField(func(co *domain.CompanyChanges, v *string) { co.TaxationType = v }),
	"eingetragen_am": func(co *domain.CompanyChanges, field, v string) error {
		if v != "" {
			if _, err := Datum(v).check(field); err != nil {
				return err
			}
		}
		co.RegisteredOn = &v
		return nil
	},
	"ust_zeitraum": func(co *domain.CompanyChanges, field, v string) error {
		period, ok := map[string]string{"monat": "month", "quartal": "quarter", "jahr": "year", "keiner": "none"}[v]
		if !ok {
			return inputError(field, "%q ist kein Zeitraum (monat, quartal, jahr oder keiner)", v)
		}
		co.VatPeriod = &period
		return nil
	},
	"dauerfrist": func(co *domain.CompanyChanges, field, v string) error {
		if v != "ja" && v != "nein" {
			return inputError(field, "ja oder nein")
		}
		on := v == "ja"
		co.PermanentExtension = &on
		return nil
	},
	"sondervorauszahlung": amountField(func(co *domain.CompanyChanges, c *domain.Cents) { co.SpecialPrepayment = c }),
	"stammkapital":        amountField(func(co *domain.CompanyChanges, c *domain.Cents) { co.ShareCapital = c }),
}

func textField(set func(*domain.CompanyChanges, *string)) func(*domain.CompanyChanges, string, string) error {
	return func(co *domain.CompanyChanges, _, v string) error {
		set(co, &v)
		return nil
	}
}

func amountField(set func(*domain.CompanyChanges, *domain.Cents)) func(*domain.CompanyChanges, string, string) error {
	return func(co *domain.CompanyChanges, field, v string) error {
		c, err := Betrag(v).required(field)
		if err != nil {
			return err
		}
		set(co, &c)
		return nil
	}
}

func companyChanges(fields map[string]string) (*domain.CompanyChanges, error) {
	co := &domain.CompanyChanges{}
	for key, v := range fields {
		set, ok := companyFields[key]
		if !ok {
			known := make([]string, 0, len(companyFields))
			for k := range companyFields {
				known = append(known, k)
			}
			sort.Strings(known)
			return nil, inputError("unternehmen."+key, "unbekanntes Feld; möglich sind %s", strings.Join(known, ", "))
		}
		if err := set(co, "unternehmen."+key, v); err != nil {
			return nil, err
		}
	}
	return co, nil
}

func (g gruendungIn) proposal() (*domain.FoundationProposal, error) {
	day, err := g.BeurkundetAm.check("gruendung.beurkundet_am")
	if err != nil {
		return nil, err
	}
	capital, err := g.Stammkapital.required("gruendung.stammkapital")
	if err != nil {
		return nil, err
	}
	costCap, err := g.GruendungsaufwandBis.cents("gruendung.gruendungsaufwand_bis")
	if err != nil {
		return nil, err
	}
	p := &domain.FoundationProposal{NotarizedOn: day, ShareCapital: capital, FoundationCostCap: costCap}
	for i, sh := range g.Gesellschafter {
		field := fmt.Sprintf("gruendung.gesellschafter[%d]", i)
		share, err := sh.Anteil.required(field + ".anteil")
		if err != nil {
			return nil, err
		}
		paid, err := sh.Eingezahlt.cents(field + ".eingezahlt")
		if err != nil {
			return nil, err
		}
		uncalled, err := sh.NichtEingefordert.cents(field + ".nicht_eingefordert")
		if err != nil {
			return nil, err
		}
		kind := domain.ContributionCash
		if sh.Sacheinlage {
			kind = domain.ContributionInKind
		}
		p.Shareholders = append(p.Shareholders, domain.FoundationShareholderProposal{
			Name: sh.Name, ShareCapital: share, PaidIn: paid, Uncalled: uncalled, Kind: kind,
		})
	}
	return p, nil
}
