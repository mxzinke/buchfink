package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestInvoiceService_CreateAndZUGFeRD(t *testing.T) {
	ctx := context.Background()
	_, _, invSvc, _, _ := setupTestServices(t)

	inv := &domain.Invoice{
		InvoiceNumber: "RE-2024-501",
		Date:          "2024-06-01",
		DueDate:       "2024-06-15",
		ContactID:     1,
		ContactName:   "Kunde Alpha GmbH",
		Currency:      "EUR",
		Items: []domain.InvoiceItem{
			{
				Position:    1,
				Description: "Consulting Dienstleistungen",
				Quantity:    10,
				Unit:        "Std",
				UnitPrice:   150.0,
				TaxRate:     0.19,
				TotalNet:    1500.0,
				TotalGross:  1785.0,
			},
		},
	}

	if err := invSvc.CreateInvoice(ctx, inv); err != nil {
		t.Fatalf("failed to create invoice: %v", err)
	}

	if inv.NetAmount != 1500.0 || inv.GrossAmount != 1785.0 {
		t.Fatalf("unexpected totals: net=%f gross=%f", inv.NetAmount, inv.GrossAmount)
	}

	xml, typst, err := invSvc.GenerateZUGFeRDAndTypst(ctx, inv.ID)
	if err != nil {
		t.Fatalf("failed to generate ZUGFeRD: %v", err)
	}

	if !strings.Contains(xml, "CrossIndustryInvoice") || !strings.Contains(xml, "RE-2024-501") {
		t.Fatalf("expected valid ZUGFeRD XML header, got %s", xml)
	}

	if !strings.Contains(typst, "RECHNUNG") || !strings.Contains(typst, "RE-2024-501") {
		t.Fatalf("expected valid Typst template, got %s", typst)
	}
}
