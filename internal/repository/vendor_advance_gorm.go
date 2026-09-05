package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type vendorAdvanceRepositoryGorm struct {
	db *gorm.DB
}

// NewVendorAdvanceRepository creates a GORM-backed VendorAdvanceRepository.
func NewVendorAdvanceRepository(db *gorm.DB) domain.VendorAdvanceRepository {
	return &vendorAdvanceRepositoryGorm{db: db}
}

func (r *vendorAdvanceRepositoryGorm) Save(ctx context.Context, advance *domain.VendorAdvance) error {
	return dbFrom(ctx, r.db).Save(advance).Error
}

func (r *vendorAdvanceRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.VendorAdvance, error) {
	var advance domain.VendorAdvance
	if err := dbFrom(ctx, r.db).First(&advance, id).Error; err != nil {
		return nil, err
	}
	return &advance, nil
}

// FindOpen liefert die noch nicht verrechneten Anzahlungen. Leer statt nil: die
// Liste geht als JSON an die Oberfläche.
func (r *vendorAdvanceRepositoryGorm) FindOpen(ctx context.Context, contactID uint) ([]domain.VendorAdvance, error) {
	advances := make([]domain.VendorAdvance, 0)
	q := dbFrom(ctx, r.db).Where("settled_by_entry_id IS NULL")
	if contactID != 0 {
		q = q.Where("contact_id = ?", contactID)
	}
	err := q.Order("paid_at asc, id asc").Find(&advances).Error
	return advances, err
}
