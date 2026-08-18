package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/models"
)

// InvoiceService manages outgoing invoices, ZUGFeRD XML generation and Typst rendering.
type InvoiceService struct {
	invoiceRepo  domain.InvoiceRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
}

func NewInvoiceService(
	invoiceRepo domain.InvoiceRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  invoiceRepo,
		contactRepo:  contactRepo,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
	}
}

func (s *InvoiceService) GetInvoices(ctx context.Context) ([]domain.Invoice, error) {
	return s.invoiceRepo.FindAll(ctx)
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, inv *domain.Invoice) error {
	// Calculate totals if not provided
	var net, tax float64
	for _, it := range inv.Items {
		net += it.TotalNet
		tax += it.TotalNet * it.TaxRate
	}
	if inv.NetAmount == 0 {
		inv.NetAmount = net
	}
	if inv.TaxAmount == 0 {
		inv.TaxAmount = tax
	}
	if inv.GrossAmount == 0 {
		inv.GrossAmount = inv.NetAmount + inv.TaxAmount
	}
	if inv.Status == "" {
		inv.Status = domain.InvoiceStatusIssued
	}

	if err := s.invoiceRepo.Save(ctx, inv); err != nil {
		return err
	}

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionCreate,
		"INVOICE",
		fmt.Sprintf("%d", inv.ID),
		fmt.Sprintf("Rechnung %s über %.2f %s erstellt", inv.InvoiceNumber, inv.GrossAmount, inv.Currency),
	)

	return nil
}

// GenerateZUGFeRDAndTypst generates Factur-X / ZUGFeRD XML and Typst markup source code.
func (s *InvoiceService) GenerateZUGFeRDAndTypst(ctx context.Context, invoiceID uint) (xml string, typst string, err error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return "", "", fmt.Errorf("invoice ID %d not found: %w", invoiceID, err)
	}

	sellerSettings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return "", "", fmt.Errorf("company settings not found: %w", err)
	}

	contact, _ := s.contactRepo.FindByID(ctx, inv.ContactID)
	if contact == nil {
		contact = &domain.Contact{
			Name:    inv.ContactName,
			Address: "Musterkundenweg 1, 10115 Berlin",
			VatID:   "DE987654321",
		}
	}

	// Adapter for internal invoice package
	legacyInv := &models.Invoice{
		InvoiceNumber: inv.InvoiceNumber,
		Date:          inv.Date,
		DueDate:       inv.DueDate,
		NetAmount:     inv.NetAmount,
		TaxAmount:     inv.TaxAmount,
		GrossAmount:   inv.GrossAmount,
		Currency:      inv.Currency,
	}
	for _, it := range inv.Items {
		legacyInv.Items = append(legacyInv.Items, models.InvoiceItem{
			Position:    it.Position,
			Description: it.Description,
			Quantity:    it.Quantity,
			Unit:        it.Unit,
			UnitPrice:   it.UnitPrice,
			TaxRate:     it.TaxRate,
			TotalNet:    it.TotalNet,
			TotalGross:  it.TotalGross,
		})
	}

	legacySeller := &models.CompanySettings{
		CompanyName: sellerSettings.CompanyName,
		TaxNumber:   sellerSettings.TaxNumber,
		VatID:       sellerSettings.VatID,
		IBAN:        sellerSettings.IBAN,
		BIC:         sellerSettings.BIC,
		BankName:    sellerSettings.BankName,
		Street:      sellerSettings.Street,
		ZipCity:     sellerSettings.ZipCity,
	}

	legacyBuyer := &models.Contact{
		Name:    contact.Name,
		Address: contact.Address,
		VatID:   contact.VatID,
	}

	xml, err = invoice.GenerateZUGFeRDXML(legacyInv, legacySeller, legacyBuyer)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ZUGFeRD XML: %w", err)
	}

	typst = invoice.GenerateTypstTemplate(legacyInv, legacySeller, legacyBuyer)
	// TODO: Compile Typst template to PDF using typst CLI or pure-Go typst compiler
	// TODO: Embed Factur-X / ZUGFeRD XML into PDF/A-3 as attachment

	return xml, typst, nil
}
