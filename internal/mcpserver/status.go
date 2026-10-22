package mcpserver

import (
	"context"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
)

type statusIn struct{}

type statusOut struct {
	Bereit           bool             `json:"bereit" jsonschema:"true, wenn ein Mandant geöffnet und für diese App freigegeben ist"`
	Hinweis          string           `json:"hinweis,omitempty" jsonschema:"warum Buchfink nicht bereit ist oder was einschränkt"`
	NaechsterSchritt string           `json:"naechster_schritt,omitempty"`
	Version          string           `json:"version" jsonschema:"Fassung der laufenden Buchfink-App"`
	Mandant          *statusTenant    `json:"mandant,omitempty"`
	GJAktuell        int              `json:"geschaeftsjahr_aktuell,omitempty" jsonschema:"Geschäftsjahr des heutigen Tages; Vorgabe für Werkzeuge mit geschaeftsjahr"`
	Geschaeftsjahre  []statusYear     `json:"geschaeftsjahre,omitempty"`
	Pruefermodus     *statusExaminer  `json:"pruefermodus,omitempty"`
	Verbindung       statusConnection `json:"verbindung"`
	VerbundeneApps   int              `json:"verbundene_apps" jsonschema:"Zahl der Apps mit gültiger Zulassung"`
	Eingang          *inboxState      `json:"eingang,omitempty" jsonschema:"der Eingang, aus dem beleg_ablegen und kontoauszug_importieren lesen; die Dateien nennt eingang"`
	Leseordner       []string         `json:"leseordner,omitempty" jsonschema:"weitere Ordner, aus denen gelesen werden darf"`
	Exportordner     string           `json:"exportordner,omitempty" jsonschema:"Ordner, in den die Exporte schreiben"`
	Direktregeln     *statusRules     `json:"direktregeln,omitempty" jsonschema:"Schwellen, bis zu denen beleg_buchen ohne Rückfrage bucht"`
	Vorschlaege      []statusProposal `json:"stammdaten_vorschlaege,omitempty" jsonschema:"offene und in den letzten 14 Tagen entschiedene Vorschläge aus stammdaten_vorschlagen"`
}

type statusProposal struct {
	ID            uint   `json:"id"`
	Stand         string `json:"stand" jsonschema:"offen, uebernommen oder abgelehnt"`
	Begruendung   string `json:"begruendung"`
	EntschiedenAm string `json:"entschieden_am,omitempty"`
	Vermerk       string `json:"vermerk,omitempty" jsonschema:"Grund der Ablehnung"`
}

type statusRules struct {
	GrenzeBrutto     Betrag `json:"grenze_brutto" jsonschema:"direkt nur, was darunter liegt"`
	Mindestbuchungen int    `json:"mindestbuchungen" jsonschema:"frühere Buchungen des Lieferanten mit seiner bestätigten Standardgruppe"`
}

type statusTenant struct {
	ID                 string `json:"id" jsonschema:"mandant_id für schreibende Werkzeuge"`
	Name               string `json:"name"`
	Rechtsform         string `json:"rechtsform,omitempty"`
	GJBeginnMonat      int    `json:"gj_beginn_monat,omitempty" jsonschema:"Monat, in dem das Geschäftsjahr beginnt (1 = Januar)"`
	Besteuerung        string `json:"besteuerung,omitempty" jsonschema:"SOLL oder IST"`
	UStZeitraum        string `json:"ust_zeitraum,omitempty" jsonschema:"monat, quartal, jahr oder keiner: wie oft die UStVA fällig ist"`
	Dauerfrist         bool   `json:"dauerfrist" jsonschema:"Dauerfristverlängerung: die UStVA ist einen Monat später fällig"`
	FestgeschriebenBis string `json:"festgeschrieben_bis,omitempty" jsonschema:"bis zu diesem Tag sind die Buchungen festgeschrieben; Korrekturen nur als Storno im offenen Zeitraum"`
	LetzteSicherung    string `json:"letzte_sicherung,omitempty"`
}

var vatPeriodLabel = map[string]string{"month": "monat", "quarter": "quartal", "year": "jahr", "none": "keiner"}

type statusYear struct {
	Jahr  int    `json:"jahr"`
	Von   string `json:"von"`
	Bis   string `json:"bis"`
	Stand string `json:"stand" jsonschema:"offen, aufgestellt, festgestellt oder offengelegt"`
}

type statusExaminer struct {
	Aktiv bool   `json:"aktiv"`
	Bis   string `json:"bis,omitempty"`
	Grund string `json:"grund,omitempty"`
}

type statusConnection struct {
	App                string   `json:"app" jsonschema:"Name der App, wie sie sich angemeldet hat"`
	Erfassungsweg      string   `json:"erfassungsweg" jsonschema:"steht an jeder Buchung dieser App"`
	Rechte             []string `json:"rechte" jsonschema:"Rechte im geöffneten Mandanten; leer, wenn er für diese App nicht freigegeben ist"`
	MandantFreigegeben bool     `json:"mandant_freigegeben"`
}

var yearStand = map[domain.FiscalYearStatus]string{
	domain.FiscalYearOpen:      "offen",
	domain.FiscalYearPrepared:  "aufgestellt",
	domain.FiscalYearAdopted:   "festgestellt",
	domain.FiscalYearDisclosed: "offengelegt",
}

