package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- Gesellschafterdarlehen -------------------------------------------------

type darlehenZeile struct {
	ID               uint          `json:"id"`
	KontaktID        uint          `json:"kontakt_id"`
	Gesellschafter   string        `json:"gesellschafter"`
	Richtung         string        `json:"richtung"`
	Betrag           Betrag        `json:"betrag" jsonschema:"vereinbarter Darlehensbetrag"`
	Zinssatz         string        `json:"zinssatz" jsonschema:"Prozent im Jahr"`
	Zinsmethode      string        `json:"zinsmethode"`
	VertragVom       string        `json:"vertrag_vom,omitempty"`
	Beginn           string        `json:"beginn"`
	FaelligAm        string        `json:"faellig_am,omitempty"`
	Konto            string        `json:"konto"`
	Zinskonto        string        `json:"zinskonto"`
	BelegID          *uint         `json:"beleg_id,omitempty" jsonschema:"der abgelegte Vertrag"`
	Sicherheiten     string        `json:"sicherheiten,omitempty"`
	Vergleichszins   string        `json:"vergleichszins,omitempty"`
	Nachrangig       bool          `json:"nachrangig,omitempty"`
	Stand            Betrag        `json:"stand" jsonschema:"gebuchter Bestand zum Stichtag"`
	ErsteAuszahlung  string        `json:"erste_auszahlung,omitempty"`
	ZinsenGebuchtBis string        `json:"zinsen_gebucht_bis,omitempty"`
	Befunde          []befundZeile `json:"befunde" jsonschema:"Hinweise zum Fremdvergleich; sie sperren nichts"`
}

func darlehenOf(v *service.LoanView) darlehenZeile {
	l := v.Loan
	out := darlehenZeile{ID: l.ID, KontaktID: l.ContactID, Gesellschafter: v.ContactName,
		Richtung: label(loanDirections, l.Direction), Betrag: betrag(l.Principal), Zinssatz: zinssatz(l.InterestRateBP),
		Zinsmethode: label(dayCounts, l.DayCount), VertragVom: l.ContractDate, Beginn: l.StartDate,
		FaelligAm: l.MaturityDate, Konto: l.Account, Zinskonto: l.InterestAccount, BelegID: l.ContractReceiptID,
		Sicherheiten: l.Collateral, Nachrangig: l.Subordinated, Stand: betrag(v.Outstanding),
		ErsteAuszahlung: v.FirstMovement, ZinsenGebuchtBis: v.InterestBookedThrough, Befunde: []befundZeile{}}
	if l.ComparisonRateBP != nil {
		out.Vergleichszins = zinssatz(*l.ComparisonRateBP)
	}
	for _, f := range v.Findings {
		out.Befunde = append(out.Befunde, befundZeile{Regel: f.Rule, Schwere: string(f.Severity), Objekt: f.ObjectName,
			Text: f.Message, Norm: f.Reference})
	}
	return out
}

type darlehenIn struct {
	KeyedWriteArgs
	DarlehenID           uint             `json:"darlehen_id,omitempty" jsonschema:"zu berichtigender Vertrag; ohne Angabe ein neuer"`
	Grund                string           `json:"grund,omitempty" jsonschema:"Pflicht beim Berichtigen"`
	KontaktID            uint             `json:"kontakt_id" jsonschema:"Gesellschafter laut Unternehmensdaten"`
	Richtung             DarlehenRichtung `json:"richtung"`
	Betrag               Betrag           `json:"betrag"`
	Zinssatz             Zinssatz         `json:"zinssatz"`
	Zinsmethode          Zinsmethode      `json:"zinsmethode,omitempty"`
	VertragVom           Datum            `json:"vertrag_vom,omitempty"`
	Beginn               Datum            `json:"beginn"`
	FaelligAm            Datum            `json:"faellig_am,omitempty" jsonschema:"ohne Angabe unbestimmte Laufzeit"`
	Konto                Konto            `json:"konto,omitempty" jsonschema:"Vorgabe nach Laufzeit"`
	Zinskonto            Konto            `json:"zinskonto,omitempty" jsonschema:"Vorgabe nach Beteiligung"`
	BelegID              uint             `json:"beleg_id,omitempty" jsonschema:"abgelegter Vertrag"`
	Sicherheiten         string           `json:"sicherheiten,omitempty"`
	Vergleichszins       Zinssatz         `json:"vergleichszins,omitempty" jsonschema:"Zins eines fremden Dritten"`
	VergleichszinsQuelle string           `json:"vergleichszins_quelle,omitempty" jsonschema:"Nachweis, etwa ein Bankangebot"`
	Nachrangig           bool             `json:"nachrangig,omitempty" jsonschema:"Rangrücktritt vereinbart"`
	Notiz                string           `json:"notiz,omitempty"`
	NurVorschau          bool             `json:"nur_vorschau,omitempty" jsonschema:"prüfen, nicht speichern"`
}

