package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes a GORM SQLite database for a specific fiscal year.
func InitDB(dataDir string, year int) (*gorm.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, fmt.Sprintf("buchfink_%d.sqlite", year))
	// DSN with WAL mode and busy timeout for concurrent safety
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)", dbPath)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite db at %s: %w", dbPath, err)
	}

	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to run database automigrations: %w", err)
	}

	if err := SeedDefaultsIfEmpty(context.Background(), db, year); err != nil {
		return nil, fmt.Errorf("failed to seed initial SKR04 data: %w", err)
	}

	return db, nil
}

// InitInMemoryDB initializes an ephemeral SQLite database for unit and integration testing.
func InitInMemoryDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

// AutoMigrate applies schema changes for all domain entities.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Account{},
		&domain.BookingEntry{},
		&domain.BankTransaction{},
		&domain.Contact{},
		&domain.Invoice{},
		&domain.InvoiceItem{},
		&domain.AuditLogEntry{},
		&domain.SettingItem{},
		&domain.ExchangeRate{},
	)
}

// SeedDefaultsIfEmpty populates initial SKR04 chart of accounts and default company settings if database is newly created.
func SeedDefaultsIfEmpty(ctx context.Context, db *gorm.DB, year int) error {
	var count int64
	if err := db.WithContext(ctx).Model(&domain.Account{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	// 1. Seed SKR04 Accounts
	defaultAccounts := accounting.DefaultSKR04Accounts()
	var domainAccounts []domain.Account
	for _, a := range defaultAccounts {
		domainAccounts = append(domainAccounts, domain.Account{
			Number:      a.Number,
			Name:        a.Name,
			Type:        domain.AccountType(a.Type),
			Category:    a.Category,
			TaxRate:     a.TaxRate,
			Description: a.Description,
			IsActive:    a.IsActive,
		})
	}

	if err := db.WithContext(ctx).Create(&domainAccounts).Error; err != nil {
		return fmt.Errorf("failed to seed SKR04 accounts: %w", err)
	}

	// 2. Seed Default Company Settings
	defaultSettings := []domain.SettingItem{
		{Key: "company_name", Value: "Musterfirma GmbH"},
		{Key: "legal_form", Value: "GmbH"},
		{Key: "fiscal_year", Value: fmt.Sprintf("%d", year)},
		{Key: "tax_number", Value: "12/345/67890"},
		{Key: "vat_id", Value: "DE123456789"},
		{Key: "tax_office", Value: "Finanzamt Berlin"},
		{Key: "iban", Value: "DE89370400440532013000"},
		{Key: "bic", Value: "COBADEFFXXX"},
		{Key: "bank_name", Value: "Commerzbank"},
		{Key: "street", Value: "Musterstraße 42"},
		{Key: "zip_city", Value: "10115 Berlin"},
		{Key: "country", Value: "Deutschland"},
		{Key: "currency", Value: "EUR"},
		{Key: "skr", Value: "SKR04"},
	}

	for _, s := range defaultSettings {
		if err := db.WithContext(ctx).Save(&s).Error; err != nil {
			return fmt.Errorf("failed to seed setting %s: %w", s.Key, err)
		}
	}

	return nil
}
