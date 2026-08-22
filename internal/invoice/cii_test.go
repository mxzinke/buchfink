package invoice

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Rundlauf: was der Generator schreibt, muss der Parser wieder lesen können.
// Beide Richtungen an einem Dokument zu prüfen ist der billigste Weg, sie
// auseinanderlaufen zu sehen.
func TestParseReadsWhatTheGeneratorWrote(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	buyer.VatID = "DE987654321"

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("XML erzeugen: %v", err)
	}

	doc, err := ParseCII([]byte(xml))
	if err != nil {
		t.Fatalf("XML lesen: %v", err)
	}

	if doc.Document.ID != inv.InvoiceNumber {
		t.Errorf("Rechnungsnummer = %q, erwartet %q", doc.Document.ID, inv.InvoiceNumber)
	}
	if doc.Document.TypeCode != "380" {
		t.Errorf("Dokumenttyp = %q, erwartet 380 (Rechnung)", doc.Document.TypeCode)
	}
	if got := doc.IssueDate(); got != inv.Date {
		t.Errorf("Belegdatum = %q, erwartet %q", got, inv.Date)
	}
	if got := doc.DeliveryDate(); got != inv.ServiceDateTo {
		t.Errorf("Leistungsdatum = %q, erwartet %q", got, inv.ServiceDateTo)
	}
	if got := doc.DueDate(); got != inv.DueDate {
		t.Errorf("Fälligkeit = %q, erwartet %q", got, inv.DueDate)
	}
	if got := doc.Currency(); got != "EUR" {
		t.Errorf("Währung = %q", got)
	}

	if got := doc.Transaction.Agreement.Seller.Name; got != seller.CompanyName {
		t.Errorf("Lieferant = %q, erwartet %q", got, seller.CompanyName)
	}
	if got := doc.Transaction.Agreement.Seller.VatID(); got != seller.VatID {
		t.Errorf("USt-IdNr. des Lieferanten = %q, erwartet %q", got, seller.VatID)
	}
	if got := doc.Transaction.Agreement.Buyer.VatID(); got != buyer.VatID {
		t.Errorf("USt-IdNr. des Empfängers = %q, erwartet %q", got, buyer.VatID)
	}

	total, err := doc.GrandTotal()
	if err != nil {
		t.Fatalf("Gesamtbetrag: %v", err)
	}
	if total != inv.GrossAmount {
		t.Errorf("Gesamtbetrag = %s, erwartet %s", total, inv.GrossAmount)
	}

	if len(doc.Transaction.Settlement.Taxes) != 1 {
		t.Fatalf("erwartet eine Steuergruppe, erhalten %d", len(doc.Transaction.Settlement.Taxes))
	}
	tax := doc.Transaction.Settlement.Taxes[0]
	if tax.CategoryCode != "S" {
		t.Errorf("Kategoriecode = %q, erwartet S", tax.CategoryCode)
	}
	rate, err := TaxRateFromPercent(tax.RatePercent)
	if err != nil {
		t.Fatalf("Steuersatz: %v", err)
	}
	if rate != domain.TaxRateStandard {
		t.Errorf("Steuersatz = %s, erwartet 19 %%", rate.Label())
	}

	if len(doc.Transaction.Lines) != 1 {
		t.Fatalf("erwartet eine Position, erhalten %d", len(doc.Transaction.Lines))
	}
	if got := doc.Transaction.Lines[0].Product.Name; got != inv.Items[0].Description {
		t.Errorf("Positionstext = %q, erwartet %q", got, inv.Items[0].Description)
	}
	net, err := domain.ParseCents(doc.Transaction.Lines[0].Settlement.Summation.LineTotal)
	if err != nil || net != inv.Items[0].TotalNet() {
		t.Errorf("Positionsbetrag = %q, erwartet %s", doc.Transaction.Lines[0].Settlement.Summation.LineTotal, inv.Items[0].TotalNet())
	}
}

// Der Kategoriecode steht aus Sicht des Ausstellers im Dokument. Auf der
// Eingangsseite muss er gedreht werden — sonst wird der halbe Vorgang gebucht.
func TestIncomingTreatmentInvertsTheIssuersPerspective(t *testing.T) {
	cases := map[string]domain.TaxTreatment{
		"S":  domain.TaxTreatmentDomestic,
		"AE": domain.TaxTreatmentReverseCharge,
		"K":  domain.TaxTreatmentIntraCommunityAcquisition,
		"Z":  domain.TaxTreatmentZeroRated,
		"E":  domain.TaxTreatmentExempt,
		"O":  domain.TaxTreatmentNotTaxable,
	}
	for code, want := range cases {
		got, err := IncomingTaxTreatment(code)
		if err != nil {
			t.Errorf("%s: %v", code, err)
			continue
		}
		if got != want {
			t.Errorf("%s → %q, erwartet %q", code, got, want)
		}
	}

	// "K" ist der Fall, an dem sich zeigt, ob gedreht wurde: beim Lieferanten
	// eine steuerfreie Lieferung, bei uns ein steuerpflichtiger Erwerb.
	if got, _ := IncomingTaxTreatment("K"); got == domain.TaxTreatmentIntraCommunitySupply {
		t.Error("der Code wurde ungedreht übernommen — eine i. g. Lieferung des Lieferanten ist bei uns ein Erwerb")
	}

	// Was Buchfink nicht ehrlich abbilden kann, wird benannt statt geraten.
	for _, code := range []string{"G", "", "XX"} {
		if _, err := IncomingTaxTreatment(code); err == nil {
			t.Errorf("der Code %q hätte abgelehnt werden müssen", code)
		}
	}
}

func TestTaxRateFromPercent(t *testing.T) {
	cases := map[string]domain.TaxRate{
		"19.00": domain.TaxRateStandard,
		"7.00":  domain.TaxRateReduced,
		"0.00":  domain.TaxRateNone,
		"":      domain.TaxRateNone,
	}
	for input, want := range cases {
		got, err := TaxRateFromPercent(input)
		if err != nil {
			t.Errorf("%q: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("%q → %s, erwartet %s", input, got.Label(), want.Label())
		}
	}
	for _, input := range []string{"21.00", "kaputt"} {
		if _, err := TaxRateFromPercent(input); err == nil {
			t.Errorf("der Steuersatz %q hätte abgelehnt werden müssen", input)
		}
	}
}

// MINIMUM und BASIC WL enthalten keine vollständige Rechnung und sind keine
// E-Rechnung im Sinne des Gesetzes.
func TestUnusableProfilesAreRejected(t *testing.T) {
	for _, profile := range []string{
		"urn:factur-x.eu:1p0:minimum",
		"urn:factur-x.eu:1p0:basicwl",
		"URN:FACTUR-X.EU:1P0:MINIMUM",
	} {
		doc := &CIIInvoice{}
		doc.Context.Guideline.ID = profile
		doc.Document.ID = "RE-1"
		if err := doc.EnsureUsableProfile(); err == nil {
			t.Errorf("das Profil %q hätte abgelehnt werden müssen", profile)
		}
	}

	doc := &CIIInvoice{}
	doc.Context.Guideline.ID = "urn:cen.eu:en16931:2017"
	if err := doc.EnsureUsableProfile(); err != nil {
		t.Errorf("EN 16931 muss zulässig sein: %v", err)
	}
}

func TestParseRejectsNonInvoiceXML(t *testing.T) {
	for _, data := range []string{
		`<?xml version="1.0"?><foo/>`,
		`kein XML`,
		``,
	} {
		if _, err := ParseCII([]byte(data)); err == nil {
			t.Errorf("%q hätte abgelehnt werden müssen", data)
		}
	}
}
