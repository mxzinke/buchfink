package repository

import (
	"context"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------
// Bestätigungsabfragen der USt-IdNr.
// -------------------------------------------------------------------------

type vatIDCheckRepositoryGorm struct {
	db *gorm.DB
}

// NewVatIDCheckRepository creates a GORM-backed VatIDCheckRepository.
func NewVatIDCheckRepository(db *gorm.DB) domain.VatIDCheckRepository {
	return &vatIDCheckRepositoryGorm{db: db}
}

func (r *vatIDCheckRepositoryGorm) FindByContact(ctx context.Context, contactID uint) ([]domain.VatIDCheck, error) {
	out := make([]domain.VatIDCheck, 0)
	err := dbFrom(ctx, r.db).
		Where("contact_id = ?", contactID).
		Order("checked_at desc, id desc").
		Find(&out).Error
	return out, err
}

// FindLatestValid sucht die jüngste gültige Bestätigung zu einer Nummer.
func (r *vatIDCheckRepositoryGorm) FindLatestValid(
	ctx context.Context, contactID uint, vatID string,
) (*domain.VatIDCheck, error) {
	var checks []domain.VatIDCheck
	err := dbFrom(ctx, r.db).
		Where("contact_id = ? AND status = ? AND vat_id = ?", contactID, domain.VatIDValid, vatID).
		Order("checked_at desc, id desc").
		Limit(1).
		Find(&checks).Error
	if err != nil || len(checks) == 0 {
		return nil, err
	}
	return &checks[0], nil
}

func (r *vatIDCheckRepositoryGorm) Save(ctx context.Context, check *domain.VatIDCheck) error {
	at, err := time.Parse(time.RFC3339, check.CheckedAt)
	if err != nil {
		return err
	}
	check.CheckedAt = at.UTC().Format(time.RFC3339)
	return dbFrom(ctx, r.db).Save(check).Error
}
