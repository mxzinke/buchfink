package xrechnung

import (
	"strings"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// The guideline identifiers of the German CIUS, current and past.
//
// All of them are listed, not just the current one. A supplier's system is not
// obliged to keep up, and an invoice built to version 2.0 is still an invoice —
// refusing to recognise it would mean treating an old but valid document as an
// unknown format.
const (
	IdentifierV12 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_1.2"
	IdentifierV20 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_2.0"
	IdentifierV21 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_2.1"
	IdentifierV22 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_2.2"
	IdentifierV23 = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_2.3"
	IdentifierV30 = "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
	// IdentifierV30Legacy is the spelling version 3.0 had before KoSIT moved to
	// the xeinkauf.de namespace. Documents in the wild carry both.
	IdentifierV30Legacy = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_3.0"
)

// Current is the identifier Buchfink writes when it issues an XRechnung.
const Current = IdentifierV30

// Applies reports whether a document claims to be an XRechnung.
//
// The check is on the guideline identifier and deliberately loose about the
// version: what matters is that the sender invoked the German CIUS, because
// that is what tells the recipient which rules the document was built to.
func Applies(inv *einvoice.Invoice) bool {
	if inv == nil {
		return false
	}
	profile := strings.ToLower(inv.Profile())
	return strings.Contains(profile, "xrechnung")
}

// UsesExtension reports whether a document invokes the XRechnung Extension,
// whose rules this package does not check.
//
// Saying so matters more than checking it: a caller who is told "passed" about
// a document with an unchecked extension has been told something misleading.
func UsesExtension(inv *einvoice.Invoice) bool {
	if inv == nil {
		return false
	}
	return strings.Contains(strings.ToLower(inv.Profile()), "extension")
}
