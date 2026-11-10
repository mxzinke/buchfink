package core

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

func TestAssetRateIsFetchedBeforeWrite(t *testing.T) {
	rates := &countingRates{}
	h := newHostWith(t, Network{Rates: rates})
	newTenant(t, h, "Fremdwährung GmbH")
	tenant := openTenant(t, h)
	var id uint
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "USD-Anteile", Class: domain.AssetClassFinancial, Account: "0900",
			AcquisitionDate: "2026-01-01", AcquisitionCost: 1_200_000, Currency: "USD",
			ForeignCost: 1_085_000, Method: domain.DepreciationNone,
		})
		if err == nil {
			id = asset.ID
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	req := service.CurrencyValuationRequest{AssetID: id, Date: "2026-09-30"}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Assets.BookCurrencyValuation(ctx, req)
		return err
	}); err == nil {
		t.Fatal("Bewertung ohne vorbereiteten Kurs")
	}
	if rates.calls.Load() != 0 {
		t.Fatal("Kurs wurde im Schreibvorgang geholt")
	}
	if err := tenant.PrefetchAssetRate(uiCtx, id, req.Date); err != nil {
		t.Fatal(err)
	}
	if rates.calls.Load() != 1 {
		t.Fatalf("Kursabrufe: %d", rates.calls.Load())
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		entry, err := s.Assets.BookCurrencyValuation(ctx, req)
		if err != nil {
			return err
		}
		if entry.Lines[0].Account != "6880" || entry.Lines[0].Amount != 200_000 {
			t.Fatalf("Kursverlust: %+v", entry.Lines)
		}
		_, err = s.Journal.Reverse(ctx, entry.ID, "Bewertung berichtigen")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if rates.calls.Load() != 1 {
		t.Fatal("erneuter Kursabruf beim Schreiben")
	}
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		detail, err := s.Assets.Get(ctx, 2026, id)
		if err == nil && detail.Asset.BookValue != 1_200_000 {
			t.Fatalf("Buchwert nach Storno: %v", detail.Asset.BookValue)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
