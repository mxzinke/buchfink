package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/models"
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

	legacySettings := &models.CompanySettings{
		CompanyName: settings.CompanyName,
		LegalForm:   settings.LegalForm,
		FiscalYear:  settings.FiscalYear,
		TaxNumber:   settings.TaxNumber,
		VatID:       settings.VatID,
	}

	var legacyAccounts []models.Account
	for _, a := range accounts {
		legacyAccounts = append(legacyAccounts, models.Account{
			Number:  a.Number,
			Name:    a.Name,
			Type:    string(a.Type),
			Balance: a.Balance,
		})
	}

	legacySummary := &models.FinancialSummary{
		TotalRevenue:  summary.TotalRevenue,
		TotalExpenses: summary.TotalExpenses,
		NetIncome:     summary.NetIncome,
	}

	xbrl, err := ebilanz.GenerateEBilanzXBRL(legacySettings, legacyAccounts, legacySummary)
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
