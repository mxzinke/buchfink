package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type documentRepositoryGorm struct {
	db *gorm.DB
}

// NewDocumentRepository creates a new GORM-backed DocumentRepository.
func NewDocumentRepository(db *gorm.DB) domain.DocumentRepository {
	return &documentRepositoryGorm{db: db}
}

// FindAll liefert die Ablage, das jüngste Dokument zuerst.
//
// Nach dem Datum des Dokuments und nicht nach dem der Ablage: gesucht wird der
// Gesellschaftsvertrag von der Beurkundung, nicht die Datei, die zuletzt
// hochgeladen wurde. Wo das Datum fehlt, entscheidet die Reihenfolge der Ablage.
func (r *documentRepositoryGorm) FindAll(ctx context.Context) ([]domain.Document, error) {
	documents := make([]domain.Document, 0)
	err := dbFrom(ctx, r.db).Order("document_date desc, id desc").Find(&documents).Error
	return documents, err
}

func (r *documentRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Document, error) {
	var doc domain.Document
	err := dbFrom(ctx, r.db).First(&doc, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *documentRepositoryGorm) FindByDutyKey(ctx context.Context, key string) ([]domain.Document, error) {
	documents := make([]domain.Document, 0)
	if key == "" {
		return documents, nil
	}
	err := dbFrom(ctx, r.db).Where("duty_key = ?", key).Order("id asc").Find(&documents).Error
	return documents, err
}

func (r *documentRepositoryGorm) Create(ctx context.Context, doc *domain.Document) error {
	return dbFrom(ctx, r.db).Create(doc).Error
}

// Delete entfernt den Datensatz. Die Datei auf der Platte räumt der Dienst ab,
// und nur, wenn kein anderes Dokument mehr auf sie zeigt.
func (r *documentRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.Document{}, id).Error
}

// CountBySHA reports how many documents share one file.
func (r *documentRepositoryGorm) CountBySHA(ctx context.Context, sha256 string) (int64, error) {
	var count int64
	err := dbFrom(ctx, r.db).Model(&domain.Document{}).
		Where("sha256 = ?", sha256).Count(&count).Error
	return count, err
}
