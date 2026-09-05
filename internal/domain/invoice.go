package domain

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// InvoiceStatus is the lifecycle state of an outgoing invoice.
//
// There is no "draft" that already sits in the journal: an invoice is either not
// issued yet, or it is issued and booked. GoBD does not allow an entered
// business transaction to stay editable, so a correction is a Storno plus a new
// invoice, never an edit.
type InvoiceStatus string

const (
	InvoiceStatusDraft  InvoiceStatus = "draft"  // erfasst, noch nicht ausgestellt und nicht gebucht
	InvoiceStatusIssued InvoiceStatus = "issued" // ausgestellt und gebucht, offener Posten
	// InvoiceStatusPendingDocument ist ausgestellt und gebucht, aber ohne
	// Dokument: Nummer und Buchung stehen, das Erzeugen des PDF ist
	// fehlgeschlagen. Der Zustand ist sichtbar und nicht still, weil der Kunde
	// noch nichts bekommen hat — und er ist nachholbar, damit die vergebene
	// Nummer nicht verfällt.
	InvoiceStatusPendingDocument InvoiceStatus = "issued_pending_document"
	InvoiceStatusPaid            InvoiceStatus = "paid"      // vollständig ausgeglichen
	InvoiceStatusCancelled       InvoiceStatus = "cancelled" // storniert
)

// IsIssued reports whether the invoice is out of the door: numbered and booked,
// with or without its document.
func (s InvoiceStatus) IsIssued() bool {
	return s == InvoiceStatusIssued || s == InvoiceStatusPendingDocument || s == InvoiceStatusPaid
}

// InvoiceKind is what the document is, and it decides the type code (BT-3) the
// structured record carries.
//
// The distinction is not cosmetic. A recipient's system books by BT-3: a
// Rechnungskorrektur read as a second invoice opens a second payable, and a
// Schlussrechnung read as an ordinary invoice charges the customer twice for the
// advances they already paid.
type InvoiceKind string

const (
	// InvoiceKindInvoice ist die gewöhnliche Rechnung (UNTDID 1001: 380).
	InvoiceKindInvoice InvoiceKind = "invoice"
	// InvoiceKindAdvance ist die Abschlags- oder Anzahlungsrechnung (386). Sie
	// ist eine vollwertige Rechnung nach § 14 Abs. 5 Satz 1 UStG, wird aber
	// nicht beim Ausstellen gebucht: die Steuer entsteht erst mit der
	// Vereinnahmung (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG).
	InvoiceKindAdvance InvoiceKind = "advance"
	// InvoiceKindFinal ist die Schlussrechnung (380) über die Gesamtleistung.
	// Sie setzt die berechneten und vereinnahmten Anzahlungen ab (BT-113) und
	// verweist auf die Abschlagsrechnungen (BG-3).
	InvoiceKindFinal InvoiceKind = "final"
	// InvoiceKindCorrection ist die Rechnungskorrektur (384): ein vollständig
	// neuer Inhalt mit Bezug auf die berichtigte Rechnung.
	InvoiceKindCorrection InvoiceKind = "correction"
	// InvoiceKindCancellation ist die Stornorechnung (384 mit negierten
	// Beträgen). Das Wort „Gutschrift" steht bewusst nirgends: eine Gutschrift
	// im Sinne des § 14 Abs. 2 Satz 2 UStG ist die Abrechnung des
	// Leistungsempfängers, und die stellt Buchfink nicht aus.
	InvoiceKindCancellation InvoiceKind = "cancellation"
)

// TypeCode is the UNTDID 1001 key of the document kind (BT-3).
func (k InvoiceKind) TypeCode() string {
	switch k {
	case InvoiceKindAdvance:
		return "386"
	case InvoiceKindCorrection, InvoiceKindCancellation:
		return "384"
	default:
		return "380"
	}
}

// Label ist der Klartext, der auf dem Dokument steht.
func (k InvoiceKind) Label() string {
	switch k {
	case InvoiceKindAdvance:
		return "Abschlagsrechnung"
	case InvoiceKindFinal:
		return "Schlussrechnung"
	case InvoiceKindCorrection:
		return "Rechnungskorrektur"
	case InvoiceKindCancellation:
		return "Stornorechnung"
	default:
		return "Rechnung"
	}
}

