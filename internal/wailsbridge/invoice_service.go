package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// RegenerateInvoiceDocument holt ein fehlendes Rechnungsdokument nach.
func (b *BuchfinkBridge) RegenerateInvoiceDocument(invoiceID uint) (*domain.Invoice, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.RegenerateDocument(ctx, invoiceID)
	})
}

// GetInvoiceNumberGaps liefert den Lückenbericht des Rechnungsnummernkreises.
func (b *BuchfinkBridge) GetInvoiceNumberGaps(year int) (*domain.NumberGapReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*domain.NumberGapReport, error) {
		return s.Invoices.NumberGaps(ctx, year)
	})
}

// RecordInvoiceNumberGapReason dokumentiert, warum eine Nummer keine Rechnung
// hat. Die Betriebsprüfung fragt danach, und eine Antwort im Kopf des
// Geschäftsführers ist keine.
func (b *BuchfinkBridge) RecordInvoiceNumberGapReason(year int, sequence int64, reason string, detail string) error {
	year = b.yearOr(year)
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Invoices.RecordNumberGapReason(
			ctx, year, sequence, domain.NumberGapReason(reason), detail)
	})
}

// GetUnitCodes liefert die Mengeneinheiten nach UN/ECE Rec. 20, die eine
// Rechnungsposition haben kann (BT-130).
func (b *BuchfinkBridge) GetUnitCodes() []domain.UnitCode {
	return domain.UnitCodes()
}

// GetEInvoiceProfiles liefert die Zielformate, in denen eine Rechnung
// ausgestellt werden kann.
func (b *BuchfinkBridge) GetEInvoiceProfiles() []domain.EInvoiceProfileInfo {
	return domain.EInvoiceProfiles()
}

// GetNumberGapReasons liefert die Gründe, mit denen eine Lücke im
// Nummernkreis begründet werden kann.
func (b *BuchfinkBridge) GetNumberGapReasons() []domain.NumberGapReasonOption {
	return domain.NumberGapReasons()
}

// RefundAdvance bucht die bestätigte Rückzahlung einer Anzahlung.
func (b *BuchfinkBridge) RefundAdvance(req service.RefundAdvanceRequest) (*domain.AdvanceItem, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.AdvanceItem, error) {
		return s.Invoices.RefundAdvance(ctx, req)
	})
}
