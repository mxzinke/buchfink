package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/service"
)

// GetClosingSettings liefert Hebesatz, Abgrenzungsmethode, Vorschlagsschwelle
// und Auflösungstakt der Abschlussbausteine.
func (b *BuchfinkBridge) GetClosingSettings() (*service.ClosingSettings, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingSettingsSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSettingsSvc.Get(context.Background())
}

// SaveClosingSettings prüft und speichert die Einstellungen der
// Abschlussbausteine.
func (b *BuchfinkBridge) SaveClosingSettings(in service.ClosingSettings) (*service.ClosingSettings, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingSettingsSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSettingsSvc.Save(context.Background(), in)
}
