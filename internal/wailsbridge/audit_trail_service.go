package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// GetAuditTrail liefert den Prüfpfad eines Belegs zur Anzeige.
//
// Dieselbe Kette, die auch ausgegeben wird: die Anwenderin sieht auf dem Schirm,
// was der Prüfer in der Datei bekommt.
func (b *BuchfinkBridge) GetAuditTrail(receiptID uint) (*service.AuditTrail, error) {
	return readOr(b, func() (*service.AuditTrail, error) {
		empty := &service.AuditTrail{ReceiptID: receiptID}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.AuditTrail, error) {
		trail, err := s.AuditTrail.Trail(ctx, receiptID)
		if err != nil {
			return nil, err
		}
		trail.EnsureLists()
		return trail, nil
	})
}
