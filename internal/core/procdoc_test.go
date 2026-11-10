package core

import (
	"strings"
	"testing"
)

// Die Verfahrensdokumentation nimmt Datenordner und Sicherungsziel vom
// Mandanten, für den sie entsteht, nicht aus einem Dienstzustand.
func TestProcedureDocumentationDescribesItsTenant(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	backup := t.TempDir()
	if _, err := h.SetBackupDir(uiCtx, backup); err != nil {
		t.Fatal(err)
	}
	tenant := openTenant(t, h)
	result, err := tenant.GenerateProcedureDocumentation(uiCtx, 2026)
	if err != nil {
		t.Fatalf("Verfahrensdokumentation: %v", err)
	}
	if !strings.Contains(result.Markdown, tc.DataDir) || !strings.Contains(result.Markdown, backup) {
		t.Errorf("Datenordner %s oder Sicherungsziel %s fehlen in der Fassung", tc.DataDir, backup)
	}
}
