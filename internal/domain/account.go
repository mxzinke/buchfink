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
	Zusatzfunktion     string      `gorm:"size:20" json:"zusatzfunktion"` // "KU", "M", "V"
	ZusatzfunktionDesc string      `gorm:"size:255" json:"zusatzfunktionDesc"`
	Abschlusszweck     string      `gorm:"size:20" json:"abschlusszweck"` // "EÜR", "HB", "SB"
	IsRange            bool        `gorm:"default:false;index" json:"isRange"`
	RangeStart         string      `gorm:"size:10;index" json:"rangeStart"`
	RangeEnd           string      `gorm:"size:10;index" json:"rangeEnd"`
	IsReserved         bool        `gorm:"default:false" json:"isReserved"`
	Description        string      `gorm:"type:text" json:"description"` // User or DATEV explanation
	IsActive           bool        `gorm:"default:true" json:"isActive"`
	CreatedAt          time.Time   `json:"createdAt"`
	UpdatedAt          time.Time   `json:"updatedAt"`

	// Dynamic calculated balances (not persisted directly in table)
	DebitSum      Cents `gorm:"-" json:"debitSum"`
	CreditSum     Cents `gorm:"-" json:"creditSum"`
	Balance       Cents `gorm:"-" json:"balance"`
	BookingsCount int   `gorm:"-" json:"bookingsCount"`
	// AggregatedAccounts is the number of Personenkonten this balance is
	// collected from; zero for ordinary Sachkonten.
	AggregatedAccounts int `gorm:"-" json:"aggregatedAccounts,omitempty"`
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

// CounterAccount is one account on the opposite side of a booking. A multi-line
// entry has several, so the Kontoblatt shows a list rather than the single
// "Gegenkonto" a two-account model would suggest.
type CounterAccount struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	Amount  Cents  `json:"amount"`
}

// AccountLedgerRow is one line of the Kontoblatt.
type AccountLedgerRow struct {
	EntryID         uint             `json:"entryId"`
	EntryNumber     string           `json:"entryNumber"`
	BookingDate     string           `json:"bookingDate"`
	DocumentDate    string           `json:"documentDate"`
	DocumentNumber  string           `json:"documentNumber,omitempty"`
	Description     string           `json:"description"`
	Kind            EntryKind        `json:"kind"`
	Side            Side             `json:"side"`
	DebitAmount     Cents            `json:"debitAmount"`
	CreditAmount    Cents            `json:"creditAmount"`
	RunningBalance  Cents            `json:"runningBalance"`
	CounterAccounts []CounterAccount `json:"counterAccounts"`
	TaxKey          string           `json:"taxKey,omitempty"`
}

// AccountLedger is the Kontoblatt of a single account.
type AccountLedger struct {
	Account    Account `json:"account"`
	FiscalYear int     `json:"fiscalYear"`
	// From und To sind die Grenzen des ausgewerteten Zeitraums. Leer heißt: das
	// ganze Geschäftsjahr. Sie stehen im Ergebnis, weil ein Kontoblatt ohne
	// seinen Zeitraum eine Zahlenreihe ohne Aussage ist.
	From           string             `json:"from,omitempty"`
	To             string             `json:"to,omitempty"`
	OpeningBalance Cents              `json:"openingBalance"`
	TotalDebit     Cents              `json:"totalDebit"`
	TotalCredit    Cents              `json:"totalCredit"`
	ClosingBalance Cents              `json:"closingBalance"`
	RowCount       int                `json:"rowCount"`
	Rows           []AccountLedgerRow `json:"rows"`
}

// SuSaClassSummary represents the subtotal and accounts for a single Kontenklasse (0-9).
type SuSaClassSummary struct {
	Kontenklasse     int       `json:"kontenklasse"`
	KontenklasseName string    `json:"kontenklasseName"`
	TotalDebit       Cents     `json:"totalDebit"`
	TotalCredit      Cents     `json:"totalCredit"`
	TotalSaldoDebit  Cents     `json:"totalSaldoDebit"`
	TotalSaldoCredit Cents     `json:"totalSaldoCredit"`
	AccountsCount    int       `json:"accountsCount"`
	Accounts         []Account `json:"accounts"`
}

// SuSaOverview represents the Summen- und Saldenliste (Soll-Haben-Übersicht).
type SuSaOverview struct {
	FiscalYear int `json:"fiscalYear"`
	// Cutoff ist der Stichtag, bis zu dem summiert wurde. Leer heißt: das ganze
	// Geschäftsjahr.
	Cutoff           string             `json:"cutoff,omitempty"`
	TotalDebit       Cents              `json:"totalDebit"`
	TotalCredit      Cents              `json:"totalCredit"`
	TotalSaldoDebit  Cents              `json:"totalSaldoDebit"`
	TotalSaldoCredit Cents              `json:"totalSaldoCredit"`
	IsBalanced       bool               `json:"isBalanced"`
	Difference       Cents              `json:"difference"`
	Classes          []SuSaClassSummary `json:"classes"`
}
