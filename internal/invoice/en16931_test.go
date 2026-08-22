package invoice

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func findingsFor(result ValidationResult, rule string) []ValidationFinding {
	var out []ValidationFinding
	for _, f := range result.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func mustParse(t *testing.T, xml string) *CIIInvoice {
	t.Helper()
	doc, err := ParseCII([]byte(xml))
	if err != nil {
		t.Fatalf("XML lesen: %v", err)
	}
	return doc
}

// Was Buchfink selbst erzeugt, muss die eigene Prüfung bestehen. Täte es das
// nicht, wäre entweder der Generator falsch oder die Prüfung — und man wüsste
// nicht, welches von beidem.
func TestGeneratedInvoicePassesValidation(t *testing.T) {
	cases := []struct {
		name      string
		treatment domain.TaxTreatment
		rate      domain.TaxRate
		buyerVat  string
	}{
		{"Inland 19 %", domain.TaxTreatmentDomestic, domain.TaxRateStandard, "DE987654321"},
		{"Inland 7 %", domain.TaxTreatmentDomestic, domain.TaxRateReduced, "DE987654321"},
		{"i. g. Lieferung", domain.TaxTreatmentIntraCommunitySupply, domain.TaxRateNone, "NL123456789B01"},
		{"§ 13b", domain.TaxTreatmentReverseChargeSupply, domain.TaxRateNone, "DE987654321"},
		{"steuerfrei", domain.TaxTreatmentExempt, domain.TaxRateNone, "DE987654321"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv, seller, buyer := sampleInvoice()
			inv.TaxTreatment = tc.treatment
			inv.Items[0].TaxRate = tc.rate
			buyer.VatID = tc.buyerVat
			if tc.treatment == domain.TaxTreatmentIntraCommunitySupply {
				buyer.CountryCode = "NL"
			}
			inv.Recalculate()

			xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
			if err != nil {
				t.Fatalf("XML erzeugen: %v", err)
			}

			result := ValidateEN16931(mustParse(t, xml))
			for _, f := range result.Findings {
				if f.Severity == SeverityError {
					t.Errorf("%s: %s", f.Rule, f.Message)
				}
			}
			if !result.Valid() {
				t.Errorf("die eigene Rechnung besteht die eigene Prüfung nicht (%d Fehler)", result.ErrorCount())
			}
			if result.Coverage != CoveragePartial {
				t.Errorf("Prüfumfang = %q, erwartet %q", result.Coverage, CoveragePartial)
			}
			if result.Version != EN16931RulesetVersion {
				t.Errorf("Regelwerksversion = %q", result.Version)
			}
		})
	}
}

// Fehlende Pflichtangaben werden mit der Regel benannt, gegen die sie verstoßen.
// Ohne die Regelnummer kann niemand nachschlagen, ob Buchfink recht hat.
func TestMissingMandatoryContentIsReportedWithItsRule(t *testing.T) {
	const skeleton = `<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
    xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
    xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument><ram:ID>RE-1</ram:ID></rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction/>
</rsm:CrossIndustryInvoice>`

	result := ValidateEN16931(mustParse(t, skeleton))

	// Jede dieser Regeln muss anschlagen — das ist der Kern von EN 16931.
	for _, rule := range []string{
		"BR-01", // Spezifikationskennung
		"BR-03", // Rechnungsdatum
		"BR-04", // Rechnungstyp
		"BR-05", // Währung
		"BR-06", // Name des Verkäufers
		"BR-07", // Name des Erwerbers
		"BR-08", // Anschrift des Verkäufers
		"BR-09", // Land des Verkäufers
		"BR-10", // Anschrift des Erwerbers
		"BR-11", // Land des Erwerbers
		"BR-12", // Summe der Positionen
		"BR-13", // Nettogesamtbetrag
		"BR-14", // Bruttogesamtbetrag
		"BR-15", // fälliger Betrag
		"BR-16", // mindestens eine Position
		"BR-45", // Steueraufschlüsselung
	} {
		if len(findingsFor(result, rule)) == 0 {
			t.Errorf("%s hätte anschlagen müssen", rule)
		}
	}

	// Die vorhandene Rechnungsnummer darf nicht bemängelt werden.
	if len(findingsFor(result, "BR-02")) != 0 {
		t.Error("BR-02 darf bei vorhandener Rechnungsnummer nicht anschlagen")
	}
}

// Ein Dokument kann formal vollständig sein und trotzdem nicht aufgehen. Diese
// Regeln sind die gefährlichsten, denn die Beträge sind das, was gebucht wird.
func TestArithmeticRules(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	buyer.VatID = "DE987654321"
	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("XML erzeugen: %v", err)
	}

	t.Run("BR-CO-15 Brutto ist Netto plus Steuer", func(t *testing.T) {
		broken := strings.Replace(xml,
			"<ram:GrandTotalAmount>2380.00</ram:GrandTotalAmount>",
			"<ram:GrandTotalAmount>2400.00</ram:GrandTotalAmount>", 1)
		result := ValidateEN16931(mustParse(t, broken))
		if len(findingsFor(result, "BR-CO-15")) == 0 {
			t.Error("ein falscher Bruttobetrag muss auffallen")
		}
	})

	t.Run("BR-CO-17 Steuer folgt aus Grundlage und Satz", func(t *testing.T) {
		broken := strings.Replace(xml,
			"<ram:CalculatedAmount>380.00</ram:CalculatedAmount>",
			"<ram:CalculatedAmount>190.00</ram:CalculatedAmount>", 1)
		result := ValidateEN16931(mustParse(t, broken))
		if len(findingsFor(result, "BR-CO-17")) == 0 {
			t.Error("ein Steuerbetrag, der nicht zum Satz passt, muss auffallen")
		}
	})

	t.Run("BR-CO-10 Summe der Positionen", func(t *testing.T) {
		broken := strings.Replace(xml,
			"<ram:LineTotalAmount>2000.00</ram:LineTotalAmount>\n\t\t\t\t</ram:SpecifiedTradeSettlementLineMonetarySummation>",
			"<ram:LineTotalAmount>1500.00</ram:LineTotalAmount>\n\t\t\t\t</ram:SpecifiedTradeSettlementLineMonetarySummation>", 1)
		result := ValidateEN16931(mustParse(t, broken))
		if len(findingsFor(result, "BR-CO-10")) == 0 {
			t.Error("eine Positionssumme, die nicht zu den Positionen passt, muss auffallen")
		}
	})
}

