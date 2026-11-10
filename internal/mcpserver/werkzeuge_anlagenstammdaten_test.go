package mcpserver

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

func TestAnlagenReadsMetadataChangedInApp(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	h.ui(func(ctx context.Context, s *core.Services) error {
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Wertpapier", Class: domain.AssetClassFinancial, Account: "0900",
			AcquisitionDate: "2026-01-10", AcquisitionCost: 1200000,
			Method: domain.DepreciationNone, Quantity: 10 * domain.UnitScale,
		})
		if err != nil {
			return err
		}
		_, err = s.Assets.UpdateMetadata(ctx, service.AssetMetadata{
			ID: asset.ID, Name: "Depotbestand", Description: "Depot 1", Identifier: "DE-Test",
			RegistrationNumber: "Inventar 42", BusinessUsePermille: 900,
			Quantity: 20 * domain.UnitScale, Reason: "Depotauszug abgeglichen",
		})
		return err
	})
	var result struct {
		Anlagen []anlage `json:"anlagen"`
	}
	h.agent("").ok("anlagen", map[string]any{"geschaeftsjahr": 2026}, &result)
	if len(result.Anlagen) != 1 {
		t.Fatalf("Anlagen: %+v", result)
	}
	got := result.Anlagen[0]
	if got.Bezeichnung != "Depotbestand" || got.Beschreibung != "Depot 1" ||
		got.Kennung != "DE-Test" || got.Kennzeichen != "Inventar 42" ||
		got.Nutzung != 900 || got.Stueckzahl != 20*domain.UnitScale {
		t.Fatalf("Stammdaten: %+v", got)
	}
}
