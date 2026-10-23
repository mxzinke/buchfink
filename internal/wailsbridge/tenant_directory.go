package wailsbridge

import "github.com/buchfink/buchfink/internal/core"

// ValidateTenantDirectory prüft den Speicherort, ohne Dateien oder Ordner anzulegen.
func (b *BuchfinkBridge) ValidateTenantDirectory(dataDir string) error {
	_, err := core.CheckTenantDirectory(dataDir)
	return err
}
