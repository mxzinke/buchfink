package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestTangiblePartialDisposalAndReversal(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	env.journal.AddEntryEffects(svc.DepreciationEffect())
	ctx := context.Background()
	asset, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Vier Maschinen", Class: domain.AssetClassTangible,
		Account: "0440", DepreciationAccount: "6220", AcquisitionDate: "2026-01-10",
		AcquisitionCost: 1_200_000, UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
		Quantity: 40_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: asset.ID, Date: "2026-06-30", Kind: domain.DisposalSale,
		Quantity: 10_000, Proceeds: 300_000, Settlement: SettlementPaid,
		PaymentAccount: "1800", TaxTreatment: domain.TaxTreatmentDomestic,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Gesamt-AfA 1.500 €, abgegangene AHK 3.000 €, Restbuchwert 2.625 €.
	if result.Asset.Cost != 900_000 || result.Asset.Accumulated != 112_500 || result.Asset.UnitsHeld != 30_000 {
		t.Fatalf("Restbestand: %+v", result.Asset)
	}
	run, err := svc.Run(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Due) != 1 || run.Due[0].Due != 112_500 {
		t.Fatalf("restliche AfA: %+v", run)
	}
	if _, err := env.journal.Reverse(ctx, result.DisposalEntry.ID, "Verkauf berichtigen"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Get(ctx, 2026, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Asset.Cost != 1_200_000 || detail.Asset.Accumulated != 150_000 || detail.Asset.UnitsHeld != 40_000 {
		t.Fatalf("nach Storno: %+v", detail.Asset)
	}
}

func TestPoolItemDisposalLeavesPoolAndCanBeReversed(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)
	ctx := context.Background()
	pool, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Sammelposten", Class: domain.AssetClassTangible, Account: "0675",
		DepreciationAccount: "6264", AcquisitionDate: "2026-01-01", AcquisitionCost: 60_000,
		Method: domain.DepreciationPool, PoolYear: 2026,
	})
	if err != nil {
		t.Fatal(err)
	}
	item := pool.Movements[0].ID
	result, err := svc.Dispose(ctx, DisposalRequest{
		AssetID: pool.ID, Date: "2026-06-30", Kind: domain.DisposalScrapped, PoolItemMovementID: item,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Asset.Cost != 60_000 || result.Asset.IsDisposed() || result.Asset.DueAmount != 12_000 {
		t.Fatalf("Sammelposten verändert: %+v", result.Asset)
	}
	var disposal uint
	for _, m := range result.Asset.Movements {
		if m.Kind == domain.AssetMovementDisposal {
			disposal = m.ID
		}
	}
	if _, err := svc.ReverseDisposal(ctx, pool.ID, disposal, "Irrtum"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReverseDisposal(ctx, pool.ID, disposal, "noch einmal"); err == nil {
		t.Fatal("doppeltes Storno")
	}
}
