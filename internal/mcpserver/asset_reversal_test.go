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
		"idempotenz_schluessel": "abgang-2026", "erloes": "12000.00", "bankkonto": "1800", "grund": "Verkauf", "nur_vorschau": true})
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
	a.ok("buchung_stornieren", a.args(map[string]any{
		"buchung_id": booked.Ergebnis.DisposalEntry.ID, "grund": "Verkauf berichtigen",
	}), nil)
	var assets anlagenOut
	a.ok("anlagen", map[string]any{"geschaeftsjahr": 2026}, &assets)
	if assets.Anlagen[0].Buchwert != Betrag("10500.00") {
		t.Fatalf("Kartei: %+v", assets)
	}
}
