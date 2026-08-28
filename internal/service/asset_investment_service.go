package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Investmentanteile in der Anlagenkartei.
//
// In der Bilanz ist ein ETF ein Wertpapier des Anlagevermögens wie jedes
// andere. Steuerlich legt das InvStG zwei Rechnungen daneben, die in keiner
// Buchung auftauchen: die Teilfreistellung nach § 20 und die Vorabpauschale
// nach § 18. Beide sind außerbilanziell — es wird nichts gebucht, weil
// handelsrechtlich nichts geschieht.
//
// Genau deshalb gehören sie hierher. Was nicht gebucht wird, hat sonst keinen
// Ort, an dem es über die Jahre stehen bleibt — und die Vorabpauschale muss
// über die ganze Besitzzeit stehen bleiben, weil sie beim Abgang wieder
// abgezogen wird.

// VorabpauschaleRequest rechnet und erfasst die Vorabpauschale eines
// Kalenderjahres.
type VorabpauschaleRequest struct {
	AssetID uint `json:"assetId"`
	Year    int  `json:"year"`
	// OpeningPrice und ClosingPrice sind der erste und der letzte im
	// Kalenderjahr festgesetzte Rücknahmepreis des gehaltenen Bestands.
	OpeningPrice  domain.Cents `json:"openingPrice"`
	ClosingPrice  domain.Cents `json:"closingPrice"`
	Distributions domain.Cents `json:"distributions"`
	// BasisPoints ist der Basiszins in Basispunkten: 253 sind 2,53 %.
	BasisPoints int `json:"basisPoints"`
	// Record sagt, ob das Ergebnis festgehalten wird. Ohne ihn rechnet der
	// Aufruf nur vor.
	Record bool   `json:"record"`
	Note   string `json:"note,omitempty"`
}

// Vorabpauschale computes the Vorabpauschale of one Kalenderjahr — und hält sie
// fest, wenn darum gebeten wird.
func (s *AssetService) Vorabpauschale(ctx context.Context, req VorabpauschaleRequest) (*accounting.Vorabpauschale, error) {
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	fund := accounting.FundClass(asset.FundClass)
	if !fund.IsFund() {
		return nil, fmt.Errorf(
			"%s ist kein Investmentanteil. Die Vorabpauschale des § 18 InvStG gibt es nur für "+
				"Anteile an einem Investmentfonds — eine Einzelaktie und eine Anleihe tragen keine",
			asset.InventoryNumber)
	}
	if req.Year <= 0 {
		return nil, fmt.Errorf("ohne Kalenderjahr lässt sich keine Vorabpauschale rechnen")
	}

	// § 18 Abs. 2 InvStG kürzt im Jahr des Erwerbs. Der Erwerbsmonat steht am
	// Anlagegut; nur für dessen eigenes Kalenderjahr ist er von Belang.
	acquisitionMonth := 0
	if len(asset.AcquisitionDate) == 10 && asset.AcquisitionDate[:4] == fmt.Sprintf("%d", req.Year) {
		var month int
		if _, err := fmt.Sscanf(asset.AcquisitionDate[5:7], "%d", &month); err == nil {
			acquisitionMonth = month
		}
	}

	result, err := accounting.ComputeVorabpauschale(accounting.VorabpauschaleInput{
		Year:             req.Year,
		OpeningPrice:     req.OpeningPrice,
		ClosingPrice:     req.ClosingPrice,
		Distributions:    req.Distributions,
		BasisPoints:      req.BasisPoints,
		AcquisitionMonth: acquisitionMonth,
	})
	if err != nil {
		return nil, err
	}
	if !req.Record {
		return &result, nil
	}

	for _, m := range asset.Movements {
		if m.Kind == domain.AssetMovementVorabpauschale && m.FiscalYear == req.Year {
			return nil, fmt.Errorf(
				"für %d ist zu %s bereits eine Vorabpauschale von %s € erfasst",
				req.Year, asset.InventoryNumber, m.TaxAmount)
		}
	}

	movement := &domain.AssetMovement{
		AssetID: asset.ID, Kind: domain.AssetMovementVorabpauschale, Account: asset.Account,
		// Sie gilt am ersten Werktag des folgenden Kalenderjahres als zugeflossen
		// (§ 18 Abs. 3 InvStG) — festgehalten wird sie trotzdem bei dem Jahr,
		// aus dem sie stammt, sonst wäre die Summe über die Besitzzeit
		// gegenüber der Wertentwicklung verschoben.
		Date: result.AccruedOn, FiscalYear: req.Year,
		TaxAmount: result.Amount,
		Note:      trimJoin(result.Explanation, req.Note),
	}
	if err := s.assetRepo.AddMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf("die Vorabpauschale konnte nicht festgehalten werden: %w", err)
	}
	s.audit(ctx, domain.AuditActionCreate, asset.ID, fmt.Sprintf(
		"Vorabpauschale %d zu %s: %s € (außerbilanziell, keine Buchung)",
		req.Year, asset.InventoryNumber, result.Amount))
	return &result, nil
}