// BooksOnIssue reports whether issuing the document also books it.
//
// Everything does except the Abschlagsrechnung: for it, the tax arises with the
// payment, so the booking waits for the money.
func (k InvoiceKind) BooksOnIssue() bool { return k != InvoiceKindAdvance }

// InvoiceSentVia records how a document reached its recipient.
type InvoiceSentVia string

const (
	InvoiceSentViaEmail  InvoiceSentVia = "email"
	InvoiceSentViaPortal InvoiceSentVia = "portal"
	InvoiceSentViaPost   InvoiceSentVia = "post"
	InvoiceSentViaOther  InvoiceSentVia = "other"
)

// InvoiceSentViaOption describes one dispatch route for the UI.
//
// Der Typ hat einen Namen, damit die Bridge ihn zurückgeben kann: die
// Oberfläche soll die Beschriftungen nicht ein zweites Mal führen, sonst
// stehen dieselben Wörter an zwei Stellen und laufen auseinander.
type InvoiceSentViaOption struct {
	Via   InvoiceSentVia `json:"via"`
	Label string         `json:"label"`
}

// InvoiceSentViaOptions lists the ways an invoice can be recorded as sent.
func InvoiceSentViaOptions() []InvoiceSentViaOption {
	return []InvoiceSentViaOption{
		{InvoiceSentViaEmail, "E-Mail"},
		{InvoiceSentViaPortal, "Portal / Rechnungseingang"},
		{InvoiceSentViaPost, "Post"},
		{InvoiceSentViaOther, "Anderer Weg"},
	}
}

// PaymentTerms are the payment conditions agreed in advance (BT-20).
//
// Sie stehen auf der Rechnung, weil § 14 Abs. 4 Nr. 7 UStG die im Voraus
// vereinbarte Minderung des Entgelts als Pflichtangabe nennt. Der Skonto selbst
// bleibt ein Vorgang des Zahlungswegs: erst wenn er in Anspruch genommen wird,
// mindern sich Entgelt und Steuer (§ 17 Abs. 1 UStG).
type PaymentTerms struct {
	// DueDays ist das Zahlungsziel in Tagen ab Rechnungsdatum.
	DueDays int `json:"dueDays"`
	// DiscountPermille ist der Skontosatz in Promille (20 = 2 %).
	DiscountPermille int `json:"discountPermille"`
	// DiscountDays ist die Frist, innerhalb derer der Skonto gilt.
	DiscountDays int `json:"discountDays"`
}

// Stated reports whether any condition was agreed at all.
func (t PaymentTerms) Stated() bool {
	return t.DueDays > 0 || (t.DiscountPermille > 0 && t.DiscountDays > 0)
}

// HasDiscount reports whether a Skonto was agreed.
func (t PaymentTerms) HasDiscount() bool {
	return t.DiscountPermille > 0 && t.DiscountDays > 0
}

// DiscountPercent renders the Skonto rate as a German decimal, e.g. "2" or "1,5".
func (t PaymentTerms) DiscountPercent() string {
	whole, fraction := t.DiscountPermille/10, t.DiscountPermille%10
	if fraction == 0 {
		return fmt.Sprintf("%d", whole)
	}
	return fmt.Sprintf("%d,%d", whole, fraction)
}

// Note is the sentence that goes on the document and into BT-20.
func (t PaymentTerms) Note(invoiceDate string) string {
	if !t.Stated() {
		return ""
	}
	var parts []string
	if t.DueDays > 0 {
		parts = append(parts, fmt.Sprintf("Zahlbar innerhalb von %d Tagen ohne Abzug", t.DueDays))
	}
	if t.HasDiscount() {
		until := addDays(invoiceDate, t.DiscountDays)
		if until != "" {
			parts = append(parts, fmt.Sprintf("bei Zahlung bis zum %s %s %% Skonto",
				GermanDate(until), t.DiscountPercent()))
		} else {
			parts = append(parts, fmt.Sprintf("bei Zahlung innerhalb von %d Tagen %s %% Skonto",
				t.DiscountDays, t.DiscountPercent()))
		}
	}
	return strings.Join(parts, ", ") + "."
}

