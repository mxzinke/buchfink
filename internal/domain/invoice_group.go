package domain

import (
	"context"
	"fmt"
	"time"
)

// Der Rechnungsverbund hält zusammen, was fachlich ein Vorgang ist: der
// vereinbarte Gesamtbetrag, die Abschlagsrechnungen mit ihrem Zahlungsstand und
// die Schlussrechnung mit der Verrechnung.
//
// Er ist eine eigene Entität und keine Kette von Verweisen zwischen Rechnungen,
// weil sich die entscheidende Frage sonst nicht beantworten lässt: welche
// Anzahlungen muss diese Schlussrechnung absetzen? Wer sie vergisst, weist die
// Steuer zweimal aus und schuldet den Mehrbetrag (§ 14c Abs. 1 UStG) — der
// teuerste Fehler dieses Themas, und er entsteht durch bloßes Weglassen.

// InvoiceGroup is a Rechnungsverbund: one order billed in advances and a final
// invoice.
type InvoiceGroup struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	FiscalYear int    `gorm:"index;not null" json:"fiscalYear"`
	ContactID  uint   `gorm:"index;not null" json:"contactId"`
	Title      string `gorm:"size:255;not null;serializer:encrypted" json:"title"`
	// TotalNet ist der vereinbarte Gesamtbetrag netto. Er ist die Obergrenze
	// der Abschläge und die Bemessungsgrundlage der Schlussrechnung.
	TotalNet Cents   `gorm:"not null" json:"totalNet"`
	TaxRate  TaxRate `gorm:"not null" json:"taxRate"`
	// Closed wird mit der Schlussrechnung gesetzt. Ein abgeschlossener Verbund
	// nimmt keine weiteren Abschläge auf.
	Closed         bool      `gorm:"not null;default:false;index" json:"closed"`
	FinalInvoiceID *uint     `gorm:"index" json:"finalInvoiceId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// Advances are the Abschlagsrechnungen, filled on read.
	Advances []AdvanceItem `gorm:"-" json:"advances"`

	// Progress ist der Stand des Verbunds, beim Lesen aus den Abschlägen
	// gefüllt (siehe ComputeProgress). Er wird nicht gespeichert: er ist die
	// Summe der Abschläge und keine eigene Wahrheit, die neben ihnen veralten
	// könnte.
	//
	// Er steht im JSON, damit die Oberfläche die Summen anzeigt, die das
	// Backend rechnet, statt sie ein zweites Mal nachzubauen. Innerhalb des
	// Backends bleibt ComputeProgress die Quelle: nur die rechnet auf dem
	// Stand, den der Aufrufer gerade in der Hand hält.
	Progress GroupProgress `gorm:"-" json:"progress"`
}

// AdvanceItem is the open item of an Abschlagsrechnung.
//
// Es ist eine eigene Quelle der OP-Liste und keine Buchung. Bei einer
// Abschlagsrechnung entsteht die Steuer erst mit der Vereinnahmung
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG); beim Ausstellen gibt es deshalb
// nichts zu buchen, aber sehr wohl etwas einzufordern. Eine Merkposten-Buchung
// auf einem statistischen Konto wäre die Alternative gewesen — sie stünde in
// der Summen- und Saldenliste und behauptete einen Geschäftsvorfall, der noch
// gar nicht stattgefunden hat.
type AdvanceItem struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	GroupID   uint `gorm:"index;not null" json:"groupId"`
	InvoiceID uint `gorm:"index;not null;uniqueIndex" json:"invoiceId"`
	ContactID uint `gorm:"index;not null" json:"contactId"`

	InvoiceNumber string  `gorm:"size:50;not null" json:"invoiceNumber"`
	InvoiceDate   string  `gorm:"size:10;not null" json:"invoiceDate"`
	NetAmount     Cents   `gorm:"not null" json:"netAmount"`
	TaxAmount     Cents   `gorm:"not null" json:"taxAmount"`
	GrossAmount   Cents   `gorm:"not null" json:"grossAmount"`
	TaxRate       TaxRate `gorm:"not null" json:"taxRate"`

	// SettledAt und SettlementEntryID halten die Vereinnahmung fest: erst mit
	// ihr entsteht die Steuer und damit die Buchung.
	SettledAt         string `gorm:"size:10;index" json:"settledAt,omitempty"`
	SettlementEntryID *uint  `gorm:"index" json:"settlementEntryId,omitempty"`

	// Cancelled markiert eine stornierte Abschlagsrechnung. Sie fällt aus der
	// Verrechnung heraus: mit ihr entfällt die Rechnung im Sinne des
	// § 14 Abs. 5 Satz 2 UStG und damit der Grund für die Absetzung.
	Cancelled bool `gorm:"not null;default:false" json:"cancelled"`

	// SettledInFinal markiert den Abschlag als in der Schlussrechnung
	// abgesetzt.
	SettledInFinal bool      `gorm:"not null;default:false" json:"settledInFinal"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Settled reports whether the advance has been paid.
