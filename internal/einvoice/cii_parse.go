package einvoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// CII reference document type codes.
//
// CII routes three different business terms through the same
// AdditionalReferencedDocument element and tells them apart by this code. A
// reader that ignores it turns a tender reference into an attachment.
const (
	ciiRefTender     = "50"  // BT-17 Ausschreibungsnummer
	ciiRefObject     = "130" // BT-18 / BT-128 Objektkennung
	ciiRefSupporting = "916" // BG-24 rechnungsbegründende Unterlage
	ciiSchemeVAT     = "VA"  // USt-IdNr.
	ciiSchemeTaxNo   = "FC"  // Steuernummer
)

// ParseCII reads a UN/CEFACT Cross Industry Invoice into the semantic model.
//
// It is deliberately forgiving about what the document contains and strict only
// about what it is: anything that parses as a CrossIndustryInvoice is mapped as
// far as it goes, and judging it is the validator's job. A reader that refused
// incomplete documents would make it impossible to say *why* one is incomplete,
// which is the more useful answer.
func ParseCII(data []byte) (*Invoice, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var doc ciiDocument
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("das XML konnte nicht gelesen werden: %w", err)
	}
	if doc.XMLName.Local != "CrossIndustryInvoice" {
		return nil, fmt.Errorf("das Wurzelelement ist %q, erwartet wird CrossIndustryInvoice", doc.XMLName.Local)
	}

	inv := &Invoice{Syntax: SyntaxCII}
	ciiMapHeader(inv, &doc)
	ciiMapParties(inv, &doc)
	ciiMapDelivery(inv, &doc)
	ciiMapSettlement(inv, &doc)
	ciiMapLines(inv, &doc)
	return inv, nil
}

func ciiMapHeader(inv *Invoice, doc *ciiDocument) {
	inv.BusinessProcess = trim(doc.Context.BusinessProcess.ID)
	inv.SpecificationID = trim(doc.Context.Guideline.ID)
	inv.Number = trim(doc.Doc.ID)
	inv.TypeCode = trim(doc.Doc.TypeCode)
	inv.IssueDate = doc.Doc.IssueDate.date()

	for _, n := range doc.Doc.Notes {
		inv.Notes = append(inv.Notes, Note{
			SubjectCode: trim(n.SubjectCode),
			Text:        trim(n.Content),
		})
	}

	agreement := doc.Trade.Agreement
	inv.BuyerReference = trim(agreement.BuyerReference)
	inv.ProjectReference = trim(agreement.Project.ID)
	inv.ContractReference = trim(agreement.Contract.IssuerAssignedID)
	inv.OrderReference = trim(agreement.BuyerOrder.IssuerAssignedID)
	inv.SalesOrderReference = trim(agreement.SellerOrder.IssuerAssignedID)
	inv.DespatchAdviceReference = trim(doc.Trade.Delivery.DespatchAdvice.IssuerAssignedID)
	inv.ReceivingAdviceReference = trim(doc.Trade.Delivery.ReceivingAdvice.IssuerAssignedID)

	// BT-17, BT-18 und BG-24 teilen sich ein Element und werden über den
	// Typschlüssel auseinandergehalten.
	for _, ref := range agreement.Additional {
		switch trim(ref.TypeCode) {
		case ciiRefTender:
			inv.TenderReference = trim(ref.IssuerAssignedID)
		case ciiRefObject:
			inv.ObjectIdentifier = Identifier{
				Value:  trim(ref.IssuerAssignedID),
				Scheme: trim(ref.ReferenceTypeID),
			}
		default:
			// 916 ist der vorgesehene Schlüssel; alles Übrige als Unterlage zu
			// behandeln ist die verlustfreie Auslegung — eine Datei, die
			// mitgeschickt wurde, gehört zum Beleg, auch wenn der Absender den
			// Schlüssel falsch gesetzt hat.
			inv.SupportingDocs = append(inv.SupportingDocs, SupportingDocument{
				Reference:   trim(ref.IssuerAssignedID),
				Description: trim(ref.Name),
				ExternalURI: trim(ref.URIID),
				Attachment:  ref.Binary.Value,
				MimeCode:    trim(ref.Binary.MimeCode),
				Filename:    trim(ref.Binary.Filename),
			})
		}
	}
}

