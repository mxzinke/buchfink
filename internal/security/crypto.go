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

// Keyfile is the on-disk envelope: an Argon2id salt plus the DEK wrapped
// (encrypted) under a key derived from the user's passphrase.
type Keyfile struct {
	Version    int          `json:"version"`
	KDF        string       `json:"kdf"` // always "argon2id"
	Params     argon2Params `json:"params"`
	Salt       []byte       `json:"salt"`       // Argon2id salt
	WrappedDEK []byte       `json:"wrappedDek"` // AES-GCM(nonce||ciphertext) of the DEK under the KEK
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

// Rewrap produces a new Keyfile that wraps the same DEK under a new passphrase.
// Existing encrypted data stays valid because the DEK is unchanged — only the
// wrapping key changes.
func (v *Vault) Rewrap(newPassphrase string) (*Keyfile, error) {
	if newPassphrase == "" {
		return nil, errors.New("passphrase must not be empty")
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("salt: %w", err)
	}
	kek := deriveKEK(newPassphrase, salt, defaultArgon2Params)
	kekGCM, err := newGCM(kek)
	if err != nil {
		return nil, err
	}
	wrapped, err := seal(kekGCM, v.dek)
	if err != nil {
		return nil, fmt.Errorf("wrap dek: %w", err)
	}
	return &Keyfile{
		Version:    1,
		KDF:        "argon2id",
		Params:     defaultArgon2Params,
		Salt:       salt,
		WrappedDEK: wrapped,
	}, nil
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
