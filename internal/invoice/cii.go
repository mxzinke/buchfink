package invoice

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/ruleset"
	"github.com/buchfink/buchfink/internal/einvoice/xrechnung"
)

// Der strukturierte Rechnungsdatensatz entsteht über das semantische Modell.
//
// Vorher stand hier eine XML-Vorlage aus Formatstrings. Sie schrieb fest C62 als
// Mengeneinheit, kannte BT-32 nicht, presste die Empfängeranschrift in eine
// Zeile und hatte für jeden neuen Pflichtwert eine weitere `%s`-Stelle nötig,
// die niemand mehr zählen konnte. Vor allem aber war sie eine zweite Fassung
// dessen, was internal/einvoice bereits kann: dasselbe Modell wird gelesen,
// geprüft und geschrieben, und nur so ist die eigene Rechnung an denselben
// Regeln gemessen wie eine empfangene.

// BuildCII turns an outgoing invoice into the EN 16931 semantic model.
//
// profile decides the guideline identifier (BT-24) and with it, which rules the
// document promises to keep: EN 16931 alone, or the German CIUS on top.
func BuildCII(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact, profile domain.EInvoiceProfile) (*einvoice.Invoice, error) {
	if inv == nil {
		return nil, fmt.Errorf("keine Rechnung übergeben")
	}
	if seller == nil {
		return nil, fmt.Errorf("ohne Unternehmensdaten lässt sich keine Rechnung erzeugen")
	}
	if err := inv.Validate(); err != nil {
		return nil, err
	}
	// Ohne Erwerber gibt es keinen normgerechten Datensatz. BR-07 verlangt
	// seinen Namen, und § 33 UStDV erlässt ihn nur für das Papier: die Norm
	// kennt die Kleinbetragsrechnung nicht. Statt einen Namen zu erfinden,
	// weist BuildCII den Fall zurück — die Rechnung geht dann als PDF hinaus,
	// wozu die Ausnahme des § 33 UStDV von der E-Rechnungspflicht sie
	// berechtigt.
	if buyer == nil && inv.ContactName == "" {
		return nil, fmt.Errorf(
			"ohne Rechnungsempfänger lässt sich kein strukturierter Rechnungsdatensatz erzeugen: " +
				"EN 16931 verlangt den Namen des Erwerbers (BR-07). Die Kleinbetragsrechnung nach " +
				"§ 33 UStDV ist von der E-Rechnungspflicht ausgenommen und wird als PDF ausgestellt")
	}

	specification := einvoice.ProfileEN16931
	if profile == domain.EInvoiceProfileXRechnungCII {
		specification = xrechnung.Current
	}

	currency := inv.Currency
	if currency == "" {
		currency = "EUR"
	}

	doc := &einvoice.Invoice{
		SpecificationID: specification,
		Number:          inv.InvoiceNumber,
		IssueDate:       einvoice.NewDate(inv.Date),
		TypeCode:        inv.ResolvedKind().TypeCode(),
		Currency:        currency,
		DueDate:         einvoice.NewDate(inv.DueDate),
		Syntax:          einvoice.SyntaxCII,
		Seller:          sellerParty(seller),
		Buyer:           buyerParty(inv, buyer),
	}

	// BT-20: die im Voraus vereinbarte Zahlungsbedingung. Sie ist
	// Pflichtangabe, soweit eine Entgeltminderung vereinbart wurde
	// (§ 14 Abs. 4 Nr. 7 UStG) — und bei XRechnung ohne Fälligkeitsdatum
	// ohnehin (BR-DE-18).
	doc.PaymentTermsNote = inv.Terms.Note(inv.Date)

	// BT-10: die Käuferreferenz. Bei einem öffentlichen Auftraggeber ist das
	// die Leitweg-ID, ohne die die Rechnung im Behördennetz nicht zugestellt
	// werden kann.
	if buyer != nil {
		doc.BuyerReference = buyer.LeitwegID
	}

	// BG-13 und BT-72: der Leistungszeitpunkt, Pflichtangabe nach
	// § 14 Abs. 4 Nr. 6 UStG. Bei einer Abschlagsrechnung tritt der Zeitpunkt
	// der Vereinnahmung an seine Stelle, sofern er feststeht — vor dem
	// Geldeingang steht er das nicht, und dann bleibt die Angabe weg.
	deliveryDate := inv.ServiceDateTo
	if inv.ResolvedKind() == domain.InvoiceKindAdvance {
		deliveryDate = inv.PaymentReceivedAt
	}
	if deliveryDate != "" {
		doc.Delivery = &einvoice.Delivery{Date: einvoice.NewDate(deliveryDate)}
	}
	// BT-80, das Bestimmungsland: bei einer innergemeinschaftlichen Lieferung
	// Pflicht (BR-IC-12) und bei einer Ausfuhr ebenso wichtig. Ohne es lässt
	// sich nicht belegen, dass die Ware das Inland verlassen hat — und daran
	// hängt die Steuerbefreiung nach § 6a bzw. § 6 UStG. Bei einem
	// Inlandsumsatz bleibt die Lieferanschrift weg: sie behauptete sonst einen
	// Lieferort, den niemand erfasst hat.
	if buyer != nil && needsDeliveryCountry(inv.TaxTreatment) {
		street, postCode, city := buyer.PostalAddress()
		if doc.Delivery == nil {
			doc.Delivery = &einvoice.Delivery{}
		}
		doc.Delivery.Name = buyer.Name
		doc.Delivery.Address = &einvoice.Address{
			LineOne:     street,
			PostCode:    postCode,
			City:        city,
			CountryCode: countryOrDE(buyer.CountryCode),
		}
	}
	// BG-14: der Abrechnungszeitraum, wenn die Leistung über einen Zeitraum
	// erbracht wurde. Ein Zeitraum von einem Tag ist keiner und bleibt weg.
	if inv.ServiceDateFrom != "" && inv.ServiceDateTo != "" && inv.ServiceDateFrom != inv.ServiceDateTo &&
		inv.ResolvedKind() != domain.InvoiceKindAdvance {
		doc.Period = &einvoice.Period{
			Start: einvoice.NewDate(inv.ServiceDateFrom),
			End:   einvoice.NewDate(inv.ServiceDateTo),
		}
	}

	// BG-3: der Bezug auf die vorausgegangene Rechnung. Ohne ihn kommt eine
	// Korrektur beim Empfänger als zweite, unverbundene Rechnung an, und die
	// XRechnung weist sie zurück (BR-DE-26).
	if inv.CorrectsInvoiceNumber != "" {
		doc.PrecedingInvoices = append(doc.PrecedingInvoices, einvoice.PrecedingInvoice{
			Number:    inv.CorrectsInvoiceNumber,
			IssueDate: einvoice.NewDate(inv.CorrectsInvoiceDate),
		})
	}
	// Und die Abschlagsrechnungen, die eine Schlussrechnung absetzt: ohne sie
	// bliebe offen, worauf sich der abgesetzte Betrag (BT-113) bezieht.
	for _, ref := range inv.PrecedingRefs {
		if ref.Number == "" || ref.Number == inv.CorrectsInvoiceNumber {
			continue
		}
		doc.PrecedingInvoices = append(doc.PrecedingInvoices, einvoice.PrecedingInvoice{
			Number:    ref.Number,
			IssueDate: einvoice.NewDate(ref.Date),
		})
	}

	// BG-16: die Zahlungsanweisung. Sie steht nur da, wo sie ausführbar ist —
	// eine Überweisung ohne IBAN ist keine Anweisung, sondern eine Zeile.
	if seller.IBAN != "" {
		doc.PaymentMeans = []einvoice.PaymentMeans{{
			TypeCode: "30", // Überweisung (UNTDID 4461)
			CreditTransfer: []einvoice.CreditTransfer{{
				AccountID:   seller.IBAN,
				AccountName: seller.CompanyName,
				ProviderID:  seller.BIC,
			}},
			RemittanceInfo: inv.InvoiceNumber,
		}}
	}

	category := vatCategoryCode(inv.TaxTreatment)
	reason := exemptionReason(inv.TaxTreatment)

	for i := range inv.Items {
		line, err := ciiLine(inv, &inv.Items[i], category)
		if err != nil {
			return nil, err
		}
		doc.Lines = append(doc.Lines, line)
	}

	for _, g := range inv.TaxGroups() {
		rate := g.Rate
		if inv.TaxTreatment != domain.TaxTreatmentDomestic {
			rate = domain.TaxRateNone
		}
		doc.VATBreakdown = append(doc.VATBreakdown, einvoice.VATBreakdown{
			TypeCode:        "VAT",
			TaxableAmount:   amount(g.Net),
			TaxAmount:       amount(g.Tax),
			CategoryCode:    category,
			Rate:            einvoice.NewAmount(ratePercent(rate)),
			ExemptionReason: reason,
		})
	}

	doc.Totals = einvoice.Totals{
		LineTotal:        amount(inv.NetAmount),
		TaxBasisTotal:    amount(inv.NetAmount),
		TaxTotal:         amount(inv.TaxAmount),
		TaxTotalCurrency: currency,
		TaxTotalCount:    1,
		GrandTotal:       amount(inv.GrossAmount),
		DuePayableAmount: amount(inv.OpenAmount()),
	}
	// BT-113: der bereits gezahlte Betrag. An der Schlussrechnung sind das die
	// verrechneten Anzahlungen — die Angabe, an der § 14 Abs. 5 Satz 2 UStG
	// hängt.
	if inv.PrepaidAmount != 0 {
		doc.Totals.PrepaidAmount = amount(inv.PrepaidAmount)
	}

	return doc, nil
}

