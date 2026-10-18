package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bridge der steuerlichen Nebenpflichten (Welle 5c).
//
// GetExchangeRate und PreviewCurrencyValuation holen einen Kurs, wenn keiner
// gespeichert ist, und legen ihn in der Historie ab — auch als Lesevorgang
// (repository.RunReadOnly). Ein Kurs, der geholt und nicht behalten würde,
// wäre bei der nächsten Frage ein anderer. Im Prüfermodus rechnen beide über
// den nur lesenden Kursdienst, der nichts holt und nichts ablegt.

// -------------------------------------------------------------------------
// Vorsteuerberichtigung nach § 15a UStG
// -------------------------------------------------------------------------

// GetInputTaxCorrections liefert das Verzeichnis mit Blick auf ein
// Geschäftsjahr.
func (b *BuchfinkBridge) GetInputTaxCorrections(year int) (*service.InputTaxCorrectionYear, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.InputTaxCorrectionYear, error) {
		return s.InputTax.Year(ctx, year)
	})
}
