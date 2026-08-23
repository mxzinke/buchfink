package einvoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// UBL reference document type codes, the counterpart of the CII ones.
const (
	ublRefObject = "130" // BT-18 / BT-128 Objektkennung
	// ublTaxSchemeVAT marks the tax registration that is the USt-IdNr.; any
	// other scheme is the national Steuernummer (BT-32).
	ublTaxSchemeVAT = "VAT"
	// ublSchemeSEPA marks the creditor identifier of a direct debit (BT-90).
	ublSchemeSEPA = "SEPA"
)

// ParseUBL reads an OASIS UBL Invoice or CreditNote into the semantic model.
//
// Both root elements are accepted. At the semantic level they are the same
// document — what separates them is the type code (BT-3), and that is a field,
// not a shape. Treating them as two document kinds would mean writing the whole
// mapping twice for a difference of one element name.
func ParseUBL(data []byte) (*Invoice, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var doc ublDocument
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("das XML konnte nicht gelesen werden: %w", err)
	}
	switch doc.XMLName.Space {
	case nsUBLInvoice, nsUBLCreditNote:
	default:
		return nil, fmt.Errorf("das Wurzelelement %q gehört zu keiner UBL-Rechnung", doc.XMLName.Local)
	}

	inv := &Invoice{Syntax: SyntaxUBL}
	ublMapHeader(inv, &doc)
	ublMapParties(inv, &doc)
	ublMapDelivery(inv, &doc)
	ublMapPayment(inv, &doc)
	ublMapTotals(inv, &doc)
	ublMapLines(inv, &doc)
	return inv, nil
}

func ublMapHeader(inv *Invoice, doc *ublDocument) {
	inv.SpecificationID = trim(doc.CustomizationID)
	inv.BusinessProcess = trim(doc.ProfileID)
	inv.Number = trim(doc.ID)
	inv.IssueDate = NewDate(doc.IssueDate)
	inv.DueDate = NewDate(doc.DueDate)
	inv.TypeCode = firstNonEmpty([]string{doc.InvoiceTypeCode, doc.CreditNoteTypeCode})
	inv.TaxPointDate = NewDate(doc.TaxPointDate)
	inv.Currency = trim(doc.DocumentCurrency)
	inv.TaxCurrency = trim(doc.TaxCurrency)
	inv.AccountingCost = trim(doc.AccountingCost)
	inv.BuyerReference = trim(doc.BuyerReference)

	for _, n := range doc.Notes {
		inv.Notes = append(inv.Notes, ublParseNote(n.trimmed()))
	}

	if p := doc.InvoicePeriod; p != nil {
		code := ublTaxPointCode(trim(p.DescriptionCode))
		inv.Period = &Period{
			Start:           NewDate(p.StartDate),
			End:             NewDate(p.EndDate),
			DescriptionCode: code,
		}
		inv.TaxPointDateCode = code
	}

	if o := doc.OrderReference; o != nil {
		inv.OrderReference = trim(o.ID)
		inv.SalesOrderReference = trim(o.SalesOrderID)
	}
	inv.DespatchAdviceReference = trim(doc.DespatchDocumentReference.ID.Value)
	inv.ReceivingAdviceReference = trim(doc.ReceiptDocumentReference.ID.Value)
	inv.TenderReference = trim(doc.OriginatorDocumentReference.ID.Value)
	inv.ContractReference = trim(doc.ContractDocumentReference.ID.Value)
	inv.ProjectReference = trim(doc.ProjectReference.ID.Value)

	for _, ref := range doc.BillingReference {
		inv.PrecedingInvoices = append(inv.PrecedingInvoices, PrecedingInvoice{
			Number:    trim(ref.InvoiceDocumentReference.ID),
			IssueDate: NewDate(ref.InvoiceDocumentReference.IssueDate),
		})
	}

	for _, ref := range doc.AdditionalDocumentReference {
		if trim(ref.DocumentTypeCode) == ublRefObject {
			inv.ObjectIdentifier = Identifier{
				Value:  ref.ID.trimmed(),
				Scheme: trim(ref.ID.SchemeID),
			}
			continue
		}
		supporting := SupportingDocument{
			Reference:   ref.ID.trimmed(),
			Description: trim(ref.Description),
		}
		if a := ref.Attachment; a != nil {
			supporting.Attachment = decodeBase64(a.EmbeddedDocument.Value)
			supporting.MimeCode = trim(a.EmbeddedDocument.MimeCode)
			supporting.Filename = trim(a.EmbeddedDocument.Filename)
			if e := a.ExternalReference; e != nil {
				supporting.ExternalURI = trim(e.URI)
			}
		}
		inv.SupportingDocs = append(inv.SupportingDocs, supporting)
	}

	for _, term := range doc.PaymentTerms {
		if inv.PaymentTermsNote == "" {
			inv.PaymentTermsNote = trim(term.Note)
		}
	}
}

