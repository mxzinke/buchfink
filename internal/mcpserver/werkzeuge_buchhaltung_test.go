package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type kontaktOut struct {
	Kontakt struct {
		ID             uint   `json:"id"`
		Typ            string `json:"typ"`
		Name           string `json:"name"`
		Personenkonto  string `json:"personenkonto"`
		IBAN           string `json:"iban"`
		Standardgruppe struct {
			Gruppe    string `json:"gruppe"`
			Vorschlag string `json:"vorschlag"`
		} `json:"standardgruppe"`
	} `json:"kontakt"`
	BereitsVorhanden bool   `json:"bereits_vorhanden"`
	Uebereinstimmung string `json:"uebereinstimmung"`
	Wiederholt       bool   `json:"wiederholt"`
}

type erfasstOut struct {
	Buchung struct {
		ID     uint   `json:"id"`
		Nummer string `json:"nummer"`
	} `json:"buchung"`
	Wiederholt bool `json:"wiederholt"`
}

// Kontakte legt der Agent direkt an; ein vorhandener Kontakt wird erkannt
// statt verdoppelt. Die IBAN nennt die Suche nur auf Nachfrage.
func TestContactsAreNotDuplicated(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")

	in := a.args(map[string]any{"typ": "kunde", "name": "Kunde AG", "strasse": "Allee 5", "plz": "10115",
		"ort": "Berlin", "ust_id": "DE111222333", "iban": "DE02120300000000202051", "idempotenz_schluessel": "kunde-ag"})
	var first, replay kontaktOut
	a.ok("kontakt_anlegen", in, &first)
	a.ok("kontakt_anlegen", in, &replay)
	if first.Kontakt.ID == 0 || first.BereitsVorhanden || first.Kontakt.Personenkonto == "" ||
		!replay.Wiederholt || replay.Kontakt.ID != first.Kontakt.ID {
		t.Fatalf("anlegen: %+v / %+v", first, replay)
	}
	var again kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kunde Aktiengesellschaft", "ust_id": "DE 111 222 333"}), &again)
	if !again.BereitsVorhanden || again.Uebereinstimmung != "ust_id" || again.Kontakt.ID != first.Kontakt.ID {
		t.Errorf("dieselbe USt-IdNr.: %+v", again)
	}
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kunde AG", "plz": "10115"}), &again)
	if !again.BereitsVorhanden || again.Uebereinstimmung != "name_plz" {
		t.Errorf("Name und PLZ: %+v", again)
	}
	var vendor kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Kunde AG", "plz": "10115",
		"standardgruppe_vorschlag": "software"}), &vendor)
	if vendor.BereitsVorhanden || vendor.Kontakt.Standardgruppe.Vorschlag != "software" || vendor.Kontakt.Standardgruppe.Gruppe != "" {
		t.Errorf("Lieferant gleichen Namens mit Vorschlag: %+v", vendor)
	}
	a.rejected("kontakt_anlegen", a.args(map[string]any{"typ": "partner", "name": "X"}))
	a.fail("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kurz GmbH", "idempotenz_schluessel": "kurz"}), CodeInvalidInput)

	var found struct {
		Kontakte []struct {
			ID   uint   `json:"id"`
			IBAN string `json:"iban"`
		} `json:"kontakte"`
		Gesamt int `json:"gesamt"`
	}
	a.ok("kontakte_suchen", map[string]any{"text": "Kunde", "typ": "kunde"}, &found)
	if found.Gesamt != 1 || found.Kontakte[0].IBAN != "" {
		t.Errorf("Suche ohne IBAN: %+v", found)
	}
	a.ok("kontakte_suchen", map[string]any{"iban": "DE02 1203 0000 0000 2020 51"}, &found)
	if len(found.Kontakte) != 1 || found.Kontakte[0].IBAN != "DE02120300000000202051" {
		t.Errorf("Suche nach IBAN: %+v", found)
	}
}

