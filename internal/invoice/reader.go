package invoice

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/ruleset"
	"github.com/buchfink/buchfink/internal/einvoice/zugferd"
)

// Reader turns a received file into the view the booking path works with.
//
// It is the one place where the two worlds see each other. On its left is
// `internal/einvoice` with the full semantic model of EN 16931; on its right is
// `domain.IncomingInvoice` with the dozen fields a booking is made of. Keeping
// the translation in one file is what lets the posting rules be tested without
// a single line of XML, and the standard module without a single account.
type Reader struct{}

// NewReader returns the adapter.
func NewReader() *Reader { return &Reader{} }

// Read parses and checks a received invoice.
//
// Whatever arrived is accepted: XML in either syntax, or a hybrid PDF with the
// record inside. Which of the two it is should not be the recipient's problem.
func (r *Reader) Read(data []byte) (*domain.IncomingInvoice, error) {
	inv, err := einvoice.ParseAny(data)
	if err != nil {
		return nil, err
	}
	if err := inv.EnsureUsableProfile(); err != nil {
		return nil, err
	}
	return convert(inv, ruleset.Validate(inv)), nil
}

// ValidateOnly checks a received invoice without mapping it.
//
// The separation matters for a Beleg that cannot be booked: a document in an
// unusable profile still deserves a validation result on file, and refusing to
// check it because it cannot be booked would leave the user without the reason.
func (r *Reader) ValidateOnly(data []byte) (domain.ReceiptValidation, error) {
	inv, err := einvoice.ParseAny(data)
	if err != nil {
		return domain.ReceiptValidation{}, err
	}
	return validationOf(inv, ruleset.Validate(inv)), nil
}

func convert(inv *einvoice.Invoice, result einvoice.Result) *domain.IncomingInvoice {
	out := &domain.IncomingInvoice{
		Syntax:         string(inv.Syntax),
		Profile:        inv.Profile(),
		ProfileLabel:   zugferd.ProfileOf(inv).Label(),
		Kind:           documentKind(inv.Kind()),
		Number:         inv.Number,
		IssueDate:      inv.IssueDate.ISO(),
		DueDate:        inv.DueDate.ISO(),
		Currency:       inv.Currency,
		Supplier:       party(inv.Seller),
		Buyer:          party(inv.Buyer),
		BuyerReference: inv.BuyerReference,
		OrderReference: inv.OrderReference,
		Validation:     validationOf(inv, result),
	}

	out.DeliveryDate = deliveryDate(inv)
	out.NetAmount, out.TaxAmount, out.GrossAmount = totals(inv)
	out.TaxCategories = categories(inv)

	for _, ref := range inv.PrecedingInvoices {
		if ref.Number != "" {
			out.PrecedingInvoices = append(out.PrecedingInvoices, ref.Number)
		}
	}
	for _, doc := range inv.SupportingDocs {
		if len(doc.Attachment) == 0 {
			continue
		}
		out.Attachments = append(out.Attachments, domain.IncomingAttachment{
			FileName:    attachmentName(doc),
			MimeType:    doc.MimeCode,
			Description: doc.Description,
			Content:     doc.Attachment,
		})
	}

	out.Positions, out.Notes = positions(inv)
	if !out.Kind.Bookable() {
		out.Notes = append(out.Notes, fmt.Sprintf(
			"Der Rechnungsdatensatz weist sich als %s aus (Rechnungstyp %s). Buchfink schlägt dafür keine Buchung vor — Vorzeichen und Zeitpunkt sind andere als bei einer gewöhnlichen Rechnung.",
			out.Kind.Label(), inv.TypeCode))
	}
	return out
}

// deliveryDate is the Leistungsdatum, or the end of the invoicing period where
// the document gives a range instead of a day.
//
// The end, not the start: § 14 Abs. 4 Nr. 6 UStG asks when the supply was
// carried out, and for a period that is when it was completed.
func deliveryDate(inv *einvoice.Invoice) string {
	if inv.Delivery != nil && inv.Delivery.Date.Valid() {
		return inv.Delivery.Date.ISO()
	}
	if p := inv.Period; p != nil {
		if p.End.Valid() {
			return p.End.ISO()
		}
		if p.Start.Valid() {
			return p.Start.ISO()
		}
	}
	return ""
}

func party(p einvoice.Party) domain.InvoiceParty {
	return domain.InvoiceParty{
		Name:        p.Name,
		VatID:       p.VATIdentifier,
		TaxID:       p.TaxRegistration,
		CountryCode: p.CountryCode(),
	}
}

func documentKind(kind einvoice.DocumentKind) domain.EInvoiceKind {
	switch kind {
	case einvoice.KindInvoice:
		return domain.EInvoiceKindInvoice
	case einvoice.KindCreditNote:
		return domain.EInvoiceKindCreditNote
	case einvoice.KindCorrection:
		return domain.EInvoiceKindCorrection
	case einvoice.KindPrepayment:
		return domain.EInvoiceKindPrepayment
	case einvoice.KindSelfBilled:
		return domain.EInvoiceKindSelfBilled
	case einvoice.KindPartialFinal:
		return domain.EInvoiceKindPartialFinal
	case einvoice.KindOther:
		return domain.EInvoiceKindOther
	default:
		return domain.EInvoiceKindUnknown
	}
}

