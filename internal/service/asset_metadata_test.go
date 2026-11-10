package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestAssetMetadataPreservesBookedValuesAndAudit(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	asset := env.machine(t, svc)
	ctx := context.Background()
	result, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || len(result.Entries[0].Lines) != 2 {
		t.Fatalf("AfA-Buchung: %+v", result)
	}
	for _, line := range result.Entries[0].Lines {
		if line.Amount != 300000 ||
			(line.Side == domain.SideDebit && line.Account != "6220") ||
			(line.Side == domain.SideCredit && line.Account != "0440") {
			t.Fatalf("AfA-Buchungssatz: %+v", line)
		}
	}
	req := AssetMetadata{
		ID: asset.ID, Name: "Fräsmaschine Halle 2", Identifier: "M-12",
		RegistrationNumber: "B-F 12", BusinessUsePermille: 900, Reason: "Standort und Nutzung geprüft",
	}
	changed, err := svc.UpdateMetadata(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if changed.AcquisitionCost != 1200000 || changed.UsefulLifeMonths != 48 || changed.Method != domain.DepreciationLinear {
		t.Fatalf("Bewertung verändert: %+v", changed)
	}
	detail, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.BookValue != 900000 || detail.Asset.Accumulated != 300000 {
		t.Fatalf("Werte verändert: %+v", detail.Asset)
	}
	audit := latestAudit(t, env, "ANLAGE")
	if fields(t, audit.Before)["name"] != "Fräsmaschine" || fields(t, audit.After)["name"] != req.Name ||
		!strings.Contains(audit.Details, req.Reason) {
		t.Fatalf("Protokoll: %+v", audit)
	}
	reversal, err := env.journal.Reverse(ctx, result.Entries[0].ID, "AfA berichtigen")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range reversal.Lines {
		if line.Amount != -300000 {
			t.Fatalf("Generalumkehr: %+v", line)
		}
	}
	detail, err = svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.Name != req.Name || detail.Asset.Identifier != req.Identifier {
		t.Fatalf("Storno: %+v", detail.Asset)
	}
	req.Reason = " "
	if _, err := svc.UpdateMetadata(ctx, req); err == nil {
		t.Fatal("Grund fehlt")
	}
	req.Reason = "Korrektur"
	req.BusinessUsePermille = 1001
	if _, err := svc.UpdateMetadata(ctx, req); err == nil {
		t.Fatal("ungültige Quote")
	}
}

func TestAssetMetadataQuantityAndCommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Wertpapier", Class: domain.AssetClassFinancial, Account: "0900",
		AcquisitionDate: "2026-01-10", AcquisitionCost: 1200000,
		Method: domain.DepreciationNone, Quantity: 10 * domain.UnitScale,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := AssetMetadata{ID: asset.ID, Name: asset.Name, Quantity: 20 * domain.UnitScale, Reason: "Stückzahl abgeglichen"}
	got, err := svc.UpdateMetadata(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitsHeld != 20*domain.UnitScale || got.BookValue != 1200000 {
		t.Fatalf("Bestand: %+v", got)
	}
	repo := repository.NewFestschreibungRepository(env.db)
	env.journal.SetFestschreibungRepo(repo)
	if err := repo.Create(ctx, &domain.Festschreibung{FiscalYear: 2026, PeriodType: domain.PeriodTypeYear, CutoffDate: "2026-12-31"}); err != nil {
		t.Fatal(err)
	}
	req.Quantity = 30 * domain.UnitScale
	if _, err := svc.UpdateMetadata(ctx, req); err == nil {
		t.Fatal("Stückzahl in fester Periode geändert")
	}
	req.Quantity = 20 * domain.UnitScale
	req.Identifier = "ISIN korrigiert"
	if _, err := svc.UpdateMetadata(ctx, req); err != nil {
		t.Fatal(err)
	}
}

func TestAssetMetadataRejectsQuantityAfterMovement(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Wertpapier", Class: domain.AssetClassFinancial, Account: "0900",
		AcquisitionDate: "2026-01-10", AcquisitionCost: 1200000,
		Method: domain.DepreciationNone, Quantity: 10 * domain.UnitScale,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BookImpairment(ctx, ImpairmentRequest{Evidence: "Wertgutachten",
		AssetID: asset.ID, Date: "2026-12-31", Amount: 100000, Reason: "Dauerhafter Wertverlust",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateMetadata(ctx, AssetMetadata{
		ID: asset.ID, Name: "Geändert", Quantity: 20 * domain.UnitScale, Reason: "Korrektur",
	}); err == nil {
		t.Fatal("Stückzahl trotz Folgebewegung geändert")
	}
	got, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset.Name != "Wertpapier" || got.Asset.UnitsHeld != 10*domain.UnitScale ||
		got.Asset.BookValue != 1100000 {
		t.Fatalf("Abgewiesene Änderung gespeichert: %+v", got.Asset)
	}
}
