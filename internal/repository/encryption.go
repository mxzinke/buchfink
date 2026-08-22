// SPDX-License-Identifier: EUPL-1.2

package repository

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/buchfink/buchfink/internal/security"
	"gorm.io/gorm/schema"
)

// activeVault holds the unlocked encryption vault for the currently active
// tenant. Buchfink keeps exactly one tenant open at a time, so a package-level
// vault is sufficient and lets the GORM serializer reach it without threading it
// through every call site.
var (
	activeVault *security.Vault
	vaultMu     sync.RWMutex
)

// SetActiveVault installs the vault used by the "encrypted" GORM serializer.
// Pass nil to disable field encryption (values are then stored in clear text —
// used by tests and unencrypted contexts).
func SetActiveVault(v *security.Vault) {
	vaultMu.Lock()
	activeVault = v
	vaultMu.Unlock()
}

func currentVault() *security.Vault {
	vaultMu.RLock()
	defer vaultMu.RUnlock()
	return activeVault
}

// EncryptedSerializer transparently encrypts string fields tagged with
// `serializer:encrypted` using the active tenant vault (AES-256-GCM). When no
// vault is active it passes values through unchanged, so the same schema works
// for encrypted tenants and plain test databases.
//
// Encryption happens strictly at the DB boundary: the in-memory model always
// carries plaintext, so the GoBD hash chain (computed over plaintext) is
// unaffected.
type EncryptedSerializer struct{}

// Value is called when writing to the database: it encrypts the plaintext field.
func (EncryptedSerializer) Value(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue interface{}) (interface{}, error) {
	plaintext, ok := asString(fieldValue)
	if !ok {
		return fieldValue, nil
	}
	v := currentVault()
	if v == nil {
		return plaintext, nil
	}
	return v.EncryptString(plaintext)
}

// Scan is called when reading from the database: it decrypts back to plaintext.
func (EncryptedSerializer) Scan(_ context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	}
	stored, ok := asString(dbValue)
	if !ok {
		return fmt.Errorf("encrypted serializer: unexpected db type %T for field %s", dbValue, field.Name)
	}

	plaintext := stored
	if v := currentVault(); v != nil && stored != "" {
		dec, err := v.DecryptString(stored)
		if err != nil {
			return fmt.Errorf("decrypt field %s: %w", field.Name, err)
		}
		plaintext = dec
	}

	field.ReflectValueOf(context.Background(), dst).SetString(plaintext)
	return nil
}

// asString normalizes the values GORM hands us (string or []byte) to a string.
func asString(v interface{}) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	case []byte:
		return string(s), true
	default:
		return "", false
	}
}

func init() {
	schema.RegisterSerializer("encrypted", EncryptedSerializer{})
}
