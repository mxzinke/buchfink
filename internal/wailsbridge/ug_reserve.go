package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

func (b *BuchfinkBridge) BookLegalReserve(year int) (*domain.JournalEntry, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Closing.BookLegalReserve(ctx, year)
	})
}
