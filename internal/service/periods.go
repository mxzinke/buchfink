package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// PeriodReader liest Jahreszeiträume und Sperren ohne Abschlussvorprüfungen.
// Journal und Abschlussbausteine teilen dieselben Repositories.
type PeriodReader struct {
	fiscalCalendar
	journalRepo        domain.JournalRepository
	festschreibungRepo domain.FestschreibungRepository
	foundationRepo     domain.FoundationRepository
}

// PeriodSource liefert den Zeitraum, ohne ein Geschäftsjahr anzulegen.
type PeriodSource interface {
	PeriodOf(context.Context, int) (*domain.FiscalYear, error)
}

// PeriodService verwaltet Geschäftsjahre, Festschreibungen und Abschlussstände.
// Schreibende Vorgänge verwenden dieselbe Transaktion wie der Journaldienst.
type PeriodService struct {
	*PeriodReader
	auditor[int]
	tx              domain.TxRunner
	adoption        PeriodAdoptionChecks
	revenue         RevenueSource
	preCommitChecks []PreCommitCheck
}

// PeriodAdoptionChecks prüft die Abschlusszahlen vor einem Statuswechsel.
type PeriodAdoptionChecks interface {
	EnsureLegalReserve(context.Context, int) error
	EnsureStatementReady(context.Context, int) error
}

func (s *PeriodService) SetAdoptionChecks(checks PeriodAdoptionChecks) { s.adoption = checks }
func (s *PeriodService) SetRevenueSource(source RevenueSource)         { s.revenue = source }

func newPeriodReader(journal domain.JournalRepository, settings domain.SettingsRepository) *PeriodReader {
	return &PeriodReader{fiscalCalendar: fiscalCalendar{settingsRepo: settings}, journalRepo: journal}
}

// NewPeriodService verbindet Jahresverwaltung und Festschreibung mit ihrer Transaktion.
func NewPeriodService(years domain.FiscalYearRepository, journal domain.JournalRepository,
	settings domain.SettingsRepository, commitments domain.FestschreibungRepository,
	foundation domain.FoundationRepository, audit domain.AuditRepository, tx domain.TxRunner) *PeriodService {
	reader := newPeriodReader(journal, settings)
	reader.fiscalYearRepo, reader.festschreibungRepo, reader.foundationRepo = years, commitments, foundation
	return &PeriodService{PeriodReader: reader, auditor: auditor[int]{auditRepo: audit, entity: "FISCAL_YEAR"}, tx: tx}
}

// ReadOnly liefert die gemeinsame Periodensicht ohne Schreibvorgänge und Hooks.
func (s *PeriodService) ReadOnly() *PeriodReader { return s.PeriodReader }

// PeriodState beschreibt den Buchungstag. ClosingPeriod sagt, ob er der
// Bilanzstichtag ist; ClosingOpen, ob er noch Abschlussbuchungen aufnimmt.
// AdoptedYear kann bei geändertem Beginnmonat eine andere Jahresnummer haben.
type PeriodState struct {
	FiscalYear       int
	EndDate          string
	Open             bool
	CommittedThrough string
	ClosingPeriod    bool
	ClosingOpen      bool
	AdoptedYear      *domain.FiscalYear
	lock             *domain.Festschreibung
	closingLock      *domain.Festschreibung
}

