package domain

import (
	"testing"
)

// Die Klärungsliste liefert leere Listen und nicht nil: die Oberfläche läuft
// über sie.
func TestReceiptFindingsEnsureLists(t *testing.T) {
	f := ReceiptFindings{}
	f.EnsureLists()
	if f.Groups == nil {
		t.Fatal("die Gruppen dürfen nicht nil sein")
	}
	f = ReceiptFindings{Groups: []ValidationFindingGroup{{Class: ValidationClassFormat}}}
	f.EnsureLists()
	if f.Groups[0].Findings == nil {
		t.Error("auch die Befunde einer Gruppe sind eine leere Liste")
	}
	for _, class := range AllValidationFindingClasses() {
		if class.Label() == "" {
			t.Errorf("die Klasse %q hat keinen Klartext", class)
		}
	}
}