// InvestmentTaxNote is the steuerliche Nebenrechnung zu einem Investmentanteil.
//
// Sie steht neben der Buchung und nicht in ihr. Handelsrechtlich ist der Abgang
// eines Fondsanteils ein Abgang wie jeder andere; was das InvStG daraus macht,
// entscheidet sich außerhalb der Bilanz — und wäre ohne diese Aufstellung im
// Jahresabschluss nicht mehr auffindbar.
type InvestmentTaxNote struct {
	FundClass      string `json:"fundClass"`
	FundClassLabel string `json:"fundClassLabel"`
	// Exemption ist die Teilfreistellung, oder der Grund, warum sie nicht
	// bestimmt werden kann.
	Exemption      accounting.PartialExemption `json:"exemption"`
	ExemptionError string                      `json:"exemptionError,omitempty"`

	// GrossAmount ist der Betrag vor allem: der Buchgewinn beim Abgang, der
	// Ertrag bei einer Ausschüttung.
	GrossAmount domain.Cents `json:"grossAmount"`
	// Vorabpauschalen ist die Summe der über die Besitzzeit angesetzten
	// Vorabpauschalen. Sie mindert allein den Veräußerungsgewinn.
	Vorabpauschalen domain.Cents `json:"vorabpauschalen"`
	// ExemptAmount ist der steuerfreie Teil, TaxableAmount der steuerpflichtige.
	ExemptAmount  domain.Cents `json:"exemptAmount"`
	TaxableAmount domain.Cents `json:"taxableAmount"`
	Explanation   string       `json:"explanation"`
}

// investmentNote builds the Nebenrechnung for one amount.
//
// disposal unterscheidet die beiden Fälle: beim Abgang sind die angesetzten
// Vorabpauschalen abzuziehen, bei einer Ausschüttung nicht — dort wären sie
// doppelt berücksichtigt.
func (s *AssetService) investmentNote(
	ctx context.Context, asset *domain.FixedAsset, gross domain.Cents, disposal bool,
) *InvestmentTaxNote {
	fund := accounting.FundClass(asset.FundClass)
	if !fund.IsFund() {
		return nil
	}
	note := &InvestmentTaxNote{
		FundClass:      string(fund),
		FundClassLabel: fund.Label(),
		GrossAmount:    gross,
	}

	investor := domain.InvestorUnknown
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil {
			investor = cfg.InvestorType
		}
	}

	taxable := gross
	if disposal {
		for _, m := range asset.Movements {
			if m.Kind == domain.AssetMovementVorabpauschale {
				note.Vorabpauschalen += m.TaxAmount
			}
		}
		taxable -= note.Vorabpauschalen
	}

	exemption, err := accounting.PartialExemptionFor(fund, investor)
	if err != nil {
		note.ExemptionError = err.Error()
		note.TaxableAmount = taxable
		note.Explanation = fmt.Sprintf(
			"%s Der Betrag steht deshalb ungekürzt: %s €.", err.Error(), taxable)
		return note
	}
	note.Exemption = exemption
	if taxable > 0 {
		note.ExemptAmount = domain.MulRound(taxable, int64(exemption.Permille), 1000)
	}
	note.TaxableAmount = taxable - note.ExemptAmount

	what := "Der Ertrag"
	if disposal {
		what = "Der Buchgewinn"
	}
	explanation := fmt.Sprintf("%s beträgt %s €.", what, gross)
	if disposal && note.Vorabpauschalen > 0 {
		explanation += fmt.Sprintf(
			" Davon gehen %s € an Vorabpauschalen ab, die über die Besitzzeit bereits versteuert "+
				"wurden — für Anteile im Privatvermögen ordnet § 19 Abs. 1 Satz 3 InvStG das "+
				"ausdrücklich an; im Betriebsvermögen folgt der Abzug daraus, dass sie dort schon "+
				"als Ertrag erfasst waren. Er geschieht außerhalb der Bilanz.",
			note.Vorabpauschalen)
	}
	explanation += fmt.Sprintf(" %s Steuerfrei bleiben %s €, zu versteuern sind %s €.",
		exemption.Explanation, note.ExemptAmount, note.TaxableAmount)
	note.Explanation = explanation
	return note
}

func trimJoin(a, b string) string {
	if b == "" {
		return a
	}
	return a + " " + b
}