func ciiMapParties(inv *Invoice, doc *ciiDocument) {
	agreement := doc.Trade.Agreement
	inv.Seller = ciiParseParty(&agreement.Seller)
	inv.Buyer = ciiParseParty(&agreement.Buyer)
	if agreement.TaxRepresentative != nil {
		p := ciiParseParty(agreement.TaxRepresentative)
		inv.TaxRepresentative = &p
	}
	if payee := doc.Trade.Settlement.Payee; payee != nil {
		p := ciiParseParty(payee)
		inv.Payee = &p
	}
}

func ciiParseParty(p *ciiParty) Party {
	party := Party{
		Name:                trim(p.Name),
		AdditionalLegalInfo: trim(p.Description),
	}
	for _, id := range p.ID {
		if trim(id) != "" {
			party.Identifiers = append(party.Identifiers, Identifier{Value: trim(id)})
		}
	}
	for _, g := range p.GlobalID {
		if trim(g.Value) != "" {
			party.Identifiers = append(party.Identifiers, Identifier{
				Value: trim(g.Value), Scheme: trim(g.SchemeID),
			})
		}
	}
	if p.LegalOrg != nil {
		party.LegalRegistration = Identifier{
			Value:  trim(p.LegalOrg.ID.Value),
			Scheme: trim(p.LegalOrg.ID.SchemeID),
		}
		party.TradingName = trim(p.LegalOrg.TradingName)
	}
	for _, r := range p.TaxRegistrations {
		switch strings.ToUpper(trim(r.ID.SchemeID)) {
		case ciiSchemeVAT:
			party.VATIdentifier = trim(r.ID.Value)
		case ciiSchemeTaxNo:
			party.TaxRegistration = trim(r.ID.Value)
		}
	}
	if p.URI != nil {
		party.ElectronicAddress = Identifier{
			Value:  trim(p.URI.URIID.Value),
			Scheme: trim(p.URI.URIID.SchemeID),
		}
	}
	if p.Address != nil {
		party.Address = &Address{
			LineOne:     trim(p.Address.LineOne),
			LineTwo:     trim(p.Address.LineTwo),
			LineThree:   trim(p.Address.LineThree),
			City:        trim(p.Address.CityName),
			PostCode:    trim(p.Address.PostCode),
			Subdivision: firstNonEmpty(p.Address.Subdivision),
			CountryCode: trim(p.Address.CountryID),
		}
	}
	if p.Contact != nil {
		name := trim(p.Contact.PersonName)
		if name == "" {
			name = trim(p.Contact.Department)
		}
		party.Contact = &Contact{
			Name:  name,
			Phone: trim(p.Contact.Phone.Number),
			Email: trim(p.Contact.Email.URIID),
		}
	}
	return party
}

func ciiMapDelivery(inv *Invoice, doc *ciiDocument) {
	d := doc.Trade.Delivery
	delivery := Delivery{Date: d.Event.Occurrence.date()}
	if d.ShipTo != nil {
		party := ciiParseParty(d.ShipTo)
		delivery.Name = party.Name
		delivery.Address = party.Address
		if len(party.Identifiers) > 0 {
			delivery.LocationID = party.Identifiers[0]
		}
	}
	if delivery.Date.Present() || delivery.Name != "" ||
		delivery.Address != nil || delivery.LocationID.Present() {
		inv.Delivery = &delivery
	}
}

