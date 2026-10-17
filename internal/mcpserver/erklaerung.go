package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// wert ist der Wert einer Zeile einer Erklärung.
type wert struct {
	Betrag *Betrag `json:"betrag,omitempty"`
	Text   *string `json:"text,omitempty"`
	Ja     *bool   `json:"ja,omitempty"`
	Datum  *string `json:"datum,omitempty"`
}

func wertOf(v declaration.Value) wert {
	w := wert{Text: v.Text, Ja: v.Bool, Datum: v.Date}
	if v.Cents != nil {
		w.Betrag = optBetrag(*v.Cents)
	}
	return w
}

func (w wert) value(field string) (declaration.Value, error) {
	set := 0
	var v declaration.Value
	if w.Betrag != nil {
		c, err := w.Betrag.required(field + ".betrag")
		if err != nil {
			return v, err
		}
		v, set = declaration.Cents(c), set+1
	}
	if w.Text != nil {
		v, set = declaration.Text(*w.Text), set+1
	}
	if w.Ja != nil {
		v, set = declaration.Bool(*w.Ja), set+1
	}
	if w.Datum != nil {
		if err := domain.CheckDate(field+".datum", *w.Datum); err != nil {
			return v, inputError(field+".datum", "%v", err)
		}
		v, set = declaration.Date(*w.Datum), set+1
	}
	if set != 1 {
		return v, inputError(field, "genau eines von betrag, text, ja oder datum angeben")
	}
	return v, nil
}

var declarationKinds = map[string]declaration.Kind{
	"ust_jahr": declaration.KindVatYear, "kst": declaration.KindCorporateTax, "gewst": declaration.KindTradeTax,
}

// --- erklaerung -------------------------------------------------------------

type erklaerungIn struct {
	Art      string `json:"art" jsonschema:"ustva, zm, ust_jahr, kst oder gewst"`
	Zeitraum string `json:"zeitraum" jsonschema:"ustva und zm: 2026-03 oder 2026-Q1; Jahreserklärungen: 2026"`
	NurOffen bool   `json:"nur_offen,omitempty" jsonschema:"Jahreserklärungen: nur Zeilen, die noch eine Angabe brauchen"`
	Formular string `json:"formular,omitempty" jsonschema:"Jahreserklärungen: nur dieses Formular, z. B. kst1"`
}

type kennzahl struct {
	Kz          string  `json:"kz"`
	Bezeichnung string  `json:"bezeichnung"`
	Bemessung   *Betrag `json:"bemessungsgrundlage,omitempty"`
	Steuer      *Betrag `json:"steuer,omitempty"`
}

type zmZeile struct {
	Land    string `json:"land"`
	UstID   string `json:"ust_id"`
	Art     string `json:"art" jsonschema:"L Lieferung, S sonstige Leistung, D Dreiecksgeschäft"`
	Betrag  Betrag `json:"betrag"`
	Kontakt string `json:"kontakt,omitempty"`
}

type herkunftKonto struct {
	Konto       string `json:"konto"`
	Bezeichnung string `json:"bezeichnung"`
	Beitrag     Betrag `json:"beitrag"`
}

type herkunft struct {
	Art          string          `json:"art"`
	Konten       []herkunftKonto `json:"konten"`
	AusZeilen    []string        `json:"aus_zeilen,omitempty"`
	Rechnung     string          `json:"rechnung,omitempty"`
	Erlaeuterung string          `json:"erlaeuterung,omitempty"`
}

type erklaerungZeile struct {
	Schluessel      string   `json:"schluessel"`
	Zeile           string   `json:"zeile,omitempty"`
	Kz              string   `json:"kz,omitempty"`
	Bezeichnung     string   `json:"bezeichnung"`
	Wert            wert     `json:"wert"`
	Status          string   `json:"status" jsonschema:"auto, vorgeschlagen, eingabe, offen oder entfaellt"`
	Zustaendig      string   `json:"zustaendig" jsonschema:"buecher, agent_rueckfrage oder mensch"`
	EingabeMoeglich bool     `json:"eingabe_moeglich"`
	Pflicht         bool     `json:"pflicht"`
	Herkunft        herkunft `json:"herkunft"`
	Hinweise        []string `json:"hinweise"`
}

type erklaerungAbschnitt struct {
	ID     string            `json:"id"`
	Titel  string            `json:"titel"`
	Zeilen []erklaerungZeile `json:"zeilen"`
}

