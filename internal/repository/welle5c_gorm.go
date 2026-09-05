package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Die Persistenz der steuerlichen Nebenpflichten (Welle 5c).
//
// Drei der vier Karteien hier haben gemeinsam, dass sie über Geschäftsjahre
// hinweg leben: das Verzeichnis nach § 15a UStG läuft fünf oder zehn Jahre, die
// Bestätigungsabfragen sind der Nachweis einer laufenden Geschäftsbeziehung, und
// die Kurshistorie wird für jedes Jahr rückwirkend gebraucht. Keine von ihnen
// bekommt deshalb ein Geschäftsjahr als Filter — sie würden in jedem Jahr außer
// einem leer aussehen.

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
		return nil, err
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

// -------------------------------------------------------------------------
// Bestätigungsabfragen der USt-IdNr.
// -------------------------------------------------------------------------

type vatIDCheckRepositoryGorm struct {
	db *gorm.DB
}

// NewVatIDCheckRepository creates a GORM-backed VatIDCheckRepository.
func NewVatIDCheckRepository(db *gorm.DB) domain.VatIDCheckRepository {
	return &vatIDCheckRepositoryGorm{db: db}
}

func (r *vatIDCheckRepositoryGorm) FindByContact(ctx context.Context, contactID uint) ([]domain.VatIDCheck, error) {
	out := make([]domain.VatIDCheck, 0)
	err := dbFrom(ctx, r.db).
		Where("contact_id = ?", contactID).
		Order("checked_at desc, id desc").
		Find(&out).Error
	return out, err
}

// FindLatestValid sucht die jüngste gültige Bestätigung zu einer Nummer.
//
// Gefiltert wird in Go und nicht in SQL: die Nummer liegt verschlüsselt in der
// Spalte, und ein Vergleich in der Abfrage träfe den Geheimtext. Die Zahl der
// Abfragen je Kontakt ist zweistellig — das ist der Preis dafür, dass die
// USt-IdNr. eines Geschäftspartners nicht im Klartext in der Datei steht.
func (r *vatIDCheckRepositoryGorm) FindLatestValid(
	ctx context.Context, contactID uint, vatID string,
) (*domain.VatIDCheck, error) {
	var checks []domain.VatIDCheck
	err := dbFrom(ctx, r.db).
		Where("contact_id = ? AND status = ?", contactID, domain.VatIDValid).
		Order("checked_at desc, id desc").
		Find(&checks).Error
	if err != nil {
		return nil, err
	}
	for i := range checks {
		if checks[i].VatID == vatID {
			return &checks[i], nil
		}
	}
	return nil, nil
}

func (r *vatIDCheckRepositoryGorm) Save(ctx context.Context, check *domain.VatIDCheck) error {
	return dbFrom(ctx, r.db).Save(check).Error
}

// -------------------------------------------------------------------------
// Belegnachweis der innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

type supplyEvidenceRepositoryGorm struct {
	db *gorm.DB
}

// NewSupplyEvidenceRepository creates a GORM-backed SupplyEvidenceRepository.
func NewSupplyEvidenceRepository(db *gorm.DB) domain.SupplyEvidenceRepository {
	return &supplyEvidenceRepositoryGorm{db: db}
}

func (r *supplyEvidenceRepositoryGorm) FindByInvoice(ctx context.Context, invoiceID uint) ([]domain.SupplyEvidence, error) {
	out := make([]domain.SupplyEvidence, 0)
	err := dbFrom(ctx, r.db).
		Where("invoice_id = ?", invoiceID).
		Order("date asc, id asc").
		Find(&out).Error
	return out, err
}

func (r *supplyEvidenceRepositoryGorm) FindByInvoices(
	ctx context.Context, invoiceIDs []uint,
) (map[uint][]domain.SupplyEvidence, error) {
	out := map[uint][]domain.SupplyEvidence{}
	if len(invoiceIDs) == 0 {
		return out, nil
	}
	var rows []domain.SupplyEvidence
	err := dbFrom(ctx, r.db).
		Where("invoice_id IN ?", invoiceIDs).
		Order("date asc, id asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.InvoiceID] = append(out[row.InvoiceID], row)
	}
	return out, nil
}

func (r *supplyEvidenceRepositoryGorm) Save(ctx context.Context, evidence *domain.SupplyEvidence) error {
	return dbFrom(ctx, r.db).Save(evidence).Error
}

func (r *supplyEvidenceRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.SupplyEvidence{}, id).Error
}

// -------------------------------------------------------------------------
// Kurshistorie und Umsatzsteuer-Durchschnittskurse
// -------------------------------------------------------------------------

type exchangeRateRepositoryGorm struct {
	db *gorm.DB
}

// NewExchangeRateRepository creates a GORM-backed ExchangeRateRepository.
func NewExchangeRateRepository(db *gorm.DB) domain.ExchangeRateRepository {
	return &exchangeRateRepositoryGorm{db: db}
}

func (r *exchangeRateRepositoryGorm) FindRate(ctx context.Context, currency, date string) (*domain.ExchangeRate, error) {
	var out domain.ExchangeRate
	err := dbFrom(ctx, r.db).Where("currency = ? AND date = ?", currency, date).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *exchangeRateRepositoryGorm) FindRateOnOrBefore(ctx context.Context, currency, date string) (*domain.ExchangeRate, error) {
	var out domain.ExchangeRate
	err := dbFrom(ctx, r.db).
		Where("currency = ? AND date <= ?", currency, date).
		Order("date desc").
		First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *exchangeRateRepositoryGorm) FindRates(ctx context.Context, currency, from, to string) ([]domain.ExchangeRate, error) {
	out := make([]domain.ExchangeRate, 0)
	q := dbFrom(ctx, r.db)
	if currency != "" {
		q = q.Where("currency = ?", currency)
	}
	if from != "" {
		q = q.Where("date >= ?", from)
	}
	if to != "" {
		q = q.Where("date <= ?", to)
	}
	err := q.Order("date asc, currency asc").Find(&out).Error
	return out, err
}

// SaveRate legt den Kurs an oder schreibt ihn fort. Als Upsert über Währung und
// Tag: ein zweiter Abruf desselben Tages ist kein zweiter Kurs.
func (r *exchangeRateRepositoryGorm) SaveRate(ctx context.Context, rate *domain.ExchangeRate) error {
	return dbFrom(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "currency"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"rate_micros", "source", "manual"}),
		}).
		Create(rate).Error
}

func (r *exchangeRateRepositoryGorm) FindVatRate(ctx context.Context, currency, month string) (*domain.VatExchangeRate, error) {
	var out domain.VatExchangeRate
	err := dbFrom(ctx, r.db).Where("currency = ? AND month = ?", currency, month).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *exchangeRateRepositoryGorm) FindVatRates(ctx context.Context, from, to string) ([]domain.VatExchangeRate, error) {
	out := make([]domain.VatExchangeRate, 0)
	q := dbFrom(ctx, r.db)
	if from != "" {
		q = q.Where("month >= ?", from)
	}
	if to != "" {
		q = q.Where("month <= ?", to)
	}
	err := q.Order("month asc, currency asc").Find(&out).Error
	return out, err
}

func (r *exchangeRateRepositoryGorm) SaveVatRate(ctx context.Context, rate *domain.VatExchangeRate) error {
	return dbFrom(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "month"}, {Name: "currency"}},
			UpdateAll: true,
		}).
		Create(rate).Error
}
