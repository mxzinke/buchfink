package mcpserver

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

type darlehenAntwort struct {
	Vorschau   bool `json:"vorschau"`
	Wiederholt bool `json:"wiederholt"`
	Darlehen   struct {
		ID             uint   `json:"id"`
		Konto          string `json:"konto"`
		Zinskonto      string `json:"zinskonto"`
		Zinssatz       string `json:"zinssatz"`
		Vergleichszins string `json:"vergleichszins"`
		BelegID        *uint  `json:"beleg_id"`
		Stand          string `json:"stand"`
		Befunde        []struct {
			Text string `json:"text"`
		} `json:"befunde"`
	} `json:"darlehen"`
}

// shareholder trägt Anna Bauer als alleinige Gesellschafterin ein und legt
// sie als Kontakt an.
func (h *harness) shareholder() uint {
	h.t.Helper()
	var id uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		cfg, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		cfg.ShareCapital = 2_500_000
		cfg.Shareholders = []domain.CompanyShareholder{{Name: "Anna Bauer", ShareCapital: 2_500_000}}
		if err := s.Settings.UpdateCompanySettings(ctx, cfg); err != nil {
			return err
		}
		anna := domain.Contact{Type: domain.ContactTypeVendor, Name: "Anna Bauer", CountryCode: "DE", IsPrivate: true}
		err = s.Contacts.SaveContact(ctx, &anna)
		id = anna.ID
		return err
	})
	return id
}

// Den Darlehensvertrag mit einer Gesellschafterin erfasst und berichtigt der
// Agent nach Rückfrage; auswertung zeigt ihn mit Stand und Befunden.
func TestShareholderLoanTool(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	anna := h.shareholder()
	a := h.agent("")
	var contract ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("darlehensvertrag.pdf", pdf("Darlehensvertrag"))}},
		"belegdatum": "2025-12-15", "aussteller": "Anna Bauer", "brutto": "50000.00"}), &contract)

	loan := map[string]any{"idempotenz_schluessel": "darlehen-anna", "kontakt_id": anna, "richtung": "an_gesellschaft",
		"betrag": "50000.00", "zinssatz": "3.5", "vertrag_vom": "2025-12-15", "beginn": "2026-01-02",
		"faellig_am": "2029-12-31", "beleg_id": contract.Beleg.ID}
	preview := a.args(loan)
	preview["nur_vorschau"] = true
	var shown darlehenAntwort
	a.ok("darlehen_erfassen", preview, &shown)
	if !shown.Vorschau || shown.Darlehen.ID != 0 || shown.Darlehen.Konto != "3642" || shown.Darlehen.Zinskonto != "7317" ||
		shown.Darlehen.Zinssatz != "3.50" || len(shown.Darlehen.Befunde) == 0 {
		t.Fatalf("Vorschau: %+v", shown)
	}
	var listed struct {
		Darlehen []struct {
			ID    uint   `json:"id"`
			Stand string `json:"stand"`
		} `json:"darlehen"`
		Stichtag string `json:"stichtag"`
	}
	a.ok("auswertung", map[string]any{"art": "darlehen"}, &listed)
	if len(listed.Darlehen) != 0 {
		t.Fatalf("die Vorschau hat gespeichert: %+v", listed)
	}

	var booked, replay darlehenAntwort
	a.ok("darlehen_erfassen", a.args(loan), &booked)
	a.ok("darlehen_erfassen", a.args(loan), &replay)
	if booked.Vorschau || booked.Darlehen.ID == 0 || booked.Darlehen.BelegID == nil || *booked.Darlehen.BelegID != contract.Beleg.ID ||
		!replay.Wiederholt || replay.Darlehen.ID != booked.Darlehen.ID {
		t.Fatalf("erfasst %+v, wiederholt %+v", booked, replay)
	}

	var stranger kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Bernd Fremd", "plz": "50667"}), &stranger)
	third := a.args(loan)
	third["kontakt_id"], third["idempotenz_schluessel"] = stranger.Kontakt.ID, "darlehen-fremd"
	if e := a.fail("darlehen_erfassen", third, CodeRule); !strings.Contains(e.Nachricht, "Gesellschafterliste") {
		t.Errorf("Darlehen eines Dritten: %+v", e)
	}

	fix := a.args(loan)
	fix["idempotenz_schluessel"], fix["darlehen_id"] = "darlehen-anna-fix", booked.Darlehen.ID
	fix["vergleichszins"] = "4.5"
	a.fail("darlehen_erfassen", fix, CodeInvalidInput)
	fix["grund"] = "Vergleichszins nachgetragen"
	if e := a.fail("darlehen_erfassen", fix, CodeRule); !strings.Contains(e.Nachricht, "Nachweis") {
		t.Errorf("Vergleichszins ohne Nachweis: %+v", e)
	}
	fix["vergleichszins_quelle"] = "Angebot der Hausbank vom 10.12.2025"
	var corrected darlehenAntwort
	a.ok("darlehen_erfassen", fix, &corrected)
	if corrected.Darlehen.ID != booked.Darlehen.ID || corrected.Darlehen.Vergleichszins != "4.50" {
		t.Errorf("Berichtigung: %+v", corrected)
	}

	a.ok("auswertung", map[string]any{"art": "darlehen", "bis": "2026-12-31"}, &listed)
	if len(listed.Darlehen) != 1 || listed.Darlehen[0].ID != booked.Darlehen.ID || listed.Darlehen[0].Stand != "0.00" ||
		listed.Stichtag != "2026-12-31" {
		t.Errorf("Darlehen in der Auswertung: %+v", listed)
	}
}

