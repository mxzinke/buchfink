package einvoice

import (
	"fmt"
	"strings"
)

// Profile identifiers of the specifications built on EN 16931.
//
// The guideline identifier (BT-24) says which flavour of the standard a
// document follows. It matters legally as well as technically: two of the
// ZUGFeRD profiles carry no complete invoice and are therefore not an
// E-Rechnung in the sense of German VAT law at all.
const (
	ProfileEN16931    = "urn:cen.eu:en16931:2017"
	ProfileMinimum    = "urn:factur-x.eu:1p0:minimum"
	ProfileBasicWL    = "urn:factur-x.eu:1p0:basicwl"
	ProfileBasic      = "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:basic"
	ProfileExtended   = "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended"
	ProfileXRechnung  = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_3.0"
	ProfilePeppolBIS3 = "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0"
)

// Profile returns the guideline identifier the document states (BT-24).
func (inv *Invoice) Profile() string { return strings.TrimSpace(inv.SpecificationID) }

// IsXRechnung reports whether the document follows the German CIUS.
//
// XRechnung adds its own rules on top of EN 16931 (the BR-DE family), which
// this module does not implement. Knowing that a document claims to be one is
// still worth having: it tells a caller that passing here is not the whole
// story.
func (inv *Invoice) IsXRechnung() bool {
	return strings.Contains(strings.ToLower(inv.Profile()), "xrechnung")
}

// EnsureUsableProfile rejects the two ZUGFeRD profiles that are not an
// E-Rechnung in the sense of German VAT law.
//
// MINIMUM and BASIC WL carry no complete invoice — no line items, and MINIMUM
// not even a tax breakdown. Booking input tax from one of them would mean
// deducting from a document that legally is not an invoice
// (UStAE 14.1 Abs. 14 Satz 4). They exist as an accompanying record for a paper
// invoice, and that is how they have to be treated.
func (inv *Invoice) EnsureUsableProfile() error {
	switch strings.ToLower(inv.Profile()) {
	case ProfileMinimum:
		return fmt.Errorf("das Profil ZUGFeRD MINIMUM enthält keine vollständige Rechnung und ist keine E-Rechnung im Sinne des Gesetzes (UStAE 14.1 Abs. 14 Satz 4)")
	case ProfileBasicWL:
		return fmt.Errorf("das Profil ZUGFeRD BASIC WL enthält keine Rechnungspositionen und ist keine E-Rechnung im Sinne des Gesetzes (UStAE 14.1 Abs. 14 Satz 4)")
	}
	return nil
}

// DocumentKind classifies what a document is, from its type code (BT-3).
//
// This is the field that decides the sign of a booking, and it is routinely
// ignored: a credit note carries positive amounts and says what it is only
// here. Reading it as an ordinary invoice books the input tax the wrong way
// round and opens a payable where one should have been closed.
type DocumentKind string

const (
	KindInvoice      DocumentKind = "invoice"       // 380, 393, 575, 623, 780 …
	KindCreditNote   DocumentKind = "credit_note"   // 381, 396, 532
	KindCorrection   DocumentKind = "correction"    // 384, 457, 458
	KindPrepayment   DocumentKind = "prepayment"    // 386
	KindSelfBilled   DocumentKind = "self_billed"   // 389
	KindPartialFinal DocumentKind = "partial_final" // 875, 876, 877
	KindOther        DocumentKind = "other"         // alles Übrige aus UNTDID 1001
	KindUnknown      DocumentKind = "unknown"       // kein oder unbekannter Schlüssel
)

var documentKinds = map[string]DocumentKind{
	"380": KindInvoice, "393": KindInvoice, "575": KindInvoice,
	"623": KindInvoice, "780": KindInvoice, "935": KindInvoice,
	"381": KindCreditNote, "396": KindCreditNote, "532": KindCreditNote,
	"384": KindCorrection, "457": KindCorrection, "458": KindCorrection,
	"386": KindPrepayment,
	"389": KindSelfBilled,
	"875": KindPartialFinal, "876": KindPartialFinal, "877": KindPartialFinal,
}

// Kind reports what the document says it is.
func (inv *Invoice) Kind() DocumentKind {
	code := strings.TrimSpace(inv.TypeCode)
	if code == "" {
		return KindUnknown
	}
	if kind, ok := documentKinds[code]; ok {
		return kind
	}
	if _, known := untdid1001[code]; known {
		return KindOther
	}
	return KindUnknown
}

// Label returns a German name for the document kind.
func (k DocumentKind) Label() string {
	switch k {
	case KindInvoice:
		return "Rechnung"
	case KindCreditNote:
		return "Gutschrift"
	case KindCorrection:
		return "Rechnungskorrektur"
	case KindPrepayment:
		return "Anzahlungsrechnung"
	case KindSelfBilled:
		return "Gutschrift im Gutschriftverfahren"
	case KindPartialFinal:
		return "Abschlags- oder Schlussrechnung"
	case KindOther:
		return "sonstiger Rechnungstyp"
	default:
		return "unbekannter Rechnungstyp"
	}
}
