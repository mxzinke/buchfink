package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type bankRepositoryGorm struct {
	db *gorm.DB
}

// NewBankRepository creates a new GORM-backed BankRepository.
func NewBankRepository(db *gorm.DB) domain.BankRepository {
	return &bankRepositoryGorm{db: db}
}

func (r *bankRepositoryGorm) FindAll(ctx context.Context) ([]domain.BankTransaction, error) {
	var txs []domain.BankTransaction
	err := r.db.WithContext(ctx).Order("booking_date desc, id desc").Find(&txs).Error
	return txs, err
}

func (r *bankRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.BankTransaction, error) {
	var tx domain.BankTransaction
	err := r.db.WithContext(ctx).First(&tx, id).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *bankRepositoryGorm) CreateBatch(ctx context.Context, transactions []domain.BankTransaction) (int, error) {
	inserted := 0
	for _, tx := range transactions {
		// Duplicate prevention: check end_to_end_id or IBAN+date+amount
		var count int64
		if tx.EndToEndID != "" {
			r.db.WithContext(ctx).Model(&domain.BankTransaction{}).
				Where("end_to_end_id = ?", tx.EndToEndID).
				Count(&count)
		} else {
			r.db.WithContext(ctx).Model(&domain.BankTransaction{}).
				Where("booking_date = ? AND amount = ? AND counterparty_iban = ?", tx.BookingDate, tx.Amount, tx.CounterpartyIBAN).
				Count(&count)
		}

		if count > 0 {
			continue
		}

		if err := r.db.WithContext(ctx).Create(&tx).Error; err == nil {
			inserted++
		}
	}
	return inserted, nil
}

func (r *bankRepositoryGorm) MarkMatched(ctx context.Context, id uint, bookingID uint) error {
	return r.db.WithContext(ctx).Model(&domain.BankTransaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"match_status":       domain.MatchStatusMatched,
			"matched_booking_id": bookingID,
		}).Error
}

func (r *bankRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.BankTransaction{}).Count(&count).Error
	return count, err
}
