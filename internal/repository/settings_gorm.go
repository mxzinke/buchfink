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
		FiscalYearStartMonth: 1,
		Currency:             "EUR",
		SKR:                  "SKR04",
		VatPeriod:            "quarter",
		TaxationType:         "SOLL",
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
		case "fiscal_year_start_month":
			m, _ := strconv.Atoi(it.Value)
			if m >= 1 && m <= 12 {
				settings.FiscalYearStartMonth = m
			}
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
		case "seat":
			settings.Seat = it.Value
		case "register_court":
			settings.RegisterCourt = it.Value
		case "register_number":
			settings.RegisterNumber = it.Value
		case "vat_period":
			settings.VatPeriod = it.Value
		case "taxation_type":
			settings.TaxationType = it.Value
		case "investor_override":
			settings.InvestorOverride = domain.InvestorType(it.Value)
		}
	}

	if settings.FiscalYearStartMonth <= 0 || settings.FiscalYearStartMonth > 12 {
		settings.FiscalYearStartMonth = 1
	}

	return settings, nil
}

func (r *settingsRepositoryGorm) UpdateCompanySettings(ctx context.Context, s *domain.CompanySettings) error {
	vatPeriod := s.VatPeriod
	if vatPeriod == "" {
		vatPeriod = "quarter"
	}
	taxationType := s.TaxationType
	if taxationType == "" {
		taxationType = "SOLL"
	}
	startMonth := s.FiscalYearStartMonth
	if startMonth <= 0 || startMonth > 12 {
		startMonth = 1
	}

	kv := map[string]string{
		"company_name":            s.CompanyName,
		"legal_form":              s.LegalForm,
		"fiscal_year":             fmt.Sprintf("%d", s.FiscalYear),
		"fiscal_year_start_month": fmt.Sprintf("%d", startMonth),
		"tax_number":              s.TaxNumber,
		"vat_id":                  s.VatID,
		"tax_office":              s.TaxOffice,
		"iban":                    s.IBAN,
		"bic":                     s.BIC,
		"bank_name":               s.BankName,
		"street":                  s.Street,
		"zip_city":                s.ZipCity,
		"country":                 s.Country,
		"seat":                    s.Seat,
		"register_court":          s.RegisterCourt,
		"register_number":         s.RegisterNumber,
		"vat_period":              vatPeriod,
		"taxation_type":           taxationType,
		// Leer bleibt leer: die Anlegerstellung folgt dann aus der Rechtsform.
		// Ein hier eingesetzter Vorgabewert wäre eine Festlegung, die niemand
		// getroffen hat — und die die Ableitung stumm überschriebe.
		"investor_override": string(s.InvestorOverride),
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