// Eine Buchung ohne E-Rechnung geht nur mit Rückfrage; storniert wird mit
// Grund und Neubuchung, das Original bleibt lesbar.
func TestManualEntryReversalAndReminder(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")

	var customer kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kunde AG", "plz": "10115", "ort": "Berlin"}), &customer)
	var receipt ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("ar-17.pdf", pdf("Rechnung 17"))}},
		"belegdatum": "2026-03-10", "aussteller": "Pfennig Ventures GmbH", "brutto": "119.00"}), &receipt)

	revenue := func(net, tax, gross string) []any {
		return []any{
			map[string]any{"konto": customer.Kontakt.Personenkonto, "seite": "S", "betrag": gross, "kontakt_id": customer.Kontakt.ID},
			map[string]any{"konto": "4400", "seite": "H", "betrag": net},
			map[string]any{"konto": "3806", "seite": "H", "betrag": tax, "steuerschluessel": "UST19", "steuerbasis": net},
		}
	}
	in := a.args(map[string]any{"buchungsdatum": "2026-03-10", "beschreibung": "Beratung März, Rechnung 17",
		"steuerfall": "domestic", "beleg_id": receipt.Beleg.ID, "zeilen": revenue("100.00", "19.00", "119.00"),
		"idempotenz_schluessel": "rechnung-17"})
	var booked, replay erfasstOut
	a.ok("buchung_erfassen", in, &booked)
	a.ok("buchung_erfassen", in, &replay)
	if booked.Buchung.ID == 0 || !replay.Wiederholt || replay.Buchung.ID != booked.Buchung.ID {
		t.Fatalf("erfassen: %+v / %+v", booked, replay)
	}
	unbalanced := a.args(map[string]any{"idempotenz_schluessel": "geht-nicht-auf", "buchungsdatum": "2026-03-10", "beschreibung": "geht nicht auf", "steuerfall": "not_taxable",
		"eigenbeleg": map[string]any{"grund": "Test", "brutto": "1.00"},
		"zeilen":     []any{map[string]any{"konto": "1800", "seite": "S", "betrag": "1.00"}, map[string]any{"konto": "1360", "seite": "H", "betrag": "0.99"}}})
	a.fail("buchung_erfassen", unbalanced, CodeRule)
	// Ohne natürlichen Schlüssel ist der Idempotenzschlüssel Pflicht.
	unkeyed := a.args(map[string]any{"buchungsdatum": "2026-03-10", "beschreibung": "ohne Schlüssel", "steuerfall": "not_taxable",
		"eigenbeleg": map[string]any{"grund": "Test", "brutto": "1.00"}, "idempotenz_schluessel": "",
		"zeilen": []any{map[string]any{"konto": "1800", "seite": "S", "betrag": "1.00"}, map[string]any{"konto": "1360", "seite": "H", "betrag": "1.00"}}})
	if e := a.fail("buchung_erfassen", unkeyed, CodeInvalidInput); !strings.Contains(e.Nachricht, "idempotenz_schluessel") {
		t.Errorf("ohne Schlüssel: %+v", e)
	}

	var read struct {
		ID         uint   `json:"id"`
		Kanal      string `json:"kanal"`
		Steuerfall string `json:"steuerfall"`
		Neubuchung *uint  `json:"neubuchung"`
		Zeilen     []struct {
			Konto string `json:"konto"`
		} `json:"zeilen"`
	}
	a.ok("buchung_lesen", map[string]any{"buchung_id": booked.Buchung.ID}, &read)
	if read.Kanal != "agent:claude-code" || read.Steuerfall != "domestic" || len(read.Zeilen) != 3 {
		t.Fatalf("buchung_lesen: %+v", read)
	}
	a.fail("buchung_lesen", map[string]any{"buchung_id": 99999}, CodeNotFound)

	// Mahnung: der Vermerk setzt den Verzug, zahlen muss der Kunde weiter.
	var dunned struct {
		ID     uint `json:"id"`
		Posten *struct {
			LetzteMahnung string `json:"letzte_mahnung"`
			Mahnstufe     int    `json:"mahnstufe"`
			Offen         string `json:"offen"`
		} `json:"posten"`
	}
	a.fail("mahnung_vermerken", a.args(map[string]any{"buchung_id": booked.Buchung.ID, "idempotenz_schluessel": "mahnung-zukunft",
		"datum": time.Now().AddDate(0, 0, 3).Format("2006-01-02")}), CodeRule)
	reminder := a.args(map[string]any{"buchung_id": booked.Buchung.ID, "datum": "2026-04-20", "notiz": "per E-Mail",
		"idempotenz_schluessel": "mahnung-17-1"})
	a.ok("mahnung_vermerken", reminder, &dunned)
	if dunned.ID == 0 || dunned.Posten == nil || dunned.Posten.LetzteMahnung != "2026-04-20" || dunned.Posten.Mahnstufe != 1 ||
		dunned.Posten.Offen != "119.00" {
		t.Fatalf("Mahnung: %+v", dunned)
	}
	var open struct {
		Posten []struct {
			BuchungID     uint   `json:"buchung_id"`
			Mahnstufe     int    `json:"mahnstufe"`
			LetzteMahnung string `json:"letzte_mahnung"`
		} `json:"posten"`
	}
	a.ok("offene_posten", map[string]any{"richtung": "forderung"}, &open)
	if len(open.Posten) != 1 || open.Posten[0].Mahnstufe != 1 {
		t.Errorf("offene Forderung mit Mahnung: %+v", open)
	}

	// Storno mit Neubuchung: der Betrag war falsch.
	a.fail("buchung_stornieren", a.args(map[string]any{"buchung_id": booked.Buchung.ID, "grund": ""}), CodeInvalidInput)
	var reversed struct {
		Storno     struct{ ID uint }  `json:"storno"`
		Neubuchung *struct{ ID uint } `json:"neubuchung"`
	}
	a.fail("buchung_stornieren", a.args(map[string]any{"buchung_id": booked.Buchung.ID, "grund": "Betrag falsch erfasst",
		"datum": "2026-03-31", "neubuchung": map[string]any{"buchungsdatum": "2026-03-31", "beschreibung": "x",
			"steuerfall": "domestic", "zeilen": revenue("200.00", "38.00", "238.00")}}), CodeInvalidInput)
	a.ok("buchung_stornieren", a.args(map[string]any{"buchung_id": booked.Buchung.ID, "grund": "Betrag falsch erfasst",
		"neubuchung": map[string]any{"buchungsdatum": "2026-03-31", "belegdatum": "2026-03-10",
			"beschreibung": "Beratung März, Rechnung 17 (berichtigt)", "steuerfall": "domestic",
			"zeilen": revenue("200.00", "38.00", "238.00")}}), &reversed)
	if reversed.Storno.ID == 0 || reversed.Neubuchung == nil {
		t.Fatalf("Storno: %+v", reversed)
	}
	a.ok("buchung_lesen", map[string]any{"buchung_id": booked.Buchung.ID}, &read)
	if read.Neubuchung == nil || *read.Neubuchung != reversed.Neubuchung.ID {
		t.Errorf("Original nach dem Storno: %+v", read)
	}
	a.fail("buchung_stornieren", a.args(map[string]any{"buchung_id": booked.Buchung.ID, "grund": "nochmal"}), CodeRule)
	storniert := true
	var list struct {
		Buchungen []struct {
			ID             uint  `json:"id"`
			StorniertDurch *uint `json:"storniert_durch"`
		} `json:"buchungen"`
	}
	a.ok("buchungen_suchen", map[string]any{"storniert": storniert}, &list)
	if len(list.Buchungen) != 1 || list.Buchungen[0].StorniertDurch == nil || *list.Buchungen[0].StorniertDurch != reversed.Storno.ID {
		t.Errorf("stornierte Buchungen: %+v", list)
	}
}

