package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type numberGapRepositoryGorm struct {
	db *gorm.DB
}

// NewNumberGapRepository creates a GORM-backed NumberGapRepository.
func NewNumberGapRepository(db *gorm.DB) domain.NumberGapRepository {
	return &numberGapRepositoryGorm{db: db}
}

// Record schreibt den Grund einer Lücke fort.
//
// Bewusst außerhalb der Transaktion, in der die Lücke entstand: der Vermerk
// erklärt, warum eine Nummer verbraucht wurde, und eine zurückgerollte
// Transaktion nähme genau diese Erklärung mit zurück. Aufgerufen wird er
// deshalb erst, wenn feststeht, dass die Nummer nicht mehr frei ist.
func (r *numberGapRepositoryGorm) Record(ctx context.Context, gap *domain.NumberGap) error {
	return dbFrom(ctx, r.db).Create(gap).Error
}

func (r *numberGapRepositoryGorm) FindByYear(ctx context.Context, key domain.NumberRangeKey, fiscalYear int) ([]domain.NumberGap, error) {
	gaps := make([]domain.NumberGap, 0)
	err := dbFrom(ctx, r.db).
		Where("key = ? AND fiscal_year = ?", key, fiscalYear).
		Order("sequence asc").Find(&gaps).Error
	return gaps, err
}
