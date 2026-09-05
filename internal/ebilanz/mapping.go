package ebilanz

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// MappingRow ist ein Konto mit Saldo, seine Gliederungsposition und das
// Taxonomie-Element, unter dem es in der Instanz erscheint.
type MappingRow struct {
	Account string       `json:"account"`
	Name    string       `json:"name"`
	Balance domain.Cents `json:"balance"`
	// PositionKey ist der Schlüssel der Gliederung, PositionLabel ihre
	// Bezeichnung nach dem Gesetzeswortlaut.
	PositionKey   string `json:"positionKey"`
	PositionLabel string `json:"positionLabel"`
	Element       string `json:"element"`
	Verified      bool   `json:"verified"`
	// Finding benennt, was fehlt, wenn etwas fehlt.
	Finding string `json:"finding,omitempty"`
}

// MappingReport ist der Zuordnungsbericht vor dem Export.
//
// Er läuft vor der Erzeugung und nicht danach: eine E-Bilanz, in der ein Konto
// auf einer Sammelposition verschwunden ist, lässt sich beim Finanzamt nicht
// mehr zurückholen. Blockierende Befunde verhindern deshalb die Erzeugung; die
// Bilanzansicht bleibt davon unberührt, dort erscheinen dieselben Konten unter
// „Nicht zugeordnet".
type MappingReport struct {
	FiscalYear      int    `json:"fiscalYear"`
	TaxonomyVersion string `json:"taxonomyVersion"`
	TaxonomyDate    string `json:"taxonomyDate"`
	TaxonomyNote    string `json:"taxonomyNote"`
	// Rows ist die echte Zuordnung aller Konten mit Saldo.
	Rows []MappingRow `json:"rows"`
	// Blocking sind die Konten ohne Gliederungsposition oder ohne
	// Taxonomie-Element.
	Blocking []MappingRow `json:"blocking"`
	// Fallbacks zählt die Auffangpositionen der Gliederung aus.
	Fallbacks []domain.FallbackCount `json:"fallbacks"`
	// Unverified zählt die Elemente, deren Name noch gegen die amtliche
	// Taxonomie zu prüfen ist.
	Unverified int  `json:"unverified"`
	CanExport  bool `json:"canExport"`
}

// BuildMappingReport prüft die Konten mit Saldo gegen Gliederung und Taxonomie.
func BuildMappingReport(fiscalYear int, stmt *domain.Statement, accounts []domain.Account) (*MappingReport, error) {
	tax, err := LoadTaxonomy()
	if err != nil {
		return nil, err
	}

	report := &MappingReport{
		FiscalYear:      fiscalYear,
		TaxonomyVersion: tax.Version,
		TaxonomyDate:    tax.Date,
		TaxonomyNote:    tax.Note,
		CanExport:       true,
	}
	if stmt != nil {
		report.Fallbacks = stmt.Assignment.Fallbacks
	}

	for _, acc := range accounting.AccountsWithBalance(accounts) {
		row := MappingRow{
			Account: acc.Number, Name: acc.Name,
			Balance: acc.DebitSum - acc.CreditSum,
		}

		key, known := accounting.StatementKeyForAccount(acc)
		switch {
		case !known:
			row.Finding = fmt.Sprintf(
				"Das Konto trägt die SKR04-Position %q, die die Gliederung nicht kennt.", acc.PositionID)
		default:
			row.PositionKey = key
			row.PositionLabel = accounting.LineLabel(key)
			element, hasElement := ElementFor(key)
			if !hasElement {
				row.Finding = fmt.Sprintf(
					"Für die Gliederungsposition %q ist kein Element der Taxonomie %s hinterlegt.",
					row.PositionLabel, tax.Version)
			} else {
				row.Element = element.Element
				row.Verified = element.Verified
				if !element.Verified {
					report.Unverified++
				}
			}
		}

		report.Rows = append(report.Rows, row)
		if row.Finding != "" {
			report.Blocking = append(report.Blocking, row)
			report.CanExport = false
		}
	}

	sort.Slice(report.Rows, func(i, j int) bool { return report.Rows[i].Account < report.Rows[j].Account })
	return report, nil
}

// BlockingError fasst die blockierenden Befunde in eine Fehlermeldung, die
// sagt, welche Konten zu klären sind.
func (r *MappingReport) BlockingError() error {
	if r == nil || len(r.Blocking) == 0 {
		return nil
	}
	msg := fmt.Sprintf(
		"die E-Bilanz kann nicht erzeugt werden: %d Konten mit Saldo haben keine Zuordnung", len(r.Blocking))
	limit := len(r.Blocking)
	if limit > 5 {
		limit = 5
	}
	for _, row := range r.Blocking[:limit] {
		msg += fmt.Sprintf("; %s %s (%s)", row.Account, row.Name, row.Finding)
	}
	if len(r.Blocking) > limit {
		msg += fmt.Sprintf("; und %d weitere", len(r.Blocking)-limit)
	}
	return fmt.Errorf("%s", msg)
}
