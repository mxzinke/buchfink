package invoice

import (
	"bytes"
	"fmt"
	"html"
	"time"

	"github.com/buchfink/buchfink/internal/models"
)

// GenerateZUGFeRDXML creates a valid Factur-X / ZUGFeRD 2.2 (EN 16931) XML invoice string.
func GenerateZUGFeRDXML(inv *models.Invoice, seller *models.CompanySettings, buyer *models.Contact) (string, error) {
	issueDate := time.Now().Format("20060102")
	if inv.Date != "" {
		if t, err := time.Parse("2006-01-02", inv.Date); err == nil {
			issueDate = t.Format("20060102")
		}
	}

	var lineItemsBuf bytes.Buffer
	for i, item := range inv.Items {
		lineItemsBuf.WriteString(fmt.Sprintf(`
		<ram:IncludedSupplyChainTradeLineItem>
			<ram:AssociatedDocumentLineDocument>
				<ram:LineID>%d</ram:LineID>
			</ram:AssociatedDocumentLineDocument>
			<ram:SpecifiedTradeProduct>
				<ram:Name>%s</ram:Name>
			</ram:SpecifiedTradeProduct>
			<ram:SpecifiedLineTradeAgreement>
				<ram:NetPriceProductTradePrice>
					<ram:ChargeAmount currencyID="%s">%.2f</ram:ChargeAmount>
				</ram:NetPriceProductTradePrice>
			</ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery>
				<ram:BilledQuantity unitCode="C62">%.2f</ram:BilledQuantity>
			</ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:CategoryCode>S</ram:CategoryCode>
					<ram:RateApplicablePercent>%.2f</ram:RateApplicablePercent>
				</ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation>
					<ram:LineTotalAmount currencyID="%s">%.2f</ram:LineTotalAmount>
				</ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>`,
			i+1,
			html.EscapeString(item.Description),
			inv.Currency,
			item.UnitPrice,
			item.Quantity,
			item.TaxRate*100,
			inv.Currency,
			item.TotalNet,
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
	<rsm:SupplyChainTradeTransaction>
		%s
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty>
				<ram:Name>%s</ram:Name>
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">%s</ram:ID>
				</ram:SpecifiedTaxRegistration>
			</ram:SellerTradeParty>
			<ram:BuyerTradeParty>
				<ram:Name>%s</ram:Name>
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">%s</ram:ID>
				</ram:SpecifiedTaxRegistration>
			</ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>%s</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount currencyID="%s">%.2f</ram:LineTotalAmount>
				<ram:TaxBasisTotalAmount currencyID="%s">%.2f</ram:TaxBasisTotalAmount>
				<ram:TaxTotalAmount currencyID="%s">%.2f</ram:TaxTotalAmount>
				<ram:GrandTotalAmount currencyID="%s">%.2f</ram:GrandTotalAmount>
				<ram:DuePayableAmount currencyID="%s">%.2f</ram:DuePayableAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`,
		html.EscapeString(inv.InvoiceNumber),
		issueDate,
		lineItemsBuf.String(),
		html.EscapeString(seller.CompanyName),
		html.EscapeString(seller.VatID),
		html.EscapeString(buyer.Name),
		html.EscapeString(buyer.VatID),
		inv.Currency,
		inv.Currency, inv.NetAmount,
		inv.Currency, inv.NetAmount,
		inv.Currency, inv.TaxAmount,
		inv.Currency, inv.GrossAmount,
		inv.Currency, inv.GrossAmount,
	)

	return xmlContent, nil
}

// GenerateTypstTemplate generates Typst markup source code for the invoice.
func GenerateTypstTemplate(inv *models.Invoice, seller *models.CompanySettings, buyer *models.Contact) string {
	return fmt.Sprintf(`#set page(paper: "a4", margin: (x: 2cm, y: 2.5cm))
#set text(font: "Manrope", size: 10pt, fill: rgb("#1c1917"))

#grid(
  columns: (1fr, 1fr),
  [
    #text(size: 8pt, fill: rgb("#78716c"))[%s · %s · %s]\
    #v(0.5cm)
    *#text(size: 11pt)[%s]*\
    %s\
    %s
  ],
  [
    #align(right)[
      #text(size: 16pt, weight: "bold", fill: rgb("#d97706"))[RECHNUNG]\
      #v(0.2cm)
      #text(size: 10pt, weight: "bold")[Nr. %s]\
      Datum: %s\
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
  // Items will be rendered here
  [1], [Software-Entwicklungsleistungen], [1.0], [%.2f €], [%.2f €]
)

#v(0.5cm)
#align(right)[
  #block(width: 60%%)[
    #grid(
      columns: (1fr, auto),
      row-gutter: 0.3cm,
      [Nettobetrag:], [%.2f €],
      [zzgl. 19%% USt:], [%.2f €],
      [*Gesamtbetrag:*], [*#text(size: 12pt, fill: rgb("#d97706"))[%.2f €]*]
    )
  ]
]

#v(2cm)
#line(length: 100%%, stroke: 0.5pt + rgb("#e7e5e4"))
#text(size: 8pt, fill: rgb("#78716c"))[
  Bankverbindung: %s · IBAN: %s · BIC: %s\
  Steuernummer: %s · USt-IdNr.: %s
]
`,
		seller.CompanyName, seller.Street, seller.ZipCity,
		buyer.Name, buyer.Address, buyer.VatID,
		inv.InvoiceNumber, inv.Date, inv.DueDate,
		inv.NetAmount, inv.NetAmount,
		inv.NetAmount, inv.TaxAmount, inv.GrossAmount,
		seller.BankName, seller.IBAN, seller.BIC,
		seller.TaxNumber, seller.VatID,
	)
}
