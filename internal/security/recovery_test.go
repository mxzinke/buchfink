// SPDX-License-Identifier: EUPL-1.2

package security

import (
	"testing"

	"github.com/zalando/go-keyring"
)

// TestRecoveryFileRoundTrip simulates the real disaster scenario: a tenant is
// created on machine A, a recovery file is exported, then the OS keychain is
// wiped (machine A dies / data moved to machine B). Recovery via the file must
// restore access to data encrypted before the loss.
func TestRecoveryFileRoundTrip(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	const tenantID = "tenant_recover"

	// Machine A: create tenant, encrypt some data.
	vault, err := CreateTenantVault(dir, tenantID)
	if err != nil {
		t.Fatalf("CreateTenantVault: %v", err)
	}
	secret, err := vault.EncryptString("Vertrauliche Buchung")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	// Export a recovery file to external storage.
	rfBytes, err := ExportTenantRecoveryFile(dir, tenantID, "Alpha GmbH", vault)
	if err != nil {
		t.Fatalf("ExportTenantRecoveryFile: %v", err)
	}

	// Disaster: the OS keychain is gone. Transparent unlock must now fail.
	keyring.MockInit() // fresh empty keychain
	if _, err := OpenTenantVault(dir, tenantID); err != ErrNoKeyringSecret {
		t.Fatalf("expected ErrNoKeyringSecret after keychain loss, got %v", err)
	}

	// Recover from the external file.
	recovered, err := RecoverTenantFromFile(dir, tenantID, rfBytes)
	if err != nil {
		t.Fatalf("RecoverTenantFromFile: %v", err)
	}
	pt, err := recovered.DecryptString(secret)
	if err != nil {
		t.Fatalf("DecryptString after recovery: %v", err)
	}
	if pt != "Vertrauliche Buchung" {
		t.Fatalf("recovered data mismatch: %q", pt)
	}

	// After recovery the keychain is re-provisioned: transparent unlock works again.
	reopened, err := OpenTenantVault(dir, tenantID)
	if err != nil {
		t.Fatalf("OpenTenantVault after recovery: %v", err)
	}
	if pt2, _ := reopened.DecryptString(secret); pt2 != "Vertrauliche Buchung" {
		t.Fatalf("transparent unlock broken after recovery: %q", pt2)
	}
}

func TestRecoverWithWrongFile(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	const tenantID = "tenant_x"

	vault, err := CreateTenantVault(dir, tenantID)
	if err != nil {
		t.Fatalf("CreateTenantVault: %v", err)
	}
	if _, err := ExportTenantRecoveryFile(dir, tenantID, "X", vault); err != nil {
		t.Fatalf("export: %v", err)
	}

	// A recovery file with a bogus key must be rejected.
	bogus := []byte(`{"type":"buchfink-recovery-key","version":1,"tenantId":"tenant_x","key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`)
	if _, err := RecoverTenantFromFile(dir, tenantID, bogus); err != ErrBadRecoveryKey {
		t.Fatalf("expected ErrBadRecoveryKey, got %v", err)
	}

	// A non-recovery JSON file must be rejected too.
	if _, err := RecoverTenantFromFile(dir, tenantID, []byte(`{"type":"something-else"}`)); err == nil {
		t.Fatal("expected error for non-recovery file")
	}
}
