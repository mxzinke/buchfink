package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Umsatzsteuer-Voranmeldung und Zusammenfassende Meldung.
//
// Buchfink übermittelt nicht selbst. Die Bridge liefert das Kennziffernblatt und
// die Meldedatei, der Anwender gibt beides in Mein ELSTER bzw. im
// BZSt-Online-Portal ein und bestätigt die Übermittlung hier mit dem
// Transferticket — das ist das Übermittlungsprotokoll.

// GetVatPeriods liefert die Voranmeldungszeiträume eines Jahres mit Fälligkeit,
// Festschreibungsstand und Stand der Anmeldung.
func (b *BuchfinkBridge) GetVatPeriods(year int) ([]service.VatPeriodStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatReturnSvc == nil {
		return []service.VatPeriodStatus{}, nil
	}
	return b.vatReturnSvc.Periods(context.Background(), year)
}

// GetVatReturn rechnet das Kennziffernblatt eines Zeitraums neu, ohne es zu
// speichern. Der Entwurf ist immer der heutige Stand des Journals.
func (b *BuchfinkBridge) GetVatReturn(periodKey string) (*domain.VatReturn, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatReturnSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatReturnSvc.Draft(context.Background(), periodKey)
}

// SaveVatReturn legt das Blatt als Entwurf ab.
func (b *BuchfinkBridge) SaveVatReturn(periodKey string) (*domain.VatReturn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.vatReturnSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatReturnSvc.Save(context.Background(), periodKey)
}

// GetVatReturns liefert die gespeicherten Voranmeldungen eines Jahres.
func (b *BuchfinkBridge) GetVatReturns(year int) ([]domain.VatReturn, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatReturnSvc == nil {
		return []domain.VatReturn{}, nil
	}
	return b.vatReturnSvc.List(context.Background(), year)
}

// ConfirmVatReturnSubmitted bestätigt die Übermittlung in Mein ELSTER. Der
// Zeitraum muss festgeschrieben sein, und ohne Transferticket gibt es keine
// Bestätigung.
func (b *BuchfinkBridge) ConfirmVatReturnSubmitted(id uint, date, ticket, note string) (*domain.VatReturn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.vatReturnSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatReturnSvc.ConfirmSubmitted(context.Background(), id, date, ticket, note)
}

// CreateVatCorrection legt eine berichtigte Voranmeldung an (Kennziffer 10).
func (b *BuchfinkBridge) CreateVatCorrection(periodKey string) (*domain.VatReturn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.vatReturnSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatReturnSvc.CreateCorrection(context.Background(), periodKey)
}

// ExportVatReturnCSV liefert das Kennziffernblatt als Text (Kennziffer;Wert).
func (b *BuchfinkBridge) ExportVatReturnCSV(id uint) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatReturnSvc == nil {
		return "", fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatReturnSvc.ExportCSV(context.Background(), id)
}

// GetSpecialPrepaymentSuggestion rechnet die Sondervorauszahlung aus den
// übermittelten Voranmeldungen des Vorjahres (§ 47 Abs. 1 UStDV).
//
// Ein Vorschlag, kein Wert: angemeldet und gezahlt wird die Sondervorauszahlung
// außerhalb von Buchfink, und in den Einstellungen steht, was der Anwender
// angemeldet hat.
func (b *BuchfinkBridge) GetSpecialPrepaymentSuggestion(year int) (*service.SpecialPrepaymentSuggestion, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatReturnSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatReturnSvc.SuggestedSpecialPrepayment(context.Background(), year)
}

// -------------------------------------------------------------
// Zusammenfassende Meldung
// -------------------------------------------------------------

// GetZMPeriods liefert die Meldezeiträume eines Jahres. Ihre Länge folgt aus den
// Umsätzen (§ 18a Abs. 1 UStG) und nicht aus einer Einstellung.
func (b *BuchfinkBridge) GetZMPeriods(year int) ([]service.ZMPeriodStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.zmSvc == nil {
		return []service.ZMPeriodStatus{}, nil
	}
	return b.zmSvc.Periods(context.Background(), year)
}

// GetZMReturn rechnet die Meldung eines Zeitraums neu, ohne sie zu speichern.
func (b *BuchfinkBridge) GetZMReturn(periodKey string) (*domain.ZMReturn, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.zmSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.zmSvc.Draft(context.Background(), periodKey)
}

// SaveZMReturn legt die Meldung als Entwurf ab.
func (b *BuchfinkBridge) SaveZMReturn(periodKey string) (*domain.ZMReturn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.zmSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.zmSvc.Save(context.Background(), periodKey)
}

// GetZMReturns liefert die gespeicherten Meldungen eines Jahres.
func (b *BuchfinkBridge) GetZMReturns(year int) ([]domain.ZMReturn, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.zmSvc == nil {
		return []domain.ZMReturn{}, nil
	}
	return b.zmSvc.List(context.Background(), year)
}

// ConfirmZMSubmitted bestätigt die Übermittlung an das Bundeszentralamt.
func (b *BuchfinkBridge) ConfirmZMSubmitted(id uint, date, ticket, note string) (*domain.ZMReturn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.zmSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.zmSvc.ConfirmSubmitted(context.Background(), id, date, ticket, note)
}

// CreateZMCorrection legt eine berichtigte Zusammenfassende Meldung an.
func (b *BuchfinkBridge) CreateZMCorrection(periodKey string) (*domain.ZMReturn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.zmSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.zmSvc.CreateCorrection(context.Background(), periodKey)
}

// ExportZMCSV liefert die Meldung im Spaltenformat des BZSt-Online-Portals.
func (b *BuchfinkBridge) ExportZMCSV(id uint) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.zmSvc == nil {
		return "", fmt.Errorf("kein aktiver Mandant")
	}
	return b.zmSvc.ExportCSV(context.Background(), id)
}
