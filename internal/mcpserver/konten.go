package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- konten -----------------------------------------------------------------

type kontenIn struct {
	Art            KontenArt `json:"art"`
	Suche          string    `json:"suche,omitempty" jsonschema:"Teil von Nummer oder Bezeichnung (art=suche)"`
	NurBebucht     bool      `json:"nur_bebucht,omitempty" jsonschema:"nur Konten mit Buchungen im Geschäftsjahr"`
	Geschaeftsjahr int       `json:"geschaeftsjahr,omitempty"`
	Richtung       Richtung  `json:"richtung,omitempty" jsonschema:"für Buchungsgruppen und Steuerfälle: eingang (Vorgabe) oder ausgang"`
	Seite
}

type kontoZeile struct {
	Konto     string `json:"konto"`
	Name      string `json:"name"`
	Typ       string `json:"typ"`
	Kategorie string `json:"kategorie,omitempty"`
	Posten    string `json:"posten,omitempty"`
	Saldo     Betrag `json:"saldo"`
	Buchungen int    `json:"buchungen"`
	Gesperrt  bool   `json:"gesperrt,omitempty"`
}

type gruppeZeile struct {
	Schluessel  string `json:"schluessel"`
	Name        string `json:"name"`
	Kategorie   string `json:"kategorie"`
	Konto       string `json:"konto"`
	Satz        Satz   `json:"satz_vorgabe"`
	Hinweis     string `json:"hinweis,omitempty"`
	Sonderregel bool   `json:"sonderregel,omitempty" jsonschema:"Bewirtung, Geschenk, GWG o. Ä.: nie direkt buchbar"`
}

type steuerfallZeile struct {
	Schluessel string `json:"schluessel"`
	Name       string `json:"name"`
	Hinweis    string `json:"hinweis"`
	MitSatz    bool   `json:"mit_satz"`
	MitUstID   bool   `json:"mit_ust_id"`
}

type kontenOut struct {
	Konten          []kontoZeile      `json:"konten"`
	Buchungsgruppen []gruppeZeile     `json:"buchungsgruppen"`
	Steuerfaelle    []steuerfallZeile `json:"steuerfaelle"`
	Seitenende
}

func direction(s Richtung) domain.Direction {
	if s == "ausgang" {
		return domain.DirectionOutgoing
	}
	return domain.DirectionIncoming
}

// --- auswertung -------------------------------------------------------------

type auswertungIn struct {
	Art            AuswertungArt `json:"art" jsonschema:"susa (Summen und Salden), kontoblatt, bilanz, guv, ust (Umsatzsteuer-Übersicht), pruefbericht oder darlehen (Gesellschafterdarlehen mit Stand)"`
	Geschaeftsjahr int           `json:"geschaeftsjahr,omitempty"`
	Konto          Konto         `json:"konto,omitempty" jsonschema:"Sachkonto des SKR04 für art=kontoblatt; Buchungen eines Personenkontos liefert buchungen_suchen mit kontakt_id"`
	Von            Datum         `json:"von,omitempty" jsonschema:"kontoblatt, ust: Beginn; Vorgabe Beginn des Geschäftsjahres"`
	Bis            Datum         `json:"bis,omitempty" jsonschema:"kontoblatt, ust, pruefbericht, darlehen: Ende oder Stichtag; Vorgabe Ende des Geschäftsjahres bzw. heute"`
	Tiefe          Gliederung    `json:"tiefe,omitempty" jsonschema:"bilanz, guv"`
	Seite
}

type susaZeile struct {
	Konto string `json:"konto"`
	Name  string `json:"name"`
	Soll  Betrag `json:"soll"`
	Haben Betrag `json:"haben"`
	Saldo Betrag `json:"saldo" jsonschema:"Soll minus Haben"`
}

type kontoblattZeile struct {
	BuchungID   uint     `json:"buchung_id"`
	Nummer      string   `json:"nummer"`
	Datum       string   `json:"datum"`
	BelegNr     string   `json:"beleg_nr,omitempty"`
	Text        string   `json:"text"`
	Soll        Betrag   `json:"soll"`
	Haben       Betrag   `json:"haben"`
	Saldo       Betrag   `json:"saldo"`
	Gegenkonten []string `json:"gegenkonten"`
}

type kontoblatt struct {
	Konto  string            `json:"konto"`
	Name   string            `json:"name"`
	Von    string            `json:"von,omitempty"`
	Bis    string            `json:"bis,omitempty"`
	Anfang Betrag            `json:"anfangssaldo"`
	Soll   Betrag            `json:"soll"`
	Haben  Betrag            `json:"haben"`
	Ende   Betrag            `json:"endsaldo"`
	Zeilen []kontoblattZeile `json:"zeilen"`
}

