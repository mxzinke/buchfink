package repository

import (
	"context"
	"errors"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type obligationRepositoryGorm struct {
	db *gorm.DB
}

// NewObligationRepository speichert Bestätigungen und Anlässe des
// Pflichtenkalenders.
func NewObligationRepository(db *gorm.DB) domain.ObligationRepository {
	return &obligationRepositoryGorm{db: db}
}

func (r *obligationRepositoryGorm) FindRecords(ctx context.Context) ([]domain.ObligationRecord, error) {
	var recs []domain.ObligationRecord
	if err := dbFrom(ctx, r.db).Order("key").Find(&recs).Error; err != nil {
		return nil, err
	}
	return recs, nil
}

func (r *obligationRepositoryGorm) FindRecord(ctx context.Context, key string) (*domain.ObligationRecord, error) {
	var rec domain.ObligationRecord
	err := dbFrom(ctx, r.db).Where("key = ?", key).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *obligationRepositoryGorm) SaveRecord(ctx context.Context, rec *domain.ObligationRecord) error {
	rec.UpdatedAt = time.Now().UTC()
	return dbFrom(ctx, r.db).Save(rec).Error
}

func (r *obligationRepositoryGorm) DeleteRecord(ctx context.Context, key string) error {
	return dbFrom(ctx, r.db).Where("key = ?", key).Delete(&domain.ObligationRecord{}).Error
}

func (r *obligationRepositoryGorm) FindEvents(ctx context.Context) ([]domain.ObligationEvent, error) {
	var events []domain.ObligationEvent
	if err := dbFrom(ctx, r.db).Order("date, id").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *obligationRepositoryGorm) CreateEvent(ctx context.Context, ev *domain.ObligationEvent) error {
	ev.CreatedAt = time.Now().UTC()
	return dbFrom(ctx, r.db).Create(ev).Error
}
