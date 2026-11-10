package mcpserver

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

func TestAssetDepreciationReversalThroughTool(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Fräsmaschine", Class: domain.AssetClassTangible,
			Account: "0440", DepreciationAccount: "6220",
			AcquisitionDate: "2026-01-10", AcquisitionCost: 1_200_000,
			UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
		})
		return err
	})
	a := h.agent("")
	var booked abschlussAntwort
	a.ok("abschlussbuchung", a.args(map[string]any{
		"art": "afa_lauf", "geschaeftsjahr": 2026, "idempotenz_schluessel": "afa-2026",
	}), &booked)
	if len(booked.BuchungIDs) != 1 {
		t.Fatalf("AfA: %+v", booked)
	}
	a.ok("buchung_stornieren", a.args(map[string]any{
		"buchung_id": booked.BuchungIDs[0], "grund": "AfA berichtigen",
	}), nil)
	var assets anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026}, &assets)
	if len(assets.Anlagen) != 1 || assets.Anlagen[0].Buchwert != Betrag("12000.00") ||
		assets.Anlagen[0].AfAFaellig != Betrag("3000.00") {
		t.Fatalf("Kartei nach Storno: %+v", assets)
	}
}

func TestAssetDisposalThroughTool(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	var id uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Maschine", Class: domain.AssetClassTangible,
			Account: "0440", DepreciationAccount: "6220", AcquisitionDate: "2026-01-01",
			AcquisitionCost: 1_200_000, Method: domain.DepreciationLinear, UsefulLifeMonths: 48,
		})
		if err == nil {
			id = asset.ID
		}
		return err
	})
	a := h.agent("")
	args := a.args(map[string]any{"anlage_id": id, "datum": "2026-06-30", "art": "verkauf",
		"idempotenz_schluessel": "abgang-2026", "erloes": "12000.00", "zahlungskonto": "1800", "grund": "Verkauf", "nur_vorschau": true})
	var preview anlageAbgangOut
	a.ok("anlage_abgehen", args, &preview)
	if preview.Vorschau == nil || preview.Vorschau.CatchUpAmount != 150_000 || preview.Vorschau.Tax != 228_000 {
		t.Fatalf("Vorschau: %+v", preview)
	}
	args["nur_vorschau"] = false
	var booked anlageAbgangOut
	a.ok("anlage_abgehen", args, &booked)
	if booked.Ergebnis == nil || booked.Ergebnis.DisposalEntry == nil {
		t.Fatalf("Abgang: %+v", booked)
	}
	var mirror anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "spiegel": true}, &mirror)
	if mirror.Spiegel == nil || mirror.Spiegel.Totals.Disposals != 1_200_000 {
		t.Fatalf("Anlagenspiegel: %+v", mirror.Spiegel)
	}
	a.ok("buchung_stornieren", a.args(map[string]any{
		"buchung_id": booked.Ergebnis.DisposalEntry.ID, "grund": "Verkauf berichtigen",
	}), nil)
	var assets anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026}, &assets)
	if assets.Anlagen[0].Buchwert != Betrag("10500.00") {
		t.Fatalf("Kartei: %+v", assets)
	}
}

func TestAssetCompletionThroughTool(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	var id uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Maschine im Bau", Class: domain.AssetClassTangible, Account: "0700",
			AcquisitionDate: "2026-01-01", AcquisitionCost: 1_200_000, Method: domain.DepreciationNone,
		})
		if err == nil {
			id = asset.ID
		}
		return err
	})
	a := h.agent("")
	var out anlageFertigOut
	a.ok("anlage_fertigstellen", a.args(map[string]any{
		"anlage_id": id, "datum": "2026-07-01", "konto": "0440", "abschreibungskonto": "6220",
		"methode": "linear", "nutzungsdauer_monate": 48, "grund": "Abnahme",
		"idempotenz_schluessel": "fertigstellung-2026",
	}), &out)
	var assets anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "anlage_id": id}, &assets)
	if assets.Detail == nil || assets.Detail.Asset.DueAmount != 150_000 || assets.Detail.Asset.Account != "0440" {
		t.Fatalf("Fertigstellung: %+v", assets)
	}
}

