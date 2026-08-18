package domain

import (
	"context"
	"time"
)

// AccountType classifies accounts according to German GAAP / SKR04.
type AccountType string

const (
	AccountTypeAsset       AccountType = "asset"       // Aktiva (Klasse 0-1)
	AccountTypeLiability   AccountType = "liability"   // Passiva (Klasse 2-3)
	AccountTypeEquity      AccountType = "equity"      // Eigenkapital (Klasse 2)
	AccountTypeRevenue     AccountType = "revenue"     // Ertrag (Klasse 4)
	AccountTypeExpense     AccountType = "expense"     // Aufwand (Klasse 5-7)
	AccountTypeStatistical AccountType = "statistical" // Statistische Konten / Vortrag (Klasse 9)
)

// Account represents a ledger account in SKR04 according to official DATEV BilRUG standard.
type Account struct {
	ID                 uint        `gorm:"primaryKey" json:"id"`
	Number             string      `gorm:"size:20;uniqueIndex;not null" json:"number"` // e.g. "1800", "4400-4409"
	Name               string      `gorm:"size:255;not null" json:"name"`              // e.g. "Bank", "Erlöse 19 % USt"
	Type               AccountType `gorm:"size:30;not null" json:"type"`               // "asset", "liability", "equity", "revenue", "expense", "statistical"
	Category           string      `gorm:"size:100;not null" json:"category"`          // e.g. "Anlagevermögen", "Umlaufvermögen", "Betriebliche Erträge"
	Subcategory        string      `gorm:"size:100" json:"subcategory"`                // e.g. "Liquide Mittel", "Sachanlagen", "Umsatzerlöse"
	Kontenklasse       int         `gorm:"default:0;index" json:"kontenklasse"`        // 0 bis 9
	KontenklasseName   string      `gorm:"size:100" json:"kontenklasseName"`           // e.g. "Klasse 1: Umlaufvermögenskonten"
	PositionID         string      `gorm:"size:150;index" json:"positionId"`           // e.g. "bilanz.aktiva_b_iv.kassenbestand_bundesbankguthaben_guth"
	Posten             string      `gorm:"size:255" json:"posten"`                     // e.g. "Kassenbestand, Guthaben bei Kreditinstituten..."
	BalanceSide        string      `gorm:"size:50" json:"balanceSide"`                 // "Aktiva", "Passiva", "GuV", "Statistisch"
	HGBCode            string      `gorm:"size:50" json:"hgbCode"`                     // e.g. "Aktiva.B.IV", "GuV.1"
	StatementType      string      `gorm:"size:50" json:"statementType"`               // "Bilanz", "GuV", "Statistisch"
	TaxRate            float64     `gorm:"default:0.0" json:"taxRate"`                 // e.g. 0.19 for 19% VAT
	Hauptfunktion      string      `gorm:"size:20" json:"hauptfunktion"`               // "AM", "AV", "F", "R", "S"
	HauptfunktionDesc  string      `gorm:"size:255" json:"hauptfunktionDesc"`
	Zusatzfunktion     string      `gorm:"size:20" json:"zusatzfunktion"`              // "KU", "M", "V"
	ZusatzfunktionDesc string      `gorm:"size:255" json:"zusatzfunktionDesc"`
	Abschlusszweck     string      `gorm:"size:20" json:"abschlusszweck"`              // "EÜR", "HB", "SB"
	IsRange            bool        `gorm:"default:false;index" json:"isRange"`
	RangeStart         string      `gorm:"size:10;index" json:"rangeStart"`
	RangeEnd           string      `gorm:"size:10;index" json:"rangeEnd"`
	IsReserved         bool        `gorm:"default:false" json:"isReserved"`
	Description        string      `gorm:"type:text" json:"description"`               // User or DATEV explanation
	IsActive           bool        `gorm:"default:true" json:"isActive"`
	CreatedAt          time.Time   `json:"createdAt"`
	UpdatedAt          time.Time   `json:"updatedAt"`

	// Dynamic calculated balances (not persisted directly in table)
	DebitSum      float64 `gorm:"-" json:"debitSum"`
	CreditSum     float64 `gorm:"-" json:"creditSum"`
	Balance       float64 `gorm:"-" json:"balance"`
	BookingsCount int     `gorm:"-" json:"bookingsCount"`
}

// AccountRepository defines the database persistence contract for accounts.
type AccountRepository interface {
	FindAll(ctx context.Context) ([]Account, error)
	FindByNumber(ctx context.Context, number string) (*Account, error)
	Create(ctx context.Context, account *Account) error
	CreateBatch(ctx context.Context, accounts []Account) error
	Update(ctx context.Context, account *Account) error
	Count(ctx context.Context) (int64, error)
}

// AccountLedgerBooking represents an enriched booking entry for an account ledger (Kontoblatt).
type AccountLedgerBooking struct {
	Booking        BookingEntry `json:"booking"`
	CounterAccount string       `json:"counterAccount"`
	CounterName    string       `json:"counterName"`
	Direction      string       `json:"direction"` // "SOLL" or "HABEN"
	DebitAmount    float64      `json:"debitAmount"`
	CreditAmount   float64      `json:"creditAmount"`
	RunningBalance float64      `json:"runningBalance"`
}

// AccountLedger represents the Kontoblatt with all transactions and balances.
type AccountLedger struct {
	Account        Account                `json:"account"`
	FiscalYear     int                    `json:"fiscalYear"`
	OpeningBalance float64                `json:"openingBalance"`
	TotalDebit     float64                `json:"totalDebit"`
	TotalCredit    float64                `json:"totalCredit"`
	ClosingBalance float64                `json:"closingBalance"`
	BookingsCount  int                    `json:"bookingsCount"`
	Entries        []AccountLedgerBooking `json:"entries"`
}

// SuSaClassSummary represents the subtotal and accounts for a single Kontenklasse (0-9).
type SuSaClassSummary struct {
	Kontenklasse     int       `json:"kontenklasse"`
	KontenklasseName string    `json:"kontenklasseName"`
	TotalDebit       float64   `json:"totalDebit"`
	TotalCredit      float64   `json:"totalCredit"`
	TotalSaldoDebit  float64   `json:"totalSaldoDebit"`
	TotalSaldoCredit float64   `json:"totalSaldoCredit"`
	AccountsCount    int       `json:"accountsCount"`
	Accounts         []Account `json:"accounts"`
}

// SuSaOverview represents the Summen- und Saldenliste (Soll-Haben-Übersicht).
type SuSaOverview struct {
	FiscalYear       int                `json:"fiscalYear"`
	TotalDebit       float64            `json:"totalDebit"`
	TotalCredit      float64            `json:"totalCredit"`
	TotalSaldoDebit  float64            `json:"totalSaldoDebit"`
	TotalSaldoCredit float64            `json:"totalSaldoCredit"`
	IsBalanced       bool               `json:"isBalanced"`
	Difference       float64            `json:"difference"`
	Classes          []SuSaClassSummary `json:"classes"`
}
