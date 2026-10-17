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
	registrar    FoundationRegistrar
}

// FoundationRegistrar trägt den Tag der Eintragung ins Handelsregister an der
// Gründung nach. Erst damit endet die Vorgesellschaft und mit ihr der Zusatz
// „i. G." am Firmennamen.
type FoundationRegistrar interface {
	RegisterFromSettings(ctx context.Context, date, court, number string) error
}

// SetFoundationRegistrar koppelt die Stammdaten an die Gründung.
func (s *SettingsService) SetFoundationRegistrar(r FoundationRegistrar) { s.registrar = r }

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
	if err := refuseAgent(ctx, "Die Änderung der Unternehmensdaten"); err != nil {
		return err
	}
	// Der bisherige Stand wird vor dem Schreiben gelesen: danach gibt es ihn
	// nicht mehr, und ohne ihn stünde im Protokoll „Stammdaten aktualisiert"
	// ohne die Angabe, was sich geändert hat. Ein Fehler beim Lesen hält das
	// Speichern nicht auf.
	var before *domain.CompanySettings
	if prev, err := s.settingsRepo.GetCompanySettings(ctx); err == nil {
		before = prev
	}

	// Ein Formular ohne Rechtsform ändert sie nicht. Eine andere Rechtsform als
	// GmbH oder UG führt Buchfink nicht.
	if settings.LegalForm == "" {
		settings.LegalForm = domain.LegalFormGmbH
		if before != nil && before.LegalForm != "" {
			settings.LegalForm = before.LegalForm
		}
	}
	if !domain.IsSupportedLegalForm(settings.LegalForm) {
		return fmt.Errorf("Buchfink führt die Bücher einer %s oder %s; die Rechtsform %q ist nicht vorgesehen",
			domain.LegalFormGmbH, domain.LegalFormUG, settings.LegalForm)
	}

	// Die Eintragung wird in den Stammdaten erfasst; die Gründung muss sie
	// kennen, sonst bleibt die Gesellschaft dort Vorgesellschaft.
	registeredNow := settings.RegisteredOn != "" && (before == nil || before.RegisteredOn != settings.RegisteredOn)
	if s.registrar != nil && registeredNow {
		if err := s.registrar.RegisterFromSettings(
			ctx, settings.RegisteredOn, settings.RegisterCourt, settings.RegisterNumber,
		); err != nil {
			return err
		}
	}

	if err := s.settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		return err
	}

	if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate,
		"SETTINGS",
		"COMPANY",
		fmt.Sprintf("Unternehmensstammdaten für %s aktualisiert", settings.CompanyName),
		before, settings,
	); err != nil {
		return err
	}

	return nil
}

// SetValue schreibt eine einzelne Einstellung und protokolliert ihren alten und
// neuen Wert.
//
// Für die Einstellungen, die nicht Teil der Unternehmensstammdaten sind: der
// Umstellungszeitpunkt der Datenübernahme, die Texte der
// Organisationsanweisung, der Sicherungsordner. Sie sind einzelne Schlüssel und
// sollen trotzdem im Änderungsprotokoll stehen — eine Einstellung, deren
// Änderung niemand sieht, ist die bequemste Stelle, eine Auswertung zu
// beeinflussen.
func (s *SettingsService) SetValue(ctx context.Context, key, value, description string) error {
	before, _ := s.settingsRepo.Get(ctx, key)
	if err := s.settingsRepo.Set(ctx, key, value); err != nil {
		return err
	}
	if s.auditRepo == nil {
		return nil
	}
	if description == "" {
		description = fmt.Sprintf("Einstellung %s geändert", key)
	}
	if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", key, description,
		map[string]string{key: before}, map[string]string{key: value}); err != nil {
		return err
	}
	return nil
}

// GetValue liest eine einzelne Einstellung. Ein fehlender Schlüssel liefert
// den leeren Wert: die Einstellungen sind nicht alle vorbelegt.
func (s *SettingsService) GetValue(ctx context.Context, key string) string {
	value, err := s.settingsRepo.Get(ctx, key)
	if err != nil {
		return ""
	}
	return value
}
