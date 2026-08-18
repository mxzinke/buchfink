package domain

import (
	"context"
	"io"
	"time"
)

// MatchStatus represents the reconciliation state of a bank transaction.
type MatchStatus string

const (
	MatchStatusUnmatched MatchStatus = "unmatched"
	MatchStatusMatched   MatchStatus = "matched"
	MatchStatusIgnored   MatchStatus = "ignored"
)

// BankTransaction represents a single transaction line from CAMT.053 or open-banking feeds.
type BankTransaction struct {
	ID                uint        `gorm:"primaryKey" json:"id"`
	AccountIBAN       string      `gorm:"size:34;not null;index" json:"accountIban"`
	BookingDate       string      `gorm:"size:10;not null;index" json:"bookingDate"`
	ValueDate         string      `gorm:"size:10;not null" json:"valueDate"`
	Amount            float64     `gorm:"not null" json:"amount"` // Positive: Credit (inflow), Negative: Debit (outflow)
	Currency          string      `gorm:"size:3;default:'EUR'" json:"currency"`
	CounterpartyName  string      `gorm:"size:255;index" json:"counterpartyName"`
	CounterpartyIBAN  string      `gorm:"size:34" json:"counterpartyIban"`
	RemittanceInfo    string      `gorm:"type:text" json:"remittanceInfo"` // Verwendungszweck
	EndToEndID        string      `gorm:"size:100;index" json:"endToEndId"`
	MatchStatus       MatchStatus `gorm:"size:20;default:'unmatched';index" json:"matchStatus"`
	MatchedBookingID  *uint       `gorm:"index" json:"matchedBookingId,omitempty"`
	SuggestedAccount  string      `gorm:"size:10" json:"suggestedAccount,omitempty"`
	SuggestedContact  string      `gorm:"size:255" json:"suggestedContact,omitempty"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`

	// TODO: Add support for CAMT.052 (intraday) and CAMT.054 (credit/debit notifications)
	// TODO: Add support for MT940 legacy format parser
	// TODO: Add multi-bank account management
}

// BankRepository defines persistence operations for bank transactions.
type BankRepository interface {
	FindAll(ctx context.Context) ([]BankTransaction, error)
	FindByID(ctx context.Context, id uint) (*BankTransaction, error)
	CreateBatch(ctx context.Context, transactions []BankTransaction) (int, error)
	MarkMatched(ctx context.Context, id uint, bookingID uint) error
	Count(ctx context.Context) (int64, error)
}

// BankParser defines the contract for bank statement file parsers.
type BankParser interface {
	Parse(r io.Reader) ([]BankTransaction, error)
	// TODO: Add support for bank statement validation rules according to ISO 20022 schemas
}