type abschlussZeile struct {
	Schluessel  string `json:"schluessel"`
	Gliederung  string `json:"gliederung,omitempty"`
	Bezeichnung string `json:"bezeichnung"`
	Ebene       int    `json:"ebene"`
	Betrag      Betrag `json:"betrag"`
	Vorjahr     Betrag `json:"vorjahr"`
	Summe       bool   `json:"summe,omitempty"`
}

type ustZeile struct {
	Kennzahl    string  `json:"kz"`
	Bezeichnung string  `json:"bezeichnung"`
	Bemessung   *Betrag `json:"bemessungsgrundlage,omitempty"`
	Steuer      *Betrag `json:"steuer,omitempty"`
}

type befundZeile struct {
	Regel   string `json:"regel"`
	Schwere string `json:"schwere"`
	Objekt  string `json:"objekt,omitempty"`
	Text    string `json:"text"`
	Norm    string `json:"norm,omitempty"`
}

type auswertungOut struct {
	Art            string           `json:"art"`
	Geschaeftsjahr int              `json:"geschaeftsjahr"`
	Stichtag       string           `json:"stichtag,omitempty"`
	Susa           []susaZeile      `json:"susa"`
	Kontoblatt     *kontoblatt      `json:"kontoblatt,omitempty"`
	Aktiva         []abschlussZeile `json:"aktiva"`
	Passiva        []abschlussZeile `json:"passiva"`
	GuV            []abschlussZeile `json:"guv"`
	Jahresergebnis *Betrag          `json:"jahresergebnis,omitempty"`
	Bilanzsumme    *Betrag          `json:"bilanzsumme,omitempty"`
	Umsatzsteuer   []ustZeile       `json:"umsatzsteuer"`
	Zahllast       *Betrag          `json:"zahllast,omitempty"`
	Befunde        []befundZeile    `json:"befunde"`
	Darlehen       []darlehenZeile  `json:"darlehen"`
	Hinweis        string           `json:"hinweis,omitempty"`
	Seitenende
}

func emptyAuswertung(art string, year int) auswertungOut {
	return auswertungOut{Art: art, Geschaeftsjahr: year, Susa: []susaZeile{}, Aktiva: []abschlussZeile{},
		Passiva: []abschlussZeile{}, GuV: []abschlussZeile{}, Umsatzsteuer: []ustZeile{}, Befunde: []befundZeile{},
		Darlehen: []darlehenZeile{}}
}

func statementLines(lines []domain.StatementLine) []abschlussZeile {
	out := make([]abschlussZeile, 0, len(lines))
	for _, l := range lines {
		if l.Omitted {
			continue
		}
		out = append(out, abschlussZeile{Schluessel: l.Key, Gliederung: l.Ordinal, Bezeichnung: l.Label,
			Ebene: l.Level, Betrag: betrag(l.Amount), Vorjahr: betrag(l.PriorAmount), Summe: l.IsSubtotal})
	}
	return out
}