func ciiMapSettlement(inv *Invoice, doc *ciiDocument) {
	s := doc.Trade.Settlement
	inv.Currency = trim(s.Currency)
	inv.TaxCurrency = trim(s.TaxCurrency)
	inv.AccountingCost = trim(s.AccountingReference.ID)

	if s.Period != nil {
		inv.Period = &Period{Start: s.Period.Start.date(), End: s.Period.End.date()}
	}

	for _, term := range s.PaymentTerms {
		if inv.PaymentTermsNote == "" {
			inv.PaymentTermsNote = trim(term.Description)
		}
		if !inv.DueDate.Present() {
			inv.DueDate = term.DueDate.date()
		}
	}

	for _, ref := range s.InvoiceRef {
		inv.PrecedingInvoices = append(inv.PrecedingInvoices, Invoicedoc{
			Number:    trim(ref.IssuerAssignedID),
			IssueDate: ref.FormattedIssue.date(),
		})
	}

	for _, m := range s.PaymentMeans {
		means := PaymentMeans{
			TypeCode:       trim(m.TypeCode),
			TypeText:       trim(m.Information),
			RemittanceInfo: trim(s.PaymentReference),
		}
		if m.PayeeAccount != nil {
			account := CreditTransfer{
				AccountID:   firstNonEmpty([]string{m.PayeeAccount.IBAN, m.PayeeAccount.ProprietaryID}),
				AccountName: trim(m.PayeeAccount.AccountName),
			}
			if m.PayeeInstitution != nil {
				account.ProviderID = trim(m.PayeeInstitution.BIC)
			}
			means.CreditTransfer = append(means.CreditTransfer, account)
		}
		if m.Card != nil {
			means.Card = &CardInformation{
				PrimaryAccountNumber: trim(m.Card.ID),
				HolderName:           trim(m.Card.HolderName),
			}
		}
		if m.PayerAccount != nil {
			means.DirectDebit = &DirectDebit{DebitedAccount: trim(m.PayerAccount.IBAN)}
		}
		inv.PaymentMeans = append(inv.PaymentMeans, means)
	}
	// BT-89 steht bei den Zahlungsbedingungen, nicht beim Zahlungsmittel.
	for _, term := range s.PaymentTerms {
		if trim(term.MandateID) == "" {
			continue
		}
		attached := false
		for i := range inv.PaymentMeans {
			if inv.PaymentMeans[i].DirectDebit != nil {
				inv.PaymentMeans[i].DirectDebit.MandateReference = trim(term.MandateID)
				attached = true
			}
		}
		if !attached && len(inv.PaymentMeans) > 0 {
			inv.PaymentMeans[0].DirectDebit = &DirectDebit{MandateReference: trim(term.MandateID)}
		}
	}

	for _, a := range s.AllowancesCharges {
		parsed := ciiParseAllowanceCharge(a)
		if a.isCharge() {
			inv.Charges = append(inv.Charges, parsed)
		} else {
			inv.Allowances = append(inv.Allowances, parsed)
		}
	}

	for _, tax := range s.Taxes {
		inv.VATBreakdown = append(inv.VATBreakdown, ciiParseTradeTax(tax))
		// BT-7 und BT-8 hängen in CII an der Steuergruppe, im Modell am Beleg.
		if !inv.TaxPointDate.Present() && tax.TaxPointDate.date().Present() {
			inv.TaxPointDate = tax.TaxPointDate.date()
		}
		if inv.TaxPointDateCode == "" {
			inv.TaxPointDateCode = trim(tax.DueDateTypeCode)
		}
	}

	sum := s.Summation
	inv.Totals = Totals{
		LineTotal:        NewAmount(sum.LineTotal),
		AllowanceTotal:   NewAmount(sum.AllowanceTotal),
		ChargeTotal:      NewAmount(sum.ChargeTotal),
		TaxBasisTotal:    NewAmount(sum.TaxBasisTotal),
		GrandTotal:       NewAmount(sum.GrandTotal),
		PrepaidAmount:    NewAmount(sum.TotalPrepaid),
		RoundingAmount:   NewAmount(sum.RoundingAmount),
		DuePayableAmount: NewAmount(sum.DuePayable),
	}
	inv.Totals.TaxTotal, inv.Totals.TaxTotalCurrency,
		inv.Totals.TaxTotalInTaxCurr, inv.Totals.TaxTotalInTaxCurrCurrency =
		ciiSplitTaxTotals(sum.TaxTotals, inv.Currency, inv.TaxCurrency)
	inv.Totals.TaxTotalCount = countInCurrency(sum.TaxTotals, inv.Currency)
}

