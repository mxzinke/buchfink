package service

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// PostingWarning is a note shown next to a booking preview.
//
// It never blocks. What follows from a missing e-invoice is a legal question,
// and the concept's rule for those is to show and not to judge.
type PostingWarning struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // "info" oder "warning"
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	// SupplierNote is a text the user can pass on to the supplier. The
	// obligation is theirs, and most of them do not know it yet.
	SupplierNote string `json:"supplierNote,omitempty"`
}

const warningEInvoiceMissing = "einvoice_missing"

// eInvoiceNotice reports that an incoming document arrived as a sonstige
// Rechnung although an E-Rechnung was, or will be, owed.
//
// The conditions come from § 14 Abs. 2 Satz 2 Nr. 1 UStG: a supply to another
// business for their business, both parties resident in Germany, and not exempt
// under § 4 Nr. 8 bis 29 UStG. The last one is approximated by the Steuerfall —
// Buchfink knows whether the transaction was treated as exempt, not which
// numbered exemption applied. The approximation is named rather than hidden.
//
// Two things deliberately do not happen here. The booking is never refused: the
// input tax deduction may be at risk, but that is the user's matter to resolve
// with their supplier. And no claim is made for 2027, because the transitional
// rule then depends on the *supplier's* previous-year turnover, which Buchfink
// cannot know.
func eInvoiceNotice(contact *domain.Contact, receipt *domain.Receipt, treatment domain.TaxTreatment, documentDate string, gross domain.Cents) *PostingWarning {
	if contact == nil || receipt == nil || documentDate == "" {
		return nil
	}
	if receipt.Direction != domain.DirectionIncoming {
		return nil
	}
	// Ein strukturierter Teil liegt vor: dann ist es eine E-Rechnung.
	if _, ok := receipt.FileByRole(domain.ReceiptRoleStructured); ok {
		return nil
	}
	// Die Pflicht trifft nur inländische Unternehmer; ein Kleinunternehmer darf
	// nach § 34a UStDV immer eine sonstige Rechnung ausstellen.
	if contact.CountryCode != "DE" || !contact.IsBusiness() || contact.IsSmallBusiness {
		return nil
	}
	// Näherung an § 4 Nr. 8 bis 29 UStG.
	if treatment == domain.TaxTreatmentExempt || treatment == domain.TaxTreatmentNotTaxable {
		return nil
	}

	params, err := accounting.TaxParametersFor(documentDate)
	if err != nil {
		return nil
	}
	// Kleinbetragsrechnungen dürfen immer sonstige Rechnungen sein (§ 33 UStDV).
	if gross > 0 && gross <= params.SmallAmountInvoiceLimit {
		return nil
	}

	supplierNote := fmt.Sprintf(
		"Für Umsätze zwischen inländischen Unternehmern besteht seit dem 01.01.2025 die Pflicht zur E-Rechnung (§ 14 Abs. 2 Satz 2 Nr. 1 UStG). Wir bitten darum, künftige Rechnungen als ZUGFeRD oder XRechnung zu übermitteln. Ein eigenes Postfach ist dafür nicht nötig; die Übermittlung per E-Mail genügt (Beleg %s).",
		receipt.ReceiptNumber)

	switch accounting.EInvoiceTransitionFor(documentDate) {
	case accounting.EInvoiceTransitionAllowed:
		return &PostingWarning{
			Code:         warningEInvoiceMissing,
			Severity:     "info",
			Title:        "Keine E-Rechnung — bis Ende 2026 noch zulässig",
			Detail:       "Für diesen Umsatz gilt die E-Rechnungspflicht, der Lieferant darf nach der Übergangsregelung des § 27 Abs. 38 Nr. 1 UStG aber noch bis zum 31.12.2026 eine sonstige Rechnung ausstellen. Der Vorsteuerabzug ist nicht gefährdet. Ab 2027 ändert sich das — ein Hinweis an den Lieferanten lohnt sich jetzt.",
			SupplierNote: supplierNote,
		}
	case accounting.EInvoiceTransitionConditional:
		return &PostingWarning{
			Code:         warningEInvoiceMissing,
			Severity:     "warning",
			Title:        "Keine E-Rechnung — Vorsteuerabzug möglicherweise gefährdet",
			Detail:       "Für diesen Umsatz gilt die E-Rechnungspflicht. Eine sonstige Rechnung ist 2027 nur noch zulässig, wenn der Gesamtumsatz des Lieferanten im Vorjahr höchstens 800.000 € betrug (§ 27 Abs. 38 Nr. 2 UStG). Das kann Buchfink nicht wissen. Besteht die Pflicht, berechtigt eine sonstige Rechnung dem Grunde nach nicht zum Vorsteuerabzug (UStAE 15.2a Abs. 1 Sätze 3 und 4).",
			SupplierNote: supplierNote,
		}
	default:
		return &PostingWarning{
			Code:         warningEInvoiceMissing,
			Severity:     "warning",
			Title:        "Keine E-Rechnung — Vorsteuerabzug dem Grunde nach gefährdet",
			Detail:       "Für diesen Umsatz gilt die E-Rechnungspflicht, und seit 2028 gibt es keine Übergangsregelung mehr. Eine sonstige Rechnung berechtigt dem Grunde nach nicht zum Vorsteuerabzug (UStAE 15.2a Abs. 1 Sätze 3 und 4). Buchfink bewertet den Fall nicht — bitte mit dem Lieferanten oder der steuerlichen Beratung klären.",
			SupplierNote: supplierNote,
		}
	}
}
