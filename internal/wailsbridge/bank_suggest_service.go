package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- Bankabgleich ---------------------------------------------------------

// SuggestBankMatches liefert die wahrscheinlichsten Zuordnungen zu einem
// Bankumsatz. Gebucht wird davon nichts.
func (b *BuchfinkBridge) SuggestBankMatches(bankTxID uint) (*service.BankSuggestions, error) {
	return readOr(b, func() (*service.BankSuggestions, error) {
		empty := &service.BankSuggestions{BankTxID: bankTxID}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.BankSuggestions, error) {
		suggestions, err := s.Bank.Suggest(ctx, bankTxID)
		if err != nil {
			return nil, err
		}
		suggestions.EnsureLists()
		return suggestions, nil
	})
}

// GetBankRules liefert die gelernten Zuordnungen wiederkehrender Umsätze.
func (b *BuchfinkBridge) GetBankRules() ([]domain.BankRule, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.BankRule, error) {
		return s.Bank.Rules(ctx)
	})
}

// DeleteBankRule entfernt eine gelernte Zuordnung.
func (b *BuchfinkBridge) DeleteBankRule(id uint) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Bank.DeleteRule(ctx, id)
	})
}
