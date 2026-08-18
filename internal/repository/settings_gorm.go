package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type settingsRepositoryGorm struct {
	db *gorm.DB
}

// NewSettingsRepository creates a new GORM-backed SettingsRepository.
func NewSettingsRepository(db *gorm.DB) domain.SettingsRepository {
	return &settingsRepositoryGorm{db: db}
}

func (r *settingsRepositoryGorm) Get(ctx context.Context, key string) (string, error) {
	var item domain.SettingItem
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&item).Error
	if err != nil {
		return "", err
	}
	return item.Value, nil
}

func (r *settingsRepositoryGorm) Set(ctx context.Context, key string, value string) error {
	item := domain.SettingItem{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Save(&item).Error
}

func (r *settingsRepositoryGorm) GetCompanySettings(ctx context.Context) (*domain.CompanySettings, error) {
	var items []domain.SettingItem
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}

	settings := &domain.CompanySettings{
		Currency: "EUR",
		SKR:      "SKR04",
	}

	for _, it := range items {
		switch it.Key {
		case "company_name":
			settings.CompanyName = it.Value
		case "legal_form":
			settings.LegalForm = it.Value
		case "fiscal_year":
			y, _ := strconv.Atoi(it.Value)
			settings.FiscalYear = y
		case "tax_number":
			settings.TaxNumber = it.Value
		case "vat_id":
			settings.VatID = it.Value
		case "tax_office":
			settings.TaxOffice = it.Value
		case "iban":
			settings.IBAN = it.Value
		case "bic":
			settings.BIC = it.Value
		case "bank_name":
			settings.BankName = it.Value
		case "street":
			settings.Street = it.Value
		case "zip_city":
			settings.ZipCity = it.Value
		case "country":
			settings.Country = it.Value
		}
	}

	return settings, nil
}

func (r *settingsRepositoryGorm) UpdateCompanySettings(ctx context.Context, s *domain.CompanySettings) error {
	kv := map[string]string{
		"company_name": s.CompanyName,
		"legal_form":   s.LegalForm,
		"fiscal_year":  fmt.Sprintf("%d", s.FiscalYear),
		"tax_number":   s.TaxNumber,
		"vat_id":       s.VatID,
		"tax_office":   s.TaxOffice,
		"iban":         s.IBAN,
		"bic":          s.BIC,
		"bank_name":    s.BankName,
		"street":       s.Street,
		"zip_city":     s.ZipCity,
		"country":      s.Country,
	}

	for k, v := range kv {
		item := domain.SettingItem{
			Key:       k,
			Value:     v,
			UpdatedAt: time.Now(),
		}
		if err := r.db.WithContext(ctx).Save(&item).Error; err != nil {
			return err
		}
	}

	return nil
}
