package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// GetTakeover liefert den festgelegten Umstieg aus einem Vorsystem und die
// Abstimmung der Eröffnung mit dessen Schlussbilanz.
func (b *BuchfinkBridge) GetTakeover() (*service.TakeoverReconciliation, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.TakeoverReconciliation, error) {
		return s.Takeover.Reconciliation(ctx)
	})
}

// SetTakeoverDate legt fest, dass das Unternehmen mit bestehenden Büchern
// beginnt und zu welchem Tag. Ein leerer Tag nimmt den Umstieg zurück.
func (b *BuchfinkBridge) SetTakeoverDate(date string) (*service.TakeoverState, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*service.TakeoverState, error) {
		return s.Takeover.SetDate(ctx, date)
	}, core.AppOnly())
}

// PreviewTakeover rechnet die Eröffnung aus der Schlussbilanz des Vorsystems,
// ohne zu buchen.
func (b *BuchfinkBridge) PreviewTakeover(req service.TakeoverRequest) (*service.TakeoverPreview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.TakeoverPreview, error) {
		return s.Takeover.Preview(ctx, req)
	})
}

// BookTakeover bucht die Eröffnung mit offenen Posten und Anlagen.
func (b *BuchfinkBridge) BookTakeover(req service.TakeoverRequest) (*service.TakeoverPreview, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*service.TakeoverPreview, error) {
		return s.Takeover.Book(ctx, req)
	})
}
