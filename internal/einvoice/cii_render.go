package einvoice

import (
	"encoding/base64"
	"fmt"
	"html"
	"strings"
)

// RenderCII writes an invoice as a UN/CEFACT Cross Industry Invoice.
//
// The element order is not decoration: CII is defined by a sequence, and a
// document whose elements come in a different order fails schema validation
// before any business rule is reached. The order below follows the XSD, which
// is why the writer is explicit rather than derived from the struct — the
// sequence has to be visible to whoever changes it next.
func RenderCII(inv *Invoice) ([]byte, error) {
	if inv == nil {
		return nil, fmt.Errorf("keine Rechnung übergeben")
	}
	var w ciiWriter
	w.raw(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	w.open("rsm:CrossIndustryInvoice",
		`xmlns:rsm="`+nsRSM+`"`,
		`xmlns:ram="`+nsRAM+`"`,
		`xmlns:udt="`+nsUDT+`"`,
		`xmlns:qdt="`+nsQDT+`"`)

	w.open("rsm:ExchangedDocumentContext")
	if inv.BusinessProcess != "" {
		w.open("ram:BusinessProcessSpecifiedDocumentContextParameter")
		w.text("ram:ID", inv.BusinessProcess)
		w.close("ram:BusinessProcessSpecifiedDocumentContextParameter")
	}
	w.open("ram:GuidelineSpecifiedDocumentContextParameter")
	w.text("ram:ID", inv.SpecificationID)
	w.close("ram:GuidelineSpecifiedDocumentContextParameter")
	w.close("rsm:ExchangedDocumentContext")

	w.open("rsm:ExchangedDocument")
	w.text("ram:ID", inv.Number)
	w.text("ram:TypeCode", inv.TypeCode)
	w.date("ram:IssueDateTime", inv.IssueDate)
	for _, note := range inv.Notes {
		w.open("ram:IncludedNote")
		w.text("ram:Content", note.Text)
		w.optional("ram:SubjectCode", note.SubjectCode)
		w.close("ram:IncludedNote")
	}
	w.close("rsm:ExchangedDocument")

	w.open("rsm:SupplyChainTradeTransaction")
	for _, line := range inv.Lines {
		w.renderLine(line)
	}
	w.renderAgreement(inv)
	w.renderDelivery(inv)
	w.renderSettlement(inv)
	w.close("rsm:SupplyChainTradeTransaction")

	w.close("rsm:CrossIndustryInvoice")
	return []byte(w.buf.String()), nil
}

func (w *ciiWriter) renderAgreement(inv *Invoice) {
	w.open("ram:ApplicableHeaderTradeAgreement")
	w.optional("ram:BuyerReference", inv.BuyerReference)
	w.party("ram:SellerTradeParty", &inv.Seller)
	w.party("ram:BuyerTradeParty", &inv.Buyer)
	w.party("ram:SellerTaxRepresentativeTradeParty", inv.TaxRepresentative)
	w.docRef("ram:SellerOrderReferencedDocument", inv.SalesOrderReference, "", "")
	w.docRef("ram:BuyerOrderReferencedDocument", inv.OrderReference, "", "")
	w.docRef("ram:ContractReferencedDocument", inv.ContractReference, "", "")

	if inv.TenderReference != "" {
		w.docRef("ram:AdditionalReferencedDocument", inv.TenderReference, ciiRefTender, "")
	}
	if inv.ObjectIdentifier.Present() {
		w.docRef("ram:AdditionalReferencedDocument", inv.ObjectIdentifier.Value,
			ciiRefObject, inv.ObjectIdentifier.Scheme)
	}
	for _, doc := range inv.SupportingDocs {
		w.supportingDoc(doc)
	}
	if inv.ProjectReference != "" {
		w.open("ram:SpecifiedProcuringProject")
		w.text("ram:ID", inv.ProjectReference)
		w.text("ram:Name", inv.ProjectReference)
		w.close("ram:SpecifiedProcuringProject")
	}
	w.close("ram:ApplicableHeaderTradeAgreement")
}

func (w *ciiWriter) renderDelivery(inv *Invoice) {
	w.open("ram:ApplicableHeaderTradeDelivery")
	if d := inv.Delivery; d != nil {
		if d.Name != "" || d.Address != nil || d.LocationID.Present() {
			w.open("ram:ShipToTradeParty")
			if d.LocationID.Present() {
				w.identifier("ram:GlobalID", d.LocationID)
			}
			w.optional("ram:Name", d.Name)
			w.address(d.Address)
			w.close("ram:ShipToTradeParty")
		}
		if d.Date.Present() {
			w.open("ram:ActualDeliverySupplyChainEvent")
			w.date("ram:OccurrenceDateTime", d.Date)
			w.close("ram:ActualDeliverySupplyChainEvent")
		}
	}
	w.docRef("ram:DespatchAdviceReferencedDocument", inv.DespatchAdviceReference, "", "")
	w.docRef("ram:ReceivingAdviceReferencedDocument", inv.ReceivingAdviceReference, "", "")
	w.close("ram:ApplicableHeaderTradeDelivery")
}

func (w *ciiWriter) renderSettlement(inv *Invoice) {
	w.open("ram:ApplicableHeaderTradeSettlement")
	w.optional("ram:CreditorReferenceID", inv.CreditorReference)
	if len(inv.PaymentMeans) > 0 {
		w.optional("ram:PaymentReference", inv.PaymentMeans[0].RemittanceInfo)
	}
	w.optional("ram:TaxCurrencyCode", inv.TaxCurrency)
	w.text("ram:InvoiceCurrencyCode", inv.Currency)
	w.party("ram:PayeeTradeParty", inv.Payee)

	for _, means := range inv.PaymentMeans {
		w.open("ram:SpecifiedTradeSettlementPaymentMeans")
		w.optional("ram:TypeCode", means.TypeCode)
		w.optional("ram:Information", means.TypeText)
		if card := means.Card; card != nil {
			w.open("ram:ApplicableTradeSettlementFinancialCard")
			w.optional("ram:ID", card.PrimaryAccountNumber)
			w.optional("ram:CardholderName", card.HolderName)
			w.close("ram:ApplicableTradeSettlementFinancialCard")
		}
		if debit := means.DirectDebit; debit != nil && debit.DebitedAccount != "" {
			w.open("ram:PayerPartyDebtorFinancialAccount")
			w.text("ram:IBANID", debit.DebitedAccount)
			w.close("ram:PayerPartyDebtorFinancialAccount")
		}
		for _, account := range means.CreditTransfer {
			w.open("ram:PayeePartyCreditorFinancialAccount")
			w.optional("ram:IBANID", account.AccountID)
			w.optional("ram:AccountName", account.AccountName)
			w.close("ram:PayeePartyCreditorFinancialAccount")
			if account.ProviderID != "" {
				w.open("ram:PayeeSpecifiedCreditorFinancialInstitution")
				w.text("ram:BICID", account.ProviderID)
				w.close("ram:PayeeSpecifiedCreditorFinancialInstitution")
			}
		}
		w.close("ram:SpecifiedTradeSettlementPaymentMeans")
	}

	for _, group := range inv.VATBreakdown {
		w.tradeTax("ram:ApplicableTradeTax", group, inv)
	}
	if p := inv.Period; p != nil && (p.Start.Present() || p.End.Present()) {
		w.open("ram:BillingSpecifiedPeriod")
		if p.Start.Present() {
			w.date("ram:StartDateTime", p.Start)
		}
		if p.End.Present() {
			w.date("ram:EndDateTime", p.End)
		}
		w.close("ram:BillingSpecifiedPeriod")
	}
	for _, a := range inv.Allowances {
		w.allowanceCharge(a, false)
	}
	for _, a := range inv.Charges {
		w.allowanceCharge(a, true)
	}

	mandate := ""
	for _, means := range inv.PaymentMeans {
		if means.DirectDebit != nil && means.DirectDebit.MandateReference != "" {
			mandate = means.DirectDebit.MandateReference
		}
	}
	if inv.PaymentTermsNote != "" || inv.DueDate.Present() || mandate != "" {
		w.open("ram:SpecifiedTradePaymentTerms")
		w.optional("ram:Description", inv.PaymentTermsNote)
		if inv.DueDate.Present() {
			w.date("ram:DueDateDateTime", inv.DueDate)
		}
		w.optional("ram:DirectDebitMandateID", mandate)
		w.close("ram:SpecifiedTradePaymentTerms")
	}

	t := inv.Totals
	w.open("ram:SpecifiedTradeSettlementHeaderMonetarySummation")
	w.amount("ram:LineTotalAmount", t.LineTotal)
	w.amount("ram:ChargeTotalAmount", t.ChargeTotal)
	w.amount("ram:AllowanceTotalAmount", t.AllowanceTotal)
	w.amount("ram:TaxBasisTotalAmount", t.TaxBasisTotal)
	if t.TaxTotal.Present() {
		w.amountWithCurrency("ram:TaxTotalAmount", t.TaxTotal, orDefault(t.TaxTotalCurrency, inv.Currency))
	}
	if t.TaxTotalInTaxCurr.Present() {
		w.amountWithCurrency("ram:TaxTotalAmount", t.TaxTotalInTaxCurr,
			orDefault(t.TaxTotalInTaxCurrCurrency, inv.TaxCurrency))
	}
	w.amount("ram:RoundingAmount", t.RoundingAmount)
	w.amount("ram:GrandTotalAmount", t.GrandTotal)
	w.amount("ram:TotalPrepaidAmount", t.PrepaidAmount)
	w.amount("ram:DuePayableAmount", t.DuePayableAmount)
	w.close("ram:SpecifiedTradeSettlementHeaderMonetarySummation")

	for _, ref := range inv.PrecedingInvoices {
		w.open("ram:InvoiceReferencedDocument")
		w.text("ram:IssuerAssignedID", ref.Number)
		if ref.IssueDate.Present() {
			w.open("ram:FormattedIssueDateTime")
			w.raw(fmt.Sprintf("<qdt:DateTimeString format=\"102\">%s</qdt:DateTimeString>", ref.IssueDate.CII()))
			w.close("ram:FormattedIssueDateTime")
		}
		w.close("ram:InvoiceReferencedDocument")
	}
	if inv.AccountingCost != "" {
		w.open("ram:ReceivableSpecifiedTradeAccountingAccount")
		w.text("ram:ID", inv.AccountingCost)
		w.close("ram:ReceivableSpecifiedTradeAccountingAccount")
	}
	w.close("ram:ApplicableHeaderTradeSettlement")
}

func (w *ciiWriter) renderLine(line Line) {
	w.open("ram:IncludedSupplyChainTradeLineItem")

	w.open("ram:AssociatedDocumentLineDocument")
	w.text("ram:LineID", line.ID)
	if line.Note != "" {
		w.open("ram:IncludedNote")
		w.text("ram:Content", line.Note)
		w.close("ram:IncludedNote")
	}
	w.close("ram:AssociatedDocumentLineDocument")

	item := line.Item
	w.open("ram:SpecifiedTradeProduct")
	if item.StandardID.Present() {
		w.identifier("ram:GlobalID", item.StandardID)
	}
	w.optional("ram:SellerAssignedID", item.SellerID)
	w.optional("ram:BuyerAssignedID", item.BuyerID)
	w.text("ram:Name", item.Name)
	w.optional("ram:Description", item.Description)
	for _, attr := range item.Attributes {
		w.open("ram:ApplicableProductCharacteristic")
		w.text("ram:Description", attr.Name)
		w.text("ram:Value", attr.Value)
		w.close("ram:ApplicableProductCharacteristic")
	}
	for _, c := range item.Classifications {
		w.open("ram:DesignatedProductClassification")
		attrs := ""
		if c.Scheme != "" {
			attrs += fmt.Sprintf(` listID="%s"`, escape(c.Scheme))
		}
		if c.SchemeVersion != "" {
			attrs += fmt.Sprintf(` listVersionID="%s"`, escape(c.SchemeVersion))
		}
		w.raw(fmt.Sprintf("<ram:ClassCode%s>%s</ram:ClassCode>", attrs, escape(c.Value)))
		w.close("ram:DesignatedProductClassification")
	}
	if item.OriginCountryCode != "" {
		w.open("ram:OriginTradeCountry")
		w.text("ram:ID", item.OriginCountryCode)
		w.close("ram:OriginTradeCountry")
	}
	w.close("ram:SpecifiedTradeProduct")

	w.open("ram:SpecifiedLineTradeAgreement")
	if line.OrderLineID != "" {
		w.open("ram:BuyerOrderReferencedDocument")
		w.text("ram:LineID", line.OrderLineID)
		w.close("ram:BuyerOrderReferencedDocument")
	}
	if line.Price.GrossPrice.Present() {
		w.open("ram:GrossPriceProductTradePrice")
		w.amount("ram:ChargeAmount", line.Price.GrossPrice)
		if line.Price.Discount.Present() {
			w.open("ram:AppliedTradeAllowanceCharge")
			w.open("ram:ChargeIndicator")
			w.raw("<udt:Indicator>false</udt:Indicator>")
			w.close("ram:ChargeIndicator")
			w.amount("ram:ActualAmount", line.Price.Discount)
			w.close("ram:AppliedTradeAllowanceCharge")
		}
		w.close("ram:GrossPriceProductTradePrice")
	}
	w.open("ram:NetPriceProductTradePrice")
	w.amount("ram:ChargeAmount", line.Price.NetPrice)
	if line.Price.BaseQuantity.Present() {
		w.quantity("ram:BasisQuantity", line.Price.BaseQuantity, line.Price.BaseUnit)
	}
	w.close("ram:NetPriceProductTradePrice")
	w.close("ram:SpecifiedLineTradeAgreement")

	w.open("ram:SpecifiedLineTradeDelivery")
	w.quantity("ram:BilledQuantity", line.Quantity, line.UnitCode)
	w.close("ram:SpecifiedLineTradeDelivery")

	w.open("ram:SpecifiedLineTradeSettlement")
	w.open("ram:ApplicableTradeTax")
	w.text("ram:TypeCode", "VAT")
	w.optional("ram:CategoryCode", line.VAT.CategoryCode)
	if line.VAT.Rate.Present() {
		w.amount("ram:RateApplicablePercent", line.VAT.Rate)
	}
	w.close("ram:ApplicableTradeTax")
	if p := line.Period; p != nil && (p.Start.Present() || p.End.Present()) {
		w.open("ram:BillingSpecifiedPeriod")
		if p.Start.Present() {
			w.date("ram:StartDateTime", p.Start)
		}
		if p.End.Present() {
			w.date("ram:EndDateTime", p.End)
		}
		w.close("ram:BillingSpecifiedPeriod")
	}
	for _, a := range line.Allowances {
		w.allowanceCharge(a, false)
	}
	for _, a := range line.Charges {
		w.allowanceCharge(a, true)
	}
	w.open("ram:SpecifiedTradeSettlementLineMonetarySummation")
	w.amount("ram:LineTotalAmount", line.NetAmount)
	w.close("ram:SpecifiedTradeSettlementLineMonetarySummation")
	if line.ObjectIdentifier.Present() {
		w.docRef("ram:AdditionalReferencedDocument", line.ObjectIdentifier.Value,
			ciiRefObject, line.ObjectIdentifier.Scheme)
	}
	if line.AccountingCost != "" {
		w.open("ram:ReceivableSpecifiedTradeAccountingAccount")
		w.text("ram:ID", line.AccountingCost)
		w.close("ram:ReceivableSpecifiedTradeAccountingAccount")
	}
	w.close("ram:SpecifiedLineTradeSettlement")

	w.close("ram:IncludedSupplyChainTradeLineItem")
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func escape(s string) string { return html.EscapeString(s) }

func encodeBase64(data []byte) string { return base64.StdEncoding.EncodeToString(data) }
