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

// RegisterInputTaxCorrection nimmt ein Wirtschaftsgut ins Verzeichnis auf.
func (b *BuchfinkBridge) RegisterInputTaxCorrection(
	req service.RegisterInputTaxRequest,
) (*domain.InputTaxCorrection, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.InputTaxCorrection, error) {
		return s.InputTax.Register(ctx, req)
	})
}

// CloseInputTaxCorrection schließt einen Eintrag vorzeitig ab.
func (b *BuchfinkBridge) CloseInputTaxCorrection(
	id uint, reason, date string,
) (*domain.InputTaxCorrection, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.InputTaxCorrection, error) {
		return s.InputTax.Close(ctx, id, reason, date)
	})
}

// SaveInputTaxUsage bestätigt oder ändert den Verwendungsanteil eines Jahres.
func (b *BuchfinkBridge) SaveInputTaxUsage(
	req service.SaveInputTaxUsageRequest,
) (*service.InputTaxCorrectionYear, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return write(b, func(ctx context.Context, s *core.Services) (*service.InputTaxCorrectionYear, error) {
		return s.InputTax.SaveUsage(ctx, req)
	})
}

// BookInputTaxCorrection bucht die Berichtigungen eines Geschäftsjahres.
func (b *BuchfinkBridge) BookInputTaxCorrection(year int) (*service.InputTaxCorrectionYear, error) {
	year = b.yearOr(year)
	return write(b, func(ctx context.Context, s *core.Services) (*service.InputTaxCorrectionYear, error) {
		return s.InputTax.BookYear(ctx, year)
	})
}

// -------------------------------------------------------------------------
// Bestätigung der USt-IdNr.
// -------------------------------------------------------------------------

// GetVatIDChecks liefert den Verlauf der Bestätigungsabfragen eines Kontakts.
func (b *BuchfinkBridge) GetVatIDChecks(contactID uint) ([]domain.VatIDCheck, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.VatIDCheck, error) {
		return s.VatIDs.Checks(ctx, contactID)
	})
}

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

// CheckVatID führt die qualifizierte Bestätigungsanfrage aus und hebt das
// Ergebnis auf.
func (b *BuchfinkBridge) CheckVatID(contactID uint) (*domain.VatIDCheck, error) {
	return operate(b, func(ctx context.Context, t *core.Tenant) (*domain.VatIDCheck, error) {
		return t.CheckVatID(ctx, contactID)
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

// RebookGiftsForRecipient bucht die noch abziehbar stehenden Geschenke eines
// Empfängers auf das nicht abziehbare Konto um.
func (b *BuchfinkBridge) RebookGiftsForRecipient(req service.RebookGiftsRequest) (*service.GiftRebooking, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return write(b, func(ctx context.Context, s *core.Services) (*service.GiftRebooking, error) {
		return s.Gifts.RebookGiftsForRecipient(ctx, req)
	})
}

// -------------------------------------------------------------------------
// Fremdwährung
// -------------------------------------------------------------------------

// GetExchangeRate liefert den Kurs eines Tages.
func (b *BuchfinkBridge) GetExchangeRate(currency, date string) (*domain.ExchangeRate, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.ExchangeRate, error) {
		return b.currencyService(s).RateAt(ctx, currency, date)
	})
}

// currencyService ist der Kursdienst für eine Auswertung: im Prüfermodus der
// nur lesende, der keinen fehlenden Kurs holt.
func (b *BuchfinkBridge) currencyService(s *core.Services) *service.CurrencyService {
	if b.host.Config().ReadOnly {
		return s.Currency.ReadOnly()
	}
	return s.Currency
}

// GetExchangeRates liefert die Kurshistorie einer Währung.
func (b *BuchfinkBridge) GetExchangeRates(currency, from, to string) ([]domain.ExchangeRate, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.ExchangeRate, error) {
		return s.Currency.Rates(ctx, currency, from, to)
	})
}

// SaveExchangeRate nimmt einen von Hand erfassten Kurs auf.
func (b *BuchfinkBridge) SaveExchangeRate(rate domain.ExchangeRate) (*domain.ExchangeRate, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.ExchangeRate, error) {
		return s.Currency.SaveRate(ctx, rate)
	})
}

// GetVatExchangeRates liefert die Umsatzsteuer-Durchschnittskurse eines
// Zeitraums.
func (b *BuchfinkBridge) GetVatExchangeRates(from, to string) ([]domain.VatExchangeRate, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.VatExchangeRate, error) {
		return s.Currency.VatRates(ctx, from, to)
	})
}

// ImportVatExchangeRatesCSV liest die Durchschnittskurse aus einer CSV-Datei.
func (b *BuchfinkBridge) ImportVatExchangeRatesCSV(path string) (*service.VatRateImport, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*service.VatRateImport, error) {
		return s.Currency.ImportVatRatesCSV(ctx, path)
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

// BookCurrencyValuation bucht die Stichtagsbewertung und ihre Auflösung.
func (b *BuchfinkBridge) BookCurrencyValuation(year int) (*service.ForeignCurrencyValuation, error) {
	year = b.yearOr(year)
	return prepared(b, func(ctx context.Context, t *core.Tenant) error {
		return t.PrefetchValuationRates(ctx, year)
	}, func(ctx context.Context, s *core.Services) (*service.ForeignCurrencyValuation, error) {
		return s.Currency.BookCurrencyValuation(ctx, year)
	})
}

// -------------------------------------------------------------------------
// Anlagen
// -------------------------------------------------------------------------

// GetAfaRules liefert die Abschreibungsregeln aus der eingebetteten Ressource —
// dieselbe Datei, aus der gerechnet wird.
func (b *BuchfinkBridge) GetAfaRules() accounting.AfARules {
	return accounting.AfARuleSet()
}

// GetWriteUpReport listet die Anlagegüter, bei denen die Wertaufholung zu prüfen
// ist.
func (b *BuchfinkBridge) GetWriteUpReport(year int) (*service.WriteUpReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.WriteUpReport, error) {
		return s.Assets.WriteUpReport(ctx, year)
	})
}

// ConfirmImpairmentPersists hält fest, dass der Grund einer außerplanmäßigen
// Abschreibung fortbesteht.
func (b *BuchfinkBridge) ConfirmImpairmentPersists(assetID uint, year int, note string) (*service.WriteUpReport, error) {
	year = b.yearOr(year)
	return write(b, func(ctx context.Context, s *core.Services) (*service.WriteUpReport, error) {
		return s.Assets.ConfirmImpairmentPersists(ctx, assetID, year, note)
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

// GetExemptionCertificateWarnings liefert die Freistellungsbescheinigungen nach
// § 48b EStG, die in den nächsten 30 Tagen ablaufen oder abgelaufen sind.
//
// today ist der Stichtag; leer heißt heute. Er steht in der Signatur, weil die
// Frist sonst nicht prüfbar wäre — und weil ein Prüfer denselben Stand zu einem
// vergangenen Tag sehen können muss.
func (b *BuchfinkBridge) GetExemptionCertificateWarnings(
	today string,
) ([]service.ExemptionCertificateWarning, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.ExemptionCertificateWarning, error) {
		return s.Contacts.ExemptionCertificateWarnings(ctx, today)
	})
}