// addDays shifts an ISO date. An unparsable date yields the empty string; the
// caller then prints the number of days instead of an invented date.
func addDays(iso string, days int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

// GermanDate renders an ISO date as TT.MM.JJJJ, leaving anything unparsable
// untouched — an invoice must never print a date nobody entered.
func GermanDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02.01.2006")
}

// InvoiceItem is a single position of an outgoing invoice.
type InvoiceItem struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	InvoiceID   uint   `gorm:"index;not null" json:"invoiceId"`
	Position    int    `gorm:"not null" json:"position"`
	Description string `gorm:"size:500;not null" json:"description"`
	// QuantityMilli holds the quantity with three decimal places (1500 = 1,5).
	// An integer keeps 0,1 h × 3 exact, which a float quantity would not.
	QuantityMilli int64   `gorm:"not null" json:"quantityMilli"`
	Unit          string  `gorm:"size:20;default:'Stück'" json:"unit"`
	UnitPrice     Cents   `gorm:"not null" json:"unitPrice"`
	TaxRate       TaxRate `gorm:"not null" json:"taxRate"`
	// PostingGroup is the fachliche Gruppe this position is booked under; it
	// resolves to the revenue account.
	PostingGroup string `gorm:"size:50" json:"postingGroup"`
}

// TotalNet is the net amount of the position.
func (i *InvoiceItem) TotalNet() Cents {
	return MulRound(i.UnitPrice, i.QuantityMilli, 1000)
}

