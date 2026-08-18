package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type bookingRepositoryGorm struct {
	db *gorm.DB
}

// NewBookingRepository creates a new GORM-backed BookingRepository.
func NewBookingRepository(db *gorm.DB) domain.BookingRepository {
	return &bookingRepositoryGorm{db: db}
}

func (r *bookingRepositoryGorm) FindAll(ctx context.Context) ([]domain.BookingEntry, error) {
	var entries []domain.BookingEntry
	err := r.db.WithContext(ctx).Order("id asc").Find(&entries).Error
	return entries, err
}

func (r *bookingRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.BookingEntry, error) {
	var entry domain.BookingEntry
	err := r.db.WithContext(ctx).First(&entry, id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *bookingRepositoryGorm) GetLastEntry(ctx context.Context) (*domain.BookingEntry, error) {
	var entry domain.BookingEntry
	err := r.db.WithContext(ctx).Order("id desc").First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *bookingRepositoryGorm) Create(ctx context.Context, entry *domain.BookingEntry) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *bookingRepositoryGorm) CalculateAccountSums(ctx context.Context, accountNumber string) (debitSum float64, creditSum float64, err error) {
	type Result struct {
		Sum float64
	}
	var debitRes, creditRes Result

	_ = r.db.WithContext(ctx).Model(&domain.BookingEntry{}).
		Select("COALESCE(SUM(amount), 0) as sum").
		Where("debit_account = ?", accountNumber).
		Scan(&debitRes)

	_ = r.db.WithContext(ctx).Model(&domain.BookingEntry{}).
		Select("COALESCE(SUM(amount), 0) as sum").
		Where("credit_account = ?", accountNumber).
		Scan(&creditRes)

	return debitRes.Sum, creditRes.Sum, nil
}

func (r *bookingRepositoryGorm) CalculateTypeSums(ctx context.Context, accType domain.AccountType) (float64, error) {
	var normalSum, reverseSum float64

	switch accType {
	case domain.AccountTypeRevenue:
		// Normal: Credit (Haben), Reverse: Debit (Soll - e.g. Storno / Erlösschmälerung)
		_ = r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON b.credit_account = a.number").
			Where("a.type = ?", string(domain.AccountTypeRevenue)).
			Select("COALESCE(SUM(b.amount), 0)").
			Scan(&normalSum)

		_ = r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON b.debit_account = a.number").
			Where("a.type = ?", string(domain.AccountTypeRevenue)).
			Select("COALESCE(SUM(b.amount), 0)").
			Scan(&reverseSum)

		return normalSum - reverseSum, nil

	case domain.AccountTypeExpense:
		// Normal: Debit (Soll), Reverse: Credit (Haben)
		_ = r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON b.debit_account = a.number").
			Where("a.type = ?", string(domain.AccountTypeExpense)).
			Select("COALESCE(SUM(b.amount), 0)").
			Scan(&normalSum)

		_ = r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON b.credit_account = a.number").
			Where("a.type = ?", string(domain.AccountTypeExpense)).
			Select("COALESCE(SUM(b.amount), 0)").
			Scan(&reverseSum)

		return normalSum - reverseSum, nil

	default:
		return 0, nil
	}
}

func (r *bookingRepositoryGorm) GetMonthlyCashflow(ctx context.Context) ([]domain.CashflowDataPoint, error) {
	// Simple monthly grouping or default scaffolding series
	// TODO: Replace with dynamic SQL GROUP BY strftime('%Y-%m', date) over bank account 1800
	return []domain.CashflowDataPoint{
		{Month: "Jan", Inflow: 4200.0, Outflow: 1800.0, Net: 2400.0},
		{Month: "Feb", Inflow: 6500.0, Outflow: 2200.0, Net: 4300.0},
		{Month: "Mär", Inflow: 5100.0, Outflow: 3100.0, Net: 2000.0},
		{Month: "Apr", Inflow: 7800.0, Outflow: 2400.0, Net: 5400.0},
		{Month: "Mai", Inflow: 9200.0, Outflow: 4100.0, Net: 5100.0},
		{Month: "Jun", Inflow: 8400.0, Outflow: 2900.0, Net: 5500.0},
	}, nil
}

func (r *bookingRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.BookingEntry{}).Count(&count).Error
	return count, err
}
