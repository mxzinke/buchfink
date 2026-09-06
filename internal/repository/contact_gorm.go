package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type contactRepositoryGorm struct {
	db *gorm.DB
}

// NewContactRepository creates a new GORM-backed ContactRepository.
func NewContactRepository(db *gorm.DB) domain.ContactRepository {
	return &contactRepositoryGorm{db: db}
}

func (r *contactRepositoryGorm) FindAll(ctx context.Context) ([]domain.Contact, error) {
	var contacts []domain.Contact
	err := dbFrom(ctx, r.db).Order("name asc").Find(&contacts).Error
	return contacts, err
}

func (r *contactRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Contact, error) {
	var contact domain.Contact
	err := dbFrom(ctx, r.db).First(&contact, id).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *contactRepositoryGorm) FindByLedgerAccount(ctx context.Context, account string) (*domain.Contact, error) {
	var contact domain.Contact
	err := dbFrom(ctx, r.db).Where("ledger_account = ?", account).First(&contact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *contactRepositoryGorm) Save(ctx context.Context, contact *domain.Contact) error {
	return dbFrom(ctx, r.db).Save(contact).Error
}

func (r *contactRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.Contact{}, id).Error
}

func (r *contactRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := dbFrom(ctx, r.db).Model(&domain.Contact{}).Count(&count).Error
	return count, err
}
