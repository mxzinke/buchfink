package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// EntryEffect hält die Kartei beim Storno mit dem Journal in derselben
// Transaktion gleich. Die ursprüngliche Bewegung bleibt nachvollziehbar.
func (s *AssetService) EntryEffect() domain.EntryEffect {
	return domain.EntryEffect{Name: "asset_register", AfterReverse: func(
		ctx context.Context, original, reversal *domain.JournalEntry,
	) error {
		assets, err := s.assetRepo.FindAll(ctx)
		if err != nil {
			return err
		}
		for _, asset := range assets {
			for _, movement := range asset.Movements {
				if movement.JournalEntryID == nil || *movement.JournalEntryID != original.ID {
					continue
				}
				switch movement.Kind {
				case domain.AssetMovementDepreciation, domain.AssetMovementDisposal, domain.AssetMovementTransfer,
					domain.AssetMovementImpairment, domain.AssetMovementWriteUp, domain.AssetMovementAcquisition:
				default:
					continue
				}
				if movement.Kind == domain.AssetMovementAcquisition && asset.Currency == "" {
					continue
				}
				if err := ensureAssetMovementReversible(&asset, movement); err != nil {
					return err
				}
				if movement.Kind == domain.AssetMovementTransfer && movement.TransferBefore != "" {
					var before assetTransferState
					if err := json.Unmarshal([]byte(movement.TransferBefore), &before); err != nil {
						return err
					}
					asset.Account, asset.DepreciationAccount = before.Account, before.DepreciationAccount
					asset.Method, asset.UsefulLifeMonths, asset.InServiceDate = before.Method, before.UsefulLifeMonths, before.InServiceDate
					if err := s.assetRepo.Save(ctx, &asset); err != nil {
						return err
					}
				}
				undo := domain.AssetMovement{
					AssetID: asset.ID, Kind: movement.Kind, Account: movement.Account,
					Date: reversal.BookingDate, FiscalYear: reversal.FiscalYear,
					DepreciationAmount: -movement.DepreciationAmount,
					CostAmount:         -movement.CostAmount, Quantity: -movement.Quantity, ReversesMovementID: &movement.ID,
					JournalEntryID: &reversal.ID,
					Note:           fmt.Sprintf("Storno der Anlagenbewegung: %s", original.EntryNumber),
				}
				if movement.Kind == domain.AssetMovementDisposal && asset.DisposalEntryID != nil &&
					*asset.DisposalEntryID == original.ID {
					asset.DisposalDate, asset.DisposalKind, asset.DisposalProceeds = "", "", 0
					asset.DisposalEntryID = nil
					if err := s.assetRepo.Save(ctx, &asset); err != nil {
						return err
					}
				}
				if err := s.assetRepo.AddMovement(ctx, &undo); err != nil {
					return err
				}
			}
		}
		return nil
	}}
}

type assetTransferState struct {
	Account             string
	DepreciationAccount string
	Method              domain.DepreciationMethod
	UsefulLifeMonths    int
	InServiceDate       string
}

// Folgebewegungen verwenden den Buchwert nach der ursprünglichen Bewegung.
// Ihre Rücknahme muss deshalb zuerst erfolgen.
func ensureAssetMovementReversible(asset *domain.FixedAsset, movement domain.AssetMovement) error {
	reversed := map[uint]bool{}
	for _, m := range asset.Movements {
		if m.ReversesMovementID != nil {
			reversed[*m.ReversesMovementID] = true
		}
	}
	for _, later := range asset.Movements {
		if later.ID <= movement.ID || later.ReversesMovementID != nil || reversed[later.ID] ||
			later.Kind == domain.AssetMovementIncome {
			continue
		}
		if later.JournalEntryID != nil && movement.JournalEntryID != nil &&
			*later.JournalEntryID == *movement.JournalEntryID {
			continue
		}
		return fmt.Errorf("zuerst die Folgebewegung vom %s zurücknehmen", later.Date)
	}
	return nil
}
