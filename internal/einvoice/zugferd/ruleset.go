package zugferd

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// Version identifies this layer in a validation result.
const Version = "zugferd/2.3"

// The profile rules.
//
// These identifiers are Buchfink's own. The reference implementation expresses
// the same restrictions as XML cardinality rules on the CII syntax — up to 943
// of them per profile — which cannot be reproduced on a semantic model. What is
// reproduced is what they amount to: a document may not carry what its profile
// cannot express.
const (
	// RuleProfileKnown: the guideline identifier names a profile at all.
	RuleProfileKnown = "ZF-PROFIL-01"
	// RuleProfileFits: the content fits the declared profile.
	RuleProfileFits = "ZF-PROFIL-02"
	// RuleProfileIsInvoice: the profile carries a complete invoice.
	RuleProfileIsInvoice = "ZF-PROFIL-03"
	// RuleProfileFollowsStandard: the profile is built on EN 16931.
	RuleProfileFollowsStandard = "ZF-PROFIL-04"
)

var profileRules = map[string]einvoice.RuleInfo{
	RuleProfileKnown:           {Terms: []string{"BT-24"}, Severity: einvoice.SeverityWarning},
	RuleProfileFits:            {Terms: []string{"BT-24"}, Severity: einvoice.SeverityFatal},
	RuleProfileIsInvoice:       {Terms: []string{"BT-24"}, Severity: einvoice.SeverityWarning},
	RuleProfileFollowsStandard: {Terms: []string{"BT-24"}, Severity: einvoice.SeverityWarning},
}

// Ruleset returns the profile checks as a layer over EN 16931.
func Ruleset() einvoice.Ruleset { return ruleset{} }

type ruleset struct{}

func (ruleset) ID() string { return Version }

func (ruleset) Check(inv *einvoice.Invoice) []einvoice.Finding {
	if inv == nil {
		return nil
	}
	out := einvoice.NewReporter(profileRules)
	profile := ProfileOf(inv)

	if profile == ProfileUnknown {
		out.Report(RuleProfileKnown, "",
			"Die Kennung der Spezifikation %q gehört zu keinem bekannten ZUGFeRD-Profil", inv.Profile())
		return out.Findings()
	}

	if !profile.FollowsEN16931() {
		out.Report(RuleProfileFollowsStandard, "",
			"Das Profil %s ist älter als EN 16931 und folgt ihr nicht; seit dem 1. Januar 2025 ist ein solches Dokument eine sonstige Rechnung, keine E-Rechnung",
			profile.Label())
		// Weiter zu prüfen, welche Gruppen das Profil tragen kann, hätte hier
		// keinen Sinn: ZUGFeRD 1 hat ein anderes Datenmodell, und die Tabelle
		// unten beschreibt die Stufen von ZUGFeRD 2.
		return out.Findings()
	}

	// ZF-PROFIL-03 ist ein Hinweis, kein Fehler: MINIMUM und BASIC WL sind
	// gültige Dokumente, nur eben keine Rechnungen. Der Fehler entstünde erst
	// beim Buchen, und darüber entscheidet nicht dieses Modul.
	if !profile.IsInvoice() {
		out.Report(RuleProfileIsInvoice, "",
			"Das Profil %s enthält keine vollständige Rechnung und ist keine E-Rechnung im Sinne des Gesetzes (UStAE 14.1 Abs. 14 Satz 4)",
			profile.Label())
	}

	can := profile.Can()
	report := func(present bool, allowed bool, what string) {
		if present && !allowed {
			out.Report(RuleProfileFits, "",
				"Das Dokument nennt das Profil %s, führt aber %s — das Profil kann das nicht abbilden",
				profile.Label(), what)
		}
	}
	report(len(inv.Lines) > 0, can.Lines, fmt.Sprintf("%d Rechnungspositionen (BG-25)", len(inv.Lines)))
	report(len(inv.VATBreakdown) > 0, can.VATBreakdown, "eine Steueraufschlüsselung (BG-23)")
	report(len(inv.Allowances)+len(inv.Charges) > 0, can.AllowanceCharge,
		"Nachlässe oder Zuschläge (BG-20, BG-21)")
	report(len(inv.SupportingDocs) > 0, can.Attachments,
		fmt.Sprintf("%d rechnungsbegründende Unterlagen (BG-24)", len(inv.SupportingDocs)))
	report(inv.Seller.Contact != nil || inv.Buyer.Contact != nil, can.Contact,
		"Kontaktangaben (BG-6, BG-9)")

	if !can.ItemDetails {
		for i, line := range inv.Lines {
			item := line.Item
			if item.StandardID.Present() || len(item.Classifications) > 0 || len(item.Attributes) > 0 {
				out.Report(RuleProfileFits, fmt.Sprintf("Position %d", i+1),
					"Das Profil %s kann keine Artikeldetails abbilden (BG-31, BG-32)", profile.Label())
				break
			}
		}
	}

	return out.Findings()
}

// CheckedRules lists the profile rules.
func CheckedRules() []string {
	out := make([]string, 0, len(profileRules))
	for rule := range profileRules {
		out = append(out, rule)
	}
	sort.Strings(out)
	return out
}
