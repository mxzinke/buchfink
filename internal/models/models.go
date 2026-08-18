package models

import "time"

// Account represents a ledger account in SKR04.
type Account struct {
	ID          int64   `json:"id"`
	Number      string  `json:"number"`      // z.B. "1800", "4400"
	Name        string  `json:"name"`        // z.B. "Bank", "Erlöse 19%"
	Type        string  `json:"type"`        // "asset", "liability", "revenue", "expense"
	Category    string  `json:"category"`    // Unterkategorie z.B. "Finanzkonten", "Forderungen"
	TaxRate     float64 `json:"taxRate"`     // Standard-Steuersatz (z.B. 0.19 für 19%)
	Description string  `json:"description"` // Leicht verständliche Erklärung für Nicht-Buchhalter
	IsActive    bool    `json:"isActive"`
	Balance     float64 `json:"balance"`     // Aktueller Saldo (EUR)
}

// BookingEntry represents a double-entry journal line with cryptographic hash chain.
type BookingEntry struct {
	ID               int64     `json:"id"`
	BookingNumber    string    `json:"bookingNumber"`    // z.B. "B-2024-0001"
	Date             string    `json:"date"`             // "YYYY-MM-DD"
	ValueDate        string    `json:"valueDate"`        // "YYYY-MM-DD"
	Description      string    `json:"description"`      // Buchungstext
	DebitAccount     string    `json:"debitAccount"`     // Soll-Konto (z.B. "1800")
	DebitAccountName string    `json:"debitAccountName"` // Optional Name für UI
	CreditAccount    string    `json:"creditAccount"`    // Haben-Konto (z.B. "4400")
	CreditAccountName string   `json:"creditAccountName"`// Optional Name für UI
	Amount           float64   `json:"amount"`           // Betrag in EUR
	Currency         string    `json:"currency"`         // "EUR", "USD", etc.
	ExchangeRate     float64   `json:"exchangeRate"`     // Umrechnungskurs (1.0 für EUR)
	TaxCode          string    `json:"taxCode"`          // z.B. "UST19", "VOST19", "NONE"
	TaxAmount        float64   `json:"taxAmount"`        // Enthaltene Steuer
	ReceiptNumber    string    `json:"receiptNumber"`    // Belegnummer (z.B. "RE-2024-101")
	ReceiptHash      string    `json:"receiptHash"`      // SHA256 des Originalbelegs
	ReceiptPath      string    `json:"receiptPath"`      // Relativer Pfad zum Beleg
	BankTxID         *int64    `json:"bankTxId,omitempty"` // Verknüpfte Banktransaktion
	PreviousHash     string    `json:"previousHash"`     // Hash der vorherigen Buchung (Hash Chain)
	EntryHash        string    `json:"entryHash"`        // Eigener SHA256-Hash
	IsStorno         bool      `json:"isStorno"`         // GoBD: Stornobuchung?
	StornoForID      *int64    `json:"stornoForId,omitempty"` // Referenz auf stornierte Buchung
	CreatedAt        time.Time `json:"createdAt"`
}

// BankTransaction represents a transaction from CAMT.053 bank statement.
type BankTransaction struct {
	ID                int64     `json:"id"`
	AccountIBAN       string    `json:"accountIban"`
	BookingDate       string    `json:"bookingDate"`
	ValueDate         string    `json:"valueDate"`
	Amount            float64   `json:"amount"` // Positiv: Gutschrift, Negativ: Lastschrift/Überweisung
	Currency          string    `json:"currency"`
	CounterpartyName  string    `json:"counterpartyName"`
	CounterpartyIBAN  string    `json:"counterpartyIban"`
	RemittanceInfo    string    `json:"remittanceInfo"` // Verwendungszweck
	EndToEndID        string    `json:"endToEndId"`
	MatchStatus       string    `json:"matchStatus"`    // "unmatched", "matched", "ignored"
	MatchedBookingID  *int64    `json:"matchedBookingId,omitempty"`
	SuggestedAccount  string    `json:"suggestedAccount,omitempty"` // Automatischer Kontovorschlag
	SuggestedContact  string    `json:"suggestedContact,omitempty"` // Automatischer Kontaktvorschlag
}