func ublMapParties(inv *Invoice, doc *ublDocument) {
	inv.Seller = ublParseParty(&doc.AccountingSupplierParty.Party)
	inv.Buyer = ublParseParty(&doc.AccountingCustomerParty.Party)
	if doc.PayeeParty != nil {
		p := ublParseParty(doc.PayeeParty)
		inv.Payee = &p
	}
	if doc.TaxRepresentativeParty != nil {
		p := ublParseParty(doc.TaxRepresentativeParty)
		inv.TaxRepresentative = &p
	}
	// BT-90 steht bei den Kennungen des Verkäufers, nicht bei der Zahlung.
	for _, id := range doc.AccountingSupplierParty.Party.PartyIdentification {
		if strings.EqualFold(trim(id.ID.SchemeID), ublSchemeSEPA) {
			inv.CreditorReference = id.ID.trimmed()
		}
	}
}

func ublParseParty(p *ublParty) Party {
	party := Party{}

	if p.EndpointID.trimmed() != "" || trim(p.EndpointID.SchemeID) != "" {
		party.ElectronicAddress = Identifier{
			Value:  p.EndpointID.trimmed(),
			Scheme: trim(p.EndpointID.SchemeID),
		}
	}
	for _, id := range p.PartyIdentification {
		// Ein Eintrag, der nur das Schema nennt, wird mitgenommen. Er ist zwar
		// als Kennung wertlos, aber genau das soll die Codelistenprüfung sagen
		// dürfen — wer ihn hier wegwirft, macht aus einem meldbaren Mangel ein
		// Feld, das es nie gab.
		if id.ID.trimmed() == "" && trim(id.ID.SchemeID) == "" {
			continue
		}
		party.Identifiers = append(party.Identifiers, Identifier{
			Value:  id.ID.trimmed(),
			Scheme: trim(id.ID.SchemeID),
		})
	}
	for _, n := range p.PartyName {
		if trim(n.Name) != "" {
			party.TradingName = trim(n.Name)
			break
		}
	}
	for _, e := range p.PartyLegalEntity {
		if party.Name == "" {
			party.Name = trim(e.RegistrationName)
		}
		if !party.LegalRegistration.Present() &&
			(e.CompanyID.trimmed() != "" || trim(e.CompanyID.SchemeID) != "") {
			party.LegalRegistration = Identifier{
				Value:  e.CompanyID.trimmed(),
				Scheme: trim(e.CompanyID.SchemeID),
			}
		}
		if party.AdditionalLegalInfo == "" {
			party.AdditionalLegalInfo = trim(e.CompanyLegalForm)
		}
	}
	// Fällt der Registrierungsname aus, ist der Handelsname der Name — sonst
	// stünde die Rechnung ohne Beteiligten da, obwohl sie einen nennt.
	if party.Name == "" {
		party.Name = party.TradingName
	}

	for _, scheme := range p.PartyTaxScheme {
		id := trim(scheme.CompanyID)
		if id == "" {
			continue
		}
		if strings.EqualFold(trim(scheme.TaxScheme.ID), ublTaxSchemeVAT) {
			party.VATIdentifier = id
		} else {
			party.TaxRegistration = id
		}
	}

	if a := p.PostalAddress; a != nil {
		address := &Address{
			LineOne:     trim(a.StreetName),
			LineTwo:     trim(a.AdditionalStreetName),
			City:        trim(a.CityName),
			PostCode:    trim(a.PostalZone),
			Subdivision: trim(a.CountrySubentity),
			CountryCode: trim(a.Country.IdentificationCode),
		}
		for _, l := range a.AddressLine {
			if trim(l.Line) != "" {
				address.LineThree = trim(l.Line)
				break
			}
		}
		party.Address = address
	}

	if c := p.Contact; c != nil {
		party.Contact = &Contact{
			Name:  trim(c.Name),
			Phone: trim(c.Telephone),
			Email: trim(c.ElectronicMail),
		}
	}
	return party
}

