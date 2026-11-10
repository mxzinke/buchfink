package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Adopt schaltet Aufstellung, Feststellung oder Offenlegung einen Schritt
// weiter. Die Prüfungen der Abschlusszahlen bleiben beim Abschlussdienst.
func (s *PeriodService) Adopt(ctx context.Context, year int, status domain.FiscalYearStatus, date, note string) (*domain.FiscalYear, error) {
	if err := requireHuman(ctx, "Aufstellen, Feststellen und Offenlegen"); err != nil {
		return nil, err
	}
	return s.changeYear(ctx, func(ctx context.Context) (*domain.FiscalYear, error) {
		return s.setFiscalYearStatus(ctx, year, status, date, note)
	})
}

// Reopen nimmt die Feststellung mit protokollierter Begründung zurück.
// Die Festschreibungen und ihre Buchungsstempel bleiben erhalten.
func (s *PeriodService) Reopen(ctx context.Context, year int, reason string) (*domain.FiscalYear, error) {
	if err := requireHuman(ctx, "Das Zurücknehmen der Feststellung"); err != nil {
		return nil, err
	}
	return s.changeYear(ctx, func(ctx context.Context) (*domain.FiscalYear, error) {
		return s.reopenFiscalYear(ctx, year, reason)
	})
}

// nextStep benennt den nächsten Schritt und, falls er noch nicht gegangen werden
// kann, was ihm entgegensteht.
func (s *PeriodService) nextStep(fy *domain.FiscalYear, hasYearCommitment bool) (domain.FiscalYearStatus, bool, string) {
	switch fy.Status {
	case domain.FiscalYearOpen:
		return domain.FiscalYearPrepared, true, ""
	case domain.FiscalYearPrepared:
		if !hasYearCommitment {
			return domain.FiscalYearAdopted, false, fmt.Sprintf(
				"Für %d fehlt die Jahres-Festschreibung. Ein Abschluss, dessen Buchungen noch änderbar "+
					"sind, kann nicht festgestellt werden.", fy.Year)
		}
		return domain.FiscalYearAdopted, true, ""
	case domain.FiscalYearAdopted:
		return domain.FiscalYearDisclosed, true, ""
	}
	return "", false, ""
}

// yearCommitment sucht die Festschreibung des ganzen Jahres.
func (s *PeriodReader) yearCommitment(ctx context.Context, year int) (bool, string, error) {
	if s.festschreibungRepo == nil {
		return false, "", nil
	}
	records, err := s.festschreibungRepo.FindByFiscalYear(ctx, year)
	if err != nil {
		return false, "", fmt.Errorf("die Festschreibungen des Jahres %d konnten nicht gelesen werden: %w", year, err)
	}
	var latest string
	found := false
	for _, rec := range records {
		if rec.CutoffDate > latest {
			latest = rec.CutoffDate
		}
		if rec.PeriodType == "year" {
			found = true
		}
	}
	return found, latest, nil
}

