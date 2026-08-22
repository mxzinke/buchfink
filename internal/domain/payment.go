// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"context"
	"fmt"
)

// DifferenceKind names why a payment does not match the open item exactly.
//
// Without these, every rounding cent and every bank fee leaves an open item
// that never closes, and users eventually clear them with a wrong booking. Each
// kind has its own account, and Skonto additionally corrects the VAT base under
// § 17 UStG.
type DifferenceKind string

const (
	DifferenceNone     DifferenceKind = "none"
	DifferenceSkonto   DifferenceKind = "skonto"   // Skonto, § 17 UStG: Entgelt und Steuer mindern sich
	DifferenceBankFee  DifferenceKind = "bank_fee" // Überweisungsentgelte, Fremdgebühren
	DifferenceRounding DifferenceKind = "rounding" // Kleinbetragsdifferenz
	DifferenceCurrency DifferenceKind = "currency" // realisierte Kursdifferenz, § 256a HGB
)

// DifferenceKindInfo describes a difference kind for the UI.
type DifferenceKindInfo struct {
	Kind  DifferenceKind `json:"kind"`
	Label string         `json:"label"`
	Hint  string         `json:"hint"`
}

// DifferenceKinds lists the difference kinds with their explanations.
func DifferenceKinds() []DifferenceKindInfo {
	return []DifferenceKindInfo{
		{DifferenceNone, "Keine Differenz", "Der Zahlbetrag entspricht dem offenen Posten."},
		{DifferenceSkonto, "Skonto", "Der Rabatt für schnelle Zahlung mindert auch die Umsatz- bzw. Vorsteuer (§ 17 UStG)."},
		{DifferenceBankFee, "Bankgebühr", "Die Bank hat zusätzlich zum Rechnungsbetrag ein Entgelt abgebucht."},
		{DifferenceRounding, "Rundungsdifferenz", "Kleinbetrag, der als Aufwand oder Ertrag ausgebucht wird."},
		{DifferenceCurrency, "Kursdifferenz", "Realisierter Kursgewinn oder -verlust bei einer Fremdwährungszahlung."},
	}
}

// PaymentAllocation records that a payment settled part of an open item.
//
// The relation is many-to-many in both directions: one payment can settle
// several documents (Sammelüberweisung) and one document can be settled by
// several payments (Teilzahlung, Raten). A single foreign key on the bank
// transaction cannot express either.
type PaymentAllocation struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// OpenItemEntryID is the booking that created the open item.
	OpenItemEntryID uint `gorm:"index;not null" json:"openItemEntryId"`
	// PaymentEntryID is the booking that settled it.
	PaymentEntryID uint  `gorm:"index;not null" json:"paymentEntryId"`
	BankTxID       *uint `gorm:"index" json:"bankTxId,omitempty"`
	ContactID      uint  `gorm:"index;not null" json:"contactId"`

	// SettledAmount is the amount the open item is reduced by, including any
	// Skonto or rounding granted.
	SettledAmount Cents `gorm:"not null" json:"settledAmount"`
	// CashAmount is what actually moved on the liquid account.
	CashAmount Cents `gorm:"not null" json:"cashAmount"`

	DifferenceKind   DifferenceKind `gorm:"size:20;default:'none'" json:"differenceKind"`
	DifferenceAmount Cents          `gorm:"default:0" json:"differenceAmount"`
}

// OpenItem is one unsettled receivable or payable.
type OpenItem struct {
	EntryID        uint        `json:"entryId"`
	EntryNumber    string      `json:"entryNumber"`
	ContactID      uint        `json:"contactId"`
	ContactName    string      `json:"contactName"`
	ContactType    ContactType `json:"contactType"`
	LedgerAccount  string      `json:"ledgerAccount"`
	DocumentNumber string      `json:"documentNumber"`
	DocumentDate   string      `json:"documentDate"`
	DueDate        string      `json:"dueDate"`
	GrossAmount    Cents       `json:"grossAmount"`
	SettledAmount  Cents       `json:"settledAmount"`
	OpenAmount     Cents       `json:"openAmount"`
	// TaxRate of the original document, needed to correct the VAT base when a
	// Skonto is granted. Zero when the document carried no VAT.
	TaxRate TaxRate `json:"taxRate"`
}

// Status derives the open item state from its balance, exactly as the concept
// describes it: the status is not stored, it follows from the amounts.
func (o *OpenItem) Status() string {
	switch {
	case o.OpenAmount == 0:
		return "bezahlt"
	case o.SettledAmount == 0:
		return "offen"
	default:
		return "teilbezahlt"
	}
}

// IsOverdue reports whether the due date has passed for a still-open item.
func (o *OpenItem) IsOverdue(today string) bool {
	return o.OpenAmount != 0 && o.DueDate != "" && o.DueDate < today
}

// PaymentAllocationRepository persists the settlement links.
type PaymentAllocationRepository interface {
	Create(ctx context.Context, allocations []PaymentAllocation) error
	FindByOpenItem(ctx context.Context, entryID uint) ([]PaymentAllocation, error)
	SettledByOpenItem(ctx context.Context, fiscalYear int) (map[uint]Cents, error)
	FindByBankTx(ctx context.Context, bankTxID uint) ([]PaymentAllocation, error)
}

// SkontoAccount returns the SKR04 account for a Skonto at a given VAT rate.
func SkontoAccount(dir Direction, rate TaxRate) (string, error) {
	if dir == Direction("") {
		return "", fmt.Errorf("Richtung des Skontos fehlt")
	}
	if dir == DirectionIncoming {
		// Erhaltene Skonti mindern den Aufwand.
		switch rate {
		case TaxRateStandard:
			return "5736", nil
		case TaxRateReduced:
			return "5731", nil
		case TaxRateNone:
			return "5730", nil
		}
		return "", fmt.Errorf("kein Skontokonto für den Steuersatz %s hinterlegt", rate.Label())
	}
	// Gewährte Skonti mindern den Erlös.
	switch rate {
	case TaxRateStandard:
		return "4736", nil
	case TaxRateReduced:
		return "4731", nil
	case TaxRateNone:
		return "4734", nil
	}
	return "", fmt.Errorf("kein Skontokonto für den Steuersatz %s hinterlegt", rate.Label())
}
