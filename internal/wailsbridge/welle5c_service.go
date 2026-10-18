package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
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

// -------------------------------------------------------------------------
// Bestätigung der USt-IdNr.
// -------------------------------------------------------------------------

// GetVatIDStatus beantwortet ohne Netzaufruf, ob eine frische Bestätigung
// vorliegt.
func (b *BuchfinkBridge) GetVatIDStatus(contactID uint) (*service.VatIDStatus, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.VatIDStatus, error) {
		contact, err := s.Repos.Contacts.FindByID(ctx, contactID)
		if err != nil {
			return nil, fmt.Errorf("der Geschäftspartner konnte nicht geladen werden: %w", err)
		}
		return s.VatIDs.Status(ctx, contact)
	})
}

// -------------------------------------------------------------------------
// Nicht abziehbare Betriebsausgaben
// -------------------------------------------------------------------------

// GetNonDeductibleReport ist der Bericht eines Geschäftsjahres.
func (b *BuchfinkBridge) GetNonDeductibleReport(year int) (*service.NonDeductibleReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.NonDeductibleReport, error) {
		return s.Gifts.NonDeductibleReport(ctx, year)
	})
}

// GetExchangeRate liefert den Kurs eines Tages.
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
// nennt es, wo einer fehlt. Die Vorschau ganz zu sperren wäre die schlechteste
// Antwort gewesen — sie ist eine Auswertung, und der Prüfermodus ist der Modus,
// in dem ausgewertet wird.
func (b *BuchfinkBridge) PreviewCurrencyValuation(year int) (*service.ForeignCurrencyValuation, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.ForeignCurrencyValuation, error) {
		return b.currencyService(s).PreviewCurrencyValuation(ctx, year)
	})
}

// GetAfaRules liefert die Abschreibungsregeln aus der eingebetteten Ressource —
// dieselbe Datei, aus der gerechnet wird.
func (b *BuchfinkBridge) GetAfaRules() accounting.AfARules {
	return accounting.AfARuleSet()
}

// -------------------------------------------------------------------------
// Anlagen
// -------------------------------------------------------------------------

// GetWriteUpReport listet die Anlagegüter, bei denen die Wertaufholung zu prüfen
// ist.
func (b *BuchfinkBridge) GetWriteUpReport(year int) (*service.WriteUpReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.WriteUpReport, error) {
		return s.Assets.WriteUpReport(ctx, year)
	})
}

// GetPoolConsistencyReport prüft die Einheitlichkeit des Wahlrechts nach
// § 6 Abs. 2a Satz 5 EStG.
func (b *BuchfinkBridge) GetPoolConsistencyReport(year int) (*service.PoolConsistencyReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.PoolConsistencyReport, error) {
		return s.Assets.PoolConsistency(ctx, year)
	})
}
