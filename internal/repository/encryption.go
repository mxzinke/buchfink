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

// vaultOverrideKey adressiert den an einen Kontext gebundenen Schlüsselbund.
type vaultOverrideKey struct{}

// vaultOverride ist die Hülle um den gebundenen Schlüsselbund. Sie wird
// gebraucht, damit sich „kein Schlüsselbund gebunden" von „ausdrücklich ohne
// Schlüsselbund lesen" unterscheiden lässt: das zweite ist der Klartextfall
// eines Mandanten ohne Keyfile und darf nicht auf den prozessweiten Schlüssel
// des gerade offenen Mandanten zurückfallen.
type vaultOverride struct{ vault *security.Vault }

// WithVault bindet einen Schlüsselbund an einen Kontext. Alle Lese- und
// Schreibvorgänge, die mit diesem Kontext laufen, benutzen ihn statt des
// prozessweiten.
//
// Das ist der Weg, auf dem die Sicherungsprüfung eine fremde Datenbank liest:
// sie öffnet den Schlüsselbund aus der Sicherung selbst. Ohne ihn entschlüsselte
// sie mit dem Schlüssel des gerade offenen Mandanten, scheiterte und meldete
// eine heile Sicherung als beschädigt. Ein prozessweiter Wechsel wäre keine
// Lösung: er zöge jeden gleichzeitigen Lesevorgang mit.
func WithVault(ctx context.Context, vault *security.Vault) context.Context {
	return context.WithValue(ctx, vaultOverrideKey{}, vaultOverride{vault: vault})
}

// vaultFor liefert den Schlüsselbund, mit dem dieser Vorgang arbeitet.
func vaultFor(ctx context.Context) *security.Vault {
	if ctx != nil {
		if override, ok := ctx.Value(vaultOverrideKey{}).(vaultOverride); ok {
			return override.vault
		}
	}
	return currentVault()
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
func (EncryptedSerializer) Value(ctx context.Context, _ *schema.Field, _ reflect.Value, fieldValue interface{}) (interface{}, error) {
	plaintext, ok := asString(fieldValue)
	if !ok {
		return fieldValue, nil
	}
	v := vaultFor(ctx)
	if v == nil {
		return plaintext, nil
	}
	return v.EncryptString(plaintext)
}

// Scan is called when reading from the database: it decrypts back to plaintext.
func (EncryptedSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	}
	stored, ok := asString(dbValue)
	if !ok {
		return fmt.Errorf("encrypted serializer: unexpected db type %T for field %s", dbValue, field.Name)
	}

	plaintext := stored
	if v := vaultFor(ctx); v != nil && stored != "" {
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

// VaultForTest legt offen, welchen Schlüsselbund ein Kontext bindet.
//
// Nur für Tests: sie sollen prüfen können, dass eine Sicherungsprüfung
// ausdrücklich ohne Schlüssel liest, und nicht bloß, dass sie nicht abstürzt.
func VaultForTest(ctx context.Context) *security.Vault { return vaultFor(ctx) }
