// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/zalando/go-keyring"
)

// keyringService is the service name under which Buchfink stores its per-tenant
// wrapping secrets in the OS keychain (macOS Keychain / Windows Credential
// Manager / Linux Secret Service).
const keyringService = "org.buchfink.app"

// ErrNoKeyringSecret indicates that no wrapping secret exists in the OS keychain
// for the given tenant — e.g. the data was moved to another machine or the
// keychain entry was removed. Recovery then requires a manual passphrase.
var ErrNoKeyringSecret = errors.New("no keychain secret for tenant")

// generateSecret returns a base64-encoded 32-byte random secret used to wrap a
// tenant's DEK. It is never shown to the user; the OS keychain holds it.
func generateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// storeSecret persists the wrapping secret for a tenant in the OS keychain.
func storeSecret(tenantID, secret string) error {
	if err := keyring.Set(keyringService, tenantID, secret); err != nil {
		return fmt.Errorf("store keychain secret: %w", err)
	}
	return nil
}

// loadSecret retrieves the wrapping secret for a tenant from the OS keychain.
func loadSecret(tenantID string) (string, error) {
	secret, err := keyring.Get(keyringService, tenantID)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoKeyringSecret
		}
		return "", fmt.Errorf("load keychain secret: %w", err)
	}
	return secret, nil
}

// deleteSecret removes a tenant's wrapping secret from the OS keychain. Missing
// entries are not treated as an error.
func deleteSecret(tenantID string) error {
	if err := keyring.Delete(keyringService, tenantID); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("delete keychain secret: %w", err)
	}
	return nil
}

// DeleteTenantSecret removes a tenant's wrapping secret from the OS keychain
// (e.g. when a tenant is removed from the app). Missing entries are not an error.
func DeleteTenantSecret(tenantID string) error {
	return deleteSecret(tenantID)
}

// CreateTenantVault provisions encryption for a new tenant: it generates a random
// wrapping secret, stores it in the OS keychain, builds the envelope keyfile in
// dataDir, and returns an unlocked Vault. Unlock is transparent afterwards.
func CreateTenantVault(dataDir, tenantID string) (*Vault, error) {
	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}
	kf, vault, err := NewKeyfile(secret)
	if err != nil {
		return nil, err
	}
	if err := SaveKeyfile(dataDir, kf); err != nil {
		return nil, err
	}
	if err := storeSecret(tenantID, secret); err != nil {
		return nil, err
	}
	return vault, nil
}

// OpenTenantVault transparently unlocks an existing tenant by reading its
// wrapping secret from the OS keychain and unwrapping the DEK. It returns
// ErrNoKeyringSecret when the keychain entry is missing (e.g. data moved to a new
// machine), in which case a manual recovery passphrase is required.
func OpenTenantVault(dataDir, tenantID string) (*Vault, error) {
	kf, err := LoadKeyfile(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load keyfile: %w", err)
	}
	secret, err := loadSecret(tenantID)
	if err != nil {
		return nil, err
	}
	return kf.Unlock(secret)
}
