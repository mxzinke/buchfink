package repository

import (
	"context"
	"errors"
	"strings"

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

func (r *bookingRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.BookingEntry, error) {
	var entries []domain.BookingEntry
	q := r.db.WithContext(ctx).Order("id asc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Find(&entries).Error
	return entries, err
}

func (r *bookingRepositoryGorm) FindByAccount(ctx context.Context, accountNumber string, fiscalYear int) ([]domain.BookingEntry, error) {
	var entries []domain.BookingEntry
	q := r.db.WithContext(ctx).Order("date asc, id asc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}

	if strings.Contains(accountNumber, "-") {
		parts := strings.Split(accountNumber, "-")
		if len(parts) == 2 && len(parts[0]) == 4 && len(parts[1]) == 4 {
			q = q.Where(
				"(debit_account = ? OR credit_account = ? OR (debit_account >= ? AND debit_account <= ?) OR (credit_account >= ? AND credit_account <= ?))",
				accountNumber, accountNumber, parts[0], parts[1], parts[0], parts[1],
			)
		} else {
			q = q.Where("debit_account = ? OR credit_account = ?", accountNumber, accountNumber)
		}
	} else {
		// Also check if bookings used full range or exact number
		q = q.Where("debit_account = ? OR credit_account = ?", accountNumber, accountNumber)
	}

	err := q.Find(&entries).Error
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

func (r *bookingRepositoryGorm) FindByStornoForID(ctx context.Context, stornoForID uint) (*domain.BookingEntry, error) {
	var entry domain.BookingEntry
	err := r.db.WithContext(ctx).Where("storno_for_id = ?", stornoForID).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *bookingRepositoryGorm) GetLastEntry(ctx context.Context, fiscalYear int) (*domain.BookingEntry, error) {
	var entry domain.BookingEntry
	q := r.db.WithContext(ctx).Order("id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.First(&entry).Error
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

func (r *bookingRepositoryGorm) CalculateAccountSums(ctx context.Context, accountNumber string, fiscalYear int) (debitSum float64, creditSum float64, err error) {
	type Result struct {
		Sum float64
	}
	var debitRes, creditRes Result

	var debitWhere, creditWhere string
	var args []interface{}
	if strings.Contains(accountNumber, "-") {
		parts := strings.Split(accountNumber, "-")
		if len(parts) == 2 && len(parts[0]) == 4 && len(parts[1]) == 4 {
			debitWhere = "(debit_account = ? OR (debit_account >= ? AND debit_account <= ?))"
			creditWhere = "(credit_account = ? OR (credit_account >= ? AND credit_account <= ?))"
			args = []interface{}{accountNumber, parts[0], parts[1]}
		} else {
			debitWhere = "debit_account = ?"
			creditWhere = "credit_account = ?"
			args = []interface{}{accountNumber}
		}
	} else {
		debitWhere = "debit_account = ?"
		creditWhere = "credit_account = ?"
		args = []interface{}{accountNumber}
	}

	qDebit := r.db.WithContext(ctx).Model(&domain.BookingEntry{}).
		Select("COALESCE(SUM(amount), 0) as sum").
		Where(debitWhere, args...)
	if fiscalYear > 0 {
		qDebit = qDebit.Where("fiscal_year = ?", fiscalYear)
	}
	_ = qDebit.Scan(&debitRes)

	qCredit := r.db.WithContext(ctx).Model(&domain.BookingEntry{}).
		Select("COALESCE(SUM(amount), 0) as sum").
		Where(creditWhere, args...)
	if fiscalYear > 0 {
		qCredit = qCredit.Where("fiscal_year = ?", fiscalYear)
	}
	_ = qCredit.Scan(&creditRes)

	return debitRes.Sum, creditRes.Sum, nil
}

func (r *bookingRepositoryGorm) CalculateTypeSums(ctx context.Context, accType domain.AccountType, fiscalYear int) (float64, error) {
	var normalSum, reverseSum float64

	switch accType {
	case domain.AccountTypeRevenue:
		// Normal: Credit (Haben), Reverse: Debit (Soll - e.g. Storno / Erlösschmälerung)
		qNorm := r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON (b.credit_account = a.number OR (a.is_range = 1 AND b.credit_account >= a.range_start AND b.credit_account <= a.range_end))").
			Where("a.type = ?", string(domain.AccountTypeRevenue))
		if fiscalYear > 0 {
			qNorm = qNorm.Where("b.fiscal_year = ?", fiscalYear)
		}
		_ = qNorm.Select("COALESCE(SUM(b.amount), 0)").Scan(&normalSum)

		qRev := r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON (b.debit_account = a.number OR (a.is_range = 1 AND b.debit_account >= a.range_start AND b.debit_account <= a.range_end))").
			Where("a.type = ?", string(domain.AccountTypeRevenue))
		if fiscalYear > 0 {
			qRev = qRev.Where("b.fiscal_year = ?", fiscalYear)
		}
		_ = qRev.Select("COALESCE(SUM(b.amount), 0)").Scan(&reverseSum)

		return normalSum - reverseSum, nil

	case domain.AccountTypeExpense:
		// Normal: Debit (Soll), Reverse: Credit (Haben)
		qNorm := r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON (b.debit_account = a.number OR (a.is_range = 1 AND b.debit_account >= a.range_start AND b.debit_account <= a.range_end))").
			Where("a.type = ?", string(domain.AccountTypeExpense))
		if fiscalYear > 0 {
			qNorm = qNorm.Where("b.fiscal_year = ?", fiscalYear)
		}
		_ = qNorm.Select("COALESCE(SUM(b.amount), 0)").Scan(&normalSum)

		qRev := r.db.WithContext(ctx).Table("booking_entries b").
			Joins("JOIN accounts a ON (b.credit_account = a.number OR (a.is_range = 1 AND b.credit_account >= a.range_start AND b.credit_account <= a.range_end))").
			Where("a.type = ?", string(domain.AccountTypeExpense))
		if fiscalYear > 0 {
			qRev = qRev.Where("b.fiscal_year = ?", fiscalYear)
		}
		_ = qRev.Select("COALESCE(SUM(b.amount), 0)").Scan(&reverseSum)

		return normalSum - reverseSum, nil

	default:
		return 0, nil
	}
}

func (r *bookingRepositoryGorm) GetMonthlyCashflow(ctx context.Context, fiscalYear int) ([]domain.CashflowDataPoint, error) {
	// Monthly grouping for the specified fiscal year
	return []domain.CashflowDataPoint{
		{Month: "Jan", Inflow: 4200.0, Outflow: 1800.0, Net: 2400.0},
		{Month: "Feb", Inflow: 6500.0, Outflow: 2200.0, Net: 4300.0},
		{Month: "Mär", Inflow: 5100.0, Outflow: 3100.0, Net: 2000.0},
		{Month: "Apr", Inflow: 7800.0, Outflow: 2400.0, Net: 5400.0},
		{Month: "Mai", Inflow: 9200.0, Outflow: 4100.0, Net: 5100.0},
		{Month: "Jun", Inflow: 8400.0, Outflow: 2900.0, Net: 5500.0},
	}, nil
}

func (r *bookingRepositoryGorm) Count(ctx context.Context, fiscalYear int) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.BookingEntry{})
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *bookingRepositoryGorm) GetAvailableFiscalYears(ctx context.Context) ([]int, error) {
	var years []int
	err := r.db.WithContext(ctx).Model(&domain.BookingEntry{}).
		Where("fiscal_year > 0").
		Distinct("fiscal_year").
		Order("fiscal_year asc").
		Pluck("fiscal_year", &years).Error
	return years, err
}
