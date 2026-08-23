package domain

import (
	"fmt"
	"strings"
)

// IncomingInvoice is a received E-Rechnung, seen from the booking side.
//
// It is deliberately not the invoice as the standard models it. EN 16931 has
// some 160 business terms; a booking needs a dozen of them, and carrying the
// rest through the posting path would tie every accounting decision to a
// format. What this type holds is what the Buchungsvorschlag is made of — and
// nothing that requires knowing what CII or UBL are.
//
// That is the point of the split: the posting rules can be tested by writing
// one of these by hand, without an XML file anywhere in sight, and the
// e-invoice module can be tested without an account.
type IncomingInvoice struct {
	// Syntax and Profile say what arrived, for display and for the record.
	Syntax       string `json:"syntax"`
	Profile      string `json:"profile"`
	ProfileLabel string `json:"profileLabel,omitempty"`

	// Kind decides the sign of the booking. A credit note carries positive
	// amounts and says what it is only here.
	Kind EInvoiceKind `json:"kind"`

	Number       string `json:"number"`
	IssueDate    string `json:"issueDate"`              // BT-2, ISO
	DeliveryDate string `json:"deliveryDate,omitempty"` // BT-72 oder das Ende des Zeitraums
	DueDate      string `json:"dueDate,omitempty"`      // BT-9
	Currency     string `json:"currency"`

	Supplier InvoiceParty `json:"supplier"`
	Buyer    InvoiceParty `json:"buyer"`

	NetAmount   Cents `json:"netAmount"`
	TaxAmount   Cents `json:"taxAmount"`
	GrossAmount Cents `json:"grossAmount"`

	Positions []IncomingPosition `json:"positions,omitempty"`

	// TaxCategories are the UNTDID 5305 codes the document uses, **from the
	// issuer's point of view**. They have to be turned around before they mean
	// anything on this side — see TaxTreatmentForIncomingCategory.
	TaxCategories []string `json:"taxCategories,omitempty"`

	// References the document carries. PrecedingInvoices is what ties a
	// correction to what it corrects and a Schlussrechnung to its
	// Anzahlungsrechnungen.
	//
	// TODO: Rechnungsverbund — die genannten Nummern gegen die abgelegten Belege
	// auflösen und den Bezug speichern, statt ihn nur anzuzeigen. Ohne das bleibt
	// die Verrechnung einer Korrektur oder einer Anzahlung Handarbeit; siehe
	// docs/anforderung-anzahlungen.md.
	BuyerReference    string   `json:"buyerReference,omitempty"`
	OrderReference    string   `json:"orderReference,omitempty"`
	PrecedingInvoices []string `json:"precedingInvoices,omitempty"`

	// Attachments are the files the document carried inside itself (BG-24).
	// They belong to the Beleg like any other received file.
	Attachments []IncomingAttachment `json:"attachments,omitempty"`

	Validation ReceiptValidation `json:"-"`

	// Notes name what the reader could not map and why, so the gaps are visible
	// rather than silently zero.
	Notes []string `json:"notes,omitempty"`
}

// EInvoiceKind is what a received document says it is (BT-3).
type EInvoiceKind string

const (
	EInvoiceKindInvoice      EInvoiceKind = "invoice"
	EInvoiceKindCreditNote   EInvoiceKind = "credit_note"
	EInvoiceKindCorrection   EInvoiceKind = "correction"
	EInvoiceKindPrepayment   EInvoiceKind = "prepayment"
	EInvoiceKindSelfBilled   EInvoiceKind = "self_billed"
	EInvoiceKindPartialFinal EInvoiceKind = "partial_final"
	EInvoiceKindOther        EInvoiceKind = "other"
	EInvoiceKindUnknown      EInvoiceKind = "unknown"
)

// Bookable reports whether Buchfink can turn this kind of document into a
// booking today.
//
// Only an ordinary invoice can. The others are not defects — they are perfectly
// valid documents whose booking is a different transaction: a credit note
// reverses, a correction replaces, a prepayment invoice is settled on payment
// and deducted again in the Schlussrechnung. Proposing an ordinary expense
// booking for any of them would put the sign or the period wrong, and it would
// look right.
//
// TODO: eigene Buchungswege für Gutschrift (Minderung von Aufwand und Vorsteuer,
// verrechnet gegen die ursprüngliche Rechnung), Rechnungskorrektur und
// Anzahlungsrechnung (§ 14 Abs. 5 UStG — steuerwirksam erst mit der Zahlung, in
// der Schlussrechnung wieder abzusetzen). Kein Sonderfall im vorhandenen Weg:
// jede von ihnen ist ein anderer Geschäftsvorfall.
func (k EInvoiceKind) Bookable() bool { return k == EInvoiceKindInvoice }

