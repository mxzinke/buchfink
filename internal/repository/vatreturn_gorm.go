package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type vatReturnRepositoryGorm struct {
	db *gorm.DB
}

// NewVatReturnRepository creates a GORM-backed VatReturnRepository.
func NewVatReturnRepository(db *gorm.DB) domain.VatReturnRepository {
	return &vatReturnRepositoryGorm{db: db}
}

// encode legt die Kennziffern als JSON ab.
//
// Der Vordruck ist ein Formular und kein Datenmodell: als Spaltensatz wäre jede
// Änderung des amtlichen Vordrucks eine Schemamigration, und eine übermittelte
// Anmeldung ließe sich danach nicht mehr in ihrer damaligen Gestalt lesen.
func encodeVatReturn(r *domain.VatReturn) error {
	figures, err := json.Marshal(r.Figures)
	if err != nil {
		return fmt.Errorf("die Kennziffern konnten nicht gespeichert werden: %w", err)
	}
	late, err := json.Marshal(r.LateEntries)
	if err != nil {
		return fmt.Errorf("die Nachträge konnten nicht gespeichert werden: %w", err)
	}
	r.FiguresJSON = string(figures)
	r.LateEntriesJSON = string(late)
	return nil
}

func decodeVatReturn(r *domain.VatReturn) {
	r.Figures = nil
	r.LateEntries = nil
	if r.FiguresJSON != "" {
		_ = json.Unmarshal([]byte(r.FiguresJSON), &r.Figures)
	}
	if r.LateEntriesJSON != "" {
		_ = json.Unmarshal([]byte(r.LateEntriesJSON), &r.LateEntries)
	}
	// Auch eine gespeicherte "null" kommt so als leere Liste zurück.
	r.EnsureLists()
}

func (r *vatReturnRepositoryGorm) Create(ctx context.Context, rec *domain.VatReturn) error {
	if err := encodeVatReturn(rec); err != nil {
		return err
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	return dbFrom(ctx, r.db).Create(rec).Error
}

func (r *vatReturnRepositoryGorm) Update(ctx context.Context, rec *domain.VatReturn) error {
	if err := encodeVatReturn(rec); err != nil {
		return err
	}
	return dbFrom(ctx, r.db).Save(rec).Error
}

func (r *vatReturnRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.VatReturn, error) {
	var rec domain.VatReturn
	if err := dbFrom(ctx, r.db).First(&rec, id).Error; err != nil {
		return nil, err
	}
	decodeVatReturn(&rec)
	return &rec, nil
}

func (r *vatReturnRepositoryGorm) FindByFiscalYear(ctx context.Context, fiscalYear int) ([]domain.VatReturn, error) {
	var recs []domain.VatReturn
	err := dbFrom(ctx, r.db).
		Where("fiscal_year = ?", fiscalYear).
		Order("period_key asc, id asc").
		Find(&recs).Error
	for i := range recs {
		decodeVatReturn(&recs[i])
	}
	return recs, err
}

func (r *vatReturnRepositoryGorm) FindByPeriod(ctx context.Context, periodKey string) ([]domain.VatReturn, error) {
	var recs []domain.VatReturn
	err := dbFrom(ctx, r.db).
		Where("period_key = ?", periodKey).
		Order("id desc").
		Find(&recs).Error
	for i := range recs {
		decodeVatReturn(&recs[i])
	}
	return recs, err
}

func (r *vatReturnRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Delete(&domain.VatReturn{}, id).Error
}

// -------------------------------------------------------------
// Zusammenfassende Meldung
// -------------------------------------------------------------

type zmReturnRepositoryGorm struct {
	db *gorm.DB
}

// NewZMReturnRepository creates a GORM-backed ZMReturnRepository.
func NewZMReturnRepository(db *gorm.DB) domain.ZMReturnRepository {
	return &zmReturnRepositoryGorm{db: db}
}

func encodeZMLines(rec *domain.ZMReturn) {
	for i := range rec.Lines {
		ids, err := json.Marshal(rec.Lines[i].EntryIDs)
		if err != nil {
			continue
		}
		rec.Lines[i].EntryIDsJSON = string(ids)
	}
}

func decodeZMLines(rec *domain.ZMReturn) {
	for i := range rec.Lines {
		rec.Lines[i].EntryIDs = nil
		if rec.Lines[i].EntryIDsJSON != "" {
			_ = json.Unmarshal([]byte(rec.Lines[i].EntryIDsJSON), &rec.Lines[i].EntryIDs)
		}
	}
	// Abstimmung und Nachträge werden nicht gespeichert; ohne diese Zusage käme
	// eine geladene Meldung mit `null`-Listen in der Oberfläche an.
	rec.EnsureLists()
}

func (r *zmReturnRepositoryGorm) Create(ctx context.Context, rec *domain.ZMReturn) error {
	encodeZMLines(rec)
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	return dbFrom(ctx, r.db).Create(rec).Error
}

// Update schreibt Kopf und Zeilen fort. Die Zeilen werden ersetzt und nicht
// zusammengeführt: eine Meldung ist der Stand eines Zeitraums, und eine Zeile,
// die aus dem Journal verschwunden ist, darf nicht als Rest stehen bleiben.
func (r *zmReturnRepositoryGorm) Update(ctx context.Context, rec *domain.ZMReturn) error {
	encodeZMLines(rec)
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("zm_return_id = ?", rec.ID).Delete(&domain.ZMLine{}).Error; err != nil {
			return err
		}
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).
			Clauses(clause.OnConflict{UpdateAll: true}).Create(rec).Error
	})
}

