package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// SettingsService manages company profile and fiscal year configurations.
type SettingsService struct {
	serviceClock
	settingsRepo       domain.SettingsRepository
	auditRepo          domain.AuditRepository
	registrar          FoundationRegistrar
	journalRepo        domain.JournalRepository
	festschreibungRepo domain.FestschreibungRepository
	events             ObligationEventRecorder
}

// ObligationEventRecorder erfasst Anlässe für den Pflichtenkalender.
type ObligationEventRecorder interface {
	RecordEvent(ctx context.Context, kind domain.ObligationEventKind, date, note string) (*domain.ObligationEvent, error)
}

// SetEventRecorder meldet geänderte Gesellschafter an den Pflichtenkalender:
// daraus folgen Gesellschafterliste und Transparenzregister.
func (s *SettingsService) SetEventRecorder(r ObligationEventRecorder) { s.events = r }

// FoundationRegistrar trägt den Tag der Eintragung ins Handelsregister an der
// Gründung nach. Erst damit endet die Vorgesellschaft und mit ihr der Zusatz
// „i. G." am Firmennamen.
type FoundationRegistrar interface {
	RegisterFromSettings(ctx context.Context, date, court, number string) error
}

// SetFoundationRegistrar koppelt die Stammdaten an die Gründung.
func (s *SettingsService) SetFoundationRegistrar(r FoundationRegistrar) { s.registrar = r }

// SetBookkeeping gibt den Stammdaten den Stand der Bücher. Mit ihm hält
// UpdateCompanySettings den Beginn des Geschäftsjahres fest, sobald gebucht
// oder festgeschrieben ist.
func (s *SettingsService) SetBookkeeping(journal domain.JournalRepository, commitments domain.FestschreibungRepository) {
	s.journalRepo, s.festschreibungRepo = journal, commitments
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
	if err := requireHuman(ctx, "Die Änderung der Unternehmensdaten"); err != nil {
		return err
	}
	// Der bisherige Stand wird vor dem Schreiben gelesen: danach gibt es ihn
	// nicht mehr, und ohne ihn stünde im Protokoll „Stammdaten aktualisiert"
	// ohne die Angabe, was sich geändert hat.
	before, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return fmt.Errorf("die bisherigen Unternehmensdaten ließen sich nicht lesen: %w", err)
	}
	if err := s.ensureStartMonthKept(ctx, before, settings.FiscalYearStartMonth); err != nil {
		return err
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
	if s.events != nil && shareholdersChanged(before, settings) {
		if _, err := s.events.RecordEvent(ctx, domain.EventShareholderChange, s.today(),
			"Gesellschafterliste in den Unternehmensdaten geändert"); err != nil {
			return err
		}
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

// shareholdersChanged meldet eine Veränderung in den Personen der
// Gesellschafter oder im Umfang ihrer Beteiligung (§ 40 Abs. 1 GmbHG). Die
// erste Erfassung ist keine Veränderung, ein Formular ohne Liste auch nicht.
func shareholdersChanged(before, after *domain.CompanySettings) bool {
	if before == nil || len(before.Shareholders) == 0 || after.Shareholders == nil {
		return false
	}
	count := map[string]int{}
	for _, sh := range before.Shareholders {
		count[fmt.Sprintf("%s|%d", strings.TrimSpace(sh.Name), sh.ShareCapital)]++
	}
	for _, sh := range after.Shareholders {
		count[fmt.Sprintf("%s|%d", strings.TrimSpace(sh.Name), sh.ShareCapital)]--
	}
	for _, n := range count {
		if n != 0 {
			return true
		}
	}
	return false
}

// ensureStartMonthKept weist einen neuen Beginn des Geschäftsjahres ab, sobald
// die Bücher Buchungen oder Festschreibungen tragen: jede gehört zu dem Jahr,
// in das sie nach dem bisherigen Beginn fällt, und an diesem Jahr hängen
// Festschreibung, Feststellung und Hash-Kette.
func (s *SettingsService) ensureStartMonthKept(ctx context.Context, before *domain.CompanySettings, next int) error {
	if next < 1 || next > 12 {
		next = 1
	}
	if before == nil || before.FiscalYearStartMonth == next || s.journalRepo == nil {
		return nil
	}
	years, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return fmt.Errorf("der Stand der Bücher ließ sich nicht prüfen: %w", err)
	}
	started := len(years) > 0
	if !started && s.festschreibungRepo != nil {
		commitments, err := s.festschreibungRepo.FindCutoffOnOrAfter(ctx, "")
		if err != nil {
			return fmt.Errorf("der Stand der Bücher ließ sich nicht prüfen: %w", err)
		}
		started = len(commitments) > 0
	}
	if !started {
		return nil
	}
	return fmt.Errorf(
		"der Beginn des Geschäftsjahres lässt sich nicht mehr ändern: die Bücher enthalten schon Buchungen " +
			"oder Festschreibungen, und jede gehört zu dem Geschäftsjahr nach dem bisherigen Beginn. Eine " +
			"Umstellung des Wirtschaftsjahres mit Rumpfgeschäftsjahr bildet Buchfink nicht ab")
}

// settableKeys sind die einzelnen Einstellungen, die SetValue schreibt. Die
// Unternehmensdaten gehören nicht dazu: sie gehen über UpdateCompanySettings
// mit ihren Prüfungen und bleiben dort der App vorbehalten.
var settableKeys = map[string]bool{
	domain.SettingSystemChangeDate: true,
}

// SetValue schreibt eine einzelne Einstellung und protokolliert ihren alten und
// neuen Wert. Eine Einstellung, deren Änderung niemand sieht, ist die
// bequemste Stelle, eine Auswertung zu beeinflussen.
func (s *SettingsService) SetValue(ctx context.Context, key, value, description string) error {
	if !settableKeys[key] {
		return fmt.Errorf("die Einstellung %q lässt sich nicht einzeln setzen", key)
	}
	before, err := s.settingsRepo.Get(ctx, key)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("der bisherige Wert von %s ließ sich nicht lesen: %w", key, err)
	}
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
