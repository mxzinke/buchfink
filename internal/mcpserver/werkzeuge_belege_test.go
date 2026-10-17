package mcpserver

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
)

type ablageOut struct {
	Beleg struct {
		ID        uint   `json:"id"`
		Nummer    string `json:"nummer"`
		ERechnung bool   `json:"e_rechnung"`
	} `json:"beleg"`
	BereitsVorhanden bool `json:"bereits_vorhanden"`
	ERechnung        bool `json:"e_rechnung"`
	Befunde          struct {
		Geprueft bool `json:"geprueft"`
		Gesamt   int  `json:"gesamt"`
		Liste    []struct {
			Regel   string `json:"regel"`
			Schwere string `json:"schwere"`
			Text    string `json:"text"`
		} `json:"liste"`
	} `json:"befunde"`
	Wiederholt bool `json:"wiederholt"`
}

type buchungAntwort struct {
	Buchung *struct {
		ID     uint `json:"id"`
		Zeilen []struct {
			Konto  string `json:"konto"`
			Seite  string `json:"seite"`
			Betrag string `json:"betrag"`
		} `json:"zeilen"`
	} `json:"buchung"`
	Vorschau *struct {
		Brutto string `json:"brutto"`
	} `json:"vorschau"`
	Einstufung string `json:"einstufung"`
	Wiederholt bool   `json:"wiederholt"`
}

// defects zählt die Befunde ohne die Hinweise.
func (o ablageOut) defects() int {
	n := 0
	for _, f := range o.Befunde.Liste {
		if f.Schwere != "information" {
			n++
		}
	}
	return n
}

// file legt eine XRechnung mit einer PDF-Darstellung ab.
func (a *agent) file(path string) ablageOut {
	a.t.Helper()
	view := a.h.inboxFile(filepath.Base(path)+".pdf", pdf("Darstellung "+filepath.Base(path)))
	var out ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{
		map[string]any{"pfad": path}, map[string]any{"pfad": view, "rolle": "darstellung"},
	}, "eingang_ueber": "email"}), &out)
	return out
}

func bookArgs(receiptID, contactID uint, date, net string) map[string]any {
	return map[string]any{
		"beleg_id": receiptID, "kontakt_id": contactID, "belegdatum": date, "leistung_von": date,
		"steuerfall": "domestic", "bezahlung": "offen",
		"positionen": []any{map[string]any{"buchungsgruppe": "software", "netto": net, "satz": "19"}},
	}
}

