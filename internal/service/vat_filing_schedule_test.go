package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestVatFilingScheduleWithoutRegularReturns(t *testing.T) {
	for _, schedule := range []string{"unknown", "none"} {
		t.Run(schedule, func(t *testing.T) {
			env := newTestEnv(t)
			ctx := context.Background()
			env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = schedule })
			cfg, err := repository.NewSettingsRepository(env.db).GetCompanySettings(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.VatPeriod != schedule {
				t.Fatalf("Meldeplan = %q", cfg.VatPeriod)
			}
			vat := env.vatReturns(t)
			if got := vat.PeriodType(ctx); string(got) != schedule {
				t.Fatalf("Meldeplan wurde zu %q geändert", got)
			}
			periods, err := vat.Periods(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			if len(periods) != 0 {
				t.Fatalf("Unerwartete Meldezeiträume: %+v", periods)
			}
			deadlines, err := env.deadlines(t).Deadlines(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range deadlines {
				if strings.HasPrefix(d.Key, DeadlineKeyVatReturn+".") || strings.HasPrefix(d.Key, DeadlineKeyAnnualVat+".") {
					t.Errorf("Unerwarteter Umsatzsteuertermin: %+v", d)
				}
			}
			proposal, err := vat.SuggestPeriodType(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			if proposal != nil {
				t.Fatalf("Unerwarteter Zeitraumvorschlag: %+v", proposal)
			}
			env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "quarter" })
			periods, err = vat.Periods(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			if len(periods) != 4 {
				t.Fatalf("Nach Klärung fehlen die Quartale: %+v", periods)
			}
		})
	}
}
