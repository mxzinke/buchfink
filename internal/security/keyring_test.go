// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package security

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestTenantVaultLifecycle(t *testing.T) {
	keyring.MockInit() // use in-memory keychain, no real OS access
	dir := t.TempDir()
	const tenantID = "tenant_123"

	vault, err := CreateTenantVault(dir, tenantID)
	if err != nil {
		t.Fatalf("CreateTenantVault: %v", err)
	}
	ct, err := vault.EncryptString("Sensibler Buchungstext")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	// Re-open transparently via the (mocked) keychain.
	reopened, err := OpenTenantVault(dir, tenantID)
	if err != nil {
		t.Fatalf("OpenTenantVault: %v", err)
	}
	pt, err := reopened.DecryptString(ct)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if pt != "Sensibler Buchungstext" {
		t.Fatalf("round-trip mismatch: %q", pt)
	}
}

func TestOpenTenantVaultMissingSecret(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()

	// Keyfile exists but no keychain secret was ever stored.
	kf, _, err := NewKeyfile("whatever")
	if err != nil {
		t.Fatalf("NewKeyfile: %v", err)
	}
	if err := SaveKeyfile(dir, kf); err != nil {
		t.Fatalf("SaveKeyfile: %v", err)
	}
	if _, err := OpenTenantVault(dir, "ghost_tenant"); err != ErrNoKeyringSecret {
		t.Fatalf("expected ErrNoKeyringSecret, got %v", err)
	}
}
