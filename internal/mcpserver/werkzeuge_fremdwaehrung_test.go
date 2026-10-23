package mcpserver

import (
	"context"
	"maps"
	"sync/atomic"
	"testing"
)

// fixedRates liefert einen festen Kurs und zählt die Abrufe.
type fixedRates struct{ calls atomic.Int32 }

func (r *fixedRates) RateAt(_ context.Context, _, date string) (int64, string, string, error) {
	r.calls.Add(1)
	return 1_120_000, date, "Testkurs", nil
}

// Der Kurs eines Belegs in Fremdwährung wird vor dem Schreibvorgang geholt,
// auch für die Vorschau.
func TestBelegInFremdwaehrung(t *testing.T) {
	rates := &fixedRates{}
	h := newHarness(t, withRates(rates))
	a := h.agent("")
	var supplier kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Codeforge Inc.", "plz": "94105"}), &supplier)
	var filed belegAblegenOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-10-02", "belegdatum": "2026-10-02",
		"aussteller": "Codeforge Inc.", "brutto": "84.00", "dateien": []any{map[string]any{"pfad": h.inboxFile("codeforge.pdf", pdf("Codeforge"))}}}), &filed)

	args := func(extra map[string]any) map[string]any {
		in := map[string]any{"beleg_id": filed.Beleg.ID, "kontakt_id": supplier.Kontakt.ID, "belegdatum": "2026-10-02",
			"leistung_von": "2026-10-02", "steuerfall": "not_taxable", "bezahlung": "offen",
			"waehrung": "usd", "fremdbetrag": "84.00",
			"positionen": []any{map[string]any{"buchungsgruppe": "gebuehren", "netto": "84.00", "satz": "0"}}}
		maps.Copy(in, extra)
		return a.args(in)
	}
	var out struct {
		Vorschau struct {
			Zeilen []struct {
				Konto  string `json:"konto"`
				Betrag string `json:"betrag"`
			} `json:"zeilen"`
		} `json:"vorschau"`
		Buchung *struct {
			ID uint `json:"id"`
		} `json:"buchung"`
	}
	a.ok("beleg_buchen_mit_rueckfrage", args(map[string]any{"nur_vorschau": true}), &out)
	if len(out.Vorschau.Zeilen) == 0 || out.Vorschau.Zeilen[0].Betrag != "75.00" {
		t.Fatalf("Vorschau in USD: %+v", out.Vorschau)
	}
	a.ok("beleg_buchen_mit_rueckfrage", args(nil), &out)
	if out.Buchung == nil || out.Buchung.ID == 0 {
		t.Fatalf("keine Buchung: %+v", out)
	}
	if n := rates.calls.Load(); n != 1 {
		t.Errorf("%d Kursabrufe, erwartet einen", n)
	}
}