func (r *zmReturnRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.ZMReturn, error) {
	var rec domain.ZMReturn
	if err := dbFrom(ctx, r.db).Preload("Lines").First(&rec, id).Error; err != nil {
		return nil, err
	}
	decodeZMLines(&rec)
	return &rec, nil
}

func (r *zmReturnRepositoryGorm) FindByFiscalYear(ctx context.Context, fiscalYear int) ([]domain.ZMReturn, error) {
	var recs []domain.ZMReturn
	err := dbFrom(ctx, r.db).Preload("Lines").
		Where("fiscal_year = ?", fiscalYear).
		Order("period_key asc, id asc").
		Find(&recs).Error
	for i := range recs {
		decodeZMLines(&recs[i])
	}
	return recs, err
}

func (r *zmReturnRepositoryGorm) FindByPeriod(ctx context.Context, periodKey string) ([]domain.ZMReturn, error) {
	var recs []domain.ZMReturn
	err := dbFrom(ctx, r.db).Preload("Lines").
		Where("period_key = ?", periodKey).
		Order("id desc").
		Find(&recs).Error
	for i := range recs {
		decodeZMLines(&recs[i])
	}
	return recs, err
}

func (r *zmReturnRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx, r.db).Select("Lines").Delete(&domain.ZMReturn{ID: id}).Error
}

// -------------------------------------------------------------
// Manuelle Haken an Terminen
// -------------------------------------------------------------

type deadlineRepositoryGorm struct {
	db *gorm.DB
}

// NewDeadlineRepository creates a GORM-backed DeadlineRepository.
func NewDeadlineRepository(db *gorm.DB) domain.DeadlineRepository {
	return &deadlineRepositoryGorm{db: db}
}

func (r *deadlineRepositoryGorm) FindAll(ctx context.Context) ([]domain.DeadlineDone, error) {
	var recs []domain.DeadlineDone
	err := dbFrom(ctx, r.db).Find(&recs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return recs, err
}

func (r *deadlineRepositoryGorm) Mark(ctx context.Context, key, doneOn string) error {
	rec := domain.DeadlineDone{Key: key, DoneOn: doneOn, UpdatedAt: time.Now()}
	return dbFrom(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, UpdateAll: true}).
		Create(&rec).Error
}