// ciiSplitTaxTotals separates BT-110 from BT-111.
//
// Both are written as TaxTotalAmount and only the currencyID tells them apart.
// Reading whichever comes last picks the wrong one on a Danish invoice that
// accounts its VAT in euro — the amount that then goes into the books is the
// converted one.
func ciiSplitTaxTotals(totals []ciiCurrencyAmount, invoiceCurrency, taxCurrency string) (
	inInvoiceCurrency Amount, invoiceCurrencyID string,
	inTaxCurrency Amount, taxCurrencyID string,
) {
	if len(totals) == 0 {
		return Amount{}, "", Amount{}, ""
	}
	want := strings.ToUpper(trim(invoiceCurrency))
	wantTax := strings.ToUpper(trim(taxCurrency))

	for _, t := range totals {
		stated := strings.ToUpper(trim(t.CurrencyID))
		switch {
		case stated == want:
			inInvoiceCurrency, invoiceCurrencyID = NewAmount(t.Value), trim(t.CurrencyID)
		case wantTax != "" && stated == wantTax:
			inTaxCurrency, taxCurrencyID = NewAmount(t.Value), trim(t.CurrencyID)
		}
	}
	if !inInvoiceCurrency.Present() {
		// Ohne passende Währungskennung ist der erste Wert der in
		// Rechnungswährung. Die Kennung wird trotzdem mitgeführt, damit
		// BR-CL-03 sie prüfen kann.
		inInvoiceCurrency = NewAmount(totals[0].Value)
		if invoiceCurrencyID == "" {
			invoiceCurrencyID = trim(totals[0].CurrencyID)
		}
	}
	return inInvoiceCurrency, invoiceCurrencyID, inTaxCurrency, taxCurrencyID
}

func ciiParseTradeTax(tax ciiTradeTax) VATBreakdown {
	return VATBreakdown{
		TypeCode:            trim(tax.TypeCode),
		TaxableAmount:       NewAmount(tax.BasisAmount),
		TaxAmount:           NewAmount(tax.CalculatedAmount),
		CategoryCode:        trim(tax.CategoryCode),
		Rate:                NewAmount(tax.RateApplicable),
		ExemptionReason:     trim(tax.ExemptionReason),
		ExemptionReasonCode: trim(tax.ExemptionReasonCode),
	}
}

func ciiParseAllowanceCharge(a ciiAllowanceCharge) AllowanceCharge {
	return AllowanceCharge{
		Amount:      NewAmount(a.ActualAmount),
		BaseAmount:  NewAmount(a.BasisAmount),
		Percentage:  NewAmount(a.CalculationPercent),
		VATTypeCode: trim(a.CategoryTradeTax.TypeCode),
		VATCategory: trim(a.CategoryTradeTax.CategoryCode),
		VATRate:     NewAmount(a.CategoryTradeTax.RateApplicable),
		Reason:      trim(a.Reason),
		ReasonCode:  trim(a.ReasonCode),
	}
}

