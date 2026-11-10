package accounting

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

func partialDisposalSchedule(plan AfAPlan) ([]AfAYear, error) {
	if plan.Quantity <= 0 {
		return nil, fmt.Errorf("für einen Teilabgang fehlt die Stückzahl")
	}
	parts := append([]AfAPartialDisposal(nil), plan.PartialDisposals...)
	remaining := plan.Quantity
	for _, part := range parts {
		if part.Quantity <= 0 || part.Quantity > remaining {
			return nil, fmt.Errorf("ungültige Stückzahl beim Teilabgang")
		}
		remaining -= part.Quantity
	}
	if remaining > 0 {
		parts = append(parts, AfAPartialDisposal{Date: plan.DisposalDate, Quantity: remaining})
	}
	plan.PartialDisposals = nil
	byYear := map[int]AfAYear{}
	remainingCost := plan.Cost
	splitMap := func(source map[int]domain.Cents, quantity domain.Units) map[int]domain.Cents {
		result := map[int]domain.Cents{}
		for year, amount := range source {
			result[year] = domain.MulRound(amount, int64(quantity), int64(plan.Quantity))
		}
		return result
	}
	costChanges := make([]map[int]domain.Cents, len(parts))
	impairments := make([]map[int]domain.Cents, len(parts))
	for i := range parts {
		costChanges[i] = map[int]domain.Cents{}
		impairments[i] = map[int]domain.Cents{}
	}
	for _, change := range plan.DatedAdjustments {
		var held domain.Units
		for _, part := range parts {
			if part.Date == "" || part.Date >= change.Date {
				held += part.Quantity
			}
		}
		cost, impairment := change.CostAmount, change.ImpairmentAmount
		for i, part := range parts {
			if part.Date != "" && part.Date < change.Date {
				continue
			}
			costShare := domain.MulRound(cost, int64(part.Quantity), int64(held))
			impairmentShare := domain.MulRound(impairment, int64(part.Quantity), int64(held))
			costChanges[i][change.FiscalYear] += costShare
			impairments[i][change.FiscalYear] += impairmentShare
			held -= part.Quantity
			cost -= costShare
			impairment -= impairmentShare
		}
	}
	for i, part := range parts {
		sub := plan
		sub.Cost = domain.MulRound(plan.Cost, int64(part.Quantity), int64(plan.Quantity))
		if i == len(parts)-1 {
			sub.Cost = remainingCost
		}
		remainingCost -= sub.Cost
		sub.DisposalDate = part.Date
		sub.BasisChangesByYear = splitMap(plan.BasisChangesByYear, part.Quantity)
		sub.ImpairmentsByYear = splitMap(plan.ImpairmentsByYear, part.Quantity)
		if len(plan.DatedAdjustments) > 0 {
			sub.BasisChangesByYear = costChanges[i]
			sub.ImpairmentsByYear = impairments[i]
		}
		rows, err := BuildAfASchedule(sub)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			sum, exists := byYear[row.FiscalYear]
			if !exists {
				sum = row
				sum.OpeningBookValue = 0
				sum.Amount = 0
				sum.SpecialAmount = 0
				sum.ClosingBookValue = 0
				sum.TaxAmount = 0
				sum.TaxClosingBookValue = 0
			}
			sum.OpeningBookValue += row.OpeningBookValue
			sum.Amount += row.Amount
			sum.SpecialAmount += row.SpecialAmount
			sum.TaxAmount += row.TaxAmount
			if part.Date == "" || row.FiscalYear < domain.GetFiscalYearForDate(part.Date, fiscalStart(plan)) {
				sum.ClosingBookValue += row.ClosingBookValue
				sum.TaxClosingBookValue += row.TaxClosingBookValue
			}
			if row.Months > sum.Months {
				sum.Months = row.Months
			}
			byYear[row.FiscalYear] = sum
		}
	}
	rows := make([]AfAYear, 0, len(byYear))
	for _, row := range byYear {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].FiscalYear < rows[j].FiscalYear })
	return rows, nil
}
