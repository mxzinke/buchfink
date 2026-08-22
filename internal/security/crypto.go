// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

// KeyfileName is the fixed filename of the envelope keyfile stored next to a
// tenant's data. It contains only the KDF salt and the passphrase-wrapped data
// encryption key — never the passphrase or the raw key itself.
const KeyfileName = "buchfink.keyfile.json"

// dekSize is the length in bytes of the AES-256 data encryption key (DEK).
const dekSize = 32

// saltSize is the length in bytes of the Argon2id salt.
const saltSize = 16

// argon2Params captures the Argon2id cost parameters. They are persisted in the
// keyfile so future parameter changes stay backward compatible.
type argon2Params struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"` // in KiB
	Threads uint8  `json:"threads"`
}

// defaultArgon2Params are sensible interactive-login defaults (~64 MiB).
var defaultArgon2Params = argon2Params{
	Time:    1,
	Memory:  64 * 1024,
	Threads: 4,
}

// Keyfile is the on-disk envelope. It holds the DEK wrapped in one or more
// independent key slots (LUKS-style): the primary slot is unlocked by the random
// secret in the OS keychain; the optional recovery slot is unlocked by a random
// recovery key the user exports and stores externally. Every slot wraps the SAME
// DEK, so any one of them can unlock the data.
type Keyfile struct {
	Version    int           `json:"version"`
	KDF        string        `json:"kdf"` // always "argon2id" (primary/keychain slot)
	Params     argon2Params  `json:"params"`
	Salt       []byte        `json:"salt"`       // Argon2id salt (primary slot)
	WrappedDEK []byte        `json:"wrappedDek"` // DEK wrapped under the keychain-derived KEK
	Recovery   *recoverySlot `json:"recovery,omitempty"`
}

// recoverySlot wraps the DEK under a random 256-bit recovery key (no passphrase
// KDF needed — the key is already high entropy). The recovery key itself lives
// only in the exported recovery file, never on disk next to the data.
type recoverySlot struct {
	WrappedDEK []byte `json:"wrappedDek"` // AES-GCM(nonce||ciphertext) of the DEK under the recovery key
}

// Vault holds an unlocked data encryption key in memory and provides
// authenticated encryption for field- and file-level payloads. The DEK is never
// written to disk in the clear.
type Vault struct {
	dek []byte
	gcm cipher.AEAD
}

// ErrWrongPassphrase is returned when unwrapping the DEK fails authentication,
// which almost always means the supplied passphrase was incorrect.
var ErrWrongPassphrase = errors.New("wrong passphrase or corrupted keyfile")

// ErrBadRecoveryKey is returned when a recovery file does not match the keyfile's
// recovery slot (wrong file, tampered, or no recovery slot present).
var ErrBadRecoveryKey = errors.New("recovery key does not match this keyfile")

// recoveryKeySize is the length of a random recovery key (AES-256 directly).
const recoveryKeySize = 32

// deriveKEK computes the key-encryption key from a passphrase and salt.
func deriveKEK(passphrase string, salt []byte, p argon2Params) []byte {
	return argon2.IDKey([]byte(passphrase), salt, p.Time, p.Memory, p.Threads, dekSize)
}

// newGCM builds an AES-256-GCM AEAD from a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return gcm, nil
}

// seal encrypts plaintext with the given AEAD, prepending a fresh random nonce.
func seal(gcm cipher.AEAD, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// open reverses seal: it splits the nonce prefix and decrypts.
func open(gcm cipher.AEAD, blob []byte) ([]byte, error) {
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

// NewKeyfile creates a fresh envelope for the given passphrase: it generates a
// random DEK, derives a KEK from the passphrase, wraps the DEK, and returns both
// the persistable Keyfile and an unlocked Vault ready for use.
func NewKeyfile(passphrase string) (*Keyfile, *Vault, error) {
	if passphrase == "" {
		return nil, nil, errors.New("passphrase must not be empty")
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, fmt.Errorf("salt: %w", err)
	}

	dek := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, nil, fmt.Errorf("dek: %w", err)
	}

	kek := deriveKEK(passphrase, salt, defaultArgon2Params)
	kekGCM, err := newGCM(kek)
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := seal(kekGCM, dek)
	if err != nil {
		return nil, nil, fmt.Errorf("wrap dek: %w", err)
	}

	kf := &Keyfile{
		Version:    1,
		KDF:        "argon2id",
		Params:     defaultArgon2Params,
		Salt:       salt,
		WrappedDEK: wrapped,
	}

	vault, err := newVault(dek)
	if err != nil {
		return nil, nil, err
	}
	return kf, vault, nil
}

// Unlock re-derives the KEK from the passphrase and unwraps the DEK, returning a
// usable Vault. A wrong passphrase yields ErrWrongPassphrase.
func (kf *Keyfile) Unlock(passphrase string) (*Vault, error) {
	if kf.KDF != "argon2id" {
		return nil, fmt.Errorf("unsupported kdf %q", kf.KDF)
	}
	kek := deriveKEK(passphrase, kf.Salt, kf.Params)
	kekGCM, err := newGCM(kek)
	if err != nil {
		return nil, err
	}
	dek, err := open(kekGCM, kf.WrappedDEK)
	if err != nil {
		return nil, ErrWrongPassphrase
	}
	return newVault(dek)
}

func newVault(dek []byte) (*Vault, error) {
	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}
	return &Vault{dek: dek, gcm: gcm}, nil
}

