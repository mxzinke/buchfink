package domain

import (
	"context"
	"time"
)

// ContactType distinguishes customers (Debitoren) from vendors (Kreditoren).
type ContactType string

const (
	ContactTypeCustomer ContactType = "customer" // Debitor (Kunde)
	ContactTypeVendor   ContactType = "vendor"   // Kreditor (Lieferant)
)

// Contact represents a debtor or creditor with master data and payment terms.
type Contact struct {
	ID               uint        `gorm:"primaryKey" json:"id"`
	Type             ContactType `gorm:"size:20;not null;index" json:"type"`
	Number           string      `gorm:"size:50;uniqueIndex;not null" json:"number"` // e.g. "K-10001" or "L-70001"
	Name             string      `gorm:"size:255;not null;index" json:"name"`
	Company          string      `gorm:"size:255" json:"company"`
	Email            string      `gorm:"size:255" json:"email"`
	Address          string      `gorm:"type:text" json:"address"`
	TaxID            string      `gorm:"size:50" json:"taxId"` // Steuernummer
	VatID            string      `gorm:"size:50" json:"vatId"` // USt-IdNr.
	IBAN             string      `gorm:"size:34" json:"iban"`
	BIC              string      `gorm:"size:11" json:"bic"`
	PaymentTermsDays int         `gorm:"default:14" json:"paymentTermsDays"` // Default payment terms
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`

	// Calculated field for open items (OPOS)
	OpenAmount float64 `gorm:"-" json:"openAmount"`

	// TODO: Add support for customer/vendor specific default revenue/expense accounts
	// TODO: Add support for SEPA direct debit mandates (SEPA-Lastschriftmandate)
}

// ContactRepository defines database operations for debtors and creditors.
type ContactRepository interface {
	FindAll(ctx context.Context) ([]Contact, error)
	FindByID(ctx context.Context, id uint) (*Contact, error)
	FindByNumber(ctx context.Context, number string) (*Contact, error)
	Save(ctx context.Context, contact *Contact) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
	// TODO: Add OPOS balance calculation per contact
}
