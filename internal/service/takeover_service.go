package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/calendar"
	"github.com/buchfink/buchfink/internal/domain"
)

// TakeoverService führt den Umstieg aus einem Vorsystem: den festgelegten
// Beginn der Bücher, die Eröffnungsbuchungen mit offenen Posten und Anlagen
// und ihre Abstimmung gegen die Schlussbilanz des Vorsystems.
type TakeoverService struct {
	settingsRepo   domain.SettingsRepository
	foundationRepo domain.FoundationRepository
	journalRepo    domain.JournalRepository
	assetRepo      domain.AssetRepository
	auditRepo      domain.AuditRepository
	journal        *JournalService
	assets         *AssetService
}

func NewTakeoverService(
	settingsRepo domain.SettingsRepository,
	foundationRepo domain.FoundationRepository,
	journalRepo domain.JournalRepository,
	assetRepo domain.AssetRepository,
	auditRepo domain.AuditRepository,
	journal *JournalService,
	assets *AssetService,
) *TakeoverService {
	return &TakeoverService{
		settingsRepo:   settingsRepo,
		foundationRepo: foundationRepo,
		journalRepo:    journalRepo,
		assetRepo:      assetRepo,
		auditRepo:      auditRepo,
		journal:        journal,
		assets:         assets,
	}
}

// TakeoverState ist der festgelegte Umstieg und sein Stand.
type TakeoverState struct {
	// Date ist der Tag, zu dem das Unternehmen mit bestehenden Büchern beginnt;
	// leer ohne Umstieg.
	Date string `json:"date"`
	// FiscalYear ist das Geschäftsjahr, das an diesem Tag beginnt.
	FiscalYear int `json:"fiscalYear"`
	// Booked meldet Eröffnungsbuchungen zum Umstieg, die nicht storniert sind.
	Booked bool `json:"booked"`
}

// State liest den festgelegten Umstieg.
func (s *TakeoverService) State(ctx context.Context) (*TakeoverState, error) {
	date, err := takeoverDate(ctx, s.settingsRepo)
	if err != nil {
		return nil, err
	}
	state := &TakeoverState{Date: date}
	if date == "" {
		return state, nil
	}
	state.FiscalYear = domain.GetFiscalYearForDate(date, s.journal.fiscalYearStartMonth(ctx))
	entries, err := s.openingEntries(ctx, state)
	if err != nil {
		return nil, err
	}
	state.Booked = len(entries) > 0
	return state, nil
}

// takeoverDate liest den festgelegten Tag des Umstiegs; leer ohne Umstieg.
func takeoverDate(ctx context.Context, settings domain.SettingsRepository) (string, error) {
	date, err := settings.Get(ctx, domain.SettingTakeoverDate)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("der festgelegte Umstieg ließ sich nicht lesen: %w", err)
	}
	return date, nil
}

// SetDate legt fest, dass das Unternehmen mit bestehenden Büchern beginnt und
// zu welchem Tag; ein leerer Tag nimmt den Umstieg zurück.
//
// Der Tag entscheidet, ob Buchfink Bestände aus einem Vorsystem annimmt. Er
// ist deshalb dem Menschen vorbehalten: ein Agent bucht die Eröffnung, ob es
// eine gibt, entscheidet er nicht. Gründung und Umstieg schließen sich aus,
// weil eine in Buchfink gegründete Gesellschaft keine Bestände mitbringt.
func (s *TakeoverService) SetDate(ctx context.Context, date string) (*TakeoverState, error) {
	if err := requireHuman(ctx, "Die Festlegung des Umstiegs"); err != nil {
		return nil, err
	}
	date = strings.TrimSpace(date)
	return atomically(ctx, s.journal, func(ctx context.Context) (*TakeoverState, error) {
		before, err := s.State(ctx)
		if err != nil {
			return nil, err
		}
		if before.Date == date {
			return before, nil
		}
		if before.Booked {
			return nil, fmt.Errorf(
				"zum %s sind Eröffnungsbuchungen gebucht. Der Umstieg lässt sich erst ändern, wenn sie storniert sind",
				domain.GermanDate(before.Date))
		}
		if date != "" {
			if err := s.checkDate(ctx, date); err != nil {
				return nil, err
			}
		}
		if err := s.settingsRepo.Set(ctx, domain.SettingTakeoverDate, date); err != nil {
			return nil, err
		}
		details := "Umstieg zurückgenommen: das Unternehmen beginnt ohne Bestände aus einem Vorsystem"
		if date != "" {
			details = fmt.Sprintf("Umstieg festgelegt: die Bücher beginnen am %s mit den Beständen des Vorsystems",
				domain.GermanDate(date))
		}
		if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", domain.SettingTakeoverDate,
			details, map[string]string{domain.SettingTakeoverDate: before.Date},
			map[string]string{domain.SettingTakeoverDate: date}); err != nil {
			return nil, err
		}
		return s.State(ctx)
	})
}

