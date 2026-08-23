package einvoice

import "strings"

// mimeCodes is the short list EN 16931 admits for an embedded attachment
// (BR-CL-24). It is deliberately narrow: a recipient has to be able to open the
// file without installing anything, and an executable has no business inside an
// invoice.
var mimeCodes = codeSet(
	"application/pdf image/png image/jpeg text/csv " +
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet " +
		"application/vnd.oasis.opendocument.spreadsheet")

// checkCodeLists runs the BR-CL family: every coded field has to come from the
// list the standard names for it.
//
// This is not bureaucracy. A code outside its list is a value the recipient
// cannot interpret — a unit of measure nobody can convert, a payment means
// nobody can execute, a tax category that maps to no account.
func (v *validator) checkCodeLists() {
	inv := v.inv

	v.inList("BR-CL-01", inv.TypeCode, untdid1001, "Der Rechnungstyp (BT-3)")
	v.inList("BR-CL-03", inv.Totals.TaxTotalCurrency, iso4217,
		"Die Währung des Steuergesamtbetrags (BT-110)")
	v.inList("BR-CL-03", inv.Totals.TaxTotalInTaxCurrCurrency, iso4217,
		"Die Währung des Steuergesamtbetrags in der Abrechnungswährung (BT-111)")
	v.inList("BR-CL-04", inv.Currency, iso4217, "Die Rechnungswährung (BT-5)")
	v.inList("BR-CL-05", inv.TaxCurrency, iso4217, "Die Abrechnungswährung (BT-6)")
	v.inList("BR-CL-06", inv.TaxPointDateCode, untdid2005, "Der Schlüssel des Steuerzeitpunkts (BT-8)")
	v.inList("BR-CL-07", inv.ObjectIdentifier.Scheme, untdid1153, "Das Schema der Objektkennung (BT-18-1)")
	for i, line := range inv.Lines {
		v.inListAt("BR-CL-07", linePos(i), line.ObjectIdentifier.Scheme, untdid1153,
			"Das Schema der Objektkennung der Position (BT-128-1)")
	}

	for _, note := range inv.Notes {
		v.inList("BR-CL-08", note.SubjectCode, untdid4451, "Der Betreff-Code einer Bemerkung (BT-21)")
	}

	// BR-CL-10 und BR-CL-11: die Kennungsschemata der Beteiligten. Die Norm
	// unterscheidet dabei nach der Art der Kennung, nicht nach dem Beteiligten.
	for _, party := range v.allParties() {
		for _, id := range party.Identifiers {
			v.inList("BR-CL-10", id.Scheme, iso6523, "Das Schema einer Kennung (BT-29-1, BT-46-1, BT-60-1)")
		}
		v.inList("BR-CL-11", party.LegalRegistration.Scheme, iso6523,
			"Das Schema der Registernummer (BT-30-1, BT-47-1, BT-61-1)")
		v.inList("BR-CL-25", party.ElectronicAddress.Scheme, eas,
			"Das Schema der elektronischen Adresse (BT-34-1, BT-49-1)")
		if party.Address != nil {
			v.inList("BR-CL-14", party.Address.CountryCode, iso3166, "Ein Länderkennzeichen")
		}
	}
	if d := inv.Delivery; d != nil {
		if d.Address != nil {
			v.inList("BR-CL-14", d.Address.CountryCode, iso3166, "Das Bestimmungsland (BT-80)")
		}
		v.inList("BR-CL-26", d.LocationID.Scheme, iso6523, "Das Schema der Kennung des Lieferorts (BT-71-1)")
	}

	for _, means := range inv.PaymentMeans {
		v.inList("BR-CL-16", means.TypeCode, untdid4461, "Das Zahlungsmittel (BT-81)")
	}

	for i, a := range inv.Allowances {
		where := "Nachlass " + itoa(i+1) + " auf Dokumentebene"
		v.inListAt("BR-CL-17", where, a.VATCategory, untdid5305, "Die Steuerkategorie (BT-95)")
		v.inListAt("BR-CL-19", where, a.ReasonCode, uncl5189, "Der Schlüssel des Grundes (BT-98)")
	}
	for i, a := range inv.Charges {
		where := "Zuschlag " + itoa(i+1) + " auf Dokumentebene"
		v.inListAt("BR-CL-17", where, a.VATCategory, untdid5305, "Die Steuerkategorie (BT-102)")
		v.inListAt("BR-CL-20", where, a.ReasonCode, uncl7161, "Der Schlüssel des Grundes (BT-105)")
	}

	for i, group := range inv.VATBreakdown {
		where := breakdownPos(i)
		v.inListAt("BR-CL-18", where, group.CategoryCode, untdid5305, "Die Steuerkategorie (BT-118)")
		v.inListAt("BR-CL-22", where, group.ExemptionReasonCode, vatex, "Der Befreiungsgrund-Schlüssel (BT-121)")
	}

	for i, line := range inv.Lines {
		where := linePos(i)
		// Die Steuerkategorie der Position läuft in beiden Syntaxen unter
		// BR-CL-18; die der Aufschlüsselung führt CII ebenfalls dort, UBL
		// dagegen unter BR-CL-17. Geprüft wird dieselbe Liste, gemeldet wird
		// unter der CII-Kennung — der Fund ist derselbe, nur sein Etikett
		// könnte in einem UBL-Prüfbericht anders lauten.
		v.inListAt("BR-CL-18", where, line.VAT.CategoryCode, untdid5305,
			"Die Steuerkategorie der Position (BT-151)")
		v.inListAt("BR-CL-23", where, line.UnitCode, recommendation20, "Die Mengeneinheit (BT-130)")
		v.inListAt("BR-CL-23", where, line.Price.BaseUnit, recommendation20,
			"Die Einheit der Preisbasismenge (BT-150)")
		v.inListAt("BR-CL-15", where, line.Item.OriginCountryCode, iso3166, "Das Ursprungsland (BT-159)")
		v.inListAt("BR-CL-21", where, line.Item.StandardID.Scheme, iso6523,
			"Das Schema der Standardkennung (BT-157-1)")
		for _, c := range line.Item.Classifications {
			v.inListAt("BR-CL-13", where, c.Scheme, untdid7143,
				"Das Schema der Artikelklassifizierung (BT-158-1)")
		}
		for _, a := range append(append([]AllowanceCharge{}, line.Allowances...), line.Charges...) {
			v.inListAt("BR-CL-17", where, a.VATCategory, untdid5305, "Die Steuerkategorie einer Position")
		}
		for j, a := range line.Allowances {
			v.inListAt("BR-CL-19", where+", Nachlass "+itoa(j+1), a.ReasonCode, uncl5189,
				"Der Schlüssel des Grundes (BT-140)")
		}
		for j, a := range line.Charges {
			v.inListAt("BR-CL-20", where+", Zuschlag "+itoa(j+1), a.ReasonCode, uncl7161,
				"Der Schlüssel des Grundes (BT-145)")
		}
	}

	for i, doc := range inv.SupportingDocs {
		if len(doc.Attachment) == 0 && strings.TrimSpace(doc.MimeCode) == "" {
			continue
		}
		v.inListAt("BR-CL-24", "Unterlage "+itoa(i+1), doc.MimeCode, mimeCodes,
			"Der Dateityp der Anlage (BT-125-1)")
	}
}

// allParties returns every party the document names, so a code list rule that
// applies to "a party" does not have to be written out four times.
func (v *validator) allParties() []Party {
	parties := []Party{v.inv.Seller, v.inv.Buyer}
	if v.inv.Payee != nil {
		parties = append(parties, *v.inv.Payee)
	}
	if v.inv.TaxRepresentative != nil {
		parties = append(parties, *v.inv.TaxRepresentative)
	}
	return parties
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
