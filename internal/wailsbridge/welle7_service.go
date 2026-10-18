package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 7: Aufgabenliste, Monatsabschluss,
// Zuordnungsvorschlag und Prüfpfad.
//
// Sie stehen in einer eigenen Datei, weil sie eine Sache sind: die Antwort auf
// die Frage, was eine Anwenderin ohne Buchhaltungsausbildung als Nächstes tun
// soll — und wie sie das, was sie tut, wieder findet.

// --- Aufgabenliste --------------------------------------------------------

// GetTasks liefert die Aufgabenliste in ihren drei Gruppen. Der Prüfermodus
// kommt aus dem Kern und wird mitgegeben.
func (b *BuchfinkBridge) GetTasks() (*domain.TaskList, error) {
	cfg := b.host.Config()
	opts := service.TaskOptions{FiscalYear: b.viewYear()}
	if cfg.ReadOnly {
		opts.ReadOnlyUntil, opts.ReadOnlyReason = cfg.ReadOnlyUntil, cfg.ReadOnlyReason
	}
	return readOr(b, func() (*domain.TaskList, error) {
		empty := &domain.TaskList{}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*domain.TaskList, error) {
		list, err := s.Tasks.Tasks(ctx, opts)
		if err != nil {
			return nil, err
		}
		list.EnsureLists()
		return list, nil
	})
}

// --- Monatsabschluss ------------------------------------------------------

// GetMonthCloseState liefert den Stand eines Monats ("JJJJ-MM") in seinen drei
// Schritten.
func (b *BuchfinkBridge) GetMonthCloseState(month string) (*service.MonthCloseState, error) {
	return readOr(b, func() (*service.MonthCloseState, error) {
		empty := &service.MonthCloseState{Month: month}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.MonthCloseState, error) {
		state, err := s.MonthClose.State(ctx, month)
		if err != nil {
			return nil, err
		}
		state.EnsureLists()
		return state, nil
	})
}

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

// GetAuditTrail liefert den Prüfpfad eines Belegs zur Anzeige.
//
// Dieselbe Kette, die auch ausgegeben wird: die Anwenderin sieht auf dem Schirm,
// was der Prüfer in der Datei bekommt.
func (b *BuchfinkBridge) GetAuditTrail(receiptID uint) (*service.AuditTrail, error) {
	return readOr(b, func() (*service.AuditTrail, error) {
		empty := &service.AuditTrail{ReceiptID: receiptID}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.AuditTrail, error) {
		trail, err := s.AuditTrail.Trail(ctx, receiptID)
		if err != nil {
			return nil, err
		}
		trail.EnsureLists()
		return trail, nil
	})
}

// SaveServiceProof schreibt den Leistungsnachweis an einen Eingangsbeleg.
func (b *BuchfinkBridge) SaveServiceProof(receiptID uint, text, date string) (*domain.Receipt, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Receipt, error) {
		return s.Receipts.SaveServiceProof(ctx, receiptID, text, date)
	})
}
