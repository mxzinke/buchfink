package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// AssetMetadata enthält nur beschreibende Angaben. Bewertungsparameter bleiben
// außerhalb dieses Vertrags, weil sie den Plan auch festgeschriebener Jahre ändern.
type AssetMetadata struct {
	ID                  uint         `json:"id"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Identifier          string       `json:"identifier"`
	RegistrationNumber  string       `json:"registrationNumber"`
	BusinessUsePermille int          `json:"businessUsePermille"`
	Quantity            domain.Units `json:"quantity"`
	Reason              string       `json:"reason"`
}

func (s *AssetService) UpdateMetadata(ctx context.Context, req AssetMetadata) (*domain.FixedAsset, error) {
	if err := requireHuman(ctx, "Anlagenstammdaten ändern"); err != nil {
		return nil, err
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.FixedAsset, error) {
		if strings.TrimSpace(req.Reason) == "" {
			return nil, fmt.Errorf("bitte begründen Sie die Änderung")
		}
		before, err := s.assetRepo.FindByID(ctx, req.ID)
		if err != nil {
			return nil, err
		}
		after := *before
		after.Name = strings.TrimSpace(req.Name)
		after.Description = strings.TrimSpace(req.Description)
		after.Identifier = strings.TrimSpace(req.Identifier)
		after.RegistrationNumber = strings.TrimSpace(req.RegistrationNumber)
		after.BusinessUsePermille = req.BusinessUsePermille
		after.Quantity = req.Quantity
		if err := after.Validate(); err != nil {
			return nil, err
		}
		if req.Quantity != before.Quantity {
			state, err := s.journalSvc.periods.State(ctx, before.AcquisitionDate)
			if err != nil {
				return nil, err
			}
			if err := state.ensureNotAdopted(before.AcquisitionDate); err != nil {
				return nil, err
			}
			if err := state.ensureOpen(before.AcquisitionDate, false); err != nil {
				return nil, err
			}
			// Stückzahlen bestimmen die Kostenanteile eines Teilabgangs. Eine Korrektur
			// ist deshalb nur vor der ersten Buchung und weiteren Bewegung zulässig.
			if len(before.Movements) != 1 || before.Movements[0].Kind != domain.AssetMovementAcquisition ||
				before.Movements[0].JournalEntryID != nil || before.AcquisitionEntryID != nil {
				return nil, fmt.Errorf("die Stückzahl lässt sich nur vor der ersten Buchung und weiteren Bewegung korrigieren")
			}
			movement := before.Movements[0]
			movement.Quantity = req.Quantity
			after.Movements = []domain.AssetMovement{movement}
			if err := s.assetRepo.AddMovement(ctx, &movement); err != nil {
				return nil, err
			}
		}
		if err := s.assetRepo.Save(ctx, &after); err != nil {
			return nil, err
		}
		details := "Anlagenstammdaten: " + strings.TrimSpace(req.Reason)
		if err := s.auditChange(ctx, domain.AuditActionUpdate, after.ID, details, before, &after); err != nil {
			return nil, err
		}
		return s.reload(ctx, after.ID, "")
	})
}
