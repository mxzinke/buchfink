package wailsbridge

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Regeln für verbundene Apps setzt der Mensch in der App. Ohne Eintrag
// gelten die Voreinstellungen mit dem Eingang als einzigem Belegordner; den
// Datenordner selbst nimmt die Bridge nicht als Belegordner an, und eine leere
// Liste sperrt alle Belegordner.
func TestAgentRulesAreSetInTheApp(t *testing.T) {
	b := testBridge(t)
	var dataDir string
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		dataDir = s.Tenant.DataDir
		return nil
	})

	rules, err := b.GetAgentRules()
	if err != nil {
		t.Fatal(err)
	}
	inbox := filepath.Join(dataDir, policy.InboxDir)
	if !rules.ReceiptFoldersDefault || rules.DirectMaxGross != domain.DefaultDirectMaxGross ||
		rules.DirectMinBookings != domain.DefaultDirectMinBookings ||
		len(rules.ReceiptFolders) != 1 || rules.ReceiptFolders[0] != inbox ||
		rules.ExportFolder != filepath.Join(dataDir, policy.ExportDir) || rules.Stored.ReceiptFolders == nil {
		t.Errorf("Voreinstellung: %+v", rules)
	}

	if _, err := b.SaveAgentRules(domain.AgentSettings{ReceiptFolders: []string{dataDir}}); err == nil {
		t.Error("den Datenordner als Belegordner angenommen")
	}

	own := t.TempDir()
	rules, err = b.SaveAgentRules(domain.AgentSettings{DirectMaxGross: 50000, ReceiptFolders: []string{own, inbox}})
	if err != nil {
		t.Fatalf("Regeln speichern: %v", err)
	}
	if rules.ReceiptFoldersDefault || rules.DirectMaxGross != 50000 ||
		rules.DirectMinBookings != domain.DefaultDirectMinBookings || len(rules.ReceiptFolders) != 2 {
		t.Errorf("eigene Regeln: %+v", rules)
	}

	rules, err = b.SaveAgentRules(domain.AgentSettings{ReceiptFolders: []string{}})
	if err != nil {
		t.Fatalf("Belegordner sperren: %v", err)
	}
	if rules.ReceiptFoldersDefault || rules.ReceiptFolders == nil || len(rules.ReceiptFolders) != 0 ||
		rules.DirectMaxGross != domain.DefaultDirectMaxGross {
		t.Errorf("alle Belegordner gesperrt: %+v", rules)
	}
}

// Die Standard-Buchungsgruppe eines Lieferanten bestätigt der Mensch; nur eine
// Gruppe für Eingangsbelege, und eine leere hebt sie auf.
func TestDefaultPostingGroupIsConfirmedInTheApp(t *testing.T) {
	b := testBridge(t)
	vendor, err := b.SaveContact(domain.Contact{Type: domain.ContactTypeVendor, Name: "Hosting GmbH", CountryCode: "DE"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.ConfirmDefaultPostingGroup(vendor.ID, "erloese"); err == nil {
		t.Error("eine Gruppe für Ausgangsrechnungen bestätigt")
	}
	confirmed, err := b.ConfirmDefaultPostingGroup(vendor.ID, "fremdleistungen")
	if err != nil {
		t.Fatalf("bestätigen: %v", err)
	}
	if confirmed.DefaultPostingGroup != "fremdleistungen" || confirmed.DefaultPostingGroupConfirmedAt == "" {
		t.Errorf("bestätigt: %+v", confirmed)
	}
	cleared, err := b.ConfirmDefaultPostingGroup(vendor.ID, "")
	if err != nil {
		t.Fatalf("aufheben: %v", err)
	}
	if cleared.DefaultPostingGroup != "" || cleared.DefaultPostingGroupConfirmedAt != "" {
		t.Errorf("aufgehoben: %+v", cleared)
	}
}
