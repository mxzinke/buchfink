package invoice

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// vatCategoryCode maps a Steuerfall to the EN 16931 / UNTDID 5305 category code
// a ZUGFeRD invoice must carry. Sending "S" for an exempt intra-community supply
// makes the document formally wrong even though the amounts add up.
func vatCategoryCode(t domain.TaxTreatment) string {
	switch t {
	case domain.TaxTreatmentIntraCommunitySupply:
		return "K" // Innergemeinschaftliche Lieferung
	case domain.TaxTreatmentExport:
		return "G" // Ausfuhr, steuerfrei
	case domain.TaxTreatmentReverseChargeSupply:
		return "AE" // Steuerschuldnerschaft des Leistungsempfängers
	case domain.TaxTreatmentExempt:
		return "E" // Steuerbefreit
	case domain.TaxTreatmentNotTaxable:
		return "O" // Nicht steuerbar
	default:
		return "S" // Regelbesteuerung
	}
}

// exemptionReason returns the legal reference EN 16931 requires whenever the
// category code is not "S".
func exemptionReason(t domain.TaxTreatment) string {
	switch t {
	case domain.TaxTreatmentIntraCommunitySupply:
		return "Steuerfreie innergemeinschaftliche Lieferung nach § 4 Nr. 1 Buchst. b i. V. m. § 6a UStG"
	case domain.TaxTreatmentExport:
		return "Steuerfreie Ausfuhrlieferung nach § 4 Nr. 1 Buchst. a i. V. m. § 6 UStG"
	case domain.TaxTreatmentReverseChargeSupply:
		return "Steuerschuldnerschaft des Leistungsempfängers nach § 13b UStG"
	case domain.TaxTreatmentExempt:
		return "Steuerfreier Umsatz nach § 4 UStG"
	case domain.TaxTreatmentNotTaxable:
		return "Nicht steuerbarer Umsatz"
	default:
		return ""
	}
}

func compactDate(iso string) string {
	if t, err := time.Parse("2006-01-02", iso); err == nil {
		return t.Format("20060102")
	}
	return time.Now().Format("20060102")
}

// typstDate renders an ISO date as a Typst datetime call, falling back to the
// current day only when the invoice carries no date at all — which the invoice
// validation prevents.
func typstDate(iso string) string {
	var y, m, d int
	if n, err := fmt.Sscanf(iso, "%4d-%2d-%2d", &y, &m, &d); err != nil || n != 3 {
		return "auto"
	}
	return fmt.Sprintf("datetime(year: %d, month: %d, day: %d)", y, m, d)
}

func quantityString(milli int64) string {
	neg := ""
	if milli < 0 {
		neg = "-"
		milli = -milli
	}
	return fmt.Sprintf("%s%d.%03d", neg, milli/1000, milli%1000)
}

func ratePercent(r domain.TaxRate) string {
	return fmt.Sprintf("%d.%02d", int(r)/100, int(r)%100)
}

