package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
)

// EBilanzService handles official XBRL taxonomy mapping and instance generation.
type EBilanzService struct {
	accountingSvc *AccountingService
	settingsRepo  domain.SettingsRepository
	auditRepo     domain.AuditRepository
}

func NewEBilanzService(
	accountingSvc *AccountingService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *EBilanzService {
	return &EBilanzService{
		accountingSvc: accountingSvc,
		settingsRepo:  settingsRepo,
		auditRepo:     auditRepo,
	}
}

// ExportXBRL generates a valid XBRL instance document (German GAAP 6.7) with Kontennachweis.
func (s *EBilanzService) ExportXBRL(ctx context.Context) (string, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load company settings: %w", err)
	}

	accounts, err := s.accountingSvc.GetAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load accounts: %w", err)
	}

	summary, err := s.accountingSvc.GetFinancialSummary(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load financial summary: %w", err)
	}

	xbrl, err := ebilanz.GenerateEBilanzXBRL(settings, accounts, summary)
	if err != nil {
		return "", err
	}

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionExport,
		"EBILANZ",
		fmt.Sprintf("%d", settings.FiscalYear),
		"E-Bilanz XBRL-Datei für Finanzamt generiert",
	)

	return xbrl, nil
}
