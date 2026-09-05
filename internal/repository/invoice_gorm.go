package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type invoiceRepositoryGorm struct {
	db *gorm.DB
}

// NewInvoiceRepository creates a new GORM-backed InvoiceRepository.
func NewInvoiceRepository(db *gorm.DB) domain.InvoiceRepository {
	return &invoiceRepositoryGorm{db: db}
}

func (r *invoiceRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.Invoice, error) {
	var invoices []domain.Invoice
	q := dbFrom(ctx, r.db).Preload("Items").Preload("PrecedingRefs").Order("date desc, id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Find(&invoices).Error
	return invoices, err
}

func (r *invoiceRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := dbFrom(ctx, r.db).Preload("Items").Preload("PrecedingRefs").First(&invoice, id).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepositoryGorm) FindByNumber(ctx context.Context, number string) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := dbFrom(ctx, r.db).Preload("Items").Preload("PrecedingRefs").Where("invoice_number = ?", number).First(&invoice).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepositoryGorm) Save(ctx context.Context, invoice *domain.Invoice) error {
	return dbFrom(ctx, r.db).Save(invoice).Error
}

func (r *invoiceRepositoryGorm) UpdateStatus(ctx context.Context, id uint, status domain.InvoiceStatus) error {
	return dbFrom(ctx, r.db).Model(&domain.Invoice{}).Where("id = ?", id).Update("status", status).Error
}

func (r *invoiceRepositoryGorm) UpdateTransportKind(ctx context.Context, id uint, kind string) error {
	return dbFrom(ctx, r.db).Model(&domain.Invoice{}).Where("id = ?", id).
		Update("transport_kind", kind).Error
}

// FindNumbers liefert die vergebenen Rechnungsnummern eines Geschäftsjahres.
func (r *invoiceRepositoryGorm) FindNumbers(ctx context.Context, fiscalYear int) ([]string, error) {
	numbers := make([]string, 0)
	q := dbFrom(ctx, r.db).Model(&domain.Invoice{}).Order("invoice_number asc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Pluck("invoice_number", &numbers).Error
	return numbers, err
}

func (r *invoiceRepositoryGorm) FindByGroup(ctx context.Context, groupID uint) ([]domain.Invoice, error) {
	invoices := make([]domain.Invoice, 0)
	err := dbFrom(ctx, r.db).Preload("Items").Preload("PrecedingRefs").
		Where("group_id = ?", groupID).Order("date asc, id asc").Find(&invoices).Error
	return invoices, err
}

func (r *invoiceRepositoryGorm) Count(ctx context.Context, fiscalYear int) (int64, error) {
	var count int64
	q := dbFrom(ctx, r.db).Model(&domain.Invoice{})
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Count(&count).Error
	return count, err
}
