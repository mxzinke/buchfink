package domain

import (
	"context"
	"time"
)

// InvoiceStatus defines the lifecycle status of an invoice.
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusIssued    InvoiceStatus = "issued"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

// InvoiceItem represents a single line item in an outgoing invoice.
type InvoiceItem struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	InvoiceID   uint    `gorm:"index;not null" json:"invoiceId"`
	Position    int     `gorm:"not null" json:"position"`
	Description string  `gorm:"size:500;not null" json:"description"`
	Quantity    float64 `gorm:"not null" json:"quantity"`
	Unit        string  `gorm:"size:20;default:'Stück'" json:"unit"` // e.g. "Stück", "Std", "Tag"
	UnitPrice   float64 `gorm:"not null" json:"unitPrice"`          // Net unit price
	TaxRate     float64 `gorm:"default:0.19" json:"taxRate"`        // 0.19, 0.07, 0.0
	TotalNet    float64 `gorm:"not null" json:"totalNet"`
	TotalGross  float64 `gorm:"not null" json:"totalGross"`
}

// Invoice represents an outgoing invoice capable of ZUGFeRD 2.2 / Factur-X EN 16931 export.
type Invoice struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	FiscalYear    int           `gorm:"index;not null" json:"fiscalYear"`
	InvoiceNumber string        `gorm:"size:50;not null;index" json:"invoiceNumber"` // Separate numbering series
	Date          string        `gorm:"size:10;not null;index" json:"date"`          // YYYY-MM-DD
	DueDate       string        `gorm:"size:10;not null" json:"dueDate"`             // YYYY-MM-DD
	ContactID     uint          `gorm:"index;not null" json:"contactId"`
	ContactName   string        `gorm:"size:255;not null;serializer:encrypted" json:"contactName"`
	Items         []InvoiceItem `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE" json:"items"`
	NetAmount     float64       `gorm:"not null" json:"netAmount"`
	TaxAmount     float64       `gorm:"not null" json:"taxAmount"`
	GrossAmount   float64       `gorm:"not null" json:"grossAmount"`
	Currency      string        `gorm:"size:3;default:'EUR'" json:"currency"`
	Status        InvoiceStatus `gorm:"size:20;default:'draft';index" json:"status"`
	ZUGFeRDXML    string        `gorm:"type:text;serializer:encrypted" json:"zugferdXml,omitempty"`
	PDFPath       string        `gorm:"size:255;serializer:encrypted" json:"pdfPath,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`

	// TODO: Add support for discount / cash discount terms (Skonto)
	// TODO: Add support for XRechnung XML (pure XML without PDF)
	// TODO: Add support for recurring invoices (Abo-Rechnungen)
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

// InvoiceRenderer defines the contract for rendering invoices into PDF or Typst format.
type InvoiceRenderer interface {
	RenderTypst(invoice *Invoice, seller *CompanySettings, buyer *Contact) (string, error)
	// TODO: Add CLI runner for Typst binary to compile PDF/A-3
}

// ZUGFeRDGenerator defines the contract for creating Factur-X / ZUGFeRD compliant XML.
type ZUGFeRDGenerator interface {
	GenerateXML(invoice *Invoice, seller *CompanySettings, buyer *Contact) (string, error)
	// TODO: Add validation against official Schematron / EN 16931 rules
}
