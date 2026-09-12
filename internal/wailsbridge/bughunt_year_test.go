package wailsbridge

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/zalando/go-keyring"
)

// Regression coverage for the year selected during setup.
func TestBugHuntCreateTenantRespectsSelectedYear(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { repository.SetActiveVault(nil) })
	base := t.TempDir()
	current := time.Now().Year()
	selected := current - 1
	b := &BuchfinkBridge{
		appCfgRepo:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		currentYear: current,
	}
	_, err := b.CreateTenant("Vorjahrestest", filepath.Join(base, "company"), domain.CompanySettings{
		CompanyName: "Vorjahrestest", FiscalYear: selected,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := b.GetFiscalYear(); got != selected {
		t.Errorf("selected %d at setup, but active year is %d", selected, got)
	}
	years, err := b.GetFiscalYears()
	if err != nil {
		t.Fatal(err)
	}
	if len(years) != 1 || years[0].Year != selected {
		t.Errorf("initial fiscal years do not match setup: %+v", years)
	}
}
