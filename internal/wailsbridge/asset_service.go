package wailsbridge

import (
	"context"
	"fmt"

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

// GetAssetRules hands the Wertgrenzen, das Zeitfenster der degressiven AfA und
// die zulässigen Methoden an die Oberfläche.
func (b *BuchfinkBridge) GetAssetRules() (*AssetRules, error) {
	year := b.viewYear()

	// Die Wertgrenzen gelten je Tag; für die Anzeige nehmen wir den ersten Tag
	// des Geschäftsjahres, weil sich Anschaffungen dieses Jahres daran messen.
	params, err := accounting.AfAParametersFor(fmt.Sprintf("%d-01-01", year))
	if err != nil {
		return nil, err
	}

	all := []domain.AssetClass{
		domain.AssetClassIntangible, domain.AssetClassTangible, domain.AssetClassFinancial,
	}
	return &AssetRules{
		FiscalYear:        year,
		GWGImmediateLimit: params.GWGImmediateLimit,
		GWGRecordFrom:     params.GWGRecordThreshold,
		PoolLowerLimit:    params.PoolLowerLimit,
		PoolUpperLimit:    params.PoolUpperLimit,
		PoolYears:         params.PoolYears,
		DegressiveWindows: accounting.DegressiveWindows(),
		// Aus derselben Ressource, aus der auch gerechnet wird.
		ElectricVehicleWindows: accounting.AfARuleSet().ElectricVehicle,
		SpecialMaxPermille:     accounting.SpecialMaxPermille,
		SpecialPeriodYears:     accounting.SpecialPeriodYears,
		// Aus derselben Ressource wie die Sätze: der Hinweis steht dort, wo die
		// Regeln stehen, und nicht als zweite Fassung im Bridge-Code.
		InvestmentDeductionNote: accounting.AfARuleSet().InvestmentDeductionNote,
		Methods: []assetMethodInfo{
			{
				Method: domain.DepreciationLinear, Label: domain.DepreciationLinear.Label(),
				Classes: []domain.AssetClass{domain.AssetClassIntangible, domain.AssetClassTangible},
				Hint: "Gleichmäßig über die betriebsgewöhnliche Nutzungsdauer, zeitanteilig ab dem " +
					"Anschaffungsmonat. Der Normalfall.",
			},
			{
				Method: domain.DepreciationDegressive, Label: domain.DepreciationDegressive.Label(),
				Classes: []domain.AssetClass{domain.AssetClassTangible},
				Hint: "Vom jeweiligen Restbuchwert, höchstens das Dreifache des linearen Satzes und " +
					"höchstens 30 %. Nur für bewegliche Wirtschaftsgüter und nur für Anschaffungen " +
					"innerhalb eines der gesetzlichen Zeitfenster. Eine Sonderabschreibung nach " +
					"§ 7g Abs. 5 EStG ist daneben zulässig.",
			},
			{
				// Ohne diesen Eintrag steht die Staffel des § 7 Abs. 2a EStG zwar im
				// Rechenkern, ist aber in keiner Maske wählbar — und ein E-Fahrzeug
				// liefe still linear.
				Method:  domain.DepreciationElectricVehicle,
				Label:   domain.DepreciationElectricVehicle.Label(),
				Classes: []domain.AssetClass{domain.AssetClassTangible},
				Hint: "Die Staffel des § 7 Abs. 2a EStG: 75, 10, 5, 5, 3 und 2 % der " +
					"Anschaffungskosten. Nur für neue, rein elektrisch betriebene Fahrzeuge und nur " +
					"für Anschaffungen nach dem 30.06.2025 und vor dem 01.01.2028. Das " +
					"Anschaffungsjahr bekommt den vollen Satz, ohne Zeitanteil.",
			},
			{
				Method:  domain.DepreciationBuildingLinear,
				Label:   domain.DepreciationBuildingLinear.Label(),
				Classes: []domain.AssetClass{domain.AssetClassTangible},
				Hint: "Die festen Sätze des § 7 Abs. 4 EStG für Gebäude. Welcher gilt, entscheidet " +
					"der Stichtag: der Bauantrag beim Betriebsgebäude, die Fertigstellung beim " +
					"Wohngebäude. Die betriebsgewöhnliche Nutzungsdauer spielt hier keine Rolle.",
			},
			{
				Method: domain.DepreciationPool, Label: domain.DepreciationPool.Label(),
				Classes: []domain.AssetClass{domain.AssetClassTangible},
				Hint: "Ein Pool je Wirtschaftsjahr, aufgelöst mit je einem Fünftel über fünf Jahre. " +
					"Das Wahlrecht gilt einheitlich für alle Wirtschaftsgüter des Jahres.",
			},
			{
				Method: domain.DepreciationImmediate, Label: domain.DepreciationImmediate.Label(),
				Classes: []domain.AssetClass{domain.AssetClassTangible},
				Hint: "Voller Aufwand im Anschaffungsjahr. Das Gut bleibt im Verzeichnis, weil ab 250 € " +
					"ein laufend geführtes Verzeichnis vorgeschrieben ist.",
			},
			{
				Method: domain.DepreciationNone, Label: domain.DepreciationNone.Label(),
				Classes: all,
				Hint: "Für alles, was sich nicht abnutzt: Grund und Boden, Anlagen im Bau und das " +
					"gesamte Finanzanlagevermögen.",
			},
		},
	}, nil
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

// GetInvestmentRules hands the UI what § 18 und § 20 InvStG an Auswahl
// vorgeben: die Fondsarten, die Anlegerstellungen und den Satz, der sich aus
// der eingestellten Kombination ergibt.
func (b *BuchfinkBridge) GetInvestmentRules() (*InvestmentRules, error) {
	// Leer statt nil: die Maske läuft über die Freistellungen, auch wenn die
	// eingestellte Anlegerstellung keine ergibt.
	rules := &InvestmentRules{Exemptions: make([]exemptionInfo, 0)}
	for _, class := range accounting.AllFundClasses() {
		rules.FundClasses = append(rules.FundClasses,
			fundClassInfo{Class: class, Label: class.Label()})
	}
	for _, investor := range domain.AllInvestorTypes() {
		rules.InvestorTypes = append(rules.InvestorTypes,
			investorTypeInfo{Type: investor, Label: investor.Label()})
	}
	cfg, err := read(b, func(ctx context.Context, s *core.Services) (*domain.CompanySettings, error) {
		return s.Settings.GetCompanySettings(ctx)
	})
	if err != nil || cfg == nil {
		return rules, nil
	}
	investor, reason := cfg.InvestorTypeOrDerived()
	rules.InvestorType = investor
	rules.InvestorLabel = investor.Label()
	rules.InvestorReason = reason
	rules.LegalForm = cfg.LegalForm

	// Der Satz je Fondsart, wie er sich aus der eingestellten Anlegerstellung
	// ergibt — und, wo er sich nicht ergibt, der Grund dafür. Beides gehört an
	// dieselbe Stelle: eine Maske, die nur den Satz zeigt, verschweigt die
	// Fälle, in denen es keinen gibt.
	for _, class := range accounting.AllFundClasses() {
		if !class.IsFund() {
			continue
		}
		exemption, err := accounting.PartialExemptionFor(class, investor)
		info := exemptionInfo{Class: class, Label: class.Label()}
		if err != nil {
			info.Problem = err.Error()
		} else {
			info.Permille = exemption.Permille
			info.Source = exemption.Source
			info.Explanation = exemption.Explanation
		}
		rules.Exemptions = append(rules.Exemptions, info)
	}
	return rules, nil
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