func (in darlehenIn) preview() bool { return in.NurVorschau }

func (in darlehenIn) request() (service.LoanRequest, error) {
	req := service.LoanRequest{ContactID: in.KontaktID, Account: string(in.Konto), InterestAccount: string(in.Zinskonto),
		ContractReceiptID: in.BelegID, Collateral: in.Sicherheiten, ComparisonSource: in.VergleichszinsQuelle,
		Subordinated: in.Nachrangig, Note: in.Notiz}
	var err error
	if req.Direction, err = pick("richtung", in.Richtung, loanDirections); err != nil {
		return req, err
	}
	if req.DayCount, err = pick("zinsmethode", in.Zinsmethode, dayCounts); err != nil {
		return req, err
	}
	if req.Principal, err = in.Betrag.required("betrag"); err != nil {
		return req, err
	}
	if req.InterestRateBP, err = in.Zinssatz.basisPoints("zinssatz"); err != nil {
		return req, err
	}
	if in.Vergleichszins != "" {
		rate, err := in.Vergleichszins.basisPoints("vergleichszins")
		if err != nil {
			return req, err
		}
		req.ComparisonRateBP = &rate
	}
	if req.StartDate, err = in.Beginn.check("beginn"); err != nil {
		return req, err
	}
	if req.ContractDate, err = in.VertragVom.check("vertrag_vom"); err != nil {
		return req, err
	}
	if req.MaturityDate, err = in.FaelligAm.check("faellig_am"); err != nil {
		return req, err
	}
	return req, nil
}

type darlehenOut struct {
	Wiederholung
	Vorschau bool          `json:"vorschau"`
	Darlehen darlehenZeile `json:"darlehen"`
	Hinweis  string        `json:"hinweis,omitempty"`
}

func (o darlehenOut) Ref() string { return fmt.Sprintf("darlehen:%d", o.Darlehen.ID) }

// --- Ergebnisverwendung -----------------------------------------------------

type ergebnisverwendungIn struct {
	WriteArgs
	Geschaeftsjahr       int    `json:"geschaeftsjahr" jsonschema:"Jahr, dessen Ergebnis verwendet wird"`
	BeschlussAm          Datum  `json:"beschluss_am" jsonschema:"Tag des Beschlusses im Folgejahr"`
	Text                 string `json:"text,omitempty" jsonschema:"Wortlaut des Beschlusses"`
	GesetzlicheRuecklage Betrag `json:"gesetzliche_ruecklage,omitempty"`
	AndereRuecklagen     Betrag `json:"andere_ruecklagen,omitempty"`
	Ausschuettung        Betrag `json:"ausschuettung,omitempty" jsonschema:"brutto; die Kapitalertragsteuer rechnet Buchfink"`
	BelegID              uint   `json:"beleg_id,omitempty" jsonschema:"abgelegter Beschluss; sonst ein Eigenbeleg"`
	NurVorschau          bool   `json:"nur_vorschau,omitempty" jsonschema:"rechnen, nicht buchen"`
}

func (in ergebnisverwendungIn) preview() bool { return in.NurVorschau }