// GenerateZUGFeRDXML creates a Factur-X / ZUGFeRD 2.2 (EN 16931) XML invoice.
func GenerateZUGFeRDXML(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact) (string, error) {
	if err := inv.Validate(); err != nil {
		return "", err
	}

	category := vatCategoryCode(inv.TaxTreatment)
	reason := exemptionReason(inv.TaxTreatment)

	var lines bytes.Buffer
	for i := range inv.Items {
		item := &inv.Items[i]
		rate := item.TaxRate
		if inv.TaxTreatment != domain.TaxTreatmentDomestic {
			rate = domain.TaxRateNone
		}
		lines.WriteString(fmt.Sprintf(`
		<ram:IncludedSupplyChainTradeLineItem>
			<ram:AssociatedDocumentLineDocument>
				<ram:LineID>%d</ram:LineID>
			</ram:AssociatedDocumentLineDocument>
			<ram:SpecifiedTradeProduct>
				<ram:Name>%s</ram:Name>
			</ram:SpecifiedTradeProduct>
			<ram:SpecifiedLineTradeAgreement>
				<ram:NetPriceProductTradePrice>
					<ram:ChargeAmount>%s</ram:ChargeAmount>
				</ram:NetPriceProductTradePrice>
			</ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery>
				<ram:BilledQuantity unitCode="C62">%s</ram:BilledQuantity>
			</ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>%s</ram:CategoryCode>
					<ram:RateApplicablePercent>%s</ram:RateApplicablePercent>
				</ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation>
					<ram:LineTotalAmount>%s</ram:LineTotalAmount>
				</ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>`,
			item.Position,
			html.EscapeString(item.Description),
			item.UnitPrice.Decimal(),
			quantityString(item.QuantityMilli),
			category,
			ratePercent(rate),
			item.TotalNet().Decimal(),
		))
	}

	// One ApplicableTradeTax block per rate group, as EN 16931 requires.
	var taxBlocks bytes.Buffer
	for _, g := range inv.TaxGroups() {
		rate := g.Rate
		if inv.TaxTreatment != domain.TaxTreatmentDomestic {
			rate = domain.TaxRateNone
		}
		var reasonTag string
		if reason != "" {
			reasonTag = fmt.Sprintf("\n\t\t\t\t<ram:ExemptionReason>%s</ram:ExemptionReason>", html.EscapeString(reason))
		}
		taxBlocks.WriteString(fmt.Sprintf(`
			<ram:ApplicableTradeTax>
				<ram:CalculatedAmount>%s</ram:CalculatedAmount>
				<ram:TypeCode>VAT</ram:TypeCode>%s
				<ram:BasisAmount>%s</ram:BasisAmount>
				<ram:CategoryCode>%s</ram:CategoryCode>
				<ram:RateApplicablePercent>%s</ram:RateApplicablePercent>
			</ram:ApplicableTradeTax>`,
			g.Tax.Decimal(), reasonTag, g.Net.Decimal(), category, ratePercent(rate),
		))
	}

	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
    xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
    xmlns:qdt="urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
    xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocumentContext>
		<ram:GuidelineSpecifiedDocumentContextParameter>
			<ram:ID>urn:cen.eu:en16931:2017</ram:ID>
		</ram:GuidelineSpecifiedDocumentContextParameter>
	</rsm:ExchangedDocumentContext>
	<rsm:ExchangedDocument>
		<ram:ID>%s</ram:ID>
		<ram:TypeCode>380</ram:TypeCode>
		<ram:IssueDateTime>
			<udt:DateTimeString format="102">%s</udt:DateTimeString>
		</ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>%s
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty>
				<ram:Name>%s</ram:Name>
				<ram:PostalTradeAddress>
					<ram:LineOne>%s</ram:LineOne>
					<ram:CityName>%s</ram:CityName>
					<ram:CountryID>DE</ram:CountryID>
				</ram:PostalTradeAddress>
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">%s</ram:ID>
				</ram:SpecifiedTaxRegistration>
			</ram:SellerTradeParty>
			<ram:BuyerTradeParty>
				<ram:Name>%s</ram:Name>
				<ram:PostalTradeAddress>
					<ram:LineOne>%s</ram:LineOne>
					<ram:CountryID>%s</ram:CountryID>
				</ram:PostalTradeAddress>
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">%s</ram:ID>
				</ram:SpecifiedTaxRegistration>
			</ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeDelivery>
			<ram:ActualDeliverySupplyChainEvent>
				<ram:OccurrenceDateTime>
					<udt:DateTimeString format="102">%s</udt:DateTimeString>
				</ram:OccurrenceDateTime>
			</ram:ActualDeliverySupplyChainEvent>
		</ram:ApplicableHeaderTradeDelivery>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>%s</ram:InvoiceCurrencyCode>%s
			<ram:SpecifiedTradePaymentTerms>
				<ram:DueDateDateTime>
					<udt:DateTimeString format="102">%s</udt:DateTimeString>
				</ram:DueDateDateTime>
			</ram:SpecifiedTradePaymentTerms>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>%s</ram:LineTotalAmount>
				<ram:TaxBasisTotalAmount>%s</ram:TaxBasisTotalAmount>
				<ram:TaxTotalAmount currencyID="%s">%s</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>%s</ram:GrandTotalAmount>
				<ram:DuePayableAmount>%s</ram:DuePayableAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`,
		html.EscapeString(inv.InvoiceNumber),
		compactDate(inv.Date),
		lines.String(),
		html.EscapeString(seller.CompanyName),
		html.EscapeString(seller.Street),
		html.EscapeString(seller.ZipCity),
		html.EscapeString(seller.VatID),
		html.EscapeString(buyer.Name),
		html.EscapeString(strings.ReplaceAll(buyer.Address, "\n", ", ")),
		html.EscapeString(countryOrDE(buyer.CountryCode)),
		html.EscapeString(buyer.VatID),
		compactDate(inv.ServiceDateTo),
		inv.Currency,
		taxBlocks.String(),
		compactDate(inv.DueDate),
		inv.NetAmount.Decimal(),
		inv.NetAmount.Decimal(),
		inv.Currency, inv.TaxAmount.Decimal(),
		inv.GrossAmount.Decimal(),
		inv.GrossAmount.Decimal(),
	)

	// Die eigene Rechnung wird gegen die eigene Prüfung gehalten. Eine Rechnung
	// ohne vollständige Empfängeranschrift ist nach § 14 Abs. 4 Nr. 1 UStG keine
	// ordnungsmäßige Rechnung — der Empfänger verlöre den Vorsteuerabzug, und er
	// merkte es erst bei der Prüfung. Lieber jetzt eine Meldung an den Aussteller.
	if doc, err := ParseCII([]byte(xmlContent)); err == nil {
		if result := ValidateEN16931(doc); !result.Valid() {
			return "", fmt.Errorf(
				"die Rechnung erfüllt EN 16931 noch nicht: %s. Bitte die fehlenden Stammdaten ergänzen",
				strings.Join(errorMessages(result), "; "))
		}
	}

	return xmlContent, nil
}

func errorMessages(result ValidationResult) []string {
	var out []string
	for _, f := range result.Findings {
		if f.Severity == SeverityError {
			out = append(out, f.Message)
		}
	}
	return out
}

func countryOrDE(code string) string {
	if code == "" {
		return "DE"
	}
	return code
}

// GenerateTypstTemplate renders the invoice as Typst markup.
func GenerateTypstTemplate(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact) string {
	var rows strings.Builder
	for i := range inv.Items {
		item := &inv.Items[i]
		rows.WriteString(fmt.Sprintf("  [%d], [%s], [%s %s], [%s €], [%s €],\n",
			item.Position,
			typstEscape(item.Description),
			strings.TrimSuffix(strings.TrimRight(quantityString(item.QuantityMilli), "0"), "."),
			typstEscape(item.Unit),
			item.UnitPrice.String(),
			item.TotalNet().String(),
		))
	}

	var totals strings.Builder
	totals.WriteString("      [Nettobetrag:], [" + inv.NetAmount.String() + " €],\n")
	if inv.TaxTreatment == domain.TaxTreatmentDomestic {
		for _, g := range inv.TaxGroups() {
			if g.Tax == 0 {
				continue
			}
			totals.WriteString(fmt.Sprintf("      [zzgl. %s USt:], [%s €],\n", g.Rate.Label(), g.Tax.String()))
		}
	}
	totals.WriteString(`      [*Gesamtbetrag:*], [*#text(size: 12pt, fill: rgb("#d97706"))[` + inv.GrossAmount.String() + ` €]*],` + "\n")

	// Every non-standard Steuerfall needs its legal reference printed on the
	// invoice; § 14 Abs. 4 Nr. 8 UStG requires the reason for an exemption.
	var note string
	if reason := exemptionReason(inv.TaxTreatment); reason != "" {
		note = fmt.Sprintf(`
#v(0.3cm)
#text(size: 8.5pt, fill: rgb("#78716c"), style: "italic")[%s]`, typstEscape(reason))
	}

	// PDF/A demands a document date, and the embedded file needs both a mime type
	// and a description — Typst refuses to emit a-3b without them, which is
	// exactly the check ZUGFeRD needs. The relationship must be "alternative":
	// the PDF and the XML are two renderings of one invoice, and for the BASIC
	// and EN-16931 profiles anything else is not legally valid in Germany.
	docDate := typstDate(inv.Date)

	return fmt.Sprintf(`#set document(title: "Rechnung %s", author: %q, date: %s)
#pdf.embed(
  "factur-x.xml",
  bytes(sys.inputs.zugferd_xml),
  relationship: "alternative",
  mime-type: "application/xml",
  description: "Factur-X / ZUGFeRD invoice data (EN 16931)",
)
#set page(paper: "a4", margin: (x: 2cm, y: 2.5cm))
#set text(font: "Manrope", size: 10pt, fill: rgb("#1c1917"))

#grid(
  columns: (1fr, 1fr),
  [
    #text(size: 8pt, fill: rgb("#78716c"))[%s · %s · %s]\
    #v(0.5cm)
    *#text(size: 11pt)[%s]*\
    %s
  ],
  [
    #align(right)[
      #text(size: 16pt, weight: "bold", fill: rgb("#d97706"))[RECHNUNG]\
      #v(0.2cm)
      #text(size: 10pt, weight: "bold")[Nr. %s]\
      Rechnungsdatum: %s\
      Leistungszeitraum: %s – %s\
      Fällig bis: %s
    ]
  ]
)

#v(1.5cm)

#table(
  columns: (auto, 1fr, auto, auto, auto),
  align: (center, left, right, right, right),
  stroke: (x, y) => if y == 0 { (bottom: 1.5pt + rgb("#d97706")) } else { (bottom: 0.5pt + rgb("#e7e5e4")) },
  table.header([*Pos*], [*Bezeichnung*], [*Menge*], [*Einzelpreis*], [*Gesamt*]),
%s)

#v(0.5cm)
#align(right)[
  #block(width: 60%%)[
    #grid(
      columns: (1fr, auto),
      row-gutter: 0.3cm,
%s    )
  ]
]%s

#v(2cm)
#line(length: 100%%, stroke: 0.5pt + rgb("#e7e5e4"))
#text(size: 8pt, fill: rgb("#78716c"))[
  Bankverbindung: %s · IBAN: %s · BIC: %s\
  Steuernummer: %s · USt-IdNr.: %s
]
`,
		typstEscape(inv.InvoiceNumber),
		// %q inside a Typst string literal: the markup escaping of typstEscape
		// would land as visible backslashes in the PDF metadata.
		seller.CompanyName,
		docDate,
		typstEscape(seller.CompanyName), typstEscape(seller.Street), typstEscape(seller.ZipCity),
		typstEscape(buyer.Name), typstEscape(buyer.Address),
		typstEscape(inv.InvoiceNumber), inv.Date, inv.ServiceDateFrom, inv.ServiceDateTo, inv.DueDate,
		rows.String(),
		totals.String(),
		note,
		typstEscape(seller.BankName), typstEscape(seller.IBAN), typstEscape(seller.BIC),
		typstEscape(seller.TaxNumber), typstEscape(seller.VatID),
	)
}

// typstEscape neutralises the Typst markup characters so that a customer name
// containing a bracket or a hash cannot break the document.
func typstEscape(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "[", "\\[", "]", "\\]",
		"#", "\\#", "$", "\\$", "*", "\\*", "_", "\\_", "@", "\\@",
		"\n", " \\\n",
	)
	return replacer.Replace(s)
}
