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

func TestValidateTenantDirectory(t *testing.T) {
	b := &BuchfinkBridge{}
	empty := t.TempDir()
	missing := filepath.Join(empty, "noch", "nicht", "angelegt")
	for _, path := range []string{empty, missing} {
		if err := b.ValidateTenantDirectory(path); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("Vorprüfung hat den Ordner angelegt: %v", err)
	}
	file := filepath.Join(empty, "beleg.pdf")
	if err := os.WriteFile(file, []byte("Beleg"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"", "   ", empty, file} {
		if err := b.ValidateTenantDirectory(path); err == nil {
			t.Errorf("Ungültiger Ordner akzeptiert: %q", path)
		}
	}
}

func TestCreateTenantRechecksPreviouslyValidDirectory(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	b := &BuchfinkBridge{}
	if err := b.ValidateTenantDirectory(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "neue-datei.txt")
	if err := os.WriteFile(path, []byte("Nach der Vorprüfung erstellt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := b.SetupApplication(dir, domain.CompanySettings{CompanyName: "Neue Firma", FiscalYear: 2026}); err == nil {
		t.Fatal("Abschluss akzeptiert einen inzwischen belegten Ordner")
	}
	if security.KeyfileExists(dir) {
		t.Error("Schlüsseldatei trotz belegtem Ordner angelegt")
	}
	if len(b.GetTenants()) != 0 {
		t.Error("Mandant trotz belegtem Ordner angelegt")
	}
}

func TestTenantDirectoryExpandsHomePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	suffix := filepath.Base(t.TempDir()) + "-buchfink-directory-check"
	got, err := checkedTenantDirectory("~/" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(homeDir, suffix); got != want {
		t.Fatalf("Pfad = %q, erwartet %q", got, want)
	}
}
