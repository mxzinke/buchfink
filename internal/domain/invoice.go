package domain

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// InvoiceStatus is the lifecycle state of an outgoing invoice.
//
// There is no "draft" that already sits in the journal: an invoice is either not
// issued yet, or it is issued and booked. GoBD does not allow an entered
// business transaction to stay editable, so a correction is a Storno plus a new
// invoice, never an edit.
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"     // erfasst, noch nicht ausgestellt und nicht gebucht
	InvoiceStatusIssued    InvoiceStatus = "issued"    // ausgestellt und gebucht, offener Posten
	InvoiceStatusPaid      InvoiceStatus = "paid"      // vollständig ausgeglichen
	InvoiceStatusCancelled InvoiceStatus = "cancelled" // storniert
)

// InvoiceItem is a single position of an outgoing invoice.
type InvoiceItem struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	InvoiceID   uint   `gorm:"index;not null" json:"invoiceId"`
	Position    int    `gorm:"not null" json:"position"`
	Description string `gorm:"size:500;not null" json:"description"`
	// QuantityMilli holds the quantity with three decimal places (1500 = 1,5).
	// An integer keeps 0,1 h × 3 exact, which a float quantity would not.
	QuantityMilli int64   `gorm:"not null" json:"quantityMilli"`
	Unit          string  `gorm:"size:20;default:'Stück'" json:"unit"`
	UnitPrice     Cents   `gorm:"not null" json:"unitPrice"`
	TaxRate       TaxRate `gorm:"not null" json:"taxRate"`
	// PostingGroup is the fachliche Gruppe this position is booked under; it
	// resolves to the revenue account.
	PostingGroup string `gorm:"size:50" json:"postingGroup"`
}

// TotalNet is the net amount of the position.
func (i *InvoiceItem) TotalNet() Cents {
	return MulRound(i.UnitPrice, i.QuantityMilli, 1000)
}

