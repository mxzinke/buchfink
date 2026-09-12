package invoice

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestInvoiceUsesCompanyAsBuyerAndIncludesBusinessLetterDetails(t *testing.T) {
	inv, seller, buyer := testInvoice(), testSeller(), testBuyer()
	buyer.Name, buyer.Company = "Timo Muster", "Musterdesign GmbH"
	seller.LegalForm, seller.Seat, seller.ManagingDirectors = "GmbH", "München", "Mara Beispiel"
	doc, err := BuildCII(inv, seller, buyer, domain.EInvoiceProfileZUGFeRD)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Buyer.Name != buyer.Company {
		t.Fatalf("invoice recipient is contact person: %+v", doc.Buyer)
	}
	xml := renderFor(t, inv, seller, buyer, domain.EInvoiceProfileZUGFeRD)
	template := GenerateTypstTemplate(inv, seller, buyer)
	for _, want := range []string{buyer.Company, buyer.Name, seller.RegisterCourt, seller.RegisterNumber, seller.ManagingDirectors} {
		if !strings.Contains(xml, want) {
			t.Errorf("XML missing %q", want)
		}
		if !strings.Contains(template, want) {
			t.Errorf("PDF template missing %q", want)
		}
	}
}

func TestXRechnungIncludesTheMandatoryBusinessProcess(t *testing.T) {
	inv, seller, buyer := testInvoice(), testSeller(), testBuyer()
	buyer.LeitwegID, buyer.Email = "04011000-12345-03", "rechnung@example.invalid"
	xml := renderFor(t, inv, seller, buyer, domain.EInvoiceProfileXRechnungCII)
	if !strings.Contains(xml, "<ram:BusinessProcessSpecifiedDocumentContextParameter>") || !strings.Contains(xml, "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0") {
		t.Fatal("XRechnung would fail PEPPOL-EN16931-R001")
	}
}