func registerKonten(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "konten",
		Title: "Konten, Buchungsgruppen, Steuerfälle",
		Description: "Nachschlagen: art=suche findet Konten des SKR04 mit Saldo im Geschäftsjahr; art=buchungsgruppen " +
			"listet die fachlichen Gruppen für beleg_buchen (Schlüssel, Konto, Vorgabesatz, ob eine Sonderregel gilt); " +
			"art=steuerfaelle die Steuerfälle. Für Salden aller Konten oder ein Kontoblatt auswertung.",
	}, func(ctx context.Context, c *Call, s *core.Services, in kontenIn) (kontenOut, error) {
		out := kontenOut{Konten: []kontoZeile{}, Buchungsgruppen: []gruppeZeile{}, Steuerfaelle: []steuerfallZeile{}}
		switch in.Art {
		case "buchungsgruppen":
			for _, g := range accounting.PostingGroups(direction(in.Richtung)) {
				out.Buchungsgruppen = append(out.Buchungsgruppen, gruppeZeile{
					Schluessel: g.Key, Name: g.Label, Kategorie: g.Category, Konto: g.Account,
					Satz: satzOf(g.DefaultRate), Hinweis: g.Hint,
					Sonderregel: g.Key == "gwg" || g.RecipientRequired || g.Limit != "" || g.InputTaxExcluded ||
						g.DeductibleQuota != "" || g.NonDeductibleAccount != "",
				})
			}
			out.Gesamt = len(out.Buchungsgruppen)
			return out, nil
		case "steuerfaelle":
			for _, t := range domain.TaxTreatments(direction(in.Richtung)) {
				out.Steuerfaelle = append(out.Steuerfaelle, steuerfallZeile{Schluessel: string(t.Treatment),
					Name: t.Label, Hinweis: t.Hint, MitSatz: t.RequiresRate, MitUstID: t.RequiresVatID})
			}
			out.Gesamt = len(out.Steuerfaelle)
			return out, nil
		case "suche", "":
		default:
			return out, inputError("art", "unbekannte Art %q (suche, buchungsgruppen, steuerfaelle)", in.Art)
		}
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return out, err
		}
		accounts, err := s.Accounting.GetAccounts(ctx, year)
		if err != nil {
			return out, err
		}
		query := strings.ToLower(strings.TrimSpace(in.Suche))
		rows := []kontoZeile{}
		for _, a := range accounts {
			if a.IsRange || a.IsReserved {
				continue
			}
			if in.NurBebucht && a.BookingsCount == 0 {
				continue
			}
			if query != "" && !strings.HasPrefix(a.Number, query) && !strings.Contains(strings.ToLower(a.Name), query) {
				continue
			}
			rows = append(rows, kontoZeile{Konto: a.Number, Name: a.Name, Typ: string(a.Type), Kategorie: a.Category,
				Posten: a.Posten, Saldo: betrag(a.Balance), Buchungen: a.BookingsCount, Gesperrt: !a.IsActive})
		}
		out.Konten, out.Seitenende, err = Paginate(rows, in.Seite)
		return out, err
	})

	AddRead(tb, Spec{
		Name:  "auswertung",
		Title: "Auswertung",
		Description: "Zahlen der Bücher: susa (Summen und Salden je Konto), kontoblatt (Buchungen eines Kontos mit " +
			"laufendem Saldo), bilanz, guv (mit Vorjahr), ust (Umsatzsteuer-Übersicht eines Zeitraums), pruefbericht " +
			"(Befunde des Prüflaufs, ohne ihn zu speichern), darlehen (Gesellschafterdarlehen mit Stand und Befunden " +
			"zum Fremdvergleich). Listen seitenweise. Für einzelne Buchungen buchung_lesen.",
	}, func(ctx context.Context, c *Call, s *core.Services, in auswertungIn) (auswertungOut, error) {
		year, err := yearOr(ctx, c, in.Geschaeftsjahr)
		if err != nil {
			return auswertungOut{}, err
		}
		out := emptyAuswertung(string(in.Art), year)
		from, err := in.Von.check("von")
		if err != nil {
			return out, err
		}
		to, err := in.Bis.check("bis")
		if err != nil {
			return out, err
		}
		switch in.Art {
		case "susa":
			susa, err := s.Accounting.GetSuSaOverview(ctx, year)
			if err != nil {
				return out, err
			}
			rows := []susaZeile{}
			for _, class := range susa.Classes {
				for _, a := range class.Accounts {
					rows = append(rows, susaZeile{Konto: a.Number, Name: a.Name, Soll: betrag(a.DebitSum),
						Haben: betrag(a.CreditSum), Saldo: betrag(a.DebitSum - a.CreditSum)})
				}
			}
			out.Susa, out.Seitenende, err = Paginate(rows, in.Seite)
			return out, err
		case "kontoblatt":
			if in.Konto == "" {
				return out, inputError("konto", "für das Kontoblatt fehlt das Konto")
			}
			if domain.IsLedgerAccount(string(in.Konto)) {
				return out, &ToolError{Code: CodeInvalidInput,
					Nachricht:        fmt.Sprintf("konto: %s ist ein Personenkonto; das Kontoblatt gibt es für Sachkonten", in.Konto),
					NaechsterSchritt: "Die Buchungen eines Kunden oder Lieferanten liefert buchungen_suchen mit kontakt_id.",
					Werkzeug:         "buchungen_suchen"}
			}
			if from == "" || to == "" {
				fy, err := s.Closing.PeriodOf(ctx, year)
				if err != nil {
					return out, err
				}
				if from == "" {
					from = fy.StartDate
				}
				if to == "" {
					to = fy.EndDate
				}
			}
			ledger, err := s.Accounting.GetAccountLedgerRange(ctx, string(in.Konto), from, to)
			if err != nil {
				return out, err
			}
			rows := make([]kontoblattZeile, 0, len(ledger.Rows))
			for _, r := range ledger.Rows {
				counter := make([]string, 0, len(r.CounterAccounts))
				for _, ca := range r.CounterAccounts {
					counter = append(counter, ca.Account)
				}
				rows = append(rows, kontoblattZeile{BuchungID: r.EntryID, Nummer: r.EntryNumber, Datum: r.BookingDate,
					BelegNr: r.DocumentNumber, Text: r.Description, Soll: betrag(r.DebitAmount),
					Haben: betrag(r.CreditAmount), Saldo: betrag(r.RunningBalance), Gegenkonten: counter})
			}
			page, end, err := Paginate(rows, in.Seite)
			if err != nil {
				return out, err
			}
			out.Kontoblatt = &kontoblatt{Konto: ledger.Account.Number, Name: ledger.Account.Name, Von: from, Bis: to,
				Anfang: betrag(ledger.OpeningBalance), Soll: betrag(ledger.TotalDebit), Haben: betrag(ledger.TotalCredit),
				Ende: betrag(ledger.ClosingBalance), Zeilen: page}
			out.Seitenende = end
			return out, nil
		case "bilanz", "guv":
			depth, err := pick("tiefe", in.Tiefe, depths)
			if err != nil {
				return out, err
			}
			if depth == "" {
				depth = domain.DepthFull
			}
			st, err := s.Statement.Statement(ctx, year, depth)
			if err != nil {
				return out, err
			}
			if in.Art == "bilanz" {
				out.Aktiva, out.Passiva = statementLines(st.Assets), statementLines(st.Liabilities)
				out.Bilanzsumme = optBetrag(st.BalanceSheetTotal)
			} else {
				out.GuV = statementLines(st.Income)
			}
			out.Jahresergebnis = optBetrag(st.NetIncome)
			return out, nil
		case "ust":
			if from == "" || to == "" {
				fy, err := s.Closing.PeriodOf(ctx, year)
				if err != nil {
					return out, err
				}
				if from == "" {
					from = fy.StartDate
				}
				if to == "" {
					to = fy.EndDate
				}
			}
			sum, err := s.Vat.Summary(ctx, year, from, to)
			if err != nil {
				return out, err
			}
			out.Umsatzsteuer = vatLines(sum)
			out.Zahllast = optBetrag(sum.Payable)
			out.Hinweis = "Für die Voranmeldung eines Zeitraums erklaerung mit art=ustva."
			return out, nil
		case "pruefbericht":
			cutoff := to
			if cutoff == "" {
				cutoff = time.Now().Format("2006-01-02")
			}
			run, err := s.Checks.Preview(ctx, service.CheckRequest{FiscalYear: year, CutoffDate: cutoff})
			if err != nil {
				return out, err
			}
			out.Stichtag = run.CutoffDate
			rows := make([]befundZeile, 0, len(run.Findings))
			for _, f := range run.Findings {
				obj := f.ObjectName
				if obj == "" && f.ObjectID != "" {
					obj = f.ObjectType + " " + f.ObjectID
				}
				rows = append(rows, befundZeile{Regel: f.Rule, Schwere: string(f.Severity), Objekt: obj,
					Text: f.Message, Norm: f.Reference})
			}
			out.Befunde, out.Seitenende, err = Paginate(rows, in.Seite)
			return out, err
		case "darlehen":
			cutoff := to
			if cutoff == "" {
				cutoff = today()
			}
			loans, err := s.Shareholders.Loans(ctx, cutoff)
			if err != nil {
				return out, err
			}
			rows := make([]darlehenZeile, 0, len(loans))
			for i := range loans {
				rows = append(rows, darlehenOf(&loans[i]))
			}
			out.Stichtag = cutoff
			out.Darlehen, out.Seitenende, err = Paginate(rows, in.Seite)
			return out, err
		}
		return out, inputError("art", "unbekannte Art %q (susa, kontoblatt, bilanz, guv, ust, pruefbericht, darlehen)", in.Art)
	})
}