type erklaerungFormular struct {
	ID           string                `json:"id"`
	Titel        string                `json:"titel"`
	Erforderlich string                `json:"erforderlich"`
	Grund        string                `json:"grund,omitempty"`
	Abschnitte   []erklaerungAbschnitt `json:"abschnitte"`
}

type erklaerungBefund struct {
	Schwere string `json:"schwere"`
	Code    string `json:"code,omitempty"`
	Zeile   string `json:"zeile,omitempty"`
	Text    string `json:"text"`
}

type erklaerungOut struct {
	Art              string               `json:"art"`
	Zeitraum         string               `json:"zeitraum"`
	Status           string               `json:"status"`
	Faellig          string               `json:"faellig,omitempty"`
	Zahllast         *Betrag              `json:"zahllast,omitempty" jsonschema:"positiv Zahlung, negativ Erstattung"`
	Kennzahlen       []kennzahl           `json:"kennzahlen" jsonschema:"UStVA: Kennziffern wie in Mein ELSTER"`
	ZM               []zmZeile            `json:"zm_zeilen"`
	Ergebnis         []kennzahl           `json:"ergebnis" jsonschema:"Jahreserklärungen: die maßgeblichen Ergebniszeilen"`
	Formulare        []erklaerungFormular `json:"formulare"`
	Abgabereif       bool                 `json:"abgabereif"`
	ZahlenHash       string               `json:"zahlen_hash,omitempty"`
	Befunde          []erklaerungBefund   `json:"befunde"`
	Uebermittelt     string               `json:"uebermittelt_am,omitempty"`
	Transferticket   string               `json:"transferticket,omitempty"`
	NaechsterSchritt string               `json:"naechster_schritt"`
}

func emptyErklaerung(art, zeitraum string) erklaerungOut {
	return erklaerungOut{Art: art, Zeitraum: zeitraum, Kennzahlen: []kennzahl{}, ZM: []zmZeile{}, Ergebnis: []kennzahl{},
		Formulare: []erklaerungFormular{}, Befunde: []erklaerungBefund{}}
}

func lineOf(l declaration.Line) erklaerungZeile {
	z := erklaerungZeile{Schluessel: l.Key, Zeile: l.Row, Kz: l.Code, Bezeichnung: l.Label, Wert: wertOf(l.Value),
		Status: string(l.Status), Zustaendig: string(l.Owner), EingabeMoeglich: l.InputAllowed, Pflicht: l.Required,
		Herkunft: herkunft{Art: string(l.Origin.Kind), Konten: []herkunftKonto{}, AusZeilen: l.Origin.FromLines,
			Rechnung: l.Origin.Formula, Erlaeuterung: l.Origin.Text},
		Hinweise: orEmpty(l.Notes)}
	for _, a := range l.Origin.Accounts {
		z.Herkunft.Konten = append(z.Herkunft.Konten, herkunftKonto{Konto: a.Account, Bezeichnung: a.Name, Beitrag: betrag(a.Amount)})
	}
	return z
}

func declarationOut(art string, d *declaration.Declaration, opts declaration.ViewOptions) erklaerungOut {
	out := emptyErklaerung(art, strconv.Itoa(d.Period.Year))
	overview := d.Overview()
	out.Status, out.Abgabereif, out.ZahlenHash = string(d.Status), overview.Complete, d.FiguresHash
	for _, r := range overview.Result {
		k := kennzahl{Kz: r.Key, Bezeichnung: r.Label}
		if r.Value.Cents != nil {
			k.Steuer = optBetrag(*r.Value.Cents)
		}
		out.Ergebnis = append(out.Ergebnis, k)
	}
	view := d.View(opts)
	for _, f := range view.Forms {
		form := erklaerungFormular{ID: f.ID, Titel: f.Title, Erforderlich: string(f.Required), Grund: f.Reason,
			Abschnitte: []erklaerungAbschnitt{}}
		for _, s := range f.Sections {
			sec := erklaerungAbschnitt{ID: s.ID, Titel: s.Title, Zeilen: []erklaerungZeile{}}
			for _, l := range s.Lines {
				sec.Zeilen = append(sec.Zeilen, lineOf(l))
			}
			if len(sec.Zeilen) > 0 {
				form.Abschnitte = append(form.Abschnitte, sec)
			}
		}
		out.Formulare = append(out.Formulare, form)
	}
	for _, f := range d.Findings {
		out.Befunde = append(out.Befunde, erklaerungBefund{Schwere: string(f.Severity), Code: f.Code, Zeile: f.LineKey, Text: f.Text})
	}
	if d.Submission != nil {
		out.Uebermittelt, out.Transferticket = d.Submission.At, d.Submission.TransferTicket
	}
	out.NaechsterSchritt = "Offene Zeilen mit erklaerung_eingabe füllen (nach Rückfrage), dann erklaerung_exportieren; übermitteln und das Transferticket erfassen nur der Mensch."
	return out
}

