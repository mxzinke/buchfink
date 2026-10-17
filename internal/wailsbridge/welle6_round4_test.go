package wailsbridge

import (
	"fmt"
	"testing"
)

// Die Antworten der Nachweisseite haben leere Listen, keine `null`. Sie werden
// im Regelfall gelesen, bevor irgendetwas erfasst wurde.
func TestNachweiseAnswersCarryEmptyListsNotNull(t *testing.T) {
	b := wiredBridge(t)

	chain, err := b.VerifyAuditChain()
	if err != nil {
		t.Fatalf("Kettenprüfung: %v", err)
	}
	if chain.Breaks == nil {
		t.Error("die Kettenprüfung liefert `null` statt einer leeren Bruchliste")
	}

	for label, load := range map[string]func() (int, error){
		"Schemaänderungen": func() (int, error) {
			rows, err := b.GetSchemaMigrations()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
		"Datenübernahmen": func() (int, error) {
			rows, err := b.GetMigrationRecords()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
		"Fassungen der Verfahrensdokumentation": func() (int, error) {
			rows, err := b.GetProcedureDocumentations()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
	} {
		if _, err := load(); err != nil {
			t.Errorf("%s: %v", label, err)
		}
	}
}