// Contact represents a customer or vendor.
type Contact struct {
	ID               int64     `json:"id"`
	Type             string    `json:"type"`             // "customer", "vendor"
	Number           string    `json:"number"`           // z.B. "K-10001" oder "L-70001"
	Name             string    `json:"name"`             // Personen- oder Firmenname
	Company          string    `json:"company"`
	Email            string    `json:"email"`
	Address          string    `json:"address"`
	TaxID            string    `json:"taxId"`            // Steuernummer
	VatID            string    `json:"vatId"`            // USt-IdNr.
	IBAN             string    `json:"iban"`
	BIC              string    `json:"bic"`
	PaymentTermsDays int       `json:"paymentTermsDays"` // Zahlungsziel in Tagen (z.B. 14)
	OpenAmount       float64   `json:"openAmount"`       // Offener Saldo
	CreatedAt        time.Time `json:"createdAt"`
}

// InvoiceItem is a single line item in an invoice.
type InvoiceItem struct {
	Position    int     `json:"position"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`        // "Stück", "Std", "Tag", etc.
	UnitPrice   float64 `json:"unitPrice"`   // Netto
	TaxRate     float64 `json:"taxRate"`     // 0.19 oder 0.07 oder 0.0
	TotalNet    float64 `json:"totalNet"`
	TotalGross  float64 `json:"totalGross"`
}

// Invoice represents an outgoing invoice with ZUGFeRD data.
type Invoice struct {
	ID            int64         `json:"id"`
	InvoiceNumber string        `json:"invoiceNumber"` // z.B. "RE-2024-001"
	Date          string        `json:"date"`
	DueDate       string        `json:"dueDate"`
	ContactID     int64         `json:"contactId"`
	ContactName   string        `json:"contactName"`
	Items         []InvoiceItem `json:"items"`
	NetAmount     float64       `json:"netAmount"`
	TaxAmount     float64       `json:"taxAmount"`
	GrossAmount   float64       `json:"grossAmount"`
	Currency      string        `json:"currency"`
	Status        string        `json:"status"`        // "draft", "issued", "paid", "cancelled"
	ZUGFeRDXML    string        `json:"zugferdXml,omitempty"`
	PDFPath       string        `json:"pdfPath,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// AuditLogEntry logs all critical operations for GoBD compliance.
type AuditLogEntry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"`       // "CREATE", "UPDATE", "STORNO", "IMPORT", "INTEGRITY_CHECK"
	EntityType   string    `json:"entityType"`   // "BOOKING", "ACCOUNT", "BANK_TX", "INVOICE", "SETTINGS"
	EntityID     string    `json:"entityId"`
	Details      string    `json:"details"`
	PreviousHash string    `json:"previousHash"`
	EntryHash    string    `json:"entryHash"`
}

// FinancialSummary contains high-level KPIs for dashboards.
type FinancialSummary struct {
	TotalRevenue    float64            `json:"totalRevenue"`
	TotalExpenses   float64            `json:"totalExpenses"`
	NetIncome       float64            `json:"netIncome"`
	BankBalance     float64            `json:"bankBalance"`
	OpenReceivables float64            `json:"openReceivables"`
	OpenPayables    float64            `json:"openPayables"`
	CashflowHistory []CashflowDataPoint `json:"cashflowHistory"`
}

// CashflowDataPoint represents monthly figures.
type CashflowDataPoint struct {
	Month    string  `json:"month"` // "Jan", "Feb", ...
	Inflow   float64 `json:"inflow"`
	Outflow  float64 `json:"outflow"`
	Net      float64 `json:"net"`
}

// IntegrityCheckResult reports the status of the cryptographic hash chain.
type IntegrityCheckResult struct {
	IsValid            bool   `json:"isValid"`
	TotalEntries       int    `json:"totalEntries"`
	CheckedEntries     int    `json:"checkedEntries"`
	FirstBrokenID      *int64 `json:"firstBrokenId,omitempty"`
	Message            string `json:"message"`
	LastVerifiedHash   string `json:"lastVerifiedHash"`
	CheckedAt          string `json:"checkedAt"`
}

// CompanySettings holds metadata for the business and fiscal year.
type CompanySettings struct {
	CompanyName    string `json:"companyName"`
	LegalForm      string `json:"legalForm"`      // z.B. "GmbH", "UG (haftungsbeschränkt)", "Einzelunternehmen"
	FiscalYear     int    `json:"fiscalYear"`     // z.B. 2024
	TaxNumber      string `json:"taxNumber"`
	VatID          string `json:"vatId"`
	TaxOffice      string `json:"taxOffice"`
	IBAN           string `json:"iban"`
	BIC            string `json:"bic"`
	BankName       string `json:"bankName"`
	Street         string `json:"street"`
	ZipCity        string `json:"zipCity"`
	Country        string `json:"country"`
	Currency       string `json:"currency"`
	SKR            string `json:"skr"`            // "SKR04"
}
