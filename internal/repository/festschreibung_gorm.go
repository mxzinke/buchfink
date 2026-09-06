package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type festschreibungRepositoryGorm struct {
	db *gorm.DB
}

// NewFestschreibungRepository creates a GORM-backed FestschreibungRepository.
func NewFestschreibungRepository(db *gorm.DB) domain.FestschreibungRepository {
	return &festschreibungRepositoryGorm{db: db}
}

func (r *festschreibungRepositoryGorm) Create(ctx context.Context, rec *domain.Festschreibung) error {
	return dbFrom(ctx, r.db).Create(rec).Error
}

func (r *festschreibungRepositoryGorm) Update(ctx context.Context, rec *domain.Festschreibung) error {
	return dbFrom(ctx, r.db).Save(rec).Error
}

func (r *festschreibungRepositoryGorm) FindByFiscalYear(ctx context.Context, fiscalYear int) ([]domain.Festschreibung, error) {
	var recs []domain.Festschreibung
	err := dbFrom(ctx, r.db).
		Where("fiscal_year = ?", fiscalYear).
		Order("cutoff_date desc").
		Find(&recs).Error
	return recs, err
}

func (r *festschreibungRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Festschreibung, error) {
	var rec domain.Festschreibung
	if err := dbFrom(ctx, r.db).First(&rec, id).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *festschreibungRepositoryGorm) LatestCutoff(ctx context.Context, fiscalYear int) (string, error) {
	var rec domain.Festschreibung
	err := dbFrom(ctx, r.db).
		Where("fiscal_year = ?", fiscalYear).
		Order("cutoff_date desc").
		First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return rec.CutoffDate, nil
}

func (r *festschreibungRepositoryGorm) FindPendingTimestamp(ctx context.Context) ([]domain.Festschreibung, error) {
	var recs []domain.Festschreibung
	err := dbFrom(ctx, r.db).
		Where("timestamp_status = ?", "pending").
		Find(&recs).Error
	return recs, err
}
