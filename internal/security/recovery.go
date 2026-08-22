// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// recoveryFileType is the magic marker identifying a Buchfink recovery file.
const recoveryFileType = "buchfink-recovery-key"

// RecoveryFile is the portable artifact the user stores externally (USB stick,
// cloud, safe). It carries the random recovery key that unlocks the keyfile's
// recovery slot. Whoever holds this file AND the tenant's data (keyfile) can
// decrypt the accounting data — so it must be kept separate and safe.
type RecoveryFile struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	TenantID   string `json:"tenantId"`
	TenantName string `json:"tenantName"`
	CreatedAt  string `json:"createdAt"`
	Key        []byte `json:"key"` // random 256-bit recovery key (base64 in JSON)
}

// ExportTenantRecoveryFile adds (or refreshes) the recovery slot in the tenant's
// keyfile and returns the recovery file bytes to be written to external storage.
// The vault must be unlocked (it holds the DEK being wrapped).
func ExportTenantRecoveryFile(dataDir, tenantID, tenantName string, vault *Vault) ([]byte, error) {
	if vault == nil {
		return nil, errors.New("vault must be unlocked to export a recovery key")
	}
	kf, err := LoadKeyfile(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load keyfile: %w", err)
	}
	recoveryKey, err := vault.addRecoverySlot(kf)
	if err != nil {
		return nil, err
	}
	if err := SaveKeyfile(dataDir, kf); err != nil {
		return nil, err
	}
	rf := RecoveryFile{
		Type:       recoveryFileType,
		Version:    1,
		TenantID:   tenantID,
		TenantName: tenantName,
		CreatedAt:  time.Now().Format(time.RFC3339),
		Key:        recoveryKey,
	}
	return json.MarshalIndent(rf, "", "  ")
}

// RecoverTenantFromFile unlocks a tenant using an exported recovery file when the
// OS keychain secret is unavailable (e.g. on a new machine). On success it also
// re-provisions a fresh keychain secret so subsequent launches are transparent
// again, and returns the unlocked vault.
func RecoverTenantFromFile(dataDir, tenantID string, recoveryFileBytes []byte) (*Vault, error) {
	var rf RecoveryFile
	if err := json.Unmarshal(recoveryFileBytes, &rf); err != nil {
		return nil, fmt.Errorf("parse recovery file: %w", err)
	}
	if rf.Type != recoveryFileType {
		return nil, errors.New("not a Buchfink recovery file")
	}

	kf, err := LoadKeyfile(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load keyfile: %w", err)
	}
	vault, err := openRecoverySlot(kf, rf.Key)
	if err != nil {
		return nil, err
	}

	// Restore transparent access on this machine: mint a new keychain secret,
	// rewrap the primary slot, persist, and store the secret in the keychain.
	newSecret, err := generateSecret()
	if err != nil {
		return nil, err
	}
	if err := vault.rewrapKeychainSlot(kf, newSecret); err != nil {
		return nil, err
	}
	if err := SaveKeyfile(dataDir, kf); err != nil {
		return nil, err
	}
	if err := storeSecret(tenantID, newSecret); err != nil {
		return nil, err
	}
	return vault, nil
}