// checkDate prüft, ob die Bücher an diesem Tag mit übernommenen Beständen
// beginnen können.
func (s *TakeoverService) checkDate(ctx context.Context, date string) error {
	if _, err := calendar.Parse(date); err != nil {
		return fmt.Errorf("der Tag des Umstiegs fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	startMonth := s.journal.fiscalYearStartMonth(ctx)
	year := domain.GetFiscalYearForDate(date, startMonth)
	if date != domain.FiscalYearFirstDay(year, startMonth) {
		return fmt.Errorf(
			"ein Umstieg beginnt am ersten Tag eines Geschäftsjahres, hier am %s. Einen Umstieg im "+
				"laufenden Geschäftsjahr bildet Buchfink nicht ab",
			domain.GermanDate(domain.FiscalYearFirstDay(year, startMonth)))
	}
	foundation, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("die erfasste Gründung ließ sich nicht lesen: %w", err)
	}
	if foundation != nil {
		return fmt.Errorf(
			"für das Unternehmen ist eine Gründung erfasst. Eine in Buchfink gegründete Gesellschaft " +
				"beginnt mit ihrer Eröffnungsbilanz und übernimmt keine Bestände aus einem Vorsystem")
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return err
	}
	if settings != nil && settings.FoundedOn != "" && settings.FoundedOn >= date {
		return fmt.Errorf(
			"das Unternehmen ist laut Stammdaten am %s gegründet. Bestände aus einem Vorsystem gibt es "+
				"erst zu einem späteren Geschäftsjahresbeginn", domain.GermanDate(settings.FoundedOn))
	}
	years, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return fmt.Errorf("die Geschäftsjahre konnten nicht gelesen werden: %w", err)
	}
	for _, booked := range years {
		if booked < year {
			return fmt.Errorf(
				"im Geschäftsjahr %d stehen Buchungen. Die Bücher in Buchfink beginnen damit vor dem %s; "+
					"die Bestände kommen dann aus dem Saldenvortrag", booked, domain.GermanDate(date))
		}
	}
	return nil
}

// ensureNoTakeover hält Gründung und Umstieg auseinander: wer Bestände aus
// einem Vorsystem übernimmt, erfasst keine Gründung.
func ensureNoTakeover(ctx context.Context, settings domain.SettingsRepository) error {
	date, err := takeoverDate(ctx, settings)
	if err != nil {
		return err
	}
	if date != "" {
		return fmt.Errorf(
			"für das Unternehmen ist ein Umstieg zum %s festgelegt: es beginnt mit den Beständen eines "+
				"Vorsystems. Eine Gründung lässt sich daneben nicht erfassen", domain.GermanDate(date))
	}
	return nil
}

// takeoverStatement ist die Schlussbilanz des Vorsystems, wie sie zur
// Eröffnung erfasst wurde.
type takeoverStatement struct {
	ReceiptID    uint   `json:"receiptId"`
	LegacySystem string `json:"legacySystem,omitempty"`
	// Accounts sind die Bestände je Sachkonto, die Sammelkonten der Forderungen
	// und Verbindlichkeiten eingeschlossen.
	Accounts []OpeningBalanceLine `json:"accounts"`
}

func (s *TakeoverService) saveStatement(ctx context.Context, statement takeoverStatement) error {
	raw, err := json.Marshal(statement)
	if err != nil {
		return err
	}
	return s.settingsRepo.Set(ctx, domain.SettingTakeoverStatement, string(raw))
}

// statement liest die erfasste Schlussbilanz; nil, solange keine Eröffnung
// gebucht wurde.
func (s *TakeoverService) statement(ctx context.Context) (*takeoverStatement, error) {
	raw, err := s.settingsRepo.Get(ctx, domain.SettingTakeoverStatement)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && raw == "") {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("die erfasste Schlussbilanz des Vorsystems ließ sich nicht lesen: %w", err)
	}
	var statement takeoverStatement
	if err := json.Unmarshal([]byte(raw), &statement); err != nil {
		return nil, fmt.Errorf("die erfasste Schlussbilanz des Vorsystems ist nicht lesbar: %w", err)
	}
	return &statement, nil
}

// openingEntries sind die Eröffnungsbuchungen zum Umstieg, die nicht im selben
// Geschäftsjahr storniert sind. Dieselbe Abgrenzung entscheidet, ob sich die
// Eröffnung erneut buchen lässt (ensureFirstOpeningBalance).
func (s *TakeoverService) openingEntries(ctx context.Context, state *TakeoverState) ([]domain.JournalEntry, error) {
	entries, err := s.journalRepo.FindAll(ctx, state.FiscalYear)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen des Jahres %d ließen sich nicht lesen: %w", state.FiscalYear, err)
	}
	out := make([]domain.JournalEntry, 0)
	for i := range entries {
		entry := &entries[i]
		if entry.Source != domain.EntrySourceOpening || entry.Kind == domain.EntryKindReversal ||
			entry.BookingDate != state.Date {
			continue
		}
		reversal, err := s.journalRepo.FindReversalOf(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
		if reversal != nil && reversal.FiscalYear == state.FiscalYear {
			continue
		}
		out = append(out, *entry)
	}
	return out, nil
}
