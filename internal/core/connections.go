package core

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// BaseDir ist der Ordner der App-Konfiguration. Daneben liegen Dateien, die
// zum Rechner gehören und nicht zu einem Mandanten, etwa die verbundenen Apps.
func (h *Host) BaseDir() string { return h.opts.BaseDir }

// SetMCPPort speichert den Port des MCP-Servers; 0 setzt die Vorgabe. Nur
// die App ändert ihn.
func (h *Host) SetMCPPort(ctx context.Context, port int) error {
	if err := requireApp(ctx); err != nil {
		return err
	}
	if port != 0 && (port < 1024 || port > 65535) {
		return fmt.Errorf("der Port muss zwischen 1024 und 65535 liegen, nicht %d", port)
	}
	_, err := h.updateConfig(func(cfg *domain.AppConfig) error {
		cfg.MCPPort = port
		return nil
	})
	return err
}
