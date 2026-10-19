package repository

import (
	"context"
	"errors"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type idempotencyRepositoryGorm struct {
	db *gorm.DB
}

// NewIdempotencyRepository liefert die Ablage der Idempotenzschlüssel.
func NewIdempotencyRepository(db *gorm.DB) domain.IdempotencyRepository {
	return &idempotencyRepositoryGorm{db: db}
}

func (r *idempotencyRepositoryGorm) Find(ctx context.Context, clientID, key string) (*domain.IdempotencyRecord, error) {
	var rec domain.IdempotencyRecord
	err := dbFrom(ctx, r.db).Where("client_id = ? AND key = ?", clientID, key).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Create legt die Zeitpunkte in UTC ab. SQLite hält sie als Text mit
// Zonenversatz, und DeleteExpired vergleicht diesen Text: nur in einer Zone
// folgt die Textordnung der Zeit.
func (r *idempotencyRepositoryGorm) Create(ctx context.Context, record *domain.IdempotencyRecord) error {
	record.CreatedAt = record.CreatedAt.UTC()
	record.ExpiresAt = record.ExpiresAt.UTC()
	return dbFrom(ctx, r.db).Create(record).Error
}

func (r *idempotencyRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.IdempotencyRecord{}, id).Error
}

func (r *idempotencyRepositoryGorm) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := dbFrom(ctx, r.db).Where("expires_at <= ?", now.UTC()).Delete(&domain.IdempotencyRecord{})
	return res.RowsAffected, res.Error
}
