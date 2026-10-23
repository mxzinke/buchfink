package repository

import (
	"context"
	"errors"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// --- Fassungen der Verfahrensdokumentation --------------------------------

type procDocRepositoryGorm struct {
	db *gorm.DB
}

// NewProcedureDocumentationRepository liefert die Fassungen der
// Verfahrensdokumentation.
func NewProcedureDocumentationRepository(db *gorm.DB) domain.ProcedureDocumentationRepository {
	return &procDocRepositoryGorm{db: db}
}

func (r *procDocRepositoryGorm) Create(ctx context.Context, doc *domain.ProcedureDocumentation) error {
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}
	return dbFrom(ctx, r.db).Create(doc).Error
}

func (r *procDocRepositoryGorm) FindAll(ctx context.Context) ([]domain.ProcedureDocumentation, error) {
	docs := make([]domain.ProcedureDocumentation, 0)
	err := dbFrom(ctx, r.db).Order("id desc").Find(&docs).Error
	return docs, err
}

func (r *procDocRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.ProcedureDocumentation, error) {
	var doc domain.ProcedureDocumentation
	err := dbFrom(ctx, r.db).First(&doc, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *procDocRepositoryGorm) CountForDay(ctx context.Context, day string) (int64, error) {
	var count int64
	// Gezählt wird über die Fassungsbezeichnung und nicht über den Zeitpunkt:
	// die Bezeichnung beginnt mit dem Tag, und ein LIKE darauf trifft genau
	// die Fassungen, deren Nummer an diesem Tag vergeben wurde — auch dann,
	// wenn die Datei später aus einer anderen Zeitzone gelesen wird.
	err := dbFrom(ctx, r.db).Model(&domain.ProcedureDocumentation{}).
		Where("version LIKE ?", day+"%").Count(&count).Error
	return count, err
}
