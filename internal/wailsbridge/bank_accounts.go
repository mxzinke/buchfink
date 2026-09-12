package wailsbridge

import (
	"context"
	"fmt"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

func (b *BuchfinkBridge) GetBankAccounts() ([]domain.BankAccount, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.bankSvc == nil {
		return []domain.BankAccount{}, nil
	}
	return b.bankSvc.Accounts(context.Background())
}
func (b *BuchfinkBridge) PreviewBankStatement(path string) (*service.BankImportPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.bankSvc == nil {
		return nil, fmt.Errorf("Bankimport ist noch nicht eingerichtet")
	}
	return b.bankSvc.PreviewFile(context.Background(), path)
}
func (b *BuchfinkBridge) ConfigureBankAccounts(accounts []domain.BankAccount) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.bankSvc == nil {
		return fmt.Errorf("Bankimport ist noch nicht eingerichtet")
	}
	return b.bankSvc.ConfigureAccounts(context.Background(), accounts)
}
