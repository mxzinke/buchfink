package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// ClosingSettings sind die drei Einstellungen, die die Abschlussbausteine
// steuern. Sie liegen als Schlüssel in der Einstellungstabelle, weil sie den
// Mandanten betreffen und nicht das Unternehmen als Rechtsträger — der
// Hebesatz gehört zur Gemeinde, die Abgrenzungsmethode zur Buchführung.
type ClosingSettings struct {
	// TradeTaxRatePercent ist der Gewerbesteuer-Hebesatz der Gemeinde in
	// Prozent (400 = 400 %).
	TradeTaxRatePercent int64 `json:"tradeTaxRatePercent"`
	// AccrualMethod ist "monthly" (Zwölftel) oder "daily" (taggenau).
	AccrualMethod string `json:"accrualMethod"`
	// AccrualThreshold ist die Vorschlagsschwelle der Abgrenzung in Cent.
	AccrualThreshold domain.Cents `json:"accrualThreshold"`
	// AccrualRelease ist der Auflösungstakt: "yearly" oder "monthly".
	AccrualRelease string `json:"accrualRelease"`
}

// ClosingSettingsService liest und schreibt diese Einstellungen.
type ClosingSettingsService struct {
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
}

// NewClosingSettingsService verdrahtet den Dienst.
func NewClosingSettingsService(settingsRepo domain.SettingsRepository, auditRepo domain.AuditRepository) *ClosingSettingsService {
	return &ClosingSettingsService{settingsRepo: settingsRepo, auditRepo: auditRepo}
}

// Get liefert die Einstellungen mit denselben Voreinstellungen, die die
// Abschlussbausteine verwenden, wenn nichts gespeichert ist.
func (s *ClosingSettingsService) Get(ctx context.Context) (*ClosingSettings, error) {
	out := &ClosingSettings{
		TradeTaxRatePercent: 400,
		AccrualMethod:       "monthly",
		AccrualThreshold:    domain.DefaultAccrualThreshold,
		AccrualRelease:      "yearly",
	}
	if v := s.read(ctx, domain.SettingTradeTaxRate); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			out.TradeTaxRatePercent = n
		}
	}
	if v := s.read(ctx, domain.SettingAccrualMethod); v == "monthly" || v == "daily" {
		out.AccrualMethod = v
	}
	if v := s.read(ctx, domain.SettingAccrualThreshold); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			out.AccrualThreshold = domain.Cents(n)
		}
	}
	if v := s.read(ctx, domain.SettingAccrualRelease); v == "yearly" || v == "monthly" {
		out.AccrualRelease = v
	}
	return out, nil
}

// Save prüft und schreibt die Einstellungen und protokolliert die Änderung mit
// Vorher- und Nachherwert.
func (s *ClosingSettingsService) Save(ctx context.Context, in ClosingSettings) (*ClosingSettings, error) {
	if in.TradeTaxRatePercent < 200 || in.TradeTaxRatePercent > 1000 {
		return nil, fmt.Errorf("der Gewerbesteuer-Hebesatz liegt zwischen 200 %% und 1000 %% (§ 16 Abs. 4 Satz 2 GewStG: mindestens 200 %%), angegeben sind %d %%", in.TradeTaxRatePercent)
	}
	if in.AccrualMethod != "monthly" && in.AccrualMethod != "daily" {
		return nil, fmt.Errorf("die Abgrenzungsmethode ist monatsgenau (monthly) oder taggenau (daily), nicht %q", in.AccrualMethod)
	}
	if in.AccrualThreshold < 0 {
		return nil, fmt.Errorf("die Vorschlagsschwelle kann nicht negativ sein")
	}
	if in.AccrualRelease != "yearly" && in.AccrualRelease != "monthly" {
		return nil, fmt.Errorf("der Auflösungstakt ist jährlich (yearly) oder monatlich (monthly), nicht %q", in.AccrualRelease)
	}
	before, _ := s.Get(ctx)
	writes := map[string]string{
		domain.SettingTradeTaxRate:     strconv.FormatInt(in.TradeTaxRatePercent, 10),
		domain.SettingAccrualMethod:    in.AccrualMethod,
		domain.SettingAccrualThreshold: strconv.FormatInt(int64(in.AccrualThreshold), 10),
		domain.SettingAccrualRelease:   in.AccrualRelease,
	}
	for key, value := range writes {
		if err := s.settingsRepo.Set(ctx, key, value); err != nil {
			return nil, fmt.Errorf("Einstellung %s konnte nicht gespeichert werden: %w", key, err)
		}
	}
	if s.auditRepo != nil && before != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "SETTINGS", "closing",
			fmt.Sprintf("Abschluss-Einstellungen geändert: Hebesatz %d %% → %d %%, Abgrenzung %s → %s, Schwelle %s → %s, Auflösung %s → %s",
				before.TradeTaxRatePercent, in.TradeTaxRatePercent, before.AccrualMethod, in.AccrualMethod,
				before.AccrualThreshold, in.AccrualThreshold, before.AccrualRelease, in.AccrualRelease))
	}
	return s.Get(ctx)
}

func (s *ClosingSettingsService) read(ctx context.Context, key string) string {
	if s.settingsRepo == nil {
		return ""
	}
	v, err := s.settingsRepo.Get(ctx, key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}
