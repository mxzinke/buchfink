package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Die Persistenz der Abschlussbausteine.
//
// Alle Entitäten dieser Welle hängen am Geschäftsjahr und werden über es
// gelesen; die Rückstellung ist die Ausnahme, weil sie wie das Anlagegut über
// Jahre hinweg lebt und ihre Bewegungen aus mehreren Jahren stammen. Deshalb
// liest FindAll bei ihr alles und FindByYear nur die eines Jahres gebildeten —
// der Rückstellungsspiegel braucht beides.

// -------------------------------------------------------------------------
// Abschlussschritte
// -------------------------------------------------------------------------

type closingStepRepositoryGorm struct {
	db *gorm.DB
}

// NewClosingStepRepository creates a GORM-backed ClosingStepRepository.
func NewClosingStepRepository(db *gorm.DB) domain.ClosingStepRepository {
	return &closingStepRepositoryGorm{db: db}
}

func (r *closingStepRepositoryGorm) FindByYear(ctx context.Context, year int) ([]domain.ClosingStep, error) {
	steps := make([]domain.ClosingStep, 0)
	err := r.db.WithContext(ctx).Where("year = ?", year).Find(&steps).Error
	return steps, err
}

// Save legt den Zustand an oder schreibt ihn fort. Als Upsert, weil der
// Schlüssel aus Jahr und Baustein besteht und damit vom Aufrufer gesetzt ist.
func (r *closingStepRepositoryGorm) Save(ctx context.Context, step *domain.ClosingStep) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "year"}, {Name: "key"}},
			UpdateAll: true,
		}).
		Create(step).Error
}

// -------------------------------------------------------------------------
// Rechnungsabgrenzung
// -------------------------------------------------------------------------

type accrualRepositoryGorm struct {
	db *gorm.DB
}

// NewAccrualRepository creates a GORM-backed AccrualRepository.
func NewAccrualRepository(db *gorm.DB) domain.AccrualRepository {
	return &accrualRepositoryGorm{db: db}
}

// Jeder Lesezugriff lädt den Auflösungsplan mit. Ein Posten ohne ihn hätte
// keinen Restbetrag — der ergibt sich aus den gebuchten Auflösungen —, und der
// Bericht zeigte für jeden Posten den vollen Betrag.
func (r *accrualRepositoryGorm) preloaded(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Preload("Releases", func(db *gorm.DB) *gorm.DB {
			return db.Order("accrual_releases.date asc, accrual_releases.id asc")
		}).
		Order("start_date asc, id asc")
}

func (r *accrualRepositoryGorm) FindAll(ctx context.Context) ([]domain.Accrual, error) {
	accruals := make([]domain.Accrual, 0)
	err := r.preloaded(ctx).Find(&accruals).Error
	return accruals, err
}

func (r *accrualRepositoryGorm) FindByYear(ctx context.Context, fiscalYear int) ([]domain.Accrual, error) {
	accruals := make([]domain.Accrual, 0)
	err := r.preloaded(ctx).Where("fiscal_year = ?", fiscalYear).Find(&accruals).Error
	return accruals, err
}

func (r *accrualRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Accrual, error) {
	var accrual domain.Accrual
	err := r.preloaded(ctx).First(&accrual, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &accrual, nil
}

// Save schreibt den Posten samt seines Plans. Der Plan entsteht als Ganzes bei
// der Bildung, weshalb er hier — anders als bei den Anlagenbewegungen —
// mitgeschrieben werden darf.
func (r *accrualRepositoryGorm) Save(ctx context.Context, accrual *domain.Accrual) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(accrual).Error
}

func (r *accrualRepositoryGorm) SaveRelease(ctx context.Context, release *domain.AccrualRelease) error {
	return r.db.WithContext(ctx).Save(release).Error
}

func (r *accrualRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Accrual{}, id).Error
}

// -------------------------------------------------------------------------
// Rückstellungen
// -------------------------------------------------------------------------

type provisionRepositoryGorm struct {
	db *gorm.DB
}

// NewProvisionRepository creates a GORM-backed ProvisionRepository.
func NewProvisionRepository(db *gorm.DB) domain.ProvisionRepository {
	return &provisionRepositoryGorm{db: db}
}

func (r *provisionRepositoryGorm) preloaded(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Preload("Movements", func(db *gorm.DB) *gorm.DB {
			return db.Order("provision_movements.date asc, provision_movements.id asc")
		}).
		Order("expected_date asc, id asc")
}

func (r *provisionRepositoryGorm) FindAll(ctx context.Context) ([]domain.Provision, error) {
	provisions := make([]domain.Provision, 0)
	err := r.preloaded(ctx).Find(&provisions).Error
	return provisions, err
}

func (r *provisionRepositoryGorm) FindByYear(ctx context.Context, fiscalYear int) ([]domain.Provision, error) {
	provisions := make([]domain.Provision, 0)
	err := r.preloaded(ctx).Where("fiscal_year = ?", fiscalYear).Find(&provisions).Error
	return provisions, err
}

