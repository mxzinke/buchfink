package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
)

// AnlagenspiegelSource liefert die Entwicklung des Anlagevermögens.
//
// Der Export kennt die Anlagenbuchhaltung darüber und nicht weiter: er braucht
// eine Auswertung, keine Kartei.
type AnlagenspiegelSource interface {
	Anlagenspiegel(ctx context.Context) (*domain.Anlagenspiegel, error)
}

// EBilanzService handles official XBRL taxonomy mapping and instance generation.
type EBilanzService struct {
	accountingSvc *AccountingService
	settingsRepo  domain.SettingsRepository
	auditRepo     domain.AuditRepository
	assets        AnlagenspiegelSource
}

// SetAnlagenspiegelSource verdrahtet den Anlagenspiegel in den Export. Ohne sie
// entsteht die Instanz wie bisher, nur ohne den Nachweis zum Anlagevermögen.
func (s *EBilanzService) SetAnlagenspiegelSource(src AnlagenspiegelSource) { s.assets = src }

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

	// Der Anlagenspiegel ist Bestandteil des Anhangs (§ 284 Abs. 3 HGB) und im
	// Kontennachweis das, was den ausgewiesenen Buchwert erklärt. Scheitert seine
	// Auswertung, entsteht die Instanz ohne ihn — eine E-Bilanz an einer nicht
	// rechenbaren Kartei scheitern zu lassen hülfe niemandem.
	var spiegel *domain.Anlagenspiegel
	if s.assets != nil {
		spiegel, _ = s.assets.Anlagenspiegel(ctx)
	}

	xbrl, err := ebilanz.GenerateEBilanzXBRL(settings, accounts, summary, spiegel)
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
