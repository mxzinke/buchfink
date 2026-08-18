package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type accountRepositoryGorm struct {
	db *gorm.DB
}

// NewAccountRepository creates a new GORM-backed AccountRepository.
func NewAccountRepository(db *gorm.DB) domain.AccountRepository {
	return &accountRepositoryGorm{db: db}
}

func (r *accountRepositoryGorm) FindAll(ctx context.Context) ([]domain.Account, error) {
	var accounts []domain.Account
	err := r.db.WithContext(ctx).Order("number asc").Find(&accounts).Error
	return accounts, err
}

func (r *accountRepositoryGorm) FindByNumber(ctx context.Context, number string) (*domain.Account, error) {
	var account domain.Account
	// Try exact match first
	err := r.db.WithContext(ctx).Where("number = ?", number).First(&account).Error
	if err == nil {
		return &account, nil
	}
	// If exact match not found and number looks like a 4-digit code, check range accounts
	if len(number) == 4 {
		rangeErr := r.db.WithContext(ctx).
			Where("is_range = ? AND range_start <= ? AND range_end >= ?", true, number, number).
			First(&account).Error
		if rangeErr == nil {
			return &account, nil
		}
	}
	return nil, err
}

func (r *accountRepositoryGorm) Create(ctx context.Context, account *domain.Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *accountRepositoryGorm) CreateBatch(ctx context.Context, accounts []domain.Account) error {
	return r.db.WithContext(ctx).Create(&accounts).Error
}

func (r *accountRepositoryGorm) Update(ctx context.Context, account *domain.Account) error {
	return r.db.WithContext(ctx).Save(account).Error
}

func (r *accountRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Account{}).Count(&count).Error
	return count, err
}