// Invoice is an outgoing invoice capable of ZUGFeRD 2.2 / Factur-X export.
type Invoice struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	FiscalYear    int    `gorm:"index;not null" json:"fiscalYear"`
	InvoiceNumber string `gorm:"size:50;not null;uniqueIndex" json:"invoiceNumber"`

	Date            string `gorm:"size:10;not null;index" json:"date"`      // Rechnungsdatum
	ServiceDateFrom string `gorm:"size:10;not null" json:"serviceDateFrom"` // § 14 Abs. 4 Nr. 6 UStG
	ServiceDateTo   string `gorm:"size:10;not null" json:"serviceDateTo"`
	DueDate         string `gorm:"size:10;not null" json:"dueDate"`

	ContactID   uint   `gorm:"index;not null" json:"contactId"`
	ContactName string `gorm:"size:255;not null;serializer:encrypted" json:"contactName"`

	Items []InvoiceItem `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE" json:"items"`

	// TaxTreatment applies to the invoice as a whole: an invoice is either a
	// domestic taxable supply, an exempt intra-community supply, an export or a
	// § 13b case — never a mixture.
	TaxTreatment TaxTreatment `gorm:"size:40;not null;default:'domestic'" json:"taxTreatment"`

	// VatIDOverrideReason hält den Grund fest, mit dem eine steuerfreie
	// innergemeinschaftliche Lieferung ohne Bestätigung der USt-IdNr. ausgestellt
	// wurde.
	//
	// Er steht an der Rechnung und nicht nur im Protokoll: die Frage, ob die
	// Befreiung trägt, wird später an dieser Rechnung gestellt, und die Antwort
	// muss dort stehen, wo sie gestellt wird.
	VatIDOverrideReason string `gorm:"size:500;serializer:encrypted" json:"vatIdOverrideReason,omitempty"`

	// TransportKind sagt, wer den Gegenstand befördert hat: „supplier" der
	// Lieferer, „customer" der Abnehmer (Abholfall). Leer wird als Regelfall
	// gelesen — Beförderung durch den Lieferer.
	//
	// Die Angabe gehört an die Rechnung und nicht an den einzelnen Nachweisbeleg:
	// sie beschreibt die Lieferung und entscheidet darüber, ob Art. 45a Abs. 1
	// Buchst. b MwStVO zusätzlich die Gelangensbestätigung des Abnehmers
	// verlangt. Ohne sie bewertete der Jahresbericht jede Lieferung als
	// Regelfall, und ein Abholfall ohne Gelangensbestätigung erschien als
	// nachgewiesen, obwohl er es nicht ist.
	TransportKind string `gorm:"size:20" json:"transportKind,omitempty"`

	NetAmount   Cents  `gorm:"not null" json:"netAmount"`
	TaxAmount   Cents  `gorm:"not null" json:"taxAmount"`
	GrossAmount Cents  `gorm:"not null" json:"grossAmount"`
	Currency    string `gorm:"size:3;default:'EUR'" json:"currency"`

	Status InvoiceStatus `gorm:"size:20;default:'draft';index" json:"status"`

	// JournalEntryID links the invoice to the booking it produced.
	JournalEntryID *uint `gorm:"index" json:"journalEntryId,omitempty"`

	ZUGFeRDXML string `gorm:"type:text;serializer:encrypted" json:"zugferdXml,omitempty"`

	// ReceiptID points at the Beleg holding the issued document: the hybrid PDF
	// as the received form and the XML as the structured part. It replaces the
	// former PDFPath, which was never set — a path to a file nothing produced.
	ReceiptID *uint     `gorm:"index" json:"receiptId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// PaidAmount is the settled part, computed on read.
	PaidAmount Cents `gorm:"-" json:"paidAmount"`

	// Kind ist die Dokumentart und bestimmt BT-3. Leer heißt „Rechnung" —
	// Bestandsdaten aus der Zeit vor dieser Welle tragen nichts.
	Kind InvoiceKind `gorm:"size:20;not null;default:'invoice';index" json:"kind"`

	// Terms sind die im Voraus vereinbarten Zahlungsbedingungen
	// (§ 14 Abs. 4 Nr. 7 UStG, BT-20).
	Terms PaymentTerms `gorm:"embedded;embeddedPrefix:terms_" json:"terms"`

	// SmallAmount markiert die Kleinbetragsrechnung nach § 33 UStDV: bis zur
	// datierten Grenze genügen die verkürzten Angaben, und ein Empfänger ist
	// nicht nötig.
	SmallAmount bool `gorm:"not null;default:false" json:"smallAmount"`

	// PaymentAccount ist das Zahlungsmittelkonto einer Rechnung ohne
	// Empfänger.
	//
	// Ohne erfassten Kunden gibt es kein Personenkonto, gegen das eine
	// Forderung liefe — und es gibt auch keine: der Barverkauf ist im selben
	// Augenblick bezahlt. Gebucht wird deshalb gegen Kasse oder Bank. Leer
	// heißt Kasse; bei einer Rechnung mit Empfänger bleibt das Feld ohne
	// Bedeutung.
	PaymentAccount string `gorm:"size:20" json:"paymentAccount,omitempty"`

	// EInvoiceProfile ist das Format, in dem das Dokument erzeugt wurde. Es
	// steht an der Rechnung und nicht nur am Kontakt, weil ein späterer Wechsel
	// des Kontaktprofils nicht rückwirkend behaupten darf, eine alte Rechnung
	// sei anders ausgestellt worden.
	EInvoiceProfile EInvoiceProfile `gorm:"size:30" json:"eInvoiceProfile,omitempty"`

	// Der Bezug auf die berichtigte oder stornierte Rechnung (BG-3). Nummer und
	// Datum stehen mit, weil ein Bezug ohne sie im XML nichts wert ist und weil
	// er lesbar bleiben muss, wenn die Ursprungsrechnung nicht mitgeladen wird.
	CorrectsInvoiceID     *uint  `gorm:"index" json:"correctsInvoiceId,omitempty"`
	CorrectsInvoiceNumber string `gorm:"size:50" json:"correctsInvoiceNumber,omitempty"`
	CorrectsInvoiceDate   string `gorm:"size:10" json:"correctsInvoiceDate,omitempty"`
	// CancelledByInvoiceID zeigt von der stornierten Rechnung auf ihr
	// Stornodokument — die Gegenrichtung, damit die Kette in beide Richtungen
	// lesbar ist.
	CancelledByInvoiceID *uint `gorm:"index" json:"cancelledByInvoiceId,omitempty"`

	// PrecedingRefs sind die vorausgegangenen Rechnungen (BG-3): bei der
	// Schlussrechnung die Abschlagsrechnungen, die sie absetzt.
	//
	// Als eigene Zeilen und nicht als Liste in einem Feld: es sind mehrere, und
	// jede trägt Nummer *und* Datum — BT-25 ohne BT-26 ist kein Bezug, den ein
	// Empfängersystem auflösen kann.
	PrecedingRefs []InvoiceReference `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE" json:"precedingRefs"`

	// GroupID ordnet Abschlags- und Schlussrechnung ihrem Rechnungsverbund zu.
	GroupID *uint `gorm:"index" json:"groupId,omitempty"`
	// PrepaidAmount ist die Summe der abgesetzten Anzahlungen (BT-113). Sie
	// steht an der Schlussrechnung und ist sonst null.
	PrepaidAmount Cents `gorm:"not null;default:0" json:"prepaidAmount"`
	// PaymentReceivedAt ist der Zeitpunkt der Vereinnahmung. Auf einer
	// Abschlagsrechnung tritt er an die Stelle des Leistungszeitpunkts, sofern
	// er feststeht und vom Rechnungsdatum abweicht (§ 14 Abs. 4 Nr. 6 UStG).
	PaymentReceivedAt string `gorm:"size:10" json:"paymentReceivedAt,omitempty"`

	// Der Versand wird vom Anwender vermerkt, nicht von Buchfink ausgeführt.
	// Ein Versandweg, den die Anwendung nicht hat, wäre eine Behauptung; der
	// Vermerk dagegen ist der Nachweis, den § 14 Abs. 1 UStG mittelbar
	// verlangt — die Rechnung muss den Empfänger erreicht haben.
	SentAt   string         `gorm:"size:10;index" json:"sentAt,omitempty"`
	SentVia  InvoiceSentVia `gorm:"size:20" json:"sentVia,omitempty"`
	SentNote string         `gorm:"size:255" json:"sentNote,omitempty"`
}

