package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// InputTaxPeriodDifference nennt die abweichenden Voranmeldungszeiträume.
type InputTaxPeriodDifference struct{ Document, Received string }

// InputTaxPeriodDifference vergleicht Eingang und Belegdatum nach der Zeitraumart des Mandanten.
func (s *ReceiptService) InputTaxPeriodDifference(ctx context.Context, r *domain.Receipt) *InputTaxPeriodDifference {
	if s.settingsRepo == nil || r.DocumentDate == "" || r.ReceivedAt == "" || r.ReceivedAt <= r.DocumentDate {
		return nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return nil
	}
	kind := domain.VatPeriodType(cfg.VatPeriod)
	doc, err1 := accounting.VatPeriodOf(r.DocumentDate, kind)
	got, err2 := accounting.VatPeriodOf(r.ReceivedAt, kind)
	if err1 != nil || err2 != nil || doc.Key == got.Key {
		return nil
	}
	return &InputTaxPeriodDifference{Document: doc.Key, Received: got.Key}
}
