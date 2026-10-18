package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
)

type kontakteIn struct {
	ID             uint       `json:"id,omitempty" jsonschema:"genau ein Kontakt"`
	Text           string     `json:"text,omitempty" jsonschema:"Teil von Name, Firma, Ort oder Personenkonto"`
	Typ            KontaktTyp `json:"typ,omitempty"`
	UstID          string     `json:"ust_id,omitempty"`
	IBAN           string     `json:"iban,omitempty" jsonschema:"sucht nach IBAN; die Ausgabe nennt sie dann"`
	MitIBAN        bool       `json:"mit_iban,omitempty" jsonschema:"IBAN und BIC ausgeben, etwa zum Abgleich mit einem Bankumsatz"`
	Geschaeftsjahr int        `json:"geschaeftsjahr,omitempty" jsonschema:"Jahr des offenen Saldos; Vorgabe das aktuelle"`
	Seite
}

type standardgruppe struct {
	Gruppe         string `json:"gruppe,omitempty" jsonschema:"vom Menschen bestätigte Standard-Buchungsgruppe"`
	BestaetigtAm   string `json:"bestaetigt_am,omitempty"`
	Vorschlag      string `json:"vorschlag,omitempty" jsonschema:"vorgeschlagen, noch nicht bestätigt"`
	VorschlagNotiz string `json:"vorschlag_notiz,omitempty"`
}

type kontakt struct {
	ID               uint           `json:"id"`
	Typ              string         `json:"typ"`
	Name             string         `json:"name"`
	Firma            string         `json:"firma,omitempty"`
	Personenkonto    string         `json:"personenkonto"`
	Strasse          string         `json:"strasse,omitempty"`
	PLZ              string         `json:"plz,omitempty"`
	Ort              string         `json:"ort,omitempty"`
	Land             string         `json:"land"`
	UstID            string         `json:"ust_id,omitempty"`
	Steuernummer     string         `json:"steuernummer,omitempty"`
	Email            string         `json:"email,omitempty"`
	IBAN             string         `json:"iban,omitempty"`
	BIC              string         `json:"bic,omitempty"`
	ZahlungszielTage int            `json:"zahlungsziel_tage"`
	Privatperson     bool           `json:"privatperson,omitempty"`
	Gesperrt         bool           `json:"gesperrt,omitempty"`
	Offen            Betrag         `json:"offen" jsonschema:"Saldo des Personenkontos im Geschäftsjahr"`
	Standardgruppe   standardgruppe `json:"standardgruppe"`
}

func typOf(t domain.ContactType) string {
	if t == domain.ContactTypeCustomer {
		return "kunde"
	}
	return "lieferant"
}

func contactType(typ KontaktTyp) (domain.ContactType, error) {
	switch typ {
	case "kunde":
		return domain.ContactTypeCustomer, nil
	case "lieferant":
		return domain.ContactTypeVendor, nil
	}
	return "", inputError("typ", "kunde oder lieferant, nicht %q", typ)
}

func kontaktOf(c domain.Contact, withIBAN bool) kontakt {
	k := kontakt{ID: c.ID, Typ: typOf(c.Type), Name: c.Name, Firma: c.Company, Personenkonto: c.LedgerAccount,
		Strasse: c.Street, PLZ: c.PostalCode, Ort: c.City, Land: c.CountryCode, UstID: c.VatID, Steuernummer: c.TaxID,
		Email: c.Email, ZahlungszielTage: c.PaymentTermsDays, Privatperson: c.IsPrivate, Gesperrt: c.Blocked,
		Offen:          betrag(c.OpenAmount),
		Standardgruppe: standardgruppe{Vorschlag: c.ProposedPostingGroup, VorschlagNotiz: c.ProposedPostingGroupNote},
	}
	if c.DefaultPostingGroupConfirmedAt != "" {
		k.Standardgruppe.Gruppe, k.Standardgruppe.BestaetigtAm = c.DefaultPostingGroup, c.DefaultPostingGroupConfirmedAt
	}
	if withIBAN {
		k.IBAN, k.BIC = c.IBAN, c.BIC
	}
	return k
}

type kontakteOut struct {
	Kontakte []kontakt `json:"kontakte"`
	Seitenende
}

