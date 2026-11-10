package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// ensureSupportedCompany weist Abschluss und Steuerrückstellung einer
// Gesellschaft ab, die als gemeinnützig, in Liquidation oder als
// Organgesellschaft gekennzeichnet ist (CompanySettings.Unsupported).
func ensureSupportedCompany(ctx context.Context, settings domain.SettingsRepository) error {
	if settings == nil {
		return nil
	}
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		return fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}
	return cfg.Unsupported()
}
