package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- aufgaben ---------------------------------------------------------------

type aufgabenIn struct {
	Geschaeftsjahr int `json:"geschaeftsjahr,omitempty" jsonschema:"Geschäftsjahr der Liste; Vorgabe geschaeftsjahr_aktuell aus status"`
}

type aufgabe struct {
	Schluessel  string            `json:"schluessel"`
	Titel       string            `json:"titel"`
	Warum       string            `json:"warum"`
	Anzahl      int               `json:"anzahl,omitempty"`
	Betrag      *Betrag           `json:"betrag,omitempty"`
	Faellig     string            `json:"faellig,omitempty"`
	Rechtsgrund string            `json:"rechtsgrund,omitempty"`
	Werkzeug    string            `json:"werkzeug" jsonschema:"das Werkzeug, mit dem es weitergeht, oder nur_app"`
	Parameter   map[string]string `json:"parameter,omitempty" jsonschema:"Vorschlag für die Eingabe des Werkzeugs"`
}

type aufgabenOut struct {
	Stichtag       string    `json:"stichtag"`
	Geschaeftsjahr int       `json:"geschaeftsjahr"`
	Ueberfaellig   []aufgabe `json:"ueberfaellig"`
	Offen          []aufgabe `json:"offen"`
	Demnaechst     []aufgabe `json:"demnaechst"`
}

// taskTool übersetzt das Ziel einer Aufgabe in der Oberfläche in einen
// Werkzeugaufruf. Pflichten führen über ihren Schlüssel zu fristen, gleich
// welcher Reiter sie in der Oberfläche zeigt.
func taskTool(t domain.Task) (string, map[string]string) {
	if strings.HasPrefix(t.Key, domain.TaskKeyObligation+".") {
		return "fristen", map[string]string{"schluessel": t.Target.ID}
	}
	switch t.Target.Area {
	case domain.AreaBooks:
		switch t.Target.Tab {
		case "receipts":
			return "belege_suchen", map[string]string{"status": "abgelegt"}
		case "bank":
			return "bankumsaetze", map[string]string{"status": "offen"}
		case "invoices":
			return "offene_posten", map[string]string{"richtung": "forderung", "nur_ueberfaellig": "true"}
		}
	case domain.AreaSettings:
		if t.Target.Tab == "audit" {
			return "auswertung", map[string]string{"art": "pruefbericht"}
		}
	case domain.AreaCompany:
		if t.Target.Tab == "contacts" {
			return "kontakte_suchen", nil
		}
	case domain.AreaClosings:
		return "abschluss_status", nil
	}
	return "nur_app", nil
}

func registerAufgaben(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "aufgaben",
		Title: "Aufgaben",
		Description: "Die Arbeitsliste des Mandanten: überfällig, offen, demnächst (Belege ohne Buchung, offene " +
			"Bankumsätze, Befunde, Pflichten, Sicherung). Nach status aufrufen. Jede Aufgabe nennt das Werkzeug, " +
			"mit dem es weitergeht, oder nur_app, wenn der Mensch in der App handeln muss. Für Fristen im Einzelnen fristen.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in aufgabenIn) (aufgabenOut, error) {
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return aufgabenOut{}, err
		}
		opts := service.TaskOptions{FiscalYear: year}
		if ex := c.Tenant.Examiner(); ex.Active {
			opts.ReadOnlyUntil, opts.ReadOnlyReason = ex.Until, ex.Reason
		}
		list, err := s.Tasks.Tasks(ctx, opts)
		if err != nil {
			return aufgabenOut{}, err
		}
		conv := func(tasks []domain.Task) []aufgabe {
			out := make([]aufgabe, 0, len(tasks))
			for _, t := range tasks {
				tool, params := taskTool(t)
				a := aufgabe{Schluessel: t.Key, Titel: t.Title, Warum: t.Why, Anzahl: t.Count, Faellig: t.DueDate,
					Rechtsgrund: t.Reference, Werkzeug: tool, Parameter: params}
				if t.Amount != 0 {
					a.Betrag = optBetrag(t.Amount)
				}
				out = append(out, a)
			}
			return out
		}
		out := aufgabenOut{Stichtag: list.Today, Geschaeftsjahr: year,
			Ueberfaellig: conv(list.Overdue), Offen: conv(list.Open), Demnaechst: conv(list.Upcoming)}
		dirs, err := folders(ctx, c, s)
		if err != nil {
			return out, err
		}
		switch st := inboxStateOf(dirs, time.Now()); {
		case st.Fehler != "":
			out.Offen = append([]aufgabe{{Schluessel: "eingang", Titel: "Der Eingang ist nicht lesbar",
				Warum: st.Fehler + ". Der Mensch gibt Buchfink in den Einstellungen des Systems Zugriff auf den Ordner " +
					"oder wählt in der App einen anderen Eingang.", Werkzeug: "nur_app"}}, out.Offen...)
		case st.Dateien > 0:
			why := "Im Eingang liegen Dateien, die noch nicht abgelegt sind."
			if st.AeltesteVorTagen > 0 {
				why += fmt.Sprintf(" Die älteste Datei ist %d Tage alt.", st.AeltesteVorTagen)
			}
			out.Offen = append([]aufgabe{{Schluessel: "eingang", Titel: st.Hinweis, Warum: why + " Abgelegt wird, wenn der Mensch es sagt.",
				Anzahl: st.Dateien, Werkzeug: "eingang"}}, out.Offen...)
		}
		return out, nil
	})
}

