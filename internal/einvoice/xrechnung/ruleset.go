package xrechnung

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// Version is the XRechnung release these rules were taken from.
const Version = "xrechnung/3.0"

// Option configures the rule set.
type Option func(*ruleset)

// WithCleanVehicles turns on the rules for procurement under the Clean Vehicles
// Directive (BR-DE-CVD-*).
//
// They are off by default because they cannot be triggered from the document:
// whether a purchase falls under the directive is something the buyer knows and
// the invoice does not say. Running them unasked would report a contract
// reference as missing on every ordinary invoice.
func WithCleanVehicles() Option {
	return func(r *ruleset) { r.cleanVehicles = true }
}

// Ruleset returns the German CIUS as a layer over EN 16931.
func Ruleset(opts ...Option) einvoice.Ruleset {
	r := &ruleset{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

type ruleset struct {
	cleanVehicles bool
}

func (r *ruleset) ID() string { return Version }

func (r *ruleset) Check(inv *einvoice.Invoice) []einvoice.Finding {
	if inv == nil {
		return nil
	}
	c := &checker{inv: inv, out: einvoice.NewReporter(xrechnungRules)}

	c.checkSeller()
	c.checkBuyer()
	c.checkDelivery()
	c.checkDocumentType()
	c.checkPayment()
	c.checkTax()
	c.checkAttachments()
	c.checkProfile()
	if r.cleanVehicles {
		c.checkCleanVehicles()
	}
	return c.out.Findings()
}

type checker struct {
	inv *einvoice.Invoice
	out *einvoice.Reporter
}

// checkSeller covers BR-DE-2 to BR-DE-7 and BR-DE-27/28.
//
// XRechnung makes the seller reachable by a human: address, contact point,
// telephone and e-mail are all mandatory. That is not bureaucracy — a public
// authority that cannot query an invoice has to reject it.
func (c *checker) checkSeller() {
	seller := c.inv.Seller

	if seller.Address != nil {
		c.out.Require("BR-DE-3", "Verkäufer", seller.Address.City, "Der Ort des Verkäufers (BT-37)")
		c.out.Require("BR-DE-4", "Verkäufer", seller.Address.PostCode, "Die Postleitzahl des Verkäufers (BT-38)")
	}

	contact := seller.Contact
	if contact == nil {
		c.out.Report("BR-DE-2", "Verkäufer", "Die Kontaktangaben des Verkäufers (BG-6) fehlen")
		return
	}
	c.out.Require("BR-DE-5", "Verkäufer", contact.Name, "Der Ansprechpartner des Verkäufers (BT-41)")
	if c.out.Require("BR-DE-6", "Verkäufer", contact.Phone, "Die Telefonnummer des Verkäufers (BT-42)") {
		if countDigits(contact.Phone) < 3 {
			c.out.Report("BR-DE-27", "Verkäufer",
				"Die Telefonnummer %q enthält weniger als drei Ziffern", contact.Phone)
		}
	}
	if c.out.Require("BR-DE-7", "Verkäufer", contact.Email, "Die E-Mail-Adresse des Verkäufers (BT-43)") {
		if !looksLikeEmail(contact.Email) {
			c.out.Report("BR-DE-28", "Verkäufer",
				"Die E-Mail-Adresse %q ist nicht als solche zu erkennen", contact.Email)
		}
	}
}

// checkBuyer covers BR-DE-8, BR-DE-9 and BR-DE-15.
func (c *checker) checkBuyer() {
	if a := c.inv.Buyer.Address; a != nil {
		c.out.Require("BR-DE-8", "Erwerber", a.City, "Der Ort des Erwerbers (BT-52)")
		c.out.Require("BR-DE-9", "Erwerber", a.PostCode, "Die Postleitzahl des Erwerbers (BT-53)")
	}
	// BR-DE-15: die Käuferreferenz ist bei einer XRechnung Pflicht. Bei
	// öffentlichen Auftraggebern ist sie die Leitweg-ID, ohne die die Rechnung
	// im Behördennetz nicht zugestellt werden kann.
	c.out.Require("BR-DE-15", "", c.inv.BuyerReference, "Die Käuferreferenz (BT-10)")
}

// checkDelivery covers BR-DE-10 and BR-DE-11.
func (c *checker) checkDelivery() {
	d := c.inv.Delivery
	if d == nil || d.Address == nil {
		return
	}
	c.out.Require("BR-DE-10", "Lieferanschrift", d.Address.City, "Der Ort der Lieferanschrift (BT-77)")
	c.out.Require("BR-DE-11", "Lieferanschrift", d.Address.PostCode, "Die Postleitzahl der Lieferanschrift (BT-78)")
}

// allowedTypeCodes are the document types XRechnung expects (BR-DE-17).
var allowedTypeCodes = map[string]bool{
	"326": true, // Teilrechnung
	"380": true, // Rechnung
	"381": true, // Gutschrift
	"384": true, // Rechnungskorrektur
	"389": true, // Gutschriftverfahren
	"875": true, // Abschlagsrechnung Bau
	"876": true, // Teilschlussrechnung Bau
	"877": true, // Schlussrechnung Bau
}

// checkDocumentType covers BR-DE-17 and BR-DE-26.
func (c *checker) checkDocumentType() {
	code := strings.TrimSpace(c.inv.TypeCode)
	if code != "" && !allowedTypeCodes[code] {
		c.out.Report("BR-DE-17", "",
			"Der Rechnungstyp %q gehört nicht zu den bei XRechnung vorgesehenen Schlüsseln (326, 380, 381, 384, 389, 875, 876, 877)",
			code)
	}
	// BR-DE-26: eine Korrektur ohne Bezug lässt offen, was sie korrigiert.
	if code == "384" && len(c.inv.PrecedingInvoices) == 0 {
		c.out.Report("BR-DE-26", "",
			"Die Rechnung ist als Korrektur ausgewiesen, nennt aber keine vorausgegangene Rechnung (BG-3)")
	}
}

// Payment means codes, grouped the way BR-DE-23 to BR-DE-25 group them.
var (
	creditTransferCodes = map[string]bool{"30": true, "58": true}
	cardCodes           = map[string]bool{"48": true, "54": true, "55": true}
	directDebitCodes    = map[string]bool{"59": true}
)

// checkPayment covers BR-DE-1, BR-DE-19 to BR-DE-25 and BR-DE-30/31.
//
// The point of these rules is that a payment instruction has to be executable.
// A transfer without an account, a direct debit without a mandate, or a
// document that says "card" and then supplies a bank account are all
// instructions the recipient cannot act on.
func (c *checker) checkPayment() {
	if len(c.inv.PaymentMeans) == 0 {
		c.out.Report("BR-DE-1", "", "Die Rechnung enthält keine Zahlungsanweisung (BG-16)")
		return
	}

	for i, means := range c.inv.PaymentMeans {
		where := fmt.Sprintf("Zahlungsanweisung %d", i+1)
		code := strings.TrimSpace(means.TypeCode)
		hasTransfer := len(means.CreditTransfer) > 0
		hasCard := means.Card != nil
		hasDebit := means.DirectDebit != nil

		switch {
		case creditTransferCodes[code]:
			if !hasTransfer {
				c.out.Report("BR-DE-23-a", where,
					"Das Zahlungsmittel %q ist eine Überweisung, die Rechnung nennt aber kein Zahlungskonto (BG-17)", code)
			}
			if hasCard || hasDebit {
				c.out.Report("BR-DE-23-b", where,
					"Neben der Überweisung stehen Karten- oder Lastschriftangaben; zulässig ist nur eines davon")
			}
		case cardCodes[code]:
			if !hasCard {
				c.out.Report("BR-DE-24-a", where,
					"Das Zahlungsmittel %q ist eine Kartenzahlung, die Rechnung nennt aber keine Kartendaten (BG-18)", code)
			}
			if hasTransfer || hasDebit {
				c.out.Report("BR-DE-24-b", where,
					"Neben der Kartenzahlung stehen Konto- oder Lastschriftangaben; zulässig ist nur eines davon")
			}
		case directDebitCodes[code]:
			if !hasDebit {
				c.out.Report("BR-DE-25-a", where,
					"Das Zahlungsmittel %q ist eine Lastschrift, die Rechnung nennt aber keine Mandatsangaben (BG-19)", code)
			}
			if hasTransfer || hasCard {
				c.out.Report("BR-DE-25-b", where,
					"Neben der Lastschrift stehen Konto- oder Kartenangaben; zulässig ist nur eines davon")
			}
		}

		// BR-DE-19 und BR-DE-20: bei SEPA muss die Kontonummer eine IBAN sein.
		if code == "58" {
			for _, account := range means.CreditTransfer {
				if id := strings.TrimSpace(account.AccountID); id != "" && !ValidIBAN(id) {
					c.out.Report("BR-DE-19", where,
						"Bei SEPA-Überweisung muss das Zahlungskonto (BT-84) eine IBAN sein, angegeben ist %q", id)
				}
			}
		}
		if code == "59" && hasDebit {
			if id := strings.TrimSpace(means.DirectDebit.DebitedAccount); id != "" && !ValidIBAN(id) {
				c.out.Report("BR-DE-20", where,
					"Bei SEPA-Lastschrift muss das belastete Konto (BT-91) eine IBAN sein, angegeben ist %q", id)
			}
		}

		// BR-DE-30 und BR-DE-31: eine Lastschrift ohne Gläubiger-ID oder ohne
		// belastetes Konto lässt sich nicht einziehen.
		if hasDebit {
			creditor := strings.TrimSpace(means.DirectDebit.CreditorID)
			if creditor == "" {
				creditor = strings.TrimSpace(c.inv.CreditorReference)
			}
			if creditor == "" {
				c.out.Report("BR-DE-30", where,
					"Bei einer Lastschrift ist die Gläubiger-Identifikationsnummer (BT-90) anzugeben")
			}
			c.out.Require("BR-DE-31", where, means.DirectDebit.DebitedAccount,
				"Das belastete Konto (BT-91)")
		}
	}

	c.checkSkonto()
}

// skontoLine is the format XRechnung prescribes for a cash discount, because
// EN 16931 has no field for one: it goes into the payment terms as a marked
// line, so that it can be read back instead of parsed out of prose.
var skontoLine = regexp.MustCompile(`^#SKONTO#TAGE=[0-9]+#PROZENT=[0-9]+\.[0-9]{2}(#BASISBETRAG=-?[0-9]+\.[0-9]{2})?#$`)

// checkSkonto covers BR-DE-18.
func (c *checker) checkSkonto() {
	terms := c.inv.PaymentTermsNote
	if terms == "" {
		return
	}
	for _, line := range strings.Split(strings.ReplaceAll(terms, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !skontoLine.MatchString(trimmed) {
			c.out.Report("BR-DE-18", "",
				"Die Skontozeile %q entspricht nicht der vorgeschriebenen Form #SKONTO#TAGE=n#PROZENT=n.nn#",
				trimmed)
		}
	}
}

// vatCategoriesNeedingIdentifier are the categories that oblige the seller to
// identify itself for tax purposes (BR-DE-16).
var vatCategoriesNeedingIdentifier = map[string]bool{
	"S": true, "Z": true, "E": true, "AE": true, "K": true, "G": true, "L": true, "M": true,
}

// checkTax covers BR-DE-14 and BR-DE-16.
func (c *checker) checkTax() {
	// BR-DE-14 ist strenger als die Norm: EN 16931 lässt den Satz bei "nicht
	// steuerbar" weg, XRechnung verlangt ihn immer.
	for i, group := range c.inv.VATBreakdowns() {
		if !group.Rate.Present() {
			c.out.Report("BR-DE-14", fmt.Sprintf("Steuergruppe %d", i+1),
				"Der Steuersatz (BT-119) ist bei XRechnung auch dann anzugeben, wenn er null ist")
		}
	}

	needsIdentifier := false
	for code := range c.inv.CategoryCodesInUse() {
		if vatCategoriesNeedingIdentifier[code] {
			needsIdentifier = true
		}
	}
	if !needsIdentifier {
		return
	}
	seller := c.inv.Seller
	identified := seller.VATIdentifier != "" || seller.TaxRegistration != "" ||
		seller.AdditionalLegalInfo != ""
	if c.inv.TaxRepresentative != nil && c.inv.TaxRepresentative.VATIdentifier != "" {
		identified = true
	}
	if !identified {
		c.out.Report("BR-DE-16", "",
			"Bei den verwendeten Steuerkategorien braucht die Rechnung die USt-IdNr. (BT-31), die Steuernummer (BT-32) oder eine rechtliche Angabe des Verkäufers (BT-33)")
	}
}

// checkAttachments covers BR-DE-22, BR-DEX-01 and BR-TMP-2.
func (c *checker) checkAttachments() {
	seen := map[string]int{}
	for i, doc := range c.inv.SupportingDocs {
		where := fmt.Sprintf("Unterlage %d", i+1)

		if name := strings.TrimSpace(doc.Filename); name != "" {
			if first, ok := seen[name]; ok {
				c.out.Report("BR-DE-22", where,
					"Der Dateiname %q kommt schon bei Unterlage %d vor; jede Anlage braucht einen eigenen", name, first+1)
			} else {
				seen[name] = i
			}
		}
		if code := strings.TrimSpace(doc.MimeCode); code != "" && !allowedMimeCodes[code] {
			c.out.Report("BR-DEX-01", where,
				"Der Dateityp %q ist bei XRechnung nicht zugelassen", code)
		}
		if uri := strings.TrimSpace(doc.ExternalURI); uri != "" && !absoluteURL(uri) {
			c.out.Report("BR-TMP-2", where,
				"Der Verweis auf die externe Datei (BT-124) ist mit %q keine absolute URL", uri)
		}
	}
}

// allowedMimeCodes is what XRechnung admits for an embedded attachment. It is
// the EN 16931 list plus XML, which the extension adds.
var allowedMimeCodes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"text/csv":        true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/vnd.oasis.opendocument.spreadsheet":                    true,
	"application/xml": true,
}

// checkProfile covers BR-DE-21 and BR-DE-TMP-32.
func (c *checker) checkProfile() {
	if !Applies(c.inv) {
		c.out.Report("BR-DE-21", "",
			"Die Kennung der Spezifikation (BT-24) nennt %q und damit nicht XRechnung", c.inv.Profile())
	}

	// BR-DE-TMP-32: ohne Leistungsdatum bleibt offen, wann geleistet wurde —
	// nach § 14 Abs. 4 Nr. 6 UStG eine Pflichtangabe, hier aber nur ein
	// Hinweis, weil die Norm es auch anders zulässt.
	delivered := c.inv.Delivery != nil && c.inv.Delivery.Date.Present()
	if !delivered && !c.inv.Period.Present() {
		c.out.Report("BR-DE-TMP-32", "",
			"Die Rechnung nennt weder ein Lieferdatum (BT-72) noch einen Abrechnungszeitraum (BG-14)")
	}
}

// vehicleCategories and vehicleAttributes are the values the Clean Vehicles
// Directive allows (BR-DE-CVD-04 and BR-DE-CVD-05).
var (
	vehicleCategories = map[string]bool{
		"M1": true, "M2": true, "M3": true,
		"N1": true, "N2": true, "N3": true,
	}
	vehicleAttributes = map[string]bool{
		"cvd-1": true, "cvd-2": true, "cvd-3": true, "cvd-4": true, "cvd-5": true,
	}
)

// checkCleanVehicles covers BR-DE-CVD-01 to BR-DE-CVD-06.
//
// It only runs when the caller says the purchase falls under the directive,
// because the invoice does not say so — see [WithCleanVehicles].
func (c *checker) checkCleanVehicles() {
	c.out.Require("BR-DE-CVD-01", "", c.inv.ContractReference, "Die Vertragsnummer (BT-12)")
	c.out.Require("BR-DE-CVD-02", "", c.inv.TenderReference, "Die Ausschreibungsnummer (BT-17)")

	classified := 0
	for i, line := range c.inv.Lines {
		where := fmt.Sprintf("Position %d", i+1)

		var vehicles []string
		for _, class := range line.Item.Classifications {
			if !strings.EqualFold(class.Scheme, "CVD") {
				continue
			}
			vehicles = append(vehicles, class.Value)
			if !vehicleCategories[strings.ToUpper(strings.TrimSpace(class.Value))] {
				c.out.Report("BR-DE-CVD-04", where,
					"Die Fahrzeugkategorie %q gehört nicht zu den zulässigen Werten (M1 bis M3, N1 bis N3)", class.Value)
			}
		}

		var attributes []string
		for _, attr := range line.Item.Attributes {
			if !strings.EqualFold(attr.Name, "cva") {
				continue
			}
			attributes = append(attributes, attr.Value)
			if !vehicleAttributes[strings.ToLower(strings.TrimSpace(attr.Value))] {
				c.out.Report("BR-DE-CVD-05", where,
					"Der Beschaffungsgrund %q gehört nicht zu den zulässigen Werten (cvd-1 bis cvd-5)", attr.Value)
			}
		}

		if len(vehicles) == 1 && len(attributes) == 1 {
			classified++
			continue
		}
		if len(vehicles) > 0 && len(attributes) != 1 {
			c.out.Report("BR-DE-CVD-06-a", where,
				"Zur Fahrzeugkategorie (BT-158) gehört genau ein Beschaffungsgrund 'cva' (BT-160), hier sind es %d",
				len(attributes))
		}
		if len(attributes) > 0 && len(vehicles) != 1 {
			c.out.Report("BR-DE-CVD-06-b", where,
				"Zum Beschaffungsgrund 'cva' gehört genau eine Fahrzeugkategorie mit dem Schema 'CVD' (BT-158), hier sind es %d",
				len(vehicles))
		}
	}
	if classified == 0 {
		c.out.Report("BR-DE-CVD-03", "",
			"Keine Position weist ein Fahrzeug nach der Clean Vehicles Directive aus")
	}
}

// CheckedRules lists the rules this layer implements.
func CheckedRules() []string {
	out := make([]string, 0, len(xrechnungRules))
	for rule := range xrechnungRules {
		if uncheckedRules[rule] == "" {
			out = append(out, rule)
		}
	}
	sort.Strings(out)
	return out
}

// UncheckedRules names the rules of the specification this layer does not
// implement, each with the reason.
//
// It is exported because a caller who is told "passed" deserves to know what
// was not looked at. Silence about a gap is worse than the gap.
func UncheckedRules() map[string]string {
	out := make(map[string]string, len(uncheckedRules))
	for rule, reason := range uncheckedRules {
		out[rule] = reason
	}
	return out
}

// uncheckedRules are the rules that cannot be decided on the semantic model of
// EN 16931, with the reason for each.
//
// TODO: die Extension-Regeln (BR-DEX-*) brauchen ein Modell der XRechnung-
// Erweiterung — Unterpositionen und Zahlungen durch Dritte —, die Syntaxregeln
// (BR-TMP-3 bis -5) eine Prüfung am XML-Baum statt am Modell. Beides ist eine
// eigene Schicht, kein Nachtrag hier.
var uncheckedRules = map[string]string{
	"BR-DEX-02": "betrifft Unterpositionen (BG-DEX-01), die EN 16931 nicht kennt",
	"BR-DEX-03": "betrifft Unterpositionen (BG-DEX-01), die EN 16931 nicht kennt",
	"BR-DEX-09": "betrifft Zahlungen durch Dritte (BG-DEX-09), die EN 16931 nicht kennt",
	"BR-DEX-10": "betrifft Zahlungen durch Dritte (BG-DEX-09), die EN 16931 nicht kennt",
	"BR-DEX-11": "betrifft Zahlungen durch Dritte (BG-DEX-09), die EN 16931 nicht kennt",
	"BR-DEX-12": "betrifft Zahlungen durch Dritte (BG-DEX-09), die EN 16931 nicht kennt",
	"BR-DEX-13": "betrifft Zahlungen durch Dritte (BG-DEX-09), die EN 16931 nicht kennt",
	"BR-DEX-14": "betrifft Zahlungen durch Dritte (BG-DEX-09), die EN 16931 nicht kennt",
	"BR-DEX-15": "erkennt Unterpositionen an der Syntax; das Modell bildet sie nicht ab",
	"BR-TMP-3":  "vergleicht die Preisbasismenge an zwei Stellen der CII-Syntax; das Modell führt sie einmal",
	"BR-TMP-4":  "begrenzt die Wiederholung eines Elements in der Syntax, nicht im Modell",
	"BR-TMP-5":  "begrenzt die Wiederholung eines Elements in der Syntax, nicht im Modell",
	// Die Codelisten-Regeln der Extension prüfen dieselben Schemata wie
	// BR-CL-10, BR-CL-11, BR-CL-21, BR-CL-25 und BR-CL-26 der Norm. Sie laufen
	// dort bereits; hier ein zweites Mal zu melden hieße, denselben Mangel
	// doppelt vorzuhalten.
	"BR-DEX-04":     "prüft dasselbe Kennungsschema wie BR-CL-10 der Norm",
	"BR-DEX-05":     "prüft dasselbe Kennungsschema wie BR-CL-11 der Norm",
	"BR-DEX-06":     "prüft dasselbe Kennungsschema wie BR-CL-21 der Norm",
	"BR-DEX-07":     "prüft dasselbe Adressschema wie BR-CL-25 der Norm",
	"BR-DEX-08":     "prüft dasselbe Kennungsschema wie BR-CL-26 der Norm",
	"BR-TMP-CVD-01": "prüft dasselbe Klassifizierungsschema wie BR-CL-13 der Norm",
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

// looksLikeEmail implements BR-DE-28: exactly one @, not at either end, with at
// least two characters on both sides and no space next to it.
func looksLikeEmail(s string) bool {
	s = strings.TrimSpace(s)
	at := strings.Index(s, "@")
	if at < 2 || at != strings.LastIndex(s, "@") || len(s)-at < 3 {
		return false
	}
	return !strings.ContainsAny(s, " \t")
}

func absoluteURL(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, scheme := range []string{"http://", "https://", "ftp://", "ftps://", "mailto:", "file://"} {
		if strings.HasPrefix(lower, scheme) {
			return true
		}
	}
	return false
}