// needsDeliveryCountry meldet, ob das Bestimmungsland anzugeben ist.
func needsDeliveryCountry(t domain.TaxTreatment) bool {
	switch t {
	case domain.TaxTreatmentIntraCommunitySupply, domain.TaxTreatmentExport:
		return true
	}
	return false
}

// ciiLine turns one position into a BG-25 line.
func ciiLine(inv *domain.Invoice, item *domain.InvoiceItem, category string) (einvoice.Line, error) {
	unit, ok := domain.ResolveUnitCode(item.Unit)
	if !ok {
		return einvoice.Line{}, fmt.Errorf(
			"Position %d: %q ist keine bekannte Mengeneinheit (BT-130, UN/ECE Rec. 20)", item.Position, item.Unit)
	}
	rate := item.TaxRate
	if inv.TaxTreatment != domain.TaxTreatmentDomestic {
		rate = domain.TaxRateNone
	}
	return einvoice.Line{
		ID:        fmt.Sprintf("%d", item.Position),
		Quantity:  einvoice.NewAmount(quantityString(item.QuantityMilli)),
		UnitCode:  unit,
		NetAmount: amount(item.TotalNet()),
		Price:     einvoice.Price{NetPrice: amount(item.UnitPrice)},
		VAT: einvoice.LineVAT{
			CategoryCode: category,
			Rate:         einvoice.NewAmount(ratePercent(rate)),
		},
		Item: einvoice.Item{Name: item.Description},
	}, nil
}

