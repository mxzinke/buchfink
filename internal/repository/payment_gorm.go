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
//
// Two things it deliberately does not do.
//
// It does not filter by fiscal year. An offener Posten does not close at the
// Jahreswechsel: § 252 Abs. 1 Nr. 5 HGB puts the Ertrag in the year of
// performance and the payment in the year it happens, so a December invoice
// settled in January has its two halves in different years — by design, not by
// accident. Counting only one year would show a paid invoice as open, and the
// allocation dialog would happily let somebody pay it a second time. A
// Stichtag view — what was open on 31.12. — is a different question with a
// different answer, and it needs a date bound rather than this sum.
//
// It does not count allocations whose payment booking was reversed. A
// Generalumkehr of a payment leaves its allocation rows behind; without this
// they would keep the open item closed, and the money would be owed by nobody.
func (r *paymentAllocationRepositoryGorm) SettledByOpenItem(ctx context.Context) (map[uint]domain.Cents, error) {
	type row struct {
		OpenItemEntryID uint
		Total           int64
	}

	var rows []row
	q := r.db.WithContext(ctx).Model(&domain.PaymentAllocation{}).
		Select("payment_allocations.open_item_entry_id as open_item_entry_id, COALESCE(SUM(payment_allocations.settled_amount),0) as total").
		Joins("JOIN journal_entries e ON e.id = payment_allocations.payment_entry_id").
		Where("e.kind <> ?", domain.EntryKindReversal).
		Where("NOT EXISTS (SELECT 1 FROM journal_entries r WHERE r.reversal_of_id = e.id)")
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
