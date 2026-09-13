package wailsbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateTenantDirectory prüft den Speicherort, ohne Dateien oder Ordner anzulegen.
func (b *BuchfinkBridge) ValidateTenantDirectory(dataDir string) error {
	_, err := checkedTenantDirectory(dataDir)
	return err
}

func checkedTenantDirectory(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", fmt.Errorf("bitte einen Ordner für die Buchungsdaten und Belege wählen")
	}
	if dataDir == "~" || strings.HasPrefix(dataDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("das Benutzerverzeichnis kann nicht ermittelt werden: %w", err)
		}
		dataDir = filepath.Join(homeDir, strings.TrimPrefix(dataDir, "~"))
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("der Datenordner kann nicht aufgelöst werden: %w", err)
	}
	entries, err := os.ReadDir(absolute)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("der Datenordner kann nicht gelesen werden: %w", err)
	}
	if len(entries) != 0 {
		return "", fmt.Errorf("der Datenordner ist nicht leer. Wählen Sie einen neuen, leeren Ordner oder öffnen Sie die vorhandene Buchhaltung")
	}
	return absolute, nil
}
