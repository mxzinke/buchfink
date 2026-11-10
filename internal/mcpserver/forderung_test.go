package mcpserver

import "testing"

func TestWriteOffPreviewAndReplay(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var customer kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Insolventer Kunde GmbH", "plz": "10115", "ort": "Berlin"}), &customer)
	var receipt ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": h.inboxFile("insolvenz.pdf", pdf("Nachweis Insolvenz"))}}, "belegdatum": "2026-03-10", "aussteller": "Insolvenzverwalter", "brutto": "119.00"}), &receipt)
	var booked erfasstOut
	a.ok("buchung_erfassen", a.args(map[string]any{"buchungsdatum": "2026-03-10", "beschreibung": "Forderung", "steuerfall": "domestic", "beleg_id": receipt.Beleg.ID, "idempotenz_schluessel": "forderung-test", "zeilen": []any{
		map[string]any{"konto": customer.Kontakt.Personenkonto, "kontakt_id": customer.Kontakt.ID, "seite": "S", "betrag": "119.00"},
		map[string]any{"konto": "4400", "seite": "H", "betrag": "100.00"},
		map[string]any{"konto": "3806", "seite": "H", "betrag": "19.00", "steuerschluessel": "UST19", "steuerbasis": "100.00"},
	}}), &booked)
	args := a.args(map[string]any{"buchung_id": booked.Buchung.ID, "beleg_id": receipt.Beleg.ID, "datum": "2026-04-10", "grund": "Insolvenz mangels Masse", "betrag": "59.50", "nur_vorschau": true, "idempotenz_schluessel": "ausbuchung-test"})
	var preview, result, replay ausbuchenOut
	a.ok("forderung_ausbuchen", args, &preview)
	if preview.Buchung.ID != 0 || len(preview.Buchung.Zeilen) != 3 {
		t.Fatalf("Vorschau: %+v", preview)
	}
	args["nur_vorschau"] = false
	a.ok("forderung_ausbuchen", args, &result)
	a.ok("forderung_ausbuchen", args, &replay)
	if result.Buchung.ID == 0 || !replay.Wiederholt || replay.Buchung.ID != result.Buchung.ID {
		t.Fatalf("Buchung/Wiederholung: %+v / %+v", result, replay)
	}
	if result.Buchung.Zeilen[0].Betrag != "50.00" || result.Buchung.Zeilen[1].Betrag != "9.50" {
		t.Fatalf("Buchungssatz: %+v", result)
	}
	var items offenePostenOut
	a.ok("offene_posten", map[string]any{"stichtag": "2026-04-30"}, &items)
	if len(items.Posten) != 1 || items.Posten[0].Offen != "59.50" {
		t.Fatalf("Rest: %+v", items)
	}
}