// Das Lohnjournal des Lohnbüros wird mit Rückfrage gebucht; die Summen
// müssen aufgehen.
func TestPayrollJournal(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var journal ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("lohn-2026-04.pdf", pdf("Lohnjournal April"))}},
		"belegdatum": "2026-04-30", "aussteller": "Lohnbüro Muster", "brutto": "3600.00"}), &journal)
	in := a.args(map[string]any{"monat": "2026-04", "beleg_id": journal.Beleg.ID, "gehaelter": "3000.00",
		"an_sozialversicherung": "600.00", "lohnsteuer": "400.00", "netto": "2000.00", "ag_sozialversicherung": "600.00"})
	wrong := a.args(map[string]any{"monat": "2026-04", "beleg_id": journal.Beleg.ID, "gehaelter": "3000.00",
		"an_sozialversicherung": "600.00", "lohnsteuer": "400.00", "netto": "1999.00", "ag_sozialversicherung": "600.00"})
	a.fail("lohnjournal_buchen", wrong, CodeRule)
	var out struct {
		Buchung struct {
			ID     uint `json:"id"`
			Datum  string
			Zeilen []struct {
				Konto  string `json:"konto"`
				Seite  string `json:"seite"`
				Betrag string `json:"betrag"`
			} `json:"zeilen"`
		} `json:"buchung"`
	}
	a.ok("lohnjournal_buchen", in, &out)
	if out.Buchung.ID == 0 || out.Buchung.Datum != "2026-04-30" {
		t.Fatalf("Lohnjournal: %+v", out)
	}
	debit := map[string]string{}
	for _, z := range out.Buchung.Zeilen {
		if z.Seite == "S" {
			debit[z.Konto] = z.Betrag
		}
	}
	if debit["6020"] != "3000.00" {
		t.Errorf("Gehälter im Soll: %+v", out.Buchung.Zeilen)
	}
}

