package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
)

// PeriodCommitRequest beschreibt den Zeitraum und die Begründung zum Übergehen
// blockierender Befunde. Das Geschäftsjahr folgt aus dem Stichtag.
type PeriodCommitRequest struct {
	PeriodType     string
	PeriodLabel    string
	CutoffDate     string
	OverrideReason string
}

// PeriodCommitResult trennt einen gespeicherten, blockierenden Prüflauf von
// einem Fehler, bei dem der gesamte Vorgang zurückrollen muss.
type PeriodCommitResult struct {
	Commitment *domain.Festschreibung
	Blocked    *CommitBlocked
}

// Commit schreibt fachlich fest. Der Kern holt danach den Zeitstempel ohne
// Schreibsperre. Ein blockierender Prüflauf bleibt gespeichert.
func (s *PeriodService) Commit(ctx context.Context, r PeriodCommitRequest) (PeriodCommitResult, error) {
	if err := requireHuman(ctx, "Die Festschreibung"); err != nil {
		return PeriodCommitResult{}, err
	}
	if err := domain.CheckDate("Stichtag", r.CutoffDate); err != nil {
		return PeriodCommitResult{}, err
	}
	var result PeriodCommitResult
	err := s.tx.RunInTx(ctx, func(ctx context.Context) (err error) {
		result.Commitment, result.Blocked, err = s.commit(ctx, r)
		return err
	})
	if err != nil {
		return PeriodCommitResult{}, err
	}
	return result, nil
}

// PreCommitCheck prüft vor der Festschreibung in derselben Transaktion.
// Die Registrierungsreihenfolge bestimmt den ersten Fehler; Netz ist untersagt.
type PreCommitCheck struct {
	Name  string
	Check func(context.Context, domain.FiscalYear, PeriodCommitRequest) error
}

// AddPreCommitChecks registriert Prüfungen beim Verdrahten des Mandanten.
func (s *PeriodService) AddPreCommitChecks(checks ...PreCommitCheck) {
	for _, check := range checks {
		if check.Name == "" || check.Check == nil {
			panic("ungültige Festschreibungsprüfung: " + check.Name)
		}
		for _, existing := range s.preCommitChecks {
			if existing.Name == check.Name {
				panic("doppelte Festschreibungsprüfung: " + check.Name)
			}
		}
		s.preCommitChecks = append(s.preCommitChecks, check)
	}
}

// PreCommitCheckNames nennt die Prüfungsreihenfolge.
func (s *PeriodService) PreCommitCheckNames() []string {
	names := make([]string, 0, len(s.preCommitChecks))
	for _, check := range s.preCommitChecks {
		names = append(names, check.Name)
	}
	return names
}

// commit legt Festschreibung und Buchungsstempel in derselben Transaktion an.
//
// Die Jahresfestschreibung darf denselben Stichtag tragen wie die des letzten
// Monats: dazwischen liegt die Abschlussperiode, in der die Abschlussbuchungen
// zum Bilanzstichtag entstehen. Erst die Jahresfestschreibung schließt sie.
func (s *PeriodService) commit(ctx context.Context, r PeriodCommitRequest) (*domain.Festschreibung, *CommitBlocked, error) {
	year := s.FiscalYearOf(ctx, r.CutoffDate)
	committed, err := s.festschreibungRepo.FindByFiscalYear(ctx, year)
	if err != nil {
		return nil, nil, err
	}
	latest := ""
	for _, rec := range committed {
		if rec.PeriodType == domain.PeriodTypeYear {
			return nil, nil, fmt.Errorf("das Geschäftsjahr %d ist bereits festgeschrieben (bis %s)", year, rec.CutoffDate)
		}
		if rec.CutoffDate > latest {
			latest = rec.CutoffDate
		}
	}
	if r.PeriodType == domain.PeriodTypeYear {
		fy, err := s.FiscalYear(ctx, year)
		if err != nil {
			return nil, nil, err
		}
		if r.CutoffDate != fy.EndDate {
			return nil, nil, fmt.Errorf("die Jahresfestschreibung gilt bis zum Bilanzstichtag %s, nicht bis %s",
				fy.EndDate, r.CutoffDate)
		}
	} else if latest != "" && r.CutoffDate <= latest {
		return nil, nil, fmt.Errorf("Zeitraum bis %s ist bereits festgeschrieben", latest)
	}
	if err := s.checkCommit(ctx, year, r); err != nil {
		var blocked *CommitBlocked
		if errors.As(err, &blocked) {
			return nil, blocked, nil
		}
		return nil, nil, err
	}

	chainHead := domain.GenesisHash
	last, err := s.journalRepo.GetLastEntry(ctx, year)
	if err != nil {
		return nil, nil, err
	}
	if last != nil {
		chainHead = last.EntryHash
	}
	count, err := s.journalRepo.Count(ctx, year)
	if err != nil {
		return nil, nil, err
	}

	// Die Systemzeit wird einmal genommen: Festschreibungszeitpunkt an den
	// Buchungen und Zeitpunkt der Festschreibung selbst sind derselbe.
	now := time.Now().UTC()
	rec := &domain.Festschreibung{
		FiscalYear:      year,
		PeriodType:      r.PeriodType,
		PeriodLabel:     r.PeriodLabel,
		CutoffDate:      r.CutoffDate,
		ChainHead:       chainHead,
		EntryCount:      int(count),
		TimestampStatus: "pending",
		AppVersion:      buildinfo.Version,
		Actor:           actor.Actor(),
		CreatedAt:       now,
	}
	if err := s.festschreibungRepo.Create(ctx, rec); err != nil {
		return nil, nil, fmt.Errorf("Festschreibung speichern: %w", err)
	}
	// Erst jetzt bekommen die Buchungen ihren Festschreibungszeitpunkt: er
	// verweist auf die Festschreibung, und die hat vorher keine Kennung.
	stamped, err := s.journalRepo.MarkCommitted(ctx, year, r.CutoffDate, rec.ID, now)
	if err != nil {
		return nil, nil, fmt.Errorf("Buchungen festschreiben: %w", err)
	}
	rec.EntriesStamped = int(stamped)
	if err := s.festschreibungRepo.Update(ctx, rec); err != nil {
		return nil, nil, err
	}

	details := fmt.Sprintf("Festschreibung %s (bis %s, %d Buchungen, davon %d neu festgeschrieben)",
		r.PeriodLabel, r.CutoffDate, count, rec.EntriesStamped)
	if reason := strings.TrimSpace(r.OverrideReason); reason != "" {
		details += fmt.Sprintf(". Blockierende Befunde übergangen mit der Begründung: %s", reason)
	}
	return rec, nil, s.auditRepo.Log(ctx, domain.AuditActionCommit, "FESTSCHREIBUNG", fmt.Sprintf("%d", rec.ID), details)
}

func (s *PeriodService) checkCommit(ctx context.Context, year int, r PeriodCommitRequest) error {
	fy, err := s.FiscalYear(ctx, year)
	if err != nil {
		return err
	}
	for _, check := range s.preCommitChecks {
		if err := check.Check(ctx, *fy, r); err != nil {
			return err
		}
	}
	return nil
}
