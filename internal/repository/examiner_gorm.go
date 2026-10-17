package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type examinerRepositoryGorm struct {
	db *gorm.DB
}

// NewExaminerRepository liefert die Zeiträume des Prüfermodus.
func NewExaminerRepository(db *gorm.DB) domain.ExaminerRepository {
	return &examinerRepositoryGorm{db: db}
}

func (r *examinerRepositoryGorm) Create(ctx context.Context, p *domain.ExaminerPeriod) error {
	return dbFrom(ctx, r.db).Create(p).Error
}

func (r *examinerRepositoryGorm) Save(ctx context.Context, p *domain.ExaminerPeriod) error {
	return dbFrom(ctx, r.db).Save(p).Error
}

func (r *examinerRepositoryGorm) Latest(ctx context.Context) (*domain.ExaminerPeriod, error) {
	var p domain.ExaminerPeriod
	err := dbFrom(ctx, r.db).Where("ended_on IS NULL").Order("id desc").Take(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
