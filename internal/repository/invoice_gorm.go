// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

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
	q := r.db.WithContext(ctx).Preload("Items").Order("date desc, id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Find(&invoices).Error
	return invoices, err
}

func (r *invoiceRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := r.db.WithContext(ctx).Preload("Items").First(&invoice, id).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepositoryGorm) FindByNumber(ctx context.Context, number string) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := r.db.WithContext(ctx).Preload("Items").Where("invoice_number = ?", number).First(&invoice).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *invoiceRepositoryGorm) Save(ctx context.Context, invoice *domain.Invoice) error {
	return r.db.WithContext(ctx).Save(invoice).Error
}

func (r *invoiceRepositoryGorm) UpdateStatus(ctx context.Context, id uint, status domain.InvoiceStatus) error {
	return r.db.WithContext(ctx).Model(&domain.Invoice{}).Where("id = ?", id).Update("status", status).Error
}

func (r *invoiceRepositoryGorm) Count(ctx context.Context, fiscalYear int) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.Invoice{})
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	err := q.Count(&count).Error
	return count, err
}
