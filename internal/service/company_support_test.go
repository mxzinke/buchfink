package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Für eine gemeinnützige, in Liquidation befindliche oder als Organgesellschaft
// geführte GmbH weist Buchfink Abschluss und Steuerrückstellung ab — mit dem
// Grund und dem Hinweis auf den Steuerberater.
func TestUnsupportedCompanyRefusesClosingAndTaxProvision(t *testing.T) {
	cases := map[string]func(*domain.CompanySettings){
		"gemeinnützig":      func(c *domain.CompanySettings) { c.NonProfit = true },
		"Liquidation":       func(c *domain.CompanySettings) { c.InLiquidation = true },
		"Organgesellschaft": func(c *domain.CompanySettings) { c.FiscalUnity = true },
	}
	for word, mark := range cases {
		env := newTestEnv(t)
		ctx := context.Background()
		modules := env.closingModules(t)
		if err := modules.closing.EnsureFiscalYears(ctx, 2026); err != nil {
			t.Fatalf("Geschäftsjahre: %v", err)
		}
		if _, err := modules.bookings.PreviewTaxProvision(ctx, 2026); err != nil {
			t.Fatalf("%s: ohne Merkmal rechnet die Steuerrückstellung: %v", word, err)
		}
		env.setSettings(t, mark)

		_, err := modules.bookings.PreviewTaxProvision(ctx, 2026)
		if err == nil || !strings.Contains(err.Error(), word) || !strings.Contains(err.Error(), "Steuerberater") {
			t.Errorf("%s: Steuerrückstellung: %v", word, err)
		}
		_, err = modules.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-01", "")
		if err == nil || !strings.Contains(err.Error(), word) {
			t.Errorf("%s: Aufstellung: %v", word, err)
		}
	}
}
