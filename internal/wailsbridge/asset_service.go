package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Anlagenbuchhaltung an der Oberfläche.
//
// Alles, was hier durchgereicht wird, rechnet der Dienst — auch die Erklärungen.
// Wertgrenzen, Zeitfenster der degressiven AfA und die Wahl des Erlöskontos beim
// Abgang stehen deshalb nirgends im Frontend ein zweites Mal: eine zweite
// Fassung derselben Regel driftet, sobald sich eine davon ändert, und die im
// Frontend ist die, die niemand prüft.

// GetFixedAssets returns the Anlagenverzeichnis, optionally narrowed to one
// class ("tangible", "financial", "intangible"; empty for all).
func (b *BuchfinkBridge) GetFixedAssets(class string) ([]domain.FixedAsset, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.FixedAsset, error) {
		return s.Assets.List(ctx, b.viewYear(), domain.AssetClass(class))
	})
}

// GetFixedAsset returns one Anlagegut with its AfA-Plan, seinen Bewegungen und
// den Erklärungen, die zu diesem Gut gehören.
func (b *BuchfinkBridge) GetFixedAsset(id uint) (*service.AssetDetail, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.AssetDetail, error) {
		return s.Assets.Get(ctx, b.viewYear(), id)
	})
}

// AssetRules are the statutory figures the view explains itself with. They come
// from the same tables the computation uses.
type AssetRules struct {
	FiscalYear        int                           `json:"fiscalYear"`
	GWGImmediateLimit domain.Cents                  `json:"gwgImmediateLimit"`
	GWGRecordFrom     domain.Cents                  `json:"gwgRecordFrom"`
	PoolLowerLimit    domain.Cents                  `json:"poolLowerLimit"`
	PoolUpperLimit    domain.Cents                  `json:"poolUpperLimit"`
	PoolYears         int                           `json:"poolYears"`
	DegressiveWindows []accounting.DegressiveWindow `json:"degressiveWindows"`
	// ElectricVehicleWindows ist das Zeitfenster der Staffel des § 7 Abs. 2a
	// EStG. Es steht neben dem der degressiven AfA und aus demselben Grund: die
	// Maske soll eine Methode gar nicht erst anbieten, die das Backend gleich
	// darauf wegen des Anschaffungsdatums zurückweist.
	ElectricVehicleWindows []accounting.ElectricVehicleWindow `json:"electricVehicleWindows"`
	// SpecialMaxPermille und SpecialPeriodYears sind die Grenzen der
	// Sonderabschreibung nach § 7g Abs. 5 EStG: höchstens 40 % der
	// Anschaffungskosten, verteilbar auf das Anschaffungsjahr und die vier
	// folgenden.
	SpecialMaxPermille int `json:"specialMaxPermille"`
	SpecialPeriodYears int `json:"specialPeriodYears"`
	// InvestmentDeductionNote steht neben der Sonderabschreibung und sagt, was
	// § 7g EStG sonst noch kennt und Buchfink nicht führt: den
	// Investitionsabzugsbetrag des Absatzes 1. Er ist außerbilanziell und gehört
	// in die Steuererklärung — ohne diesen Satz sähe die Maske aus, als kenne
	// Buchfink die Vorschrift nur zur Hälfte.
	InvestmentDeductionNote string            `json:"investmentDeductionNote"`
	Methods                 []assetMethodInfo `json:"methods"`
}

type assetMethodInfo struct {
	Method domain.DepreciationMethod `json:"method"`
	Label  string                    `json:"label"`
	// Classes names the Anlagenklassen the method is available for. Finanzanlagen
	// haben keine planmäßige Abschreibung — das steht hier und nicht als
	// Sonderfall in der Maske.
	Classes []domain.AssetClass `json:"classes"`
	Hint    string              `json:"hint"`
}

// GetAnlagenspiegel returns the Entwicklung des Anlagevermögens of the active
// fiscal year.
func (b *BuchfinkBridge) GetAnlagenspiegel() (*domain.Anlagenspiegel, error) {
	return readOr(b, func() (*domain.Anlagenspiegel, error) {
		empty := &domain.Anlagenspiegel{}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*domain.Anlagenspiegel, error) {
		return s.Assets.Anlagenspiegel(ctx, b.viewYear())
	})
}

// GetAssetAcquisitionCandidates lists bookings on Anlagekonten that no Anlagegut
// points at yet.
func (b *BuchfinkBridge) GetAssetAcquisitionCandidates() ([]service.AcquisitionCandidate, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.AcquisitionCandidate, error) {
		return s.Assets.AcquisitionCandidates(ctx, b.viewYear())
	})
}

// -------------------------------------------------------------------------
// Investmentanteile
// -------------------------------------------------------------------------

// InvestmentRules are the choices and rates of the Investmentbesteuerung.
type InvestmentRules struct {
	FundClasses   []fundClassInfo     `json:"fundClasses"`
	InvestorTypes []investorTypeInfo  `json:"investorTypes"`
	InvestorType  domain.InvestorType `json:"investorType"`
	InvestorLabel string              `json:"investorLabel"`
	// InvestorReason sagt, woher die Anlegerstellung kommt: aus der Rechtsform
	// oder aus einer ausdrücklichen Festlegung. Ein Satz, der sich aus einer
	// anderen Angabe ergibt, sähe sonst wie eine Voreinstellung aus.
	InvestorReason string          `json:"investorReason"`
	LegalForm      string          `json:"legalForm"`
	Exemptions     []exemptionInfo `json:"exemptions"`
}

type fundClassInfo struct {
	Class accounting.FundClass `json:"class"`
	Label string               `json:"label"`
}

type investorTypeInfo struct {
	Type  domain.InvestorType `json:"type"`
	Label string              `json:"label"`
}

type exemptionInfo struct {
	Class       accounting.FundClass `json:"class"`
	Label       string               `json:"label"`
	Permille    int                  `json:"permille"`
	Source      string               `json:"source,omitempty"`
	Explanation string               `json:"explanation,omitempty"`
	// Problem sagt, warum es keinen Satz gibt — etwa bei fehlender
	// Anlegerstellung.
	Problem string `json:"problem,omitempty"`
}
