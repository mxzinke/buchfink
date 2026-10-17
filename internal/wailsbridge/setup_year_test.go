package wailsbridge

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Regression coverage for the year selected during setup.
func TestCreateTenantRespectsSelectedYear(t *testing.T) {
	base := t.TempDir()
	selected := time.Now().Year() - 1
	b := newBridge(t)
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
