package mcpserver

import (
	"testing"
)

type abschlussAntwort struct {
	Art          string   `json:"art"`
	Vorschau     bool     `json:"vorschau"`
	BuchungIDs   []uint   `json:"buchung_ids"`
	Betrag       *string  `json:"betrag"`
	Erlaeuterung string   `json:"erlaeuterung"`
	Hinweise     []string `json:"hinweise"`
	AnlageID     *uint    `json:"anlage_id"`
	Wiederholt   bool     `json:"wiederholt"`
}

// Jede Art von abschlussbuchung rechnet erst eine Vorschau und bucht dann mit
// demselben Schlüssel; die Wiederholung bucht nichts doppelt.
func TestEveryClosingBookingBooks(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendorID := h.vendor()
	a := h.agent("")

	// Ein Laptop auf 0650, eine Ausgangsrechnung als Forderung.
	var laptop ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-03-02", "dateien": []any{map[string]any{"pfad": h.inboxFile("laptop.pdf", pdf("Laptop"))}},
		"belegdatum": "2026-03-02", "aussteller": supplier.CompanyName, "brutto": "2380.00"}), &laptop)
	var bought buchungAntwort
	a.ok("beleg_buchen_mit_rueckfrage", a.args(map[string]any{"beleg_id": laptop.Beleg.ID, "kontakt_id": vendorID,
		"belegdatum": "2026-03-02", "leistung_von": "2026-03-02", "steuerfall": "domestic", "bezahlung": "offen",
		"leistungsnachweis": "geprüft gegen Bestellung 4711", "leistungsnachweis_am": "2026-03-03",
		"positionen": []any{map[string]any{"konto": "0650", "netto": "2000.00", "satz": "19", "text": "Laptop"}}}), &bought)
	customer := customerLedger(t, a)
	var customerID struct {
		Kontakte []struct {
			ID uint `json:"id"`
		} `json:"kontakte"`
	}
	a.ok("kontakte_suchen", map[string]any{"text": customer}, &customerID)
	var sale ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-04-30", "dateien": []any{map[string]any{"pfad": h.inboxFile("ar-1.pdf", pdf("Rechnung 1"))}},
		"belegdatum": "2026-04-30", "aussteller": "Pfennig Ventures GmbH", "brutto": "11900.00"}), &sale)
	var revenue struct {
		Buchung *struct {
			ID uint `json:"id"`
		} `json:"buchung"`
	}
	a.ok("buchung_erfassen", a.args(map[string]any{"idempotenz_schluessel": "ausgang-1", "buchungsdatum": "2026-04-30",
		"beschreibung": "Beratung April, Rechnung 1", "steuerfall": "domestic", "beleg_id": sale.Beleg.ID,
		"zeilen": []any{
			map[string]any{"konto": customer, "seite": "S", "betrag": "11900.00", "kontakt_id": customerID.Kontakte[0].ID},
			map[string]any{"konto": "4400", "seite": "H", "betrag": "10000.00"},
			map[string]any{"konto": "3806", "seite": "H", "betrag": "1900.00", "steuerschluessel": "UST19", "steuerbasis": "10000.00"},
		}}), &revenue)

	book := func(art, key string, in map[string]any) abschlussAntwort {
		t.Helper()
		args := a.args(map[string]any{"art": art, "geschaeftsjahr": 2026, "idempotenz_schluessel": key})
		for k, v := range in {
			args[k] = v
		}
		args["nur_vorschau"] = true
		var preview abschlussAntwort
		a.ok("abschlussbuchung", args, &preview)
		if !preview.Vorschau || len(preview.BuchungIDs) != 0 {
			t.Fatalf("%s: die Vorschau bucht: %+v", art, preview)
		}
		delete(args, "nur_vorschau")
		var booked, replay abschlussAntwort
		a.ok("abschlussbuchung", args, &booked)
		a.ok("abschlussbuchung", args, &replay)
		if booked.Vorschau || !replay.Wiederholt || len(replay.BuchungIDs) != len(booked.BuchungIDs) {
			t.Fatalf("%s: gebucht %+v, wiederholt %+v", art, booked, replay)
		}
		return booked
	}

	var assets struct {
		Kandidaten []struct {
			BuchungID uint `json:"buchung_id"`
		} `json:"aktivierungskandidaten"`
	}
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026}, &assets)
	if len(assets.Kandidaten) != 1 || assets.Kandidaten[0].BuchungID != bought.Buchung.ID {
		t.Fatalf("Aktivierungskandidat: %+v", assets)
	}
	activated := book("anlage_aktivieren", "laptop-aktivieren", map[string]any{"anlage": map[string]any{
		"buchung_id": bought.Buchung.ID, "bezeichnung": "Laptop", "nutzungsdauer_monate": 36}})
	if activated.AnlageID == nil {
		t.Fatalf("anlage_aktivieren: %+v", activated)
	}
	var listed struct {
		Anlagen []struct {
			Klasse  string `json:"klasse"`
			Methode string `json:"methode"`
		} `json:"anlagen"`
	}
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "klasse": "sachanlage"}, &listed)
	if len(listed.Anlagen) != 1 || listed.Anlagen[0].Klasse != "sachanlage" || listed.Anlagen[0].Methode != "linear" {
		t.Fatalf("Anlagenverzeichnis: %+v", listed)
	}

	cases := []struct {
		art, key string
		in       map[string]any
	}{
		{"afa_lauf", "afa-2026", nil},
		{"abgrenzung", "rap-versicherung", map[string]any{"abgrenzung": map[string]any{"art": "aktiv",
			"text": "Versicherung 7/2026 bis 6/2027", "gesamtbetrag": "1200.00", "von": "2026-07-01", "bis": "2027-06-30", "konto": "6400"}}},
		{"rueckstellung", "rst-abschluss", map[string]any{"rueckstellung": map[string]any{"art": "abschlusskosten",
			"text": "Jahresabschluss 2026", "betrag": "1500.00", "erwartet_am": "2027-03-31", "grund": "Angebot Steuerbüro"}}},
		{"wertberichtigung", "ewb-ar-1", map[string]any{"wertberichtigung": map[string]any{"grund": "Kunde zahlt schleppend",
			"einzeln": []any{map[string]any{"buchung_id": revenue.Buchung.ID, "prozent": 50, "grund": "Mahnverfahren"}}}}},
		{"unfertige_leistungen", "wip-2026", map[string]any{"unfertige_leistungen": map[string]any{"grund": "Projektliste",
			"projekte": []any{map[string]any{"name": "Projekt Nord", "einzelkosten": "1000.00"}}}}},
		{"ust_verrechnung", "ust-2026", nil},
		{"steuerrueckstellung", "steuer-2026", map[string]any{"steuerrueckstellung": map[string]any{"hebesatz": 400}}},
	}
	for _, tc := range cases {
		booked := book(tc.art, tc.key, tc.in)
		if len(booked.BuchungIDs) == 0 {
			t.Errorf("%s bucht nichts: %+v", tc.art, booked)
		}
		for _, id := range booked.BuchungIDs {
			var entry struct {
				ID uint `json:"id"`
			}
			a.ok("buchung_lesen", map[string]any{"buchung_id": id}, &entry)
		}
	}

	// Ohne Posten in Fremdwährung gibt es nichts zu bewerten; die UG-Rücklage
	// gilt nur für die UG.
	none := book("fremdwaehrung", "fremdwaehrung-2026", nil)
	if len(none.BuchungIDs) != 0 || none.Erlaeuterung == "" {
		t.Errorf("fremdwaehrung ohne Posten: %+v", none)
	}
	a.fail("abschlussbuchung", a.args(map[string]any{"art": "ug_ruecklage", "idempotenz_schluessel": "ruecklage-2026"}), CodeRule)
	a.rejected("abschlussbuchung", a.args(map[string]any{"art": "abgrenzung", "idempotenz_schluessel": "abgrenzung-x",
		"abgrenzung": map[string]any{"art": "active", "text": "x", "gesamtbetrag": "1.00", "von": "2026-01-01", "bis": "2026-12-31", "konto": "6400"}}))
}
