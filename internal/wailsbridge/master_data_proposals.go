package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// GetMasterDataProposals liefert die offenen Vorschläge eines Agenten für die
// Stammdaten mit ihrer Vorschau gegen den heutigen Stand.
func (b *BuchfinkBridge) GetMasterDataProposals() ([]service.MasterDataProposalView, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.MasterDataProposalView, error) {
		return s.MasterData.Pending(ctx)
	})
}

// AcceptMasterDataProposal übernimmt einen Vorschlag so, wie die Vorschau mit
// diesem Fingerabdruck ihn gezeigt hat; der Firmenname geht mit in die
// Mandantenliste. Nur in der App.
func (b *BuchfinkBridge) AcceptMasterDataProposal(id uint, fingerprint string) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		if _, err := s.MasterData.Accept(ctx, id, fingerprint); err != nil {
			return err
		}
		settings, err := s.Settings.GetCompanySettings(ctx)
		if err != nil || settings.CompanyName == "" || settings.CompanyName == s.Tenant.Name {
			return err
		}
		return b.host.RenameTenant(s.Tenant.ID, settings.CompanyName)
	}, core.AppOnly())
}

// RejectMasterDataProposal lehnt einen Vorschlag ab; die Notiz sieht der
// Agent. Nur in der App.
func (b *BuchfinkBridge) RejectMasterDataProposal(id uint, note string) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		_, err := s.MasterData.Reject(ctx, id, note)
		return err
	}, core.AppOnly())
}
