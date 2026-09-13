package domain

import (
	"context"
	"strings"
	"unicode"
)

// BankAccount binds a real account to its own ledger account across all years.
type BankAccount struct {
	IBAN          string `gorm:"primaryKey;size:34" json:"iban"`
	Name          string `gorm:"type:text;serializer:encrypted" json:"name"`
	LedgerAccount string `gorm:"size:10;uniqueIndex;not null" json:"ledgerAccount"`
	Currency      string `gorm:"size:3" json:"currency"`
	BIC           string `gorm:"size:11" json:"bic"`
	BankName      string `gorm:"size:120" json:"bankName"`
	// IsInvoiceAccount markiert das Konto, dessen Verbindung auf Rechnungen und
	// Mahnschreiben steht. Höchstens ein Konto trägt die Markierung.
	IsInvoiceAccount bool `gorm:"not null;default:false" json:"isInvoiceAccount"`
}

type BankAccountRepository interface {
	FindAll(ctx context.Context) ([]BankAccount, error)
	Save(ctx context.Context, account *BankAccount) error
}

// NormalizeIBAN removes whitespace and upper-cases the IBAN.
func NormalizeIBAN(iban string) string {
	return strings.ToUpper(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, iban))
}

// ValidIBAN checks the structure and the ISO 7064 check digits of a normalized
// IBAN.
func ValidIBAN(iban string) bool {
	if len(iban) < 15 || len(iban) > 34 || iban[0] < 'A' || iban[0] > 'Z' || iban[1] < 'A' || iban[1] > 'Z' || iban[2] < '0' || iban[2] > '9' || iban[3] < '0' || iban[3] > '9' {
		return false
	}
	n := 0
	for _, c := range iban[4:] + iban[:4] {
		switch {
		case c >= '0' && c <= '9':
			n = (n*10 + int(c-'0')) % 97
		case c >= 'A' && c <= 'Z':
			n = (n*100 + int(c-'A') + 10) % 97
		default:
			return false
		}
	}
	return n == 1
}

// ValidBIC checks the shape of a BIC: eight or eleven characters, bank code
// and country in letters.
func ValidBIC(bic string) bool {
	if len(bic) != 8 && len(bic) != 11 {
		return false
	}
	for i, r := range bic {
		switch {
		case i < 6 && (r < 'A' || r > 'Z'):
			return false
		case i >= 6 && (r < 'A' || r > 'Z') && (r < '0' || r > '9'):
			return false
		}
	}
	return true
}
