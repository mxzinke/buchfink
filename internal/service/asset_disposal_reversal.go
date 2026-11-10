package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// ReverseDisposal nimmt auch rein mengenmäßige Abgänge ohne Journalbuchung zurück.
func (s *AssetService) ReverseDisposal(ctx context.Context, assetID, movementID uint, reason string) (*domain.FixedAsset, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.FixedAsset, error) {
		if strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("der Stornogrund fehlt")
		}
		asset, err := s.assetRepo.FindByID(ctx, assetID)
		if err != nil {
			return nil, err
		}
		for _, m := range asset.Movements {
			if m.ReversesMovementID != nil && *m.ReversesMovementID == movementID {
				return nil, fmt.Errorf("der Abgang ist bereits zurückgenommen")
			}
		}
		for _, m := range asset.Movements {
			if m.ID != movementID || m.Kind != domain.AssetMovementDisposal || m.ReversesMovementID != nil {
				continue
			}
			if m.JournalEntryID != nil {
				if _, err := s.journalSvc.Reverse(ctx, *m.JournalEntryID, reason); err != nil {
					return nil, err
				}
			} else {
				date := s.today()
				state, err := s.journalSvc.periods.State(ctx, date)
				if err != nil {
					return nil, err
				}
				if err := state.ensureOpen(date, false); err != nil {
					return nil, err
				}
				undo := domain.AssetMovement{
					AssetID: assetID, Kind: m.Kind, Account: m.Account, Date: date,
					FiscalYear: domain.GetFiscalYearForDate(date, s.fiscalYearStartMonth(ctx)),
					CostAmount: -m.CostAmount, DepreciationAmount: -m.DepreciationAmount,
					Quantity: -m.Quantity, ReversesMovementID: &m.ID, Note: reason,
				}
				if err := s.assetRepo.AddMovement(ctx, &undo); err != nil {
					return nil, err
				}
				if asset.DisposalDate == m.Date && asset.DisposalEntryID == nil {
					asset.DisposalDate, asset.DisposalKind, asset.DisposalProceeds = "", "", 0
					if err := s.assetRepo.Save(ctx, asset); err != nil {
						return nil, err
					}
				}
				if err := s.audit(ctx, domain.AuditActionUpdate, assetID, "Abgang zurückgenommen: "+reason); err != nil {
					return nil, err
				}
			}
			return s.reload(ctx, assetID, "")
		}
		return nil, fmt.Errorf("der Abgang wurde nicht gefunden")
	})
}
