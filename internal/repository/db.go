package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitTenantDB initializes a GORM SQLite database for a tenant.
// Master data (accounts, contacts, settings) is overarching, and bookings exist within fiscal years.
func InitTenantDB(dataDir string) (*gorm.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "buchfink.sqlite")

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

	currentYear := time.Now().Year()
	if err := SeedDefaultsIfEmpty(context.Background(), db, currentYear); err != nil {
		return nil, fmt.Errorf("failed to seed initial SKR04 data: %w", err)
	}

	return db, nil
}

// InitDB initializes a GORM SQLite database (backward compatibility alias for InitTenantDB).
func InitDB(dataDir string, year int) (*gorm.DB, error) {
	return InitTenantDB(dataDir)
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
		&domain.JournalEntry{},
		&domain.JournalLine{},
		&domain.EntertainmentDetail{},
		&domain.Receipt{},
		&domain.ReceiptFile{},
		&domain.NumberRange{},
		&domain.PaymentAllocation{},
		&domain.BankTransaction{},
		&domain.Contact{},
		&domain.Invoice{},
		&domain.InvoiceItem{},
		&domain.AuditLogEntry{},
		&domain.SettingItem{},
		&domain.ExchangeRate{},
		&domain.Festschreibung{},
	)
}

// SeedDefaultsIfEmpty populates initial SKR04 chart of accounts and default company settings if database is newly created.
func SeedDefaultsIfEmpty(ctx context.Context, db *gorm.DB, year int) error {
	var count int64
	if err := db.WithContext(ctx).Model(&domain.Account{}).Count(&count).Error; err != nil {
		return err
	}

	if count < 100 {
		if count > 0 {
			_ = db.WithContext(ctx).Exec("DELETE FROM accounts").Error
		}

		// 1. Seed complete SKR04 2026 Accounts
		defaultAccounts := accounting.DefaultSKR04Accounts()
		if len(defaultAccounts) > 0 {
			if err := db.WithContext(ctx).CreateInBatches(&defaultAccounts, 100).Error; err != nil {
				return fmt.Errorf("failed to seed SKR04 accounts: %w", err)
			}
		}
	}

	// 2. Seed Default Company Settings
	var sCount int64
	if err := db.WithContext(ctx).Model(&domain.SettingItem{}).Count(&sCount).Error; err == nil && sCount == 0 {
		defaultSettings := []domain.SettingItem{
			{Key: "company_name", Value: ""},
			{Key: "legal_form", Value: "GmbH"},
			{Key: "fiscal_year", Value: fmt.Sprintf("%d", year)},
			{Key: "fiscal_year_start_month", Value: "1"},
			{Key: "tax_number", Value: ""},
			{Key: "vat_id", Value: ""},
			{Key: "tax_office", Value: ""},
			{Key: "iban", Value: ""},
			{Key: "bic", Value: ""},
			{Key: "bank_name", Value: ""},
			{Key: "street", Value: ""},
			{Key: "zip_city", Value: ""},
			{Key: "country", Value: "Deutschland"},
			{Key: "currency", Value: "EUR"},
			{Key: "skr", Value: "SKR04"},
			{Key: "vat_period", Value: "quarter"},
			{Key: "taxation_type", Value: "SOLL"},
		}

		for _, s := range defaultSettings {
			if err := db.WithContext(ctx).Save(&s).Error; err != nil {
				return fmt.Errorf("failed to seed setting %s: %w", s.Key, err)
			}
		}
	}

	return nil
}
