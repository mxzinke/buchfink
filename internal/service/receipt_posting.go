package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// ReceiptDisplayError bezeichnet einen offenen Beleg ohne lesbare Darstellung.
type ReceiptDisplayError struct{ ReceiptNumber string }

func (e *ReceiptDisplayError) Error() string {
	return fmt.Sprintf("Beleg %s hat keine ansehbare Darstellung; eine reine XRechnung braucht vor dem Buchen ein PDF.", e.ReceiptNumber)
}

// GetForPosting lädt den Beleg und prüft die Darstellung vor der Buchungsplanung.
func (s *ReceiptService) GetForPosting(ctx context.Context, id uint) (*domain.Receipt, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.IsOpen() && !r.IsDisplayable() {
		return nil, &ReceiptDisplayError{ReceiptNumber: r.ReceiptNumber}
	}
	return r, nil
}
