package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// EInvoiceProposal is what Buchfink reads out of a received E-Rechnung.
//
// It is a proposal, never a booking. The user confirms it — every check the
// posting core applies still runs afterwards, and the fachliche Gruppe is a
// decision no XML field can make.
type EInvoiceProposal struct {
	// Request is the prefilled booking. Its Positions carry amounts and rates but
	// no PostingGroup: which expense account a supply belongs on is the user's
	// call, not something the invoice knows.
	Request ReceiptRequest `json:"request"`

	Format  string `json:"format"`
	Profile string `json:"profile"`

	SupplierName   string       `json:"supplierName"`
	SupplierVatID  string       `json:"supplierVatId,omitempty"`
	SupplierTaxID  string       `json:"supplierTaxId,omitempty"`
	InvoiceNumber  string       `json:"invoiceNumber"`
	GrossAmount    domain.Cents `json:"grossAmount"`
	MatchedContact bool         `json:"matchedContact"`

	// Notes name what could not be filled in and why. They are shown next to the
	// form so the gaps are visible rather than silently zero.
	Notes []string `json:"notes,omitempty"`
}

// EInvoiceService reads received E-Rechnungen.
type EInvoiceService struct {
	receiptSvc  *ReceiptService
	contactRepo domain.ContactRepository
	fiscalYear  int
}

// NewEInvoiceService creates the reader for received E-Rechnungen.
func NewEInvoiceService(receiptSvc *ReceiptService, contactRepo domain.ContactRepository, fiscalYear int) *EInvoiceService {
	return &EInvoiceService{receiptSvc: receiptSvc, contactRepo: contactRepo, fiscalYear: fiscalYear}
}

// SetFiscalYear updates the active year.
func (s *EInvoiceService) SetFiscalYear(year int) { s.fiscalYear = year }

// ExtractStructuredPart pulls the invoice data out of a filed Beleg and attaches
// it as the file with role `structured`.
//
// Extraction is a separate step from filing on purpose: a Beleg has to be
// storable in the form it arrived (GoBD Rz. 131) before anybody knows what is in
// it. The extracted XML is marked as derived — it came out of the original
// rather than being received alongside it.
func (s *EInvoiceService) ExtractStructuredPart(ctx context.Context, receiptID uint) (*domain.Receipt, error) {
	receipt, err := s.receiptSvc.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if !receipt.IsOpen() {
		return nil, fmt.Errorf("Beleg %s ist gebucht und lässt sich nicht mehr ergänzen", receipt.ReceiptNumber)
	}
	if _, ok := receipt.FileByRole(domain.ReceiptRoleStructured); ok {
		return receipt, nil
	}

	original, ok := receipt.FileByRole(domain.ReceiptRoleOriginal)
	if !ok {
		return nil, fmt.Errorf("Beleg %s hat keine Originaldatei", receipt.ReceiptNumber)
	}
	content, err := s.receiptSvc.Content(ctx, receiptID, original.ID)
	if err != nil {
		return nil, err
	}

	var structured NewFile
	switch {
	case invoice.IsPDF(content.Data):
		embedded, err := invoice.ExtractEmbeddedInvoice(content.Data)
		if err != nil {
			return nil, err
		}
		structured = NewFile{
			Role: domain.ReceiptRoleStructured, FileName: embedded.Name,
			Content: embedded.Data, Derived: true,
		}

	case isXML(content.MimeType, content.Data):
		// Eine XRechnung ist strukturierter Teil und Original zugleich. Beide
		// Rollen zeigen auf denselben Inhalt, der auf der Platte nur einmal liegt.
		structured = NewFile{
			Role: domain.ReceiptRoleStructured, FileName: original.FileName,
			Content: content.Data,
		}

	default:
		return nil, fmt.Errorf(
			"Beleg %s ist weder ein PDF noch ein XML — ein Scan oder Foto enthält keinen strukturierten Rechnungsdatensatz",
			receipt.ReceiptNumber)
	}

	doc, err := invoice.ParseCII(structured.Content)
	if err != nil {
		return nil, err
	}

	updated, err := s.receiptSvc.AddFile(ctx, receiptID, structured)
	if err != nil {
		return nil, err
	}

	// Geprüft wird direkt nach dem Auslesen: das Ergebnis gehört zum Beleg, und
	// wer eine Rechnung bucht, soll vorher sehen, was an ihr nicht stimmt.
	if err := s.receiptSvc.SaveValidation(ctx, receiptID, invoice.ValidateEN16931(doc)); err != nil {
		return nil, err
	}
	return s.receiptSvc.Get(ctx, updated.ID)
}

