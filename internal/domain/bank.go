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
	ID          uint   `gorm:"primaryKey" json:"id"`
	FiscalYear  int    `gorm:"index" json:"fiscalYear"`
	AccountIBAN string `gorm:"size:34;not null;index" json:"accountIban"`
	BookingDate string `gorm:"size:10;not null;index" json:"bookingDate"`
	ValueDate   string `gorm:"size:10;not null" json:"valueDate"`
	// Amount is positive for money coming in and negative for money going out.
	Amount           Cents       `gorm:"not null" json:"amount"`
	Currency         string      `gorm:"size:3;default:'EUR'" json:"currency"`
	CounterpartyName string      `gorm:"size:255;index" json:"counterpartyName"`
	CounterpartyIBAN string      `gorm:"size:34;serializer:encrypted" json:"counterpartyIban"`
	RemittanceInfo   string      `gorm:"type:text;serializer:encrypted" json:"remittanceInfo"` // Verwendungszweck
	EndToEndID       string      `gorm:"size:100;index" json:"endToEndId"`
	MatchStatus      MatchStatus `gorm:"size:20;default:'unmatched';index" json:"matchStatus"`
	// LedgerAccount is the own liquid account this statement belongs to, e.g.
	// "1800". Hard-coding a single bank account stops working as soon as a
	// company has a second one, a credit card or a payment provider.
	LedgerAccount string    `gorm:"size:10;default:'1800'" json:"ledgerAccount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// MatchedAmount is the part of the transaction already assigned to bookings,
	// computed on read. A statement line can settle several open items.
	MatchedAmount Cents `gorm:"-" json:"matchedAmount"`

	// TODO: Add support for CAMT.052 (intraday) and CAMT.054 (credit/debit notifications)
	// TODO: Add support for MT940 legacy format parser
}

// BankRepository defines persistence operations for bank transactions.
type BankRepository interface {
	FindAll(ctx context.Context, fiscalYear int) ([]BankTransaction, error)
	FindByID(ctx context.Context, id uint) (*BankTransaction, error)
	CreateBatch(ctx context.Context, transactions []BankTransaction) (int, error)
	SetMatchStatus(ctx context.Context, id uint, status MatchStatus) error
	Count(ctx context.Context, fiscalYear int) (int64, error)
}

// BankParser defines the contract for bank statement file parsers.
type BankParser interface {
	Parse(r io.Reader) ([]BankTransaction, error)
	// TODO: Add support for bank statement validation rules according to ISO 20022 schemas
}