// Auswertungen, Konten, Anlagen und der Abschluss lesen; Abschlussbuchungen
// gehen mit Rückfrage und lassen sich vorher berechnen.
func TestReportsAndClosing(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var statement ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("einzahlung.pdf", pdf("Kontoauszug"))}},
		"belegdatum": "2026-02-03", "aussteller": "Hausbank", "brutto": "25000.00"}), &statement)
	// Das Stammkapital bucht der Agent selbst, mit dem Kontoauszug als Beleg.
	var capital buchungAntwort
	a.ok("buchung_erfassen", a.args(map[string]any{"idempotenz_schluessel": "kapital-2026", "buchungsdatum": "2026-02-03",
		"beschreibung": "Stammkapital eingezahlt", "steuerfall": "not_taxable", "beleg_id": statement.Beleg.ID,
		"zeilen": []any{map[string]any{"konto": "1800", "seite": "S", "betrag": "25000.00"},
			map[string]any{"konto": "2900", "seite": "H", "betrag": "25000.00"}}}), &capital)
	if capital.Buchung == nil || capital.Buchung.ID == 0 {
		t.Fatalf("Stammkapital: %+v", capital)
	}

	var report struct {
		Susa []struct {
			Konto string `json:"konto"`
			Saldo string `json:"saldo"`
		} `json:"susa"`
		Kontoblatt *struct {
			Endsaldo string `json:"endsaldo"`
		} `json:"kontoblatt"`
		Aktiva      []struct{} `json:"aktiva"`
		GuV         []struct{} `json:"guv"`
		Bilanzsumme *string    `json:"bilanzsumme"`
	}
	a.ok("auswertung", map[string]any{"art": "susa"}, &report)
	if len(report.Susa) != 2 {
		t.Errorf("SuSa: %+v", report.Susa)
	}
	a.ok("auswertung", map[string]any{"art": "kontoblatt", "konto": "1800"}, &report)
	if report.Kontoblatt == nil || report.Kontoblatt.Endsaldo != "25000.00" {
		t.Errorf("Kontoblatt: %+v", report.Kontoblatt)
	}
	a.fail("auswertung", map[string]any{"art": "kontoblatt"}, CodeInvalidInput)
	a.ok("auswertung", map[string]any{"art": "bilanz", "tiefe": "kurz"}, &report)
	a.rejected("auswertung", map[string]any{"art": "bilanz", "tiefe": "short"})
	if e := a.fail("auswertung", map[string]any{"art": "kontoblatt", "konto": customerLedger(t, a)}, CodeInvalidInput); e.Werkzeug != "buchungen_suchen" {
		t.Errorf("Kontoblatt eines Personenkontos: %+v", e)
	}
	if len(report.Aktiva) == 0 || report.Bilanzsumme == nil || *report.Bilanzsumme != "25000.00" {
		t.Errorf("Bilanz: %+v", report)
	}
	for _, art := range []string{"guv", "ust", "pruefbericht"} {
		a.ok("auswertung", map[string]any{"art": art}, nil)
	}

	var accounts struct {
		Konten []struct {
			Konto string `json:"konto"`
		} `json:"konten"`
		Buchungsgruppen []struct {
			Schluessel string `json:"schluessel"`
		} `json:"buchungsgruppen"`
		Steuerfaelle []struct {
			Schluessel string `json:"schluessel"`
		} `json:"steuerfaelle"`
	}
	a.ok("konten", map[string]any{"art": "suche", "suche": "Bank"}, &accounts)
	if len(accounts.Konten) == 0 {
		t.Error("keine Konten zu Bank")
	}
	a.ok("konten", map[string]any{"art": "suche", "nur_bebucht": true}, &accounts)
	if len(accounts.Konten) != 2 {
		t.Errorf("bebuchte Konten: %+v", accounts.Konten)
	}
	a.ok("konten", map[string]any{"art": "buchungsgruppen"}, &accounts)
	if !strings.Contains(strings.Join(keys(accounts.Buchungsgruppen), ","), "software") {
		t.Errorf("Buchungsgruppen: %+v", accounts.Buchungsgruppen)
	}
	a.ok("konten", map[string]any{"art": "steuerfaelle", "richtung": "ausgang"}, &accounts)
	if len(accounts.Steuerfaelle) == 0 {
		t.Error("keine Steuerfälle")
	}

	a.ok("anlagen", nil, nil)
	var closing struct {
		Stand    string `json:"stand"`
		Schritte []struct {
			Schluessel     string `json:"schluessel"`
			NaechsteAktion struct {
				Werkzeug string `json:"werkzeug"`
			} `json:"naechste_aktion"`
		} `json:"schritte"`
	}
	a.ok("abschluss_status", nil, &closing)
	if closing.Stand != "offen" || len(closing.Schritte) == 0 {
		t.Fatalf("Abschluss: %+v", closing)
	}
	for _, s := range closing.Schritte {
		if s.NaechsteAktion.Werkzeug == "" {
			t.Errorf("Schritt %s ohne nächste Aktion", s.Schluessel)
		}
	}
	var preview struct {
		Vorschau   bool   `json:"vorschau"`
		BuchungIDs []uint `json:"buchung_ids"`
	}
	a.ok("abschlussbuchung", a.args(map[string]any{"art": "ust_verrechnung", "nur_vorschau": true, "idempotenz_schluessel": "ust-2026"}), &preview)
	if !preview.Vorschau || len(preview.BuchungIDs) != 0 {
		t.Errorf("Vorschau bucht nicht: %+v", preview)
	}
	a.rejected("abschlussbuchung", a.args(map[string]any{"art": "festschreibung", "idempotenz_schluessel": "abschluss-1"}))
	a.fail("abschlussbuchung", a.args(map[string]any{"art": "rueckstellung", "idempotenz_schluessel": "abschluss-2"}), CodeInvalidInput)
}

