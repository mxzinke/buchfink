package wailsbridge

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/security"
	"github.com/zalando/go-keyring"
)

func TestCreateTenantPreservesExistingCompany(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { repository.SetActiveVault(nil) })
	base := t.TempDir()
	b := &BuchfinkBridge{appCfgRepo: repository.NewAppConfigRepository(filepath.Join(base, "config")), currentYear: 2026}
	dir := filepath.Join(base, "company")
	first, err := b.CreateTenant("Erste Firma", dir, domain.CompanySettings{CompanyName: "Erste Firma", FiscalYear: 2026})
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(dir, security.KeyfileName)
	before, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	_, createErr := b.CreateTenant("Zweite Firma", dir, domain.CompanySettings{CompanyName: "Zweite Firma", FiscalYear: 2026})
	after, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("a second company overwrote the original encryption keyfile")
	}
	if createErr == nil {
		t.Error("an occupied company directory must be rejected")
	}
	if _, err := security.OpenTenantVault(dir, first.ID); err != nil {
		t.Errorf("original company can no longer be unlocked: %v", err)
	}
	if len(b.GetTenants()) != 1 {
		t.Error("failed creation changed the tenant list")
	}
	if b.GetAppConfig().ActiveTenantID != first.ID {
		t.Error("failed creation changed the active tenant")
	}
}

func TestCreateTenantRejectsNonemptyDirectoryBeforeProvisioning(t *testing.T) {
	keyring.MockInit()
	base := t.TempDir()
	dir := filepath.Join(base, "existing")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "buchfink.sqlite"), []byte("existing database"), 0600); err != nil {
		t.Fatal(err)
	}
	b := &BuchfinkBridge{appCfgRepo: repository.NewAppConfigRepository(filepath.Join(base, "config")), currentYear: 2026}
	if _, err := b.CreateTenant("Neue Firma", dir, domain.CompanySettings{FiscalYear: 2026}); err == nil {
		t.Fatal("nonempty directory accepted")
	}
	if security.KeyfileExists(dir) {
		t.Error("a keyfile was created before rejecting existing data")
	}
}
