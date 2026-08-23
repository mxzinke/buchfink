package service

import (
	"context"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
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
			// Signed in the account's natural direction, base included.
			credit, base := line.Amount, line.TaxBase
			if line.Side == domain.SideDebit {
				credit, base = -credit, -base
			}
			debit := -credit

			// Die Bemessungsgrundlage folgt derselben Seitenlogik wie der
			// Steuerbetrag, und das ist keine Kosmetik: die Steuerkorrektur
			// eines Skontos nach § 17 Abs. 1 UStG steht mit positivem Betrag
			// und positiver Grundlage auf der Gegenseite des Steuerkontos.
			// Roh addiert senkte ein Skonto die Steuer und erhöhte zugleich den
			// Umsatz, aus dem sie stammt — zwei Zahlen, die einander in
			// derselben Voranmeldung widersprechen. Die Generalumkehr bleibt
			// unberührt: sie negiert Betrag und Grundlage und behält die Seite.
			switch line.Account {
			// Vereinnahmte Umsatzsteuer auf steuerpflichtige Inlandsumsätze.
			case domain.AccountUmsatzsteuer19, domain.AccountUmsatzsteuer7, domain.AccountUmsatzsteuer:
				rate := rateForOutputAccount(line.Account)
				figure := taxableByRate[rate]
				if figure == nil {
					figure = &domain.VatFigure{Rate: rate}
					taxableByRate[rate] = figure
				}
				figure.Net += base
				figure.Tax += credit
				summary.OutputTax += credit

			// Geschuldete Steuer nach § 13b UStG.
			case domain.AccountUmsatzsteuer13b19, domain.AccountUmsatzsteuer13b:
				summary.ReverseChargeTax += credit
				summary.ReverseChargeBase += base

			// Erwerbsteuer aus innergemeinschaftlichem Erwerb.
			case domain.AccountUmsatzsteuerIG19, domain.AccountUmsatzsteuerIG:
				summary.IntraCommunityAcquisitionTax += credit
				summary.IntraCommunityAcquisitionBase += base

			// Abziehbare Vorsteuer, unabhängig von ihrer Herkunft.
			case domain.AccountVorsteuer19, domain.AccountVorsteuer7, domain.AccountVorsteuer,
				domain.AccountVorsteuer13b19, domain.AccountVorsteuer13b,
				domain.AccountVorsteuerIG19, domain.AccountVorsteuerIG:
				summary.InputTax += debit

			// Erlöse mit eigenem Konto je Steuerfall. Welches Konto für welchen
			// Fall steht, sagt der Buchungsgruppen-Katalog — hier steht es nicht
			// ein zweites Mal, sonst fiele der nächste Steuerfall stillschweigend
			// aus der Voranmeldung.
			default:
				switch revenueTreatments[line.Account] {
				case domain.TaxTreatmentIntraCommunitySupply:
					summary.IntraCommunitySupply += credit
				case domain.TaxTreatmentExport:
					summary.Export += credit
				case domain.TaxTreatmentReverseChargeSupply:
					summary.ReverseChargeSupply += credit
				// Nullsteuersatz § 12 Abs. 3 UStG: steuerpflichtig zum Satz null,
				// deshalb ein eigener Ausweis und nicht der Topf der steuerfreien
				// Umsätze.
				case domain.TaxTreatmentZeroRated:
					summary.ZeroRatedRevenue += credit
				case domain.TaxTreatmentExempt:
					summary.ExemptRevenue += credit
				}
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

// revenueTreatments is the catalogue's account-to-Steuerfall table, plus the
// steuerfreie Erlöskonten of the SKR04 that no Buchungsgruppe offers but a
// handwritten journal entry can still reach.
var revenueTreatments = func() map[string]domain.TaxTreatment {
	out := accounting.RevenueAccountTreatments()
	for _, account := range []string{"4110", "4160", "4165"} {
		out[account] = domain.TaxTreatmentExempt
	}
	return out
}()

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
