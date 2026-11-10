package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

// SaveRate legt den Kurs eines Tages an. Ein von Hand erfasster Kurs ersetzt
// den gespeicherten desselben Tages; ein geholter ersetzt keinen vorhandenen,
// damit ein Kursabruf nie einen Handkurs samt Quelle überschreibt.
func (r *exchangeRateRepositoryGorm) SaveRate(ctx context.Context, rate *domain.ExchangeRate) error {
	conflict := clause.OnConflict{
		Columns:   []clause.Column{{Name: "currency"}, {Name: "date"}},
		DoNothing: true,
	}
	if rate.Manual {
		conflict.DoNothing = false
		conflict.DoUpdates = clause.AssignmentColumns([]string{"rate_micros", "source", "manual"})
	}
	return writer(ctx, r.db).Clauses(conflict).Create(rate).Error
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
