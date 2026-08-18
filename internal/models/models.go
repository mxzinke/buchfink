package models

import "time"

// Account represents a ledger account in SKR04.
type Account struct {
	ID                 int64   `json:"id"`
	Number             string  `json:"number"`
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	Category           string  `json:"category"`
	Subcategory        string  `json:"subcategory,omitempty"`
	Kontenklasse       int     `json:"kontenklasse,omitempty"`
	KontenklasseName   string  `json:"kontenklasseName,omitempty"`
	PositionID         string  `json:"positionId,omitempty"`
	Posten             string  `json:"posten,omitempty"`
	BalanceSide        string  `json:"balanceSide,omitempty"`
	HGBCode            string  `json:"hgbCode,omitempty"`
	StatementType      string  `json:"statementType,omitempty"`
	TaxRate            float64 `json:"taxRate"`
	Hauptfunktion      string  `json:"hauptfunktion,omitempty"`
	HauptfunktionDesc  string  `json:"hauptfunktionDesc,omitempty"`
	Zusatzfunktion     string  `json:"zusatzfunktion,omitempty"`
	ZusatzfunktionDesc string  `json:"zusatzfunktionDesc,omitempty"`
	Abschlusszweck     string  `json:"abschlusszweck,omitempty"`
	IsRange            bool    `json:"isRange,omitempty"`
	RangeStart         string  `json:"rangeStart,omitempty"`
	RangeEnd           string  `json:"rangeEnd,omitempty"`
	IsReserved         bool    `json:"isReserved,omitempty"`
	Description        string  `json:"description"`
	IsActive           bool    `json:"isActive"`
	DebitSum           float64 `json:"debitSum"`
	CreditSum          float64 `json:"creditSum"`
	Balance            float64 `json:"balance"`
	BookingsCount      int     `json:"bookingsCount"`
}

// BookingEntry represents a double-entry journal transaction.
type BookingEntry struct {
	ID                int64     `json:"id"`
	BookingNumber     string    `json:"bookingNumber"`
	Date              string    `json:"date"`
	ValueDate         string    `json:"valueDate"`
	Description       string    `json:"description"`
	DebitAccount      string    `json:"debitAccount"`
	DebitAccountName  string    `json:"debitAccountName,omitempty"`
	CreditAccount     string    `json:"creditAccount"`
	CreditAccountName string    `json:"creditAccountName,omitempty"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	ExchangeRate      float64   `json:"exchangeRate"`
	TaxCode           string    `json:"taxCode"`
	TaxAmount         float64   `json:"taxAmount"`
	ReceiptNumber     string    `json:"receiptNumber"`
	ReceiptPath       string    `json:"receiptPath,omitempty"`
	ReceiptHash       string    `json:"receiptHash"`
	EntryHash         string    `json:"entryHash"`
	PreviousHash      string    `json:"previousHash"`
	IsStorno          bool      `json:"isStorno"`
	StornoForID       *int64    `json:"stornoForId,omitempty"`
	BankTxID          *int64    `json:"bankTxId,omitempty"`
	FiscalYear        int       `json:"fiscalYear,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
}

// BankTransaction represents a parsed statement entry from MT940 / CAMT.053.
type BankTransaction struct {
	ID                int64   `json:"id"`
	AccountIBAN       string  `json:"accountIban"`
	BookingDate       string  `json:"bookingDate"`
	ValueDate         string  `json:"valueDate"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	CounterpartyName  string  `json:"counterpartyName"`
	CounterpartyIBAN  string  `json:"counterpartyIban"`
	RemittanceInfo    string  `json:"remittanceInfo"`
	EndToEndID        string  `json:"endToEndId"`
	MatchStatus       string  `json:"matchStatus"`
	MatchedBookingID  *int64  `json:"matchedBookingId,omitempty"`
	SuggestedAccount  string  `json:"suggestedAccount,omitempty"`
	SuggestedContact  string  `json:"suggestedContact,omitempty"`
	FiscalYear        int     `json:"fiscalYear,omitempty"`
}

// Contact represents a Debitor (customer) or Kreditor (vendor).
type Contact struct {
	ID               int64     `json:"id"`
	Number           string    `json:"number"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Company          string    `json:"company"`
	Email            string    `json:"email"`
	Address          string    `json:"address"`
	TaxID            string    `json:"taxId"`
	VatID            string    `json:"vatId"`
	IBAN             string    `json:"iban"`
	BIC              string    `json:"bic"`
	PaymentTermsDays int       `json:"paymentTermsDays"`
	OpenAmount       float64   `json:"openAmount"`
	CreatedAt        time.Time `json:"createdAt"`
}