type ergebnisverwendungOut struct {
	Wiederholung
	Vorschau              bool            `json:"vorschau"`
	Geschaeftsjahr        int             `json:"geschaeftsjahr"`
	Buchungsjahr          int             `json:"buchungsjahr"`
	BeschlussAm           string          `json:"beschluss_am"`
	Festgestellt          bool            `json:"festgestellt" jsonschema:"der Abschluss des Geschäftsjahres ist festgestellt; sonst bucht Buchfink nicht"`
	VerwendbaresErgebnis  Betrag          `json:"verwendbares_ergebnis" jsonschema:"Saldo des Vortragskontos im Buchungsjahr"`
	Jahresergebnis        Betrag          `json:"jahresergebnis"`
	GesetzlicheRuecklage  Betrag          `json:"gesetzliche_ruecklage"`
	AndereRuecklagen      Betrag          `json:"andere_ruecklagen"`
	Ausschuettung         Betrag          `json:"ausschuettung"`
	Kapitalertragsteuer   Betrag          `json:"kapitalertragsteuer"`
	Solidaritaetszuschlag Betrag          `json:"solidaritaetszuschlag"`
	Vortrag               Betrag          `json:"vortrag" jsonschema:"auf neue Rechnung, ohne Buchung"`
	Zeilen                []buchungsZeile `json:"zeilen"`
	BuchungID             *uint           `json:"buchung_id,omitempty"`
	Erlaeuterung          string          `json:"erlaeuterung"`
	Hinweise              []string        `json:"hinweise"`
}

func (o ergebnisverwendungOut) Ref() string {
	return fmt.Sprintf("ergebnisverwendung:%d", o.Geschaeftsjahr)
}

func (in ergebnisverwendungIn) request() (service.AppropriationRequest, error) {
	req := service.AppropriationRequest{Text: strings.TrimSpace(in.Text), ReceiptID: in.BelegID}
	var err error
	if req.DecisionDate, err = in.BeschlussAm.check("beschluss_am"); err != nil {
		return req, err
	}
	if req.DecisionDate == "" {
		return req, inputError("beschluss_am", "der Tag des Beschlusses fehlt")
	}
	for _, f := range []struct {
		name string
		in   Betrag
		out  *domain.Cents
	}{
		{"gesetzliche_ruecklage", in.GesetzlicheRuecklage, &req.LegalReserve},
		{"andere_ruecklagen", in.AndereRuecklagen, &req.OtherReserves},
		{"ausschuettung", in.Ausschuettung, &req.Distribution},
	} {
		if *f.out, err = f.in.cents(f.name); err != nil {
			return req, err
		}
	}
	return req, nil
}