// --- erklaerung_eingabe -----------------------------------------------------

type erklaerungEingabeIn struct {
	WriteArgs
	Art        string `json:"art" jsonschema:"ust_jahr, kst oder gewst"`
	Jahr       int    `json:"jahr"`
	Schluessel string `json:"schluessel" jsonschema:"Schlüssel der Zeile aus erklaerung"`
	Wert       wert   `json:"wert" jsonschema:"genau eines von betrag, text, ja, datum"`
	Quelle     string `json:"quelle,omitempty" jsonschema:"bescheid, beschluss, beleg, mensch oder buecher; Vorgabe mensch"`
	BelegID    uint   `json:"beleg_id,omitempty"`
	Notiz      string `json:"notiz,omitempty"`
}

type erklaerungEingabeOut struct {
	Wiederholung
	Schluessel string `json:"schluessel"`
	Wert       wert   `json:"wert"`
	Bestaetigt bool   `json:"bestaetigt" jsonschema:"false: bleibt ein Vorschlag, bis der Mensch ihn in der App bestätigt"`
	Hinweis    string `json:"hinweis"`
}

func (o erklaerungEingabeOut) Ref() string { return "eingabe:" + o.Schluessel }

// --- Exporte ----------------------------------------------------------------

type erklaerungExportIn struct {
	TenantArgs
	Art  string `json:"art" jsonschema:"ust_jahr, kst oder gewst"`
	Jahr int    `json:"jahr"`
}

type exportOut struct {
	Pfad             string             `json:"pfad"`
	SHA256           string             `json:"sha256"`
	ZahlenHash       string             `json:"zahlen_hash,omitempty"`
	Format           string             `json:"format"`
	Befunde          []erklaerungBefund `json:"befunde"`
	NaechsterSchritt string             `json:"naechster_schritt"`
}

type ebilanzExportIn struct {
	TenantArgs
	Jahr      int    `json:"jahr"`
	Taxonomie string `json:"taxonomie,omitempty" jsonschema:"z. B. 6.9 oder 6.10; Vorgabe die Regelfassung des Wirtschaftsjahres"`
	Kleinst   bool   `json:"kleinstkapitalgesellschaft,omitempty" jsonschema:"GuV nach § 275 Abs. 5 HGB"`
	Umschlag  bool   `json:"umschlag,omitempty" jsonschema:"in den Nutzdatenteil der ELSTER-Übermittlung betten"`
}

