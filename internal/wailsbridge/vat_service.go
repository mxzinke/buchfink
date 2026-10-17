package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
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
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.VatPeriodStatus, error) {
		return s.VatReturns.Periods(ctx, year)
	})
}

// GetVatReturn rechnet das Kennziffernblatt eines Zeitraums neu, ohne es zu
// speichern. Der Entwurf ist immer der heutige Stand des Journals.
func (b *BuchfinkBridge) GetVatReturn(periodKey string) (*domain.VatReturn, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.VatReturn, error) {
		return s.VatReturns.Draft(ctx, periodKey)
	})
}

// SaveVatReturn legt das Blatt als Entwurf ab.
func (b *BuchfinkBridge) SaveVatReturn(periodKey string) (*domain.VatReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.VatReturn, error) {
		return s.VatReturns.Save(ctx, periodKey)
	})
}

// GetVatReturns liefert die gespeicherten Voranmeldungen eines Jahres.
func (b *BuchfinkBridge) GetVatReturns(year int) ([]domain.VatReturn, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.VatReturn, error) {
		return s.VatReturns.List(ctx, year)
	})
}

// ConfirmVatReturnSubmitted bestätigt die Übermittlung in Mein ELSTER. Der
// Zeitraum muss festgeschrieben sein, und ohne Transferticket gibt es keine
// Bestätigung.
func (b *BuchfinkBridge) ConfirmVatReturnSubmitted(id uint, date, ticket, note string) (*domain.VatReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.VatReturn, error) {
		return s.VatReturns.ConfirmSubmitted(ctx, id, date, ticket, note)
	}, core.AppOnly())
}

// CreateVatCorrection legt eine berichtigte Voranmeldung an (Kennziffer 10).
func (b *BuchfinkBridge) CreateVatCorrection(periodKey string) (*domain.VatReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.VatReturn, error) {
		return s.VatReturns.CreateCorrection(ctx, periodKey)
	})
}

// ExportVatReturnCSV liefert das Kennziffernblatt als Text (Kennziffer;Wert).
func (b *BuchfinkBridge) ExportVatReturnCSV(id uint) (string, error) {
	return read(b, func(ctx context.Context, s *core.Services) (string, error) {
		return s.VatReturns.ExportCSV(ctx, id)
	})
}

// GetSpecialPrepaymentSuggestion rechnet die Sondervorauszahlung aus den
// übermittelten Voranmeldungen des Vorjahres (§ 47 Abs. 1 UStDV).
//
// Ein Vorschlag, kein Wert: angemeldet und gezahlt wird die Sondervorauszahlung
// außerhalb von Buchfink, und in den Einstellungen steht, was der Anwender
// angemeldet hat.
func (b *BuchfinkBridge) GetSpecialPrepaymentSuggestion(year int) (*service.SpecialPrepaymentSuggestion, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.SpecialPrepaymentSuggestion, error) {
		return s.VatReturns.SuggestedSpecialPrepayment(ctx, year)
	})
}

// -------------------------------------------------------------
// Zusammenfassende Meldung
// -------------------------------------------------------------

// GetZMPeriods liefert die Meldezeiträume eines Jahres. Ihre Länge folgt aus den
// Umsätzen (§ 18a Abs. 1 UStG) und nicht aus einer Einstellung.
func (b *BuchfinkBridge) GetZMPeriods(year int) ([]service.ZMPeriodStatus, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.ZMPeriodStatus, error) {
		return s.ZM.Periods(ctx, year)
	})
}

// GetZMReturn rechnet die Meldung eines Zeitraums neu, ohne sie zu speichern.
func (b *BuchfinkBridge) GetZMReturn(periodKey string) (*domain.ZMReturn, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.ZMReturn, error) {
		return s.ZM.Draft(ctx, periodKey)
	})
}

// SaveZMReturn legt die Meldung als Entwurf ab.
func (b *BuchfinkBridge) SaveZMReturn(periodKey string) (*domain.ZMReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.ZMReturn, error) {
		return s.ZM.Save(ctx, periodKey)
	})
}

// GetZMReturns liefert die gespeicherten Meldungen eines Jahres.
func (b *BuchfinkBridge) GetZMReturns(year int) ([]domain.ZMReturn, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.ZMReturn, error) {
		return s.ZM.List(ctx, year)
	})
}

// ConfirmZMSubmitted bestätigt die Übermittlung an das Bundeszentralamt.
func (b *BuchfinkBridge) ConfirmZMSubmitted(id uint, date, ticket, note string) (*domain.ZMReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.ZMReturn, error) {
		return s.ZM.ConfirmSubmitted(ctx, id, date, ticket, note)
	}, core.AppOnly())
}

// CreateZMCorrection legt eine berichtigte Zusammenfassende Meldung an.
func (b *BuchfinkBridge) CreateZMCorrection(periodKey string) (*domain.ZMReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.ZMReturn, error) {
		return s.ZM.CreateCorrection(ctx, periodKey)
	})
}

// ExportZMCSV liefert die Meldung im Spaltenformat des BZSt-Online-Portals.
func (b *BuchfinkBridge) ExportZMCSV(id uint) (string, error) {
	return read(b, func(ctx context.Context, s *core.Services) (string, error) {
		return s.ZM.ExportCSV(ctx, id)
	})
}
