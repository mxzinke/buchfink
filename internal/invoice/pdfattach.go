package invoice

import "github.com/buchfink/buchfink/internal/einvoice"

// Die Arbeit an hybriden PDFs liegt im Modul `internal/einvoice`, weil sie zur
// E-Rechnung gehört und nicht zu Buchfink. Hier stehen nur die Namen, unter
// denen der bestehende Buchungspfad sie kennt.

// EmbeddedFile is one attachment pulled out of a PDF.
type EmbeddedFile = einvoice.EmbeddedFile

// ExtractEmbeddedInvoice returns the structured invoice data embedded in a
// hybrid PDF.
func ExtractEmbeddedInvoice(pdf []byte) (*EmbeddedFile, error) {
	return einvoice.ExtractFromPDF(pdf)
}

// IsPDF reports whether the bytes start with a PDF header.
func IsPDF(data []byte) bool { return einvoice.IsPDF(data) }

// LooksLikeXML reports whether the bytes begin like an XML document.
func LooksLikeXML(data []byte) bool { return einvoice.LooksLikeXML(data) }
