package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

type assetRateReader interface {
	RateAt(context.Context, string, string) (*domain.ExchangeRate, error)
}

func (s *AssetService) SetCurrencyRates(r assetRateReader) { s.currencyRates = r }

// ForeignAssetAcquisitionRequest ergänzt den gebuchten Zugang um die Originalwährung.
type ForeignAssetAcquisitionRequest struct {
	EntryID     uint         `json:"entryId"`
	Account     string       `json:"account"`
	Name        string       `json:"name"`
	Date        string       `json:"date"`
	Currency    string       `json:"currency"`
	ForeignCost domain.Cents `json:"foreignCost"`
	Quantity    domain.Units `json:"quantity"`
	// MaturityDate bestimmt bei Ausleihungen die Restlaufzeit am Bewertungsstichtag.
	MaturityDate string `json:"maturityDate,omitempty"`
	// RatePerEuro darf vom Beleg stammen; ohne Angabe wird der vorbereitete Kurs gelesen.
	RatePerEuro int64  `json:"ratePerEuro,omitempty"`
	RateSource  string `json:"rateSource,omitempty"`
	// Preview prüft die Zuordnung ohne einen Zugang zu speichern.
	Preview bool `json:"preview,omitempty"`
}

func (s *AssetService) RecordForeignAcquisition(
	ctx context.Context, req ForeignAssetAcquisitionRequest,
) (*domain.FixedAsset, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.FixedAsset, error) {
		if err := domain.CheckDate("Zugang", req.Date); err != nil {
			return nil, err
		}
		state, err := s.journalSvc.periods.State(ctx, req.Date)
		if err != nil {
			return nil, err
		}
		if err := state.ensureOpen(req.Date, false); err != nil {
			return nil, err
		}
		req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
		candidates, err := s.AcquisitionCandidates(ctx, s.journalSvc.FiscalYearOf(ctx, req.Date))
		if err != nil {
			return nil, err
		}
		var found *AcquisitionCandidate
		for i := range candidates {
			if candidates[i].EntryID == req.EntryID && candidates[i].Account == req.Account {
				found = &candidates[i]
			}
		}
		if found == nil {
			return nil, fmt.Errorf("die Buchung ist kein offener Aktivierungskandidat auf diesem Konto")
		}
		if req.Date != found.BookingDate {
			return nil, fmt.Errorf("das Zugangsdatum muss dem Buchungsdatum entsprechen")
		}
		rate := req.RatePerEuro
		if rate == 0 {
			if s.currencyRates == nil {
				return nil, fmt.Errorf("die Kurshistorie ist nicht verfügbar")
			}
			stored, err := s.currencyRates.RateAt(ctx, req.Currency, req.Date)
			if err != nil {
				return nil, err
			}
			rate, req.RateSource = stored.RateMicros, stored.Source
		}
		if rate <= 0 || strings.TrimSpace(req.RateSource) == "" {
			return nil, fmt.Errorf("Kurs und Kursquelle fehlen")
		}
		converted := domain.MulRound(req.ForeignCost, RateScale, rate)
		if converted != found.Amount {
			return nil, fmt.Errorf("zum Kurs ergeben sich %s €, gebucht sind %s €", converted, found.Amount)
		}
		asset := &domain.FixedAsset{
			Name: req.Name, Class: domain.AssetClassFinancial, Account: req.Account,
			AcquisitionDate: req.Date, AcquisitionCost: found.Amount, Method: domain.DepreciationNone,
			MaturityDate: req.MaturityDate, Currency: req.Currency, ForeignCost: req.ForeignCost, Quantity: req.Quantity,
			AcquisitionEntryID: &req.EntryID, ContactID: found.ContactID,
			Description: "Zugangskurs: " + rateLabel(rate) + "; Quelle: " + req.RateSource,
		}
		if req.Preview {
			if err := asset.Validate(); err != nil {
				return nil, err
			}
			if err := s.validateAccounts(ctx, asset); err != nil {
				return nil, err
			}
			return asset, nil
		}
		return s.Save(ctx, asset)
	})
}
