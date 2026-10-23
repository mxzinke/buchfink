package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// CommitPeriod schreibt einen Zeitraum bis einschließlich cutoffDate fest. Der
// Kern legt die Festschreibung an, holt den Zeitstempel ohne Schreibsperre und
// trägt ihn nach (core.Tenant.CommitPeriod). Nur in der App.
func (b *BuchfinkBridge) CommitPeriod(periodType, periodLabel, cutoffDate, overrideReason string) (*domain.Festschreibung, error) {
	return operate(b, func(ctx context.Context, t *core.Tenant) (*domain.Festschreibung, error) {
		return t.CommitPeriod(ctx, core.CommitRequest{
			PeriodType: periodType, PeriodLabel: periodLabel,
			CutoffDate: cutoffDate, OverrideReason: overrideReason,
		})
	})
}

// GetFestschreibungen liefert die Festschreibungen des angesehenen
// Geschäftsjahres.
func (b *BuchfinkBridge) GetFestschreibungen() ([]domain.Festschreibung, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Festschreibung, error) {
		return s.Repos.Festschreibung.FindByFiscalYear(ctx, b.viewYear())
	})
}

// VerifyFestschreibung prüft eine Festschreibung ohne Netz: Zeitstempel und
// Kettenkopf.
func (b *BuchfinkBridge) VerifyFestschreibung(id uint) (*domain.FestschreibungVerification, error) {
	return inspect(b, func(ctx context.Context, t *core.Tenant) (*domain.FestschreibungVerification, error) {
		return t.VerifyCommitment(ctx, id)
	})
}