// Der Weg einer E-Rechnung: ablegen, lesen, direkt nur als Routine; mit
// Rückfrage geht alles, was die Dienste zulassen. Die Standardgruppe schlägt
// der Agent vor, bestätigen kann nur die App.
func TestReceiptFromInboxToDirectBooking(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendorID := h.vendor()
	a := h.agent("")

	first := a.file(h.eInvoice("RE-LIEF-1001", "2026-03-02", 10000))
	if !first.ERechnung || !first.Befunde.Geprueft || first.defects() != 0 {
		t.Fatalf("E-Rechnung abgelegt: %+v", first)
	}
	if again := a.file(h.eInvoice("RE-LIEF-1001", "2026-03-02", 10000)); !again.BereitsVorhanden || again.Beleg.ID != first.Beleg.ID {
		t.Errorf("dieselbe Datei zweimal: %+v", again)
	}

	var lesen struct {
		Vorschlag *struct {
			KontaktID      uint   `json:"kontakt_id"`
			KontaktErkannt bool   `json:"kontakt_erkannt"`
			Brutto         string `json:"brutto"`
		} `json:"vorschlag"`
		AusBeleg *struct {
			Rechnungsnummer string `json:"rechnungsnummer"`
		} `json:"aus_beleg"`
		Einstufung *struct {
			Stufe   string `json:"stufe"`
			Gruende []struct {
				Schluessel string `json:"schluessel"`
			} `json:"gruende"`
		} `json:"einstufung"`
	}
	a.ok("beleg_lesen", map[string]any{"beleg_id": first.Beleg.ID}, &lesen)
	if lesen.Vorschlag == nil || !lesen.Vorschlag.KontaktErkannt || lesen.Vorschlag.KontaktID != vendorID ||
		lesen.Vorschlag.Brutto != "119.00" || lesen.AusBeleg == nil || lesen.AusBeleg.Rechnungsnummer != "RE-LIEF-1001" {
		t.Fatalf("beleg_lesen: %+v %+v", lesen.Vorschlag, lesen.AusBeleg)
	}
	if lesen.Einstufung == nil || lesen.Einstufung.Stufe != "rueckfrage" {
		t.Fatalf("Einstufung ohne bestätigte Gruppe: %+v", lesen.Einstufung)
	}

	// Ohne bestätigte Standardgruppe ist nichts Routine.
	e := a.fail("beleg_buchen", a.args(bookArgs(first.Beleg.ID, vendorID, "2026-03-02", "100.00")), CodeApprovalNeeded)
	if e.Werkzeug != "beleg_buchen_mit_rueckfrage" || !hasReason(e, "keine_bestaetigte_standardgruppe") {
		t.Fatalf("freigabe_noetig: %+v", e)
	}
	a.ok("standardgruppe_vorschlagen", a.args(map[string]any{"kontakt_id": vendorID, "buchungsgruppe": "software",
		"begruendung": "Server-Wartung"}), nil)
	e = a.fail("beleg_buchen", a.args(bookArgs(first.Beleg.ID, vendorID, "2026-03-02", "100.00")), CodeApprovalNeeded)
	if !hasReason(e, "keine_bestaetigte_standardgruppe") {
		t.Fatalf("der Vorschlag allein genügt nicht: %+v", e)
	}

	// Mit Rückfrage bucht der Agent; derselbe Schlüssel bucht kein zweites Mal.
	confirmArgs := a.args(bookArgs(first.Beleg.ID, vendorID, "2026-03-02", "100.00"))
	confirmArgs["idempotenz_schluessel"] = "beleg-1001-buchen"
	var booked, replay buchungAntwort
	a.ok("beleg_buchen_mit_rueckfrage", confirmArgs, &booked)
	a.ok("beleg_buchen_mit_rueckfrage", confirmArgs, &replay)
	if booked.Buchung == nil || replay.Buchung == nil || booked.Buchung.ID != replay.Buchung.ID || !replay.Wiederholt {
		t.Fatalf("Wiederholung: %+v / %+v", booked, replay)
	}

	// Bestätigen kann nur die App.
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Contacts.ConfirmDefaultPostingGroup(ctx, vendorID, "software")
		return err
	})
	for i, inv := range []struct {
		number, date, net string
		cents             domain.Cents
	}{{"RE-LIEF-1002", "2026-04-01", "200.00", 20000}, {"RE-LIEF-1003", "2026-04-02", "201.00", 20100}} {
		r := a.file(h.eInvoice(inv.number, inv.date, inv.cents))
		args := a.args(bookArgs(r.Beleg.ID, vendorID, inv.date, inv.net))
		e := a.fail("beleg_buchen", args, CodeApprovalNeeded)
		if !hasReason(e, "zu_wenige_buchungen") || hasReason(e, "keine_bestaetigte_standardgruppe") {
			t.Fatalf("nach Bestätigung, %d Buchungen: %+v", i+1, e)
		}
		a.ok("beleg_buchen_mit_rueckfrage", args, nil)
	}

	// Dritte Buchung vorhanden: jetzt ist die nächste E-Rechnung Routine.
	routine := a.file(h.eInvoice("RE-LIEF-1004", "2026-05-04", 50000))
	args := a.args(bookArgs(routine.Beleg.ID, vendorID, "2026-05-04", "500.00"))
	preview := args
	preview["nur_vorschau"] = true
	var p buchungAntwort
	a.ok("beleg_buchen", preview, &p)
	if p.Einstufung != "direkt" || p.Vorschau == nil || p.Vorschau.Brutto != "595.00" || p.Buchung != nil {
		t.Fatalf("Vorschau: %+v", p)
	}
	delete(args, "nur_vorschau")
	var direct buchungAntwort
	a.ok("beleg_buchen", args, &direct)
	if direct.Einstufung != "direkt" || direct.Buchung == nil || direct.Buchung.ID == 0 {
		t.Fatalf("Routinebuchung: %+v", direct)
	}

	// Ein Cent Abweichung, über der Grenze, ein Foto: wieder Rückfrage.
	cent := a.file(h.eInvoice("RE-LIEF-1005", "2026-05-05", 30000))
	e = a.fail("beleg_buchen", a.args(bookArgs(cent.Beleg.ID, vendorID, "2026-05-05", "300.01")), CodeApprovalNeeded)
	if !hasReason(e, "betraege_weichen_ab") {
		t.Errorf("Abweichung von der E-Rechnung: %+v", e)
	}
	big := a.file(h.eInvoice("RE-LIEF-1006", "2026-05-06", 90000))
	e = a.fail("beleg_buchen", a.args(bookArgs(big.Beleg.ID, vendorID, "2026-05-06", "900.00")), CodeApprovalNeeded)
	if !hasReason(e, "ueber_direktgrenze") {
		t.Errorf("über der Direktgrenze: %+v", e)
	}
	var photo ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("foto.pdf", pdf("Quittung"))}},
		"belegdatum": "2026-05-07", "aussteller": supplier.CompanyName, "brutto": "11.90"}), &photo)
	if photo.ERechnung {
		t.Fatalf("ein PDF ohne Datensatz ist keine E-Rechnung: %+v", photo)
	}
	e = a.fail("beleg_buchen", a.args(bookArgs(photo.Beleg.ID, vendorID, "2026-05-07", "10.00")), CodeApprovalNeeded)
	if !hasReason(e, "keine_e_rechnung") {
		t.Errorf("PDF ohne Datensatz: %+v", e)
	}

	// Verworfen wird nur mit Grund; der Beleg bleibt auffindbar.
	a.fail("beleg_verwerfen", a.args(map[string]any{"beleg_id": photo.Beleg.ID, "grund": " "}), CodeInvalidInput)
	a.ok("beleg_verwerfen", a.args(map[string]any{"beleg_id": photo.Beleg.ID, "grund": "Dublette"}), nil)
	var list struct {
		Belege []struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"belege"`
		Gesamt int `json:"gesamt"`
	}
	a.ok("belege_suchen", map[string]any{"status": "verworfen"}, &list)
	if len(list.Belege) != 1 || list.Belege[0].ID != photo.Beleg.ID {
		t.Errorf("verworfene Belege: %+v", list)
	}
	a.ok("belege_suchen", map[string]any{"status": "gebucht", "limit": 2}, &list)
	if list.Gesamt != 4 || len(list.Belege) != 2 {
		t.Errorf("gebuchte Belege seitenweise: %+v", list)
	}

	// Die gebuchten Belege stehen unter dem Erfassungsweg des Agenten.
	var found struct {
		Buchungen []struct {
			Kanal string `json:"kanal"`
		} `json:"buchungen"`
	}
	a.ok("buchungen_suchen", map[string]any{"kanal": "agent:"}, &found)
	if len(found.Buchungen) != 4 || found.Buchungen[0].Kanal != "agent:claude-code" {
		t.Errorf("Buchungen der App: %+v", found)
	}
}

func hasReason(e ToolError, key string) bool {
	return slices.ContainsFunc(e.Gruende, func(r policy.Reason) bool { return r.Key == key })
}
