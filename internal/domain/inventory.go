package domain

import (
	"context"
	"fmt"
	"time"
)

// InventoryCount ist der Inventurwert eines Vorratskontos zum Bilanzstichtag.
//
// Er wird erfasst und nicht gerechnet. § 240 Abs. 1 und 2 HGB verlangt eine
// körperliche Bestandsaufnahme; was auf dem Konto steht, ist der Wert der Zu-
// und Abgänge, die gebucht wurden, und nicht der Bestand, der im Lager liegt.
// Die Differenz zwischen beiden ist genau das, was der Abschluss buchen muss.
//
// Bewertet wird ebenfalls nicht: Verbrauchsfolgeverfahren (§ 256 HGB) und das
// Niederstwertprinzip (§ 253 Abs. 4 HGB) sind Entscheidungen über den einzelnen
// Gegenstand, die niemand aus der Buchführung ableiten kann. Der erfasste Wert
// ist der bewertete Wert.
type InventoryCount struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	FiscalYear int  `gorm:"index;not null" json:"fiscalYear"`
	// Account ist das Vorratskonto (Kontenklasse 1, 1000 bis 1179).
	Account string `gorm:"size:10;not null;index" json:"account"`
	// Amount ist der Inventurwert zum Stichtag.
	Amount Cents `gorm:"not null" json:"amount"`
	// BookValue ist der Buchwert des Kontos vor der Abschlussbuchung. Er wird
	// festgehalten, weil er sich nach der Buchung nicht mehr rekonstruieren
	// lässt und die Bestandsveränderung aus beiden entsteht.
	BookValue Cents `gorm:"not null" json:"bookValue"`

	CountedOn string `gorm:"size:10;not null" json:"countedOn"`
	// Method ist das Aufnahmeverfahren im Klartext — Stichtagsinventur,
	// permanente Inventur, Stichprobe. § 241 HGB lässt mehrere zu, und welches
	// verwendet wurde, ist eine Angabe des Anwenders.
	Method string `gorm:"size:255;not null;serializer:encrypted" json:"method"`
	// ReceiptID verweist auf die Inventurliste im Belegspeicher. Sie ist
	// Pflicht: die Aufnahme selbst ist der Beleg, und ein Bestandswert ohne
	// Liste ist eine Behauptung.
	ReceiptID *uint `gorm:"index" json:"receiptId,omitempty"`

	JournalEntryID *uint `gorm:"index" json:"journalEntryId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// Change ist die Bestandsveränderung, die aus dem Inventurwert folgt: positiv
// bei einer Bestandserhöhung, negativ bei einer Minderung.
func (c *InventoryCount) Change() Cents { return c.Amount - c.BookValue }

// Validate prüft die Erfassung.
func (c *InventoryCount) Validate() error {
	if c.Account == "" {
		return fmt.Errorf("zur Inventur gehört das Vorratskonto")
	}
	if c.Amount < 0 {
		return fmt.Errorf("ein negativer Inventurwert ist kein Bestand")
	}
	if c.CountedOn == "" {
		return fmt.Errorf("zur Inventur gehört der Tag der Aufnahme")
	}
	if c.Method == "" {
		return fmt.Errorf("zur Inventur gehört das Aufnahmeverfahren (§ 241 HGB)")
	}
	if c.ReceiptID == nil {
		return fmt.Errorf("zur Inventur gehört die Inventurliste als Beleg")
	}
	return nil
}

// InventoryRepository persistiert die Inventurwerte.
type InventoryRepository interface {
	FindByYear(ctx context.Context, fiscalYear int) ([]InventoryCount, error)
	Save(ctx context.Context, count *InventoryCount) error
}
