package mcpserver

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
)

// Die Fälligkeit des offenen Postens kommt aus dem Beleg, sonst aus dem
// Zahlungsziel des Lieferanten; dann sagt die Vorschau das. Die Gebühr eines
// Gerichts unter den Beiträgen meldet die Vorschau als Warnung.
func TestFaelligkeitUndGebuehrengruppe(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")
	var court, chamber kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Amtsgericht Hamburg", "plz": "20355"}), &court)
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Handelskammer Hamburg", "plz": "20457"}), &chamber)
	file := func(name string) uint {
		var out belegAblegenOut
		a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-10-02", "belegdatum": "2026-10-02",
			"aussteller": name, "brutto": "100.00", "dateien": []any{map[string]any{"pfad": h.inboxFile(name+".pdf", pdf(name))}}}), &out)
		return out.Beleg.ID
	}
	type vorschauAntwort struct {
		Vorschau struct {
			Zeilen []struct {
				Konto string `json:"konto"`
			} `json:"zeilen"`
			FaelligAm string `json:"faellig_am"`
			Hinweise  []struct {
				Code string `json:"code"`
			} `json:"hinweise"`
		} `json:"vorschau"`
		Buchung *struct {
			ID uint `json:"id"`
		} `json:"buchung"`
	}
	hint := func(v vorschauAntwort, code string) bool {
		for _, h := range v.Vorschau.Hinweise {
			if h.Code == code {
				return true
			}
		}
		return false
	}
	args := func(beleg, kontakt uint, gruppe string, extra map[string]any) map[string]any {
		in := map[string]any{"beleg_id": beleg, "kontakt_id": kontakt, "belegdatum": "2026-10-02",
			"leistung_von": "2026-10-02", "steuerfall": "not_taxable", "bezahlung": "offen",
			"positionen": []any{map[string]any{"buchungsgruppe": gruppe, "netto": "100.00", "satz": "0"}}}
		for k, v := range extra {
			in[k] = v
		}
		return a.args(in)
	}

	fee := file("Kostenrechnung")
	var v vorschauAntwort
	a.ok("beleg_buchen_mit_rueckfrage", args(fee, court.Kontakt.ID, "beitraege", map[string]any{"nur_vorschau": true}), &v)
	if !hint(v, "gruppe_passt_nicht_zum_aussteller") || !hint(v, "faelligkeit_aus_zahlungsziel") || v.Vorschau.FaelligAm != "2026-10-16" {
		t.Fatalf("Gericht unter Beiträgen: %+v", v.Vorschau)
	}
	a.ok("beleg_buchen_mit_rueckfrage", args(fee, court.Kontakt.ID, "gebuehren", map[string]any{"nur_vorschau": true}), &v)
	if hint(v, "gruppe_passt_nicht_zum_aussteller") || len(v.Vorschau.Zeilen) == 0 || v.Vorschau.Zeilen[0].Konto != "6430" {
		t.Fatalf("Gericht unter Gebühren: %+v", v.Vorschau)
	}

	dues := file("Beitragsbescheid")
	a.fail("beleg_buchen_mit_rueckfrage", args(dues, chamber.Kontakt.ID, "beitraege",
		map[string]any{"faellig_am": "2026-09-30"}), CodeRule)
	a.ok("beleg_buchen_mit_rueckfrage", args(dues, chamber.Kontakt.ID, "beitraege",
		map[string]any{"faellig_am": "2026-11-02", "nur_vorschau": true}), &v)
	if hint(v, "faelligkeit_aus_zahlungsziel") || hint(v, "gruppe_passt_nicht_zum_aussteller") || v.Vorschau.FaelligAm != "2026-11-02" {
		t.Fatalf("Kammer mit Fälligkeit laut Bescheid: %+v", v.Vorschau)
	}
	a.ok("beleg_buchen_mit_rueckfrage", args(dues, chamber.Kontakt.ID, "beitraege",
		map[string]any{"faellig_am": "2026-11-02", "idempotenz_schluessel": "ihk-beitrag-2026"}), &v)
	h.read(func(ctx context.Context, s *core.Services) error {
		items, err := s.Payments.OpenItems(ctx)
		if err != nil {
			return err
		}
		for _, it := range items {
			if it.EntryID == v.Buchung.ID {
				if it.DueDate != "2026-11-02" {
					t.Fatalf("offener Posten fällig am %s", it.DueDate)
				}
				return nil
			}
		}
		t.Fatalf("offener Posten fehlt: %+v", items)
		return nil
	})
}
