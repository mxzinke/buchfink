package wailsbridge

import (
	"os"
	"path/filepath"
	"testing"
)

// Das Prüferpaket der Bridge hat die Fassungen der Verfahrensdokumentation:
// die Verdrahtung muss sie an den Export reichen.
func TestBridgeAuditPackageCarriesTheGeneratedProcedureDocumentation(t *testing.T) {
	b := wiredBridge(t)

	result, err := b.GenerateProcedureDocumentation()
	if err != nil {
		t.Fatalf("Verfahrensdokumentation erzeugen: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "pruefer")
	if _, err := b.ExportAuditPackage(2026, dir); err != nil {
		t.Fatalf("Prüferpaket: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "verfahrensdokumentation", result.Document.FileName)); err != nil {
		t.Errorf("die erzeugte Verfahrensdokumentation liegt dem Paket nicht bei: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); err != nil {
		t.Errorf("die Versionshistorie liegt dem Paket nicht bei: %v", err)
	}
}