func ciiMapLines(inv *Invoice, doc *ciiDocument) {
	for _, l := range doc.Trade.Lines {
		line := Line{
			ID:             trim(l.Document.LineID),
			Quantity:       NewAmount(l.Delivery.BilledQuantity.Value),
			UnitCode:       trim(l.Delivery.BilledQuantity.UnitCode),
			NetAmount:      NewAmount(l.Settlement.Summation.LineTotal),
			OrderLineID:    trim(l.Agreement.BuyerOrder.LineID),
			AccountingCost: trim(l.Settlement.AccountingReference.ID),
		}
		if len(l.Document.Notes) > 0 {
			line.Note = trim(l.Document.Notes[0].Content)
		}
		if ref := l.Settlement.ObjectReference; trim(ref.IssuerAssignedID) != "" {
			line.ObjectIdentifier = Identifier{
				Value:  trim(ref.IssuerAssignedID),
				Scheme: trim(ref.ReferenceTypeID),
			}
		}
		if l.Settlement.Period != nil {
			line.Period = &Period{
				Start: l.Settlement.Period.Start.date(),
				End:   l.Settlement.Period.End.date(),
			}
		}
		for _, a := range l.Settlement.AllowancesCharges {
			parsed := ciiParseAllowanceCharge(a)
			if a.isCharge() {
				line.Charges = append(line.Charges, parsed)
			} else {
				line.Allowances = append(line.Allowances, parsed)
			}
		}
		line.Price = Price{
			NetPrice:     NewAmount(l.Agreement.NetPrice.ChargeAmount),
			GrossPrice:   NewAmount(l.Agreement.GrossPrice.ChargeAmount),
			BaseQuantity: NewAmount(l.Agreement.NetPrice.BasisQuantity.Value),
			BaseUnit:     trim(l.Agreement.NetPrice.BasisQuantity.UnitCode),
		}
		if !line.Price.BaseQuantity.Present() {
			line.Price.BaseQuantity = NewAmount(l.Agreement.GrossPrice.BasisQuantity.Value)
			line.Price.BaseUnit = trim(l.Agreement.GrossPrice.BasisQuantity.UnitCode)
		}
		for _, applied := range l.Agreement.GrossPrice.AllowanceCharges {
			if !applied.isCharge() {
				line.Price.Discount = NewAmount(applied.ActualAmount)
				break
			}
		}
		line.VAT = LineVAT{
			CategoryCode: trim(l.Settlement.Tax.CategoryCode),
			Rate:         NewAmount(l.Settlement.Tax.RateApplicable),
		}
		line.Item = ciiParseItem(l.Product)
		inv.Lines = append(inv.Lines, line)
	}
}

func ciiParseItem(p ciiProduct) Item {
	item := Item{
		Name:              trim(p.Name),
		Description:       trim(p.Description),
		SellerID:          trim(p.ID),
		BuyerID:           trim(p.BuyerID),
		OriginCountryCode: trim(p.Origin.ID),
	}
	if trim(p.GlobalID.Value) != "" || trim(p.GlobalID.SchemeID) != "" {
		item.StandardID = Identifier{
			Value:  trim(p.GlobalID.Value),
			Scheme: trim(p.GlobalID.SchemeID),
		}
	}
	for _, c := range p.Classifications {
		if trim(c.ClassCode.Value) == "" && trim(c.ClassCode.ListID) == "" {
			continue
		}
		item.Classifications = append(item.Classifications, Identifier{
			Value:         trim(c.ClassCode.Value),
			Scheme:        trim(c.ClassCode.ListID),
			SchemeVersion: trim(c.ClassCode.ListVersionID),
		})
	}
	for _, a := range p.Attributes {
		item.Attributes = append(item.Attributes, ItemAttribute{
			Name:  trim(a.Description),
			Value: trim(a.Value),
		})
	}
	return item
}

func trim(s string) string { return strings.TrimSpace(s) }

func firstNonEmpty(values []string) string {
	for _, v := range values {
		if trim(v) != "" {
			return trim(v)
		}
	}
	return ""
}

// countInCurrency counts how often a tax total is stated in one currency.
//
// A total without a currency identifier counts as the invoice currency, which
// is how both syntaxes read it — and it is the case that matters, because two
// unlabelled totals are exactly the ambiguity BR-CO-15 exists to catch.
func countInCurrency(totals []ciiCurrencyAmount, currency string) int {
	want := strings.ToUpper(trim(currency))
	n := 0
	for _, t := range totals {
		stated := strings.ToUpper(trim(t.CurrencyID))
		if stated == "" || stated == want {
			n++
		}
	}
	return n
}