// Encrypt seals arbitrary bytes (e.g. a receipt file) under the DEK.
func (v *Vault) Encrypt(plaintext []byte) ([]byte, error) {
	return seal(v.gcm, plaintext)
}

// Decrypt reverses Encrypt.
func (v *Vault) Decrypt(blob []byte) ([]byte, error) {
	return open(v.gcm, blob)
}

// EncryptString seals a string and returns a base64 string suitable for storing
// in a text column. Empty input maps to empty output so NULL/"" round-trips.
func (v *Vault) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	blob, err := v.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(blob), nil
}

// DecryptString reverses EncryptString.
func (v *Vault) DecryptString(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	blob, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	plaintext, err := v.Decrypt(blob)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// keyMatches is a constant-time comparison helper (used in tests / rotation checks).
func (v *Vault) keyMatches(other []byte) bool {
	return subtle.ConstantTimeCompare(v.dek, other) == 1
}

// rewrapKeychainSlot rebuilds the primary (keychain) slot so it unwraps under
// the given random secret, deriving a fresh salt. Any recovery slot is left
// untouched.
func (v *Vault) rewrapKeychainSlot(kf *Keyfile, secret string) error {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("salt: %w", err)
	}
	kekGCM, err := newGCM(deriveKEK(secret, salt, defaultArgon2Params))
	if err != nil {
		return err
	}
	wrapped, err := seal(kekGCM, v.dek)
	if err != nil {
		return fmt.Errorf("wrap dek: %w", err)
	}
	kf.Version = 1
	kf.KDF = "argon2id"
	kf.Params = defaultArgon2Params
	kf.Salt = salt
	kf.WrappedDEK = wrapped
	return nil
}

// addRecoverySlot generates a random recovery key, wraps the DEK under it, and
// stores the wrap in kf.Recovery. The returned key is what the user exports.
func (v *Vault) addRecoverySlot(kf *Keyfile) ([]byte, error) {
	recoveryKey := make([]byte, recoveryKeySize)
	if _, err := io.ReadFull(rand.Reader, recoveryKey); err != nil {
		return nil, fmt.Errorf("recovery key: %w", err)
	}
	gcm, err := newGCM(recoveryKey)
	if err != nil {
		return nil, err
	}
	wrapped, err := seal(gcm, v.dek)
	if err != nil {
		return nil, fmt.Errorf("wrap dek: %w", err)
	}
	kf.Recovery = &recoverySlot{WrappedDEK: wrapped}
	return recoveryKey, nil
}

// openRecoverySlot unwraps the DEK from the keyfile's recovery slot using a
// recovery key, returning an unlocked Vault.
func openRecoverySlot(kf *Keyfile, recoveryKey []byte) (*Vault, error) {
	if kf.Recovery == nil {
		return nil, ErrBadRecoveryKey
	}
	gcm, err := newGCM(recoveryKey)
	if err != nil {
		return nil, ErrBadRecoveryKey
	}
	dek, err := open(gcm, kf.Recovery.WrappedDEK)
	if err != nil {
		return nil, ErrBadRecoveryKey
	}
	return newVault(dek)
}

// SaveKeyfile writes the keyfile as JSON with 0600 permissions.
func SaveKeyfile(dataDir string, kf *Keyfile) error {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(kf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal keyfile: %w", err)
	}
	path := filepath.Join(dataDir, KeyfileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write keyfile: %w", err)
	}
	return nil
}

// LoadKeyfile reads and parses the keyfile from a data directory.
func LoadKeyfile(dataDir string) (*Keyfile, error) {
	path := filepath.Join(dataDir, KeyfileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var kf Keyfile
	if err := json.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("parse keyfile: %w", err)
	}
	return &kf, nil
}

// KeyfileExists reports whether a tenant data directory already has a keyfile.
func KeyfileExists(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, KeyfileName))
	return err == nil
}
