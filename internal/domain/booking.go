package domain

import (
	"context"
	"time"
)

// GenesisHash is the root anchor for the first booking entry in a fiscal year.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// BookingEntry represents a double-entry journal line with cryptographic hash chain.
type BookingEntry struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	FiscalYear        int       `gorm:"index;not null" json:"fiscalYear"`                  // e.g. 2026
	BookingNumber     string    `gorm:"size:50;not null;index" json:"bookingNumber"`       // e.g. "B-2024-0001"
	Date              string    `gorm:"size:10;not null;index" json:"date"`                // YYYY-MM-DD
	ValueDate         string    `gorm:"size:10;not null" json:"valueDate"`                 // YYYY-MM-DD
	Description       string    `gorm:"size:500;not null;serializer:encrypted" json:"description"` // Buchungstext (verschlüsselt)
	DebitAccount      string    `gorm:"size:10;not null;index" json:"debitAccount"`        // Soll-Konto (e.g. "1800")
	DebitAccountName  string    `gorm:"-" json:"debitAccountName,omitempty"`
	CreditAccount     string    `gorm:"size:10;not null;index" json:"creditAccount"`       // Haben-Konto (e.g. "4400")
	CreditAccountName string    `gorm:"-" json:"creditAccountName,omitempty"`
	Amount            float64   `gorm:"not null" json:"amount"`                            // Betrag in EUR
	Currency          string    `gorm:"size:3;default:'EUR'" json:"currency"`
	ExchangeRate      float64   `gorm:"default:1.0" json:"exchangeRate"`
	TaxCode           string    `gorm:"size:20;default:'NONE'" json:"taxCode"` // e.g. "UST19", "VOST19"
	TaxAmount         float64   `gorm:"default:0.0" json:"taxAmount"`
	ReceiptNumber     string    `gorm:"size:100;index" json:"receiptNumber"` // e.g. "RE-2024-042"
	ReceiptHash       string    `gorm:"size:64" json:"receiptHash"`          // SHA256 of attached file
	ReceiptPath       string    `gorm:"size:255;serializer:encrypted" json:"receiptPath"`
	BankTxID          *uint     `gorm:"index" json:"bankTxId,omitempty"`
	PreviousHash      string    `gorm:"size:64;not null" json:"previousHash"` // Hash chain anchor
	EntryHash         string    `gorm:"size:64;not null" json:"entryHash"`    // Entry SHA256
	IsStorno          bool      `gorm:"default:false;index" json:"isStorno"`  // GoBD correction flag
	StornoForID       *uint     `gorm:"index" json:"stornoForId,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`

	// TODO: Add support for split bookings (Split-Buchungen / Splittabellen)
	// TODO: Add support for cost centers (Kostenstellen / Kostenträger)
}

// FinancialSummary contains high-level KPIs for dashboards and reports.
type FinancialSummary struct {
	TotalRevenue    float64             `json:"totalRevenue"`
	TotalExpenses   float64             `json:"totalExpenses"`
	NetIncome       float64             `json:"netIncome"`
	BankBalance     float64             `json:"bankBalance"`
	OpenReceivables float64             `json:"openReceivables"`
	OpenPayables    float64             `json:"openPayables"`
	CashflowHistory []CashflowDataPoint `json:"cashflowHistory"`
}

// CashflowDataPoint represents aggregated monthly figures.
type CashflowDataPoint struct {
	Month   string  `json:"month"` // "Jan", "Feb", ...
	Inflow  float64 `json:"inflow"`
	Outflow float64 `json:"outflow"`
	Net     float64 `json:"net"`
}

// BookingRepository defines database operations for journal entries.
type BookingRepository interface {
	FindAll(ctx context.Context, fiscalYear int) ([]BookingEntry, error)
	FindByAccount(ctx context.Context, accountNumber string, fiscalYear int) ([]BookingEntry, error)
	FindByID(ctx context.Context, id uint) (*BookingEntry, error)
	FindByStornoForID(ctx context.Context, stornoForID uint) (*BookingEntry, error)
	GetLastEntry(ctx context.Context, fiscalYear int) (*BookingEntry, error)
	Create(ctx context.Context, entry *BookingEntry) error
	CalculateAccountSums(ctx context.Context, accountNumber string, fiscalYear int) (debitSum float64, creditSum float64, err error)
	CalculateTypeSums(ctx context.Context, accType AccountType, fiscalYear int) (total float64, err error)
	GetMonthlyCashflow(ctx context.Context, fiscalYear int) ([]CashflowDataPoint, error)
	Count(ctx context.Context, fiscalYear int) (int64, error)
	GetAvailableFiscalYears(ctx context.Context) ([]int, error)
}
