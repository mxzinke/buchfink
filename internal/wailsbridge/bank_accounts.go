package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

func (b *BuchfinkBridge) GetBankAccounts() ([]domain.BankAccount, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.BankAccount, error) {
		return s.Bank.Accounts(ctx)
	})
}
func (b *BuchfinkBridge) PreviewBankStatement(path string) (*service.BankImportPreview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.BankImportPreview, error) {
		return s.Bank.PreviewFile(ctx, path)
	})
}
func (b *BuchfinkBridge) ConfigureBankAccounts(accounts []domain.BankAccount) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Bank.ConfigureAccounts(ctx, accounts)
	}, core.AppOnly())
}

// SetInvoiceBankAccount legt das Konto für Rechnungen fest.
func (b *BuchfinkBridge) SetInvoiceBankAccount(iban string) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Bank.SetInvoiceAccount(ctx, iban)
	}, core.AppOnly())
}
