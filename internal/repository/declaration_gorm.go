package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type declarationRepositoryGorm struct {
	db *gorm.DB
}

// NewDeclarationRepository liefert die Ablage der Eingaben und Stände der
// Jahressteuererklärungen.
func NewDeclarationRepository(db *gorm.DB) domain.DeclarationRepository {
	return &declarationRepositoryGorm{db: db}
}

func (r *declarationRepositoryGorm) FindInputs(ctx context.Context, kind string, year int) ([]domain.DeclarationInput, error) {
	out := []domain.DeclarationInput{}
	err := dbFrom(ctx, r.db).Where("kind = ? AND year = ?", kind, year).Order("key").Find(&out).Error
	return out, err
}

func (r *declarationRepositoryGorm) FindInput(ctx context.Context, kind string, year int, key string) (*domain.DeclarationInput, error) {
	var in domain.DeclarationInput
	err := dbFrom(ctx, r.db).Where("kind = ? AND year = ? AND key = ?", kind, year, key).Take(&in).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &in, nil
}

func (r *declarationRepositoryGorm) SaveInput(ctx context.Context, input *domain.DeclarationInput) error {
	return dbFrom(ctx, r.db).Save(input).Error
}

func (r *declarationRepositoryGorm) DeleteInput(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.DeclarationInput{}, id).Error
}

func (r *declarationRepositoryGorm) CreateRecord(ctx context.Context, record *domain.DeclarationRecord) error {
	return dbFrom(ctx, r.db).Create(record).Error
}

func (r *declarationRepositoryGorm) FindRecords(ctx context.Context, kind string, year int) ([]domain.DeclarationRecord, error) {
	out := []domain.DeclarationRecord{}
	err := dbFrom(ctx, r.db).Where("kind = ? AND year = ?", kind, year).Order("id").Find(&out).Error
	return out, err
}
