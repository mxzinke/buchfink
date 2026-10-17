package ebilanz

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// MappingRow ist ein Konto mit Saldo, seine Gliederungsposition und die
// Position der Taxonomie, unter der es in Instanz und Kontennachweis steht.
type MappingRow struct {
	Account       string       `json:"account"`
	Name          string       `json:"name"`
	Balance       domain.Cents `json:"balance"`
	PositionKey   string       `json:"positionKey"`
	PositionLabel string       `json:"positionLabel"`
	Element       string       `json:"element"`
	// Fallback heißt: Auffangposition der Gliederungsposition statt einer
	// Kontenregel.
	Fallback bool   `json:"fallback"`
	Finding  string `json:"finding,omitempty"`
}

// MappingReport ist der Prüfbericht vor dem Export: wohin jedes Konto geht,
// welche Fassung gilt und was die Erzeugung verhindert.
type MappingReport struct {
	FiscalYear      int    `json:"fiscalYear"`
	TaxonomyVersion string `json:"taxonomyVersion"`
	TaxonomyDate    string `json:"taxonomyDate"`
	TaxonomyNote    string `json:"taxonomyNote"`
	// AccountingStandard ist HAOE (Einheitsbilanz) oder HGBM (Handelsbilanz
	// mit Überleitungsrechnung).
	AccountingStandard string       `json:"accountingStandard"`
	Rows               []MappingRow `json:"rows"`
	// Blocking sind die blockierenden Befunde, je Zeile mit den betroffenen
	// Konten.
	Blocking  []MappingRow           `json:"blocking"`
	Fallbacks []domain.FallbackCount `json:"fallbacks"`
	Findings  []Finding              `json:"findings"`
	CanExport bool                   `json:"canExport"`
}

func (g *generator) report() *MappingReport {
	r := &MappingReport{
		TaxonomyVersion:    g.tax.Version,
		TaxonomyDate:       g.tax.Release,
		TaxonomyNote:       g.version.Reason + " " + g.tax.Copyright + "; " + g.tax.Notice,
		AccountingStandard: "HAOE",
		Rows:               []MappingRow{},
		Blocking:           []MappingRow{},
		Fallbacks:          []domain.FallbackCount{},
		Findings:           append([]Finding{}, g.findings...),
		CanExport:          true,
	}
	if g.commercial {
		r.AccountingStandard = "HGBM"
	}
	_, _ = fmt.Sscanf(g.end, "%4d", &r.FiscalYear)
	if g.in.Statement != nil {
		r.FiscalYear = g.in.Statement.FiscalYear
	}
	fallbacks := map[string]*domain.FallbackCount{}
	for _, p := range g.placements {
		row := MappingRow{Account: p.Account, Name: p.Name, Balance: p.Balance, PositionKey: p.PositionKey,
			Element: p.Element, Fallback: p.Fallback, Finding: p.Finding}
		if p.PositionKey != "" {
			row.PositionLabel = accounting.LineLabel(p.PositionKey)
		}
		r.Rows = append(r.Rows, row)
		if p.Fallback && p.Finding == "" {
			fb := fallbacks[p.PositionKey]
			if fb == nil {
				fb = &domain.FallbackCount{Key: p.PositionKey, Label: row.PositionLabel}
				fallbacks[p.PositionKey] = fb
			}
			fb.Accounts++
			fb.Amount += p.Value
		}
	}
	for _, fb := range fallbacks {
		r.Fallbacks = append(r.Fallbacks, *fb)
	}
	sort.Slice(r.Fallbacks, func(i, j int) bool { return r.Fallbacks[i].Key < r.Fallbacks[j].Key })
	sort.Slice(r.Rows, func(i, j int) bool { return r.Rows[i].Account < r.Rows[j].Account })

	names := map[string]string{}
	for _, p := range g.placements {
		names[p.Account] = p.Name
	}
	for _, f := range g.findings {
		if f.Severity != SeverityBlocking {
			continue
		}
		r.CanExport = false
		row := MappingRow{Account: strings.Join(f.Accounts, ", "), Element: f.Element, Finding: f.Message}
		if len(f.Accounts) == 1 {
			row.Name = names[f.Accounts[0]]
		} else if el, ok := g.tax.Element(f.Element); ok {
			row.Name = el.Label
		}
		r.Blocking = append(r.Blocking, row)
	}
	return r
}

// BlockingError fasst die blockierenden Befunde in eine Fehlermeldung.
func (r *MappingReport) BlockingError() error {
	if r == nil || len(r.Blocking) == 0 {
		return nil
	}
	limit := len(r.Blocking)
	if limit > 5 {
		limit = 5
	}
	parts := make([]string, 0, limit+1)
	for _, row := range r.Blocking[:limit] {
		who := row.Account
		if who == "" {
			who = row.Name
		}
		parts = append(parts, strings.TrimSpace(who+": "+row.Finding))
	}
	if len(r.Blocking) > limit {
		parts = append(parts, fmt.Sprintf("und %d weitere", len(r.Blocking)-limit))
	}
	return fmt.Errorf("die E-Bilanz kann nicht erzeugt werden (%d Befunde): %s", len(r.Blocking), strings.Join(parts, "; "))
}

func sortStrings(s []string) { sort.Strings(s) }
