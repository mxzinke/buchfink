package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type fiscalYearRepositoryGorm struct {
	db *gorm.DB
}

// NewFiscalYearRepository creates a GORM-backed FiscalYearRepository.
func NewFiscalYearRepository(db *gorm.DB) domain.FiscalYearRepository {
	return &fiscalYearRepositoryGorm{db: db}
}

func (r *fiscalYearRepositoryGorm) FindAll(ctx context.Context) ([]domain.FiscalYear, error) {
	var years []domain.FiscalYear
	err := r.db.WithContext(ctx).Order("year asc").Find(&years).Error
	return years, err
}

func (r *fiscalYearRepositoryGorm) FindByYear(ctx context.Context, year int) (*domain.FiscalYear, error) {
	var fy domain.FiscalYear
	err := r.db.WithContext(ctx).Where("year = ?", year).First(&fy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fy, nil
}

// Save legt das Geschäftsjahr an oder schreibt es fort.
//
// Als Upsert und nicht als Save von GORM: der Schlüssel ist die Jahreszahl und
// damit vom Aufrufer gesetzt, nicht vergeben. GORMs Save entscheidet anhand
// eines leeren Schlüssels zwischen Anlegen und Ändern und legte deshalb nie
// etwas an.
func (r *fiscalYearRepositoryGorm) Save(ctx context.Context, fy *domain.FiscalYear) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "year"}}, UpdateAll: true}).
		Create(fy).Error
}