func ublMapDelivery(inv *Invoice, doc *ublDocument) {
	d := doc.Delivery
	if d == nil {
		return
	}
	delivery := Delivery{Date: NewDate(d.ActualDeliveryDate)}
	if loc := d.DeliveryLocation; loc != nil {
		if loc.ID.trimmed() != "" {
			delivery.LocationID = Identifier{Value: loc.ID.trimmed(), Scheme: trim(loc.ID.SchemeID)}
		}
		if a := loc.Address; a != nil {
			address := &Address{
				LineOne:     trim(a.StreetName),
				LineTwo:     trim(a.AdditionalStreetName),
				City:        trim(a.CityName),
				PostCode:    trim(a.PostalZone),
				Subdivision: trim(a.CountrySubentity),
				CountryCode: trim(a.Country.IdentificationCode),
			}
			for _, l := range a.AddressLine {
				if trim(l.Line) != "" {
					address.LineThree = trim(l.Line)
					break
				}
			}
			delivery.Address = address
		}
	}
	if party := d.DeliveryParty; party != nil {
		for _, n := range party.PartyName {
			if trim(n.Name) != "" {
				delivery.Name = trim(n.Name)
				break
			}
		}
	}
	inv.Delivery = &delivery
}

func ublMapPayment(inv *Invoice, doc *ublDocument) {
	for _, m := range doc.PaymentMeans {
		means := PaymentMeans{
			TypeCode:       m.PaymentMeansCode.trimmed(),
			TypeText:       trim(m.PaymentMeansCode.Name),
			RemittanceInfo: firstNonEmpty(m.PaymentID),
		}
		if account := m.PayeeFinancialAccount; account != nil {
			transfer := CreditTransfer{
				AccountID:   trim(account.ID),
				AccountName: trim(account.Name),
			}
			if branch := account.FinancialInstitutionBranch; branch != nil {
				transfer.ProviderID = trim(branch.ID)
			}
			means.CreditTransfer = append(means.CreditTransfer, transfer)
		}
		if card := m.CardAccount; card != nil {
			means.Card = &CardInformation{
				PrimaryAccountNumber: trim(card.PrimaryAccountNumberID),
				HolderName:           trim(card.HolderName),
			}
		}
		if mandate := m.PaymentMandate; mandate != nil {
			debit := &DirectDebit{
				MandateReference: trim(mandate.ID),
				CreditorID:       inv.CreditorReference,
			}
			if account := mandate.PayerFinancialAccount; account != nil {
				debit.DebitedAccount = trim(account.ID)
			}
			means.DirectDebit = debit
		}
		inv.PaymentMeans = append(inv.PaymentMeans, means)
	}
}

