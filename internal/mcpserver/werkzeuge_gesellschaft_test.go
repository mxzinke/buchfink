package mcpserver

import (
	"context"
	"slices"
	"strings"
	"testing"

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

// Eine Berichtigung ändert nur, was der Agent nennt; sind Auszahlungen
// gebucht, bleibt das Darlehenskonto. Zinsen bucht darlehen_erfassen mit
// art=zinsen samt Zinsperiode, danach ändert sich der Zinssatz nicht mehr.
// Eine Zeile auf einem Gesellschafterkonto braucht den Gesellschafter.
func TestShareholderLoanCorrectionAndInterest(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	// Die Zinsbuchung setzt einen Eigenbeleg; das Satzprogramm braucht beim
	// ersten Lauf länger, als ein Schreibvorgang unter Last warten darf.
	h.warmRenderer()
	anna := h.shareholder()
	a := h.agent("")
	var contract ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("vertrag.pdf", pdf("Darlehensvertrag kurz"))}},
		"belegdatum": "2025-12-15", "aussteller": "Anna Bauer", "brutto": "50000.00"}), &contract)
	var booked darlehenAntwort
	a.ok("darlehen_erfassen", a.args(map[string]any{"idempotenz_schluessel": "darlehen-kurz", "kontakt_id": anna,
		"richtung": "an_gesellschaft", "betrag": "50000.00", "zinssatz": "6", "zinsmethode": "30/360",
		"vertrag_vom": "2025-12-15", "beginn": "2026-01-02", "faellig_am": "2026-12-31", "beleg_id": contract.Beleg.ID}), &booked)
	if booked.Darlehen.Konto != "3641" {
		t.Fatalf("Konto bei einem Jahr Laufzeit: %+v", booked.Darlehen)
	}
	loanID := booked.Darlehen.ID

	payout := func(key string, line map[string]any) map[string]any {
		return a.args(map[string]any{"idempotenz_schluessel": key, "buchungsdatum": "2026-02-01",
			"beschreibung": "Auszahlung Gesellschafterdarlehen Anna Bauer", "steuerfall": "not_taxable",
			"beleg_id": contract.Beleg.ID,
			"zeilen":   []any{map[string]any{"konto": "1800", "seite": "S", "betrag": "50000.00"}, line}})
	}
	if e := a.fail("buchung_erfassen", payout("auszahlung-ohne", map[string]any{"konto": "3641", "seite": "H", "betrag": "50000.00"}),
		CodeRule); !strings.Contains(e.Nachricht, "Gesellschafter als Kontakt") {
		t.Fatalf("Gesellschafterkonto ohne Kontakt: %+v", e)
	}
	var entry buchungOut
	a.ok("buchung_erfassen", payout("auszahlung", map[string]any{"konto": "3641", "seite": "H", "betrag": "50000.00",
		"kontakt_id": anna}), &entry)

	type darlehenVoll struct {
		Stichtag string `json:"stichtag"`
		Darlehen struct {
			Konto            string `json:"konto"`
			Zinsmethode      string `json:"zinsmethode"`
			FaelligAm        string `json:"faellig_am"`
			BelegID          *uint  `json:"beleg_id"`
			Stand            string `json:"stand"`
			ErsteAuszahlung  string `json:"erste_auszahlung"`
			ZinsenGebuchtBis string `json:"zinsen_gebucht_bis"`
			Befunde          []struct {
				Text string `json:"text"`
			} `json:"befunde"`
		} `json:"darlehen"`
		Zinsen *struct {
			Von       string `json:"von"`
			Bis       string `json:"bis"`
			Betrag    string `json:"betrag"`
			BuchungID *uint  `json:"buchung_id"`
		} `json:"zinsen"`
	}
	var prolonged darlehenVoll
	a.ok("darlehen_erfassen", a.args(map[string]any{"idempotenz_schluessel": "darlehen-kurz-verlaengert",
		"darlehen_id": loanID, "grund": "Laufzeit verlängert", "faellig_am": "2028-12-31"}), &prolonged)
	d := prolonged.Darlehen
	if d.FaelligAm != "2028-12-31" || d.BelegID == nil || *d.BelegID != contract.Beleg.ID || d.Zinsmethode != "30/360" ||
		d.Konto != "3641" || d.Stand != "50000.00" || d.ErsteAuszahlung != "2026-02-01" || prolonged.Stichtag != today() {
		t.Fatalf("Berichtigung nur der Fälligkeit: %+v (Stichtag %s)", d, prolonged.Stichtag)
	}
	if e := a.fail("darlehen_erfassen", a.args(map[string]any{"idempotenz_schluessel": "darlehen-kurz-konto",
		"darlehen_id": loanID, "grund": "Ausweis nach Laufzeit", "konto": "3642"}), CodeRule); !strings.Contains(e.Nachricht, "Darlehenskonto") {
		t.Errorf("Kontowechsel trotz Auszahlung: %+v", e)
	}

	var interest darlehenVoll
	a.ok("darlehen_erfassen", a.args(map[string]any{"idempotenz_schluessel": "zinsen-h1", "art": "zinsen",
		"darlehen_id": loanID, "zinsen_bis": "2026-06-30", "nur_vorschau": true}), &interest)
	if interest.Zinsen == nil || interest.Zinsen.Von != "2026-01-02" || interest.Zinsen.BuchungID != nil || interest.Zinsen.Betrag == "0.00" {
		t.Fatalf("Zinsvorschau: %+v", interest.Zinsen)
	}
	a.ok("darlehen_erfassen", a.args(map[string]any{"idempotenz_schluessel": "zinsen-h1", "art": "zinsen",
		"darlehen_id": loanID, "zinsen_bis": "2026-06-30"}), &interest)
	if interest.Zinsen == nil || interest.Zinsen.BuchungID == nil || interest.Darlehen.ZinsenGebuchtBis != "2026-06-30" {
		t.Fatalf("Zinsen gebucht: %+v %+v", interest.Zinsen, interest.Darlehen)
	}
	if e := a.fail("darlehen_erfassen", a.args(map[string]any{"idempotenz_schluessel": "darlehen-kurz-zins",
		"darlehen_id": loanID, "grund": "Zins erhöht", "zinssatz": "9"}), CodeRule); !strings.Contains(e.Nachricht, "Zinsen bis") {
		t.Errorf("Zinssatz trotz gebuchter Zinsen geändert: %+v", e)
	}
	var listed struct {
		Darlehen []struct {
			ZinsenGebuchtBis string `json:"zinsen_gebucht_bis"`
			Befunde          []struct {
				Text string `json:"text"`
			} `json:"befunde"`
		} `json:"darlehen"`
	}
	a.ok("auswertung", map[string]any{"art": "darlehen", "bis": "2026-12-31"}, &listed)
	if len(listed.Darlehen) != 1 || listed.Darlehen[0].ZinsenGebuchtBis != "2026-06-30" ||
		!slices.ContainsFunc(listed.Darlehen[0].Befunde, func(b struct {
			Text string `json:"text"`
		}) bool {
			return strings.Contains(b.Text, "Zinsen vom 01.07.2026")
		}) {
		t.Errorf("Zinsen in der Auswertung: %+v", listed.Darlehen)
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
	early["nur_vorschau"] = true
	var earlyShown verwendung
	a.ok("ergebnisverwendung_buchen", early, &earlyShown)
	if !slices.ContainsFunc(earlyShown.Hinweise, func(s string) bool { return strings.Contains(s, "vor der Feststellung") }) {
		t.Errorf("die Vorschau verschweigt einen Beschluss vor der Feststellung: %+v", earlyShown.Hinweise)
	}
	delete(early, "nur_vorschau")
	if e := a.fail("ergebnisverwendung_buchen", early, CodeRule); !strings.Contains(e.Nachricht, "vor der Feststellung") {
		t.Fatalf("Beschluss vor der Feststellung: %+v", e)
	}
	var onTime verwendung
	a.ok("ergebnisverwendung_buchen", preview, &onTime)
	if slices.ContainsFunc(onTime.Hinweise, func(s string) bool { return strings.Contains(s, "So bucht Buchfink") }) {
		t.Errorf("Vorschau nach der Feststellung: %v", onTime.Hinweise)
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
	until := harnessNow().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	a.ok("verfahrensdoku_text", a.args(nil), &read)
	a.fail("verfahrensdoku_text", a.args(map[string]any{"texte": map[string]any{"scannen": "neu"}}), CodeExaminer)
}

// Die Unterbilanz rechnet der Gründungsdienst; auswertung zeigt sie dem
// Agenten. Notarkosten von 3.000 € mindern das Reinvermögen, die Satzung
// übernimmt 1.000 € als Gründungsaufwand, die Gesellschafterin haftet für
// 2.000 €. Vor der Eintragung ist die Zahl vorläufig.
func TestAuswertungShowsTheUnterbilanz(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")

	var none struct {
		Unterbilanz *struct{} `json:"unterbilanz"`
		Hinweis     string    `json:"hinweis"`
	}
	a.ok("auswertung", map[string]any{"art": "unterbilanz"}, &none)
	if none.Unterbilanz != nil || !strings.Contains(none.Hinweis, "keine Gründung") {
		t.Fatalf("ohne Gründung: %+v", none)
	}

	h.ui(func(ctx context.Context, s *core.Services) error {
		if _, err := s.Foundation.Save(ctx, &domain.Foundation{
			NotarizedOn: "2026-01-15", ShareCapital: 2_500_000, FoundationCostCap: 100_000,
			Shareholders: []domain.Shareholder{
				{Name: "Anna Bauer", ShareCapital: 2_500_000, PaidIn: 1_250_000, Kind: domain.ContributionCash},
			},
		}); err != nil {
			return err
		}
		if _, err := s.Foundation.BookPostings(ctx); err != nil {
			return err
		}
		_, err := s.Journal.Post(ctx, &domain.JournalEntry{BookingDate: "2026-01-20", DocumentDate: "2026-01-20",
			ServiceDateFrom: "2026-01-20", ServiceDateTo: "2026-01-20", Description: "Notarkosten Gründung",
			Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "6825", Amount: 300_000},
				{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 300_000}}})
		return err
	})

	var got struct {
		Stichtag    string `json:"stichtag"`
		Unterbilanz struct {
			Endgueltig     bool   `json:"endgueltig"`
			Stammkapital   string `json:"stammkapital"`
			Reinvermoegen  string `json:"reinvermoegen"`
			Unterdeckung   string `json:"unterdeckung"`
			Gedeckt        string `json:"gedeckt"`
			Haftung        string `json:"haftung"`
			Gesellschafter []struct {
				Name    string `json:"name"`
				Haftung string `json:"haftung"`
			} `json:"gesellschafter"`
		} `json:"unterbilanz"`
	}
	a.ok("auswertung", map[string]any{"art": "unterbilanz"}, &got)
	u := got.Unterbilanz
	if u.Endgueltig || u.Stammkapital != "25000.00" || u.Reinvermoegen != "22000.00" || u.Unterdeckung != "3000.00" ||
		u.Gedeckt != "1000.00" || u.Haftung != "2000.00" || got.Stichtag == "" {
		t.Fatalf("Unterbilanz: %+v", got)
	}
	if len(u.Gesellschafter) != 1 || u.Gesellschafter[0].Name != "Anna Bauer" || u.Gesellschafter[0].Haftung != "2000.00" {
		t.Errorf("Anteile: %+v", u.Gesellschafter)
	}
}
