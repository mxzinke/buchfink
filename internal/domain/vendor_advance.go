package domain

import (
	"context"
	"time"
)

// Die geleistete Anzahlung ist die Eingangsseite des Anzahlungsfalls.
//
// Sie ist kein Aufwand und keine Anschaffung: bezahlt ist etwas, geliefert
// nichts. Bilanziell steht sie deshalb als eigener Posten im Vermögen (§ 266
// Abs. 2 A I 4, A II 4, B I 4 HGB) und wird erst mit der Schlussrechnung des
// Lieferanten in den Aufwand oder die Anschaffungskosten umgebucht.
//
// Umsatzsteuerlich hängt der Vorsteuerabzug an der Zahlung: § 15 Abs. 1 Satz 1
// Nr. 1 Satz 3 UStG lässt ihn bei einer Anzahlungsrechnung erst zu, wenn die
// Rechnung vorliegt *und* das Entgelt entrichtet ist. Deshalb wird eine
// Anzahlungsrechnung des Lieferanten nicht bei Erhalt gebucht, sondern bei der
// Zahlung — spiegelbildlich zur eigenen Abschlagsrechnung.

// VendorAdvance is a geleistete Anzahlung: one paid advance to a supplier.
type VendorAdvance struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	ContactID uint `gorm:"index;not null" json:"contactId"`
	// ReceiptID ist der Beleg des Lieferanten, EntryID die Buchung der Zahlung.
	ReceiptID uint `gorm:"index;not null" json:"receiptId"`
	EntryID   uint `gorm:"index;not null" json:"entryId"`

	DocumentNumber string `gorm:"size:50;not null" json:"documentNumber"`
	// Account ist das Konto, auf dem die Anzahlung steht. Es folgt aus der
	// Verwendung und muss festgehalten werden: die Schlussrechnung löst genau
	// dieses Konto wieder auf.
	Account string `gorm:"size:20;not null" json:"account"`
	Target  string `gorm:"size:20;not null" json:"target"`

	NetAmount   Cents   `gorm:"not null" json:"netAmount"`
	TaxAmount   Cents   `gorm:"not null" json:"taxAmount"`
	GrossAmount Cents   `gorm:"not null" json:"grossAmount"`
	TaxRate     TaxRate `gorm:"not null" json:"taxRate"`
	// PaidAt ist der Tag der Zahlung und damit der Zeitraum des
	// Vorsteuerabzugs (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG).
	PaidAt string `gorm:"size:10;not null;index" json:"paidAt"`

	// SettledByEntryID ist die Schlussrechnung des Lieferanten, die die
	// Anzahlung abgesetzt hat. Solange sie fehlt, steht die Anzahlung offen.
	SettledByEntryID *uint     `gorm:"index" json:"settledByEntryId,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// Settled meldet, ob die Anzahlung durch eine Schlussrechnung aufgelöst ist.
func (a *VendorAdvance) Settled() bool { return a.SettledByEntryID != nil }

// VendorAdvanceRepository persistiert die geleisteten Anzahlungen.
type VendorAdvanceRepository interface {
	Save(ctx context.Context, advance *VendorAdvance) error
	FindByID(ctx context.Context, id uint) (*VendorAdvance, error)
	// FindOpen liefert die noch nicht verrechneten Anzahlungen eines
	// Lieferanten; ohne Lieferant alle.
	FindOpen(ctx context.Context, contactID uint) ([]VendorAdvance, error)
}