func keys(rows []struct {
	Schluessel string `json:"schluessel"`
}) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Schluessel
	}
	return out
}

// Erklärungen liest der Agent formularnah; Angaben setzt er mit Rückfrage,
// was der Mensch gesetzt hat, bleibt. Exporte schreiben in den Exportordner,
// auch im Prüfermodus.
func TestDeclarationsInputsAndExports(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	// Ohne Buchung hat die Bilanz keine Summe; die E-Bilanz blockiert.
	if e := a.fail("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026}), CodeRule); e.NaechsterSchritt == "" {
		t.Errorf("E-Bilanz ohne Bilanzsumme: %+v", e)
	}
	h.capital()

	var ustva struct {
		Zeitraum   string `json:"zeitraum"`
		Kennzahlen []struct {
			Kz string `json:"kz"`
		} `json:"kennzahlen"`
		NaechsterSchritt string `json:"naechster_schritt"`
	}
	a.ok("erklaerung", map[string]any{"art": "ustva", "zeitraum": "2026-Q2"}, &ustva)
	if ustva.Zeitraum != "2026-Q2" || ustva.NaechsterSchritt == "" {
		t.Errorf("UStVA: %+v", ustva)
	}
	a.rejected("erklaerung", map[string]any{"art": "kst", "zeitraum": "Q1"})
	a.fail("erklaerung", map[string]any{"art": "kst", "zeitraum": "2026-Q1"}, CodeInvalidInput)
	var kst struct {
		Formulare []struct {
			Abschnitte []struct {
				Zeilen []struct {
					Schluessel      string `json:"schluessel"`
					Zustaendig      string `json:"zustaendig"`
					EingabeMoeglich bool   `json:"eingabe_moeglich"`
				} `json:"zeilen"`
			} `json:"abschnitte"`
		} `json:"formulare"`
	}
	a.ok("erklaerung", map[string]any{"art": "kst", "zeitraum": "2026", "nur_offen": true}, &kst)
	if len(kst.Formulare) == 0 {
		t.Fatalf("KSt ohne Formulare: %+v", kst)
	}

	const agentLine, humanLine = "kst.gk.z88", "kst.gk.vga"
	var set struct {
		Bestaetigt bool `json:"bestaetigt"`
	}
	input := a.args(map[string]any{"art": "kst", "jahr": 2026, "schluessel": agentLine, "wert": map[string]any{"betrag": "120.00"},
		"quelle": "beleg", "idempotenz_schluessel": "kst-zeile-88"})
	a.ok("erklaerung_eingabe", input, &set)
	if !set.Bestaetigt {
		t.Errorf("Zeile des Agenten gilt sofort: %+v", set)
	}
	a.ok("erklaerung_eingabe", a.args(map[string]any{"art": "kst", "jahr": 2026, "schluessel": humanLine,
		"wert": map[string]any{"betrag": "0.00"}}), &set)
	if set.Bestaetigt {
		t.Errorf("Zeile des Menschen bleibt Vorschlag: %+v", set)
	}
	a.fail("erklaerung_eingabe", a.args(map[string]any{"art": "kst", "jahr": 2026, "schluessel": agentLine,
		"wert": map[string]any{"betrag": "1.00", "text": "beides"}}), CodeInvalidInput)

	// Was der Mensch bestätigt oder in der App gesetzt hat, ändert der Agent nicht.
	h.ui(func(ctx context.Context, s *core.Services) error {
		if _, err := s.Declarations.ConfirmInput(ctx, declaration.KindCorporateTax, 2026, humanLine); err != nil {
			return err
		}
		_, err := s.Declarations.SetInput(ctx, service.InputRequest{Kind: declaration.KindCorporateTax, Year: 2026, Key: agentLine,
			Value: declaration.Value{Cents: ptr(domain.Cents(15000))}})
		return err
	})
	for _, key := range []string{humanLine, agentLine} {
		e := a.fail("erklaerung_eingabe", a.args(map[string]any{"art": "kst", "jahr": 2026, "schluessel": key,
			"wert": map[string]any{"betrag": "1.00"}}), CodeAppOnly)
		if e.Werkzeug != "nur_app" {
			t.Errorf("%s: %+v", key, e)
		}
	}

	until := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	a.fail("erklaerung_eingabe", input, CodeExaminer)
	var exp struct {
		Pfad   string `json:"pfad"`
		SHA256 string `json:"sha256"`
		Format string `json:"format"`
	}
	a.ok("erklaerung_exportieren", a.args(map[string]any{"art": "kst", "jahr": 2026}), &exp)
	exportDir := filepath.Join(h.tenant.DataDir, "Export")
	if filepath.Dir(exp.Pfad) != exportDir || exp.Format != "json" || len(exp.SHA256) != 64 {
		t.Fatalf("Export: %+v", exp)
	}
	if data, err := os.ReadFile(exp.Pfad); err != nil || !strings.Contains(string(data), agentLine) {
		t.Errorf("Exportdatei: %v", err)
	}
	a.rejected("erklaerung_exportieren", a.args(map[string]any{"art": "ustva", "jahr": 2026}))

	var eb struct {
		Pfad   string `json:"pfad"`
		Format string `json:"format"`
		SHA256 string `json:"sha256"`
	}
	a.ok("ebilanz_exportieren", a.args(map[string]any{"jahr": 2026}), &eb)
	if filepath.Dir(eb.Pfad) != exportDir || eb.Format != "xbrl" || len(eb.SHA256) != 64 {
		t.Fatalf("E-Bilanz: %+v", eb)
	}
	if data, err := os.ReadFile(eb.Pfad); err != nil || !strings.Contains(string(data), "xbrl") {
		t.Errorf("XBRL-Datei: %v", err)
	}
}

