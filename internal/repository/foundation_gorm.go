package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type foundationRepositoryGorm struct {
	db *gorm.DB
}

// NewFoundationRepository creates a GORM-backed FoundationRepository.
func NewFoundationRepository(db *gorm.DB) domain.FoundationRepository {
	return &foundationRepositoryGorm{db: db}
}

// Get returns the Gründung of this tenant, or nil if there is none.
//
// Je Mandant gibt es höchstens eine — ein Mandant ist ein Unternehmen, und ein
// Unternehmen wird einmal gegründet. Statt das über eine Bedingung beim
// Schreiben zu erzwingen, liest Get schlicht die erste Zeile: eine zweite könnte
// nur durch einen Fehler entstehen, und dann ist die erste die richtige.
func (r *foundationRepositoryGorm) Get(ctx context.Context) (*domain.Foundation, error) {
	var f domain.Foundation
	err := dbFrom(ctx, r.db).
		Preload("Shareholders", func(db *gorm.DB) *gorm.DB {
			return db.Order("id asc")
		}).
		Order("id asc").
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.EnsureLists()
	return &f, nil
}

// Save writes the Gründung and replaces its shareholder list.
//
// Ersetzt, nicht abgeglichen: Die Liste ist kurz, sie wird als Ganzes bearbeitet,
// und ein Abgleich Zeile für Zeile brächte nur die Frage mit, was mit einer
// entfernten Zeile geschieht. Beides zusammen in einer Transaktion, damit nicht
// eine gelöschte Liste ohne ihren Ersatz zurückbleibt.
func (r *foundationRepositoryGorm) Save(ctx context.Context, f *domain.Foundation) error {
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		shareholders := f.Shareholders
		f.Shareholders = nil

		if err := tx.Omit("Shareholders").Save(f).Error; err != nil {
			return err
		}

		if err := tx.Where("foundation_id = ?", f.ID).Delete(&domain.Shareholder{}).Error; err != nil {
			return err
		}

		for i := range shareholders {
			// Die Kennung stammt aus der alten Liste und würde die gerade
			// gelöschte Zeile wieder auferstehen lassen wollen.
			shareholders[i].ID = 0
			shareholders[i].FoundationID = f.ID
			if shareholders[i].Kind == "" {
				shareholders[i].Kind = domain.ContributionCash
			}
			if err := tx.Create(&shareholders[i]).Error; err != nil {
				return err
			}
		}

		f.Shareholders = shareholders
		return nil
	})
}

func (r *foundationRepositoryGorm) Tasks(ctx context.Context, foundationID uint) ([]domain.FoundationTask, error) {
	var tasks []domain.FoundationTask
	err := dbFrom(ctx, r.db).
		Where("foundation_id = ?", foundationID).
		Order("done_on asc").
		Find(&tasks).Error
	return tasks, err
}

// CompleteTask records a fulfilled duty, replacing an earlier record of the same
// key so a corrected date does not leave two answers behind.
func (r *foundationRepositoryGorm) CompleteTask(ctx context.Context, task *domain.FoundationTask) error {
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("foundation_id = ? AND key = ?", task.FoundationID, task.Key).
			Delete(&domain.FoundationTask{}).Error; err != nil {
			return err
		}
		task.ID = 0
		return tx.Create(task).Error
	})
}

func (r *foundationRepositoryGorm) ClearTask(ctx context.Context, foundationID uint, key string) error {
	return dbFrom(ctx, r.db).
		Where("foundation_id = ? AND key = ?", foundationID, key).
		Delete(&domain.FoundationTask{}).Error
}