// Validate re-runs the rule check against the structured part of a Beleg and
// records the result.
//
// Re-running matters because the rule set is versioned: a document checked under
// an older version can be checked again without being re-filed.
func (s *EInvoiceService) Validate(ctx context.Context, receiptID uint) (*invoice.ValidationResult, error) {
	doc, _, err := s.structuredDocument(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	result := invoice.ValidateEN16931(doc)
	if err := s.receiptSvc.SaveValidation(ctx, receiptID, result); err != nil {
		return nil, err
	}
	return &result, nil
}

// structuredDocument loads and parses the structured part of a Beleg.
func (s *EInvoiceService) structuredDocument(ctx context.Context, receiptID uint) (*invoice.CIIInvoice, *domain.Receipt, error) {
	receipt, err := s.receiptSvc.Get(ctx, receiptID)
	if err != nil {
		return nil, nil, err
	}
	structured, ok := receipt.FileByRole(domain.ReceiptRoleStructured)
	if !ok {
		return nil, nil, fmt.Errorf(
			"Beleg %s trägt keinen strukturierten Rechnungsdatensatz. Der Vorsteuerabzug ist nur aus diesem Teil möglich (UStAE 14c.1 Abs. 4a Satz 4)",
			receipt.ReceiptNumber)
	}
	content, err := s.receiptSvc.Content(ctx, receiptID, structured.ID)
	if err != nil {
		return nil, nil, err
	}
	doc, err := invoice.ParseCII(content.Data)
	if err != nil {
		return nil, nil, err
	}
	return doc, receipt, nil
}

// Propose turns the structured part of a filed Beleg into a booking proposal.
func (s *EInvoiceService) Propose(ctx context.Context, receiptID uint) (*EInvoiceProposal, error) {
	doc, receipt, err := s.structuredDocument(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if err := doc.EnsureUsableProfile(); err != nil {
		return nil, err
	}

	proposal := &EInvoiceProposal{
		Format:        string(invoice.FormatCII),
		Profile:       doc.Profile(),
		SupplierName:  doc.Transaction.Agreement.Seller.Name,
		SupplierVatID: doc.Transaction.Agreement.Seller.VatID(),
		SupplierTaxID: doc.Transaction.Agreement.Seller.TaxNumber(),
		InvoiceNumber: doc.Document.ID,
	}
	if total, err := doc.GrandTotal(); err == nil {
		proposal.GrossAmount = total
	}

	// Der Steuerfall kommt aus dem Kategoriecode — gedreht, weil er aus Sicht
	// des Ausstellers im Dokument steht.
	treatment, note := s.resolveTreatment(doc)
	if note != "" {
		proposal.Notes = append(proposal.Notes, note)
	}

	positions, notes := s.positions(doc, treatment)
	proposal.Notes = append(proposal.Notes, notes...)

	contact, matched := s.matchSupplier(ctx, doc)
	proposal.MatchedContact = matched
	if !matched {
		proposal.Notes = append(proposal.Notes, fmt.Sprintf(
			"Der Lieferant %q ist noch nicht als Kontakt angelegt. Ohne Personenkonto lässt sich der Beleg nicht buchen.",
			doc.Transaction.Agreement.Seller.Name))
	}

	serviceDate := doc.DeliveryDate()
	if serviceDate == "" {
		serviceDate = doc.IssueDate()
		proposal.Notes = append(proposal.Notes,
			"Der Rechnungsdatensatz nennt kein Leistungsdatum; ersatzweise steht das Belegdatum. Der Leistungszeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG.")
	}

	proposal.Request = ReceiptRequest{
		ReceiptID:       receipt.ID,
		BookingDate:     doc.IssueDate(),
		DocumentDate:    doc.IssueDate(),
		ServiceDateFrom: serviceDate,
		ServiceDateTo:   serviceDate,
		TaxTreatment:    treatment,
		Positions:       positions,
		Settlement:      SettlementOpen,
		Currency:        doc.Currency(),
	}
	if contact != nil {
		proposal.Request.ContactID = contact.ID
	}
	if doc.Document.ID != "" {
		proposal.Request.Description = fmt.Sprintf("Rechnung %s, %s",
			doc.Document.ID, doc.Transaction.Agreement.Seller.Name)
	}

	return proposal, nil
}

// resolveTreatment derives the Steuerfall from the tax breakdown.
func (s *EInvoiceService) resolveTreatment(doc *invoice.CIIInvoice) (domain.TaxTreatment, string) {
	taxes := doc.Transaction.Settlement.Taxes
	if len(taxes) == 0 {
		return "", "Der Rechnungsdatensatz enthält keine Steueraufschlüsselung. Der Steuerfall ist von Hand zu wählen."
	}

	codes := map[string]bool{}
	for _, tax := range taxes {
		codes[strings.ToUpper(strings.TrimSpace(tax.CategoryCode))] = true
	}
	if len(codes) > 1 {
		// Buchfink führt den Steuerfall je Beleg, nicht je Position. Eine
		// Rechnung, die beides mischt, ist von Hand zu teilen.
		return "", "Die Rechnung mischt mehrere Steuerkategorien. Buchfink führt den Steuerfall je Beleg — bitte von Hand wählen oder den Beleg aufteilen."
	}

	treatment, err := invoice.IncomingTaxTreatment(taxes[0].CategoryCode)
	if err != nil {
		return "", err.Error()
	}
	return treatment, ""
}

// positions maps the invoice lines to booking positions, without choosing an
// account: the fachliche Gruppe is a decision the user makes and the document
// cannot.
func (s *EInvoiceService) positions(doc *invoice.CIIInvoice, treatment domain.TaxTreatment) ([]ReceiptPosition, []string) {
	var notes []string

	if len(doc.Transaction.Lines) == 0 {
		// Ohne Positionen bleibt die Steueraufschlüsselung als Quelle — sie trägt
		// Bemessungsgrundlage und Satz je Gruppe, und genau das braucht die Buchung.
		var positions []ReceiptPosition
		for _, tax := range doc.Transaction.Settlement.Taxes {
			net, err := domain.ParseCents(tax.BasisAmount)
			if err != nil || net <= 0 {
				continue
			}
			rate, err := invoice.TaxRateFromPercent(tax.RatePercent)
			if err != nil {
				notes = append(notes, err.Error())
				continue
			}
			positions = append(positions, ReceiptPosition{Net: net, TaxRate: rate})
		}
		if len(positions) > 0 {
			notes = append(notes, "Der Rechnungsdatensatz enthält keine Positionen; die Beträge stammen aus der Steueraufschlüsselung.")
		}
		return positions, notes
	}

	positions := make([]ReceiptPosition, 0, len(doc.Transaction.Lines))
	for i, line := range doc.Transaction.Lines {
		net, err := domain.ParseCents(line.Settlement.Summation.LineTotal)
		if err != nil {
			notes = append(notes, fmt.Sprintf("Position %d: der Betrag %q ist unlesbar.", i+1, line.Settlement.Summation.LineTotal))
			continue
		}
		rate, err := invoice.TaxRateFromPercent(line.Settlement.Tax.RatePercent)
		if err != nil {
			notes = append(notes, fmt.Sprintf("Position %d: %s", i+1, err))
			continue
		}
		positions = append(positions, ReceiptPosition{
			Net: net, TaxRate: rate, Text: line.Product.Name,
		})
	}

	if len(positions) > 0 {
		notes = append(notes, "Die Buchungsgruppe ist noch zu wählen — welches Aufwandskonto zutrifft, sagt keine Rechnung.")
	}
	return positions, notes
}

// matchSupplier finds the contact behind the seller, by VAT id, tax number or
// name — in that order, because the first two identify and the last only hints.
func (s *EInvoiceService) matchSupplier(ctx context.Context, doc *invoice.CIIInvoice) (*domain.Contact, bool) {
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, false
	}
	seller := doc.Transaction.Agreement.Seller

	for _, key := range []struct {
		value string
		field func(domain.Contact) string
	}{
		{seller.VatID(), func(c domain.Contact) string { return c.VatID }},
		{seller.TaxNumber(), func(c domain.Contact) string { return c.TaxID }},
	} {
		if key.value == "" {
			continue
		}
		for i := range contacts {
			if contacts[i].Type != domain.ContactTypeVendor {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(key.field(contacts[i])), key.value) {
				return &contacts[i], true
			}
		}
	}

	if name := strings.TrimSpace(seller.Name); name != "" {
		for i := range contacts {
			if contacts[i].Type == domain.ContactTypeVendor && strings.EqualFold(contacts[i].Name, name) {
				return &contacts[i], true
			}
		}
	}
	return nil, false
}

func isXML(mimeType string, data []byte) bool {
	if strings.Contains(mimeType, "xml") {
		return true
	}
	trimmed := strings.TrimLeft(strings.TrimPrefix(string(data), "\ufeff"), " \t\r\n")
	return strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<")
}