type kontaktAnlegenIn struct {
	WriteArgs
	Typ                     KontaktTyp     `json:"typ"`
	Name                    string         `json:"name" jsonschema:"Name oder Ansprechpartner"`
	Firma                   string         `json:"firma,omitempty" jsonschema:"rechtlicher Name, falls abweichend"`
	Strasse                 string         `json:"strasse,omitempty"`
	PLZ                     string         `json:"plz,omitempty"`
	Ort                     string         `json:"ort,omitempty"`
	Land                    string         `json:"land,omitempty" jsonschema:"ISO-Ländercode, Vorgabe DE"`
	UstID                   string         `json:"ust_id,omitempty"`
	Steuernummer            string         `json:"steuernummer,omitempty"`
	Email                   string         `json:"email,omitempty"`
	IBAN                    string         `json:"iban,omitempty"`
	BIC                     string         `json:"bic,omitempty"`
	ZahlungszielTage        int            `json:"zahlungsziel_tage,omitempty"`
	Privatperson            bool           `json:"privatperson,omitempty" jsonschema:"Verbraucher, kein Unternehmer"`
	Kleinunternehmer        bool           `json:"kleinunternehmer,omitempty" jsonschema:"Lieferant nach § 19 UStG"`
	StandardgruppeVorschlag Buchungsgruppe `json:"standardgruppe_vorschlag,omitempty" jsonschema:"Vorschlag für Lieferanten; bestätigt erst der Mensch in der App"`
}

type kontaktAnlegenOut struct {
	Wiederholung
	Kontakt          kontakt  `json:"kontakt"`
	BereitsVorhanden bool     `json:"bereits_vorhanden" jsonschema:"ein vorhandener Kontakt passte; es wurde nichts angelegt"`
	Uebereinstimmung string   `json:"uebereinstimmung,omitempty" jsonschema:"ust_id, iban oder name_plz"`
	Hinweise         []string `json:"hinweise"`
}

func (o kontaktAnlegenOut) Ref() string { return fmt.Sprintf("kontakt:%d", o.Kontakt.ID) }

type standardgruppeIn struct {
	WriteArgs
	KontaktID   uint           `json:"kontakt_id"`
	Gruppe      Buchungsgruppe `json:"buchungsgruppe"`
	Begruendung string         `json:"begruendung" jsonschema:"warum diese Gruppe, z. B. die bisherigen Buchungen"`
}

type standardgruppeOut struct {
	Wiederholung
	Kontakt kontakt `json:"kontakt"`
	Hinweis string  `json:"hinweis"`
}

func (o standardgruppeOut) Ref() string { return fmt.Sprintf("kontakt:%d", o.Kontakt.ID) }