// InvoiceReference is one preceding invoice a document refers to (BG-3).
type InvoiceReference struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	InvoiceID uint   `gorm:"index;not null" json:"invoiceId"`
	Number    string `gorm:"size:50;not null" json:"number"` // BT-25
	Date      string `gorm:"size:10" json:"date"`            // BT-26
}

// EnsureLists belegt die Listen, die als JSON an die Oberfläche gehen.
//
// Ein nicht belegter Slice wird dort zu `null`, und `precedingRefs.length`
// wirft im Render einen TypeError. Betroffen wäre der Regelfall: jede Rechnung
// ohne Bezug auf eine vorausgegangene. Aus der Datenbank gelesen sind die
// Listen belegt; frisch aufgebaut — beim Ausstellen — sind sie es nicht.
func (inv *Invoice) EnsureLists() {
	if inv.Items == nil {
		inv.Items = []InvoiceItem{}
	}
	if inv.PrecedingRefs == nil {
		inv.PrecedingRefs = []InvoiceReference{}
	}
}

// ResolvedKind fills in the default for records written before the field
// existed.
func (inv *Invoice) ResolvedKind() InvoiceKind {
	if inv.Kind == "" {
		return InvoiceKindInvoice
	}
	return inv.Kind
}

// OpenAmount is what is still to be paid after the settled advances (BT-115).
func (inv *Invoice) OpenAmount() Cents { return inv.GrossAmount - inv.PrepaidAmount }

// TaxGroup is the net base and tax of one VAT rate on an invoice.
type TaxGroup struct {
	Rate TaxRate `json:"rate"`
	Net  Cents   `json:"net"`
	Tax  Cents   `json:"tax"`
}

// TaxGroups splits the positions by VAT rate.
//
// The tax is rounded once per rate group, not per position. Rounding per
// position and summing afterwards produces a total that differs from the tax on
// the invoice total by a cent or two — which is exactly the difference that
// later leaves an open item that never closes.
func (inv *Invoice) TaxGroups() []TaxGroup {
	nets := map[TaxRate]Cents{}
	for i := range inv.Items {
		nets[inv.Items[i].TaxRate] += inv.Items[i].TotalNet()
	}

	rates := make([]TaxRate, 0, len(nets))
	for r := range nets {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })

	groups := make([]TaxGroup, 0, len(rates))
	for _, r := range rates {
		net := nets[r]
		tax := Cents(0)
		// An exempt or shifted-liability invoice shows no tax regardless of the
		// rate stored on its positions.
		if inv.TaxTreatment == TaxTreatmentDomestic {
			tax = r.Tax(net)
		}
		groups = append(groups, TaxGroup{Rate: r, Net: net, Tax: tax})
	}
	return groups
}

// Recalculate derives the invoice totals from its positions.
func (inv *Invoice) Recalculate() {
	var net, tax Cents
	for _, g := range inv.TaxGroups() {
		net += g.Net
		tax += g.Tax
	}
	inv.NetAmount = net
	inv.TaxAmount = tax
	inv.GrossAmount = net + tax
}

