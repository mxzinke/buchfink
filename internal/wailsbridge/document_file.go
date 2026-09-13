package wailsbridge

import (
	"fmt"
	"os"
	"path/filepath"
)

type DocumentFileInfo struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	LastModified int64  `json:"lastModified"`
}

// GetDocumentFileInfo liest die Metadaten vor der Ablage einer nativ ausgewählten Datei.
func (b *BuchfinkBridge) GetDocumentFileInfo(path string) (*DocumentFileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("die Datei konnte nicht gelesen werden: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("bitte eine einzelne Datei auswählen")
	}
	if info.Size() == 0 || info.Size() > 20<<20 {
		return nil, fmt.Errorf("bitte eine nicht leere Datei mit bis zu 20 MB auswählen")
	}
	return &DocumentFileInfo{Name: filepath.Base(path), Size: info.Size(), LastModified: info.ModTime().UnixMilli()}, nil
}
