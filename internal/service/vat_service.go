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
	// Der Zeitraum wird in der Abfrage eingegrenzt, nicht hinterher: eine
	// monatliche Voranmeldung liest sonst ein ganzes Jahr, um einen Monat
	// auszuweisen.
	entries, err := s.journalRepo.FindByBookingDateRange(ctx, s.fiscalYear, from, to)
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
		// Der Saldenvortrag bringt den Bestand der Steuerkonten aus dem
		// Vorjahr ins neue Jahr. Er ist kein Umsatz und keine Vorsteuer dieses
		// Zeitraums — die Auswertung liest die Konten und nicht die
		// Steuerschlüssel, und ohne diese Zeile stünde die Vorsteuer des alten
		// Jahres ein zweites Mal in der Voranmeldung des Januars.
		// Dasselbe gilt für die Umsatzsteuer-Jahresverrechnung: sie stellt die
		// Steuerkonten zum Bilanzstichtag auf null. Zählte sie mit, hätte der
		// Dezember weder Umsatz noch Vorsteuer.
		if entry.Source == domain.EntrySourceOpening || entry.Source == domain.EntrySourceClosing {
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
	// Ein Zeitraum ohne steuerpflichtigen Umsatz ist der Regelfall im ersten
	// Monat einer Gesellschaft; die Ansicht läuft trotzdem über die Liste.
	summary.EnsureLists()
	return summary, nil
}

// revenueTreatments is the account-to-Steuerfall table. It lives in the
// accounting package, where the Kennziffern of the Voranmeldung read it too — a
// second copy here would drift the moment a Steuerfall is added.
var revenueTreatments = accounting.RevenueTreatments()

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
