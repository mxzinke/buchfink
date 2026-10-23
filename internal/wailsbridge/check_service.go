package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Prüfberichte und Fristen.
//
// Der Prüflauf ist der Bericht vor der Festschreibung: er sagt, was danach nicht
// mehr zu ändern wäre. Die Fristen kommen aus denselben Daten — was übermittelt
// oder festgeschrieben ist, ist erledigt, und niemand hakt es zusätzlich ab.

// RunChecks führt einen Prüflauf bis zum Stichtag aus und gibt ihn zurück, ohne
// ihn zu speichern.
//
// Gespeichert wird der Lauf, den `CommitPeriod` vor der Festschreibung ausführt:
// an ihm hängt die Festschreibung, und an ihm steht die Begründung für ein
// Übergehen. Würde auch die Vorschau abgelegt, stünden je Festschreibung zwei
// Läufe im Protokoll — und die Begründung an dem, der nichts festgeschrieben hat.
//
// `periodType` ist der Anlass ("month", "quarter", "year"). Er schaltet die
// Regeln zu, die nur vor der Jahresfestschreibung gelten; ohne ihn zeigte die
// Vorschau weniger Befunde, als die Festschreibung danach verlangt.
func (b *BuchfinkBridge) RunChecks(cutoffDate, periodType string) (*domain.CheckRun, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.CheckRun, error) {
		return s.Checks.Preview(ctx, service.CheckRequest{
			CutoffDate: cutoffDate,
			PeriodType: periodType,
		})
	})
}

// GetCheckRuns liefert die gespeicherten Prüfläufe eines Jahres, der jüngste
// zuerst.
func (b *BuchfinkBridge) GetCheckRuns(year int) ([]domain.CheckRun, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.CheckRun, error) {
		return s.Checks.Runs(ctx, year)
	})
}

// GetDeadlines liefert die Pflichten eines Geschäftsjahres aus dem
// Pflichtenkalender in der Form der Fristenseite.
func (b *BuchfinkBridge) GetDeadlines(year int) ([]domain.Deadline, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Deadline, error) {
		return s.Obligations.Deadlines(ctx, year)
	})
}
