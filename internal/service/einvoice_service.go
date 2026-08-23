package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// EInvoiceReader turns a received file into the view the booking path works
// with.
//
// The interface belongs to the booking path, not to the reader. What it names
// is what a booking needs — not what a format happens to offer — and that is
// what keeps the two apart: the posting rules below can be exercised by writing
// a domain.IncomingInvoice by hand, with no XML anywhere in the test, and the
// e-invoice module can be exercised with no account anywhere in its.
type EInvoiceReader interface {
	// Read parses and checks whatever arrived: XML in either syntax, or a
	// hybrid PDF with the record inside.
	Read(data []byte) (*domain.IncomingInvoice, error)
	// ValidateOnly checks without mapping. A document Buchfink cannot book
	// still deserves a validation result on file — refusing to check it would
	// leave the user without the reason.
	ValidateOnly(data []byte) (domain.ReceiptValidation, error)
}

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
	// Kind is what the document says it is. A credit note carries positive
	// amounts and says so only here.
	Kind      string `json:"kind"`
	KindLabel string `json:"kindLabel"`

	SupplierName   string       `json:"supplierName"`
	SupplierVatID  string       `json:"supplierVatId,omitempty"`
	SupplierTaxID  string       `json:"supplierTaxId,omitempty"`
	InvoiceNumber  string       `json:"invoiceNumber"`
	GrossAmount    domain.Cents `json:"grossAmount"`
	MatchedContact bool         `json:"matchedContact"`

	// PrecedingInvoices are the invoices this one refers to (BG-3) — what ties
	// a correction to what it corrects.
	PrecedingInvoices []string `json:"precedingInvoices,omitempty"`

	// Notes name what could not be filled in and why. They are shown next to the
	// form so the gaps are visible rather than silently zero.
	Notes []string `json:"notes,omitempty"`
}

// EInvoiceService reads received E-Rechnungen.
type EInvoiceService struct {
	receiptSvc  *ReceiptService
	contactRepo domain.ContactRepository
	reader      EInvoiceReader
	fiscalYear  int
}

// NewEInvoiceService creates the reader for received E-Rechnungen.
func NewEInvoiceService(receiptSvc *ReceiptService, contactRepo domain.ContactRepository, reader EInvoiceReader, fiscalYear int) *EInvoiceService {
	return &EInvoiceService{
		receiptSvc: receiptSvc, contactRepo: contactRepo,
		reader: reader, fiscalYear: fiscalYear,
	}
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

	// Erst lesen, dann ablegen: was sich nicht lesen lässt, wird auch nicht als
	// strukturierter Teil an den Beleg gehängt.
	validation, err := s.reader.ValidateOnly(structured.Content)
	if err != nil {
		return nil, err
	}

	updated, err := s.receiptSvc.AddFile(ctx, receiptID, structured)
	if err != nil {
		return nil, err
	}

	// Die im Datensatz mitgeschickten Unterlagen gehören zum Beleg. Sie wegzuwerfen
	// hieße, einen Teil des Empfangenen zu verlieren — der Stundenzettel zur
	// Rechnung ist Aufbewahrungsgegenstand wie die Rechnung selbst.
	if updated, err = s.attachEnclosures(ctx, updated, structured.Content); err != nil {
		return nil, err
	}

	// Geprüft wird direkt nach dem Auslesen: das Ergebnis gehört zum Beleg, und
	// wer eine Rechnung bucht, soll vorher sehen, was an ihr nicht stimmt.
	if err := s.receiptSvc.SaveValidation(ctx, receiptID, validation); err != nil {
		return nil, err
	}
	return s.receiptSvc.Get(ctx, updated.ID)
}

// attachEnclosures files the documents the invoice carried inside itself.
//
// A record that cannot be mapped is not an error here: a ZUGFeRD MINIMUM
// document has no enclosures to lose, and refusing to file the Beleg over it
// would leave the user with neither the record nor the reason. A failure to
// *store* an enclosure is a different matter and passes through — a file that
// arrived and did not get written is exactly what must not disappear quietly.
func (s *EInvoiceService) attachEnclosures(ctx context.Context, receipt *domain.Receipt, structured []byte) (*domain.Receipt, error) {
	read, err := s.reader.Read(structured)
	if err != nil || len(read.Attachments) == 0 {
		return receipt, nil
	}
	for _, enclosure := range read.Attachments {
		receipt, err = s.receiptSvc.AddFile(ctx, receipt.ID, NewFile{
			Role:     domain.ReceiptRoleAttachment,
			FileName: enclosure.FileName,
			Content:  enclosure.Content,
			Derived:  true,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"die mitgeschickte Unterlage %q konnte nicht abgelegt werden: %w",
				enclosure.FileName, err)
		}
	}
	return receipt, nil
}

