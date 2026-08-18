package domain

import (
	"context"
	"time"
)

// SettingItem represents a key-value configuration item in the SQLite database.
type SettingItem struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CompanySettings holds metadata for the business and fiscal year.
type CompanySettings struct {
	CompanyName string `json:"companyName"`
	LegalForm   string `json:"legalForm"`   // e.g. "GmbH", "UG (haftungsbeschränkt)", "Einzelunternehmen"
	FiscalYear  int    `json:"fiscalYear"`  // e.g. 2024
	TaxNumber   string `json:"taxNumber"`   // Steuernummer
	VatID       string `json:"vatId"`       // USt-IdNr.
	TaxOffice   string `json:"taxOffice"`   // Finanzamt
	IBAN        string `json:"iban"`
	BIC         string `json:"bic"`
	BankName    string `json:"bankName"`
	Street      string `json:"street"`
	ZipCity     string `json:"zipCity"`
	Country     string `json:"country"`
	Currency    string `json:"currency"`
	SKR         string `json:"skr"` // "SKR04"

	// TODO: Add company logo path / embedding
	// TODO: Add default tax rates configuration
}

// SettingsRepository defines database operations for application configuration.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	GetCompanySettings(ctx context.Context) (*CompanySettings, error)
	UpdateCompanySettings(ctx context.Context, settings *CompanySettings) error
}
