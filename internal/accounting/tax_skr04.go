// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// taxAccounts is the closed set of accounts the tax automation owns. Booking to
// one of them by hand would desynchronise the Umsatzsteuer-Voranmeldung from the
// journal, so the journal service rejects manual lines on these accounts.
//
// DATEV provides rate-specific accounts only for the standard rate in the § 13b
// and innergemeinschaftlicher-Erwerb groups; the generic account carries every
// other rate.
var taxAccounts = map[string]bool{
	domain.AccountVorsteuer: true, domain.AccountVorsteuer7: true, domain.AccountVorsteuer19: true,
	domain.AccountVorsteuerIG: true, domain.AccountVorsteuerIG19: true,
	domain.AccountVorsteuer13b: true, domain.AccountVorsteuer13b19: true,
	domain.AccountUmsatzsteuer: true, domain.AccountUmsatzsteuer7: true, domain.AccountUmsatzsteuer19: true,
	domain.AccountUmsatzsteuerIG: true, domain.AccountUmsatzsteuerIG19: true,
	domain.AccountUmsatzsteuer13b: true, domain.AccountUmsatzsteuer13b19: true,
}

// SKR04TaxResolver derives the tax lines of a booking from its Steuerfall.
type SKR04TaxResolver struct{}

// NewSKR04TaxResolver returns the SKR04 tax resolver.
func NewSKR04TaxResolver() *SKR04TaxResolver { return &SKR04TaxResolver{} }

// IsTaxAccount reports whether an account belongs to the tax automation.
func (r *SKR04TaxResolver) IsTaxAccount(account string) bool {
	return taxAccounts[account]
}

// Resolve returns the tax legs for a net amount under a given Steuerfall.
//
// Three shapes occur:
//   - one leg for a domestic transaction (input tax on Soll, output tax on Haben),
//   - two legs for Reverse Charge and innergemeinschaftlicher Erwerb, where the
//     recipient owes the tax and deducts the same amount as input tax, so both
//     sides are booked and the net effect on the result is zero,
//   - no leg at all for exempt, not-taxable and shifted-liability supplies.
func (r *SKR04TaxResolver) Resolve(dir domain.Direction, treatment domain.TaxTreatment, rate domain.TaxRate, net domain.Cents) ([]domain.TaxLeg, error) {
	switch treatment {
	case domain.TaxTreatmentDomestic:
		if rate == domain.TaxRateNone {
			return nil, nil
		}
		amount := rate.Tax(net)
		if dir == domain.DirectionIncoming {
			acc, err := inputTaxAccount(rate)
			if err != nil {
				return nil, err
			}
			return []domain.TaxLeg{{
				Account: acc, Side: domain.SideDebit, Amount: amount, Base: net,
				Key: fmt.Sprintf("VST%d", int(rate)/100),
			}}, nil
		}
		acc, err := outputTaxAccount(rate)
		if err != nil {
			return nil, err
		}
		return []domain.TaxLeg{{
			Account: acc, Side: domain.SideCredit, Amount: amount, Base: net,
			Key: fmt.Sprintf("UST%d", int(rate)/100),
		}}, nil

	case domain.TaxTreatmentReverseCharge:
		if dir != domain.DirectionIncoming {
			return nil, fmt.Errorf("§ 13b als Leistungsempfänger ist nur bei Eingangsbelegen möglich")
		}
		if rate == domain.TaxRateNone {
			return nil, fmt.Errorf("§ 13b UStG braucht einen Steuersatz")
		}
		amount := rate.Tax(net)
		vst, ust := domain.AccountVorsteuer13b, domain.AccountUmsatzsteuer13b
		if rate == domain.TaxRateStandard {
			vst, ust = domain.AccountVorsteuer13b19, domain.AccountUmsatzsteuer13b19
		}
		return []domain.TaxLeg{
			{Account: vst, Side: domain.SideDebit, Amount: amount, Base: net, Key: fmt.Sprintf("RC%d_VST", int(rate)/100)},
			{Account: ust, Side: domain.SideCredit, Amount: amount, Base: net, Key: fmt.Sprintf("RC%d_UST", int(rate)/100)},
		}, nil

	case domain.TaxTreatmentIntraCommunityAcquisition:
		if dir != domain.DirectionIncoming {
			return nil, fmt.Errorf("ein innergemeinschaftlicher Erwerb ist nur bei Eingangsbelegen möglich")
		}
		if rate == domain.TaxRateNone {
			return nil, fmt.Errorf("ein innergemeinschaftlicher Erwerb braucht einen Steuersatz")
		}
		amount := rate.Tax(net)
		vst, ust := domain.AccountVorsteuerIG, domain.AccountUmsatzsteuerIG
		if rate == domain.TaxRateStandard {
			vst, ust = domain.AccountVorsteuerIG19, domain.AccountUmsatzsteuerIG19
		}
		return []domain.TaxLeg{
			{Account: vst, Side: domain.SideDebit, Amount: amount, Base: net, Key: fmt.Sprintf("IG%d_VST", int(rate)/100)},
			{Account: ust, Side: domain.SideCredit, Amount: amount, Base: net, Key: fmt.Sprintf("IG%d_UST", int(rate)/100)},
		}, nil

	case domain.TaxTreatmentIntraCommunitySupply,
		domain.TaxTreatmentExport,
		domain.TaxTreatmentReverseChargeSupply,
		domain.TaxTreatmentExempt,
		domain.TaxTreatmentNotTaxable:
		// Steuerfrei, nicht steuerbar oder Steuerschuld beim Empfänger: die
		// Buchung erzeugt keine Steuerzeile. Der Steuerfall bleibt trotzdem am
		// Beleg gespeichert, weil er für die UStVA-Kennzahlen gebraucht wird.
		return nil, nil

	default:
		return nil, fmt.Errorf("unbekannter Steuerfall %q", treatment)
	}
}

func inputTaxAccount(rate domain.TaxRate) (string, error) {
	switch rate {
	case domain.TaxRateStandard:
		return domain.AccountVorsteuer19, nil
	case domain.TaxRateReduced:
		return domain.AccountVorsteuer7, nil
	default:
		return "", fmt.Errorf("kein Vorsteuerkonto für den Steuersatz %s hinterlegt", rate.Label())
	}
}

func outputTaxAccount(rate domain.TaxRate) (string, error) {
	switch rate {
	case domain.TaxRateStandard:
		return domain.AccountUmsatzsteuer19, nil
	case domain.TaxRateReduced:
		return domain.AccountUmsatzsteuer7, nil
	default:
		return "", fmt.Errorf("kein Umsatzsteuerkonto für den Steuersatz %s hinterlegt", rate.Label())
	}
}