func totals(inv *einvoice.Invoice) (net, tax, gross domain.Cents) {
	read := func(a einvoice.Amount) domain.Cents {
		cents, err := a.Cents()
		if err != nil {
			return 0
		}
		return domain.Cents(cents)
	}
	return read(inv.Totals.TaxBasisTotal), read(inv.Totals.TaxTotal), read(inv.Totals.GrandTotal)
}

// categories collects the VAT category codes the document uses, sorted so the
// order does not depend on where in the document they appeared.
func categories(inv *einvoice.Invoice) []string {
	seen := map[string]bool{}
	var out []string
	for _, group := range inv.VATBreakdowns() {
		code := strings.ToUpper(strings.TrimSpace(group.CategoryCode))
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// positions maps the invoice lines to booking positions.
//
// Where the document has no lines the tax breakdown takes over: it carries the
// taxable amount and the rate per group, which is what a booking needs. That is
// the ZUGFeRD BASIC WL case — no lines, but a complete tax statement.
func positions(inv *einvoice.Invoice) ([]domain.IncomingPosition, []string) {
	var notes []string

	if len(inv.Lines) == 0 {
		var out []domain.IncomingPosition
		for i, group := range inv.VATBreakdowns() {
			net, err := group.TaxableAmount.Cents()
			if err != nil || net <= 0 {
				continue
			}
			rate, err := taxRate(group.Rate)
			if err != nil {
				notes = append(notes, fmt.Sprintf("Steuergruppe %d: %s", i+1, err))
				continue
			}
			out = append(out, domain.IncomingPosition{
				Net: domain.Cents(net), TaxRate: rate,
				TaxCategory: strings.ToUpper(strings.TrimSpace(group.CategoryCode)),
			})
		}
		if len(out) > 0 {
			notes = append(notes,
				"Der Rechnungsdatensatz enthält keine Positionen; die Beträge stammen aus der Steueraufschlüsselung.")
		}
		return out, notes
	}

	out := make([]domain.IncomingPosition, 0, len(inv.Lines))
	for i, line := range inv.Lines {
		net, err := line.NetAmount.Cents()
		if err != nil {
			notes = append(notes, fmt.Sprintf("Position %d: der Betrag %q ist unlesbar.", i+1, line.NetAmount))
			continue
		}
		rate, err := taxRate(line.VAT.Rate)
		if err != nil {
			notes = append(notes, fmt.Sprintf("Position %d: %s", i+1, err))
			continue
		}
		out = append(out, domain.IncomingPosition{
			Text: line.Item.Name, Net: domain.Cents(net), TaxRate: rate,
			TaxCategory: strings.ToUpper(strings.TrimSpace(line.VAT.CategoryCode)),
		})
	}
	if len(out) > 0 {
		notes = append(notes,
			"Die Buchungsgruppe ist noch zu wählen — welches Aufwandskonto zutrifft, sagt keine Rechnung.")
	}
	return out, notes
}

// taxRate turns a rate as the document wrote it into Buchfink's basis points.
func taxRate(rate einvoice.Amount) (domain.TaxRate, error) {
	if !rate.Present() {
		return domain.TaxRateNone, nil
	}
	hundredths, err := rate.Cents()
	if err != nil {
		return 0, fmt.Errorf("der Steuersatz %q ist unlesbar", rate)
	}
	// Hundertstel und Basispunkte sind dasselbe Maß — "19.00" wird zu 1900.
	value := domain.TaxRate(hundredths)
	for _, valid := range domain.ValidTaxRates() {
		if value == valid {
			return value, nil
		}
	}
	return 0, fmt.Errorf("der Steuersatz %s %% ist in Deutschland nicht vorgesehen", rate)
}

// validationOf records the check the way the Beleg stores it.
func validationOf(inv *einvoice.Invoice, result einvoice.Result) domain.ReceiptValidation {
	messages := make([]string, 0, len(result.Findings))
	for _, f := range result.Findings {
		where := ""
		if f.Where != "" {
			where = f.Where + ": "
		}
		messages = append(messages, fmt.Sprintf("[%s] %s%s", f.Rule, where, f.Message))
	}
	return domain.ReceiptValidation{
		Format:   string(inv.Syntax),
		Profile:  inv.Profile(),
		At:       time.Now().Format("2006-01-02 15:04:05"),
		Ruleset:  strings.Join(result.Rulesets, " + "),
		Version:  einvoice.RulesetVersion,
		Coverage: coverage(inv),
		Errors:   result.ErrorCount(),
		Findings: strings.Join(messages, "\n"),
	}
}

// coverage says how far the check reached.
//
// EN 16931 is covered completely, so a plain invoice is fully checked. An
// XRechnung is not: its extension rules concern business terms the semantic
// model does not carry, and saying "geprüft" without that caveat would be the
// one failure worse than the gap.
//
// TODO: sobald die Extension- und Syntaxregeln geprüft werden (siehe
// xrechnung.UncheckedRules), fällt die Unterscheidung weg.
func coverage(inv *einvoice.Invoice) string {
	if inv.IsXRechnung() {
		return "partial"
	}
	return "full"
}

func attachmentName(doc einvoice.SupportingDocument) string {
	if name := strings.TrimSpace(doc.Filename); name != "" {
		return name
	}
	if ref := strings.TrimSpace(doc.Reference); ref != "" {
		return ref
	}
	return "anlage"
}
