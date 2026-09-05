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

func (r *paymentAllocationRepositoryGorm) FindAll(ctx context.Context) ([]domain.PaymentAllocation, error) {
	out := make([]domain.PaymentAllocation, 0)
	err := r.db.WithContext(ctx).Order("id asc").Find(&out).Error
	return out, err
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
// Neither bound the query lacks — no fiscal year, no reversed payments — is an
// oversight; domain.PaymentAllocationRepository states why.
func (r *paymentAllocationRepositoryGorm) SettledByOpenItem(ctx context.Context) (map[uint]domain.Cents, error) {
	type row struct {
		OpenItemEntryID uint
		Total           int64
	}

	var rows []row
	q := r.db.WithContext(ctx).Model(&domain.PaymentAllocation{}).
		Select("payment_allocations.open_item_entry_id as open_item_entry_id, COALESCE(SUM(payment_allocations.settled_amount),0) as total").
		Joins("JOIN journal_entries e ON e.id = payment_allocations.payment_entry_id")
	q = notReversed(q, "e")
	if err := q.Group("payment_allocations.open_item_entry_id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint]domain.Cents, len(rows))
	for _, x := range rows {
		result[x.OpenItemEntryID] = domain.Cents(x.Total)
	}
	return result, nil
}

// SettledByOpenItemAt zählt nur, was bis zum Stichtag ausgeglichen war.
//
// Die Generalumkehr wird ebenfalls auf ihr Datum begrenzt: eine Zahlung, die im
// Dezember gebucht und im März storniert wurde, hat den Posten am 31.12.
// ausgeglichen. Erst der Storno holt ihn zurück — im neuen Jahr, wo er gebucht
// ist, und nicht rückwirkend in der Eröffnungsbilanz.
func (r *paymentAllocationRepositoryGorm) SettledByOpenItemAt(ctx context.Context, cutoff string) (map[uint]domain.Cents, error) {
	type row struct {
		OpenItemEntryID uint
		Total           int64
	}

	var rows []row
	q := r.db.WithContext(ctx).Model(&domain.PaymentAllocation{}).
		Select("payment_allocations.open_item_entry_id as open_item_entry_id, COALESCE(SUM(payment_allocations.settled_amount),0) as total").
		Joins("JOIN journal_entries e ON e.id = payment_allocations.payment_entry_id").
		Where("e.kind <> ?", domain.EntryKindReversal).
		Where("e.booking_date <= ?", cutoff).
		Where("NOT EXISTS (SELECT 1 FROM journal_entries gu WHERE gu.reversal_of_id = e.id AND gu.booking_date <= ?)", cutoff)
	if err := q.Group("payment_allocations.open_item_entry_id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint]domain.Cents, len(rows))
	for _, x := range rows {
		result[x.OpenItemEntryID] = domain.Cents(x.Total)
	}
	return result, nil
}

// FindByPayment liefert die Posten, die eine Zahlung ausgeglichen hat.
func (r *paymentAllocationRepositoryGorm) FindByPayment(ctx context.Context, paymentEntryID uint) ([]domain.PaymentAllocation, error) {
	out := make([]domain.PaymentAllocation, 0)
	err := r.db.WithContext(ctx).Where("payment_entry_id = ?", paymentEntryID).Order("id asc").Find(&out).Error
	return out, err
}

func (r *paymentAllocationRepositoryGorm) FindByBankTx(ctx context.Context, bankTxID uint) ([]domain.PaymentAllocation, error) {
	var out []domain.PaymentAllocation
	err := r.db.WithContext(ctx).Where("bank_tx_id = ?", bankTxID).Order("id asc").Find(&out).Error
	return out, err
}
