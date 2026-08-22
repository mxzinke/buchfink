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

// Contact is a business partner with its own Personenkonto.
//
// The collective accounts 1200 (Forderungen aus LuL) and 3300 (Verbindlichkeiten
// aus LuL) carry the balance sheet figures, but they cannot answer "who owes me
// what". Every partner therefore gets a real Personenkonto from the DATEV ranges
// (10000-69999 Debitoren, 70000-99999 Kreditoren); open items are booked on it,
// and the balance sheet aggregates them into the collective account.
type Contact struct {
	ID   uint        `gorm:"primaryKey" json:"id"`
	Type ContactType `gorm:"size:20;not null;index" json:"type"`

	// LedgerAccount is the Personenkonto, e.g. "10001" or "70023".
	LedgerAccount string `gorm:"size:10;uniqueIndex;not null" json:"ledgerAccount"`

	Name             string `gorm:"size:255;not null;index" json:"name"`
	Company          string `gorm:"size:255;serializer:encrypted" json:"company"`
	Email            string `gorm:"size:255;serializer:encrypted" json:"email"`
	Address          string `gorm:"type:text;serializer:encrypted" json:"address"`
	TaxID            string `gorm:"size:50;serializer:encrypted" json:"taxId"` // Steuernummer
	VatID            string `gorm:"size:50;serializer:encrypted" json:"vatId"` // USt-IdNr.
	CountryCode      string `gorm:"size:2;default:'DE'" json:"countryCode"`
	IBAN             string `gorm:"size:34;serializer:encrypted" json:"iban"`
	BIC              string `gorm:"size:11;serializer:encrypted" json:"bic"`
	PaymentTermsDays int    `gorm:"default:14" json:"paymentTermsDays"`

	// IsPrivate marks a partner who is not an Unternehmer. It decides whether the
	// e-invoice obligation of § 14 Abs. 2 Satz 2 Nr. 1 UStG can apply to a
	// document they issue.
	//
	// The field is phrased negatively on purpose. Business partners are the
	// overwhelming default, so that case has to be the zero value: a boolean with
	// a database default of true can never be set back to false through a struct,
	// and the flag would silently stick.
	//
	// It is master data rather than a guess from the VAT id or a company name — a
	// note about the input tax deduction must not hang on whether somebody filled
	// in a field.
	IsPrivate bool `gorm:"not null;default:false" json:"isPrivate"`
	// IsSmallBusiness marks a partner under § 19 UStG. They may always issue a
	// sonstige Rechnung (§ 34a UStDV), so no e-invoice is owed. This is a
	// property of the *counterparty*; Buchfink's own client is always a
	// bilanzierende Kapitalgesellschaft.
	IsSmallBusiness bool      `gorm:"not null;default:false" json:"isSmallBusiness"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`

	// OpenAmount is the balance of the Personenkonto, computed on read.
	OpenAmount Cents `gorm:"-" json:"openAmount"`

	// TODO: Add support for partner-specific default revenue/expense accounts
	// TODO: Add support for SEPA direct debit mandates (SEPA-Lastschriftmandate)
}

// IsBusiness reports whether the partner is an Unternehmer.
func (c *Contact) IsBusiness() bool { return !c.IsPrivate }

// CollectiveAccount returns the SKR04 Sammelkonto a partner's open items roll up
// into for the balance sheet.
func (c *Contact) CollectiveAccount() string {
	if c.Type == ContactTypeCustomer {
		return AccountForderungenLuL
	}
	return AccountVerbindlichkeitenLuL
}

// IsEUCounterparty reports whether the partner sits in another EU member state,
// which is what makes an innergemeinschaftlicher Erwerb or a tax-exempt
// intra-community supply possible.
func (c *Contact) IsEUCounterparty() bool {
	if c.CountryCode == "" || c.CountryCode == "DE" {
		return false
	}
	_, ok := euMemberStates[c.CountryCode]
	return ok
}

var euMemberStates = map[string]struct{}{
	"AT": {}, "BE": {}, "BG": {}, "CY": {}, "CZ": {}, "DK": {}, "EE": {}, "ES": {},
	"FI": {}, "FR": {}, "GR": {}, "HR": {}, "HU": {}, "IE": {}, "IT": {}, "LT": {},
	"LU": {}, "LV": {}, "MT": {}, "NL": {}, "PL": {}, "PT": {}, "RO": {}, "SE": {},
	"SI": {}, "SK": {},
}

// ContactRepository defines database operations for debtors and creditors.
type ContactRepository interface {
	FindAll(ctx context.Context) ([]Contact, error)
	FindByID(ctx context.Context, id uint) (*Contact, error)
	FindByLedgerAccount(ctx context.Context, account string) (*Contact, error)
	Save(ctx context.Context, contact *Contact) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
}