func registerKontakte(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "kontakte_suchen",
		Title: "Kontakte suchen",
		Description: "Kunden und Lieferanten mit Personenkonto, offenem Saldo und Standard-Buchungsgruppe (bestätigt " +
			"oder nur vorgeschlagen). Vor kontakt_anlegen und beleg_buchen aufrufen. IBAN nur mit mit_iban oder " +
			"bei Suche nach iban.",
	}, func(ctx context.Context, c *Call, s *core.Services, in kontakteIn) (kontakteOut, error) {
		out := kontakteOut{Kontakte: []kontakt{}}
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return out, err
		}
		var typ domain.ContactType
		if in.Typ != "" {
			if typ, err = contactType(in.Typ); err != nil {
				return out, err
			}
		}
		contacts, err := s.Contacts.GetContacts(ctx, year)
		if err != nil {
			return out, err
		}
		text := strings.ToLower(strings.TrimSpace(in.Text))
		withIBAN := in.MitIBAN || in.IBAN != ""
		rows := []kontakt{}
		for _, ct := range contacts {
			switch {
			case in.ID != 0 && ct.ID != in.ID,
				typ != "" && ct.Type != typ,
				in.UstID != "" && !strings.EqualFold(compactID(ct.VatID), compactID(in.UstID)),
				in.IBAN != "" && compactID(ct.IBAN) != compactID(in.IBAN),
				text != "" && !strings.Contains(strings.ToLower(ct.Name+" "+ct.Company+" "+ct.City+" "+ct.LedgerAccount), text):
				continue
			}
			rows = append(rows, kontaktOf(ct, withIBAN))
		}
		if in.ID != 0 && len(rows) == 0 {
			return out, &ToolError{Code: CodeNotFound, Nachricht: fmt.Sprintf("Kontakt %d gibt es nicht", in.ID)}
		}
		out.Kontakte, out.Seitenende, err = Paginate(rows, in.Seite)
		return out, err
	})

	AddWrite(tb, Spec{
		Name:  "kontakt_anlegen",
		Title: "Kontakt anlegen",
		Tier:  TierDirect, Idempotent: true,
		Description: "Legt einen Kunden oder Lieferanten mit Personenkonto an. Vorher prüft Buchfink auf Dubletten " +
			"(gleiche USt-IdNr., IBAN oder Name mit PLZ) und liefert dann den vorhandenen Kontakt mit " +
			"bereits_vorhanden=true. Eine mitgegebene Standardgruppe ist nur ein Vorschlag. Ändern vorhandener " +
			"Kontakte geht nur in der App.",
	}, func(ctx context.Context, c *Call, s *core.Services, in kontaktAnlegenIn) (kontaktAnlegenOut, error) {
		out := kontaktAnlegenOut{Hinweise: []string{}}
		typ, err := contactType(in.Typ)
		if err != nil {
			return out, err
		}
		if strings.TrimSpace(in.Name) == "" && strings.TrimSpace(in.Firma) == "" {
			return out, inputError("name", "Name oder Firma fehlt")
		}
		land := strings.ToUpper(strings.TrimSpace(in.Land))
		if land == "" {
			land = "DE"
		}
		contact := domain.Contact{
			Type: typ, Name: in.Name, Company: in.Firma, Street: in.Strasse, PostalCode: in.PLZ, City: in.Ort,
			CountryCode: land, VatID: strings.TrimSpace(in.UstID), TaxID: in.Steuernummer, Email: in.Email,
			IBAN: strings.TrimSpace(in.IBAN), BIC: strings.TrimSpace(in.BIC), PaymentTermsDays: in.ZahlungszielTage,
			IsPrivate: in.Privatperson, IsSmallBusiness: in.Kleinunternehmer,
		}
		if contact.PaymentTermsDays == 0 {
			contact.PaymentTermsDays = 14
		}
		if in.StandardgruppeVorschlag != "" {
			if typ != domain.ContactTypeVendor {
				return out, inputError("standardgruppe_vorschlag", "eine Standardgruppe gilt nur für Lieferanten")
			}
			contact.ProposedPostingGroup = string(in.StandardgruppeVorschlag)
			contact.ProposedPostingGroupNote = "beim Anlegen vorgeschlagen"
		}
		year, err := yearOr(ctx, c, 0)
		if err != nil {
			return out, err
		}
		existing, err := s.Contacts.GetContacts(ctx, year)
		if err != nil {
			return out, err
		}
		if dup, match := policy.DuplicateContact(existing, contact); dup != nil {
			out.Kontakt, out.BereitsVorhanden, out.Uebereinstimmung = kontaktOf(*dup, in.IBAN != ""), true, match
			out.Hinweise = append(out.Hinweise, "Der Kontakt ist schon angelegt; Abweichungen ändert der Mensch in der App.")
			return out, nil
		}
		if err := s.Contacts.SaveContact(ctx, &contact); err != nil {
			return out, err
		}
		out.Kontakt = kontaktOf(contact, in.IBAN != "")
		if contact.VatIDNotice != "" {
			out.Hinweise = append(out.Hinweise, contact.VatIDNotice)
		}
		if contact.ProposedPostingGroup != "" {
			out.Hinweise = append(out.Hinweise, confirmHint)
		}
		return out, nil
	})

	AddWrite(tb, Spec{
		Name:  "standardgruppe_vorschlagen",
		Title: "Standard-Buchungsgruppe vorschlagen",
		Tier:  TierDirect, Idempotent: true,
		Description: "Schlägt für einen Lieferanten die Buchungsgruppe vor, mit der seine Eingangsrechnungen in der " +
			"Regel gebucht werden. Bestätigen kann nur der Mensch in der App; erst dann und nach genug früheren " +
			"Buchungen bucht beleg_buchen seine E-Rechnungen ohne Rückfrage.",
	}, func(ctx context.Context, c *Call, s *core.Services, in standardgruppeIn) (standardgruppeOut, error) {
		contact, err := s.Contacts.ProposeDefaultPostingGroup(ctx, in.KontaktID, string(in.Gruppe), in.Begruendung)
		if err != nil {
			return standardgruppeOut{}, err
		}
		return standardgruppeOut{Kontakt: kontaktOf(*contact, false), Hinweis: confirmHint}, nil
	})
}

// confirmHint sagt, was nach einem Vorschlag der Standardgruppe geschieht.
// Die Bestätigung gehört dem Menschen; bis dahin bleibt jede Belegbuchung des
// Lieferanten eine Rückfrage.
const confirmHint = "Vorgeschlagen, nicht bestätigt. Bestätigen kann nur der Mensch in der App; bis dahin bucht " +
	"beleg_buchen diesen Lieferanten nicht direkt, beleg_buchen_mit_rueckfrage geht immer."

func compactID(s string) string { return strings.ToUpper(strings.Join(strings.Fields(s), "")) }
