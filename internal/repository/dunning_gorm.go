package repository

import (
	"context"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type dunningRepositoryGorm struct {
	db *gorm.DB
}

// NewDunningRepository speichert die Mahnvermerke.
func NewDunningRepository(db *gorm.DB) domain.DunningRepository {
	return &dunningRepositoryGorm{db: db}
}

func (r *dunningRepositoryGorm) Create(ctx context.Context, note *domain.DunningNote) error {
	note.CreatedAt = time.Now().UTC()
	return dbFrom(ctx, r.db).Create(note).Error
}

func (r *dunningRepositoryGorm) FindAll(ctx context.Context) ([]domain.DunningNote, error) {
	var notes []domain.DunningNote
	if err := dbFrom(ctx, r.db).Order("date, id").Find(&notes).Error; err != nil {
		return nil, err
	}
	return notes, nil
}