func ublMapTotals(inv *Invoice, doc *ublDocument) {
	for _, a := range doc.AllowanceCharge {
		parsed := ublParseAllowanceCharge(a)
		if a.isCharge() {
			inv.Charges = append(inv.Charges, parsed)
		} else {
			inv.Allowances = append(inv.Allowances, parsed)
		}
	}

	// UBL trägt den Steuergesamtbetrag je Währung in einem eigenen TaxTotal.
	// Die Aufschlüsselung steht nur in dem der Rechnungswährung; das andere
	// nennt allein den umgerechneten Betrag (BT-111).
	wantCurrency := strings.ToUpper(trim(inv.Currency))
	for _, total := range doc.TaxTotal {
		currency := strings.ToUpper(trim(total.TaxAmount.CurrencyID))
		if currency != "" && currency != wantCurrency {
			inv.Totals.TaxTotalInTaxCurr = NewAmount(total.TaxAmount.Value)
			inv.Totals.TaxTotalInTaxCurrCurrency = trim(total.TaxAmount.CurrencyID)
			continue
		}
		inv.Totals.TaxTotalCount++
		if !inv.Totals.TaxTotal.Present() {
			inv.Totals.TaxTotal = NewAmount(total.TaxAmount.Value)
			inv.Totals.TaxTotalCurrency = trim(total.TaxAmount.CurrencyID)
		}
		for _, sub := range total.TaxSubtotal {
			inv.VATBreakdown = append(inv.VATBreakdown, VATBreakdown{
				TypeCode:            trim(sub.TaxCategory.TaxScheme.ID),
				TaxableAmount:       NewAmount(sub.TaxableAmount.Value),
				TaxAmount:           NewAmount(sub.TaxAmount.Value),
				CategoryCode:        sub.TaxCategory.ID.trimmed(),
				Rate:                NewAmount(sub.TaxCategory.Percent),
				ExemptionReason:     trim(sub.TaxCategory.TaxExemptionReason),
				ExemptionReasonCode: trim(sub.TaxCategory.TaxExemptionReasonCode),
			})
		}
	}

	sum := doc.LegalMonetaryTotal
	inv.Totals.LineTotal = NewAmount(sum.LineExtensionAmount)
	inv.Totals.AllowanceTotal = NewAmount(sum.AllowanceTotalAmount)
	inv.Totals.ChargeTotal = NewAmount(sum.ChargeTotalAmount)
	inv.Totals.TaxBasisTotal = NewAmount(sum.TaxExclusiveAmount)
	inv.Totals.GrandTotal = NewAmount(sum.TaxInclusiveAmount)
	inv.Totals.PrepaidAmount = NewAmount(sum.PrepaidAmount)
	inv.Totals.RoundingAmount = NewAmount(sum.PayableRoundingAmount)
	inv.Totals.DuePayableAmount = NewAmount(sum.PayableAmount)
}

func ublParseAllowanceCharge(a ublAllowanceCharge) AllowanceCharge {
	return AllowanceCharge{
		Amount:      NewAmount(a.Amount),
		BaseAmount:  NewAmount(a.BaseAmount),
		Percentage:  NewAmount(a.MultiplierFactor),
		VATTypeCode: trim(a.TaxCategory.TaxScheme.ID),
		VATCategory: a.TaxCategory.ID.trimmed(),
		VATRate:     NewAmount(a.TaxCategory.Percent),
		Reason:      trim(a.AllowanceChargeReason),
		ReasonCode:  trim(a.AllowanceChargeReasonCode),
	}
}