// sellerParty maps the company settings to BG-4.
//
// BT-31 und BT-32 stehen beide da, wo sie beide vorliegen: § 14 Abs. 4 Nr. 2
// UStG lässt die Wahl zwischen Steuernummer und USt-IdNr., aber EN 16931
// verlangt für die maschinelle Zuordnung des Lieferanten eine Kennung aus
// BT-29, BT-30 oder BT-31 (BR-CO-26) — die Steuernummer allein genügt dafür
// nicht. Wo keine USt-IdNr. vorliegt, tritt deshalb die Registernummer als
// BT-30 ein.
func sellerParty(seller *domain.CompanySettings) einvoice.Party {
	postCode, city := splitZipCity(seller.ZipCity)
	party := einvoice.Party{
		Name:            seller.CompanyName,
		VATIdentifier:   seller.VatID,
		TaxRegistration: seller.TaxNumber,
		Address: &einvoice.Address{
			LineOne:     seller.Street,
			PostCode:    postCode,
			City:        city,
			CountryCode: "DE",
		},
	}
	if seller.VatID == "" && seller.RegisterNumber != "" {
		party.LegalRegistration = einvoice.Identifier{Value: seller.RegisterNumber}
	}
	if seller.ContactName != "" || seller.ContactPhone != "" || seller.ContactEmail != "" {
		party.Contact = &einvoice.Contact{
			Name:  seller.ContactName,
			Phone: seller.ContactPhone,
			Email: seller.ContactEmail,
		}
	}
	if seller.ContactEmail != "" {
		party.ElectronicAddress = einvoice.Identifier{Value: seller.ContactEmail, Scheme: "EM"}
	}
	return party
}

