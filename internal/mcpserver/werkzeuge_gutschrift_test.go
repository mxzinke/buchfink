package mcpserver

import "testing"

// Die Gutschrift eines Lieferanten bucht der Agent mit gutschrift_zu: Guthaben
// beim Kreditor, Aufwand und Vorsteuer im Haben.
func TestGutschriftDesLieferanten(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")
	var supplier kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Isarhost GmbH", "plz": "80331"}), &supplier)
	var filed belegAblegenOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-09-21", "belegdatum": "2026-09-21",
		"aussteller": "Isarhost GmbH", "brutto": "23.80", "dateien": []any{map[string]any{"pfad": h.inboxFile("gs.pdf", pdf("Gutschrift"))}}}), &filed)
	var out struct {
		Vorschau struct {
			Zeilen []struct {
				Konto string `json:"konto"`
				Seite string `json:"seite"`
			} `json:"zeilen"`
		} `json:"vorschau"`
	}
	a.ok("beleg_buchen_mit_rueckfrage", a.args(map[string]any{"beleg_id": filed.Beleg.ID, "kontakt_id": supplier.Kontakt.ID,
		"belegdatum": "2026-09-21", "leistung_von": "2026-09-21", "steuerfall": "domestic", "bezahlung": "offen",
		"gutschrift_zu": "IH-2026-08-1421", "nur_vorschau": true,
		"positionen": []any{map[string]any{"buchungsgruppe": "software", "netto": "20.00", "satz": "19"}}}), &out)
	sides := map[string]string{}
	for _, z := range out.Vorschau.Zeilen {
		sides[z.Konto] = z.Seite
	}
	if sides["6495"] != "H" || sides["1406"] != "H" {
		t.Errorf("Gutschrift: %+v", out.Vorschau.Zeilen)
	}
}
