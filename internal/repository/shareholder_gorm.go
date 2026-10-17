package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type shareholderLoanRepositoryGorm struct {
	db *gorm.DB
}

// NewShareholderLoanRepository liefert die Ablage der Gesellschafterdarlehen.
func NewShareholderLoanRepository(db *gorm.DB) domain.ShareholderLoanRepository {
	return &shareholderLoanRepositoryGorm{db: db}
}

func (r *shareholderLoanRepositoryGorm) Save(ctx context.Context, loan *domain.ShareholderLoan) error {
	return dbFrom(ctx, r.db).Save(loan).Error
}

func (r *shareholderLoanRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.ShareholderLoan, error) {
	var loan domain.ShareholderLoan
	if err := dbFrom(ctx, r.db).First(&loan, id).Error; err != nil {
		return nil, err
	}
	return &loan, nil
}

func (r *shareholderLoanRepositoryGorm) FindAll(ctx context.Context) ([]domain.ShareholderLoan, error) {
	loans := make([]domain.ShareholderLoan, 0)
	err := dbFrom(ctx, r.db).Order("id asc").Find(&loans).Error
	return loans, err
}

func (r *shareholderLoanRepositoryGorm) SaveInterest(ctx context.Context, interest *domain.ShareholderLoanInterest) error {
	return dbFrom(ctx, r.db).Save(interest).Error
}

func (r *shareholderLoanRepositoryGorm) FindInterest(ctx context.Context, loanID uint) ([]domain.ShareholderLoanInterest, error) {
	out := make([]domain.ShareholderLoanInterest, 0)
	err := dbFrom(ctx, r.db).Where("loan_id = ?", loanID).Order("period_from asc, id asc").Find(&out).Error
	return out, err
}

type payrollRepositoryGorm struct {
	db *gorm.DB
}

// NewPayrollRepository liefert die Ablage der gebuchten Lohnjournale.
func NewPayrollRepository(db *gorm.DB) domain.PayrollRepository {
	return &payrollRepositoryGorm{db: db}
}

func (r *payrollRepositoryGorm) Save(ctx context.Context, run *domain.PayrollRun) error {
	return dbFrom(ctx, r.db).Save(run).Error
}

func (r *payrollRepositoryGorm) FindByMonth(ctx context.Context, month string) ([]domain.PayrollRun, error) {
	out := make([]domain.PayrollRun, 0)
	err := dbFrom(ctx, r.db).Where("month = ?", month).Order("id asc").Find(&out).Error
	return out, err
}

func (r *payrollRepositoryGorm) FindAll(ctx context.Context) ([]domain.PayrollRun, error) {
	out := make([]domain.PayrollRun, 0)
	err := dbFrom(ctx, r.db).Order("month asc, id asc").Find(&out).Error
	return out, err
}