// InvoiceItem represents a single position in an invoice.
type InvoiceItem struct {
	Position    int     `json:"position"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unitPrice"`
	TaxRate     float64 `json:"taxRate"`
	TotalNet    float64 `json:"totalNet"`
	TotalGross  float64 `json:"totalGross"`
}

// Invoice represents an outgoing or incoming invoice.
type Invoice struct {
	ID            int64         `json:"id"`
	InvoiceNumber string        `json:"invoiceNumber"`
	ContactID     int64         `json:"contactId"`
	ContactName   string        `json:"contactName"`
	Date          string        `json:"date"`
	DueDate       string        `json:"dueDate"`
	Currency      string        `json:"currency"`
	NetAmount     float64       `json:"netAmount"`
	TaxAmount     float64       `json:"taxAmount"`
	GrossAmount   float64       `json:"grossAmount"`
	Status        string        `json:"status"`
	Items         []InvoiceItem `json:"items"`
	PDFPath       string        `json:"pdfPath,omitempty"`
	ZUGFeRDXML    string        `json:"zugferdXml,omitempty"`
	FiscalYear    int           `json:"fiscalYear,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// AuditLogEntry represents an immutable record of system actions.
type AuditLogEntry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"`
	EntityType   string    `json:"entityType"`
	EntityID     string    `json:"entityId"`
	Details      string    `json:"details"`
	EntryHash    string    `json:"entryHash"`
	PreviousHash string    `json:"previousHash"`
}

// FinancialSummary contains high-level KPIs for reporting.
type FinancialSummary struct {
	TotalRevenue    float64             `json:"totalRevenue"`
	TotalExpenses   float64             `json:"totalExpenses"`
	NetIncome       float64             `json:"netIncome"`
	BankBalance     float64             `json:"bankBalance"`
	OpenReceivables float64             `json:"openReceivables"`
	OpenPayables    float64             `json:"openPayables"`
	CashflowHistory []CashflowDataPoint `json:"cashflowHistory"`
}

// CashflowDataPoint represents aggregated cashflow for a single month.
type CashflowDataPoint struct {
	Month   string  `json:"month"`
	Inflow  float64 `json:"inflow"`
	Outflow float64 `json:"outflow"`
	Net     float64 `json:"net"`
}

// IntegrityCheckResult represents the verification report of the hash chain.
type IntegrityCheckResult struct {
	IsValid          bool   `json:"isValid"`
	TotalEntries     int    `json:"totalEntries"`
	CheckedEntries   int    `json:"checkedEntries"`
	FirstBrokenID    *int64 `json:"firstBrokenId,omitempty"`
	Message          string `json:"message"`
	LastVerifiedHash string `json:"lastVerifiedHash"`
	CheckedAt        string `json:"checkedAt"`
}

// CompanySettings holds company configuration.
type CompanySettings struct {
	CompanyName          string `json:"companyName"`
	LegalForm            string `json:"legalForm"`
	FiscalYear           int    `json:"fiscalYear"`
	FiscalYearStartMonth int    `json:"fiscalYearStartMonth"`
	TaxNumber            string `json:"taxNumber"`
	VatID                string `json:"vatId"`
	TaxOffice            string `json:"taxOffice"`
	IBAN                 string `json:"iban"`
	BIC                  string `json:"bic"`
	BankName             string `json:"bankName"`
	Street               string `json:"street"`
	ZipCity              string `json:"zipCity"`
	Country              string `json:"country"`
	Currency             string `json:"currency"`
	SKR                  string `json:"skr"`
	IsSmallBusiness      bool   `json:"isSmallBusiness"`
	VatPeriod            string `json:"vatPeriod"`
	TaxationType         string `json:"taxationType"`
}
