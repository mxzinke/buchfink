package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// GetExchangeRate liefert den Kurs eines Tages und speichert einen fehlenden
// Kurs in der Historie. Im Prüfermodus liest der Kursdienst nur gespeicherte
// Kurse; dasselbe gilt für PreviewCurrencyValuation.
func (b *BuchfinkBridge) GetExchangeRate(currency, date string) (*domain.ExchangeRate, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.ExchangeRate, error) {
		return b.currencyService(s).RateAt(ctx, currency, date)
	})
}

// -------------------------------------------------------------------------
// Fremdwährung
// -------------------------------------------------------------------------

// currencyService ist der Kursdienst für eine Auswertung: im Prüfermodus der
// nur lesende, der keinen fehlenden Kurs holt.
func (b *BuchfinkBridge) currencyService(s *core.Services) *service.CurrencyService {
	if b.host.Config().ReadOnly {
		return s.Currency.ReadOnly()
	}
	return s.Currency
}

// GetVatExchangeRates liefert die Umsatzsteuer-Durchschnittskurse eines
// Zeitraums.
func (b *BuchfinkBridge) GetVatExchangeRates(from, to string) ([]domain.VatExchangeRate, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.VatExchangeRate, error) {
		return s.Currency.VatRates(ctx, from, to)
	})
}

// PreviewCurrencyValuation rechnet die Stichtagsbewertung, ohne sie zu buchen.
//
// Wie GetExchangeRate: im laufenden Betrieb holt sie den fehlenden Stichtagskurs
// und legt ihn ab, im Prüfermodus rechnet sie mit den gespeicherten Kursen und
// meldet fehlende Kurse.
func (b *BuchfinkBridge) PreviewCurrencyValuation(year int) (*service.ForeignCurrencyValuation, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.ForeignCurrencyValuation, error) {
		return b.currencyService(s).PreviewCurrencyValuation(ctx, year)
	})
}
