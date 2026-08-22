package einvoice

import "strings"

// The EN 16931 semantic data model.
//
// This is the centre of the module, and it is deliberately not a picture of
// either syntax. EN 16931 defines the invoice as a set of business terms (BT)
// grouped into business groups (BG); CII and UBL are two ways of writing that
// same model down. The standard's own rule set is built the same way — the
// business rules live in an abstract Schematron over the semantic model, and
// each syntax only supplies the bindings.
//
// Following that split has one concrete payoff: the rules are written once and
// hold for every syntax. A UBL invoice is checked by the same code as a CII
// one, which is the only way the two can be guaranteed to be judged alike.
//
// Every field carries its BT or BG number. They are not decoration: the rules,
// the error messages and any conversation with a supplier all speak in those
// numbers, and a field whose number is unclear is a field nobody can discuss.

// Invoice is the whole document (BG-25 and everything around it).
type Invoice struct {
	// Process control
	BusinessProcess string // BT-23 Geschäftsprozesstyp
	SpecificationID string // BT-24 Kennung der Spezifikation

	// Document header
	Number           string // BT-1  Rechnungsnummer
	IssueDate        Date   // BT-2  Rechnungsdatum
	TypeCode         string // BT-3  Rechnungstyp (UNTDID 1001)
	Currency         string // BT-5  Rechnungswährung
	TaxCurrency      string // BT-6  Währung für die Umsatzsteuerabrechnung
	TaxPointDate     Date   // BT-7  Datum des Steuerzeitpunkts
	TaxPointDateCode string // BT-8  Schlüssel des Steuerzeitpunkts (UNTDID 2005)
	DueDate          Date   // BT-9  Fälligkeitsdatum
	PaymentTermsNote string // BT-20 Zahlungsbedingungen

	// References
	BuyerReference           string       // BT-10 Käuferreferenz (Leitweg-ID)
	ProjectReference         string       // BT-11 Projektnummer
	ContractReference        string       // BT-12 Vertragsnummer
	OrderReference           string       // BT-13 Bestellnummer
	SalesOrderReference      string       // BT-14 Auftragsnummer
	ReceivingAdviceReference string       // BT-15 Wareneingangsmeldung
	DespatchAdviceReference  string       // BT-16 Lieferavis
	TenderReference          string       // BT-17 Ausschreibungsnummer
	ObjectIdentifier         Identifier   // BT-18 Objektkennung (+ BT-18-1 Schema)
	AccountingCost           string       // BT-19 Buchungsreferenz des Erwerbers
	PrecedingInvoices        []Invoicedoc // BG-3  Rechnungsbezug

	Notes  []Note  // BG-1  Bemerkungen
	Period *Period // BG-14 Rechnungszeitraum

	// Parties
	Seller            Party  // BG-4  Verkäufer
	Buyer             Party  // BG-7  Erwerber
	Payee             *Party // BG-10 Zahlungsempfänger
	TaxRepresentative *Party // BG-11 Steuervertreter des Verkäufers

	Delivery *Delivery // BG-13 Lieferinformationen

	PaymentMeans []PaymentMeans // BG-16 Zahlungsanweisungen

	Allowances []AllowanceCharge // BG-20 Nachlässe auf Dokumentebene
	Charges    []AllowanceCharge // BG-21 Zuschläge auf Dokumentebene

	Totals         Totals               // BG-22 Dokumentsummen
	VATBreakdown   []VATBreakdown       // BG-23 Umsatzsteueraufschlüsselung
	SupportingDocs []SupportingDocument // BG-24 Rechnungsbegründende Unterlagen
	Lines          []Line               // BG-25 Rechnungspositionen

	// Syntax records which wire format the document was read from. It carries no
	// semantics; it exists so a finding can say what was actually inspected.
	Syntax Syntax
}

// Syntax names the wire format an invoice was read from or is written to.
type Syntax string

const (
	SyntaxUnknown Syntax = ""
	// SyntaxCII is UN/CEFACT Cross Industry Invoice — the syntax of ZUGFeRD,
	// Factur-X and one of the two XRechnung flavours.
	SyntaxCII Syntax = "cii"
	// SyntaxUBL is OASIS Universal Business Language — the other XRechnung
	// flavour and the one Peppol BIS Billing leads with.
	SyntaxUBL Syntax = "ubl"
)

// Invoicedoc is a reference to another invoice (BG-3).
type Invoicedoc struct {
	Number    string // BT-25 Nummer der vorausgegangenen Rechnung
	IssueDate Date   // BT-26 Datum der vorausgegangenen Rechnung
}

// Note is a free-text remark with an optional subject code (BG-1).
type Note struct {
	SubjectCode string // BT-21 Betreff-Code (UNTDID 4451)
	Text        string // BT-22 Freitext
}

// Period is a date range (BG-14 for the document, BG-26 for a line).
type Period struct {
	Start Date // BT-73 / BT-134
	End   Date // BT-74 / BT-135
}

// Present reports whether the period states either end of the range.
func (p *Period) Present() bool {
	return p != nil && (p.Start.Present() || p.End.Present())
}

