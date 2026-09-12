package domain

import "context"

// BankAccount binds a real account to its own ledger account across all years.
type BankAccount struct {
	IBAN          string `gorm:"primaryKey;size:34" json:"iban"`
	Name          string `gorm:"type:text;serializer:encrypted" json:"name"`
	LedgerAccount string `gorm:"size:10;uniqueIndex;not null" json:"ledgerAccount"`
	Currency      string `gorm:"size:3" json:"currency"`
}
type BankAccountRepository interface {
	FindAll(ctx context.Context) ([]BankAccount, error)
	Save(ctx context.Context, account *BankAccount) error
}
