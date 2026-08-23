package einvoice

import (
	"fmt"
	"strings"
)

// ciiWriter builds an XML document with indentation.
//
// It is a writer rather than a marshaller because CII is a sequence type: the
// order of elements is part of the schema, and a document whose elements come
// in a different order is rejected before any business rule runs. With a writer
// the order is written down where somebody changing it will see it.
type ciiWriter struct {
	buf   strings.Builder
	depth int
}

func (w *ciiWriter) indent() {
	for i := 0; i < w.depth; i++ {
		w.buf.WriteString("  ")
	}
}

func (w *ciiWriter) raw(s string) {
	if strings.HasPrefix(s, "<?xml") {
		w.buf.WriteString(s)
		return
	}
	w.indent()
	w.buf.WriteString(s)
	w.buf.WriteString("\n")
}

func (w *ciiWriter) open(name string, attrs ...string) {
	w.indent()
	w.buf.WriteString("<" + name)
	for _, a := range attrs {
		w.buf.WriteString("\n")
		for i := 0; i < w.depth+1; i++ {
			w.buf.WriteString("  ")
		}
		w.buf.WriteString(a)
	}
	w.buf.WriteString(">\n")
	w.depth++
}

func (w *ciiWriter) close(name string) {
	w.depth--
	w.indent()
	w.buf.WriteString("</" + name + ">\n")
}

// text writes an element even when the value is empty — for a mandatory field,
// an empty element and a missing one are different findings, and the writer
// must not quietly turn one into the other.
func (w *ciiWriter) text(name, value string) {
	w.raw(fmt.Sprintf("<%s>%s</%s>", name, escape(value), name))
}

// optional writes the element only when there is something to write.
func (w *ciiWriter) optional(name, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	w.text(name, value)
}

func (w *ciiWriter) amount(name string, a Amount) {
	if !a.Present() {
		return
	}
	w.text(name, a.String())
}

func (w *ciiWriter) amountWithCurrency(name string, a Amount, currency string) {
	if !a.Present() {
		return
	}
	if strings.TrimSpace(currency) == "" {
		w.text(name, a.String())
		return
	}
	w.raw(fmt.Sprintf(`<%s currencyID="%s">%s</%s>`, name, escape(currency), escape(a.String()), name))
}

func (w *ciiWriter) quantity(name string, value Amount, unit string) {
	if !value.Present() && unit == "" {
		return
	}
	if unit == "" {
		w.text(name, value.String())
		return
	}
	w.raw(fmt.Sprintf(`<%s unitCode="%s">%s</%s>`, name, escape(unit), escape(value.String()), name))
}

func (w *ciiWriter) date(name string, d Date) {
	if !d.Present() {
		return
	}
	w.open(name)
	w.raw(fmt.Sprintf(`<udt:DateTimeString format="102">%s</udt:DateTimeString>`, escape(d.CII())))
	w.close(name)
}

func (w *ciiWriter) identifier(name string, id Identifier) {
	if !id.Present() && id.Scheme == "" {
		return
	}
	if id.Scheme == "" {
		w.text(name, id.Value)
		return
	}
	w.raw(fmt.Sprintf(`<%s schemeID="%s">%s</%s>`, name, escape(id.Scheme), escape(id.Value), name))
}

func (w *ciiWriter) address(a *Address) {
	if a == nil {
		return
	}
	w.open("ram:PostalTradeAddress")
	w.optional("ram:PostcodeCode", a.PostCode)
	w.optional("ram:LineOne", a.LineOne)
	w.optional("ram:LineTwo", a.LineTwo)
	w.optional("ram:LineThree", a.LineThree)
	w.optional("ram:CityName", a.City)
	w.optional("ram:CountryID", a.CountryCode)
	w.optional("ram:CountrySubDivisionName", a.Subdivision)
	w.close("ram:PostalTradeAddress")
}

