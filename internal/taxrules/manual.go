package taxrules

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// validateManualTaxLines verlangt dieselben steuerlichen Angaben wie der
// Belegweg (BEL-01 K2): Steuerfall, Steuerschlüssel und Bemessungsgrundlage.
// Ohne sie fehlten Kennziffer oder Umsatz in der Voranmeldung.
//
// Drei Regeln, und jede hat ihren eigenen Grund:
//
//   - Der Steuerfall ist Pflicht. Er ist die Aussage darüber, warum Steuer
//     entsteht oder nicht; „nicht steuerbar" ist eine Antwort, ein leeres Feld
//     ist keine.
//   - Eine Zeile auf einem Steuerkonto braucht Schlüssel und
//     Bemessungsgrundlage. Der Schlüssel entscheidet über die Kennziffer, die
//     Bemessungsgrundlage über den Umsatz darunter — die Voranmeldung meldet
//     beides.
//   - Ein steuerpflichtiger Inlandsumsatz braucht eine Steuerzeile. Ohne sie
//     wäre er ein Umsatz, der Steuer auslöst und keine ausweist.
func validateManualTaxLines(entry *domain.JournalEntry) error {
	if strings.TrimSpace(string(entry.TaxTreatment)) == "" {
		return fmt.Errorf(
			"der Buchung fehlt der Steuerfall. Er sagt, warum Steuer entsteht oder nicht — beim " +
				"Inlandsumsatz, beim innergemeinschaftlichen Erwerb, bei § 13b UStG oder bei einem " +
				"steuerfreien Vorgang. Ohne ihn läuft die Buchung an der Voranmeldung vorbei")
	}
	resolver := accounting.NewSKR04TaxResolver()
	hasDomesticTax := false
	// Eine Zeile, deren Vorsteuer ausdrücklich ausgeschlossen ist (§ 15 Abs. 1a
	// UStG), erklärt die fehlende Steuerzeile: der Umsatz ist steuerpflichtig,
	// die Steuer nur nicht abziehbar. Das ist der Fall des Geschenks über der
	// Freigrenze und keine unvollständige Aufzeichnung.
	inputTaxExcluded := false
	for i, l := range entry.Lines {
		if l.InputTaxShare == domain.InputTaxExcluded {
			inputTaxExcluded = true
		}
		if !resolver.IsTaxAccount(l.Account) {
			continue
		}
		if strings.TrimSpace(l.TaxKey) == "" {
			return fmt.Errorf(
				"Zeile %d bucht auf das Steuerkonto %s ohne Steuerschlüssel. Ohne ihn kommt der Betrag "+
					"in keiner Kennziffer der Voranmeldung an", i+1, l.Account)
		}
		// Die unrichtig ausgewiesene Steuer nach § 14c UStG hat keine
		// Bemessungsgrundlage: die Kennziffer 69 meldet den geschuldeten
		// Betrag und keinen Umsatz darunter. Eine Zahl zu verlangen, die es
		// nicht gibt, hieße sie erfinden.
		if l.TaxBase == 0 && l.TaxKey != accounting.TaxKeyUnlawful {
			return fmt.Errorf(
				"Zeile %d hat den Steuerschlüssel %s, aber keine Bemessungsgrundlage. Die "+
					"Voranmeldung meldet den Umsatz und die Steuer daraus — ohne die "+
					"Bemessungsgrundlage fehlte der Umsatz", i+1, l.TaxKey)
		}
		if accounting.IsDomesticOutputTaxKey(l.TaxKey) || l.TaxKey == "VST19" ||
			l.TaxKey == "VST7" || l.TaxKey == accounting.TaxKeyUnlawful {
			// Die unrichtig ausgewiesene Steuer zählt mit: sie wird nach
			// § 14c Abs. 1 UStG geschuldet, und eine Buchung, die sie hat,
			// weist die Steuer aus — auch wenn sie es nicht dürfte.
			hasDomesticTax = true
		}
	}
	if entry.TaxTreatment == domain.TaxTreatmentDomestic && !hasDomesticTax && !inputTaxExcluded {
		return fmt.Errorf(
			"die Buchung hat den Steuerfall „steuerpflichtiger Inlandsumsatz\", aber keine " +
				"Steuerzeile. Ein steuerpflichtiger Umsatz löst Umsatzsteuer oder Vorsteuer aus; " +
				"fällt keine an, ist der Steuerfall ein anderer — steuerfrei, nicht steuerbar oder " +
				"Nullsteuersatz nach § 12 Abs. 3 UStG")
	}
	return nil
}