func TestAssetValuationAndWriteUpThroughTool(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	var id uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Beteiligung", Class: domain.AssetClassFinancial, Account: "0850",
			AcquisitionDate: "2026-01-01", AcquisitionCost: 1_000_000, Method: domain.DepreciationNone,
		})
		if err == nil {
			id = asset.ID
		}
		return err
	})
	a := h.agent("")
	args := func(kind, amount string, preview bool) map[string]any {
		return a.args(map[string]any{
			"art": "anlagenbewertung", "geschaeftsjahr": 2026, "nur_vorschau": preview,
			"idempotenz_schluessel": "bewertung-" + kind,
			"anlagenbewertung": map[string]any{"anlage_id": id, "art": kind, "datum": "2026-09-30",
				"betrag": amount, "grund": "Bewertung", "nachweis": "Gutachten vom 30.09.2026"},
		})
	}
	var preview, impaired, restored abschlussAntwort
	a.ok("abschlussbuchung", args("abschreibung", "2000.00", true), &preview)
	if len(preview.BuchungIDs) != 0 {
		t.Fatal("Vorschau hat gebucht")
	}
	a.ok("abschlussbuchung", args("abschreibung", "2000.00", false), &impaired)
	var report anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "wertaufholung": true}, &report)
	if report.Wertaufholung == nil || len(report.Wertaufholung.Candidates) != 1 ||
		report.Wertaufholung.Candidates[0].MaxWriteUp != 200_000 {
		t.Fatalf("Wertaufholung: %+v", report)
	}
	a.ok("abschlussbuchung", args("zuschreibung", "1000.00", false), &restored)
	for _, booked := range []abschlussAntwort{restored, impaired} {
		if len(booked.BuchungIDs) != 1 {
			t.Fatalf("Bewertung: %+v", booked)
		}
		a.ok("buchung_stornieren", a.args(map[string]any{"buchung_id": booked.BuchungIDs[0], "grund": "Bewertung berichtigen"}), nil)
	}
	var assets anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "anlage_id": id}, &assets)
	if assets.Detail.Asset.BookValue != 1_000_000 {
		t.Fatalf("Buchwert: %v", assets.Detail.Asset.BookValue)
	}
}

func TestForeignAssetAcquisitionValuationAndReversal(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	var purchase uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		entry, err := s.Journal.Post(ctx, &domain.JournalEntry{
			BookingDate: "2026-01-15", Description: "USD-Anteile", Source: domain.EntrySourceManual,
			TaxTreatment: domain.TaxTreatmentNotTaxable,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "0900", Amount: 1_000_000},
				{Side: domain.SideCredit, Account: "1800", Amount: 1_000_000},
			},
		})
		if err == nil {
			purchase = entry.ID
		}
		return err
	})
	a := h.agent("")
	var acquired abschlussAntwort
	a.ok("abschlussbuchung", a.args(map[string]any{
		"art": "fremdwaehrungsanlage", "geschaeftsjahr": 2026, "idempotenz_schluessel": "usd-zugang",
		"fremdwaehrungsanlage": map[string]any{"art": "zugang", "datum": "2026-01-15", "buchung_id": purchase,
			"konto": "0900", "bezeichnung": "USD-Anteile", "waehrung": "USD", "fremdbetrag": "12000.00",
			"kurs_millionstel": 1_200_000, "kursquelle": "Bankabrechnung"},
	}), &acquired)
	if acquired.AnlageID == nil {
		t.Fatalf("Zugang: %+v", acquired)
	}
	var valued abschlussAntwort
	a.ok("abschlussbuchung", a.args(map[string]any{
		"art": "fremdwaehrungsanlage", "geschaeftsjahr": 2026, "idempotenz_schluessel": "usd-bewertung",
		"fremdwaehrungsanlage": map[string]any{"art": "bewertung", "datum": "2026-09-30", "anlage_id": *acquired.AnlageID,
			"kurs_millionstel": 1_500_000},
	}), &valued)
	if len(valued.BuchungIDs) != 1 {
		t.Fatalf("Bewertung: %+v", valued)
	}
	var assets anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "anlage_id": *acquired.AnlageID}, &assets)
	if assets.Detail.Asset.BookValue != 800_000 {
		t.Fatalf("bewertet: %v", assets.Detail.Asset.BookValue)
	}
	for _, entry := range []uint{valued.BuchungIDs[0], purchase} {
		a.ok("buchung_stornieren", a.args(map[string]any{"buchung_id": entry, "grund": "Berichtigung"}), nil)
	}
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026, "anlage_id": *acquired.AnlageID}, &assets)
	if assets.Detail.Asset.BookValue != 0 {
		t.Fatalf("nach Storno: %v", assets.Detail.Asset.BookValue)
	}
}
