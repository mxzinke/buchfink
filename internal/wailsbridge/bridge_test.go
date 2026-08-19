package wailsbridge_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/wailsbridge"
	"github.com/zalando/go-keyring"
)

func TestBridge_MultiTenantManagement(t *testing.T) {
	keyring.MockInit() // in-memory keychain: no real OS keychain access in tests
	tempDir, err := os.MkdirTemp("", "buchfink_bridge_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override HOME to use tempDir for testing
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	bridge, err := wailsbridge.NewBuchfinkBridge()
	if err != nil {
		t.Fatalf("failed to create bridge: %v", err)
	}

	// 1. Initial State: No tenants
	tenants := bridge.GetTenants()
	if len(tenants) != 0 {
		t.Fatalf("expected 0 initial tenants, got %d", len(tenants))
	}

	// 2. Create Tenant 1 (Alpha GmbH)
	tenant1Dir := filepath.Join(tempDir, "alpha_data")
	t1, err := bridge.CreateTenant("Alpha GmbH", tenant1Dir, domain.CompanySettings{
		CompanyName: "Alpha GmbH",
		FiscalYear:  2026,
	})
	if err != nil {
		t.Fatalf("failed to create tenant 1: %v", err)
	}
	if t1.Name != "Alpha GmbH" {
		t.Fatalf("expected tenant name Alpha GmbH, got %s", t1.Name)
	}

	// Add a contact to Tenant 1
	c1, err := bridge.SaveContact(domain.Contact{
		Type:   domain.ContactTypeCustomer,
		Number: "K-100",
		Name:   "Alpha Kunde",
	})
	if err != nil || c1 == nil {
		t.Fatalf("failed to save contact in tenant 1: %v", err)
	}

	// 3. Create Tenant 2 (Beta UG)
	tenant2Dir := filepath.Join(tempDir, "beta_data")
	t2, err := bridge.CreateTenant("Beta UG", tenant2Dir, domain.CompanySettings{
		CompanyName: "Beta UG",
		FiscalYear:  2026,
	})
	if err != nil {
		t.Fatalf("failed to create tenant 2: %v", err)
	}

	// Verify Tenant 2 is isolated from Tenant 1
	contactsT2, err := bridge.GetContacts()
	if err != nil {
		t.Fatalf("failed to get contacts for tenant 2: %v", err)
	}
	if len(contactsT2) != 0 {
		t.Fatalf("expected 0 contacts in tenant 2, got %d", len(contactsT2))
	}

	// Check total tenants
	allTenants := bridge.GetTenants()
	if len(allTenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(allTenants))
	}

	// 4. Switch back to Tenant 1
	if err := bridge.SwitchTenant(t1.ID); err != nil {
		t.Fatalf("failed to switch back to tenant 1: %v", err)
	}

	contactsT1Again, err := bridge.GetContacts()
	if err != nil || len(contactsT1Again) != 1 || contactsT1Again[0].Name != "Alpha Kunde" {
		t.Fatalf("expected 1 contact 'Alpha Kunde' in tenant 1, got %v", contactsT1Again)
	}

	// 5. Delete Tenant 2
	if err := bridge.DeleteTenant(t2.ID); err != nil {
		t.Fatalf("failed to delete tenant 2: %v", err)
	}

	remainingTenants := bridge.GetTenants()
	if len(remainingTenants) != 1 || remainingTenants[0].ID != t1.ID {
		t.Fatalf("expected 1 remaining tenant (t1), got %v", remainingTenants)
	}
}
