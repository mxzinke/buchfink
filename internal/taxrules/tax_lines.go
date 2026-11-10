package taxrules

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// ensureLawfulTaxDisclosure hält § 14c UStG auch auf dem Handbuchungsweg.
//
// Der Rechnungsdienst weist eine Rechnung zurück, die zu einem Steuerfall ohne
// Steuerpflicht Umsatzsteuer ausweist. Ohne dieselbe Regel hier bliebe die
// Handbuchung offen: eine Leistung nach § 13b UStG mit einer UST19-Zeile
// stünde in Kennziffer 21 *und* in 81, und in der Zusammenfassenden Meldung
// stünde ein Umsatz, den die Voranmeldung als steuerpflichtig führt. Wer die
// Steuer tatsächlich schuldet, weil die Rechnung außerhalb von
// Buchfink entstanden ist, bucht sie mit dem Schlüssel UST14C in Kennziffer 69.
//
// Die Generalumkehr läuft durch: eine Regel über künftige Buchungen darf die
// Rücknahme vorhandener nicht verhindern.
//
// Fehlt der Steuerfall an der Buchung, entscheiden die Erlöskonten. Eine
// Handbuchung hat ihn nicht zwingend — sie kommt ohne Buchungsgruppe zustande
// —, und ohne diese Ableitung stünde der Fall offen, den die Regel
// verhindern soll: Konto 4337 und eine UST19-Zeile in einer Buchung, gemeldet in
// Kennziffer 21 *und* in 81.
func ensureLawfulTaxDisclosure(_ context.Context, e *domain.JournalEntry, _ domain.EntryRuleView) error {
	treatment := e.TaxTreatment
	derived := false
	if treatment == "" {
		treatment = revenueTreatmentOf(e)
		derived = treatment != ""
	}
	if treatment == "" || treatment.MayShowTax() {
		return nil
	}
	for i, l := range e.Lines {
		if !accounting.IsDomesticOutputTaxKey(l.TaxKey) {
			continue
		}
		source := fmt.Sprintf("der Steuerfall %q", treatment)
		if derived {
			source = fmt.Sprintf("das Erlöskonto der Buchung führt den Steuerfall %q und", treatment)
		}
		return fmt.Errorf(
			"Zeile %d weist %s € Umsatzsteuer aus, %s lässt aber keine entstehen. "+
				"Ein solcher Ausweis wird nach § 14c UStG trotzdem geschuldet – er gehört auf den "+
				"Steuerschlüssel %s (Kennziffer 69), oder der Umsatz ist steuerpflichtig im Inland",
			i+1, l.Amount, source, accounting.TaxKeyUnlawful)
	}
	return nil
}

// revenueTreatmentOf leitet den Steuerfall einer Buchung aus ihren Erlöskonten
// ab.
//
// Abgeleitet wird nur, wenn *jede* Erlöszeile der Buchung einen Steuerfall ohne
// Steuerpflicht hat. Eine Buchung, die daneben einen steuerpflichtigen
// Inlandsumsatz enthält, darf Umsatzsteuer ausweisen — sie gehört dann zu diesem
// Erlös, und ein Verbot träfe die richtige Buchung.
func revenueTreatmentOf(e *domain.JournalEntry) domain.TaxTreatment {
	treatments := accounting.RevenueTreatments()
	taxable := accounting.TaxableRevenueAccounts()
	found := domain.TaxTreatment("")
	for _, l := range e.Lines {
		if taxable[l.Account] {
			return ""
		}
		if t, ok := treatments[l.Account]; ok && t != "" && !t.MayShowTax() && found == "" {
			found = t
		}
	}
	return found
}

// validateTaxLines hält jeden Steuerschlüssel gegen den Katalog. Die
// Generalumkehr übernimmt die Zeilen der stornierten Buchung und läuft durch.
func validateTaxLines(_ context.Context, e *domain.JournalEntry, _ domain.EntryRuleView) error {
	for i, l := range e.Lines {
		if err := accounting.CheckTaxLine(l); err != nil {
			return fmt.Errorf("Zeile %d: %w", i+1, err)
		}
	}
	return nil
}