func (s *PeriodService) setFiscalYearStatus(
	ctx context.Context, year int, status domain.FiscalYearStatus, date, note string,
) (*domain.FiscalYear, error) {
	if err := ensureSupportedCompany(ctx, s.settingsRepo); err != nil {
		return nil, err
	}
	fy, err := s.yearOf(ctx, year)
	if err != nil {
		return nil, err
	}
	// Der Stand vor dem Schritt, festgehalten bevor er überschrieben wird.
	before := *fy
	if !status.Valid() || status == domain.FiscalYearOpen {
		return nil, fmt.Errorf(
			"unbekannter Abschlussstand %q; ein Jahr wird über die Rücksetzung wieder geöffnet", status)
	}
	if status == fy.Status {
		return nil, fmt.Errorf("das Geschäftsjahr %d steht bereits auf %q", year, status.Label())
	}
	if !isNextStatus(fy.Status, status) {
		return nil, fmt.Errorf(
			"das Geschäftsjahr %d steht auf %q; der nächste Schritt ist %q und nicht %q",
			year, fy.Status.Label(), nextStatus(fy.Status).Label(), status.Label())
	}
	if len(date) != 10 {
		return nil, fmt.Errorf("das Datum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("%q ist kein Datum (erwartet JJJJ-MM-TT)", date)
	}
	if date < fy.EndDate {
		return nil, fmt.Errorf(
			"der %s liegt vor dem Ende des Geschäftsjahres am %s; ein Abschluss lässt sich nicht "+
				"aufstellen, bevor das Jahr vorbei ist", date, fy.EndDate)
	}

	if s.adoption == nil {
		return nil, fmt.Errorf("Abschlussprüfung ist nicht eingerichtet")
	}
	if err := s.adoption.EnsureLegalReserve(ctx, year); err != nil {
		return nil, err
	}
	if err := s.adoption.EnsureStatementReady(ctx, year); err != nil {
		return nil, err
	}
	switch status {
	case domain.FiscalYearPrepared:
		fy.PreparedOn = date
	case domain.FiscalYearAdopted:
		hasYear, _, err := s.yearCommitment(ctx, year)
		if err != nil {
			return nil, err
		}
		// Die Feststellung ohne Festschreibung wäre ein Beschluss über Zahlen,
		// die sich danach noch ändern lassen.
		if !hasYear {
			return nil, fmt.Errorf(
				"für %d liegt keine Jahres-Festschreibung vor. Das Geschäftsjahr ist zuerst festzuschreiben – "+
					"erst dann steht fest, worüber die Gesellschafter beschließen (§ 42a Abs. 2 GmbHG)", year)
		}
		if date < fy.PreparedOn {
			return nil, fmt.Errorf(
				"die Feststellung am %s läge vor der Aufstellung am %s", date, fy.PreparedOn)
		}
		fy.AdoptedOn = date
		fy.AdoptionNote = strings.TrimSpace(note)
	case domain.FiscalYearDisclosed:
		if date < fy.AdoptedOn {
			return nil, fmt.Errorf(
				"die Offenlegung am %s läge vor der Feststellung am %s", date, fy.AdoptedOn)
		}
		if strings.TrimSpace(note) == "" {
			return nil, fmt.Errorf("bitte den Übermittlungsnachweis oder die Auftragsnummer zur Offenlegung angeben; Buchfink übermittelt den Abschluss nicht selbst")
		}
		fy.DisclosureNote = strings.TrimSpace(note)
		fy.DisclosedOn = date
	}
	fy.Status = status

	if err := fy.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return nil, fmt.Errorf("der Abschlussstand konnte nicht gespeichert werden: %w", err)
	}

	details := fmt.Sprintf("Geschäftsjahr %d: %s am %s", year, status.Label(), date)
	if n := strings.TrimSpace(note); n != "" {
		details += " (" + n + ")"
	}
	if err := s.logChange(ctx, domain.AuditActionUpdate, year, details, &before, fy); err != nil {
		return nil, err
	}
	return fy, nil
}

func (s *PeriodService) reopenFiscalYear(ctx context.Context, year int, reason string) (*domain.FiscalYear, error) {
	fy, err := s.yearOf(ctx, year)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("für die Rücksetzung des Jahresabschlusses ist ein Grund anzugeben")
	}
	if fy.Status == domain.FiscalYearOpen || fy.Status == domain.FiscalYearPrepared {
		return nil, fmt.Errorf(
			"das Geschäftsjahr %d ist nicht festgestellt (Stand: %s); es gibt nichts zurückzusetzen",
			year, fy.Status.Label())
	}

	previous := fy.Status
	before := *fy
	fy.Status = domain.FiscalYearPrepared
	fy.AdoptedOn = ""
	fy.AdoptionNote = ""
	fy.DisclosedOn = ""
	fy.DisclosureNote = ""

	if err := fy.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return nil, fmt.Errorf("die Rücksetzung konnte nicht gespeichert werden: %w", err)
	}

	if err := s.logChange(ctx, domain.AuditActionUpdate, year, fmt.Sprintf(
		"Geschäftsjahr %d von %q auf %q zurückgesetzt (Grund: %s)",
		year, previous.Label(), domain.FiscalYearPrepared.Label(), strings.TrimSpace(reason)), &before, fy); err != nil {
		return nil, err
	}
	return fy, nil
}

func isNextStatus(current, target domain.FiscalYearStatus) bool {
	return nextStatus(current) == target
}

func nextStatus(current domain.FiscalYearStatus) domain.FiscalYearStatus {
	switch current {
	case domain.FiscalYearOpen:
		return domain.FiscalYearPrepared
	case domain.FiscalYearPrepared:
		return domain.FiscalYearAdopted
	case domain.FiscalYearAdopted:
		return domain.FiscalYearDisclosed
	}
	return ""
}

func (s *PeriodService) changeYear(ctx context.Context, change func(context.Context) (*domain.FiscalYear, error)) (*domain.FiscalYear, error) {
	var fy *domain.FiscalYear
	err := s.tx.RunInTx(ctx, func(ctx context.Context) (err error) {
		fy, err = change(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return fy, nil
}
