package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// readSpecificBalances liest die Zuordnung aus den Buchungszeilen einschließlich
// Generalumkehr. Saldenvorträge enthalten keine erneute Zuordnung.
func readSpecificBalances(ctx context.Context, repo domain.JournalRepository, cutoff string) (map[string]map[uint]domain.Cents, error) {
	out := map[string]map[uint]domain.Cents{}
	for _, account := range []string{accounting.AccountEWBShortTerm, accounting.AccountEWBLongTerm} {
		out[account] = map[uint]domain.Cents{}
		entries, err := repo.FindByAccountRange(ctx, account, "", cutoff)
		if err != nil {
			return nil, err
		}
		initialYear := 0
		if len(entries) > 0 && entries[0].Source == domain.EntrySourceOpening {
			initialYear = entries[0].FiscalYear
		}
		for _, e := range entries {
			if e.Source == domain.EntrySourceOpening && e.FiscalYear != initialYear {
				continue
			}
			for _, l := range e.Lines {
				if l.Account != account {
					continue
				}
				amount := l.Amount
				if l.Side == domain.SideDebit {
					amount = -amount
				}
				out[account][l.AllowanceEntryID] += amount
			}
		}
	}
	return out, nil
}

func specificBalances(ctx context.Context, repo domain.JournalRepository, cutoff string) (map[string]map[uint]domain.Cents, error) {
	balances, err := readSpecificBalances(ctx, repo, cutoff)
	if err != nil {
		return nil, err
	}
	for account, rows := range balances {
		if rows[0] != 0 {
			return nil, fmt.Errorf("auf %s steht eine Einzelwertberichtigung ohne Zuordnung zur Forderung; vor der Ausbuchung muss die Zuordnung fachlich geklärt werden", account)
		}
	}
	return balances, nil
}
