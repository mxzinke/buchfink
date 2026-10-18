package service

import (
	"context"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AgeOpenItems berechnet die Altersstruktur einer Auswahl offener Posten.
func (s *PaymentService) AgeOpenItems(items []domain.OpenItem, cutoff string) *domain.OpenItemsAging {
	aging := accounting.AgeOpenItems(items, cutoff)
	aging.EnsureLists()
	return &aging
}

// OpenItemsAging liefert die Altersstruktur zum Stichtag; leer heißt heute.
func (s *PaymentService) OpenItemsAging(ctx context.Context, cutoff string) (*domain.OpenItemsAging, error) {
	if cutoff == "" {
		cutoff = time.Now().UTC().Format("2006-01-02")
	}
	items, err := s.OpenItemsAt(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	return s.AgeOpenItems(items, cutoff), nil
}