// Validate checks the mandatory content of an invoice.
func (inv *Invoice) Validate() error {
	// § 33 UStDV lässt die Angabe des Leistungsempfängers bei einer
	// Kleinbetragsrechnung weg — das ist der ganze Sinn der Vorschrift, und ein
	// Barverkauf hat keinen erfassten Kunden.
	if inv.ContactID == 0 && !inv.SmallAmount {
		return fmt.Errorf("Rechnungsempfänger fehlt")
	}
	if len(inv.Items) == 0 {
		return fmt.Errorf("die Rechnung hat keine Positionen")
	}
	if inv.Date == "" {
		return fmt.Errorf("Rechnungsdatum fehlt")
	}
	if inv.ServiceDateFrom == "" || inv.ServiceDateTo == "" {
		return fmt.Errorf("Leistungsdatum fehlt (der Leistungszeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG)")
	}
	if inv.ServiceDateTo < inv.ServiceDateFrom {
		return fmt.Errorf("Leistungsende liegt vor dem Leistungsbeginn")
	}
	// § 14c UStG: ein ausgewiesener Steuerbetrag zu einem Steuerfall, bei dem
	// keine Steuer entsteht, wird trotzdem geschuldet. Die Rechnung wird deshalb
	// gar nicht erst ausgestellt — die Berichtigung setzt später die Zustimmung
	// des Finanzamts voraus (§ 14c Abs. 2 Sätze 3 bis 5 UStG).
	if inv.TaxAmount != 0 && !inv.TaxTreatment.MayShowTax() {
		return fmt.Errorf(
			"die Rechnung weist %s € Umsatzsteuer aus, der Steuerfall %q lässt aber keine entstehen. "+
				"Ein solcher Ausweis wird nach § 14c UStG trotzdem geschuldet – wähle entweder den "+
				"steuerpflichtigen Inlandsumsatz oder nimm den Steuerausweis heraus",
			inv.TaxAmount, inv.TaxTreatment)
	}
	for i := range inv.Items {
		it := &inv.Items[i]
		if it.Description == "" {
			return fmt.Errorf("Position %d: Beschreibung fehlt", i+1)
		}
		if it.QuantityMilli == 0 {
			return fmt.Errorf("Position %d: Menge darf nicht null sein", i+1)
		}
		if _, ok := ResolveUnitCode(it.Unit); !ok {
			return fmt.Errorf(
				"Position %d: %q ist keine bekannte Mengeneinheit. EN 16931 verlangt einen Schlüssel aus UN/ECE Rec. 20 (BT-130)",
				i+1, it.Unit)
		}
	}
	return nil
}

// ValidateParties checks the mandatory particulars that depend on the master
// data of both sides (§ 14 Abs. 4 UStG).
//
// It sits apart from Validate because Validate knows only the invoice. Whether
// the recipient has a complete address and whether the issuer can be identified
// at all is a question to the Stammdaten, and it is the question that decides
// whether the recipient keeps their input tax deduction.
func (inv *Invoice) ValidateParties(sellerTaxNumber, sellerVatID string, buyer *Contact) error {
	if sellerTaxNumber == "" && sellerVatID == "" {
		return fmt.Errorf(
			"auf der Rechnung fehlt die Steuernummer oder die USt-IdNr. des Ausstellers " +
				"(§ 14 Abs. 4 Nr. 2 UStG). Beides steht in den Unternehmensdaten")
	}
	// Die Kleinbetragsrechnung braucht den Empfänger nicht (§ 33 UStDV) — und
	// ohne erfassten Kontakt gibt es auch nichts zu prüfen.
	if inv.SmallAmount || buyer == nil {
		return nil
	}
	street, postalCode, city := buyer.PostalAddress()
	switch {
	case street == "":
		return fmt.Errorf(
			"%s hat keine Straße hinterlegt. Die vollständige Anschrift des Empfängers ist Pflichtangabe "+
				"(§ 14 Abs. 4 Nr. 1 UStG); ohne sie verliert der Empfänger den Vorsteuerabzug", buyer.Name)
	case postalCode == "":
		return fmt.Errorf(
			"%s hat keine Postleitzahl hinterlegt. Die vollständige Anschrift des Empfängers ist Pflichtangabe "+
				"(§ 14 Abs. 4 Nr. 1 UStG)", buyer.Name)
	case city == "":
		return fmt.Errorf(
			"%s hat keinen Ort hinterlegt. Die vollständige Anschrift des Empfängers ist Pflichtangabe "+
				"(§ 14 Abs. 4 Nr. 1 UStG)", buyer.Name)
	}
	// § 14a Abs. 1 und 3 UStG: bei der innergemeinschaftlichen Lieferung und
	// bei der Steuerschuldnerschaft des Leistungsempfängers ist dessen
	// USt-IdNr. anzugeben. Sie ist zugleich materielle Voraussetzung der
	// Steuerbefreiung (§ 6a Abs. 1 Nr. 4 UStG).
	if inv.RequiresBuyerVatID() && buyer.VatID == "" {
		return fmt.Errorf(
			"beim Steuerfall %q gehört die USt-IdNr. des Empfängers auf die Rechnung (§ 14a UStG), "+
				"bei %s ist keine hinterlegt", inv.TaxTreatment, buyer.Name)
	}
	return nil
}

