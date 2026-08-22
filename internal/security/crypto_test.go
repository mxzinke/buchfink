// SPDX-License-Identifier: EUPL-1.2

package security

import (
	"bytes"
	"testing"
)

func TestNewKeyfileAndUnlockRoundTrip(t *testing.T) {
	kf, vault, err := NewKeyfile("correct horse battery staple")
	if err != nil {
		t.Fatalf("NewKeyfile: %v", err)
	}

	// Encrypt with the freshly created vault.
	ct, err := vault.EncryptString("Rechnung Bürobedarf 2026")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if ct == "" || ct == "Rechnung Bürobedarf 2026" {
		t.Fatal("ciphertext should be non-empty and differ from plaintext")
	}

	// Unlock the same keyfile with the right passphrase and decrypt.
	vault2, err := kf.Unlock("correct horse battery staple")
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	pt, err := vault2.DecryptString(ct)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if pt != "Rechnung Bürobedarf 2026" {
		t.Fatalf("round-trip mismatch: got %q", pt)
	}

	// Both vaults must hold the same DEK.
	if !vault.keyMatches(vault2.dek) {
		t.Fatal("unlocked DEK differs from original")
	}
}

func TestUnlockWrongPassphrase(t *testing.T) {
	kf, _, err := NewKeyfile("s3cret")
	if err != nil {
		t.Fatalf("NewKeyfile: %v", err)
	}
	if _, err := kf.Unlock("wrong"); err != ErrWrongPassphrase {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
}

func TestEmptyStringRoundTrip(t *testing.T) {
	_, vault, err := NewKeyfile("pw")
	if err != nil {
		t.Fatalf("NewKeyfile: %v", err)
	}
	ct, err := vault.EncryptString("")
	if err != nil {
		t.Fatalf("EncryptString empty: %v", err)
	}
	if ct != "" {
		t.Fatalf("empty plaintext should map to empty ciphertext, got %q", ct)
	}
	pt, err := vault.DecryptString("")
	if err != nil {
		t.Fatalf("DecryptString empty: %v", err)
	}
	if pt != "" {
		t.Fatalf("expected empty, got %q", pt)
	}
}

func TestNonceIsRandom(t *testing.T) {
	_, vault, err := NewKeyfile("pw")
	if err != nil {
		t.Fatalf("NewKeyfile: %v", err)
	}
	a, _ := vault.EncryptString("same")
	b, _ := vault.EncryptString("same")
	if a == b {
		t.Fatal("identical plaintext produced identical ciphertext (nonce reuse)")
	}
}

func TestSaveLoadKeyfile(t *testing.T) {
	dir := t.TempDir()
	kf, vault, err := NewKeyfile("pw")
	if err != nil {
		t.Fatalf("NewKeyfile: %v", err)
	}
	if err := SaveKeyfile(dir, kf); err != nil {
		t.Fatalf("SaveKeyfile: %v", err)
	}
	if !KeyfileExists(dir) {
		t.Fatal("KeyfileExists should be true after save")
	}
	loaded, err := LoadKeyfile(dir)
	if err != nil {
		t.Fatalf("LoadKeyfile: %v", err)
	}
	if !bytes.Equal(loaded.Salt, kf.Salt) || !bytes.Equal(loaded.WrappedDEK, kf.WrappedDEK) {
		t.Fatal("loaded keyfile differs from saved")
	}
	v2, err := loaded.Unlock("pw")
	if err != nil {
		t.Fatalf("Unlock loaded: %v", err)
	}
	if !vault.keyMatches(v2.dek) {
		t.Fatal("DEK mismatch after save/load/unlock")
	}
}
