package service

import "slices"

// receiptSpecial ordnet die Angaben für die Freigabegründe: je Position,
// danach die Angaben des Belegs und zuletzt die Befunde der Rechnungsprüfung.
// Die Reihenfolge ist unabhängig von der fachlichen Ausführungsreihenfolge.
type receiptSpecial int

const (
	specialAccount receiptSpecial = iota
	specialInputTaxShare
	specialGift
	specialUnknownGroup
	specialGroup
	specialOverride
	specialEntertainment
	specialProvision
	specialAdvance
	specialExpense
	specialForeignAmount
	specialInputTaxFinding
)

func (s receiptSpecial) String() string {
	return [...]string{
		"konto_statt_gruppe", "vorsteueranteil", "geschenk", "unbekannte_gruppe",
		"gruppe_mit_sonderregel", "uebersteuerung", "bewirtung", "rueckstellung",
		"anzahlung", "auslage", "fremdbetrag", "befund_vorsteuer",
	}[s]
}

// Specials nennt die Sonderangaben der Anfrage anhand derselben Regeln wie
// der Plan, ohne Datenzugriff. Die Policy braucht diesen Teil auch für noch
// ungruppierte Vorschläge und Fremdwährungen ohne vorbereiteten Kurs.
// Berechnete Befunde ergänzt erst die vollständige Vorschau.
func (req ReceiptRequest) Specials() []string {
	var out []string
	appendSpecials := func(specials []receiptSpecial) {
		slices.Sort(specials)
		for _, special := range specials {
			out = append(out, special.String())
		}
	}
	for _, p := range req.Positions {
		var specials []receiptSpecial
		for _, rule := range incomingPositionRules {
			if rule.specials != nil {
				specials = append(specials, rule.specials(p)...)
			}
		}
		appendSpecials(specials)
	}
	var specials []receiptSpecial
	for _, rule := range incomingReceiptRules {
		if rule.specials != nil {
			specials = append(specials, rule.specials(req)...)
		}
	}
	appendSpecials(specials)
	return out
}

// Specials gibt die Sonderangaben des Plans samt Befunden zurück. Sie sind
// Metadaten für die Policy und verändern die serialisierte Vorschau nicht.
func (p *PostingPreview) Specials() []string {
	return slices.Clone(p.specials)
}