func vatLines(sum *domain.VatSummary) []ustZeile {
	out := []ustZeile{}
	for _, f := range sum.TaxableRevenue {
		out = append(out, ustZeile{Bezeichnung: "Steuerpflichtige Umsätze " + string(satzOf(f.Rate)) + " %",
			Bemessung: optBetrag(f.Net), Steuer: optBetrag(f.Tax)})
	}
	add := func(label string, base, tax *domain.Cents) {
		z := ustZeile{Bezeichnung: label}
		if base != nil {
			z.Bemessung = optBetrag(*base)
		}
		if tax != nil {
			z.Steuer = optBetrag(*tax)
		}
		out = append(out, z)
	}
	add("Umsätze zum Nullsteuersatz", &sum.ZeroRatedRevenue, nil)
	add("Steuerfreie Umsätze", &sum.ExemptRevenue, nil)
	add("Leistungen mit Steuerschuld des Empfängers", &sum.ReverseChargeSupply, nil)
	add("Bezogene Leistungen nach § 13b UStG", &sum.ReverseChargeBase, &sum.ReverseChargeTax)
	add("Innergemeinschaftliche Erwerbe", &sum.IntraCommunityAcquisitionBase, &sum.IntraCommunityAcquisitionTax)
	add("Umsatzsteuer gesamt", nil, &sum.TotalOwedTax)
	add("Vorsteuer", nil, &sum.InputTax)
	return out
}
