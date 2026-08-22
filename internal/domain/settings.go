// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

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
	CompanyName          string `json:"companyName"`
	LegalForm            string `json:"legalForm"`            // e.g. "GmbH", "UG (haftungsbeschränkt)", "Einzelunternehmen"
	FiscalYear           int    `json:"fiscalYear"`           // Active fiscal year (e.g. 2026)
	FiscalYearStartMonth int    `json:"fiscalYearStartMonth"` // 1 = Jan (Kalenderjahr), 7 = Jul (abweichendes Geschäftsjahr), etc.
	TaxNumber            string `json:"taxNumber"`            // Steuernummer
	VatID                string `json:"vatId"`                // USt-IdNr.
	TaxOffice            string `json:"taxOffice"`            // Finanzamt
	IBAN                 string `json:"iban"`
	BIC                  string `json:"bic"`
	BankName             string `json:"bankName"`
	Street               string `json:"street"`
	ZipCity              string `json:"zipCity"`
	Country              string `json:"country"`
	Currency             string `json:"currency"`
	SKR                  string `json:"skr"`             // "SKR04"
	IsSmallBusiness      bool   `json:"isSmallBusiness"` // § 19 UStG Kleinunternehmer
	VatPeriod            string `json:"vatPeriod"`       // "month", "quarter", "year", "exempt"
	TaxationType         string `json:"taxationType"`    // "IST", "SOLL"
}

// GetFiscalYearForDate computes the fiscal year for a given date (YYYY-MM-DD)
// taking into account the configured starting month (1 = Jan, 7 = Jul, etc.).
func GetFiscalYearForDate(dateStr string, startMonth int) int {
	if startMonth <= 0 || startMonth > 12 {
		startMonth = 1
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t = time.Now()
	}
	year := t.Year()
	month := int(t.Month())

	if startMonth == 1 {
		return year
	}

	// Deviating fiscal year (e.g., starts in July):
	// Dates Jan..Jun belong to the fiscal year that started in (year - 1).
	// Dates Jul..Dec belong to the fiscal year starting in (year).
	if month < startMonth {
		return year - 1
	}
	return year
}

// SettingsRepository defines database operations for application configuration.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	GetCompanySettings(ctx context.Context) (*CompanySettings, error)
	UpdateCompanySettings(ctx context.Context, settings *CompanySettings) error
}
