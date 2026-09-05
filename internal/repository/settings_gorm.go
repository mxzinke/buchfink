package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	err := dbFrom(ctx, r.db).Where("key = ?", key).First(&item).Error
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
	return dbFrom(ctx, r.db).Save(&item).Error
}

func (r *settingsRepositoryGorm) GetCompanySettings(ctx context.Context) (*domain.CompanySettings, error) {
	var items []domain.SettingItem
	if err := dbFrom(ctx, r.db).Find(&items).Error; err != nil {
		return nil, err
	}

	settings := &domain.CompanySettings{
		FiscalYearStartMonth: 1,
		Currency:             "EUR",
		SKR:                  "SKR04",
		VatPeriod:            "quarter",
		TaxationType:         "SOLL",
		// Zehn Tage sind die Erfassungsfrist der GoBD Rz. 47; ohne Vorgabe
		// stünde hier null und der Prüflauf meldete jeden Beleg am Tag seines
		// Eingangs als überfällig.
		ReceiptCaptureDays: 10,
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
		case "contact_name":
			settings.ContactName = it.Value
		case "contact_phone":
			settings.ContactPhone = it.Value
		case "contact_email":
			settings.ContactEmail = it.Value
		case "invoice_number_format":
			settings.InvoiceNumberFormat = it.Value
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
		case "permanent_extension":
			settings.PermanentExtension = it.Value == "true"
		case "special_prepayment":
			v, _ := strconv.ParseInt(it.Value, 10, 64)
			settings.SpecialPrepayment = domain.Cents(v)
		case "receipt_capture_days":
			if d, err := strconv.Atoi(it.Value); err == nil && d > 0 {
				settings.ReceiptCaptureDays = d
			}
		case "commit_grace_days":
			if d, err := strconv.Atoi(it.Value); err == nil && d >= 0 {
				settings.CommitGraceDays = d
			}
		}
	}

	if settings.FiscalYearStartMonth <= 0 || settings.FiscalYearStartMonth > 12 {
		settings.FiscalYearStartMonth = 1
	}

	return settings, nil
}

// numberFormatOrDefault prüft die Systematik des Rechnungsnummernkreises.
//
// Ein leeres Feld heißt „nicht festgelegt" und bekommt die Voreinstellung. Ein
// ausgefülltes, aber untaugliches Format wird abgewiesen und nicht ersetzt:
// wer `RE-{JAHR}` einträgt, hat einen Nummernkreis gemeint, in dem jede
// Rechnung dieselbe Nummer trüge — das stillschweigend durch die Voreinstellung
// zu ersetzen ließe ihn glauben, sein Format sei gespeichert
// (siehe domain.ValidateInvoiceNumberFormat).
func numberFormatOrDefault(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return domain.DefaultInvoiceNumberFormat, nil
	}
	if err := domain.ValidateInvoiceNumberFormat(format); err != nil {
		return "", err
	}
	return format, nil
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
	captureDays := s.ReceiptCaptureDays
	if captureDays <= 0 {
		captureDays = 10
	}
	graceDays := s.CommitGraceDays
	if graceDays < 0 {
		graceDays = 0
	}
	numberFormat, err := numberFormatOrDefault(s.InvoiceNumberFormat)
	if err != nil {
		return err
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
		"contact_name":            s.ContactName,
		"contact_phone":           s.ContactPhone,
		"contact_email":           s.ContactEmail,
		"invoice_number_format":   numberFormat,
		"seat":                    s.Seat,
		"register_court":          s.RegisterCourt,
		"register_number":         s.RegisterNumber,
		"vat_period":              vatPeriod,
		"taxation_type":           taxationType,
		// Leer bleibt leer: die Anlegerstellung folgt dann aus der Rechtsform.
		// Ein hier eingesetzter Vorgabewert wäre eine Festlegung, die niemand
		// getroffen hat — und die die Ableitung stumm überschriebe.
		"investor_override": string(s.InvestorOverride),
		// Die Dauerfristverlängerung verschiebt jede Fälligkeit um einen Monat;
		// die Sondervorauszahlung wird im letzten Zeitraum des Jahres
		// angerechnet. Beide gehören zusammen und stehen deshalb nebeneinander.
		"permanent_extension":  strconv.FormatBool(s.PermanentExtension),
		"special_prepayment":   strconv.FormatInt(int64(s.SpecialPrepayment), 10),
		"receipt_capture_days": strconv.Itoa(captureDays),
		"commit_grace_days":    strconv.Itoa(graceDays),
	}

	for k, v := range kv {
		item := domain.SettingItem{
			Key:       k,
			Value:     v,
			UpdatedAt: time.Now(),
		}
		if err := dbFrom(ctx, r.db).Save(&item).Error; err != nil {
			return err
		}
	}

	return nil
}
