// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// VatService aggregates the VAT figures from the journal.
//
// The figures come from the tax lines the Steuerautomatik wrote, not from
// guessing at account numbers: every tax line carries its Steuerschlüssel and
// its Bemessungsgrundlage. A Generalumkehr nets itself out automatically,
// because its amounts are negative on the same side.
//
// This is an aggregation for orientation, not an Umsatzsteuer-Voranmeldung. The
// official return needs its own form and its own validation.
type VatService struct {
	journalRepo domain.JournalRepository
	fiscalYear  int
}

// NewVatService creates the VAT aggregation service.
func NewVatService(journalRepo domain.JournalRepository, fiscalYear int) *VatService {
	return &VatService{journalRepo: journalRepo, fiscalYear: fiscalYear}
}

// SetFiscalYear updates the year the figures are computed for.
func (s *VatService) SetFiscalYear(year int) { s.fiscalYear = year }

// Summary computes the VAT figures for a date range. Empty bounds mean the
// whole fiscal year.
func (s *VatService) Summary(ctx context.Context, from, to string) (*domain.VatSummary, error) {
	entries, err := s.journalRepo.FindAll(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}

	summary := &domain.VatSummary{
		FiscalYear: s.fiscalYear,
		PeriodFrom: from,
		PeriodTo:   to,
	}
	taxableByRate := map[domain.TaxRate]*domain.VatFigure{}

	for i := range entries {
		entry := &entries[i]
		if from != "" && entry.BookingDate < from {
			continue
		}
		if to != "" && entry.BookingDate > to {
			continue
		}

		for _, line := range entry.Lines {
			// Signed amount in the account's natural direction.
			credit := line.Amount
			debit := line.Amount
			if line.Side == domain.SideDebit {
				credit = -credit
			} else {
				debit = -debit
			}

			// TaxBase is already negative on a Generalumkehr, so a correction
			// reduces the reported base without a special case here.
			switch line.Account {
			// Vereinnahmte Umsatzsteuer auf steuerpflichtige Inlandsumsätze.
			case domain.AccountUmsatzsteuer19, domain.AccountUmsatzsteuer7, domain.AccountUmsatzsteuer:
				rate := rateForOutputAccount(line.Account)
				figure := taxableByRate[rate]
				if figure == nil {
					figure = &domain.VatFigure{Rate: rate}
					taxableByRate[rate] = figure
				}
				figure.Net += line.TaxBase
				figure.Tax += credit
				summary.OutputTax += credit

			// Geschuldete Steuer nach § 13b UStG.
			case domain.AccountUmsatzsteuer13b19, domain.AccountUmsatzsteuer13b:
				summary.ReverseChargeTax += credit
				summary.ReverseChargeBase += line.TaxBase

			// Erwerbsteuer aus innergemeinschaftlichem Erwerb.
			case domain.AccountUmsatzsteuerIG19, domain.AccountUmsatzsteuerIG:
				summary.IntraCommunityAcquisitionTax += credit
				summary.IntraCommunityAcquisitionBase += line.TaxBase

			// Abziehbare Vorsteuer, unabhängig von ihrer Herkunft.
			case domain.AccountVorsteuer19, domain.AccountVorsteuer7, domain.AccountVorsteuer,
				domain.AccountVorsteuer13b19, domain.AccountVorsteuer13b,
				domain.AccountVorsteuerIG19, domain.AccountVorsteuerIG:
				summary.InputTax += debit

			// Steuerfreie Umsätze mit eigenem Erlöskonto.
			case "4125":
				summary.IntraCommunitySupply += credit
			case "4120":
				summary.Export += credit
			case "4337":
				summary.ReverseChargeSupply += credit
			case "4150", "4110", "4160", "4165":
				summary.ExemptRevenue += credit
			}
		}
	}

	rates := make([]domain.TaxRate, 0, len(taxableByRate))
	for rate := range taxableByRate {
		rates = append(rates, rate)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })
	for _, rate := range rates {
		summary.TaxableRevenue = append(summary.TaxableRevenue, *taxableByRate[rate])
	}

	summary.TotalOwedTax = summary.OutputTax + summary.ReverseChargeTax + summary.IntraCommunityAcquisitionTax
	summary.Payable = summary.TotalOwedTax - summary.InputTax
	return summary, nil
}

func rateForOutputAccount(account string) domain.TaxRate {
	switch account {
	case domain.AccountUmsatzsteuer19:
		return domain.TaxRateStandard
	case domain.AccountUmsatzsteuer7:
		return domain.TaxRateReduced
	default:
		return domain.TaxRateNone
	}
}
