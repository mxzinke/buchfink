package einvoice

import (
	"fmt"
	"strings"
)

// The business rules of EN 16931, grouped by the context the standard's own
// abstract rule set hangs them off. The order follows that rule set, so a
// reader with the norm open can find each rule where they expect it.

// checkDocument covers the rules the standard attaches to the invoice itself.
func (v *validator) checkDocument() {
	inv := v.inv

	v.require("BR-01", inv.SpecificationID, "Die Kennung der Spezifikation (BT-24)")
	v.require("BR-02", inv.Number, "Die Rechnungsnummer (BT-1)")
	v.require("BR-03", inv.IssueDate.Raw(), "Das Rechnungsdatum (BT-2)")
	v.require("BR-04", inv.TypeCode, "Der Rechnungstyp (BT-3)")
	v.require("BR-05", inv.Currency, "Der Währungscode (BT-5)")

	if len(inv.Lines) == 0 {
		v.fail("BR-16", "Die Rechnung enthält keine Rechnungsposition (BG-25)")
	}
	if len(inv.VATBreakdowns()) == 0 {
		v.fail("BR-CO-18", "Die Rechnung enthält keine Aufschlüsselung der Umsatzsteuer (BG-23)")
	}

	// BR-53: rechnet der Aussteller die Steuer in einer zweiten Währung ab,
	// muss er sie dort auch beziffern. Sonst steht die Abrechnungswährung da,
	// ohne dass jemand den Betrag kennt.
	if inv.TaxCurrency != "" && !strings.EqualFold(inv.TaxCurrency, inv.Currency) &&
		!inv.Totals.TaxTotalInTaxCurr.Present() {
		v.fail("BR-53", "Die Rechnung nennt %s als Abrechnungswährung (BT-6), aber keinen Steuerbetrag darin (BT-111)",
			inv.TaxCurrency)
	}

	// BR-CO-03: Steuerzeitpunkt entweder als Datum oder als Schlüssel.
	if inv.TaxPointDate.Present() && inv.TaxPointDateCode != "" {
		v.fail("BR-CO-03", "Der Steuerzeitpunkt ist doppelt angegeben: als Datum (BT-7) und als Schlüssel (BT-8)")
	}

	v.decimals("BR-DEC-13", inv.Totals.TaxTotal, "Der Gesamtbetrag der Umsatzsteuer (BT-110)")
	v.decimals("BR-DEC-15", inv.Totals.TaxTotalInTaxCurr,
		"Der Gesamtbetrag der Umsatzsteuer in der Abrechnungswährung (BT-111)")
}