// Party is a participant: seller, buyer, payee or tax representative.
type Party struct {
	Name                string       // BT-27 / BT-44 / BT-59 / BT-62
	TradingName         string       // BT-28 / BT-45
	Identifiers         []Identifier // BT-29 / BT-46 / BT-60 (+ Schema)
	LegalRegistration   Identifier   // BT-30 / BT-47 / BT-61 (+ Schema)
	VATIdentifier       string       // BT-31 / BT-48 / BT-63 USt-IdNr.
	TaxRegistration     string       // BT-32 Steuernummer, nur der Verkäufer
	AdditionalLegalInfo string       // BT-33 Weitere rechtliche Angaben
	ElectronicAddress   Identifier   // BT-34 / BT-49 (+ Schema)
	Address             *Address     // BG-5 / BG-8 / BG-12
	Contact             *Contact     // BG-6 / BG-9
}

// Identified reports whether the party carries any identifier at all — the
// question BR-CO-26 asks, and the one that decides whether a supplier can be
// matched to a Kontakt without the user typing anything.
func (p Party) Identified() bool {
	return len(p.Identifiers) > 0 || p.LegalRegistration.Present() || p.VATIdentifier != ""
}

// CountryCode returns the party's country, or the empty string if it has no
// address at all.
func (p Party) CountryCode() string {
	if p.Address == nil {
		return ""
	}
	return p.Address.CountryCode
}

// Address is a postal address (BG-5, BG-8, BG-12, BG-15).
type Address struct {
	LineOne     string // BT-35 / BT-50 / BT-64 / BT-75
	LineTwo     string // BT-36 / BT-51 / BT-65 / BT-76
	LineThree   string // BT-162 / BT-163 / BT-164 / BT-165
	City        string // BT-37 / BT-52 / BT-66 / BT-77
	PostCode    string // BT-38 / BT-53 / BT-67 / BT-78
	Subdivision string // BT-39 / BT-54 / BT-68 / BT-79
	CountryCode string // BT-40 / BT-55 / BT-69 / BT-80
}

// Contact is a named contact point (BG-6, BG-9).
type Contact struct {
	Name  string // BT-41 / BT-56
	Phone string // BT-42 / BT-57
	Email string // BT-43 / BT-58
}

// Identifier is a value together with the scheme it is issued under.
//
// The scheme matters as much as the value: "123456" means nothing until it is
// said to be a GLN, a Leitweg-ID or a DUNS number, and four rules (BR-62 to
// BR-65) exist for nothing but the presence of that scheme.
type Identifier struct {
	Value         string // der Wert
	Scheme        string // schemeID, z. B. ISO-6523-Kennung oder EAS
	SchemeVersion string // nur bei der Artikelklassifizierung (BT-158-2)
}

// Present reports whether the identifier carries a value.
func (i Identifier) Present() bool { return strings.TrimSpace(i.Value) != "" }

// Delivery is where and when the supply happened (BG-13).
type Delivery struct {
	Name       string     // BT-70 Name des Empfängers
	LocationID Identifier // BT-71 Kennung des Lieferorts (+ Schema)
	Date       Date       // BT-72 Tatsächliches Lieferdatum
	Address    *Address   // BG-15 Lieferanschrift
}

// PaymentMeans is one way the invoice may be settled (BG-16).
type PaymentMeans struct {
	TypeCode       string // BT-81 Zahlungsmittel (UNTDID 4461)
	TypeText       string // BT-82 Zahlungsmittel im Klartext
	RemittanceInfo string // BT-83 Verwendungszweck
	CreditTransfer []CreditTransfer
	Card           *CardInformation
	DirectDebit    *DirectDebit
}

// CreditTransfer is a bank account for a transfer (BG-17).
type CreditTransfer struct {
	AccountID   string // BT-84 IBAN oder Kontonummer
	AccountName string // BT-85 Kontoinhaber
	ProviderID  string // BT-86 BIC
}

// CardInformation is a payment card (BG-18).
type CardInformation struct {
	PrimaryAccountNumber string // BT-87 — gekürzt, siehe BR-51
	HolderName           string // BT-88
}

// DirectDebit is a SEPA mandate (BG-19).
type DirectDebit struct {
	MandateReference string // BT-89
	CreditorID       string // BT-90
	DebitedAccount   string // BT-91
}

// AllowanceCharge is a Nachlass (BG-20, BG-27) or a Zuschlag (BG-21, BG-28).
//
// The two are one structure because they differ only in sign and in the code
// list their reason comes from. Which of the two an instance is follows from
// where it sits in the Invoice or the Line — the model does not carry a flag,
// so a caller cannot iterate the wrong list by accident.
type AllowanceCharge struct {
	Amount      Amount // BT-92  / BT-99  / BT-136 / BT-141
	BaseAmount  Amount // BT-93  / BT-100 / BT-137 / BT-142
	Percentage  Amount // BT-94  / BT-101 / BT-138 / BT-143
	VATCategory string // BT-95  / BT-102
	VATRate     Amount // BT-96  / BT-103
	Reason      string // BT-97  / BT-104 / BT-139 / BT-144
	ReasonCode  string // BT-98  / BT-105 / BT-140 / BT-145
}

