package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die kanonischen Formen sind eingefroren.
//
// Ein Prüfer rechnet die Hashes aus dem Export nach der Feldbeschreibung
// (internal/export/fielddoc.go) nach. Ändert sich die Form, ohne dass
// Feldbeschreibung und Export mitgehen, bricht jede Kette, die nach der
// Beschreibung geprüft wird. Wer diesen Test anpasst, passt beide mit an.
func TestCanonicalFormsAreFrozen(t *testing.T) {
	const wantEntry = "5615f6b4b42d7abd1f7abe5fbf0f6a6041fe8e5a9c7881d6d75cb1fe9c479600"
	const wantAudit = "07c916ec65e5407c796509cd718f8d7668a66ceaa871ff0bc87f0a4b83b7525a"

	if got := NewHashChain().CalculateHash(sampleEntry(), domain.GenesisHash); got != wantEntry {
		t.Errorf("Buchung: Hash %s, eingefroren %s", got, wantEntry)
	}
	entry := auditEntry(1, domain.AuditActionUpdate, "geändert")
	if got := NewAuditChain().CalculateHash(&entry, domain.GenesisHash); got != wantAudit {
		t.Errorf("Protokolleintrag: Hash %s, eingefroren %s", got, wantAudit)
	}
}