// checkParties covers seller, buyer, payee and tax representative.
func (v *validator) checkParties() {
	inv := v.inv

	v.require("BR-06", inv.Seller.Name, "Der Name des Verkäufers (BT-27)")
	v.require("BR-07", inv.Buyer.Name, "Der Name des Erwerbers (BT-44)")

	// BR-08 und BR-10 fragen nach dem Vorhandensein der Anschrift, nicht nach
	// ihrem Inhalt. Eine Anschrift, die nur aus dem Land besteht, ist gültig —
	// dafür sind BR-09 und BR-11 zuständig, und das sind eigene Regeln.
	if inv.Seller.Address == nil {
		v.fail("BR-08", "Die Anschrift des Verkäufers (BG-5) fehlt")
	} else {
		v.require("BR-09", inv.Seller.Address.CountryCode, "Das Länderkennzeichen des Verkäufers (BT-40)")
	}
	if inv.Buyer.Address == nil {
		v.fail("BR-10", "Die Anschrift des Erwerbers (BG-8) fehlt")
	} else {
		v.require("BR-11", inv.Buyer.Address.CountryCode, "Das Länderkennzeichen des Erwerbers (BT-55)")
	}

	// BR-CO-26: ohne Kennung ist der Lieferant maschinell nicht zuzuordnen.
	if !inv.Seller.Identified() {
		v.fail("BR-CO-26", "Der Verkäufer trägt keine Kennung: weder eine Nummer (BT-29), noch eine Registernummer (BT-30), noch eine USt-IdNr. (BT-31)")
	}

	// BR-62 und BR-63: eine elektronische Adresse ohne Schema ist nicht
	// zustellbar — "12345" sagt nicht, ob eine GLN oder eine Leitweg-ID gemeint
	// ist.
	if inv.Seller.ElectronicAddress.Present() && inv.Seller.ElectronicAddress.Scheme == "" {
		v.fail("BR-62", "Die elektronische Adresse des Verkäufers (BT-34) nennt kein Schema")
	}
	if inv.Buyer.ElectronicAddress.Present() && inv.Buyer.ElectronicAddress.Scheme == "" {
		v.fail("BR-63", "Die elektronische Adresse des Erwerbers (BT-49) nennt kein Schema")
	}

	// BR-CO-09: die USt-IdNr. beginnt mit dem Länderkennzeichen des
	// ausstellenden Staates. Hier zeigt sich eine vertauschte oder
	// abgeschnittene Nummer noch vor der Buchung.
	v.checkVATPrefix(inv.Seller.VATIdentifier, "des Verkäufers (BT-31)")
	v.checkVATPrefix(inv.Buyer.VATIdentifier, "des Erwerbers (BT-48)")
	if inv.TaxRepresentative != nil {
		v.checkVATPrefix(inv.TaxRepresentative.VATIdentifier, "des Steuervertreters (BT-63)")
	}

	// BR-17: ein vom Verkäufer verschiedener Zahlungsempfänger braucht einen
	// Namen. Ohne ihn weiß der Erwerber nicht, an wen er zahlen soll.
	if payee := inv.Payee; payee != nil && v.payeeDiffersFromSeller(payee) && payee.Name == "" {
		v.fail("BR-17", "Der Zahlungsempfänger (BG-10) weicht vom Verkäufer ab, trägt aber keinen Namen (BT-59)")
	}

	// BR-18 bis BR-20 und BR-56: der Steuervertreter ist vollständig zu nennen
	// oder gar nicht.
	if rep := inv.TaxRepresentative; rep != nil {
		v.require("BR-18", rep.Name, "Der Name des Steuervertreters (BT-62)")
		if rep.Address == nil {
			v.fail("BR-19", "Die Anschrift des Steuervertreters (BG-12) fehlt")
		} else {
			v.require("BR-20", rep.Address.CountryCode, "Das Länderkennzeichen des Steuervertreters (BT-69)")
		}
		v.require("BR-56", rep.VATIdentifier, "Die USt-IdNr. des Steuervertreters (BT-63)")
	}
}

// payeeDiffersFromSeller mirrors what the standard compares: name, identifier
// and legal registration. A payee repeating the seller is not a separate party.
func (v *validator) payeeDiffersFromSeller(payee *Party) bool {
	seller := v.inv.Seller
	if payee.Name != "" && payee.Name == seller.Name {
		return false
	}
	if payee.LegalRegistration.Present() &&
		payee.LegalRegistration.Value == seller.LegalRegistration.Value {
		return false
	}
	for _, a := range payee.Identifiers {
		for _, b := range seller.Identifiers {
			if a.Value != "" && a.Value == b.Value {
				return false
			}
		}
	}
	return true
}

// checkVATPrefix implements BR-CO-09.
//
// The standard's list is ISO 3166-1 alpha-2 plus two entries that are not in
// it: Greece issues its VAT identifiers with "EL", and "1A" is reserved for
// Kosovo.
func (v *validator) checkVATPrefix(id, whose string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	if len(id) < 2 {
		v.fail("BR-CO-09", "Die USt-IdNr. %s ist mit %q zu kurz für ein Länderkennzeichen", whose, id)
		return
	}
	prefix := strings.ToUpper(id[:2])
	if prefix == "EL" || prefix == "1A" || inCodeList(prefix, iso3166) {
		return
	}
	v.fail("BR-CO-09", "Die USt-IdNr. %s beginnt mit %q, was kein Länderkennzeichen nach ISO 3166-1 ist", whose, prefix)
}

// checkDelivery covers BR-57: a delivery address needs a country.
func (v *validator) checkDelivery() {
	d := v.inv.Delivery
	if d == nil || d.Address == nil {
		return
	}
	if strings.TrimSpace(d.Address.CountryCode) == "" {
		v.fail("BR-57", "Die Lieferanschrift (BG-15) nennt kein Bestimmungsland (BT-80)")
	}
}

