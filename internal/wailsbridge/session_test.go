package wailsbridge

import (
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// Was über die Bridge kommt, trägt den Erfassungsweg der App.
func TestBridgeRecordsTheAppChannel(t *testing.T) {
	b := testBridge(t)
	entry, err := b.PostManualEntry(manualEntryRequest(t, b))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if entry.Channel != string(actor.ChannelUI) {
		t.Errorf("die Buchung trägt %q, erwartet ui", entry.Channel)
	}
	logged := auditEntries(t, b, domain.AuditFilter{EntityType: "JOURNAL"})
	if len(logged) == 0 {
		t.Fatal("die Buchung steht nicht im Protokoll")
	}
	for _, e := range logged {
		if e.Channel != string(actor.ChannelUI) {
			t.Errorf("Protokolleintrag %d trägt %q, erwartet ui", e.ID, e.Channel)
		}
	}
}
