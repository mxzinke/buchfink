package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// SaveServiceProof schreibt den Leistungsnachweis an einen Eingangsbeleg.
func (b *BuchfinkBridge) SaveServiceProof(receiptID uint, text, date string) (*domain.Receipt, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Receipt, error) {
		return s.Receipts.SaveServiceProof(ctx, receiptID, text, date)
	})
}