// checkPeriods covers BR-29, BR-30, BR-CO-19 and BR-CO-20.
func (v *validator) checkPeriods() {
	if p := v.inv.Period; p != nil {
		// Ein Zeitraum, der allein den Schlüssel des Steuerzeitpunkts trägt,
		// ist zulässig: der Aussteller sagt damit, woran die Steuer hängt, ohne
		// einen Zeitraum zu behaupten.
		if !p.Start.Present() && !p.End.Present() && strings.TrimSpace(p.DescriptionCode) == "" {
			v.fail("BR-CO-19", "Der angegebene Rechnungszeitraum (BG-14) nennt weder Beginn (BT-73) noch Ende (BT-74)")
		}
		if p.Start.Present() && p.End.Present() && p.End.Before(p.Start) {
			v.fail("BR-29", "Der Rechnungszeitraum endet am %s und beginnt am %s", p.End, p.Start)
		}
	}
	for i, line := range v.inv.Lines {
		p := line.Period
		if p == nil {
			continue
		}
		where := linePos(i)
		if !p.Start.Present() && !p.End.Present() {
			v.failAt("BR-CO-20", where, "Der angegebene Zeitraum der Position (BG-26) nennt weder Beginn (BT-134) noch Ende (BT-135)")
		}
		if p.Start.Present() && p.End.Present() && p.End.Before(p.Start) {
			v.failAt("BR-30", where, "Der Zeitraum endet am %s und beginnt am %s", p.End, p.Start)
		}
	}
}

// checkPaymentInstructions covers BR-49, BR-50, BR-51 and BR-61.
func (v *validator) checkPaymentInstructions() {
	// creditTransferCodes are the UNTDID 4461 codes that mean a transfer: SEPA,
	// local and non-SEPA international credit transfer. For those, BR-61
	// requires the account, because a transfer instruction without an account
	// cannot be carried out.
	creditTransferCodes := map[string]bool{"30": true, "58": true, "59": true}

	for i, means := range v.inv.PaymentMeans {
		where := fmt.Sprintf("Zahlungsanweisung %d", i+1)

		v.requireAt("BR-49", where, means.TypeCode, "Das Zahlungsmittel (BT-81)")

		for _, account := range means.CreditTransfer {
			if strings.TrimSpace(account.AccountID) == "" {
				v.failAt("BR-50", where, "Das Zahlungskonto (BT-84) nennt keine Kennung")
			}
		}
		if creditTransferCodes[strings.TrimSpace(means.TypeCode)] {
			hasAccount := false
			for _, account := range means.CreditTransfer {
				if strings.TrimSpace(account.AccountID) != "" {
					hasAccount = true
				}
			}
			if !hasAccount {
				v.failAt("BR-61", where,
					"Das Zahlungsmittel %q ist eine Überweisung, die Rechnung nennt aber kein Zahlungskonto (BT-84)",
					means.TypeCode)
			}
		}

		// BR-51 ist die einzige Regel, die die Norm als Hinweis führt und nicht
		// als Fehler — sie schützt den Erwerber, nicht die Rechnung.
		if card := means.Card; card != nil {
			if n := len(strings.TrimSpace(card.PrimaryAccountNumber)); n > 10 {
				v.failAt("BR-51", where,
					"Die Kartennummer (BT-87) steht mit %d Zeichen in der Rechnung; zulässig sind höchstens die ersten sechs und die letzten vier Stellen",
					n)
			}
		}
	}
}

// checkSupportingDocuments covers BR-52.
func (v *validator) checkSupportingDocuments() {
	for i, doc := range v.inv.SupportingDocs {
		where := fmt.Sprintf("Unterlage %d", i+1)
		v.requireAt("BR-52", where, doc.Reference, "Die Kennung der rechnungsbegründenden Unterlage (BT-122)")
	}
}

// checkPrecedingInvoices covers BR-55.
func (v *validator) checkPrecedingInvoices() {
	for i, ref := range v.inv.PrecedingInvoices {
		where := fmt.Sprintf("Rechnungsbezug %d", i+1)
		v.requireAt("BR-55", where, ref.Number, "Die Nummer der vorausgegangenen Rechnung (BT-25)")
	}
}

