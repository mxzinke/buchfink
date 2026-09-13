package wailsbridge

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDocumentFileInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "IHK-Schreiben.pdf")
	content := []byte("%PDF-1.4 Testnachweis")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	b := &BuchfinkBridge{}
	info, err := b.GetDocumentFileInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "IHK-Schreiben.pdf" || info.Size != int64(len(content)) || info.LastModified != modified.UnixMilli() {
		t.Fatalf("Dateimetadaten: %+v", info)
	}
	for _, size := range []int64{0, (20 << 20) + 1} {
		if err := os.Truncate(path, size); err != nil {
			t.Fatal(err)
		}
		if _, err := b.GetDocumentFileInfo(path); err == nil {
			t.Fatalf("Dateigröße %d akzeptiert", size)
		}
	}
	for _, invalid := range []string{dir, filepath.Join(dir, "missing.pdf")} {
		if _, err := b.GetDocumentFileInfo(invalid); err == nil {
			t.Fatalf("Ungültige Datei akzeptiert: %s", invalid)
		}
	}
}
