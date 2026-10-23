package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Einheitlichkeit des Wahlrechts (§ 6 Abs. 2a Satz 5 EStG)
// -------------------------------------------------------------------------

// PoolConsistencyRow ist ein Zugang, der zur Frage der Einheitlichkeit gehört.
type PoolConsistencyRow struct {
	AssetID         uint                      `json:"assetId"`
	InventoryNumber string                    `json:"inventoryNumber"`
	Name            string                    `json:"name"`
	AcquisitionDate string                    `json:"acquisitionDate"`
	Cost            domain.Cents              `json:"cost"`
	Method          domain.DepreciationMethod `json:"method"`
}

// PoolConsistencyReport beantwortet für ein Wirtschaftsjahr, ob das Wahlrecht
// des § 6 Abs. 2a EStG einheitlich ausgeübt wurde.
type PoolConsistencyReport struct {
	FiscalYear int `json:"fiscalYear"`
	// LowerLimit und UpperLimit sind die Grenzen, zwischen denen sich die Frage
	// überhaupt stellt.
	LowerLimit domain.Cents `json:"lowerLimit"`
	UpperLimit domain.Cents `json:"upperLimit"`
	// Pooled und Immediate sind die Zugänge des Jahres in diesem Wertbereich.
	Pooled    []PoolConsistencyRow `json:"pooled"`
	Immediate []PoolConsistencyRow `json:"immediate"`
	// Consistent sagt, ob das Wahlrecht einheitlich ausgeübt ist.
	Consistent bool   `json:"consistent"`
	Note       string `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *PoolConsistencyReport) EnsureLists() {
	if r.Pooled == nil {
		r.Pooled = make([]PoolConsistencyRow, 0)
	}
	if r.Immediate == nil {
		r.Immediate = make([]PoolConsistencyRow, 0)
	}
}

// PoolConsistency prüft die Einheitlichkeit des Wahlrechts eines Jahres.
//
// § 6 Abs. 2a Satz 5 EStG: „Die Wahlrechte nach den Sätzen 1 und 4 sind für alle
// in einem Wirtschaftsjahr angeschafften … Wirtschaftsgüter einheitlich
// auszuüben." Wer im Mai poolt, darf im Oktober kein gleichartiges Gut mehr
// sofort abziehen — und umgekehrt. Der Wertbereich, in dem sich die Frage
// stellt, liegt zwischen der Sammelposten-Untergrenze und der GWG-Grenze; unter
// ihr gibt es keinen Sammelposten, über ihr keinen Sofortabzug.
func (s *AssetService) PoolConsistency(ctx context.Context, year int) (*PoolConsistencyReport, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	out := &PoolConsistencyReport{FiscalYear: year, Consistent: true}
	out.EnsureLists()

	startMonth := s.fiscalYearStartMonth(ctx)
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range assets {
		a := &assets[i]
		if domain.GetFiscalYearForDate(a.AcquisitionDate, startMonth) != year {
			continue
		}
		params, err := accounting.AfAParametersFor(a.AcquisitionDate)
		if err != nil {
			continue
		}
		out.LowerLimit, out.UpperLimit = params.PoolLowerLimit, params.GWGImmediateLimit
		if a.AcquisitionCost <= params.PoolLowerLimit || a.AcquisitionCost > params.GWGImmediateLimit {
			continue
		}
		row := PoolConsistencyRow{
			AssetID: a.ID, InventoryNumber: a.InventoryNumber, Name: a.Name,
			AcquisitionDate: a.AcquisitionDate, Cost: a.AcquisitionCost, Method: a.Method,
		}
		switch a.Method {
		case domain.DepreciationPool:
			out.Pooled = append(out.Pooled, row)
		case domain.DepreciationImmediate:
			out.Immediate = append(out.Immediate, row)
		}
	}

	out.Consistent = len(out.Pooled) == 0 || len(out.Immediate) == 0
	if out.Consistent {
		out.Note = fmt.Sprintf(
			"Das Wahlrecht des § 6 Abs. 2a EStG ist im Wirtschaftsjahr %d einheitlich ausgeübt. "+
				"Es betrifft die Zugänge über %s € und bis %s € netto.",
			year, out.LowerLimit, out.UpperLimit)
		return out, nil
	}
	out.Note = fmt.Sprintf(
		"Im Wirtschaftsjahr %d stehen %d Wirtschaftsgüter im Sammelposten und %d im Sofortabzug. "+
			"§ 6 Abs. 2a Satz 5 EStG verlangt die einheitliche Ausübung für alle Wirtschaftsgüter "+
			"eines Wirtschaftsjahres zwischen %s € und %s € netto — eines der beiden ist zu ändern.",
		year, len(out.Pooled), len(out.Immediate), out.LowerLimit, out.UpperLimit)
	return out, nil
}

// checkPoolConsistency weist einen Zugang zurück, der das Wahlrecht des Jahres
// brechen würde.
func (s *AssetService) checkPoolConsistency(ctx context.Context, asset *domain.FixedAsset) error {
	if asset.Method != domain.DepreciationPool && asset.Method != domain.DepreciationImmediate {
		return nil
	}
	params, err := accounting.AfAParametersFor(asset.AcquisitionDate)
	if err != nil {
		return err
	}
	if asset.AcquisitionCost <= params.PoolLowerLimit ||
		asset.AcquisitionCost > params.GWGImmediateLimit {
		// Außerhalb des Wertbereichs gibt es kein Wahlrecht, das einheitlich
		// auszuüben wäre: unter der Untergrenze kennt § 6 Abs. 2a EStG keinen
		// Sammelposten, über der GWG-Grenze § 6 Abs. 2 EStG keinen Sofortabzug.
		return nil
	}

	year := domain.GetFiscalYearForDate(asset.AcquisitionDate, s.fiscalYearStartMonth(ctx))
	report, err := s.PoolConsistency(ctx, year)
	if err != nil {
		return err
	}
	conflicting := report.Immediate
	other := "Sofortabzug"
	if asset.Method == domain.DepreciationImmediate {
		conflicting, other = report.Pooled, "Sammelposten"
	}
	for _, row := range conflicting {
		if row.AssetID == asset.ID {
			continue
		}
		return fmt.Errorf(
			"im Wirtschaftsjahr %d steht %s (%s) bereits im %s. § 6 Abs. 2a Satz 5 EStG verlangt, "+
				"das Wahlrecht für alle Wirtschaftsgüter eines Wirtschaftsjahres zwischen %s € und "+
				"%s € netto einheitlich auszuüben — entweder alle in den Sammelposten oder alle in "+
				"den Sofortabzug",
			year, row.Name, row.InventoryNumber, other, report.LowerLimit, report.UpperLimit)
	}
	return nil
}