// --- fristen ---------------------------------------------------------------

type fristenIn struct {
	Schluessel     string          `json:"schluessel,omitempty" jsonschema:"eine Pflicht, z. B. ustva.2026-Q1; die übrigen Filter entfallen dann"`
	Von            Datum           `json:"von,omitempty" jsonschema:"Fälligkeit ab"`
	Bis            Datum           `json:"bis,omitempty" jsonschema:"Fälligkeit bis; ohne Angabe heute plus 180 Tage"`
	Geschaeftsjahr int             `json:"geschaeftsjahr,omitempty"`
	Status         []PflichtStatus `json:"status,omitempty" jsonschema:"ohne Angabe alles noch nicht Erledigte"`
	Art            []PflichtArt    `json:"art,omitempty"`
	Seite
}

type pflichtVorbereitet struct {
	Am    string `json:"am"`
	Notiz string `json:"notiz,omitempty"`
	Kanal string `json:"kanal"`
}

type pflichtNachweis struct {
	Datum       string `json:"datum"`
	Art         string `json:"art"`
	Referenz    string `json:"referenz,omitempty"`
	Automatisch bool   `json:"automatisch" jsonschema:"aus den Daten abgeleitet statt vom Menschen bestätigt"`
}

type pflicht struct {
	Schluessel       string              `json:"schluessel"`
	Art              string              `json:"art"`
	Titel            string              `json:"titel"`
	Zeitraum         string              `json:"zeitraum"`
	Geschaeftsjahr   int                 `json:"geschaeftsjahr"`
	Faellig          string              `json:"faellig"`
	GesetzlicheFrist string              `json:"gesetzliche_frist,omitempty" jsonschema:"Fristende vor Verschiebung nach § 108 Abs. 3 AO"`
	Verschoben       string              `json:"verschoben_wegen,omitempty"`
	Unverzueglich    bool                `json:"unverzueglich,omitempty"`
	Status           string              `json:"status" jsonschema:"offen, vorbereitet, ueberfaellig, erledigt oder entfaellt"`
	Kanal            string              `json:"kanal" jsonschema:"Weg der Erfüllung, z. B. elster, bzst, unternehmensregister"`
	Zustaendig       string              `json:"zustaendig" jsonschema:"agent_bereitet_vor, mensch oder steuerberater"`
	NachweisAus      string              `json:"nachweis_aus" jsonschema:"daten, mensch oder daten_oder_mensch"`
	Grundlage        string              `json:"grundlage,omitempty"`
	Verweis          string              `json:"verweis" jsonschema:"Abschnitt in der Skill-Referenz"`
	Anker            string              `json:"anker" jsonschema:"Anker des Abschnitts in pflichten.md; weicht bei einigen Arten von art ab"`
	Vorbereitet      *pflichtVorbereitet `json:"vorbereitet,omitempty"`
	Nachweis         *pflichtNachweis    `json:"nachweis,omitempty"`
	EntfaelltGrund   string              `json:"entfaellt_grund,omitempty"`
	Hinweise         []string            `json:"hinweise"`
}

type fristenOut struct {
	Stichtag  string    `json:"stichtag"`
	Pflichten []pflicht `json:"pflichten"`
	Seitenende
}

