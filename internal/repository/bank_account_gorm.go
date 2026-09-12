package repository

import (
	"context"
	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type bankAccountRepository struct{ db *gorm.DB }

func NewBankAccountRepository(db *gorm.DB) domain.BankAccountRepository {
	return &bankAccountRepository{db}
}
func (r *bankAccountRepository) FindAll(ctx context.Context) ([]domain.BankAccount, error) {
	out := []domain.BankAccount{}
	err := dbFrom(ctx, r.db).Order("ledger_account").Find(&out).Error
	return out, err
}
func (r *bankAccountRepository) Save(ctx context.Context, account *domain.BankAccount) error {
	return dbFrom(ctx, r.db).Save(account).Error
}
