package mcpserver

import "testing"

// Die Gebühr eines Gerichts unter den Beiträgen meldet die Vorschau als
// Warnung; unter den Gebühren geht sie auf 6430.
func TestGebuehrengruppe(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")
	var court kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Amtsgericht Hamburg", "plz": "20355"}), &court)
	var filed belegAblegenOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-10-02", "belegdatum": "2026-10-02",
		"aussteller": "Amtsgericht Hamburg", "brutto": "100.00",
		"dateien": []any{map[string]any{"pfad": h.inboxFile("Kostenrechnung.pdf", pdf("Kostenrechnung"))}}}), &filed)
	var v struct {
		Vorschau struct {
			Zeilen []struct {
				Konto string `json:"konto"`
			} `json:"zeilen"`
			Hinweise []struct {
				Code string `json:"code"`
			} `json:"hinweise"`
		} `json:"vorschau"`
	}
	args := func(gruppe string) map[string]any {
		return a.args(map[string]any{"beleg_id": filed.Beleg.ID, "kontakt_id": court.Kontakt.ID, "belegdatum": "2026-10-02",
			"leistung_von": "2026-10-02", "steuerfall": "not_taxable", "bezahlung": "offen", "nur_vorschau": true,
			"positionen": []any{map[string]any{"buchungsgruppe": gruppe, "netto": "100.00", "satz": "0"}}})
	}
	warned := func() bool {
		for _, h := range v.Vorschau.Hinweise {
			if h.Code == "gruppe_passt_nicht_zum_aussteller" {
				return true
			}
		}
		return false
	}
	a.ok("beleg_buchen_mit_rueckfrage", args("beitraege"), &v)
	if !warned() {
		t.Fatalf("Gericht unter Beiträgen: %+v", v.Vorschau)
	}
	a.ok("beleg_buchen_mit_rueckfrage", args("gebuehren"), &v)
	if warned() || len(v.Vorschau.Zeilen) == 0 || v.Vorschau.Zeilen[0].Konto != "6430" {
		t.Fatalf("Gericht unter Gebühren: %+v", v.Vorschau)
	}
}