// Validate re-runs the rule check against the structured part of a Beleg and
// records the result.
//
// Re-running matters because the rule set is versioned: a document checked under
// an older version can be checked again without being re-filed.
func (s *EInvoiceService) Validate(ctx context.Context, receiptID uint) (*domain.ReceiptValidation, error) {
	content, _, err := s.structuredContent(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	validation, err := s.reader.ValidateOnly(content)
	if err != nil {
		return nil, err
	}
	if err := s.receiptSvc.SaveValidation(ctx, receiptID, validation); err != nil {
		return nil, err
	}
	return &validation, nil
}

// structuredContent loads the structured part of a Beleg.
func (s *EInvoiceService) structuredContent(ctx context.Context, receiptID uint) ([]byte, *domain.Receipt, error) {
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
	return content.Data, receipt, nil
}

// Propose turns the structured part of a filed Beleg into a booking proposal.
func (s *EInvoiceService) Propose(ctx context.Context, receiptID uint) (*EInvoiceProposal, error) {
	content, receipt, err := s.structuredContent(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	read, err := s.reader.Read(content)
	if err != nil {
		return nil, err
	}

	// Was keine gewöhnliche Rechnung ist, wird nicht als eine vorgeschlagen.
	// Eine Gutschrift mindert Aufwand und Vorsteuer und ist gegen die
	// ursprüngliche Rechnung zu verrechnen; sie als Eingangsrechnung zu buchen
	// dreht das Vorzeichen und eröffnet einen offenen Posten, wo einer zu
	// schließen wäre. Das sähe richtig aus, und genau das ist das Problem.
	if !read.Kind.Bookable() {
		return nil, fmt.Errorf(
			"Beleg %s ist eine %s (Rechnungstyp aus dem Datensatz). Buchfink schlägt dafür noch keine Buchung vor — Vorzeichen und Zeitpunkt sind andere als bei einer Eingangsrechnung",
			receipt.ReceiptNumber, read.Kind.Label())
	}

	proposal := &EInvoiceProposal{
		Format:            read.Syntax,
		Profile:           read.Profile,
		Kind:              string(read.Kind),
		KindLabel:         read.Kind.Label(),
		SupplierName:      read.Supplier.Name,
		SupplierVatID:     read.Supplier.VatID,
		SupplierTaxID:     read.Supplier.TaxID,
		InvoiceNumber:     read.Number,
		GrossAmount:       read.GrossAmount,
		PrecedingInvoices: read.PrecedingInvoices,
		Notes:             append([]string{}, read.Notes...),
	}

	// Der Steuerfall kommt aus dem Kategoriecode — gedreht, weil er aus Sicht
	// des Ausstellers im Dokument steht.
	treatment, note := resolveTreatment(read)
	if note != "" {
		proposal.Notes = append(proposal.Notes, note)
	}

	contact, matched := s.matchSupplier(ctx, read.Supplier)
	proposal.MatchedContact = matched
	if !matched {
		proposal.Notes = append(proposal.Notes, fmt.Sprintf(
			"Der Lieferant %q ist noch nicht als Kontakt angelegt. Ohne Personenkonto lässt sich der Beleg nicht buchen.",
			read.Supplier.Name))
	}

	serviceDate := read.DeliveryDate
	if serviceDate == "" {
		serviceDate = read.IssueDate
		proposal.Notes = append(proposal.Notes,
			"Der Rechnungsdatensatz nennt kein Leistungsdatum; ersatzweise steht das Belegdatum. Der Leistungszeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG.")
	}

	positions := make([]ReceiptPosition, 0, len(read.Positions))
	for _, p := range read.Positions {
		positions = append(positions, ReceiptPosition{Net: p.Net, TaxRate: p.TaxRate, Text: p.Text})
	}

	proposal.Request = ReceiptRequest{
		ReceiptID:       receipt.ID,
		BookingDate:     read.IssueDate,
		DocumentDate:    read.IssueDate,
		ServiceDateFrom: serviceDate,
		ServiceDateTo:   serviceDate,
		TaxTreatment:    treatment,
		Positions:       positions,
		Settlement:      SettlementOpen,
		Currency:        read.Currency,
	}
	if contact != nil {
		proposal.Request.ContactID = contact.ID
	}
	if read.Number != "" {
		proposal.Request.Description = fmt.Sprintf("Rechnung %s, %s", read.Number, read.Supplier.Name)
	}

	return proposal, nil
}

// resolveTreatment derives the Steuerfall from the VAT categories the document
// uses.
//
// It works on the codes alone — no document, no format. That is deliberate: the
// perspective flip is the step that is easiest to get wrong and the one with
// the largest consequence, and it has to be testable without an invoice file
// anywhere near it.
func resolveTreatment(read *domain.IncomingInvoice) (domain.TaxTreatment, string) {
	switch len(read.TaxCategories) {
	case 0:
		return "", "Der Rechnungsdatensatz enthält keine Steueraufschlüsselung. Der Steuerfall ist von Hand zu wählen."
	case 1:
	default:
		// Buchfink führt den Steuerfall je Beleg, nicht je Position. Eine
		// Rechnung, die beides mischt, ist von Hand zu teilen.
		return "", fmt.Sprintf(
			"Die Rechnung mischt die Steuerkategorien %s. Buchfink führt den Steuerfall je Beleg — bitte von Hand wählen oder den Beleg aufteilen.",
			strings.Join(read.TaxCategories, ", "))
	}

	treatment, err := domain.TaxTreatmentForIncomingCategory(read.TaxCategories[0])
	if err != nil {
		return "", err.Error()
	}
	return treatment, ""
}

// matchSupplier finds the contact behind the seller, by VAT id, tax number or
// name — in that order, because the first two identify and the last only hints.
func (s *EInvoiceService) matchSupplier(ctx context.Context, supplier domain.InvoiceParty) (*domain.Contact, bool) {
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, false
	}

	for _, key := range []struct {
		value string
		field func(domain.Contact) string
	}{
		{supplier.VatID, func(c domain.Contact) string { return c.VatID }},
		{supplier.TaxID, func(c domain.Contact) string { return c.TaxID }},
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

	if name := strings.TrimSpace(supplier.Name); name != "" {
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
	// Ein UTF-8-BOM am Anfang ist verbreitet und kein Fehler.
	trimmed := strings.TrimLeft(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})), " \t\r\n")
	return strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<")
}
