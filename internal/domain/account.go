package domain

import (
	"context"
	"time"
)

// AccountType classifies accounts according to German GAAP / SKR04.
type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"     // Aktiva (Klasse 0-1)
	AccountTypeLiability AccountType = "liability" // Passiva (Klasse 2-3)
	AccountTypeRevenue   AccountType = "revenue"   // Ertrag (Klasse 4)
	AccountTypeExpense   AccountType = "expense"   // Aufwand (Klasse 6)
)

// Account represents a ledger account in SKR04.
type Account struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	Number      string      `gorm:"size:10;uniqueIndex;not null" json:"number"` // e.g. "1800", "4400"
	Name        string      `gorm:"size:255;not null" json:"name"`              // e.g. "Bank", "Erlöse 19%"
	Type        AccountType `gorm:"size:30;not null" json:"type"`               // "asset", "liability", "revenue", "expense"
	Category    string      `gorm:"size:100;not null" json:"category"`          // e.g. "Liquide Mittel", "Umsatzerlöse"
	TaxRate     float64     `gorm:"default:0.0" json:"taxRate"`                 // e.g. 0.19 for 19% VAT
	Description string      `gorm:"type:text" json:"description"`               // User explanation
	IsActive    bool        `gorm:"default:true" json:"isActive"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`

	// Dynamic calculated balance (not persisted directly in table)
	Balance float64 `gorm:"-" json:"balance"`
}

// AccountRepository defines the database persistence contract for accounts.
type AccountRepository interface {
	FindAll(ctx context.Context) ([]Account, error)
	FindByNumber(ctx context.Context, number string) (*Account, error)
	Create(ctx context.Context, account *Account) error
	CreateBatch(ctx context.Context, accounts []Account) error
	Update(ctx context.Context, account *Account) error
	Count(ctx context.Context) (int64, error)
	// TODO: Add support for filtering accounts by active status or category
	// TODO: Add support for sub-account hierarchies if needed
}