// Die Ergebnisverwendung bucht der Agent erst, wenn der Mensch den Abschluss
// festgestellt hat, und nicht mit einem Beschluss, der davor liegt.
func TestAppropriationTool(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	h.warmRenderer()
	a := h.agent("")
	h.ui(func(ctx context.Context, s *core.Services) error {
		for _, def := range domain.AllNotesSections() {
			if _, err := s.Appropriation.SaveNotesText(ctx, 2025, def.Section, "Keine Angaben."); err != nil {
				return err
			}
		}
		_, err := s.Journal.Post(ctx, &domain.JournalEntry{BookingDate: "2026-01-01", DocumentDate: "2026-01-01",
			ServiceDateFrom: "2026-01-01", ServiceDateTo: "2026-01-01", Description: "Saldenvortrag",
			Source: domain.EntrySourceOpening, Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
				{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000}}})
		return err
	})

	type verwendung struct {
		Vorschau            bool     `json:"vorschau"`
		Wiederholt          bool     `json:"wiederholt"`
		Festgestellt        bool     `json:"festgestellt"`
		Buchungsjahr        int      `json:"buchungsjahr"`
		Ergebnis            string   `json:"verwendbares_ergebnis"`
		Kapitalertragsteuer string   `json:"kapitalertragsteuer"`
		Soli                string   `json:"solidaritaetszuschlag"`
		Vortrag             string   `json:"vortrag"`
		BuchungID           *uint    `json:"buchung_id"`
		Hinweise            []string `json:"hinweise"`
		Zeilen              []struct {
			Konto  string `json:"konto"`
			Seite  string `json:"seite"`
			Betrag string `json:"betrag"`
		} `json:"zeilen"`
	}
	decision := map[string]any{"geschaeftsjahr": 2025, "beschluss_am": "2026-05-20", "ausschuettung": "1000.00",
		"text": "Ausschüttung von 1.000 €, Rest auf neue Rechnung"}
	preview := a.args(decision)
	preview["nur_vorschau"] = true
	var shown verwendung
	a.ok("ergebnisverwendung_buchen", preview, &shown)
	if !shown.Vorschau || shown.Festgestellt || shown.Buchungsjahr != 2026 || shown.Ergebnis != "4000.00" ||
		shown.Kapitalertragsteuer != "250.00" || shown.Soli != "13.75" || shown.Vortrag != "3000.00" || len(shown.Zeilen) != 3 ||
		!slices.ContainsFunc(shown.Hinweise, func(s string) bool { return strings.Contains(s, "nicht festgestellt") }) {
		t.Fatalf("Vorschau: %+v", shown)
	}
	if e := a.fail("ergebnisverwendung_buchen", a.args(decision), CodeRule); !strings.Contains(e.Nachricht, "nicht festgestellt") {
		t.Fatalf("ohne Feststellung: %+v", e)
	}

	t1, release, err := h.host.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := t1.CommitPeriod(uiCtx, core.CommitRequest{PeriodType: domain.PeriodTypeYear, PeriodLabel: "Geschäftsjahr 2025",
		CutoffDate: "2025-12-31", OverrideReason: "Jahr ohne Buchungen"}); err != nil {
		release()
		t.Fatalf("Festschreibung 2025: %v", err)
	}
	release()
	h.ui(func(ctx context.Context, s *core.Services) error {
		if _, err := s.Closing.SetFiscalYearStatus(ctx, 2025, domain.FiscalYearPrepared, "2026-03-31", ""); err != nil {
			return err
		}
		_, err := s.Closing.SetFiscalYearStatus(ctx, 2025, domain.FiscalYearAdopted, "2026-05-20", "Gesellschafterversammlung")
		return err
	})

	early := a.args(decision)
	early["beschluss_am"] = "2026-05-19"
	if e := a.fail("ergebnisverwendung_buchen", early, CodeRule); !strings.Contains(e.Nachricht, "vor der Feststellung") {
		t.Fatalf("Beschluss vor der Feststellung: %+v", e)
	}
	in := a.args(decision)
	in["idempotenz_schluessel"] = "ergebnisverwendung-2025"
	var booked, replay verwendung
	a.ok("ergebnisverwendung_buchen", in, &booked)
	a.ok("ergebnisverwendung_buchen", in, &replay)
	if booked.Vorschau || !booked.Festgestellt || booked.BuchungID == nil || !replay.Wiederholt ||
		replay.BuchungID == nil || *replay.BuchungID != *booked.BuchungID {
		t.Fatalf("gebucht %+v, wiederholt %+v", booked, replay)
	}
	var entry struct {
		Kanal  string `json:"kanal"`
		Zeilen []struct {
			Konto  string `json:"konto"`
			Betrag string `json:"betrag"`
		} `json:"zeilen"`
	}
	a.ok("buchung_lesen", map[string]any{"buchung_id": *booked.BuchungID}, &entry)
	credit := map[string]string{}
	for _, z := range entry.Zeilen {
		credit[z.Konto] = z.Betrag
	}
	if !strings.HasPrefix(entry.Kanal, "agent:") || credit[domain.AccountAusschuettung] != "736.25" ||
		credit[domain.AccountKapitalertragsteuer] != "263.75" {
		t.Errorf("Buchung der Ergebnisverwendung: %+v", entry)
	}

	var status struct {
		Schritte []struct {
			Schluessel     string `json:"schluessel"`
			NaechsteAktion struct {
				Werkzeug string `json:"werkzeug"`
			} `json:"naechste_aktion"`
		} `json:"schritte"`
	}
	a.ok("abschluss_status", map[string]any{"geschaeftsjahr": 2025}, &status)
	for _, step := range status.Schritte {
		if step.Schluessel == string(domain.ClosingStepAppropriation) && step.NaechsteAktion.Werkzeug != "ergebnisverwendung_buchen" {
			t.Errorf("Schritt Ergebnisverwendung führt zu %q", step.NaechsteAktion.Werkzeug)
		}
	}
}