// checkAllowancesCharges covers the Nachlässe and Zuschläge on both levels:
// BR-31 to BR-33, BR-36 to BR-38, BR-41 to BR-44, BR-CO-05 to BR-CO-08,
// BR-CO-21 to BR-CO-24 and their decimal limits.
func (v *validator) checkAllowancesCharges() {
	for i, a := range v.inv.Allowances {
		where := fmt.Sprintf("Nachlass %d auf Dokumentebene", i+1)
		v.requireAt("BR-31", where, a.Amount.String(), "Der Betrag des Nachlasses (BT-92)")
		v.requireAt("BR-32", where, a.VATCategory, "Die Steuerkategorie des Nachlasses (BT-95)")
		if !a.HasReason() {
			v.failAt("BR-33", where, "Es fehlt der Grund (BT-97) oder der Grundschlüssel (BT-98)")
			v.failAt("BR-CO-21", where, "Es fehlt der Grund (BT-97) oder der Grundschlüssel (BT-98)")
		}
		v.decimalsAt("BR-DEC-01", where, a.Amount, "Der Betrag des Nachlasses (BT-92)")
		v.decimalsAt("BR-DEC-02", where, a.BaseAmount, "Der Grundbetrag des Nachlasses (BT-93)")
	}

	for i, a := range v.inv.Charges {
		where := fmt.Sprintf("Zuschlag %d auf Dokumentebene", i+1)
		v.requireAt("BR-36", where, a.Amount.String(), "Der Betrag des Zuschlags (BT-99)")
		v.requireAt("BR-37", where, a.VATCategory, "Die Steuerkategorie des Zuschlags (BT-102)")
		if !a.HasReason() {
			v.failAt("BR-38", where, "Es fehlt der Grund (BT-104) oder der Grundschlüssel (BT-105)")
			v.failAt("BR-CO-22", where, "Es fehlt der Grund (BT-104) oder der Grundschlüssel (BT-105)")
		}
		v.decimalsAt("BR-DEC-05", where, a.Amount, "Der Betrag des Zuschlags (BT-99)")
		v.decimalsAt("BR-DEC-06", where, a.BaseAmount, "Der Grundbetrag des Zuschlags (BT-100)")
	}

	for i, line := range v.inv.Lines {
		for j, a := range line.Allowances {
			where := fmt.Sprintf("%s, Nachlass %d", linePos(i), j+1)
			v.requireAt("BR-41", where, a.Amount.String(), "Der Betrag des Nachlasses (BT-136)")
			if !a.HasReason() {
				v.failAt("BR-42", where, "Es fehlt der Grund (BT-139) oder der Grundschlüssel (BT-140)")
				v.failAt("BR-CO-23", where, "Es fehlt der Grund (BT-139) oder der Grundschlüssel (BT-140)")
			}
			v.decimalsAt("BR-DEC-24", where, a.Amount, "Der Betrag des Nachlasses (BT-136)")
			v.decimalsAt("BR-DEC-25", where, a.BaseAmount, "Der Grundbetrag des Nachlasses (BT-137)")
		}
		for j, a := range line.Charges {
			where := fmt.Sprintf("%s, Zuschlag %d", linePos(i), j+1)
			v.requireAt("BR-43", where, a.Amount.String(), "Der Betrag des Zuschlags (BT-141)")
			if !a.HasReason() {
				v.failAt("BR-44", where, "Es fehlt der Grund (BT-144) oder der Grundschlüssel (BT-145)")
				v.failAt("BR-CO-24", where, "Es fehlt der Grund (BT-144) oder der Grundschlüssel (BT-145)")
			}
			v.decimalsAt("BR-DEC-27", where, a.Amount, "Der Betrag des Zuschlags (BT-141)")
			v.decimalsAt("BR-DEC-28", where, a.BaseAmount, "Der Grundbetrag des Zuschlags (BT-142)")
		}
	}

	// BR-CO-05 bis BR-CO-08 verlangen, dass Grundschlüssel und Grundtext
	// dasselbe bedeuten. Das ist ein Vergleich zwischen einem Schlüssel und
	// freiem Text und maschinell nicht entscheidbar — der Referenzprüfer der
	// Norm führt diese vier Regeln als `true()`. Buchfink hält es genauso und
	// sagt es, statt eine Prüfung zu behaupten, die keine ist.
}

