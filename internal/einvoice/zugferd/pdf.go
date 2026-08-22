package zugferd

import "strings"

// The hybrid PDF contract.
//
// ZUGFeRD delivers the structured invoice inside a PDF/A-3: the same file is a
// document a person reads and a record a machine reads. Three things have to be
// right for that, and each of them is checked by every receiving system:
// the attachment's file name, its relationship to the page content, and the
// PDF/A conformance level.
const (
	// AttachmentName is what the invoice data is called inside the PDF. Since
	// ZUGFeRD 2.1 it is the Factur-X name; a reader still has to know the older
	// ones, because invoices from 2018 are still being archived and audited.
	AttachmentName = "factur-x.xml"
	// AttachmentNameZUGFeRD1 was used by ZUGFeRD 1.0.
	AttachmentNameZUGFeRD1 = "ZUGFeRD-invoice.xml"
	// AttachmentNameZUGFeRD2 was used by ZUGFeRD 2.0.
	AttachmentNameZUGFeRD2 = "zugferd-invoice.xml"

	// Relationship is the /AFRelationship value the specification prescribes.
	// "Alternative" says the attachment is another rendering of the same
	// content — not a supplement, and not a source the page was built from.
	Relationship = "Alternative"

	// MimeType is what the embedded file is declared as.
	MimeType = "text/xml"

	// Conformance is the PDF/A level a hybrid invoice has to meet. Only PDF/A-3
	// permits arbitrary embedded files at all.
	Conformance = "a-3b"

	// Description is the human-readable label of the attachment.
	Description = "Rechnungsdaten im ZUGFeRD-Format"
)

// KnownAttachmentNames are the file names an embedded invoice may carry, newest
// first.
func KnownAttachmentNames() []string {
	return []string{AttachmentName, AttachmentNameZUGFeRD2, AttachmentNameZUGFeRD1}
}

// IsInvoiceAttachment reports whether a file name inside a PDF is the invoice
// data rather than an ordinary enclosure.
//
// The distinction matters: a hybrid invoice may legitimately carry a timesheet
// or a delivery note alongside the record, and booking from the wrong file
// would mean booking from a document that is not the invoice.
func IsInvoiceAttachment(name string) bool {
	for _, known := range KnownAttachmentNames() {
		if strings.EqualFold(strings.TrimSpace(name), known) {
			return true
		}
	}
	return false
}