var obligationStatus = map[domain.ObligationStatus]string{
	domain.ObligationOpen: "offen", domain.ObligationPrepared: "vorbereitet", domain.ObligationOverdue: "ueberfaellig",
	domain.ObligationDone: "erledigt", domain.ObligationWaived: "entfaellt",
}

var responsibility = map[domain.ObligationResponsibility]string{
	domain.ResponsibilityAgentPrepares: "agent_bereitet_vor", domain.ResponsibilityHumanConfirms: "mensch",
	domain.ResponsibilityTaxAdvisor: "steuerberater",
}

var evidenceSource = map[domain.EvidenceSource]string{
	domain.EvidenceFromData: "daten", domain.EvidenceFromHuman: "mensch", domain.EvidenceFromEither: "daten_oder_mensch",
}

func pflichtOf(o domain.Obligation) pflicht {
	p := pflicht{
		Schluessel: o.Key, Art: string(o.Kind), Titel: o.Title, Zeitraum: o.PeriodLabel, Geschaeftsjahr: o.FiscalYear,
		Faellig: o.DueDate, Verschoben: o.ShiftReason, Unverzueglich: o.Immediate,
		Status: obligationStatus[o.Status], Kanal: string(o.Channel), Zustaendig: responsibility[o.Responsibility],
		NachweisAus: evidenceSource[o.Evidence], Grundlage: o.Basis, Verweis: o.SkillRef, Anker: o.Kind.Anchor(),
		EntfaelltGrund: o.WaivedReason, Hinweise: orEmpty(o.Hints),
	}
	if o.StatutoryDate != o.DueDate {
		p.GesetzlicheFrist = o.StatutoryDate
	}
	if o.Preparation != nil {
		p.Vorbereitet = &pflichtVorbereitet{Am: o.Preparation.At, Notiz: o.Preparation.Note, Kanal: o.Preparation.Channel}
	}
	if o.Proof != nil {
		p.Nachweis = &pflichtNachweis{Datum: o.Proof.Date, Art: string(o.Proof.Kind), Referenz: o.Proof.Reference,
			Automatisch: o.Proof.Automatic}
	}
	return p
}

var statusFromLabel = map[string]domain.ObligationStatus{
	"offen": domain.ObligationOpen, "vorbereitet": domain.ObligationPrepared, "ueberfaellig": domain.ObligationOverdue,
	"erledigt": domain.ObligationDone, "entfaellt": domain.ObligationWaived,
}

// --- pflicht_vorbereitet / anlass_erfassen -------------------------------------

type pflichtVorbereitetIn struct {
	WriteArgs
	Schluessel string `json:"schluessel" jsonschema:"Schlüssel der Pflicht aus fristen, z. B. ustva.2026-Q1"`
	Notiz      string `json:"notiz" jsonschema:"was vorbereitet ist und wo die Datei liegt"`
}

type pflichtOut struct {
	Wiederholung
	Pflicht pflicht `json:"pflicht"`
}

func (o pflichtOut) Ref() string { return "pflicht:" + o.Pflicht.Schluessel }

type anlassIn struct {
	KeyedWriteArgs
	Art   AnlassArt `json:"art"`
	Datum Datum     `json:"datum" jsonschema:"Tag des Anlasses (Wirksamwerden, Bescheiddatum)"`
	Notiz string    `json:"notiz,omitempty" jsonschema:"was geschehen ist, mit Fundstelle des Nachweises"`
}

type anlassOut struct {
	Wiederholung
	ID      uint   `json:"id"`
	Art     string `json:"art"`
	Datum   string `json:"datum"`
	Hinweis string `json:"hinweis"`
}

func (o anlassOut) Ref() string { return fmt.Sprintf("anlass:%d", o.ID) }

// obligationKey nimmt den Schlüssel einer Pflicht auch mit dem Anker statt
// der Art an („ebilanz.2026" für „e-bilanz.2026").
func obligationKey(key string) string {
	kind, rest, ok := strings.Cut(key, ".")
	if !ok {
		return key
	}
	if k, found := domain.ObligationKindOf(kind); found {
		return string(k) + "." + rest
	}
	return key
}