// Ein Dokument, das "Steuerschuld beim Empfänger" sagt und 19 % ausweist,
// widerspricht sich. Wer daraus bucht, hat die Steuer zweimal.
func TestContradictoryCategoryAndRateIsRejected(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	inv.TaxTreatment = domain.TaxTreatmentReverseChargeSupply
	inv.Items[0].TaxRate = domain.TaxRateNone
	buyer.VatID = "DE987654321"
	inv.Recalculate()

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("XML erzeugen: %v", err)
	}

	broken := strings.ReplaceAll(xml,
		"<ram:RateApplicablePercent>0.00</ram:RateApplicablePercent>",
		"<ram:RateApplicablePercent>19.00</ram:RateApplicablePercent>")
	result := ValidateEN16931(mustParse(t, broken))
	if len(findingsFor(result, "BR-AE-05")) == 0 {
		t.Error("§ 13b mit 19 % muss auffallen")
	}
}

// Fehlt der Befreiungsgrund, ist die Rechnung nach § 14 Abs. 4 Nr. 8 UStG
// unvollständig — und EN 16931 verlangt ihn ebenso.
func TestMissingExemptionReasonIsReported(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	inv.TaxTreatment = domain.TaxTreatmentIntraCommunitySupply
	inv.Items[0].TaxRate = domain.TaxRateNone
	buyer.CountryCode = "NL"
	buyer.VatID = "NL123456789B01"
	inv.Recalculate()

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("XML erzeugen: %v", err)
	}

	start := strings.Index(xml, "<ram:ExemptionReason>")
	end := strings.Index(xml, "</ram:ExemptionReason>")
	if start < 0 || end < 0 {
		t.Fatal("die erzeugte Rechnung enthält keinen Befreiungsgrund")
	}
	broken := xml[:start] + xml[end+len("</ram:ExemptionReason>"):]

	result := ValidateEN16931(mustParse(t, broken))
	if len(findingsFor(result, "BR-K-10")) == 0 {
		t.Error("ein fehlender Befreiungsgrund bei i. g. Lieferung muss auffallen")
	}
}

// Codelisten: was nicht in der Norm steht, wird nicht durchgewunken.
func TestCodeListsAreChecked(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	buyer.VatID = "DE987654321"
	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("XML erzeugen: %v", err)
	}

	t.Run("Währung", func(t *testing.T) {
		broken := strings.Replace(xml,
			"<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>",
			"<ram:InvoiceCurrencyCode>XYZ</ram:InvoiceCurrencyCode>", 1)
		if len(findingsFor(ValidateEN16931(mustParse(t, broken)), "BR-CL-03")) == 0 {
			t.Error("ein unbekannter Währungscode muss auffallen")
		}
	})

	t.Run("Land", func(t *testing.T) {
		broken := strings.Replace(xml,
			"<ram:CountryID>DE</ram:CountryID>",
			"<ram:CountryID>XX</ram:CountryID>", 1)
		if len(findingsFor(ValidateEN16931(mustParse(t, broken)), "BR-CL-10")) == 0 {
			t.Error("ein unbekanntes Länderkennzeichen muss auffallen")
		}
	})

	t.Run("Steuerkategorie", func(t *testing.T) {
		broken := strings.ReplaceAll(xml,
			"<ram:CategoryCode>S</ram:CategoryCode>",
			"<ram:CategoryCode>Q</ram:CategoryCode>")
		if len(findingsFor(ValidateEN16931(mustParse(t, broken)), "BR-CL-17")) == 0 {
			t.Error("ein unbekannter Steuerkategoriecode muss auffallen")
		}
	})
}

// Der Prüfumfang wird benannt, nicht behauptet. Es gibt bewusst keinen Wert, der
// "vollständig nach EN 16931 geprüft" bedeutet.
func TestCoverageIsHonestAboutItsLimits(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	buyer.VatID = "DE987654321"
	xml, _ := GenerateZUGFeRDXML(inv, seller, buyer)

	result := ValidateEN16931(mustParse(t, xml))
	if result.Coverage != CoveragePartial {
		t.Errorf("Prüfumfang = %q — es darf keinen Wert für Vollständigkeit geben", result.Coverage)
	}
	if len(ValidationRules()) < 20 {
		t.Error("die Liste der geprüften Regeln gehört zum Ergebnis")
	}
	if result.Ruleset != EN16931RulesetID {
		t.Errorf("Regelwerkskennung = %q", result.Ruleset)
	}
}
