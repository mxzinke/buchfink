package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
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
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return []domain.FixedAsset{}, nil
	}
	return b.assetSvc.List(context.Background(), domain.AssetClass(class))
}

// GetAssetSummary aggregates the register of one class for the head of the view.
func (b *BuchfinkBridge) GetAssetSummary(class string) (*service.AssetSummary, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return &service.AssetSummary{}, nil
	}
	sum, err := b.assetSvc.Summary(context.Background(), domain.AssetClass(class))
	if err != nil {
		return nil, err
	}
	return &sum, nil
}

// GetFixedAsset returns one Anlagegut with its AfA-Plan, seinen Bewegungen und
// den Erklärungen, die zu genau diesem Gut gehören.
func (b *BuchfinkBridge) GetFixedAsset(id uint) (*service.AssetDetail, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.Get(context.Background(), id)
}

// SaveFixedAsset creates or updates an Anlagegut.
func (b *BuchfinkBridge) SaveFixedAsset(asset domain.FixedAsset) (*domain.FixedAsset, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.Save(context.Background(), &asset)
}

// DeleteFixedAsset removes an Anlagegut that never carried a booking.
func (b *BuchfinkBridge) DeleteFixedAsset(id uint) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.Delete(context.Background(), id)
}

// RecordAssetCostAdjustment records nachträgliche Anschaffungskosten oder eine
// Anschaffungspreisminderung.
func (b *BuchfinkBridge) RecordAssetCostAdjustment(req service.CostAdjustmentRequest) (*domain.FixedAsset, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.RecordCostAdjustment(context.Background(), req)
}

// GetAssetAccounts returns the curated Anlagekonten of one class, each with its
// AfA-Konto und dem Hinweis, was auf ihm steht.
func (b *BuchfinkBridge) GetAssetAccounts(class string) []accounting.AssetAccount {
	return accounting.AssetAccounts(domain.AssetClass(class))
}

// ClassifyAcquisition answers the first question of every acquisition: Aufwand
// sofort, Sammelposten oder aktivieren?
//
// netCost is in Cents and net of Vorsteuer; selfUsable is the answer to the
// question Buchfink cannot decide itself — ob das Wirtschaftsgut für sich allein
// nutzbar ist.
func (b *BuchfinkBridge) ClassifyAcquisition(netCost int64, date string, selfUsable bool) (*accounting.AcquisitionAdvice, error) {
	advice, err := accounting.ClassifyAcquisition(domain.Cents(netCost), date, selfUsable)
	if err != nil {
		return nil, err
	}
	return &advice, nil
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
	// SpecialMaxPermille und SpecialPeriodYears sind die Grenzen der
	// Sonderabschreibung nach § 7g Abs. 5 EStG: höchstens 40 % der
	// Anschaffungskosten, verteilbar auf das Anschaffungsjahr und die vier
	// folgenden.
	SpecialMaxPermille int               `json:"specialMaxPermille"`
	SpecialPeriodYears int               `json:"specialPeriodYears"`
	Methods            []assetMethodInfo `json:"methods"`
}

type assetMethodInfo struct {
	Method domain.DepreciationMethod `json:"method"`
	Label  string                    `json:"label"`
	// Classes names the Anlagenklassen the method is available for. Finanzanlagen
	// tragen keine planmäßige Abschreibung — das steht hier und nicht als
	// Sonderfall in der Maske.
	Classes []domain.AssetClass `json:"classes"`
	Hint    string              `json:"hint"`
}

// GetAssetRules hands the Wertgrenzen, das Zeitfenster der degressiven AfA und
// die zulässigen Methoden an die Oberfläche.
func (b *BuchfinkBridge) GetAssetRules() (*AssetRules, error) {
	b.mu.RLock()
	year := b.currentYear
	b.mu.RUnlock()

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
		FiscalYear:         year,
		GWGImmediateLimit:  params.GWGImmediateLimit,
		GWGRecordFrom:      params.GWGRecordThreshold,
		PoolLowerLimit:     params.PoolLowerLimit,
		PoolUpperLimit:     params.PoolUpperLimit,
		PoolYears:          params.PoolYears,
		DegressiveWindows:  accounting.DegressiveWindows(),
		SpecialMaxPermille: accounting.SpecialMaxPermille,
		SpecialPeriodYears: accounting.SpecialPeriodYears,
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

// PreviewDepreciationPlan rechnet den Abschreibungsplan für eine Eingabe, die
// noch kein Anlagegut ist. Die Erfassungsmaske zeigt damit beim Tippen, was die
// Nutzungsdauer bedeutet, statt es den Nutzer schätzen zu lassen.
func (b *BuchfinkBridge) PreviewDepreciationPlan(req service.PlanRequest) ([]accounting.AfAYear, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.PreviewPlan(context.Background(), req)
}

// GetDepreciationRun computes the AfA of the active fiscal year without writing
// anything.
func (b *BuchfinkBridge) GetDepreciationRun() (*service.DepreciationRun, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return &service.DepreciationRun{}, nil
	}
	return b.assetSvc.Run(context.Background())
}