// Fristen liest der Agent, vorbereitet meldet er; erledigt setzt nur der
// Mensch. Anlässe erfasst er mit Rückfrage.
func TestObligationsAndStatus(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")

	st := statusResult(t, callTool(t, a.cs, "status", nil))
	if !st.Bereit || len(st.Belegordner) == 0 || st.Exportordner == "" || st.Direktregeln == nil ||
		st.Direktregeln.GrenzeBrutto != Betrag("1000.00") || st.Direktregeln.Mindestbuchungen != 3 {
		t.Fatalf("status: %+v", st)
	}
	if st.Belegordner[0] != filepath.Join(h.tenant.DataDir, "Belegeingang") {
		t.Errorf("Belegeingang: %v", st.Belegordner)
	}

	type pflicht struct {
		Schluessel  string `json:"schluessel"`
		Status      string `json:"status"`
		Verweis     string `json:"verweis"`
		Vorbereitet *struct {
			Kanal string `json:"kanal"`
		} `json:"vorbereitet"`
	}
	var due struct {
		Pflichten []pflicht `json:"pflichten"`
	}
	a.ok("fristen", map[string]any{"art": []any{"ustva"}, "geschaeftsjahr": 2026, "bis": "2027-03-01"}, &due)
	if len(due.Pflichten) != 4 {
		t.Fatalf("UStVA vierteljährlich: %+v", due)
	}
	for _, p := range due.Pflichten {
		if p.Verweis != "references/pflichten.md#ustva" && !strings.HasSuffix(p.Verweis, "#ustva") {
			t.Errorf("Verweis %q", p.Verweis)
		}
	}
	// Eine fällige, noch nicht überfällige Pflicht; überfällig bleibt
	// überfällig, auch wenn sie vorbereitet ist.
	key := ""
	for _, p := range due.Pflichten {
		if p.Status == "offen" {
			key = p.Schluessel
			break
		}
	}
	if key == "" {
		t.Fatalf("keine offene UStVA: %+v", due)
	}
	var prepared struct {
		Pflicht pflicht `json:"pflicht"`
	}
	in := a.args(map[string]any{"schluessel": key, "notiz": "Kennzahlen geprüft, Datei im Exportordner", "idempotenz_schluessel": "vorbereitet-" + key})
	a.ok("pflicht_vorbereitet", in, &prepared)
	if prepared.Pflicht.Status != "vorbereitet" || prepared.Pflicht.Vorbereitet == nil || prepared.Pflicht.Vorbereitet.Kanal != "agent:claude-code" {
		t.Fatalf("vorbereitet: %+v", prepared)
	}
	a.ok("pflicht_vorbereitet", in, nil)
	a.ok("fristen", map[string]any{"schluessel": key}, &due)
	if len(due.Pflichten) != 1 || due.Pflichten[0].Status != "vorbereitet" {
		t.Errorf("nach vorbereitet: %+v", due)
	}
	a.fail("pflicht_vorbereitet", a.args(map[string]any{"schluessel": "ustva.1999-Q1", "notiz": "x"}), CodeRule)

	var tasks struct {
		Geschaeftsjahr int `json:"geschaeftsjahr"`
		Offen          []struct {
			Werkzeug string `json:"werkzeug"`
		} `json:"offen"`
	}
	a.ok("aufgaben", nil, &tasks)
	if tasks.Geschaeftsjahr == 0 {
		t.Errorf("aufgaben: %+v", tasks)
	}

	var event struct {
		ID   uint   `json:"id"`
		Art  string `json:"art"`
		Hinw string `json:"hinweis"`
	}
	a.ok("anlass_erfassen", a.args(map[string]any{"art": "gesellschafterwechsel", "datum": "2026-06-01", "idempotenz_schluessel": "anlass-1",
		"notiz": "Anteilsübertragung, Urkunde UR 123/2026"}), &event)
	if event.ID == 0 || event.Art != "gesellschafterwechsel" {
		t.Fatalf("Anlass: %+v", event)
	}
	a.rejected("anlass_erfassen", a.args(map[string]any{"art": "kapitalerhoehung", "datum": "2026-06-01", "idempotenz_schluessel": "anlass-2"}))
	a.ok("fristen", map[string]any{"art": []any{"gesellschafterliste"}, "geschaeftsjahr": 2026}, &due)
	if len(due.Pflichten) == 0 {
		t.Errorf("der Anlass löst die Gesellschafterliste aus: %+v", due)
	}
}

