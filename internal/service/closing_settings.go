package service

import (
	"context"
	"errors"
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
	// Prozent (400 = 400 %); null heißt: nicht hinterlegt. Eine Voreinstellung
	// gibt es nicht, denn der Hebesatz ist eine Angabe der Gemeinde.
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
		AccrualMethod:    "monthly",
		AccrualThreshold: domain.DefaultAccrualThreshold,
		AccrualRelease:   "yearly",
	}
	values := map[string]string{}
	for _, key := range []string{domain.SettingTradeTaxRate, domain.SettingAccrualMethod,
		domain.SettingAccrualThreshold, domain.SettingAccrualRelease} {
		v, err := s.read(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("die Abschluss-Einstellung %s ließ sich nicht lesen: %w", key, err)
		}
		values[key] = v
	}
	if v := values[domain.SettingTradeTaxRate]; v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			out.TradeTaxRatePercent = n
		}
	}
	if v := values[domain.SettingAccrualMethod]; v == "monthly" || v == "daily" {
		out.AccrualMethod = v
	}
	if v := values[domain.SettingAccrualThreshold]; v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			out.AccrualThreshold = domain.Cents(n)
		}
	}
	if v := values[domain.SettingAccrualRelease]; v == "yearly" || v == "monthly" {
		out.AccrualRelease = v
	}
	return out, nil
}

// Save prüft und schreibt die Einstellungen und protokolliert die Änderung mit
// Vorher- und Nachherwert.
func (s *ClosingSettingsService) Save(ctx context.Context, in ClosingSettings) (*ClosingSettings, error) {
	if err := requireHuman(ctx, "Die Änderung der Abschluss-Einstellungen"); err != nil {
		return nil, err
	}
	if in.TradeTaxRatePercent != 0 && (in.TradeTaxRatePercent < 200 || in.TradeTaxRatePercent > 1000) {
		return nil, fmt.Errorf("der Gewerbesteuer-Hebesatz liegt zwischen 200 %% und 1000 %% (§ 16 Abs. 4 Satz 2 GewStG: mindestens 200 %%, ab 2027 280 %%), angegeben sind %d %%", in.TradeTaxRatePercent)
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
	before, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	writes := map[string]string{
		domain.SettingTradeTaxRate:     tradeTaxRateSetting(in.TradeTaxRatePercent),
		domain.SettingAccrualMethod:    in.AccrualMethod,
		domain.SettingAccrualThreshold: strconv.FormatInt(int64(in.AccrualThreshold), 10),
		domain.SettingAccrualRelease:   in.AccrualRelease,
	}
	for key, value := range writes {
		if err := s.settingsRepo.Set(ctx, key, value); err != nil {
			return nil, fmt.Errorf("Einstellung %s konnte nicht gespeichert werden: %w", key, err)
		}
	}
	if s.auditRepo != nil {
		if err := logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", "closing",
			fmt.Sprintf("Abschluss-Einstellungen geändert: Hebesatz %d %% → %d %%, Abgrenzung %s → %s, Schwelle %s → %s, Auflösung %s → %s",
				before.TradeTaxRatePercent, in.TradeTaxRatePercent, before.AccrualMethod, in.AccrualMethod,
				before.AccrualThreshold, in.AccrualThreshold, before.AccrualRelease, in.AccrualRelease)); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx)
}

// read liest eine Einstellung; ein fehlender Schlüssel ist der leere Wert.
func (s *ClosingSettingsService) read(ctx context.Context, key string) (string, error) {
	if s.settingsRepo == nil {
		return "", nil
	}
	v, err := s.settingsRepo.Get(ctx, key)
	if errors.Is(err, domain.ErrNotFound) {
		return "", nil
	}
	return strings.TrimSpace(v), err
}

// tradeTaxRateSetting schreibt einen nicht hinterlegten Hebesatz als leeren
// Wert statt als null Prozent.
func tradeTaxRateSetting(rate int64) string {
	if rate == 0 {
		return ""
	}
	return strconv.FormatInt(rate, 10)
}
