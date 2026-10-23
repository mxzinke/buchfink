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

// -------------------------------------------------------------
// Zusammenfassende Meldung
// -------------------------------------------------------------

// GetZMPeriods liefert die Meldezeiträume eines Jahres. Ihre Länge folgt aus den
// Umsätzen (§ 18a Abs. 1 UStG) und nicht aus einer Einstellung.
func (b *BuchfinkBridge) GetZMPeriods(year int) ([]service.ZMPeriodStatus, error) {
	year = b.yearOr(year)
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

// ConfirmZMSubmitted bestätigt die Übermittlung an das Bundeszentralamt.
func (b *BuchfinkBridge) ConfirmZMSubmitted(id uint, date, ticket, note string) (*domain.ZMReturn, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.ZMReturn, error) {
		return s.ZM.ConfirmSubmitted(ctx, id, date, ticket, note)
	}, core.AppOnly())
}

// --- Umsatzsteuer ---------------------------------------------------------

// GetVatPeriodProposal liefert den Voranmeldungszeitraum, der sich aus der
// Steuer des Vorjahres ergibt (§ 18 Abs. 2 UStG).
func (b *BuchfinkBridge) GetVatPeriodProposal(year int) (*service.VatPeriodProposal, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.VatPeriodProposal, error) {
		return s.VatReturns.SuggestPeriodType(ctx, year)
	})
}