// Die Vorschau rechnet ohne zu schreiben; ausgestellt wird mit Rückfrage.
func TestInvoicePreviewAndIssue(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var customer kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kunde AG", "strasse": "Allee 5", "plz": "10115",
		"ort": "Berlin", "land": "DE"}), &customer)
	invoice := map[string]any{"kontakt_id": customer.Kontakt.ID, "rechnungsdatum": "2026-05-04", "leistung_von": "2026-04-01",
		"leistung_bis": "2026-04-30", "positionen": []any{map[string]any{"text": "Beratung April", "menge": "10", "einheit": "HUR",
			"einzelpreis": "120.00", "satz": "19"}}}
	var preview struct {
		Netto  string `json:"netto"`
		Steuer string `json:"steuer"`
		Brutto string `json:"brutto"`
	}
	a.ok("rechnung_vorschau", invoice, &preview)
	if preview.Netto != "1200.00" || preview.Brutto != "1428.00" {
		t.Fatalf("Vorschau: %+v", preview)
	}
	a.fail("rechnung_vorschau", map[string]any{"kontakt_id": customer.Kontakt.ID, "leistung_von": "2026-04-01",
		"positionen": []any{map[string]any{"text": "x", "menge": "1.2345", "einzelpreis": "1.00", "satz": "19"}}}, CodeInvalidInput)
	h.warmRenderer()
	var issued struct {
		RechnungID uint   `json:"rechnung_id"`
		Nummer     string `json:"nummer"`
		Brutto     string `json:"brutto"`
		BuchungID  *uint  `json:"buchung_id"`
		Wiederholt bool   `json:"wiederholt"`
		Dateien    []struct {
			Pfad string `json:"pfad"`
			Mime string `json:"mime"`
		} `json:"dateien"`
	}
	in := a.args(map[string]any{"rechnung": invoice, "idempotenz_schluessel": "rechnung-april"})
	a.ok("rechnung_ausstellen", in, &issued)
	if issued.RechnungID == 0 || issued.Nummer == "" || issued.Brutto != "1428.00" || issued.BuchungID == nil {
		t.Fatalf("ausgestellt: %+v", issued)
	}
	pdfs := 0
	for _, f := range issued.Dateien {
		if _, err := os.Stat(f.Pfad); err != nil {
			t.Errorf("Datei zum Versand: %v", err)
		}
		if f.Mime == "application/pdf" {
			pdfs++
		}
	}
	if pdfs != 1 {
		t.Errorf("Dateien zum Versand: %+v", issued.Dateien)
	}
	var replay struct {
		RechnungID uint `json:"rechnung_id"`
		Wiederholt bool `json:"wiederholt"`
	}
	a.ok("rechnung_ausstellen", in, &replay)
	if !replay.Wiederholt || replay.RechnungID != issued.RechnungID {
		t.Errorf("Wiederholung: %+v", replay)
	}
}