// State liest die Sperren anhand des Tages und der gespeicherten Jahresgrenzen.
// Ein geänderter Beginnmonat öffnet weder Festschreibung noch Feststellung.
func (s *PeriodReader) State(ctx context.Context, date string) (PeriodState, error) {
	year := s.FiscalYearOf(ctx, date)
	state := PeriodState{FiscalYear: year, EndDate: s.fiscalYearEnd(ctx, year)}
	state.ClosingPeriod = date == state.EndDate
	if s.fiscalYearRepo != nil {
		years, err := s.fiscalYearRepo.FindAll(ctx)
		if err != nil {
			return state, fmt.Errorf("der Abschlussstand des Geschäftsjahres %d konnte nicht geprüft werden: %w", year, err)
		}
		for i := range years {
			fy := &years[i]
			if (fy.Year == year || (fy.StartDate <= date && date <= fy.EndDate)) && fy.IsAdopted() {
				state.AdoptedYear = fy
				break
			}
		}
	}
	if s.festschreibungRepo != nil {
		records, err := s.festschreibungRepo.FindCutoffOnOrAfter(ctx, "")
		if err != nil {
			return state, fmt.Errorf("Festschreibungsstand konnte nicht geprüft werden: %w", err)
		}
		for i := range records {
			rec := &records[i]
			if rec.FiscalYear != year && (rec.CutoffDate < date || date < s.fiscalYearStart(ctx, rec.FiscalYear)) {
				continue
			}
			if state.CommittedThrough == "" {
				state.CommittedThrough = rec.CutoffDate
			}
			if rec.CutoffDate < date {
				continue
			}
			if state.lock == nil {
				state.lock = rec
			}
			if state.closingLock == nil && (rec.FiscalYear != year || rec.PeriodType == domain.PeriodTypeYear) {
				state.closingLock = rec
			}
		}
	}
	state.Open = state.AdoptedYear == nil && state.lock == nil
	state.ClosingOpen = state.ClosingPeriod && state.AdoptedYear == nil && state.closingLock == nil
	return state, nil
}

func (state PeriodState) ensureNotAdopted(date string) error {
	fy := state.AdoptedYear
	if fy == nil {
		return nil
	}
	return fmt.Errorf(
		"Der Jahresabschluss %d ist am %s festgestellt. Eine Buchung zum %s ist deshalb nicht mehr "+
			"möglich. Muss der Abschluss geändert werden, setzen Sie die Feststellung unter "+
			"Abschlüsse › Jahr mit Angabe des Grundes zurück", fy.Year, fy.AdoptedOn, date)
}

func (state PeriodState) ensureOpen(date string, closing bool) error {
	rec := state.lock
	if closing {
		rec = state.closingLock
	}
	if rec == nil {
		return nil
	}
	if closing && rec.FiscalYear == state.FiscalYear {
		return fmt.Errorf("Das Geschäftsjahr %d ist festgeschrieben, auch seine Abschlussperiode. Eine Buchung zum %s ist "+
			"nicht mehr möglich – bitte über eine Stornierung korrigieren", state.FiscalYear, date)
	}
	message := fmt.Sprintf("Der Zeitraum bis %s ist festgeschrieben. Eine Buchung zum %s ist nicht mehr möglich – bitte über eine Stornierung korrigieren", rec.CutoffDate, date)
	if rec.FiscalYear == state.FiscalYear && date == state.EndDate {
		message += ". Eine Abschlussbuchung zum Bilanzstichtag ist bis zur Jahresfestschreibung als Buchung der Abschlussperiode möglich"
	}
	return errors.New(message)
}

// assignClosingPeriod ordnet Abschlussbuchungen und AfA zum Bilanzstichtag
// automatisch der Abschlussperiode zu; Handbuchungen auf ausdrücklichen Wunsch.
func (s *PeriodReader) assignClosingPeriod(ctx context.Context, e *domain.JournalEntry) error {
	end := s.fiscalYearEnd(ctx, e.FiscalYear)
	if (e.Source == domain.EntrySourceClosing || e.Source == domain.EntrySourceDepreciation) && e.BookingDate == end {
		e.ClosingPeriod = true
	}
	if e.ClosingPeriod && e.BookingDate != end {
		return fmt.Errorf("eine Buchung der Abschlussperiode trägt den Bilanzstichtag des Geschäftsjahres %d (%s), nicht den %s", e.FiscalYear, end, e.BookingDate)
	}
	return nil
}

// FiscalYearOf ordnet den Tag nach dem Beginnmonat der Unternehmensdaten zu.
func (s *PeriodReader) FiscalYearOf(ctx context.Context, date string) int {
	return domain.GetFiscalYearForDate(date, s.fiscalYearStartMonth(ctx))
}

// CommittedThrough ist der jüngste Stichtag über alle Geschäftsjahre.
func (s *PeriodReader) CommittedThrough(ctx context.Context) (string, error) {
	if s.festschreibungRepo == nil {
		return "", nil
	}
	all, err := s.festschreibungRepo.FindCutoffOnOrAfter(ctx, "")
	if err != nil || len(all) == 0 {
		return "", err
	}
	return all[0].CutoffDate, nil
}
