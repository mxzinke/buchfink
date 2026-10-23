package mcpserver

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

var mengePattern = regexp.MustCompile(`^[0-9]{1,9}(\.[0-9]{1,3})?$`)

// RechnungPosition ist eine Position einer Ausgangsrechnung.
type RechnungPosition struct {
	Text         string       `json:"text"`
	Menge        string       `json:"menge" jsonschema:"z. B. \"1\" oder \"2.5\", höchstens drei Nachkommastellen"`
	Einheit      string       `json:"einheit,omitempty" jsonschema:"UN/ECE-Code wie C62 (Stück), HUR (Stunde), DAY; Vorgabe C62"`
	Einzelpreis  Betrag       `json:"einzelpreis" jsonschema:"netto"`
	Satz         Satz         `json:"satz"`
	Erloesgruppe Erloesgruppe `json:"erloesgruppe,omitempty" jsonschema:"Vorgabe erloese"`
}

// Rechnung sind die Angaben einer Ausgangsrechnung.
type Rechnung struct {
	KontaktID                uint               `json:"kontakt_id" jsonschema:"Kunde"`
	Rechnungsdatum           Datum              `json:"rechnungsdatum,omitempty" jsonschema:"Vorgabe heute"`
	LeistungVon              Datum              `json:"leistung_von" jsonschema:"Leistungszeitpunkt oder -beginn"`
	LeistungBis              Datum              `json:"leistung_bis,omitempty"`
	Steuerfall               Steuerfall         `json:"steuerfall,omitempty" jsonschema:"Vorgabe domestic"`
	Positionen               []RechnungPosition `json:"positionen"`
	ZahlungszielTage         int                `json:"zahlungsziel_tage,omitempty" jsonschema:"Vorgabe aus dem Kontakt"`
	SkontoPromille           int                `json:"skonto_promille,omitempty"`
	SkontoTage               int                `json:"skonto_tage,omitempty"`
	UstIDUebersteuerungGrund string             `json:"ust_id_uebersteuerung_grund,omitempty" jsonschema:"nur wenn die Bestätigung der USt-IdNr. ausbleibt"`
}

func milli(menge string, field string) (int64, error) {
	if !mengePattern.MatchString(menge) {
		return 0, inputError(field, "%q ist keine Menge (z. B. \"1\" oder \"2.5\")", menge)
	}
	whole, frac, _ := strings.Cut(menge, ".")
	for len(frac) < 3 {
		frac += "0"
	}
	return strconv.ParseInt(whole+frac, 10, 64)
}

func (r Rechnung) invoice() (*domain.Invoice, error) {
	inv := &domain.Invoice{ContactID: r.KontaktID, TaxTreatment: domain.TaxTreatment(r.Steuerfall), Currency: "EUR",
		VatIDOverrideReason: strings.TrimSpace(r.UstIDUebersteuerungGrund),
		Terms:               domain.PaymentTerms{DueDays: r.ZahlungszielTage, DiscountPermille: r.SkontoPromille, DiscountDays: r.SkontoTage}}
	var err error
	if inv.Date, err = r.Rechnungsdatum.check("rechnungsdatum"); err != nil {
		return nil, err
	}
	if inv.ServiceDateFrom, err = r.LeistungVon.check("leistung_von"); err != nil {
		return nil, err
	}
	if inv.ServiceDateTo, err = r.LeistungBis.check("leistung_bis"); err != nil {
		return nil, err
	}
	items, err := invoiceItems(r.Positionen)
	if err != nil {
		return nil, err
	}
	inv.Items = items
	return inv, nil
}

func invoiceItems(positions []RechnungPosition) ([]domain.InvoiceItem, error) {
	if len(positions) == 0 || len(positions) > 100 {
		return nil, inputError("positionen", "1 bis 100 Positionen, nicht %d", len(positions))
	}
	items := make([]domain.InvoiceItem, 0, len(positions))
	for i, p := range positions {
		field := fmt.Sprintf("positionen[%d]", i)
		q, err := milli(p.Menge, field+".menge")
		if err != nil {
			return nil, err
		}
		price, err := p.Einzelpreis.required(field + ".einzelpreis")
		if err != nil {
			return nil, err
		}
		unit := p.Einheit
		if unit == "" {
			unit = "C62"
		}
		group := string(p.Erloesgruppe)
		if group == "" {
			group = "erloese"
		}
		items = append(items, domain.InvoiceItem{Position: i + 1, Description: p.Text, QuantityMilli: q, Unit: unit,
			UnitPrice: price, TaxRate: p.Satz.rate(), PostingGroup: group})
	}
	return items, nil
}

type rechnungVorschauIn struct {
	Rechnung
}

type rechnungVorschauOut struct {
	vorschau
	Kleinbetragsgrenze *Betrag `json:"kleinbetragsgrenze,omitempty"`
}