// buyerParty maps the contact to BG-7.
//
// Ohne erfassten Kontakt bleibt nur der Name, den die Rechnung selbst trägt.
// Trägt sie keinen, entsteht hier kein Datensatz: EN 16931 verlangt den Namen
// des Erwerbers (BR-07), und einen zu erfinden hieße, eine Pflichtangabe zu
// behaupten. Der Fall ist die Kleinbetragsrechnung ohne Empfänger, und sie geht
// als PDF hinaus — siehe BuildCII.
func buyerParty(inv *domain.Invoice, buyer *domain.Contact) einvoice.Party {
	if buyer == nil {
		return einvoice.Party{
			Name:    inv.ContactName,
			Address: &einvoice.Address{CountryCode: "DE"},
		}
	}
	street, postCode, city := buyer.PostalAddress()
	party := einvoice.Party{
		Name: buyer.Name,
		Address: &einvoice.Address{
			LineOne:     street,
			PostCode:    postCode,
			City:        city,
			CountryCode: countryOrDE(buyer.CountryCode),
		},
	}
	// BT-48: die USt-IdNr. des Erwerbers. Sie gehört auf jede Rechnung, auf der
	// sie erfasst ist, und ist Pflicht bei innergemeinschaftlicher Lieferung und
	// § 13b-Fällen (§ 14a Abs. 1 und 3 UStG).
	party.VATIdentifier = buyer.VatID
	if buyer.Email != "" {
		party.ElectronicAddress = einvoice.Identifier{Value: buyer.Email, Scheme: "EM"}
	}
	return party
}

// splitZipCity reads "80331 München" into its two halves. The company address
// is stored as one line; EN 16931 wants BT-38 and BT-37 apart.
func splitZipCity(zipCity string) (postCode, city string) {
	fields := strings.Fields(strings.TrimSpace(zipCity))
	if len(fields) < 2 {
		return "", strings.TrimSpace(zipCity)
	}
	if isDigits(fields[0]) {
		return fields[0], strings.Join(fields[1:], " ")
	}
	return "", strings.TrimSpace(zipCity)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func amount(c domain.Cents) einvoice.Amount {
	return einvoice.AmountFromCents(einvoice.Cents(c))
}

// RenderInvoiceXML builds the structured record and checks it against the rule
// sets it invokes.
//
// Geprüft wird das eigene Erzeugnis, bevor es hinausgeht. Eine Rechnung, die
// EN 16931 verfehlt, kostet den Empfänger den Vorsteuerabzug, und er merkt es
// erst bei seiner Prüfung — die Meldung an den Aussteller ist um Jahre
// billiger. Zurückgegeben wird auch das Prüfergebnis: es gehört als
// Validierungsbericht an den Beleg.
func RenderInvoiceXML(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact, profile domain.EInvoiceProfile) (string, domain.ReceiptValidation, error) {
	doc, err := BuildCII(inv, seller, buyer, profile)
	if err != nil {
		return "", domain.ReceiptValidation{}, err
	}
	xml, err := einvoice.RenderCII(doc)
	if err != nil {
		return "", domain.ReceiptValidation{}, fmt.Errorf("der Rechnungsdatensatz konnte nicht geschrieben werden: %w", err)
	}

	// Gelesen und dann geprüft, nicht das gebaute Modell geprüft: was der
	// Empfänger bekommt, ist die Datei — und nur was sich aus ihr wieder lesen
	// lässt, ist geprüft.
	parsed, err := einvoice.ParseCII(xml)
	if err != nil {
		return "", domain.ReceiptValidation{}, fmt.Errorf("der erzeugte Rechnungsdatensatz ist nicht lesbar: %w", err)
	}
	result := ruleset.Validate(parsed)
	// Der Bericht entsteht in derselben Form wie beim eingehenden Beleg: eine
	// zweite Fassung davon wäre die, die beim nächsten Regelwerk vergessen wird.
	report := validationOf(parsed, result)
	if !result.Valid() {
		return string(xml), report, fmt.Errorf(
			"die Rechnung erfüllt %s noch nicht: %s. Bitte die fehlenden Angaben ergänzen",
			rulesetLabel(profile), strings.Join(errorMessages(result), "; "))
	}
	return string(xml), report, nil
}

func rulesetLabel(profile domain.EInvoiceProfile) string {
	if profile == domain.EInvoiceProfileXRechnungCII {
		return "EN 16931 mit der deutschen Ausprägung (XRechnung)"
	}
	return "EN 16931"
}
