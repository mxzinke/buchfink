package domain

import (
	"context"
	"fmt"
	"time"
)

// Appropriation ist der Beschluss der Gesellschafter über die Verwendung des
// Ergebnisses eines Geschäftsjahres (§ 29 GmbHG, § 42a Abs. 2 GmbHG).
//
// Er hängt am Geschäftsjahr, dessen Ergebnis verwendet wird, gebucht wird er
// aber im Folgejahr: der Beschluss fällt nach dem Stichtag, und § 252 Abs. 1
// Nr. 1 HGB verbietet, die Eröffnungsbilanz nachträglich zu ändern. Deshalb
// bringt der Saldenvortrag das Ergebnis zunächst unverwendet auf 2970 bzw.
// 2978, und erst der Beschluss verteilt es.
//
// Eigene Tabelle statt weiterer Felder an FiscalYear: der Beschluss trägt
// Datum, Text, Belegverweis und vier Beträge, und eine Verwendung ohne Beschluss
// gibt es nicht — ein leerer Satz Felder am Geschäftsjahr könnte das nicht
// ausdrücken.
type Appropriation struct {
	// Year ist das Geschäftsjahr, dessen Ergebnis verwendet wird.
	Year int `gorm:"primaryKey;autoIncrement:false" json:"year"`

	DecisionDate string `gorm:"size:10;not null" json:"decisionDate"`
	Text         string `gorm:"size:1000;serializer:encrypted" json:"text,omitempty"`
	// ReceiptID verweist auf das Beschlussprotokoll im Belegspeicher, falls
	// eines abgelegt wurde.
	ReceiptID *uint `gorm:"index" json:"receiptId,omitempty"`

	// NetIncome ist das verwendbare Ergebnis, wie es zum Zeitpunkt des
	// Beschlusses auf dem Vortragskonto stand.
	NetIncome Cents `gorm:"not null" json:"netIncome"`

	// LegalReserve ist die Einstellung in die gesetzliche Rücklage. Bei der UG
	// ist sie Pflicht: § 5a Abs. 3 GmbHG verlangt ein Viertel des um einen
	// Verlustvortrag geminderten Jahresüberschusses, solange das Stammkapital
	// unter 25.000 Euro liegt.
	LegalReserve Cents `gorm:"default:0" json:"legalReserve"`
	// OtherReserves ist die Einstellung in andere Gewinnrücklagen.
	OtherReserves Cents `gorm:"default:0" json:"otherReserves"`
	// Distribution ist die Ausschüttung (brutto, vor Kapitalertragsteuer).
	Distribution Cents `gorm:"default:0" json:"distribution"`
	// WithholdingTax und SolidarityOnWithholding sind die einbehaltene
	// Kapitalertragsteuer (§ 43 Abs. 1 Satz 1 Nr. 1, § 43a Abs. 1 Nr. 1 EStG:
	// 25 %) und der Solidaritätszuschlag darauf (5,5 %).
	WithholdingTax          Cents `gorm:"default:0" json:"withholdingTax"`
	SolidarityOnWithholding Cents `gorm:"default:0" json:"solidarityOnWithholding"`

	// CarryForward ist der Rest, der auf neue Rechnung vorgetragen wird. Er
	// erzeugt keine Buchung — der Betrag steht schon auf dem Vortragskonto.
	CarryForward Cents `gorm:"default:0" json:"carryForward"`

	JournalEntryID *uint `gorm:"index" json:"journalEntryId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// Distributed ist die Summe der verwendeten Beträge ohne den Vortrag.
func (a *Appropriation) Distributed() Cents {
	return a.LegalReserve + a.OtherReserves + a.Distribution
}

// Validate prüft den Beschluss.
func (a *Appropriation) Validate() error {
	if a.Year <= 0 {
		return fmt.Errorf("zum Beschluss gehört das Geschäftsjahr, dessen Ergebnis verwendet wird")
	}
	if a.DecisionDate == "" {
		return fmt.Errorf("zum Beschluss gehört sein Datum")
	}
	if a.LegalReserve < 0 || a.OtherReserves < 0 || a.Distribution < 0 {
		return fmt.Errorf("ein negativer Verwendungsbetrag ist keine Verwendung")
	}
	if a.NetIncome <= 0 && a.Distributed() > 0 {
		return fmt.Errorf(
			"es gibt kein Ergebnis zu verwenden: das Vortragskonto steht auf %s €", a.NetIncome)
	}
	if a.Distributed() > a.NetIncome {
		return fmt.Errorf(
			"die Verwendung von %s € übersteigt das Ergebnis von %s €",
			a.Distributed(), a.NetIncome)
	}
	return nil
}

// AppropriationRepository persistiert die Ergebnisverwendung.
type AppropriationRepository interface {
	FindByYear(ctx context.Context, year int) (*Appropriation, error)
	Save(ctx context.Context, appropriation *Appropriation) error
}