type verbundIn struct {
	Titel       string `json:"titel" jsonschema:"Auftrag, über den abgerechnet wird"`
	GesamtNetto Betrag `json:"gesamt_netto"`
	Satz        Satz   `json:"satz"`
}

type abschlagIn struct {
	Beschreibung      string `json:"beschreibung"`
	Netto             Betrag `json:"netto"`
	ZahlungErhaltenAm Datum  `json:"zahlung_erhalten_am,omitempty" jsonschema:"bei einer schon gezahlten Anzahlung"`
}

type rechnungAusstellenIn struct {
	KeyedWriteArgs
	Art       RechnungArt `json:"art,omitempty" jsonschema:"rechnung (Vorgabe), abschlag (Anzahlungsrechnung) oder schlussrechnung"`
	Rechnung  *Rechnung   `json:"rechnung,omitempty" jsonschema:"bei rechnung und schlussrechnung"`
	VerbundID uint        `json:"verbund_id,omitempty" jsonschema:"Rechnungsverbund aus einer früheren Abschlagsrechnung"`
	Verbund   *verbundIn  `json:"verbund,omitempty" jsonschema:"neuer Verbund für die erste Abschlagsrechnung"`
	KontaktID uint        `json:"kontakt_id,omitempty" jsonschema:"Kunde eines neuen Verbunds"`
	Datum     Datum       `json:"datum,omitempty" jsonschema:"Datum der Abschlagsrechnung"`
	Abschlag  *abschlagIn `json:"abschlag,omitempty"`
}

type rechnungOut struct {
	Wiederholung
	RechnungID uint          `json:"rechnung_id"`
	Nummer     string        `json:"nummer"`
	Datum      string        `json:"datum"`
	Faellig    string        `json:"faellig,omitempty"`
	Netto      Betrag        `json:"netto"`
	Steuer     Betrag        `json:"steuer"`
	Brutto     Betrag        `json:"brutto"`
	BuchungID  *uint         `json:"buchung_id,omitempty"`
	BelegID    *uint         `json:"beleg_id,omitempty" jsonschema:"das archivierte Rechnungsdokument"`
	VerbundID  *uint         `json:"verbund_id,omitempty"`
	Dateien    []exportDatei `json:"dateien" jsonschema:"Kopien von PDF und E-Rechnung im Exportordner, zum Versand"`
	Hinweis    string        `json:"hinweis"`
}

func (o rechnungOut) Ref() string { return fmt.Sprintf("rechnung:%d", o.RechnungID) }

func rechnungOutOf(inv *domain.Invoice) rechnungOut {
	return rechnungOut{RechnungID: inv.ID, Nummer: inv.InvoiceNumber, Datum: inv.Date, Faellig: inv.DueDate,
		Netto: betrag(inv.NetAmount), Steuer: betrag(inv.TaxAmount), Brutto: betrag(inv.GrossAmount),
		BuchungID: inv.JournalEntryID, BelegID: inv.ReceiptID, VerbundID: inv.GroupID, Dateien: []exportDatei{},
		Hinweis: "Den Versand übernimmt der Agent oder der Mensch außerhalb von Buchfink; die Dateien liegen im Exportordner."}
}

// withFiles legt PDF und E-Rechnung einer ausgestellten Rechnung in den
// Exportordner.
func withFiles(ctx context.Context, c *Call, s core.AgentServices, out rechnungOut) (rechnungOut, error) {
	if out.BelegID == nil {
		return out, nil
	}
	r, err := s.Receipts.Get(ctx, *out.BelegID)
	if err != nil {
		return out, err
	}
	dirs, err := folders(ctx, c, s)
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for i := range r.Files {
		f := &r.Files[i]
		if f.Role == domain.ReceiptRoleAttachment || seen[f.SHA256] {
			continue
		}
		seen[f.SHA256] = true
		file, err := exportReceiptFile(ctx, s, dirs, r, f)
		if err != nil {
			return out, err
		}
		out.Dateien = append(out.Dateien, file)
	}
	return out, nil
}

