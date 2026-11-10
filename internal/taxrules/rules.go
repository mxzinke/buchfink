package taxrules

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// EntryRules ergänzt die festen Journalprüfungen. Jede Regel legt fest, ob sie
// auch die Generalumkehr vorhandener Buchungen prüfen muss.
func EntryRules() []domain.EntryRule {
	return []domain.EntryRule{
		{Name: "accrual_taxation", Position: domain.EntryRulesBeforeValidation, AppliesToReversal: false, Check: ensureAccrualTaxation},
		{Name: "entertainment", Position: domain.EntryRulesAfterValidation, AppliesToReversal: true, Check: validateEntertainment},
		{Name: "collective_accounts", Position: domain.EntryRulesAfterAccount, AppliesToReversal: true, Check: validateCollectiveAccount},
		{Name: "lawful_tax_disclosure", Position: domain.EntryRulesAfterAccounts, AppliesToReversal: false, Check: ensureLawfulTaxDisclosure},
		{Name: "tax_lines", Position: domain.EntryRulesAfterAccounts, AppliesToReversal: false, Check: validateTaxLines},
		{Name: "manual_tax_lines", Position: domain.EntryRulesManual, AppliesToReversal: false, Check: func(_ context.Context, e *domain.JournalEntry, _ domain.EntryRuleView) error {
			if e.Source == domain.EntrySourceManual {
				return validateManualTaxLines(e)
			}
			return nil
		}},
		{Name: "receipt_header", Position: domain.EntryRulesAfterPeriods, AppliesToReversal: false, Check: ensureReceiptHeader},
	}
}

func validateEntertainment(_ context.Context, e *domain.JournalEntry, _ domain.EntryRuleView) error {
	if e.Entertainment != nil {
		return e.Entertainment.Validate()
	}
	return nil
}