// Die Organisationstexte der Verfahrensdokumentation liest und schreibt der
// Agent; ein leerer Text setzt das Muster zurück, Unverändertes schreibt
// nichts.
func TestOrganisationTextTool(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")
	type texte struct {
		Wiederholt bool `json:"wiederholt"`
		Texte      struct {
			Scannen   string `json:"scannen"`
			Sonstiges string `json:"sonstiges"`
		} `json:"texte"`
		Muster    []string `json:"muster"`
		Geaendert []string `json:"geaendert"`
		Hinweis   string   `json:"hinweis"`
	}
	var read texte
	a.ok("verfahrensdoku_text", a.args(nil), &read)
	if !slices.Contains(read.Muster, "scannen") || slices.Contains(read.Muster, "sonstiges") || len(read.Geaendert) != 0 {
		t.Fatalf("Stand: %+v", read)
	}

	scanning := "Papierbelege scannt die Geschäftsführerin am Tag des Eingangs mit 300 dpi."
	in := a.args(map[string]any{"idempotenz_schluessel": "org-scannen", "texte": map[string]any{"scannen": scanning}})
	var written, replay texte
	a.ok("verfahrensdoku_text", in, &written)
	a.ok("verfahrensdoku_text", in, &replay)
	if written.Texte.Scannen != scanning || !slices.Equal(written.Geaendert, []string{"scannen"}) ||
		slices.Contains(written.Muster, "scannen") || written.Hinweis == "" || !replay.Wiederholt {
		t.Fatalf("geschrieben %+v, wiederholt %+v", written, replay)
	}
	var same texte
	a.ok("verfahrensdoku_text", a.args(map[string]any{"texte": map[string]any{"scannen": scanning}}), &same)
	if len(same.Geaendert) != 0 {
		t.Errorf("unveränderter Text als Änderung gemeldet: %+v", same)
	}
	var reset texte
	a.ok("verfahrensdoku_text", a.args(map[string]any{"texte": map[string]any{"scannen": ""}}), &reset)
	if !slices.Equal(reset.Geaendert, []string{"scannen"}) || !slices.Contains(reset.Muster, "scannen") ||
		reset.Texte.Scannen != domain.DefaultOrganisationTexts().Scanning {
		t.Errorf("Muster zurückgesetzt: %+v", reset)
	}

	// Lesen braucht nur das Recht zu lesen und geht im Prüfermodus; schreiben
	// geht dann nicht.
	reader := h.mustSession(h.login("Nur lesen", "buchfink:lesen").access)
	if res := callTool(t, reader, "verfahrensdoku_text", a.args(nil)); res.IsError {
		t.Errorf("Lesen mit buchfink:lesen: %s", resultText(res))
	}
	until := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	a.ok("verfahrensdoku_text", a.args(nil), &read)
	a.fail("verfahrensdoku_text", a.args(map[string]any{"texte": map[string]any{"scannen": "neu"}}), CodeExaminer)
}
