package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// GetClosingSettings liefert Hebesatz, Abgrenzungsmethode, Vorschlagsschwelle
// und Auflösungstakt der Abschlussbausteine.
func (b *BuchfinkBridge) GetClosingSettings() (*service.ClosingSettings, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.ClosingSettings, error) {
		return s.ClosingSettings.Get(ctx)
	})
}

// SaveClosingSettings prüft und speichert die Einstellungen der
// Abschlussbausteine.
func (b *BuchfinkBridge) SaveClosingSettings(in service.ClosingSettings) (*service.ClosingSettings, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*service.ClosingSettings, error) {
		return s.ClosingSettings.Save(ctx, in)
	})
}
