package accounting

import "github.com/buchfink/buchfink/internal/domain"

// TaxCase bündelt die Eigenschaften eines Steuerfalls. Journalzeilen werden
// nach ihrem Schlüssel ausgewertet, auch bei gemischten oder manuellen Buchungen.
type TaxCase struct {
	treatment domain.TaxTreatment
}

func TaxCaseFor(treatment domain.TaxTreatment) TaxCase {
	return TaxCase{treatment: treatment}
}

func (c TaxCase) Legs(direction domain.Direction, rate domain.TaxRate, net domain.Cents) ([]domain.TaxLeg, error) {
	return NewSKR04TaxResolver().Resolve(direction, c.treatment, rate, net)
}

func (c TaxCase) MayShowTax() bool { return c.treatment.MayShowTax() }

// TaxCaseSkonto enthält die Entgeltminderung und die gegenläufigen Steuerzeilen.
// Eine Minderung von Anschaffungskosten ordnet der Zahlungsdienst dem Anlagegut zu.
type TaxCaseSkonto struct {
	Account string
	Net     domain.Cents
	Legs    []domain.TaxLeg
}

func (c TaxCase) Skonto(direction domain.Direction, rate domain.TaxRate, gross domain.Cents) (TaxCaseSkonto, error) {
	skontoRate := domain.TaxRateNone
	if c.MayShowTax() {
		skontoRate = rate
	}
	net := skontoRate.NetFromGross(gross)
	legs, err := c.Legs(direction, rate, net)
	if err != nil {
		return TaxCaseSkonto{}, err
	}
	account, err := domain.SkontoAccount(direction, skontoRate)
	if err != nil {
		return TaxCaseSkonto{}, err
	}
	if direction == domain.DirectionOutgoing && c.treatment == domain.TaxTreatmentReverseChargeSupply {
		account = ReverseChargeSupplySkontoAccount
	}
	corrections := make([]domain.TaxLeg, 0, len(legs))
	for _, leg := range legs {
		if leg.Amount == 0 {
			continue
		}
		leg.Side = leg.Side.Opposite()
		leg.Key = TaxKeySkontoPrefix + leg.Key
		corrections = append(corrections, leg)
	}
	return TaxCaseSkonto{Account: account, Net: net, Legs: corrections}, nil
}

// TaxPartnerCheck beschreibt die Partnerprüfung ohne Datenbank- oder Netzaufruf.
// ConfirmEUVatID setzt einen angeschlossenen Bestätigungsdienst voraus.
type TaxPartnerCheck struct {
	Allowed            bool
	RequiresVatID      bool
	RequiresEU         bool
	RequiresVatIDForEU bool
	ConfirmEUVatID     bool
}

func (c TaxCase) CheckPartner(direction domain.Direction) TaxPartnerCheck {
	var check TaxPartnerCheck
	for _, info := range domain.TaxTreatments(direction) {
		if info.Treatment == c.treatment {
			check.Allowed, check.RequiresVatID = true, info.RequiresVatID
			break
		}
	}
	if direction == domain.DirectionIncoming {
		// § 13b Abs. 1 UStG und der ig. Erwerb setzen einen Unternehmer im übrigen EU-Gebiet voraus.
		check.RequiresEU = c.treatment == domain.TaxTreatmentReverseChargeEU || c.treatment == domain.TaxTreatmentIntraCommunityAcquisition
		// Bei inländischen Leistungen nach Abs. 2 hängt die Steuerschuld am Empfänger (§ 13b Abs. 5 UStG).
		check.RequiresVatIDForEU = c.treatment == domain.TaxTreatmentReverseCharge
	}
	check.ConfirmEUVatID = direction == domain.DirectionOutgoing && c.treatment == domain.TaxTreatmentReverseChargeSupply
	return check
}

// InvoiceCategory ist die Kategorie nach EN 16931 / UNTDID 5305.
func (c TaxCase) InvoiceCategory() string {
	switch c.treatment {
	case domain.TaxTreatmentReverseChargeSupply:
		return "AE"
	case domain.TaxTreatmentExempt:
		return "E"
	case domain.TaxTreatmentNotTaxable:
		return "O"
	case domain.TaxTreatmentZeroRated:
		// § 12 Abs. 3 UStG: steuerpflichtig zum Satz null. Kategorie S mit 0 % verletzt BR-S-05.
		return "Z"
	default:
		return "S"
	}
}
