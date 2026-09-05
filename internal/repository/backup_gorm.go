package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type backupRunRepositoryGorm struct {
	db *gorm.DB
}

// NewBackupRunRepository creates a GORM-backed BackupRunRepository.
func NewBackupRunRepository(db *gorm.DB) domain.BackupRunRepository {
	return &backupRunRepositoryGorm{db: db}
}

func (r *backupRunRepositoryGorm) Create(ctx context.Context, run *domain.BackupRun) error {
	return dbFrom(ctx, r.db).Create(run).Error
}

func (r *backupRunRepositoryGorm) FindRecent(ctx context.Context, limit int) ([]domain.BackupRun, error) {
	runs := make([]domain.BackupRun, 0)
	q := dbFrom(ctx, r.db).Order("id desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&runs).Error
	return runs, err
}

// LatestSuccessful sucht die jüngste gelungene Sicherung — nicht den jüngsten
// Lauf. Ein fehlgeschlagener Lauf von gestern ist keine Sicherung von gestern,
// und die Aufgabenliste dürfte ihn nicht als eine ausweisen.
func (r *backupRunRepositoryGorm) LatestSuccessful(ctx context.Context) (*domain.BackupRun, error) {
	var run domain.BackupRun
	err := dbFrom(ctx, r.db).
		Where("success = ?", true).
		Where("kind IN ?", []domain.BackupKind{domain.BackupKindManual, domain.BackupKindAutomatic}).
		Order("id desc").First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}
