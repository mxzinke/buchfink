// SPDX-License-Identifier: EUPL-1.2

package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type paymentAllocationRepositoryGorm struct {
	db *gorm.DB
}

// NewPaymentAllocationRepository creates a GORM-backed PaymentAllocationRepository.
func NewPaymentAllocationRepository(db *gorm.DB) domain.PaymentAllocationRepository {
	return &paymentAllocationRepositoryGorm{db: db}
}

func (r *paymentAllocationRepositoryGorm) Create(ctx context.Context, allocations []domain.PaymentAllocation) error {
	if len(allocations) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&allocations).Error
}

func (r *paymentAllocationRepositoryGorm) FindByOpenItem(ctx context.Context, entryID uint) ([]domain.PaymentAllocation, error) {
	var out []domain.PaymentAllocation
	err := r.db.WithContext(ctx).Where("open_item_entry_id = ?", entryID).Order("id asc").Find(&out).Error
	return out, err
}

// SettledByOpenItem sums what has already been settled per open item, so the
// remaining amount can be derived instead of stored. A stored status would drift
// apart from the journal on the first correction.
func (r *paymentAllocationRepositoryGorm) SettledByOpenItem(ctx context.Context, fiscalYear int) (map[uint]domain.Cents, error) {
	type row struct {
		OpenItemEntryID uint
		Total           int64
	}

	var rows []row
	q := r.db.WithContext(ctx).Model(&domain.PaymentAllocation{}).
		Select("payment_allocations.open_item_entry_id as open_item_entry_id, COALESCE(SUM(payment_allocations.settled_amount),0) as total").
		Joins("JOIN journal_entries e ON e.id = payment_allocations.payment_entry_id")
	if fiscalYear > 0 {
		q = q.Where("e.fiscal_year = ?", fiscalYear)
	}
	if err := q.Group("payment_allocations.open_item_entry_id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint]domain.Cents, len(rows))
	for _, x := range rows {
		result[x.OpenItemEntryID] = domain.Cents(x.Total)
	}
	return result, nil
}

func (r *paymentAllocationRepositoryGorm) FindByBankTx(ctx context.Context, bankTxID uint) ([]domain.PaymentAllocation, error) {
	var out []domain.PaymentAllocation
	err := r.db.WithContext(ctx).Where("bank_tx_id = ?", bankTxID).Order("id asc").Find(&out).Error
	return out, err
}
