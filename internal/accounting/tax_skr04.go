package accounting

import (
	"fmt"
	"strconv"
	"strings"

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
	domain.AccountUmsatzsteuer14c: true,
	// Die vier Konten der Vorsteuerberichtigung nach § 15a UStG. Sie werden nur
	// aus dem Verzeichnis heraus bebucht, und immer mit dem Steuerschlüssel:
	// eine Handbuchung darauf käme in der Kennziffer 64 an, ohne dass im
	// Verzeichnis stünde, aus welchem Wirtschaftsgut sie stammt.
	InputTaxCorrectionDeductibleMovable:   true,
	InputTaxCorrectionRepayableMovable:    true,
	InputTaxCorrectionDeductibleImmovable: true,
	InputTaxCorrectionRepayableImmovable:  true,
}

// TaxKeyUnlawful ist der Steuerschlüssel der nach § 14c UStG geschuldeten
// Beträge.
//
// Er entsteht nicht aus einem Steuerfall, sondern aus einem Fehler: einer
// Rechnung, die Steuer ausweist, obwohl keine entstanden ist. Buchfink stellt
// eine solche Rechnung nicht aus (siehe Invoice.Validate), aber der Betrag wird
// trotzdem geschuldet, wenn die Rechnung außerhalb von Buchfink entstanden ist.
// Für diesen Fall gibt es den Schlüssel und den Weg über eine Handbuchung auf
// das Konto 3851 — nicht als Automatik, weil es nichts zu automatisieren gibt.
const TaxKeyUnlawful = "UST14C"

// IsDomesticOutputTaxKey meldet, ob ein Steuerschlüssel die Umsatzsteuer eines
// steuerpflichtigen Inlandsumsatzes trägt (UST19, UST7).
//
// Der Schlüssel des § 14c gehört ausdrücklich nicht dazu: er benennt gerade den
// Betrag, der ohne Steuerpflicht ausgewiesen wurde, und ist der einzige Weg,
// eine solche Steuer überhaupt zu buchen.
func IsDomesticOutputTaxKey(key string) bool {
	if key == TaxKeyUnlawful || !strings.HasPrefix(key, "UST") {
		return false
	}
	_, err := strconv.Atoi(strings.TrimPrefix(key, "UST"))
	return err == nil
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
			// TaxRateNone means "no rate applies", and a domestic taxable
			// transaction always has one. Accepting it here would silently book
			// three different cases — exempt, not taxable and the Nullsteuersatz
			// of § 12 Abs. 3 UStG — as the same thing.
			return nil, fmt.Errorf(
				"ein steuerpflichtiger Inlandsumsatz hat 19 %% oder 7 %%. Fällt keine Steuer an, ist der Steuerfall zu nennen: steuerfrei, nicht steuerbar oder Nullsteuersatz nach § 12 Abs. 3 UStG")
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

	case domain.TaxTreatmentZeroRated:
		// § 12 Abs. 3 UStG: "Die Steuer ermäßigt sich auf 0 Prozent". Es entsteht
		// keine Steuerzeile, aber der Umsatz ist steuerpflichtig — der
		// Vorsteuerabzug des Leistenden bleibt erhalten, und in der Auswertung
		// gehört der Betrag zu den steuerpflichtigen Umsätzen, nicht zu den
		// steuerfreien. Diesen Unterschied trägt der Steuerfall an der Buchung,
		// auf der Ausgangsseite zusätzlich das eigene Erlöskonto 4290.
		return nil, nil

	case domain.TaxTreatmentIntraCommunitySupply,
		domain.TaxTreatmentExport,
		domain.TaxTreatmentReverseChargeSupply,
		domain.TaxTreatmentExempt,
		domain.TaxTreatmentNotTaxable:
		// Steuerfrei, nicht steuerbar oder Steuerschuld beim Empfänger: die
		// Buchung erzeugt keine Steuerzeile. Der Steuerfall bleibt trotzdem an
		// der Buchung gespeichert, weil er für die UStVA-Kennzahlen gebraucht
		// wird.
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
