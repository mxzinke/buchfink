package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// --- Monatsabschluss ------------------------------------------------------

// GetMonthCloseState liefert den Stand eines Monats ("JJJJ-MM") in seinen drei
// Schritten.
func (b *BuchfinkBridge) GetMonthCloseState(month string) (*service.MonthCloseState, error) {
	return readOr(b, func() (*service.MonthCloseState, error) {
		empty := &service.MonthCloseState{Month: month}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.MonthCloseState, error) {
		state, err := s.MonthClose.State(ctx, month)
		if err != nil {
			return nil, err
		}
		state.EnsureLists()
		return state, nil
	})
}
