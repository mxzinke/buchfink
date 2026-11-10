package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
)

// Der Agent bucht die Eröffnung eines Umsteigers aus der abgelegten
// Schlussbilanz: abgewiesen ohne festgelegten Umstieg, danach Vorschau,
// Buchung, Wiederholung und Abstimmung.
func TestOpeningFromLegacySystem(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")

	var customer, vendor kontaktOut
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "kunde", "name": "Kunde GmbH", "plz": "10115", "ort": "Berlin"}), &customer)
	a.ok("kontakt_anlegen", a.args(map[string]any{"typ": "lieferant", "name": "Lieferant GmbH", "plz": "20095", "ort": "Hamburg"}), &vendor)
	var statement ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": today(), "art": "sonstiges",
		"dateien":    []any{map[string]any{"pfad": h.inboxFile("schlussbilanz.pdf", pdf("Schlussbilanz 2025"))}},
		"belegdatum": "2025-12-31", "aussteller": "Vorgängerprogramm", "betreff": "Schlussbilanz zum 31.12.2025"}), &statement)

	args := a.args(map[string]any{
		"beleg_id": statement.Beleg.ID, "vorsystem": "Vorgängerprogramm", "nur_vorschau": true,
		"idempotenz_schluessel": "eroeffnung-2026",
		"konten": []any{
			map[string]any{"konto": "0520", "seite": "S", "betrag": "20000.00"},
			map[string]any{"konto": "1200", "seite": "S", "betrag": "1190.00"},
			map[string]any{"konto": "1800", "seite": "S", "betrag": "15000.00"},
			map[string]any{"konto": "2900", "seite": "H", "betrag": "25000.00"},
			map[string]any{"konto": "2970", "seite": "H", "betrag": "7110.00"},
			map[string]any{"konto": "3300", "seite": "H", "betrag": "4080.00"},
		},
		"offene_posten": []any{
			map[string]any{"kontakt_id": customer.Kontakt.ID, "betrag": "1190.00", "rechnungsnummer": "RE-2025-0098",
				"rechnungsdatum": "2025-12-10", "faellig_am": "2026-01-09", "satz": "19"},
			map[string]any{"kontakt_id": vendor.Kontakt.ID, "betrag": "4080.00", "rechnungsnummer": "ER-4711",
				"rechnungsdatum": "2025-12-15"},
		},
		"anlagen": []any{map[string]any{"bezeichnung": "Pkw Kombi", "konto": "0520", "angeschafft_am": "2024-01-15",
			"anschaffungskosten": "30000.00", "afa_bisher": "10000.00", "nutzungsdauer_monate": 72,
			"inventarnummer": "INV-0007"}},
	})

	// Ohne den Umstieg, den nur der Mensch festlegt, weist der Dienst ab.
	if e := a.fail("eroeffnung_buchen", args, CodeRule); !strings.Contains(e.Nachricht, "kein Umstieg festgelegt") {
		t.Fatalf("ohne Umstieg: %+v", e)
	}
	var report auswertungOut
	a.ok("auswertung", map[string]any{"art": "umstieg"}, &report)
	if report.Umstieg != nil || !strings.Contains(report.Hinweis, "kein Umstieg festgelegt") {
		t.Fatalf("Auswertung ohne Umstieg: %+v", report)
	}
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Takeover.SetDate(ctx, "2026-01-01")
		return err
	})

	var preview, booked, replay eroeffnungOut
	a.ok("eroeffnung_buchen", args, &preview)
	if !preview.Vorschau || preview.Stichtag != "2026-01-01" || len(preview.Differenzen) != 0 ||
		len(preview.Buchungen) != 3 || preview.Buchungen[0].ID != 0 || preview.Aktiva != "36190.00" ||
		len(preview.Anlagen) != 1 || preview.Anlagen[0].Buchwert != "20000.00" ||
		preview.Anlagen[0].AfaErstesJahr != "5000.00" {
		t.Fatalf("Vorschau: %+v", preview)
	}

	// Ein Anlagenverzeichnis, das vom Anlagekonto abweicht, zeigt die Vorschau
	// als Differenz; gebucht wird es nicht.
	wrong := a.args(map[string]any{})
	for k, v := range args {
		wrong[k] = v
	}
	wrong["idempotenz_schluessel"] = "eroeffnung-2026-falsch"
	wrong["anlagen"] = []any{map[string]any{"bezeichnung": "Pkw Kombi", "konto": "0520", "angeschafft_am": "2024-01-15",
		"anschaffungskosten": "30000.00", "afa_bisher": "9000.00", "nutzungsdauer_monate": 72}}
	var differing eroeffnungOut
	a.ok("eroeffnung_buchen", wrong, &differing)
	if len(differing.Differenzen) != 1 || !strings.Contains(differing.Differenzen[0], "Konto 0520") {
		t.Fatalf("Vorschau mit Differenz: %+v", differing.Differenzen)
	}
	wrong["nur_vorschau"] = false
	if e := a.fail("eroeffnung_buchen", wrong, CodeRule); !strings.Contains(e.Nachricht, "Differenz 1.000,00 €") {
		t.Fatalf("Buchung mit Differenz: %+v", e)
	}

	args["nur_vorschau"] = false
	a.ok("eroeffnung_buchen", args, &booked)
	a.ok("eroeffnung_buchen", args, &replay)
	if booked.Vorschau || len(booked.Buchungen) != 3 || booked.Buchungen[0].ID == 0 ||
		booked.Anlagen[0].AnlageID == 0 || booked.Anlagen[0].Inventarnummer == "" {
		t.Fatalf("Buchung: %+v", booked)
	}
	if !replay.Wiederholt || replay.Buchungen[0].ID != booked.Buchungen[0].ID {
		t.Fatalf("Wiederholung: %+v", replay)
	}

	a.ok("auswertung", map[string]any{"art": "umstieg"}, &report)
	if report.Umstieg == nil || !report.Umstieg.Gebucht || !report.Umstieg.Abgestimmt ||
		len(report.Umstieg.Differenzen) != 0 || len(report.Umstieg.Konten) != 6 ||
		len(report.Umstieg.Nebenbuecher) != 3 || report.Umstieg.BelegID != statement.Beleg.ID {
		t.Fatalf("Abstimmung: %+v", report.Umstieg)
	}
	for _, k := range report.Umstieg.Konten {
		if k.Differenz != "0.00" {
			t.Errorf("Differenz auf Konto %s: %+v", k.Konto, k)
		}
		if k.Konto == "1200" && (k.Schlussbilanz != "1190.00" || k.Gebucht != "1190.00") {
			t.Errorf("Sammelkonto der Forderungen: %+v", k)
		}
	}

	var items offenePostenOut
	a.ok("offene_posten", map[string]any{"stichtag": "2026-01-31"}, &items)
	if len(items.Posten) != 2 {
		t.Fatalf("offene Posten nach der Eröffnung: %+v", items.Posten)
	}
	var assets struct {
		Anlagen []anlage `json:"anlagen"`
	}
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026}, &assets)
	if len(assets.Anlagen) != 1 || assets.Anlagen[0].AK != "30000.00" || assets.Anlagen[0].Buchwert != "20000.00" ||
		assets.Anlagen[0].AfAFaellig != "5000.00" {
		t.Fatalf("Anlagenverzeichnis: %+v", assets.Anlagen)
	}
}
