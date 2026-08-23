package zugferd

import (
	"strings"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// Profile is one of the ZUGFeRD / Factur-X conformance levels.
type Profile string

const (
	// ProfileUnknown means the document declares something this package does
	// not recognise as a ZUGFeRD profile.
	ProfileUnknown Profile = ""
	// ProfileMinimum carries no lines and no VAT breakdown. It is an
	// accompanying record, not an invoice.
	ProfileMinimum Profile = "minimum"
	// ProfileBasicWL carries a VAT breakdown but no lines.
	ProfileBasicWL Profile = "basic-wl"
	// ProfileBasic is the smallest complete invoice: lines, items, VAT.
	ProfileBasic Profile = "basic"
	// ProfileEN16931 is the full standard, unrestricted. Also called COMFORT.
	ProfileEN16931 Profile = "en16931"
	// ProfileExtended goes beyond the standard with additional fields.
	ProfileExtended Profile = "extended"
	// ProfileXRechnung is the German CIUS carried as a ZUGFeRD profile.
	ProfileXRechnung Profile = "xrechnung"
	// ProfileZUGFeRD1 is any of the ZUGFeRD 1.x profiles. They predate
	// EN 16931 and do not follow it — which since 2025 makes such a document a
	// sonstige Rechnung rather than an E-Rechnung. Recognising them anyway
	// matters: they are still in archives, still get audited, and a recipient
	// needs to be told what arrived rather than "unbekanntes Format".
	ProfileZUGFeRD1 Profile = "zugferd-1"
)

// Identifier returns the guideline identifier (BT-24) Buchfink writes for a
// profile.
func (p Profile) Identifier() string {
	switch p {
	case ProfileMinimum:
		return "urn:factur-x.eu:1p0:minimum"
	case ProfileBasicWL:
		return "urn:factur-x.eu:1p0:basicwl"
	case ProfileBasic:
		return "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:basic"
	case ProfileEN16931:
		return einvoice.ProfileEN16931
	case ProfileExtended:
		return "urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended"
	case ProfileXRechnung:
		return "urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0"
	case ProfileZUGFeRD1:
		// Buchfink schreibt ZUGFeRD 1 nicht. Es zu erzeugen hieße, ein Format
		// auszustellen, das seit 2025 keine E-Rechnung mehr ist.
		return ""
	}
	return ""
}

// Label returns a German name for the profile.
func (p Profile) Label() string {
	switch p {
	case ProfileMinimum:
		return "MINIMUM"
	case ProfileBasicWL:
		return "BASIC WL"
	case ProfileBasic:
		return "BASIC"
	case ProfileEN16931:
		return "EN 16931 (COMFORT)"
	case ProfileExtended:
		return "EXTENDED"
	case ProfileXRechnung:
		return "XRECHNUNG"
	case ProfileZUGFeRD1:
		return "ZUGFeRD 1.x"
	}
	return "unbekannt"
}

// rank orders the profiles by how much they can carry, so that "at least this
// profile" is a comparison and not a table of special cases.
func (p Profile) rank() int {
	switch p {
	case ProfileMinimum:
		return 1
	case ProfileBasicWL:
		return 2
	case ProfileBasic:
		return 3
	case ProfileXRechnung, ProfileEN16931:
		return 4
	case ProfileExtended:
		return 5
	case ProfileZUGFeRD1:
		// ZUGFeRD 1 lässt sich in diese Rangfolge nicht einordnen: es ist ein
		// anderes Datenmodell, kein kleineres. Der Rang bleibt deshalb null,
		// und AtLeast beantwortet die Frage für ein solches Dokument mit nein —
		// was richtig ist, denn es erfüllt keine der Stufen.
		return 0
	}
	return 0
}

// AtLeast reports whether the profile carries at least as much as the other.
func (p Profile) AtLeast(other Profile) bool { return p.rank() >= other.rank() }

// IsInvoice reports whether a document in this profile carries a complete
// invoice.
//
// MINIMUM and BASIC WL do not (UStAE 14.1 Abs. 14 Satz 4): they have no lines,
// and MINIMUM not even a tax breakdown. The input tax deduction cannot rest on
// them.
func (p Profile) IsInvoice() bool { return p.AtLeast(ProfileBasic) }

// FollowsEN16931 reports whether the profile is built on the standard.
//
// ZUGFeRD 1.x is not: it predates EN 16931 and uses a different data model.
// Since 1 January 2025 a document in that format is a sonstige Rechnung, not an
// E-Rechnung — the content may be complete, but the form is not the one the law
// now names.
func (p Profile) FollowsEN16931() bool {
	return p != ProfileZUGFeRD1 && p != ProfileUnknown
}

// Capabilities says what a profile can carry.
//
// The table is derived from the profile Schematron of the reference
// implementation: an element that no rule of a profile mentions is an element
// that profile does not have.
type Capabilities struct {
	Lines           bool // BG-25 Rechnungspositionen
	VATBreakdown    bool // BG-23 Steueraufschlüsselung
	AllowanceCharge bool // BG-20, BG-21 Nachlässe und Zuschläge
	Attachments     bool // BG-24 rechnungsbegründende Unterlagen
	ItemDetails     bool // BG-31 Artikelinformationen
	Contact         bool // BG-6, BG-9 Kontaktangaben
}

// Can returns what the profile is able to express.
func (p Profile) Can() Capabilities {
	switch p {
	case ProfileMinimum:
		return Capabilities{}
	case ProfileBasicWL:
		return Capabilities{VATBreakdown: true, AllowanceCharge: true, Contact: true}
	case ProfileBasic:
		return Capabilities{Lines: true, VATBreakdown: true, AllowanceCharge: true,
			ItemDetails: true, Contact: true}
	case ProfileEN16931, ProfileExtended, ProfileXRechnung:
		return Capabilities{Lines: true, VATBreakdown: true, AllowanceCharge: true,
			Attachments: true, ItemDetails: true, Contact: true}
	}
	return Capabilities{}
}

// identifiers maps every guideline identifier seen in the wild onto a profile.
//
// The list is long on purpose. Producers spell the identifier differently
// across ZUGFeRD versions, and some of them spell it wrongly — with colons
// where the specification has a hash. Those documents exist and have to be
// read; refusing them would mean rejecting a valid invoice over a punctuation
// mark in a field nobody looks at.
var identifiers = map[string]Profile{
	"urn:factur-x.eu:1p0:minimum": ProfileMinimum,
	"urn:zugferd.de:2p0:minimum":  ProfileMinimum,

	"urn:factur-x.eu:1p0:basicwl": ProfileBasicWL,
	"urn:zugferd.de:2p0:basicwl":  ProfileBasicWL,

	"urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:basic": ProfileBasic,
	"urn:cen.eu:en16931:2017#compliant#urn:zugferd.de:2p0:basic":  ProfileBasic,
	"urn:cen.eu:en16931:2017:compliant:factur-x.eu:1p0:basic":     ProfileBasic,

	"urn:cen.eu:en16931:2017": ProfileEN16931,
	"urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931": ProfileEN16931,
	"urn:cen.eu:en16931:2017:compliant:factur-x.eu:1p0:en16931":     ProfileEN16931,

	"urn:ferd:CrossIndustryDocument:invoice:1p0:basic":    ProfileZUGFeRD1,
	"urn:ferd:CrossIndustryDocument:invoice:1p0:comfort":  ProfileZUGFeRD1,
	"urn:ferd:CrossIndustryDocument:invoice:1p0:extended": ProfileZUGFeRD1,

	"urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended": ProfileExtended,
	"urn:cen.eu:en16931:2017#conformant#urn:zugferd.de:2p0:extended":  ProfileExtended,
	"urn:cen.eu:en16931:2017:compliant:factur-x.eu:1p0:extended":      ProfileExtended,
}

// ProfileOf returns the profile a document declares.
//
// An identifier naming XRechnung counts as the XRechnung profile whatever its
// version, because that is the CIUS the sender invoked.
func ProfileOf(inv *einvoice.Invoice) Profile {
	if inv == nil {
		return ProfileUnknown
	}
	declared := strings.TrimSpace(inv.Profile())
	if profile, ok := identifiers[declared]; ok {
		return profile
	}
	lower := strings.ToLower(declared)
	if strings.Contains(lower, "xrechnung") {
		return ProfileXRechnung
	}
	if strings.HasPrefix(lower, "urn:ferd:") {
		return ProfileZUGFeRD1
	}
	// Die Kennung trägt den Profilnamen hinter dem letzten Doppelpunkt. Das ist
	// die letzte Auskunft, die ein unbekanntes Dokument noch hergibt.
	for suffix, profile := range map[string]Profile{
		"minimum": ProfileMinimum, "basicwl": ProfileBasicWL, "basic": ProfileBasic,
		"extended": ProfileExtended, "en16931": ProfileEN16931,
	} {
		if strings.HasSuffix(lower, suffix) {
			return profile
		}
	}
	return ProfileUnknown
}

// MinimumProfileFor returns the smallest profile that can carry the invoice.
//
// This is the question a generator has: declaring a profile the content does
// not fit into produces a document that fails at the recipient, and declaring a
// larger one than needed narrows who can read it.
func MinimumProfileFor(inv *einvoice.Invoice) Profile {
	if inv == nil {
		return ProfileUnknown
	}
	switch {
	case len(inv.SupportingDocs) > 0:
		return ProfileEN16931
	case len(inv.Lines) > 0:
		return ProfileBasic
	case len(inv.VATBreakdown) > 0 || len(inv.Allowances) > 0 || len(inv.Charges) > 0:
		return ProfileBasicWL
	default:
		return ProfileMinimum
	}
}