func registerRechnung(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "rechnung_vorschau",
		Title: "Rechnung durchrechnen",
		Description: "Rechnet eine Ausgangsrechnung durch, ohne sie auszustellen: Buchungssatz, Netto, Steuer, Brutto " +
			"und fehlende Pflichtangaben. Vor rechnung_ausstellen aufrufen und dem Menschen vorlegen.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in rechnungVorschauIn) (rechnungVorschauOut, error) {
		inv, err := in.Rechnung.invoice()
		if err != nil {
			return rechnungVorschauOut{}, err
		}
		p, err := s.Invoices.Preview(ctx, inv)
		if err != nil {
			return rechnungVorschauOut{}, err
		}
		out := rechnungVorschauOut{vorschau: vorschauOf(p)}
		if p.SmallAmountLimit != 0 {
			out.Kleinbetragsgrenze = optBetrag(p.SmallAmountLimit)
		}
		return out, nil
	})

	AddWritePrepared(tb, Spec{
		Name:  "rechnung_ausstellen",
		Title: "Rechnung ausstellen",
		Description: "Stellt eine Ausgangsrechnung mit fortlaufender Nummer aus, bucht die Forderung und archiviert " +
			"das Dokument (E-Rechnung nach dem Format des Kunden). art=abschlag stellt eine Anzahlungsrechnung " +
			"in einem Rechnungsverbund aus (neu mit verbund und kontakt_id, sonst verbund_id); art=schlussrechnung " +
			"rechnet den Verbund ab. Vorher rechnung_vorschau dem Menschen vorlegen und sein Ja einholen. PDF und " +
			"E-Rechnung liegen danach im Exportordner (dateien); den Versand übernimmt der Agent oder der Mensch.",
	}, func(ctx context.Context, c *Call, t *core.Tenant, in rechnungAusstellenIn) error {
		if (in.Art == "" || in.Art == "rechnung") && in.Rechnung != nil {
			inv, err := in.Rechnung.invoice()
			if err != nil {
				return err
			}
			return t.PrepareInvoice(ctx, inv)
		}
		return nil
	}, func(ctx context.Context, c *Call, s core.AgentServices, in rechnungAusstellenIn) (rechnungOut, error) {
		switch in.Art {
		case "", "rechnung":
			if in.Rechnung == nil {
				return rechnungOut{}, inputError("rechnung", "die Angaben der Rechnung fehlen")
			}
			inv, err := in.Rechnung.invoice()
			if err != nil {
				return rechnungOut{}, err
			}
			if err := s.Invoices.Issue(ctx, inv); err != nil {
				return rechnungOut{}, err
			}
			return withFiles(ctx, c, s, rechnungOutOf(inv))
		case "abschlag":
			if in.Abschlag == nil {
				return rechnungOut{}, inputError("abschlag", "die Angaben der Abschlagsrechnung fehlen")
			}
			date, err := in.Datum.check("datum")
			if err != nil {
				return rechnungOut{}, err
			}
			groupID := in.VerbundID
			if groupID == 0 {
				if in.Verbund == nil || in.KontaktID == 0 {
					return rechnungOut{}, inputError("verbund", "für die erste Abschlagsrechnung verbund und kontakt_id angeben")
				}
				total, err := in.Verbund.GesamtNetto.required("verbund.gesamt_netto")
				if err != nil {
					return rechnungOut{}, err
				}
				group, err := s.Invoices.CreateInvoiceGroup(ctx, service.AdvanceGroupRequest{Date: date,
					ContactID: in.KontaktID, Title: in.Verbund.Titel, TotalNet: total, TaxRate: in.Verbund.Satz.rate()})
				if err != nil {
					return rechnungOut{}, err
				}
				groupID = group.ID
			}
			net, err := in.Abschlag.Netto.required("abschlag.netto")
			if err != nil {
				return rechnungOut{}, err
			}
			paid, err := in.Abschlag.ZahlungErhaltenAm.check("abschlag.zahlung_erhalten_am")
			if err != nil {
				return rechnungOut{}, err
			}
			inv, err := s.Invoices.IssueAdvanceInvoice(ctx, service.AdvanceInvoiceRequest{GroupID: groupID, Date: date,
				Description: in.Abschlag.Beschreibung, Net: net, PaymentReceivedAt: paid})
			if err != nil {
				return rechnungOut{}, err
			}
			out := rechnungOutOf(inv)
			out.VerbundID = &groupID
			out.Hinweis = "Eine Anzahlungsrechnung bucht Buchfink erst mit der Zahlung. " + out.Hinweis
			return withFiles(ctx, c, s, out)
		case "schlussrechnung":
			if in.Rechnung == nil || in.VerbundID == 0 {
				return rechnungOut{}, inputError("verbund_id", "für die Schlussrechnung verbund_id und rechnung angeben")
			}
			base, err := in.Rechnung.invoice()
			if err != nil {
				return rechnungOut{}, err
			}
			inv, err := s.Invoices.IssueFinalInvoice(ctx, service.FinalInvoiceRequest{GroupID: in.VerbundID, Date: base.Date,
				ServiceDateFrom: base.ServiceDateFrom, ServiceDateTo: base.ServiceDateTo, Items: base.Items, Terms: base.Terms})
			if err != nil {
				return rechnungOut{}, err
			}
			return withFiles(ctx, c, s, rechnungOutOf(inv))
		}
		return rechnungOut{}, inputError("art", "rechnung, abschlag oder schlussrechnung, nicht %q", in.Art)
	})
}