func (r *provisionRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Provision, error) {
	var provision domain.Provision
	err := r.preloaded(ctx).First(&provision, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &provision, nil
}

// Save schreibt die Stammdaten und lässt die Bewegungen ausdrücklich aus: sie
// entstehen einzeln neben ihrer Buchung, und ein vollständiges Save würde die
// vorhandenen stillschweigend ersetzen.
func (r *provisionRepositoryGorm) Save(ctx context.Context, provision *domain.Provision) error {
	return r.db.WithContext(ctx).Omit("Movements").Save(provision).Error
}

func (r *provisionRepositoryGorm) AddMovement(ctx context.Context, movement *domain.ProvisionMovement) error {
	return r.db.WithContext(ctx).Create(movement).Error
}

func (r *provisionRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Provision{}, id).Error
}

// -------------------------------------------------------------------------
// Abzinsungszinssätze
// -------------------------------------------------------------------------

type discountRateRepositoryGorm struct {
	db *gorm.DB
}

// NewDiscountRateRepository creates a GORM-backed DiscountRateRepository.
func NewDiscountRateRepository(db *gorm.DB) domain.DiscountRateRepository {
	return &discountRateRepositoryGorm{db: db}
}

func (r *discountRateRepositoryGorm) FindByMonth(ctx context.Context, month string) ([]domain.DiscountRate, error) {
	rates := make([]domain.DiscountRate, 0)
	err := r.db.WithContext(ctx).Where("month = ?", month).Order("average asc, years asc").Find(&rates).Error
	return rates, err
}

func (r *discountRateRepositoryGorm) FindLatestUpTo(ctx context.Context, month string) ([]domain.DiscountRate, error) {
	// coalesce, weil max() über eine leere Menge NULL liefert und das Scannen
	// von NULL in einen string abbricht — eine frische Installation ohne
	// gepflegte Zinstabelle ist der Normalfall, kein Fehlerfall.
	var latest string
	err := r.db.WithContext(ctx).Model(&domain.DiscountRate{}).
		Where("month <= ?", month).
		Select("coalesce(max(month), '')").Scan(&latest).Error
	if err != nil {
		return nil, err
	}
	if latest == "" {
		return make([]domain.DiscountRate, 0), nil
	}
	return r.FindByMonth(ctx, latest)
}

func (r *discountRateRepositoryGorm) Months(ctx context.Context) ([]string, error) {
	months := make([]string, 0)
	err := r.db.WithContext(ctx).Model(&domain.DiscountRate{}).
		Distinct().Order("month desc").Pluck("month", &months).Error
	return months, err
}

func (r *discountRateRepositoryGorm) Save(ctx context.Context, rates []domain.DiscountRate) error {
	if len(rates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "month"}, {Name: "years"}, {Name: "average"}},
			UpdateAll: true,
		}).
		CreateInBatches(rates, 100).Error
}

// -------------------------------------------------------------------------
// Inventurwerte
// -------------------------------------------------------------------------

type inventoryRepositoryGorm struct {
	db *gorm.DB
}

// NewInventoryRepository creates a GORM-backed InventoryRepository.
func NewInventoryRepository(db *gorm.DB) domain.InventoryRepository {
	return &inventoryRepositoryGorm{db: db}
}

func (r *inventoryRepositoryGorm) FindByYear(ctx context.Context, fiscalYear int) ([]domain.InventoryCount, error) {
	counts := make([]domain.InventoryCount, 0)
	err := r.db.WithContext(ctx).Where("fiscal_year = ?", fiscalYear).
		Order("account asc, id asc").Find(&counts).Error
	return counts, err
}

func (r *inventoryRepositoryGorm) Save(ctx context.Context, count *domain.InventoryCount) error {
	return r.db.WithContext(ctx).Save(count).Error
}

// -------------------------------------------------------------------------
// Anhangtexte
// -------------------------------------------------------------------------

type notesTextRepositoryGorm struct {
	db *gorm.DB
}

// NewNotesTextRepository creates a GORM-backed NotesTextRepository.
func NewNotesTextRepository(db *gorm.DB) domain.NotesTextRepository {
	return &notesTextRepositoryGorm{db: db}
}

func (r *notesTextRepositoryGorm) FindByYear(ctx context.Context, year int) ([]domain.NotesText, error) {
	texts := make([]domain.NotesText, 0)
	err := r.db.WithContext(ctx).Where("year = ?", year).Find(&texts).Error
	return texts, err
}

func (r *notesTextRepositoryGorm) Save(ctx context.Context, text *domain.NotesText) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "year"}, {Name: "section"}},
			UpdateAll: true,
		}).
		Create(text).Error
}

// -------------------------------------------------------------------------
// Ergebnisverwendung
// -------------------------------------------------------------------------

type appropriationRepositoryGorm struct {
	db *gorm.DB
}

// NewAppropriationRepository creates a GORM-backed AppropriationRepository.
func NewAppropriationRepository(db *gorm.DB) domain.AppropriationRepository {
	return &appropriationRepositoryGorm{db: db}
}

func (r *appropriationRepositoryGorm) FindByYear(ctx context.Context, year int) (*domain.Appropriation, error) {
	var appropriation domain.Appropriation
	err := r.db.WithContext(ctx).Where("year = ?", year).First(&appropriation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &appropriation, nil
}

func (r *appropriationRepositoryGorm) Save(ctx context.Context, appropriation *domain.Appropriation) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "year"}}, UpdateAll: true}).
		Create(appropriation).Error
}
