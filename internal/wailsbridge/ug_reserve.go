package wailsbridge

import (
	"context"
	"fmt"
	"github.com/buchfink/buchfink/internal/domain"
)

func (b *BuchfinkBridge) BookLegalReserve(year int) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingSvc == nil {
		return nil, fmt.Errorf("Jahresabschluss ist nicht eingerichtet")
	}
	return b.closingSvc.BookLegalReserve(context.Background(), year)
}
