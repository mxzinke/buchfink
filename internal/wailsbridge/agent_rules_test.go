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
// gelten die Voreinstellungen ohne weiteren Leseordner; den Datenordner nimmt
// die Bridge nicht als Leseordner an.
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
	if rules.DirectMaxGross != domain.DefaultDirectMaxGross || rules.DirectMinBookings != domain.DefaultDirectMinBookings ||
		rules.ExtraReadFolders == nil || len(rules.ExtraReadFolders) != 0 || rules.Stored.ExtraReadFolders == nil ||
		rules.ExportFolder != filepath.Join(dataDir, policy.ExportDir) {
		t.Errorf("Voreinstellung: %+v", rules)
	}

	for _, folder := range []string{dataDir, filepath.Join(dataDir, policy.InboxDir), filepath.Dir(dataDir)} {
		if _, err := b.SaveAgentRules(domain.AgentSettings{ExtraReadFolders: []string{folder}}); err == nil {
			t.Errorf("Leseordner %s angenommen", folder)
		}
	}

	own := t.TempDir()
	rules, err = b.SaveAgentRules(domain.AgentSettings{DirectMaxGross: 50000, ExtraReadFolders: []string{own}})
	if err != nil {
		t.Fatalf("Regeln speichern: %v", err)
	}
	if rules.DirectMaxGross != 50000 || rules.DirectMinBookings != domain.DefaultDirectMinBookings ||
		len(rules.ExtraReadFolders) != 1 || rules.ExtraReadFolders[0] != own {
		t.Errorf("eigene Regeln: %+v", rules)
	}

	rules, err = b.SaveAgentRules(domain.AgentSettings{})
	if err != nil {
		t.Fatalf("zurücksetzen: %v", err)
	}
	if len(rules.ExtraReadFolders) != 0 || rules.DirectMaxGross != domain.DefaultDirectMaxGross {
		t.Errorf("zurückgesetzt: %+v", rules)
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