// RequiresBuyerVatID reports whether the recipient's VAT identification number
// belongs on the document.
func (inv *Invoice) RequiresBuyerVatID() bool {
	switch inv.TaxTreatment {
	case TaxTreatmentIntraCommunitySupply, TaxTreatmentReverseChargeSupply:
		return true
	}
	return false
}

// Negate turns the invoice into its own Storno: same content, negated amounts.
//
// It works on the receiver, so the caller passes a copy. A Stornorechnung with
// the original amounts and a note saying "please ignore" is not a Storno — the
// recipient's system books what the numbers say.
func (inv *Invoice) Negate() {
	for i := range inv.Items {
		inv.Items[i].QuantityMilli = -inv.Items[i].QuantityMilli
	}
	inv.Recalculate()
	inv.PrepaidAmount = -inv.PrepaidAmount
}

// InvoiceRepository defines persistence operations for invoices.
type InvoiceRepository interface {
	FindAll(ctx context.Context, fiscalYear int) ([]Invoice, error)
	FindByID(ctx context.Context, id uint) (*Invoice, error)
	FindByNumber(ctx context.Context, number string) (*Invoice, error)
	// FindNumbers liefert die vergebenen Rechnungsnummern eines
	// Geschäftsjahres. Der Lückenbericht braucht sie und nicht die ganzen
	// Rechnungen: er vergleicht Zähler und Nummern.
	FindNumbers(ctx context.Context, fiscalYear int) ([]string, error)
	// FindByGroup liefert die Rechnungen eines Rechnungsverbunds.
	FindByGroup(ctx context.Context, groupID uint) ([]Invoice, error)
	Save(ctx context.Context, invoice *Invoice) error
	UpdateStatus(ctx context.Context, id uint, status InvoiceStatus) error
	// UpdateTransportKind hält fest, wer den Gegenstand befördert hat.
	//
	// Ein eigener Schreibweg und kein Save der ganzen Rechnung: die Angabe
	// entsteht beim Ablegen eines Nachweisbelegs, oft lange nach dem Ausstellen,
	// und eine ausgestellte Rechnung im Ganzen zurückzuschreiben hieße, ihre
	// Positionen und ihre Beträge anzufassen, an denen sich nichts geändert hat.
	UpdateTransportKind(ctx context.Context, id uint, kind string) error
	Count(ctx context.Context, fiscalYear int) (int64, error)
}

// InvoiceRenderer renders invoices into Typst markup.
type InvoiceRenderer interface {
	RenderTypst(invoice *Invoice, seller *CompanySettings, buyer *Contact) (string, error)
}

// ZUGFeRDGenerator creates Factur-X / ZUGFeRD compliant XML.
type ZUGFeRDGenerator interface {
	GenerateXML(invoice *Invoice, seller *CompanySettings, buyer *Contact) (string, error)
	// Die EN-16931-Prüfung sitzt in internal/invoice/en16931.go und deckt eine
	// belegte Teilmenge der Regeln ab. Vollständige Schematron-Äquivalenz bliebe
	// offen — sie setzt einen XSLT-2.0-Prozessor voraus, den Go nicht hat.
}