func (a *AdvanceItem) Settled() bool { return a.SettledAt != "" }

// DeductibleInFinal reports whether the final invoice has to deduct this
// advance.
//
// Two conditions, both from § 14 Abs. 5 Satz 2 UStG: an Anzahlungsrechnung must
// have been issued, and the money must have come in. Money without an invoice
// was never disclosed and needs no deduction; an invoice without money has not
// produced any tax yet.
func (a *AdvanceItem) DeductibleInFinal() bool { return !a.Cancelled && a.Settled() }

// GroupProgress is what the Verbund view shows.
type GroupProgress struct {
	AgreedNet Cents `json:"agreedNet"`
	// BilledNet ist die Summe der nicht stornierten Abschlagsrechnungen.
	BilledNet Cents `json:"billedNet"`
	// ReceivedNet ist davon der vereinnahmte Teil.
	ReceivedNet   Cents `json:"receivedNet"`
	ReceivedTax   Cents `json:"receivedTax"`
	ReceivedGross Cents `json:"receivedGross"`
	// OpenNet ist der noch nicht abgerechnete Rest des Auftrags.
	OpenNet Cents `json:"openNet"`
	Closed  bool  `json:"closed"`
}

// ComputeProgress sums the advances of the group.
//
// Sie heißt nicht Progress, weil das Feld gleichen Namens den zuletzt gelesenen
// Stand für die Oberfläche trägt; gerechnet wird hier und nur hier.
func (g *InvoiceGroup) ComputeProgress() GroupProgress {
	p := GroupProgress{AgreedNet: g.TotalNet, Closed: g.Closed}
	for i := range g.Advances {
		a := &g.Advances[i]
		if a.Cancelled {
			continue
		}
		p.BilledNet += a.NetAmount
		if a.Settled() {
			p.ReceivedNet += a.NetAmount
			p.ReceivedTax += a.TaxAmount
			p.ReceivedGross += a.GrossAmount
		}
	}
	p.OpenNet = g.TotalNet - p.BilledNet
	return p
}

// DeductibleAdvances returns the advances a final invoice has to settle.
func (g *InvoiceGroup) DeductibleAdvances() []AdvanceItem {
	out := make([]AdvanceItem, 0, len(g.Advances))
	for i := range g.Advances {
		if g.Advances[i].DeductibleInFinal() {
			out = append(out, g.Advances[i])
		}
	}
	return out
}

// EnsureAdvanceFits checks the two invariants that guard a new
// Abschlagsrechnung.
func (g *InvoiceGroup) EnsureAdvanceFits(net Cents) error {
	if g.Closed {
		return fmt.Errorf(
			"der Rechnungsverbund %q ist mit der Schlussrechnung abgeschlossen und nimmt keine weiteren Abschläge auf",
			g.Title)
	}
	progress := g.ComputeProgress()
	if progress.BilledNet+net > g.TotalNet {
		return fmt.Errorf(
			"die Abschläge des Verbunds %q summieren sich mit diesem auf %s € und überschreiten den vereinbarten Gesamtbetrag von %s €",
			g.Title, progress.BilledNet+net, g.TotalNet)
	}
	return nil
}

// InvoiceGroupRepository persists the Rechnungsverbund and its advances.
type InvoiceGroupRepository interface {
	FindAll(ctx context.Context, fiscalYear int) ([]InvoiceGroup, error)
	FindByID(ctx context.Context, id uint) (*InvoiceGroup, error)
	Save(ctx context.Context, group *InvoiceGroup) error
	SaveAdvance(ctx context.Context, advance *AdvanceItem) error
	FindAdvanceByInvoice(ctx context.Context, invoiceID uint) (*AdvanceItem, error)
	// FindOpenAdvances liefert die ausgestellten, noch nicht vereinnahmten
	// Abschläge — die zweite Quelle der OP-Liste.
	FindOpenAdvances(ctx context.Context) ([]AdvanceItem, error)
}
