package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// GetFiscalYears returns the fiscal years as entities: period, Rumpfjahr flag
// and Abschlussstand.
func (b *BuchfinkBridge) GetFiscalYears() ([]domain.FiscalYear, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.FiscalYear, error) {
		return s.Closing.FiscalYears(ctx)
	})
}

// GetFiscalYearCandidates liefert das kommende und das vergangene Geschäftsjahr
// mit dem Grund, falls sich eines nicht anlegen lässt.
func (b *BuchfinkBridge) GetFiscalYearCandidates() (*service.FiscalYearCandidates, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.FiscalYearCandidates, error) {
		return s.Closing.FiscalYearCandidates(ctx)
	})
}

// CreateFiscalYear legt das Geschäftsjahr an und schaltet auf es um.
func (b *BuchfinkBridge) CreateFiscalYear(year int) error {
	if err := b.change(func(ctx context.Context, s *core.Services) error {
		_, err := s.Closing.CreateFiscalYear(ctx, year)
		return err
	}); err != nil {
		return err
	}
	return b.SetFiscalYear(year)
}

// GetCarryForwardPreview zeigt den Vortragsstand ins Zieljahr: je Konto
// Schlusssaldo des Vorjahres, vorgetragener Wert und Differenz.
func (b *BuchfinkBridge) GetCarryForwardPreview(toYear int) (*service.CarryForwardPreview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.CarryForwardPreview, error) {
		return s.Closing.CarryForwardState(ctx, toYear)
	})
}

// CarryForward bucht den Saldenvortrag ins Zieljahr; ein erneuter Lauf nimmt den
// bestehenden Vortrag per Generalumkehr zurück und bucht neu.
func (b *BuchfinkBridge) CarryForward(toYear int) ([]domain.JournalEntry, error) {
	return emptyList(write(b, func(ctx context.Context, s *core.Services) ([]domain.JournalEntry, error) {
		return s.Closing.CarryForward(ctx, toYear)
	}))
}

// GetClosingState liefert den Abschlussstand eines Geschäftsjahres samt der
// Frage, ob die Jahres-Festschreibung vorliegt und der Vortrag ins Folgejahr
// aktuell ist.
func (b *BuchfinkBridge) GetClosingState(year int) (*service.ClosingState, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.ClosingState, error) {
		return s.Closing.ClosingStateFor(ctx, year)
	})
}

// SetFiscalYearStatus schaltet den Abschluss einen Schritt weiter: aufgestellt,
// festgestellt, offengelegt — jeweils mit Datum.
func (b *BuchfinkBridge) SetFiscalYearStatus(year int, status, date, note string) (*domain.FiscalYear, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.FiscalYear, error) {
		return s.Closing.SetFiscalYearStatus(
			ctx, year, domain.FiscalYearStatus(status), date, note)
	}, core.AppOnly())
}

// ReopenFiscalYear nimmt die Feststellung mit Angabe eines Grundes zurück.
func (b *BuchfinkBridge) ReopenFiscalYear(year int, reason string) (*domain.FiscalYear, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.FiscalYear, error) {
		return s.Closing.ReopenFiscalYear(ctx, year, reason)
	}, core.AppOnly())
}