// Kapital, Rücklagen, Ausschüttung und Gesellschafterkonten bucht der Agent
// von Hand wie jedes andere Konto; ein Fehler lässt sich stornieren.
func TestManualEntryReachesEquityAndShareholderAccounts(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	h.warmRenderer()
	a := h.agent("")
	for _, lines := range [][2]string{{"2970", "3519"}, {"2970", "2930"}, {"2970", "2960"}, {"1800", "3640"}, {"1298", "2900"}} {
		var out buchungAntwort
		a.ok("buchung_erfassen", a.args(map[string]any{"idempotenz_schluessel": "beschluss-" + lines[1], "buchungsdatum": "2026-06-30",
			"beschreibung": "Gesellschafterbeschluss", "steuerfall": "not_taxable",
			"eigenbeleg": map[string]any{"grund": "Gesellschafterbeschluss", "brutto": "1000.00"},
			"zeilen": []any{map[string]any{"konto": lines[0], "seite": "S", "betrag": "1000.00"},
				map[string]any{"konto": lines[1], "seite": "H", "betrag": "1000.00"}}}), &out)
		if out.Buchung == nil || len(out.Buchung.Zeilen) != 2 || out.Buchung.Zeilen[1].Konto != lines[1] {
			t.Errorf("%v: %+v", lines, out.Buchung)
		}
	}
}

// customerLedger legt einen Kunden an und liefert sein Personenkonto.
func customerLedger(t *testing.T, a *agent) string {
	t.Helper()
	var c kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kontoblatt KG", "plz": "50667"}), &c)
	return c.Kontakt.Personenkonto
}