func registerFristen(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "fristen",
		Title: "Fristen und Pflichten",
		Description: "Pflichtenkalender: je Pflicht Art, Zeitraum, Fälligkeit, Status (offen, vorbereitet, ueberfaellig, " +
			"erledigt, entfaellt), Kanal, Zuständigkeit und verweis auf den Abschnitt in " +
			"buchfink-grundlagen/references/pflichten.md. Ohne Filter alles noch nicht Erledigte bis 180 Tage voraus. " +
			"Nach dem Vorbereiten pflicht_vorbereitet melden; erledigt setzt nur die App oder der Datenstand.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in fristenIn) (fristenOut, error) {
		out := fristenOut{Stichtag: today(), Pflichten: []pflicht{}}
		if in.Schluessel != "" {
			o, err := s.Obligations.Get(ctx, obligationKey(in.Schluessel), "")
			if err != nil {
				return out, err
			}
			out.Pflichten = append(out.Pflichten, pflichtOf(*o))
			out.Gesamt = 1
			return out, nil
		}
		q := service.ObligationQuery{FiscalYear: in.Geschaeftsjahr}
		var err error
		if q.From, err = in.Von.check("von"); err != nil {
			return out, err
		}
		if q.To, err = in.Bis.check("bis"); err != nil {
			return out, err
		}
		if q.To == "" && q.FiscalYear == 0 {
			q.To = time.Now().AddDate(0, 0, 180).Format("2006-01-02")
		}
		for _, st := range in.Status {
			status, ok := statusFromLabel[string(st)]
			if !ok {
				return out, inputError("status", "unbekannter Stand %q", st)
			}
			q.Statuses = append(q.Statuses, status)
		}
		if len(q.Statuses) == 0 {
			q.Statuses = []domain.ObligationStatus{domain.ObligationOpen, domain.ObligationPrepared, domain.ObligationOverdue}
		}
		for _, k := range in.Art {
			kind, ok := domain.ObligationKindOf(string(k))
			if !ok {
				return out, inputError("art", "unbekannte Pflichtart %q", k)
			}
			q.Kinds = append(q.Kinds, kind)
		}
		list, err := s.Obligations.List(ctx, q)
		if err != nil {
			return out, err
		}
		rows := make([]pflicht, 0, len(list))
		for _, o := range list {
			rows = append(rows, pflichtOf(o))
		}
		out.Pflichten, out.Seitenende, err = Paginate(rows, in.Seite)
		return out, err
	})

	AddWrite(tb, Spec{
		Name:       "pflicht_vorbereitet",
		Title:      "Pflicht als vorbereitet melden",
		Idempotent: true,
		Description: "Meldet eine Pflicht aus fristen als vorbereitet, mit Notiz (was fertig ist, wo die Datei liegt). " +
			"Danach dem Menschen Weg, Fälligkeit und Nachweis nennen. „Erledigt“ meldet ein Agent nie: das setzen " +
			"die Daten (Transferticket, Festschreibung) oder der Mensch in der App.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in pflichtVorbereitetIn) (pflichtOut, error) {
		if in.Schluessel == "" {
			return pflichtOut{}, inputError("schluessel", "der Schlüssel der Pflicht fehlt")
		}
		o, err := s.Obligations.MarkPrepared(ctx, obligationKey(in.Schluessel), in.Notiz)
		if err != nil {
			return pflichtOut{}, err
		}
		return pflichtOut{Pflicht: pflichtOf(*o)}, nil
	})

	AddWrite(tb, Spec{
		Name:  "anlass_erfassen",
		Title: "Anlass erfassen",
		Description: "Erfasst einen Anlass, der Pflichten auslöst: Gesellschafterwechsel (Gesellschafterliste, " +
			"Transparenzregister), Änderung der wirtschaftlich Berechtigten, Vorauszahlungsbescheid KSt oder GewSt. " +
			"Vorher im Chat bestätigen lassen. Danach fristen aufrufen.",
	}, func(ctx context.Context, c *Call, s core.AgentServices, in anlassIn) (anlassOut, error) {
		if !domain.ObligationEventKind(in.Art).Valid() {
			return anlassOut{}, inputError("art", "gesellschafterwechsel, wirtschaftlich-berechtigte, "+
				"vorauszahlungsbescheid-kst oder vorauszahlungsbescheid-gewst, nicht %q", in.Art)
		}
		date, err := in.Datum.check("datum")
		if err != nil {
			return anlassOut{}, err
		}
		ev, err := s.Obligations.RecordEvent(ctx, domain.ObligationEventKind(in.Art), date, in.Notiz)
		if err != nil {
			return anlassOut{}, err
		}
		return anlassOut{ID: ev.ID, Art: string(ev.Kind), Datum: ev.Date,
			Hinweis: "Die ausgelösten Pflichten zeigt fristen."}, nil
	})
}