func registerErklaerung(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "erklaerung",
		Title: "Anmeldung oder Erklärung anzeigen",
		Description: "Zeigt eine Anmeldung oder Erklärung formularnah als Eingabehilfe für Mein ELSTER: ustva und zm " +
			"je Zeitraum mit Kennzahlen und Zahllast; ust_jahr, kst und gewst je Jahr mit Formularen, Zeilen, Wert, " +
			"Status, Zuständigkeit und Herkunft (Konten, Rechnung). nur_offen zeigt, was noch eine Angabe braucht. " +
			"Danach erklaerung_eingabe und erklaerung_exportieren. Übermitteln kann nur der Mensch.",
	}, func(ctx context.Context, c *Call, s *core.Services, in erklaerungIn) (erklaerungOut, error) {
		out := emptyErklaerung(in.Art, in.Zeitraum)
		switch in.Art {
		case "ustva":
			ret, err := s.VatReturns.Draft(ctx, in.Zeitraum)
			if err != nil {
				return out, err
			}
			out.Zeitraum, out.Status, out.Faellig = ret.PeriodKey, string(ret.Status), ret.DueDate
			out.Zahllast = optBetrag(ret.Payable)
			out.Uebermittelt, out.Transferticket = ret.SubmittedAt, ret.TransferTicket
			for _, f := range ret.Figures {
				k := kennzahl{Kz: f.Code, Bezeichnung: f.Label}
				if f.HasBase {
					k.Bemessung = optBetrag(f.Base)
				}
				if f.HasTax {
					k.Steuer = optBetrag(f.Tax)
				}
				out.Kennzahlen = append(out.Kennzahlen, k)
			}
			for _, f := range ret.Findings {
				out.Befunde = append(out.Befunde, erklaerungBefund{Schwere: "hinweis", Text: f})
			}
			out.Abgabereif = len(ret.Findings) == 0
			out.NaechsterSchritt = "Kennzahlen in Mein ELSTER übertragen lassen; Festschreibung und Transferticket erfasst nur der Mensch in der App. Danach pflicht_vorbereitet."
			return out, nil
		case "zm":
			ret, err := s.ZM.Draft(ctx, in.Zeitraum)
			if err != nil {
				return out, err
			}
			out.Zeitraum, out.Status, out.Faellig = ret.PeriodKey, string(ret.Status), ret.DueDate
			out.Uebermittelt, out.Transferticket = ret.SubmittedAt, ret.TransferTicket
			for _, l := range ret.Lines {
				out.ZM = append(out.ZM, zmZeile{Land: l.CountryCode, UstID: l.VatID, Art: string(l.Kind), Betrag: betrag(l.Amount),
					Kontakt: l.ContactName})
			}
			for _, f := range ret.Findings {
				out.Befunde = append(out.Befunde, erklaerungBefund{Schwere: "hinweis", Text: f})
			}
			out.Abgabereif = len(ret.Findings) == 0
			out.NaechsterSchritt = "Zeilen im BZSt-Online-Portal erfassen lassen; das Transferticket erfasst nur der Mensch in der App."
			return out, nil
		}
		kind, ok := declarationKinds[in.Art]
		if !ok {
			return out, inputError("art", "ustva, zm, ust_jahr, kst oder gewst, nicht %q", in.Art)
		}
		year, err := strconv.Atoi(in.Zeitraum)
		if err != nil || year < 2000 {
			return out, inputError("zeitraum", "für %s ein Jahr wie 2026, nicht %q", in.Art, in.Zeitraum)
		}
		d, err := s.Declarations.Build(ctx, kind, year)
		if err != nil {
			return out, err
		}
		return declarationOut(in.Art, d, declaration.ViewOptions{Form: in.Formular, Pending: in.NurOffen, Applicable: true}), nil
	})

	AddWrite(tb, Spec{
		Name:  "erklaerung_eingabe",
		Title: "Angabe zur Erklärung setzen",
		Tier:  TierConfirm,
		Description: "Setzt eine Angabe zu einer Zeile einer Jahreserklärung, die eine Eingabe zulässt (Wert aus " +
			"Bescheid, Beschluss, Beleg). Vorher im Chat bestätigen lassen. Bei Zeilen, die dem Menschen vorbehalten " +
			"sind, bleibt die Angabe ein Vorschlag bis zur Bestätigung in der App; eine vom Menschen bestätigte Angabe " +
			"ändert nur er (nur_in_app).",
	}, func(ctx context.Context, c *Call, s *core.Services, in erklaerungEingabeIn) (erklaerungEingabeOut, error) {
		kind, ok := declarationKinds[in.Art]
		if !ok {
			return erklaerungEingabeOut{}, inputError("art", "ust_jahr, kst oder gewst, nicht %q", in.Art)
		}
		v, err := in.Wert.value("wert")
		if err != nil {
			return erklaerungEingabeOut{}, err
		}
		req := service.InputRequest{Kind: kind, Year: in.Jahr, Key: in.Schluessel, Value: v,
			Source: declaration.InputSource(in.Quelle), Note: in.Notiz}
		if in.BelegID != 0 {
			id := in.BelegID
			req.ReceiptID = &id
		}
		input, err := s.Declarations.SetInput(ctx, req)
		if err != nil {
			return erklaerungEingabeOut{}, err
		}
		out := erklaerungEingabeOut{Schluessel: input.Key, Wert: wertOf(input.Value), Bestaetigt: input.Confirmed,
			Hinweis: "Die Angabe gilt."}
		if !input.Confirmed {
			out.Hinweis = "Die Angabe ist ein Vorschlag; der Mensch bestätigt sie in der App."
		}
		return out, nil
	})

	AddFlow(tb, Spec{
		Name:  "erklaerung_exportieren",
		Title: "Erklärung exportieren",
		Tier:  TierDirect, Idempotent: true, ExaminerExempt: true,
		Description: "Schreibt eine Jahreserklärung (ust_jahr, kst, gewst) als JSON (Schema buchfink.erklaerung/1) " +
			"in den Exportordner und hält den Export fest; liefert Pfad, Hashwerte und Befunde. Die Bücher ändert " +
			"das nicht. Übermitteln und Transferticket erfassen nur der Mensch.",
	}, func(ctx context.Context, c *Call, t *core.Tenant, in erklaerungExportIn) (exportOut, error) {
		out := exportOut{Format: "json", Befunde: []erklaerungBefund{}}
		kind, ok := declarationKinds[in.Art]
		if !ok {
			return out, inputError("art", "ust_jahr, kst oder gewst, nicht %q", in.Art)
		}
		var dir string
		if err := t.Read(ctx, func(ctx context.Context, s *core.Services) error {
			dirs, err := folders(ctx, c, s)
			if err != nil {
				return err
			}
			probe, err := dirs.ExportPath("erklaerung.json")
			dir = filepath.Dir(probe)
			return err
		}); err != nil {
			return out, err
		}
		exp, err := t.ExportDeclaration(ctx, kind, in.Jahr, dir)
		if err != nil {
			return out, err
		}
		out.Pfad, out.SHA256, out.ZahlenHash = exp.Path, exp.SHA256, exp.FiguresHash
		if d, err := t.Declaration(ctx, kind, in.Jahr); err == nil {
			for _, f := range d.Findings {
				out.Befunde = append(out.Befunde, erklaerungBefund{Schwere: string(f.Severity), Code: f.Code, Zeile: f.LineKey, Text: f.Text})
			}
		}
		out.NaechsterSchritt = "Werte in Mein ELSTER übertragen lassen; das Transferticket erfasst der Mensch in der App. Danach pflicht_vorbereitet."
		return out, nil
	})

	AddFlow(tb, Spec{
		Name:  "ebilanz_exportieren",
		Title: "E-Bilanz exportieren",
		Tier:  TierDirect, Idempotent: true, ExaminerExempt: true,
		Description: "Prüft die E-Bilanz eines Geschäftsjahres (Zuordnung der Konten, Pflichtfelder der Taxonomie) " +
			"und schreibt die XBRL-Instanz in den Exportordner; liefert Pfad und Befunde. Blockierende Befunde " +
			"verhindern den Export. Übermitteln (z. B. mit myebilanz oder ERiC) kann nur der Mensch.",
	}, func(ctx context.Context, c *Call, t *core.Tenant, in ebilanzExportIn) (exportOut, error) {
		out := exportOut{Format: "xbrl", Befunde: []erklaerungBefund{}}
		opts := service.EBilanzOptions{Version: in.Taxonomie, MicroBilG: in.Kleinst, Envelope: in.Umschlag}
		var data []byte
		var path string
		if err := t.Read(ctx, func(ctx context.Context, s *core.Services) error {
			res, err := s.EBilanz.Generate(ctx, in.Jahr, opts)
			if err != nil {
				return err
			}
			for _, f := range res.Report.Findings {
				out.Befunde = append(out.Befunde, erklaerungBefund{Schwere: string(f.Severity), Code: f.Code, Text: f.Message})
			}
			if err := res.Report.BlockingError(); err != nil {
				return &ToolError{Code: CodeRule, Nachricht: err.Error(),
					NaechsterSchritt: "Die blockierenden Zuordnungen klären (auswertung, konten); einige setzt nur der Mensch in der App."}
			}
			xml, err := s.EBilanz.ExportXBRLWith(ctx, in.Jahr, opts)
			if err != nil {
				return err
			}
			data = []byte(xml)
			dirs, err := folders(ctx, c, s)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			path, err = dirs.ExportPath(fmt.Sprintf("e-bilanz-%d-%s.xml", in.Jahr, hex.EncodeToString(sum[:])[:12]))
			return err
		}); err != nil {
			return out, err
		}
		if err := writeExport(path, data); err != nil {
			return out, err
		}
		sum := sha256.Sum256(data)
		out.Pfad, out.SHA256 = path, hex.EncodeToString(sum[:])
		out.NaechsterSchritt = "Die Datei übermittelt der Mensch (myebilanz, ERiC); danach in der App den Nachweis erfassen."
		return out, nil
	})
}

// writeExport legt eine Exportdatei über eine Zwischendatei ab, damit kein
// halber Export unter dem endgültigen Namen liegt.
func writeExport(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".export-*")
	if err != nil {
		return fmt.Errorf("die Exportdatei ließ sich nicht schreiben: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("die Exportdatei ließ sich nicht schreiben: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("die Exportdatei ließ sich nicht ablegen: %w", err)
	}
	return nil
}