// checkLines covers BR-21 to BR-28, BR-CO-04, BR-DEC-23 and the item rules
// BR-54, BR-64 and BR-65.
func (v *validator) checkLines() {
	for i, line := range v.inv.Lines {
		where := linePos(i)

		v.requireAt("BR-21", where, line.ID, "Die Positionsnummer (BT-126)")
		v.requireAt("BR-22", where, line.Quantity.String(), "Die Menge (BT-129)")
		v.requireAt("BR-23", where, line.UnitCode, "Die Mengeneinheit (BT-130)")
		v.requireAt("BR-25", where, line.Item.Name, "Die Artikelbezeichnung (BT-153)")
		v.requireAt("BR-26", where, line.Price.NetPrice.String(), "Der Nettopreis (BT-146)")
		v.requireAt("BR-CO-04", where, line.VAT.CategoryCode, "Die Steuerkategorie (BT-151)")

		if line.NetAmount.Present() {
			if _, err := line.NetAmount.Cents(); err != nil {
				v.failAt("BR-24", where, "Der Nettobetrag (BT-131) ist mit %q unlesbar", line.NetAmount)
			}
		} else {
			v.failAt("BR-24", where, "Der Nettobetrag (BT-131) fehlt")
		}
		v.decimalsAt("BR-DEC-23", where, line.NetAmount, "Der Nettobetrag (BT-131)")

		// BR-27 und BR-28: ein negativer Preis ist keine Gutschrift. Eine
		// Gutschrift ist ein eigener Rechnungstyp (BT-3), und wer sie über ein
		// Minuszeichen im Preis abbildet, bekommt in jeder Auswertung das
		// falsche Vorzeichen.
		if line.Price.NetPrice.Sign() < 0 {
			v.failAt("BR-27", where, "Der Nettopreis (BT-146) ist mit %s negativ", line.Price.NetPrice)
		}
		if line.Price.GrossPrice.Present() && line.Price.GrossPrice.Sign() < 0 {
			v.failAt("BR-28", where, "Der Bruttopreis (BT-148) ist mit %s negativ", line.Price.GrossPrice)
		}

		for j, attr := range line.Item.Attributes {
			attrWhere := fmt.Sprintf("%s, Artikelattribut %d", where, j+1)
			if strings.TrimSpace(attr.Name) == "" || strings.TrimSpace(attr.Value) == "" {
				v.failAt("BR-54", attrWhere, "Ein Artikelattribut braucht Bezeichnung (BT-160) und Wert (BT-161)")
			}
		}
		if line.Item.StandardID.Present() && line.Item.StandardID.Scheme == "" {
			v.failAt("BR-64", where, "Die Standardkennung des Artikels (BT-157) nennt kein Schema")
		}
		for _, c := range line.Item.Classifications {
			if c.Present() && c.Scheme == "" {
				v.failAt("BR-65", where, "Die Artikelklassifizierung (BT-158) nennt kein Schema")
			}
		}
	}
}

// checkVATBreakdown covers BR-45 to BR-48, BR-CO-17 and the decimal limits of
// the breakdown.
func (v *validator) checkVATBreakdown() {
	for i, group := range v.inv.VATBreakdown {
		if !isVATTypeCode(group.TypeCode) {
			continue
		}
		where := breakdownPos(i)
		category := normaliseCategory(group.CategoryCode)

		v.requireAt("BR-45", where, group.TaxableAmount.String(), "Die Bemessungsgrundlage (BT-116)")
		v.requireAt("BR-46", where, group.TaxAmount.String(), "Der Steuerbetrag (BT-117)")
		v.requireAt("BR-47", where, group.CategoryCode, "Der Steuerkategoriecode (BT-118)")
		if category != categoryNotSubject {
			v.requireAt("BR-48", where, group.Rate.String(), "Der Steuersatz (BT-119)")
		}

		v.decimalsAt("BR-DEC-19", where, group.TaxableAmount, "Die Bemessungsgrundlage (BT-116)")
		v.decimalsAt("BR-DEC-20", where, group.TaxAmount, "Der Steuerbetrag (BT-117)")

		v.checkTaxFollowsFromRate("BR-CO-17", where, group)
	}
}

// checkTaxFollowsFromRate implements BR-CO-17 and the per-category -09 rules
// that share its arithmetic.
//
// The standard allows one currency unit of slack. That is not sloppiness:
// currencies without decimal places and systems that round per line rather than
// per group land a cent beside the exact result often enough that an exact
// comparison would report correct invoices as broken.
func (v *validator) checkTaxFollowsFromRate(rule, where string, group VATBreakdown) {
	base, errBase := group.TaxableAmount.Cents()
	amount, errAmount := group.TaxAmount.Cents()
	if errBase != nil || errAmount != nil {
		return
	}
	if !group.Rate.Present() {
		// Ohne Satz muss der Steuerbetrag null sein.
		if amount.Abs() >= roundsToZero {
			v.report(rule, where, "Die Steuergruppe nennt keinen Satz, weist aber %s Steuer aus", amount)
		}
		return
	}
	expected, ok := MulPercent(base.Abs(), group.Rate)
	if !ok {
		return
	}
	if diff := (amount.Abs() - expected).Abs(); diff >= vatRoundingTolerance {
		v.report(rule, where, "Aus %s zu %s %% folgt ein Steuerbetrag von %s, angegeben ist %s",
			base, group.Rate, expected, amount)
	}
}