// HasReason reports whether a reason or a reason code is stated — what BR-33,
// BR-38, BR-42 and BR-44 require, either one being enough.
func (a AllowanceCharge) HasReason() bool {
	return strings.TrimSpace(a.Reason) != "" || strings.TrimSpace(a.ReasonCode) != ""
}

// Totals are the document totals (BG-22).
type Totals struct {
	LineTotal         Amount // BT-106 Summe der Positionsbeträge
	AllowanceTotal    Amount // BT-107 Summe der Nachlässe
	ChargeTotal       Amount // BT-108 Summe der Zuschläge
	TaxBasisTotal     Amount // BT-109 Gesamtbetrag ohne Umsatzsteuer
	TaxTotal          Amount // BT-110 Gesamtbetrag der Umsatzsteuer
	TaxTotalInTaxCurr Amount // BT-111 dieselbe Steuer in der Abrechnungswährung
	GrandTotal        Amount // BT-112 Gesamtbetrag mit Umsatzsteuer
	PrepaidAmount     Amount // BT-113 Bereits gezahlter Betrag
	RoundingAmount    Amount // BT-114 Rundungsbetrag
	DuePayableAmount  Amount // BT-115 Fälliger Betrag
}

// VATBreakdown is one group of the VAT breakdown (BG-23).
type VATBreakdown struct {
	TaxableAmount       Amount // BT-116 Bemessungsgrundlage
	TaxAmount           Amount // BT-117 Steuerbetrag
	CategoryCode        string // BT-118 Steuerkategorie (UNTDID 5305)
	Rate                Amount // BT-119 Steuersatz
	ExemptionReason     string // BT-120 Grund im Klartext
	ExemptionReasonCode string // BT-121 Grund als Schlüssel (VATEX)
}

// HasExemptionReason reports whether the group states a reason in either form.
func (v VATBreakdown) HasExemptionReason() bool {
	return strings.TrimSpace(v.ExemptionReason) != "" ||
		strings.TrimSpace(v.ExemptionReasonCode) != ""
}

// SupportingDocument is an attachment or a reference to one (BG-24).
//
// Reading this is what turns an invoice with an embedded timesheet into a Beleg
// with two files instead of one silently discarded.
type SupportingDocument struct {
	Reference   string // BT-122 Kennung der Unterlage
	Description string // BT-123 Beschreibung
	ExternalURI string // BT-124 Verweis auf eine externe Datei
	Attachment  []byte // BT-125 eingebettete Datei
	MimeCode    string // BT-125-1
	Filename    string // BT-125-2
}

// Line is one invoice line (BG-25).
type Line struct {
	ID               string     // BT-126 Positionsnummer
	Note             string     // BT-127 Freitext zur Position
	ObjectIdentifier Identifier // BT-128 Objektkennung (+ BT-128-1 Schema)
	Quantity         Amount     // BT-129 Menge
	UnitCode         string     // BT-130 Mengeneinheit (UN/ECE Rec 20)
	NetAmount        Amount     // BT-131 Nettobetrag der Position
	OrderLineID      string     // BT-132 Bestellpositionsnummer
	AccountingCost   string     // BT-133 Buchungsreferenz

	Period     *Period           // BG-26 Zeitraum der Position
	Allowances []AllowanceCharge // BG-27 Nachlässe auf die Position
	Charges    []AllowanceCharge // BG-28 Zuschläge auf die Position

	Price Price // BG-29 Detailinformationen zum Preis
	VAT   LineVAT
	Item  Item // BG-31 Artikelinformationen
}

// Price holds the price details of a line (BG-29).
type Price struct {
	NetPrice     Amount // BT-146 Nettopreis des Artikels
	Discount     Amount // BT-147 Im Preis enthaltener Rabatt
	GrossPrice   Amount // BT-148 Bruttopreis des Artikels
	BaseQuantity Amount // BT-149 Preisbasismenge
	BaseUnit     string // BT-150 Einheit der Preisbasismenge
}

// LineVAT is the tax information of a line (BG-30).
type LineVAT struct {
	CategoryCode string // BT-151 Steuerkategorie der Position
	Rate         Amount // BT-152 Steuersatz der Position
}

// Item describes what was supplied (BG-31).
type Item struct {
	Name              string          // BT-153 Artikelname
	Description       string          // BT-154 Artikelbeschreibung
	SellerID          string          // BT-155 Artikelnummer des Verkäufers
	BuyerID           string          // BT-156 Artikelnummer des Erwerbers
	StandardID        Identifier      // BT-157 Standardkennung (+ BT-157-1 Schema)
	Classifications   []Identifier    // BT-158 Klassifizierung (+ Schema, + Version)
	OriginCountryCode string          // BT-159 Ursprungsland
	Attributes        []ItemAttribute // BG-32 Artikelattribute
}

// ItemAttribute is a name/value pair describing the item (BG-32).
type ItemAttribute struct {
	Name  string // BT-160
	Value string // BT-161
}