func registerGesellschaft(tb *Toolbox) {
	AddWrite(tb, Spec{
		Name:  "darlehen_erfassen",
		Title: "Gesellschafterdarlehen erfassen",
		Tier:  TierConfirm,
		Description: "Erfasst einen Darlehensvertrag zwischen der GmbH und einem Gesellschafter oder berichtigt ihn " +
			"(darlehen_id mit grund). Erst mit nur_vorschau prüfen, dann dem Menschen Gesellschafter, Richtung, " +
			"Betrag, Zinssatz, Laufzeit, Konten, den Vertrag (beleg_id) und die Befunde zum Fremdvergleich vorlegen " +
			"und nach seinem Ja erfassen. Auszahlung und Tilgung danach über bankumsatz_buchen mit darlehen_id; " +
			"den Stand zeigt auswertung mit art=darlehen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in darlehenIn) (darlehenOut, error) {
		req, err := in.request()
		if err != nil {
			return darlehenOut{}, err
		}
		reason := strings.TrimSpace(in.Grund)
		if in.DarlehenID != 0 && reason == "" {
			return darlehenOut{}, inputError("grund", "eine Berichtigung des Vertrags braucht einen Grund")
		}
		if _, err := s.Contacts.Contact(ctx, in.KontaktID); err != nil {
			return darlehenOut{}, notFound("kontakt_id", in.KontaktID, err)
		}
		var view *service.LoanView
		switch {
		case in.NurVorschau:
			view, err = s.Shareholders.PreviewLoan(ctx, in.DarlehenID, req)
		case in.DarlehenID != 0:
			view, err = s.Shareholders.CorrectLoan(ctx, in.DarlehenID, req, reason)
		default:
			view, err = s.Shareholders.RegisterLoan(ctx, req)
		}
		if err != nil {
			return darlehenOut{}, err
		}
		return darlehenOut{Vorschau: in.NurVorschau, Darlehen: darlehenOf(view),
			Hinweis: "Auszahlung und Tilgung bucht bankumsatz_buchen mit darlehen_id."}, nil
	})

	AddWrite(tb, Spec{
		Name:  "ergebnisverwendung_buchen",
		Title: "Ergebnisverwendung buchen",
		Tier:  TierConfirm,
		Description: "Bucht den Gesellschafterbeschluss über das Ergebnis eines Geschäftsjahres im Folgejahr: " +
			"Rücklagen, Ausschüttung mit einbehaltener Kapitalertragsteuer, Rest als Vortrag ohne Buchung. " +
			"Voraussetzung sind der Saldenvortrag und der vom Menschen in der App festgestellte Abschluss; der " +
			"Beschluss liegt nicht vor der Feststellung. Erst mit nur_vorschau rechnen, dann dem Menschen Beschluss " +
			"(Tag, Wortlaut, beleg_id), Beträge und Buchungssatz vorlegen und nach seinem Ja buchen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in ergebnisverwendungIn) (ergebnisverwendungOut, error) {
		req, err := in.request()
		if err != nil {
			return ergebnisverwendungOut{}, err
		}
		if in.Geschaeftsjahr <= 0 {
			return ergebnisverwendungOut{}, inputError("geschaeftsjahr", "das Geschäftsjahr fehlt")
		}
		preview, err := s.Appropriation.PreviewAppropriation(ctx, in.Geschaeftsjahr, req)
		if err != nil {
			return ergebnisverwendungOut{}, err
		}
		a := preview.Appropriation
		out := ergebnisverwendungOut{Vorschau: in.NurVorschau, Geschaeftsjahr: preview.Year,
			Buchungsjahr: preview.BookingYear, BeschlussAm: preview.BookingDate,
			VerwendbaresErgebnis: betrag(preview.NetIncome), Jahresergebnis: betrag(preview.YearResult),
			GesetzlicheRuecklage: betrag(a.LegalReserve), AndereRuecklagen: betrag(a.OtherReserves),
			Ausschuettung: betrag(a.Distribution), Kapitalertragsteuer: betrag(a.WithholdingTax),
			Solidaritaetszuschlag: betrag(a.SolidarityOnWithholding), Vortrag: betrag(a.CarryForward),
			Zeilen: zeilenOf(preview.Lines), Erlaeuterung: preview.Explanation, Hinweise: orEmpty(preview.Warnings)}
		if fy, err := s.Closing.YearOf(ctx, in.Geschaeftsjahr); err == nil && fy != nil {
			out.Festgestellt = fy.IsAdopted()
		}
		if in.NurVorschau {
			if !out.Festgestellt {
				out.Hinweise = append(out.Hinweise, fmt.Sprintf("Der Abschluss %d ist noch nicht festgestellt; "+
					"gebucht wird erst, wenn der Mensch die Feststellung in der App erfasst hat.", in.Geschaeftsjahr))
			}
			return out, nil
		}
		decided, err := s.Appropriation.BookAppropriation(ctx, in.Geschaeftsjahr, req)
		if err != nil {
			return ergebnisverwendungOut{}, err
		}
		out.BuchungID = decided.JournalEntryID
		if decided.JournalEntryID == nil {
			out.Hinweise = append(out.Hinweise, "Der Beschluss trägt alles vor; dafür ist keine Buchung nötig.")
		}
		return out, nil
	})
}
