package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// DepreciationEffect hält die Kartei beim Storno mit dem Journal in derselben
// Transaktion gleich. Die ursprüngliche Bewegung bleibt nachvollziehbar.
func (s *AssetService) DepreciationEffect() domain.EntryEffect {
	return domain.EntryEffect{Name: "asset_depreciation", AfterReverse: func(
		ctx context.Context, original, reversal *domain.JournalEntry,
	) error {
		assets, err := s.assetRepo.FindAll(ctx)
		if err != nil {
			return err
		}
		for _, asset := range assets {
			for _, movement := range asset.Movements {
				if movement.JournalEntryID == nil || *movement.JournalEntryID != original.ID ||
					movement.Kind != domain.AssetMovementDepreciation {
					continue
				}
				undo := domain.AssetMovement{
					AssetID: asset.ID, Kind: movement.Kind, Account: movement.Account,
					Date: reversal.BookingDate, FiscalYear: reversal.FiscalYear,
					DepreciationAmount: -movement.DepreciationAmount,
					JournalEntryID:     &reversal.ID,
					Note:               fmt.Sprintf("Storno der Abschreibung: %s", original.EntryNumber),
				}
				if err := s.assetRepo.AddMovement(ctx, &undo); err != nil {
					return err
				}
			}
		}
		return nil
	}}
}