func registerStatus(tb *Toolbox) {
	AddInfo(tb, Spec{
		Name:  "status",
		Title: "Status von Buchfink",
		Description: "Überblick und Selbstauskunft: geöffneter Mandant (mandant_id für schreibende Werkzeuge), " +
			"Geschäftsjahre, Prüfermodus, Eingang mit Zahl der Dateien, Fassung und ob Buchfink für diese App bereit ist. " +
			"Zu Beginn jeder Sitzung und nach jedem Fehler mandant_gewechselt aufrufen.",
	}, func(ctx context.Context, c *Call, _ statusIn) (statusOut, error) {
		out := statusOut{
			Version: buildinfo.Version,
			Verbindung: statusConnection{
				App: c.App, Erfassungsweg: string(c.Channel), Rechte: scopesOrEmpty(c.Scopes),
			},
		}
		n, err := tb.store.CountConnected(ctx)
		if err != nil {
			return out, err
		}
		out.VerbundeneApps = n
		switch {
		case c.Tenant == nil:
			out.Hinweis = "In Buchfink ist kein Mandant geöffnet."
			out.NaechsterSchritt = "Den Menschen bitten, in Buchfink einen Mandanten zu öffnen; danach status erneut aufrufen."
			return out, nil
		case !c.Granted:
			out.Hinweis = "Diese App ist für den geöffneten Mandanten nicht freigegeben."
			out.NaechsterSchritt = "Die Anmeldung im Client wiederholen und in Buchfink für diesen Mandanten zustimmen."
			return out, nil
		}
		out.Verbindung.MandantFreigegeben = true
		if !c.Can(oauth.ScopeRead) {
			out.Bereit = true
			out.Hinweis = "Die Anmeldung erlaubt kein Lesen; Mandantenangaben fehlen deshalb."
			return out, nil
		}
		t := c.Tenant
		err = t.ReadAgent(ctx, func(ctx context.Context, s core.AgentServices) error {
			out.Mandant = &statusTenant{ID: t.ID(), Name: t.Name()}
			if settings, err := s.Settings.GetCompanySettings(ctx); err == nil && settings != nil {
				out.Mandant.Rechtsform = settings.LegalForm
				out.Mandant.GJBeginnMonat = settings.FiscalYearStartMonth
				out.Mandant.Besteuerung = settings.TaxationType
				out.Mandant.UStZeitraum = vatPeriodLabel[settings.VatPeriod]
				out.Mandant.Dauerfrist = settings.PermanentExtension
			}
			for _, tc := range c.Host.Config().Tenants {
				if tc.ID == t.ID() {
					out.Mandant.LetzteSicherung = tc.LastBackupAt
				}
			}
			years, err := s.Closing.FiscalYears(ctx)
			if err != nil {
				return err
			}
			for _, fy := range years {
				out.Geschaeftsjahre = append(out.Geschaeftsjahre, statusYear{
					Jahr: fy.Year, Von: fy.StartDate, Bis: fy.EndDate, Stand: yearStand[fy.Status],
				})
			}
			if out.Mandant.FestgeschriebenBis, err = s.Journal.CommittedThrough(ctx); err != nil {
				return err
			}
			year, err := t.CurrentFiscalYear(ctx)
			if err == nil {
				out.GJAktuell = year
			}
			dirs, err := folders(ctx, c, s)
			if err != nil {
				return err
			}
			dirs.EnsureInbox()
			state := inboxStateOf(dirs, time.Now())
			if state.Fehler != "" {
				out.Hinweis = state.Fehler + "."
			}
			out.Eingang, out.Leseordner, out.Exportordner = &state, dirs.Read, dirs.Export
			limits, err := policy.LimitsFor(ctx, s)
			if err != nil {
				return err
			}
			out.Direktregeln = &statusRules{GrenzeBrutto: betrag(limits.MaxGross), Mindestbuchungen: limits.MinBookings}
			out.Vorschlaege, err = statusProposals(ctx, s, time.Now())
			return err
		})
		if err != nil {
			return out, err
		}
		out.Bereit = true
		if ex := t.Examiner(); ex.Active {
			out.Pruefermodus = &statusExaminer{Aktiv: true, Bis: ex.Until, Grund: ex.Reason}
			out.Hinweis = strings.TrimSpace("Der Prüfermodus ist bis " + ex.Until + " aktiv: lesen ja, schreiben nein. " +
				out.Hinweis)
		} else {
			out.Pruefermodus = &statusExaminer{}
		}
		return out, nil
	})
}

func scopesOrEmpty(scopes []string) []string {
	if scopes == nil {
		return []string{}
	}
	return scopes
}

// statusProposals nennt die offenen Vorschläge für die Stammdaten und die,
// über die der Mensch in den letzten 14 Tagen entschieden hat.
func statusProposals(ctx context.Context, s core.AgentServices, now time.Time) ([]statusProposal, error) {
	recent, err := s.MasterData.Recent(ctx, 20)
	if err != nil {
		return nil, err
	}
	var out []statusProposal
	for _, p := range recent {
		entry := statusProposal{ID: p.ID, Stand: string(p.Status), Begruendung: p.Reason, Vermerk: p.DecisionNote}
		if p.DecidedAt != nil {
			if now.Sub(*p.DecidedAt) > 14*24*time.Hour {
				continue
			}
			entry.EntschiedenAm = p.DecidedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	return out, nil
}