// BookDepreciationRun writes the AfA of the active fiscal year.
func (b *BuchfinkBridge) BookDepreciationRun(req service.BookDepreciationRequest) (*service.DepreciationResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.BookDepreciation(context.Background(), req)
}

// BookAssetImpairment writes an außerplanmäßige Abschreibung.
func (b *BuchfinkBridge) BookAssetImpairment(req service.ImpairmentRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.BookImpairment(context.Background(), req)
}

// BookAssetWriteUp writes a Zuschreibung.
func (b *BuchfinkBridge) BookAssetWriteUp(req service.WriteUpRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.BookWriteUp(context.Background(), req)
}

// TransferFixedAsset bucht die Fertigstellung: von der Anlage im Bau auf ihr
// endgültiges Konto, und ab da läuft die Abschreibung.
func (b *BuchfinkBridge) TransferFixedAsset(req service.TransferRequest) (*domain.FixedAsset, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.Transfer(context.Background(), req)
}

// PreviewAssetDisposal computes an Abgang without writing it.
func (b *BuchfinkBridge) PreviewAssetDisposal(req service.DisposalRequest) (*service.DisposalPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.PreviewDisposal(context.Background(), req)
}

// DisposeFixedAsset books the Abgang.
func (b *BuchfinkBridge) DisposeFixedAsset(req service.DisposalRequest) (*service.DisposalResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.Dispose(context.Background(), req)
}

// GetAnlagenspiegel returns the Entwicklung des Anlagevermögens of the active
// fiscal year.
func (b *BuchfinkBridge) GetAnlagenspiegel() (*domain.Anlagenspiegel, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return &domain.Anlagenspiegel{}, nil
	}
	return b.assetSvc.Anlagenspiegel(context.Background())
}

// GetAssetAcquisitionCandidates lists bookings on Anlagekonten that no Anlagegut
// points at yet.
func (b *BuchfinkBridge) GetAssetAcquisitionCandidates() ([]service.AcquisitionCandidate, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return []service.AcquisitionCandidate{}, nil
	}
	return b.assetSvc.AcquisitionCandidates(context.Background())
}

// GetSammelposten returns the Sammelposten of a fiscal year, or nothing if none
// was formed. Zero means the active fiscal year.
func (b *BuchfinkBridge) GetSammelposten(fiscalYear int) (*domain.FixedAsset, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, nil
	}
	return b.assetSvc.Pool(context.Background(), fiscalYear)
}

// BookAssetMaintenance bucht Erhaltungsaufwand und verknüpft ihn mit dem
// Anlagegut, ohne dessen Buchwert anzurühren.
func (b *BuchfinkBridge) BookAssetMaintenance(req service.MaintenanceRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.BookMaintenance(context.Background(), req)
}

// BookAssetIncome bucht einen laufenden Ertrag aus einer Finanzanlage —
// Dividende, Ausschüttung, Zins — und hängt ihn an den Anteil, aus dem er stammt.
func (b *BuchfinkBridge) BookAssetIncome(req service.AssetIncomeRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.BookAssetIncome(context.Background(), req)
}

// ValuateAssetCurrency rechnet eine Fremdwährungs-Finanzanlage zum
// Devisenkassamittelkurs des Stichtags um (§ 256a HGB). Es bucht nichts: es
// nennt den Betrag, den eine außerplanmäßige Abschreibung oder eine
// Zuschreibung hätte.
func (b *BuchfinkBridge) ValuateAssetCurrency(req service.CurrencyValuationRequest) (*service.CurrencyValuation, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("Anlagenbuchhaltung ist noch nicht initialisiert")
	}
	return b.assetSvc.ValuateCurrency(context.Background(), req)
}