func (w *ciiWriter) party(element string, p *Party) {
	if p == nil {
		return
	}
	w.open(element)
	for _, id := range p.Identifiers {
		if id.Scheme == "" {
			w.text("ram:ID", id.Value)
		} else {
			w.identifier("ram:GlobalID", id)
		}
	}
	w.optional("ram:Name", p.Name)
	w.optional("ram:Description", p.AdditionalLegalInfo)
	if p.LegalRegistration.Present() || p.TradingName != "" {
		w.open("ram:SpecifiedLegalOrganization")
		w.identifier("ram:ID", p.LegalRegistration)
		w.optional("ram:TradingBusinessName", p.TradingName)
		w.close("ram:SpecifiedLegalOrganization")
	}
	if c := p.Contact; c != nil {
		w.open("ram:DefinedTradeContact")
		w.optional("ram:PersonName", c.Name)
		if c.Phone != "" {
			w.open("ram:TelephoneUniversalCommunication")
			w.text("ram:CompleteNumber", c.Phone)
			w.close("ram:TelephoneUniversalCommunication")
		}
		if c.Email != "" {
			w.open("ram:EmailURIUniversalCommunication")
			w.text("ram:URIID", c.Email)
			w.close("ram:EmailURIUniversalCommunication")
		}
		w.close("ram:DefinedTradeContact")
	}
	w.address(p.Address)
	// Auch eine elektronische Adresse, die nur ihr Schema nennt, wird
	// geschrieben. Sie ist als Adresse wertlos — aber sie stand im Dokument,
	// und BR-62 hat dazu etwas zu sagen.
	if p.ElectronicAddress.Present() || p.ElectronicAddress.Scheme != "" {
		w.open("ram:URIUniversalCommunication")
		w.identifier("ram:URIID", p.ElectronicAddress)
		w.close("ram:URIUniversalCommunication")
	}
	if p.VATIdentifier != "" {
		w.open("ram:SpecifiedTaxRegistration")
		w.identifier("ram:ID", Identifier{Value: p.VATIdentifier, Scheme: ciiSchemeVAT})
		w.close("ram:SpecifiedTaxRegistration")
	}
	if p.TaxRegistration != "" {
		w.open("ram:SpecifiedTaxRegistration")
		w.identifier("ram:ID", Identifier{Value: p.TaxRegistration, Scheme: ciiSchemeTaxNo})
		w.close("ram:SpecifiedTaxRegistration")
	}
	w.close(element)
}

func (w *ciiWriter) docRef(element, id, typeCode, scheme string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	w.open(element)
	w.text("ram:IssuerAssignedID", id)
	w.optional("ram:TypeCode", typeCode)
	w.optional("ram:ReferenceTypeCode", scheme)
	w.close(element)
}

func (w *ciiWriter) supportingDoc(doc SupportingDocument) {
	w.open("ram:AdditionalReferencedDocument")
	w.text("ram:IssuerAssignedID", doc.Reference)
	w.optional("ram:URIID", doc.ExternalURI)
	w.text("ram:TypeCode", ciiRefSupporting)
	w.optional("ram:Name", doc.Description)
	if len(doc.Attachment) > 0 {
		attrs := ""
		if doc.MimeCode != "" {
			attrs += fmt.Sprintf(` mimeCode="%s"`, escape(doc.MimeCode))
		}
		if doc.Filename != "" {
			attrs += fmt.Sprintf(` filename="%s"`, escape(doc.Filename))
		}
		w.raw(fmt.Sprintf("<ram:AttachmentBinaryObject%s>%s</ram:AttachmentBinaryObject>",
			attrs, encodeBase64(doc.Attachment)))
	}
	w.close("ram:AdditionalReferencedDocument")
}

func (w *ciiWriter) tradeTax(element string, group VATBreakdown, inv *Invoice) {
	w.open(element)
	w.amount("ram:CalculatedAmount", group.TaxAmount)
	w.text("ram:TypeCode", orDefault(group.TypeCode, "VAT"))
	w.optional("ram:ExemptionReason", group.ExemptionReason)
	w.amount("ram:BasisAmount", group.TaxableAmount)
	w.optional("ram:CategoryCode", group.CategoryCode)
	w.optional("ram:ExemptionReasonCode", group.ExemptionReasonCode)
	if inv != nil && inv.TaxPointDate.Present() {
		w.date("ram:TaxPointDate", inv.TaxPointDate)
	}
	if inv != nil {
		w.optional("ram:DueDateTypeCode", inv.TaxPointDateCode)
	}
	if group.Rate.Present() {
		w.amount("ram:RateApplicablePercent", group.Rate)
	}
	w.close(element)
}

func (w *ciiWriter) allowanceCharge(a AllowanceCharge, isCharge bool) {
	w.open("ram:SpecifiedTradeAllowanceCharge")
	w.open("ram:ChargeIndicator")
	w.raw(fmt.Sprintf("<udt:Indicator>%t</udt:Indicator>", isCharge))
	w.close("ram:ChargeIndicator")
	w.amount("ram:CalculationPercent", a.Percentage)
	w.amount("ram:BasisAmount", a.BaseAmount)
	w.amount("ram:ActualAmount", a.Amount)
	w.optional("ram:ReasonCode", a.ReasonCode)
	w.optional("ram:Reason", a.Reason)
	if a.VATCategory != "" || a.VATRate.Present() {
		w.open("ram:CategoryTradeTax")
		w.text("ram:TypeCode", orDefault(a.VATTypeCode, "VAT"))
		w.optional("ram:CategoryCode", a.VATCategory)
		if a.VATRate.Present() {
			w.amount("ram:RateApplicablePercent", a.VATRate)
		}
		w.close("ram:CategoryTradeTax")
	}
	w.close("ram:SpecifiedTradeAllowanceCharge")
}
