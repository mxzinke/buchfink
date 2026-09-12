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

func TestOnboardingRecoveryExportSurvivesLostKeychainAndFailedReexport(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { repository.SetActiveVault(nil) })
	base := t.TempDir()
	b := &BuchfinkBridge{appCfgRepo: repository.NewAppConfigRepository(filepath.Join(base, "config")), currentYear: 2026}
	tenant, err := b.CreateTenant("Schlüsseltest", filepath.Join(base, "company"), domain.CompanySettings{FiscalYear: 2025})
	if err != nil {
		t.Fatal(err)
	}
	defer repository.CloseDB(b.db)
	if b.activeTenantLocked().RecoveryExportedAt != "" {
		t.Fatal("new tenant incorrectly marked as backed up")
	}
	keyBefore, err := os.ReadFile(filepath.Join(tenant.DataDir, security.KeyfileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.ExportRecoveryKeyToDirectory(tenant.DataDir); err == nil {
		t.Fatal("key backup allowed inside accounting directory")
	}
	keyAfter, _ := os.ReadFile(filepath.Join(tenant.DataDir, security.KeyfileName))
	if !bytes.Equal(keyBefore, keyAfter) {
		t.Fatal("rejected export modified the keyfile")
	}
	external := t.TempDir()
	path, err := b.ExportRecoveryKeyToDirectory(external)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("recovery file mode: %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stamp := b.activeTenantLocked().RecoveryExportedAt
	if stamp == "" {
		t.Fatal("successful export not recorded")
	}
	secret, err := b.vault.EncryptString("Buchhaltung nach Schlüsselverlust")
	if err != nil {
		t.Fatal(err)
	}
	// Force final rename to fail after recovery material has been prepared.
	blocked := t.TempDir()
	if err := os.Mkdir(filepath.Join(blocked, filepath.Base(path)), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ExportRecoveryKeyToDirectory(blocked); err == nil {
		t.Fatal("write failure was reported as success")
	}
	if b.activeTenantLocked().RecoveryExportedAt != stamp {
		t.Fatal("failed export changed backup timestamp")
	}
	keyring.MockInit()
	if _, err := security.OpenTenantVault(tenant.DataDir, tenant.VaultID()); err == nil {
		t.Fatal("unexpected keychain access after simulated loss")
	}
	vault, err := security.RecoverTenantFromFile(tenant.DataDir, tenant.VaultID(), data)
	if err != nil {
		t.Fatalf("previous recovery file invalidated by failed re-export: %v", err)
	}
	plain, err := vault.DecryptString(secret)
	if err != nil || plain != "Buchhaltung nach Schlüsselverlust" {
		t.Fatalf("recovery failed: %v", err)
	}
}

func TestCreateTenantRollsBackWhenConfigurationCannotBeSaved(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { repository.SetActiveVault(nil) })
	base := t.TempDir()
	blocked := filepath.Join(base, "blocked")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	b := &BuchfinkBridge{appCfgRepo: repository.NewAppConfigRepository(filepath.Join(blocked, "config")), currentYear: 2026}
	dir := filepath.Join(base, "company")
	if _, err := b.CreateTenant("Fehlerfall", dir, domain.CompanySettings{FiscalYear: 2025}); err == nil {
		t.Fatal("creation succeeded without saving configuration")
	}
	if b.appConfig.IsConfigured || len(b.appConfig.Tenants) != 0 || b.currentYear != 2026 {
		t.Fatalf("partial tenant remains active: %+v", b.appConfig)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("partial data directory remains: %v", err)
	}
	if data, err := os.ReadFile(blocked); err != nil || string(data) != "keep" {
		t.Fatal("existing file was changed")
	}
}