// vatRoundingTolerance is the one currency unit EN 16931 allows when checking a
// tax amount against base times rate.
//
// It is not sloppiness: currencies without decimal places, and systems that
// round per line rather than per group, land a cent beside the exact result
// often enough that an exact comparison would report correct invoices as
// broken.
//
// The comparison is strict — a difference of exactly one unit is a violation.
// The two syntax bindings of the standard disagree here by a hair: for BR-CO-17
// the CII binding compares inclusively while the UBL one and every per-category
// rule in both syntaxes compare strictly. Buchfink takes the strict reading, so
// that the same document is judged the same way whichever syntax it arrives in.
const vatRoundingTolerance = Cents(100)

// roundsToZero is the threshold for "this amount rounds to zero" — what the
// standard requires of the tax amount when the rate itself rounds to zero.
const roundsToZero = Cents(50)

// checkTotals covers BR-12 to BR-15 and the arithmetic of the document:
// BR-CO-10 to BR-CO-16 and the decimal limits of the totals.
func (v *validator) checkTotals() {
	inv := v.inv
	totals := inv.Totals

	lineTotal, okLineTotal := v.requireAmount("BR-12", totals.LineTotal, "Die Summe der Positionsbeträge (BT-106)")
	taxBasis, okBasis := v.requireAmount("BR-13", totals.TaxBasisTotal, "Der Gesamtbetrag ohne Umsatzsteuer (BT-109)")
	grand, okGrand := v.requireAmount("BR-14", totals.GrandTotal, "Der Gesamtbetrag mit Umsatzsteuer (BT-112)")
	due, okDue := v.requireAmount("BR-15", totals.DuePayableAmount, "Der fällige Betrag (BT-115)")

	allowanceTotal, okAllowance := optionalCents(totals.AllowanceTotal)
	chargeTotal, okCharge := optionalCents(totals.ChargeTotal)
	prepaid, okPrepaid := optionalCents(totals.PrepaidAmount)
	rounding, okRounding := optionalCents(totals.RoundingAmount)

	v.decimals("BR-DEC-09", totals.LineTotal, "Die Summe der Positionsbeträge (BT-106)")
	v.decimals("BR-DEC-10", totals.AllowanceTotal, "Die Summe der Nachlässe (BT-107)")
	v.decimals("BR-DEC-11", totals.ChargeTotal, "Die Summe der Zuschläge (BT-108)")
	v.decimals("BR-DEC-12", totals.TaxBasisTotal, "Der Gesamtbetrag ohne Umsatzsteuer (BT-109)")
	v.decimals("BR-DEC-14", totals.GrandTotal, "Der Gesamtbetrag mit Umsatzsteuer (BT-112)")
	v.decimals("BR-DEC-16", totals.PrepaidAmount, "Der bereits gezahlte Betrag (BT-113)")
	v.decimals("BR-DEC-17", totals.RoundingAmount, "Der Rundungsbetrag (BT-114)")
	v.decimals("BR-DEC-18", totals.DuePayableAmount, "Der fällige Betrag (BT-115)")

	// BR-CO-10: die Summe der Positionsbeträge muss den Positionen entsprechen.
	if okLineTotal && len(inv.Lines) > 0 {
		if sum, ok := sumLineNetAmounts(inv.Lines); ok && sum != lineTotal {
			v.fail("BR-CO-10", "Die Summe der Positionsbeträge ist %s, die Positionen ergeben aber %s",
				lineTotal, sum)
		}
	}

	// BR-CO-11 und BR-CO-12: die ausgewiesenen Summen der Nachlässe und
	// Zuschläge müssen den einzelnen Beträgen entsprechen.
	if okAllowance {
		if sum, ok := sumAllowanceCharges(inv.Allowances); ok && sum != allowanceTotal {
			v.fail("BR-CO-11", "Die Summe der Nachlässe ist mit %s ausgewiesen, die einzelnen Nachlässe ergeben aber %s",
				allowanceTotal, sum)
		}
	}
	if okCharge {
		if sum, ok := sumAllowanceCharges(inv.Charges); ok && sum != chargeTotal {
			v.fail("BR-CO-12", "Die Summe der Zuschläge ist mit %s ausgewiesen, die einzelnen Zuschläge ergeben aber %s",
				chargeTotal, sum)
		}
	}

	// BR-CO-13: netto = Positionen - Nachlässe + Zuschläge.
	if okLineTotal && okBasis && okAllowance && okCharge {
		if expected := lineTotal - allowanceTotal + chargeTotal; expected != taxBasis {
			v.fail("BR-CO-13", "Der Gesamtbetrag ohne Umsatzsteuer ist %s, aus Positionen, Nachlässen und Zuschlägen folgt aber %s",
				taxBasis, expected)
		}
	}

	// BR-CO-14: die ausgewiesene Gesamtsteuer entspricht der Summe der Gruppen.
	taxSum, taxKnown := sumBreakdownTax(inv.VATBreakdowns())
	if declared, ok := optionalCents(totals.TaxTotal); ok && taxKnown &&
		totals.TaxTotal.Present() && declared != taxSum {
		v.fail("BR-CO-14", "Der Gesamtbetrag der Umsatzsteuer ist mit %s ausgewiesen, die Steuergruppen ergeben aber %s",
			declared, taxSum)
	}

	// BR-CO-15 verlangt zuerst, dass es genau einen Steuergesamtbetrag in der
	// Rechnungswährung gibt. Zwei verschiedene lassen den Empfänger wählen, und
	// was er wählt, ist geraten.
	if totals.TaxTotalCount > 1 {
		v.fail("BR-CO-15", "Die Rechnung weist %d Steuergesamtbeträge in der Rechnungswährung aus; zulässig ist genau einer (BT-110)",
			totals.TaxTotalCount)
	}

	// BR-CO-15: brutto = netto + der ausgewiesene Steuerbetrag.
	//
	// Maßgeblich ist BT-110, nicht die Summe der Steuergruppen. Dass beide
	// übereinstimmen, ist die Aussage von BR-CO-14 — hier die Gruppensumme
	// einzusetzen würde denselben Fehler zweimal melden und den eigentlichen
	// verdecken, wenn BT-110 selbst danebenliegt.
	if okBasis && okGrand {
		declared, ok := optionalCents(totals.TaxTotal)
		if ok {
			if expected := taxBasis + declared; expected != grand {
				v.fail("BR-CO-15", "Der Gesamtbetrag mit Umsatzsteuer ist %s, netto plus ausgewiesene Steuer ergeben aber %s",
					grand, expected)
			}
		}
	}

	// BR-CO-16: fälliger Betrag = brutto - bereits gezahlt + Rundung.
	if okGrand && okDue && okPrepaid && okRounding {
		if expected := grand - prepaid + rounding; expected != due {
			v.fail("BR-CO-16", "Der fällige Betrag ist %s, aus Gesamtbetrag, Anzahlung und Rundung folgt aber %s",
				due, expected)
		}
	}
}

// optionalCents reads a field that may legitimately be absent. Absent counts as
// zero — that is how every sum rule of the standard treats it.
func optionalCents(a Amount) (Cents, bool) {
	if !a.Present() {
		return 0, true
	}
	value, err := a.Cents()
	if err != nil {
		return 0, false
	}
	return value, true
}

func sumLineNetAmounts(lines []Line) (Cents, bool) {
	var total Cents
	for _, l := range lines {
		amount, err := l.NetAmount.Cents()
		if err != nil {
			return 0, false
		}
		total += amount
	}
	return total, true
}

func sumAllowanceCharges(entries []AllowanceCharge) (Cents, bool) {
	var total Cents
	for _, e := range entries {
		amount, err := e.Amount.Cents()
		if err != nil {
			return 0, false
		}
		total += amount
	}
	return total, true
}

func sumBreakdownTax(groups []VATBreakdown) (Cents, bool) {
	var total Cents
	for _, g := range groups {
		amount, err := g.TaxAmount.Cents()
		if err != nil {
			return 0, false
		}
		total += amount
	}
	return total, true
}

func inCodeList(code string, list map[string]struct{}) bool {
	_, ok := list[strings.TrimSpace(code)]
	return ok
}
