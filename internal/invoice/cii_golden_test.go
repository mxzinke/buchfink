package invoice

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/goldentest"
)

func TestGoldenInvoiceXMLTaxTreatments(t *testing.T) {
	for _, profile := range []domain.EInvoiceProfile{domain.EInvoiceProfileZUGFeRD, domain.EInvoiceProfileXRechnungCII} {
		for _, info := range domain.TaxTreatments(domain.DirectionOutgoing) {
			name := string(profile) + "/" + string(info.Treatment)
			t.Run(name, func(t *testing.T) {
				inv, seller, buyer := testInvoice(), testSeller(), testBuyer()
				inv.TaxTreatment = info.Treatment
				buyer.LeitwegID, buyer.Email = "991-12345-67", "rechnung@kunde.example"
				if info.Treatment != domain.TaxTreatmentDomestic {
					inv.Items[0].TaxRate = domain.TaxRateNone
				}
				if info.Treatment == domain.TaxTreatmentReverseChargeSupply {
					buyer.CountryCode, buyer.VatID = "AT", "ATU12345678"
				}
				if info.Treatment == domain.TaxTreatmentNotTaxable {
					seller.VatID = ""
				}
				inv.Recalculate()
				goldentest.Assert(t, "invoice_xml/"+name, renderFor(t, inv, seller, buyer, profile))
			})
		}
	}
}
