package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// PostIncomingReceiptWithHeader ergänzt freiwillige Kopfdaten und berichtigt
// gegebenenfalls den Eingangstag in derselben Transaktion wie die Buchung.
func (s *PostingService) PostIncomingReceiptWithHeader(ctx context.Context, req ReceiptRequest, receivedAt string) (*domain.JournalEntry, error) {
	plan, err := s.planIncomingReceipt(ctx, req)
	if err != nil {
		return nil, err
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		if receivedAt != "" {
			if _, err := s.receiptSvc.SaveReceivedAt(ctx, req.ReceiptID, receivedAt); err != nil {
				return nil, err
			}
		}
		if plan.header != nil {
			if _, err := s.receiptSvc.SaveHeader(ctx, req.ReceiptID, *plan.header); err != nil {
				return nil, err
			}
		}
		// Die Buchung verwendet den aktualisierten Beleg einschließlich Hash und Befunden.
		return s.PostIncomingReceipt(ctx, req)
	})
}