func ublMapLines(inv *Invoice, doc *ublDocument) {
	lines := doc.InvoiceLine
	if len(lines) == 0 {
		lines = doc.CreditNoteLine
	}
	for _, l := range lines {
		quantity := l.InvoicedQuantity
		if quantity.trimmed() == "" && trim(quantity.UnitCode) == "" {
			quantity = l.CreditedQuantity
		}

		line := Line{
			ID:             trim(l.ID),
			Note:           firstNonEmpty(l.Note),
			Quantity:       NewAmount(quantity.Value),
			UnitCode:       trim(quantity.UnitCode),
			NetAmount:      NewAmount(l.LineExtensionAmount),
			AccountingCost: trim(l.AccountingCost),
		}
		if r := l.OrderLineReference; r != nil {
			line.OrderLineID = trim(r.LineID)
		}
		for _, ref := range l.DocumentReference {
			if trim(ref.DocumentTypeCode) == ublRefObject {
				line.ObjectIdentifier = Identifier{
					Value:  ref.ID.trimmed(),
					Scheme: trim(ref.ID.SchemeID),
				}
			}
		}
		if p := l.InvoicePeriod; p != nil {
			line.Period = &Period{Start: NewDate(p.StartDate), End: NewDate(p.EndDate)}
		}
		for _, a := range l.AllowanceCharge {
			parsed := ublParseAllowanceCharge(a)
			if a.isCharge() {
				line.Charges = append(line.Charges, parsed)
			} else {
				line.Allowances = append(line.Allowances, parsed)
			}
		}

		line.Price = Price{
			NetPrice:     NewAmount(l.Price.PriceAmount),
			BaseQuantity: NewAmount(l.Price.BaseQuantity.Value),
			BaseUnit:     trim(l.Price.BaseQuantity.UnitCode),
		}
		for _, a := range l.Price.AllowanceCharge {
			line.Price.Discount = NewAmount(a.Amount)
			line.Price.GrossPrice = NewAmount(a.BaseAmount)
			break
		}

		item := l.Item
		line.Item = Item{
			Name:        trim(item.Name),
			Description: trim(item.Description),
		}
		if id := item.SellersItemIdentification; id != nil {
			line.Item.SellerID = trim(id.ID)
		}
		if id := item.BuyersItemIdentification; id != nil {
			line.Item.BuyerID = trim(id.ID)
		}
		if id := item.StandardItemIdentification; id != nil &&
			(id.ID.trimmed() != "" || trim(id.ID.SchemeID) != "") {
			line.Item.StandardID = Identifier{
				Value:  id.ID.trimmed(),
				Scheme: trim(id.ID.SchemeID),
			}
		}
		if c := item.OriginCountry; c != nil {
			line.Item.OriginCountryCode = trim(c.IdentificationCode)
		}
		for _, c := range item.CommodityClassification {
			if c.ItemClassificationCode.trimmed() == "" && trim(c.ItemClassificationCode.ListID) == "" {
				continue
			}
			line.Item.Classifications = append(line.Item.Classifications, Identifier{
				Value:         c.ItemClassificationCode.trimmed(),
				Scheme:        trim(c.ItemClassificationCode.ListID),
				SchemeVersion: trim(c.ItemClassificationCode.ListVersionID),
			})
		}
		for _, p := range item.AdditionalItemProperty {
			line.Item.Attributes = append(line.Item.Attributes, ItemAttribute{
				Name:  trim(p.Name),
				Value: trim(p.Value),
			})
		}
		for _, c := range item.ClassifiedTaxCategory {
			line.VAT = LineVAT{CategoryCode: c.ID.trimmed(), Rate: NewAmount(c.Percent)}
			break
		}

		inv.Lines = append(inv.Lines, line)
	}
}

// ublParseNote splits a UBL note into subject code and text.
//
// UBL has no separate element for the subject (BT-21); the convention is to
// prefix the text with "#CODE#". The prefix is only taken as a code when it is
// actually one — a note that happens to start with a hash is a note, and
// inventing a subject code from it would produce a finding about a field the
// document does not have.
func ublParseNote(text string) Note {
	if !strings.HasPrefix(text, "#") {
		return Note{Text: text}
	}
	end := strings.Index(text[1:], "#")
	if end < 0 {
		return Note{Text: text}
	}
	code := text[1 : end+1]
	if !inCodeList(code, untdid4451) {
		return Note{Text: text}
	}
	return Note{SubjectCode: code, Text: strings.TrimSpace(text[end+2:])}
}

// ublTaxPointCode converts the UBL spelling of BT-8 into the semantic one.
//
// The two syntaxes encode the same three meanings with different code lists —
// UBL writes UNTDID 2475, the semantic model of EN 16931 uses UNTDID 2005. The
// value has to be converted rather than passed through, or a UBL invoice would
// be reported for a code that is perfectly correct in its own syntax.
//
// An unknown value is kept as written, so that the code list rule reports what
// the document actually said.
func ublTaxPointCode(code string) string {
	switch code {
	case "3": // Rechnungsdatum
		return "5"
	case "35": // Lieferdatum
		return "29"
	case "432": // Zahlungsdatum
		return "72"
	}
	return code
}
