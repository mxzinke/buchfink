// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type numberRangeRepositoryGorm struct {
	db *gorm.DB
}

// NewNumberRangeRepository creates a GORM-backed NumberRangeRepository.
func NewNumberRangeRepository(db *gorm.DB) domain.NumberRangeRepository {
	return &numberRangeRepositoryGorm{db: db}
}

func (r *numberRangeRepositoryGorm) Allocate(ctx context.Context, key domain.NumberRangeKey, fiscalYear int) (int64, error) {
	var seq int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		seq, err = allocateNumber(tx, key, fiscalYear)
		return err
	})
	return seq, err
}

func (r *numberRangeRepositoryGorm) Peek(ctx context.Context, key domain.NumberRangeKey, fiscalYear int) (int64, error) {
	var rec domain.NumberRange
	err := r.db.WithContext(ctx).
		Where("key = ? AND fiscal_year = ?", key, fiscalYear).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return rec.Next, nil
}

// allocateNumber consumes the next value of a counter inside an existing
// transaction. The caller's transaction boundary is what guarantees that a
// failed insert releases the number again instead of leaving a gap.
func allocateNumber(tx *gorm.DB, key domain.NumberRangeKey, fiscalYear int) (int64, error) {
	rec := domain.NumberRange{Key: key, FiscalYear: fiscalYear, Next: 1}

	// Create the counter if it does not exist yet, then read it back under the
	// transaction so concurrent writers serialise on the same row.
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rec).Error; err != nil {
		return 0, fmt.Errorf("Nummernkreis %s konnte nicht angelegt werden: %w", key, err)
	}

	var current domain.NumberRange
	if err := tx.Where("key = ? AND fiscal_year = ?", key, fiscalYear).First(&current).Error; err != nil {
		return 0, fmt.Errorf("Nummernkreis %s konnte nicht gelesen werden: %w", key, err)
	}

	seq := current.Next
	if err := tx.Model(&domain.NumberRange{}).
		Where("key = ? AND fiscal_year = ?", key, fiscalYear).
		Update("next", seq+1).Error; err != nil {
		return 0, fmt.Errorf("Nummernkreis %s konnte nicht fortgeschrieben werden: %w", key, err)
	}

	return seq, nil
}