// Invoice is an outgoing invoice capable of ZUGFeRD 2.2 / Factur-X export.
type Invoice struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	FiscalYear    int    `gorm:"index;not null" json:"fiscalYear"`
	InvoiceNumber string `gorm:"size:50;not null;uniqueIndex" json:"invoiceNumber"`

	Date            string `gorm:"size:10;not null;index" json:"date"`      // Rechnungsdatum
	ServiceDateFrom string `gorm:"size:10;not null" json:"serviceDateFrom"` // § 14 Abs. 4 Nr. 6 UStG
	ServiceDateTo   string `gorm:"size:10;not null" json:"serviceDateTo"`
	DueDate         string `gorm:"size:10;not null" json:"dueDate"`

	ContactID   uint   `gorm:"index;not null" json:"contactId"`
	ContactName string `gorm:"size:255;not null;serializer:encrypted" json:"contactName"`

	Items []InvoiceItem `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE" json:"items"`

	// TaxTreatment applies to the invoice as a whole: an invoice is either a
	// domestic taxable supply, an exempt intra-community supply, an export or a
	// § 13b case — never a mixture.
	TaxTreatment TaxTreatment `gorm:"size:40;not null;default:'domestic'" json:"taxTreatment"`

	NetAmount   Cents  `gorm:"not null" json:"netAmount"`
	TaxAmount   Cents  `gorm:"not null" json:"taxAmount"`
	GrossAmount Cents  `gorm:"not null" json:"grossAmount"`
	Currency    string `gorm:"size:3;default:'EUR'" json:"currency"`

	Status InvoiceStatus `gorm:"size:20;default:'draft';index" json:"status"`

	// JournalEntryID links the invoice to the booking it produced.
	JournalEntryID *uint `gorm:"index" json:"journalEntryId,omitempty"`

	ZUGFeRDXML string `gorm:"type:text;serializer:encrypted" json:"zugferdXml,omitempty"`

	// ReceiptID points at the Beleg holding the issued document: the hybrid PDF
	// as the received form and the XML as the structured part. It replaces the
	// former PDFPath, which was never set — a path to a file nothing produced.
	ReceiptID *uint     `gorm:"index" json:"receiptId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// PaidAmount is the settled part, computed on read.
	PaidAmount Cents `gorm:"-" json:"paidAmount"`

	// TODO: Add support for cash discount terms (Skonto) on the invoice itself
	// TODO: XRechnung auch ausstellen (reines XML ohne PDF). Empfangen und
	// gebucht werden kann sie bereits.
}

// TaxGroup is the net base and tax of one VAT rate on an invoice.
type TaxGroup struct {
	Rate TaxRate `json:"rate"`
	Net  Cents   `json:"net"`
	Tax  Cents   `json:"tax"`
}

// TaxGroups splits the positions by VAT rate.
//
// The tax is rounded once per rate group, not per position. Rounding per
// position and summing afterwards produces a total that differs from the tax on
// the invoice total by a cent or two — which is exactly the difference that
// later leaves an open item that never closes.
func (inv *Invoice) TaxGroups() []TaxGroup {
	nets := map[TaxRate]Cents{}
	for i := range inv.Items {
		nets[inv.Items[i].TaxRate] += inv.Items[i].TotalNet()
	}

	rates := make([]TaxRate, 0, len(nets))
	for r := range nets {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })

	groups := make([]TaxGroup, 0, len(rates))
	for _, r := range rates {
		net := nets[r]
		tax := Cents(0)
		// An exempt or shifted-liability invoice shows no tax regardless of the
		// rate stored on its positions.
		if inv.TaxTreatment == TaxTreatmentDomestic {
			tax = r.Tax(net)
		}
		groups = append(groups, TaxGroup{Rate: r, Net: net, Tax: tax})
	}
	return groups
}

// Recalculate derives the invoice totals from its positions.
func (inv *Invoice) Recalculate() {
	var net, tax Cents
	for _, g := range inv.TaxGroups() {
		net += g.Net
		tax += g.Tax
	}
	inv.NetAmount = net
	inv.TaxAmount = tax
	inv.GrossAmount = net + tax
}

// Validate checks the mandatory content of an invoice.
func (inv *Invoice) Validate() error {
	if inv.ContactID == 0 {
		return fmt.Errorf("Rechnungsempfänger fehlt")
	}
	if len(inv.Items) == 0 {
		return fmt.Errorf("die Rechnung hat keine Positionen")
	}
	if inv.Date == "" {
		return fmt.Errorf("Rechnungsdatum fehlt")
	}
	if inv.ServiceDateFrom == "" || inv.ServiceDateTo == "" {
		return fmt.Errorf("Leistungsdatum fehlt (der Leistungszeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG)")
	}
	if inv.ServiceDateTo < inv.ServiceDateFrom {
		return fmt.Errorf("Leistungsende liegt vor dem Leistungsbeginn")
	}
	for i := range inv.Items {
		it := &inv.Items[i]
		if it.Description == "" {
			return fmt.Errorf("Position %d: Beschreibung fehlt", i+1)
		}
		if it.QuantityMilli == 0 {
			return fmt.Errorf("Position %d: Menge darf nicht null sein", i+1)
		}
	}
	return nil
}

// InvoiceRepository defines persistence operations for invoices.
type InvoiceRepository interface {
	FindAll(ctx context.Context, fiscalYear int) ([]Invoice, error)
	FindByID(ctx context.Context, id uint) (*Invoice, error)
	FindByNumber(ctx context.Context, number string) (*Invoice, error)
	Save(ctx context.Context, invoice *Invoice) error
	UpdateStatus(ctx context.Context, id uint, status InvoiceStatus) error
	Count(ctx context.Context, fiscalYear int) (int64, error)
}

// InvoiceRenderer renders invoices into Typst markup.
type InvoiceRenderer interface {
	RenderTypst(invoice *Invoice, seller *CompanySettings, buyer *Contact) (string, error)
}

// ZUGFeRDGenerator creates Factur-X / ZUGFeRD compliant XML.
type ZUGFeRDGenerator interface {
	GenerateXML(invoice *Invoice, seller *CompanySettings, buyer *Contact) (string, error)
	// Die EN-16931-Prüfung sitzt in internal/invoice/en16931.go und deckt eine
	// belegte Teilmenge der Regeln ab. Vollständige Schematron-Äquivalenz bliebe
	// offen — sie setzt einen XSLT-2.0-Prozessor voraus, den Go nicht hat.
}
