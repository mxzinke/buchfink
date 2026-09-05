package repository

import (
	"context"
	"errors"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type checkRunRepositoryGorm struct {
	db *gorm.DB
}

// NewCheckRunRepository creates a GORM-backed CheckRunRepository.
func NewCheckRunRepository(db *gorm.DB) domain.CheckRunRepository {
	return &checkRunRepositoryGorm{db: db}
}

// Create speichert den Lauf mit seinen Befunden.
//
// Es gibt kein Update: ein Prüflauf ist die Aussage über einen Zeitpunkt. Wird
// erneut geprüft, entsteht ein neuer Lauf — nur die Begründung eines übergangenen
// Befundes wird am selben Lauf nachgetragen, weil sie zu ihm gehört.
func (r *checkRunRepositoryGorm) Create(ctx context.Context, run *domain.CheckRun) error {
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now()
	}
	return dbFrom(ctx, r.db).Create(run).Error
}

func (r *checkRunRepositoryGorm) FindByFiscalYear(ctx context.Context, fiscalYear int) ([]domain.CheckRun, error) {
	var runs []domain.CheckRun
	err := dbFrom(ctx, r.db).Preload("Findings").
		Where("fiscal_year = ?", fiscalYear).
		Order("created_at desc, id desc").
		Find(&runs).Error
	for i := range runs {
		runs[i].EnsureLists()
	}
	return runs, err
}

func (r *checkRunRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.CheckRun, error) {
	var run domain.CheckRun
	if err := dbFrom(ctx, r.db).Preload("Findings").First(&run, id).Error; err != nil {
		return nil, err
	}
	run.EnsureLists()
	return &run, nil
}

func (r *checkRunRepositoryGorm) Latest(ctx context.Context, fiscalYear int) (*domain.CheckRun, error) {
	var run domain.CheckRun
	err := dbFrom(ctx, r.db).Preload("Findings").
		Where("fiscal_year = ?", fiscalYear).
		Order("created_at desc, id desc").
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	run.EnsureLists()
	return &run, nil
}
