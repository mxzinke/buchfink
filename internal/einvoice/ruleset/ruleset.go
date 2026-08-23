// Package ruleset picks the rule sets that apply to an invoice.
//
// EN 16931 alone is rarely the whole answer. A document says in its guideline
// identifier (BT-24) which specification it was built to, and that decides what
// else has to hold: the German CIUS adds two dozen rules, a ZUGFeRD profile
// restricts what the document may contain at all. Choosing those by hand at
// every call site is how one of them gets forgotten.
//
//	inv, err := einvoice.ParseAny(data)
//	result := ruleset.Validate(inv)   // Norm plus alles, was das Dokument angibt
//
// The package exists because of Go's import rules and is the better design for
// it: the layers cannot import each other, and the core cannot import the
// layers, so the composition lives in one place where it can be read.
package ruleset

import (
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/xrechnung"
	"github.com/buchfink/buchfink/internal/einvoice/zugferd"
)

// For returns the layers that apply to a document, in the order they should
// run.
//
// The choice follows what the document declares, not what the recipient hopes:
// a sender who invokes XRechnung has promised those rules, and checking them is
// what tells the recipient whether the promise holds.
func For(inv *einvoice.Invoice) []einvoice.Ruleset {
	if inv == nil {
		return nil
	}
	var layers []einvoice.Ruleset

	// Die Profilprüfung läuft für jedes Dokument, das eine ZUGFeRD-Kennung
	// trägt — auch für XRechnung, die als ZUGFeRD-Profil ausgeliefert werden
	// kann. Ein unbekanntes Profil ist kein Grund, nicht zu prüfen: dass die
	// Kennung unbekannt ist, ist selbst das Ergebnis.
	layers = append(layers, zugferd.Ruleset())

	if xrechnung.Applies(inv) {
		layers = append(layers, xrechnung.Ruleset())
	}
	return layers
}

// Validate checks an invoice against EN 16931 and every layer it declares.
func Validate(inv *einvoice.Invoice) einvoice.Result {
	return einvoice.ValidateWith(inv, For(inv)...)
}

// Description is what a document says about itself.
//
// It answers the question a receiving path asks first — what is this? — before
// anything is judged. The fields are deliberately separate: a document can be a
// perfectly good ZUGFeRD BASIC and still not be an invoice in the sense of the
// law, and collapsing that into one boolean loses exactly the distinction that
// decides whether input tax may be deducted.
type Description struct {
	Syntax      einvoice.Syntax       // CII oder UBL
	Profile     zugferd.Profile       // die ZUGFeRD-Stufe, falls erkennbar
	Identifier  string                // BT-24, wie das Dokument es schreibt
	Kind        einvoice.DocumentKind // Rechnung, Gutschrift, Korrektur …
	IsXRechnung bool
	// UsesExtension says the document invokes the XRechnung Extension, whose
	// rules are not checked.
	UsesExtension bool
	// BookableAsInvoice says whether the profile carries a complete invoice at
	// all. False for ZUGFeRD MINIMUM and BASIC WL.
	BookableAsInvoice bool
	// FollowsEN16931 is false for ZUGFeRD 1.x, which predates the standard.
	FollowsEN16931 bool
}

// Describe reports what a document declares itself to be.
func Describe(inv *einvoice.Invoice) Description {
	if inv == nil {
		return Description{}
	}
	profile := zugferd.ProfileOf(inv)
	return Description{
		Syntax:            inv.Syntax,
		Profile:           profile,
		Identifier:        inv.Profile(),
		Kind:              inv.Kind(),
		IsXRechnung:       xrechnung.Applies(inv),
		UsesExtension:     xrechnung.UsesExtension(inv),
		BookableAsInvoice: profile.IsInvoice(),
		FollowsEN16931:    profile.FollowsEN16931(),
	}
}
