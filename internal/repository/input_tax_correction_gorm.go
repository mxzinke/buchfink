package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Das Verzeichnis nach § 15a UStG läuft über mehrere Geschäftsjahre.
// FindAll liest deshalb alle Einträge ohne Geschäftsjahresfilter.

// -------------------------------------------------------------------------
// Verzeichnis der Vorsteuerberichtigung (§ 15a UStG)
// -------------------------------------------------------------------------

type inputTaxCorrectionRepositoryGorm struct {
	db *gorm.DB
}

// NewInputTaxCorrectionRepository creates a GORM-backed
// InputTaxCorrectionRepository.
func NewInputTaxCorrectionRepository(db *gorm.DB) domain.InputTaxCorrectionRepository {
	return &inputTaxCorrectionRepositoryGorm{db: db}
}

// Jeder Lesezugriff lädt die Jahre mit. Ein Eintrag ohne sie hätte keinen
// Verwendungsanteil, und das Verzeichnis zeigte für jedes Wirtschaftsgut nur
// die Anschaffung.
func (r *inputTaxCorrectionRepositoryGorm) preloaded(ctx context.Context) *gorm.DB {
	return dbFrom(ctx, r.db).Preload("Usages", func(db *gorm.DB) *gorm.DB {
		return db.Order("input_tax_usages.fiscal_year asc")
	})
}

func (r *inputTaxCorrectionRepositoryGorm) FindAll(ctx context.Context) ([]domain.InputTaxCorrection, error) {
	out := make([]domain.InputTaxCorrection, 0)
	err := r.preloaded(ctx).Order("acquisition_date asc, id asc").Find(&out).Error
	return out, err
}

func (r *inputTaxCorrectionRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.InputTaxCorrection, error) {
	var out domain.InputTaxCorrection
	if err := r.preloaded(ctx).First(&out, id).Error; err != nil {
		return nil, mapError(err)
	}
	return &out, nil
}

func (r *inputTaxCorrectionRepositoryGorm) FindByAsset(ctx context.Context, assetID uint) (*domain.InputTaxCorrection, error) {
	var out domain.InputTaxCorrection
	err := r.preloaded(ctx).Where("asset_id = ?", assetID).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Save schreibt den Eintrag ohne seine Jahre. Die Jahre haben ihren eigenen Weg:
// sie entstehen einzeln, jedes mit einer Bestätigung des Anwenders, und ein
// Save, das die Liste mitschriebe, könnte ein bestätigtes Jahr mit einem
// mitgeschleppten Vorschlag überschreiben.
func (r *inputTaxCorrectionRepositoryGorm) Save(ctx context.Context, c *domain.InputTaxCorrection) error {
	return dbFrom(ctx, r.db).Omit("Usages").Save(c).Error
}

func (r *inputTaxCorrectionRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.InputTaxCorrection{}, id).Error
}

func (r *inputTaxCorrectionRepositoryGorm) SaveUsage(ctx context.Context, usage *domain.InputTaxUsage) error {
	return dbFrom(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "correction_id"}, {Name: "fiscal_year"}},
			UpdateAll: true,
		}).
		Create(usage).Error
}
