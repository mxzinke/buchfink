package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// GetFiscalYears returns the fiscal years as entities: period, Rumpfjahr flag
// and Abschlussstand.
func (b *BuchfinkBridge) GetFiscalYears() ([]domain.FiscalYear, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingSvc == nil {
		return []domain.FiscalYear{}, nil
	}
	return b.closingSvc.FiscalYears(context.Background())
}

// CreateFiscalYear legt das Geschäftsjahr an und schaltet auf es um.
//
// Bis hierher legte diese Methode nichts an, sondern setzte nur den Filter: das
// Geschäftsjahr war eine Zahl an der Buchung und sonst nichts. Angelegt wird es
// jetzt als Entität mit Zeitraum — das Folgejahr beginnt am Tag nach dem Ende
// des Vorjahres, auch nach einem Rumpfgeschäftsjahr.
func (b *BuchfinkBridge) CreateFiscalYear(year int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closingSvc != nil {
		if _, err := b.closingSvc.CreateFiscalYear(context.Background(), year); err != nil {
			return err
		}
	}
	b.setFiscalYearLocked(year)
	return nil
}

// GetCarryForwardPreview zeigt den Vortragsstand ins Zieljahr: je Konto
// Schlusssaldo des Vorjahres, bereits vorgetragener Wert und Differenz.
func (b *BuchfinkBridge) GetCarryForwardPreview(toYear int) (*service.CarryForwardPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.CarryForwardState(context.Background(), toYear)
}

// CarryForward bucht den Saldenvortrag ins Zieljahr; ein erneuter Lauf nimmt den
// bestehenden Vortrag per Generalumkehr zurück und bucht neu.
func (b *BuchfinkBridge) CarryForward(toYear int) ([]domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.CarryForward(context.Background(), toYear)
}

// GetClosingState liefert den Abschlussstand eines Geschäftsjahres samt der
// Frage, ob die Jahres-Festschreibung vorliegt und der Vortrag ins Folgejahr
// aktuell ist.
func (b *BuchfinkBridge) GetClosingState(year int) (*service.ClosingState, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.ClosingStateFor(context.Background(), year)
}

// SetFiscalYearStatus schaltet den Abschluss einen Schritt weiter: aufgestellt,
// festgestellt, offengelegt — jeweils mit Datum.
func (b *BuchfinkBridge) SetFiscalYearStatus(year int, status, date, note string) (*domain.FiscalYear, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.SetFiscalYearStatus(
		context.Background(), year, domain.FiscalYearStatus(status), date, note)
}

// ReopenFiscalYear nimmt die Feststellung mit Angabe eines Grundes zurück.
func (b *BuchfinkBridge) ReopenFiscalYear(year int, reason string) (*domain.FiscalYear, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.ReopenFiscalYear(context.Background(), year, reason)
}
