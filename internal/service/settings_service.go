// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// SettingsService manages company profile and fiscal year configurations.
type SettingsService struct {
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
}

func NewSettingsService(
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *SettingsService {
	return &SettingsService{
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
	}
}

func (s *SettingsService) GetCompanySettings(ctx context.Context) (*domain.CompanySettings, error) {
	return s.settingsRepo.GetCompanySettings(ctx)
}

func (s *SettingsService) UpdateCompanySettings(ctx context.Context, settings *domain.CompanySettings) error {
	if err := s.settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		return err
	}

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionUpdate,
		"SETTINGS",
		"COMPANY",
		fmt.Sprintf("Unternehmensstammdaten für %s aktualisiert", settings.CompanyName),
	)

	return nil
}