// Label returns a German name.
func (k EInvoiceKind) Label() string {
	switch k {
	case EInvoiceKindInvoice:
		return "Rechnung"
	case EInvoiceKindCreditNote:
		return "Gutschrift"
	case EInvoiceKindCorrection:
		return "Rechnungskorrektur"
	case EInvoiceKindPrepayment:
		return "Anzahlungsrechnung"
	case EInvoiceKindSelfBilled:
		return "Gutschrift im Gutschriftverfahren"
	case EInvoiceKindPartialFinal:
		return "Abschlags- oder Schlussrechnung"
	case EInvoiceKindOther:
		return "sonstiger Rechnungstyp"
	default:
		return "unbekannter Rechnungstyp"
	}
}

// InvoiceParty is one side of a received invoice, reduced to what identifies it.
type InvoiceParty struct {
	Name        string `json:"name"`
	VatID       string `json:"vatId,omitempty"`
	TaxID       string `json:"taxId,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// IncomingPosition is one line of a received invoice.
//
// It carries no account: which Aufwandskonto a supply belongs on is a decision
// the user makes, and no invoice knows it.
type IncomingPosition struct {
	Text    string  `json:"text,omitempty"`
	Net     Cents   `json:"net"`
	TaxRate TaxRate `json:"taxRate"`
	// TaxCategory is the UNTDID 5305 code of this line, from the issuer's point
	// of view.
	TaxCategory string `json:"taxCategory,omitempty"`
}

// IncomingAttachment is a file that travelled inside the invoice.
type IncomingAttachment struct {
	FileName    string `json:"fileName"`
	MimeType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
	Content     []byte `json:"-"`
}

// TaxTreatmentForIncomingCategory maps the VAT category code of a received
// invoice to the Steuerfall it produces *for the recipient*.
//
// The code in the document is written from the issuer's point of view, and the
// mapping is not symmetric. "K" says the supplier made a tax-exempt
// intra-community supply; on this side that is an innergemeinschaftlicher
// Erwerb, with acquisition tax and matching input tax. Taking the code at face
// value books half the transaction and leaves the Voranmeldung short.
//
// This lives here, next to the tax model, rather than in the reader: it is a
// tax decision, not a format question, and it has to be testable without an XML
// file anywhere near it.
//
// The result is a proposal the user confirms. Where no honest mapping exists
// the function says so rather than picking something plausible.
func TaxTreatmentForIncomingCategory(categoryCode string) (TaxTreatment, error) {
	switch strings.ToUpper(strings.TrimSpace(categoryCode)) {
	case "S":
		// Regelbesteuerung: der Lieferant hat Steuer ausgewiesen.
		return TaxTreatmentDomestic, nil
	case "AE":
		// Steuerschuldnerschaft des Leistungsempfängers — das sind wir.
		return TaxTreatmentReverseCharge, nil
	case "K":
		// Beim Lieferanten eine steuerfreie innergemeinschaftliche Lieferung,
		// bei uns ein innergemeinschaftlicher Erwerb.
		return TaxTreatmentIntraCommunityAcquisition, nil
	case "Z":
		// Nullsteuersatz — steuerpflichtig zum Satz null, nicht steuerfrei.
		return TaxTreatmentZeroRated, nil
	case "E":
		return TaxTreatmentExempt, nil
	case "O":
		return TaxTreatmentNotTaxable, nil
	case "G":
		// Ausfuhr beim Lieferanten heißt Einfuhr bei uns, mit
		// Einfuhrumsatzsteuer aus dem Zollbescheid statt aus dieser Rechnung.
		// Buchfink bildet den Fall nicht ab, und ihn als "steuerfrei" zu buchen
		// wäre falsch.
		//
		// TODO: Einfuhr abbilden — Einfuhrumsatzsteuer aus dem Zollbescheid als
		// eigener Beleg, verknüpft mit dieser Rechnung.
		return "", fmt.Errorf("der Kategoriecode G steht für eine Ausfuhr des Lieferanten. Für den Empfänger ist das eine Einfuhr, die Buchfink noch nicht abbildet — die Einfuhrumsatzsteuer steht im Zollbescheid, nicht in dieser Rechnung")
	case "L", "M":
		// IGIC und IPSI sind spanische Sondergebietsteuern. Sie kommen in
		// deutschen Eingangsrechnungen nicht vor, und ein Aufwand mit
		// spanischer Sondersteuer ist kein Fall für die Vorsteuer.
		return "", fmt.Errorf("der Kategoriecode %q steht für eine spanische Sondergebietsteuer (IGIC oder IPSI); ein Vorsteuerabzug ergibt sich daraus nicht", strings.ToUpper(categoryCode))
	case "B":
		return "", fmt.Errorf("der Kategoriecode B steht für das italienische Split Payment, das im deutschen Umsatzsteuerrecht keine Entsprechung hat")
	case "":
		return "", fmt.Errorf("der Rechnungsdatensatz nennt keinen Steuerkategoriecode")
	default:
		return "", fmt.Errorf("der Steuerkategoriecode %q ist Buchfink nicht bekannt", categoryCode)
	}
}
