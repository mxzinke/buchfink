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

func (r *festschreibungRepositoryGorm) FindCutoffOnOrAfter(ctx context.Context, date string) ([]domain.Festschreibung, error) {
	var recs []domain.Festschreibung
	err := dbFrom(ctx, r.db).
		Where("cutoff_date >= ?", date).
		Order("cutoff_date desc, id desc").
		Find(&recs).Error
	return recs, err
}

func (r *festschreibungRepositoryGorm) ConfirmTimestamp(ctx context.Context, rec *domain.Festschreibung) (bool, error) {
	res := dbFrom(ctx, r.db).Model(&domain.Festschreibung{}).
		Where("id = ? AND timestamp_status = ?", rec.ID, "pending").
		Select("TimestampToken", "TSAName", "TSAGenTime", "TimestampStatus", "TimeDriftNote").
		Updates(&domain.Festschreibung{
			TimestampToken: rec.TimestampToken, TSAName: rec.TSAName, TSAGenTime: rec.TSAGenTime,
			TimestampStatus: "confirmed", TimeDriftNote: rec.TimeDriftNote,
		})
	return res.RowsAffected == 1, res.Error
}

func (r *festschreibungRepositoryGorm) FindPendingTimestamp(ctx context.Context) ([]domain.Festschreibung, error) {
	var recs []domain.Festschreibung
	err := dbFrom(ctx, r.db).
		Where("timestamp_status = ?", "pending").
		Find(&recs).Error
	return recs, err
}
