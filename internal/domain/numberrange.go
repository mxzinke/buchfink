// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"context"
	"fmt"
	"time"
)

// NumberRangeKey identifies a gapless counter.
//
// GoBD requires numbering to be gapless and free of duplicates, and § 14 Abs. 4
// Nr. 4 UStG requires outgoing invoice numbers to be unique and consecutive.
// Deriving a number from a database row id does not satisfy either: ids are
// shared across fiscal years and skip on rollback. Every series therefore has
// its own counter, allocated inside the same transaction that writes the record.
type NumberRangeKey string

const (
	NumberRangeJournal  NumberRangeKey = "journal"  // Buchungsnummer, je Geschäftsjahr
	NumberRangeReceipt  NumberRangeKey = "receipt"  // Eingangsbeleg, je Geschäftsjahr
	NumberRangeInvoice  NumberRangeKey = "invoice"  // Ausgangsrechnung, je Geschäftsjahr
	NumberRangeDebitor  NumberRangeKey = "debitor"  // Debitorenkonto, jahresübergreifend
	NumberRangeCreditor NumberRangeKey = "creditor" // Kreditorenkonto, jahresübergreifend
)

// Personenkonten-Nummernkreise nach DATEV SKR04:
// Sollsalden Forderungen aus LuL 10000-69999 = Debitoren,
// Habensalden Verbindlichkeiten aus LuL 70000-99999 = Kreditoren.
const (
	DebitorRangeStart  = 10000
	DebitorRangeEnd    = 69999
	CreditorRangeStart = 70000
	CreditorRangeEnd   = 99999
)

// NumberRange persists the next free value of one counter.
type NumberRange struct {
	Key        NumberRangeKey `gorm:"primaryKey;size:30" json:"key"`
	FiscalYear int            `gorm:"primaryKey" json:"fiscalYear"` // 0 = jahresübergreifend
	Next       int64          `gorm:"not null" json:"next"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

// NumberRangeRepository allocates numbers atomically.
type NumberRangeRepository interface {
	// Allocate consumes and returns the next value of a counter.
	Allocate(ctx context.Context, key NumberRangeKey, fiscalYear int) (int64, error)
	// Peek reports the next value without consuming it.
	Peek(ctx context.Context, key NumberRangeKey, fiscalYear int) (int64, error)
}

// FormatJournalNumber renders a Buchungsnummer, e.g. "2026-000001".
func FormatJournalNumber(fiscalYear int, seq int64) string {
	return fmt.Sprintf("%d-%06d", fiscalYear, seq)
}

// FormatReceiptNumber renders an Eingangsbeleg number, e.g. "ER-2026-0001".
func FormatReceiptNumber(fiscalYear int, seq int64) string {
	return fmt.Sprintf("ER-%d-%04d", fiscalYear, seq)
}

// FormatInvoiceNumber renders an Ausgangsrechnung number, e.g. "RE-2026-0001".
func FormatInvoiceNumber(fiscalYear int, seq int64) string {
	return fmt.Sprintf("RE-%d-%04d", fiscalYear, seq)
}

// FormatLedgerAccount renders a Personenkonto number from a sequence value.
func FormatLedgerAccount(kind ContactType, seq int64) (string, error) {
	switch kind {
	case ContactTypeCustomer:
		n := DebitorRangeStart + seq - 1
		if n > DebitorRangeEnd {
			return "", fmt.Errorf("Debitoren-Nummernkreis %d-%d ist erschöpft", DebitorRangeStart, DebitorRangeEnd)
		}
		return fmt.Sprintf("%d", n), nil
	case ContactTypeVendor:
		n := CreditorRangeStart + seq - 1
		if n > CreditorRangeEnd {
			return "", fmt.Errorf("Kreditoren-Nummernkreis %d-%d ist erschöpft", CreditorRangeStart, CreditorRangeEnd)
		}
		return fmt.Sprintf("%d", n), nil
	default:
		return "", fmt.Errorf("unbekannter Kontakttyp %q", kind)
	}
}

// IsLedgerAccount reports whether an account number is a Personenkonto.
func IsLedgerAccount(account string) bool {
	if len(account) != 5 {
		return false
	}
	n := 0
	for _, r := range account {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
	}
	return n >= DebitorRangeStart && n <= CreditorRangeEnd
}

// LedgerAccountKind classifies a Personenkonto as Debitor or Kreditor.
func LedgerAccountKind(account string) (ContactType, bool) {
	if !IsLedgerAccount(account) {
		return "", false
	}
	n := 0
	for _, r := range account {
		n = n*10 + int(r-'0')
	}
	if n <= DebitorRangeEnd {
		return ContactTypeCustomer, true
	}
	return ContactTypeVendor, true
}
