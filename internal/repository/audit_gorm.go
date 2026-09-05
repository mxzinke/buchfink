package repository

import (
	"context"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type auditRepositoryGorm struct {
	db *gorm.DB
}

// NewAuditRepository creates a new GORM-backed AuditRepository.
func NewAuditRepository(db *gorm.DB) domain.AuditRepository {
	return &auditRepositoryGorm{db: db}
}

func (r *auditRepositoryGorm) Log(ctx context.Context, action domain.AuditAction, entityType, entityID, details string) error {
	entry := domain.AuditLogEntry{
		Timestamp:  time.Now(),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
	}
	return dbFrom(ctx, r.db).Create(&entry).Error
}

func (r *auditRepositoryGorm) FindAll(ctx context.Context, limit int) ([]domain.AuditLogEntry, error) {
	var entries []domain.AuditLogEntry
	query := dbFrom(ctx, r.db).Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&entries).Error
	return entries, err
}

func (r *auditRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := dbFrom(ctx, r.db).Model(&domain.AuditLogEntry{}).Count(&count).Error
	return count, err
}
