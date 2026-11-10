package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// -------------------------------------------------------------------------
// Nicht abziehbare Betriebsausgaben
// -------------------------------------------------------------------------

// GetNonDeductibleReport ist der Bericht eines Geschäftsjahres.
func (b *BuchfinkBridge) GetNonDeductibleReport(year int) (*service.NonDeductibleReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.NonDeductibleReport, error) {
		return s.Gifts.NonDeductibleReport(ctx, year)
	})
}
