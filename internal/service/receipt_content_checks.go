package service

import (
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// invoiceRelief ist die Erleichterung der UStDV, die für eine Eingangsrechnung
// gilt. Sie verlangt Name und Anschrift des Leistenden, das Ausstellungsdatum
// und Entgelt samt Steuer in einer Summe, aber weder Steuernummer noch
// Leistungsempfänger, Leistungsdatum oder gesonderten Steuerausweis. Die
// Tankquittung über 60 € ist so eine Rechnung; Befunde zu dem, was sie nicht
// braucht, sperrten den Vorsteuerabzug zu Unrecht.
type invoiceRelief string

const (
	reliefNone        invoiceRelief = ""
	reliefSmallAmount invoiceRelief = "small_amount" // § 33 UStDV
	reliefTicket      invoiceRelief = "ticket"       // § 34 UStDV
)

// norm nennt die Vorschrift der Erleichterung zu einer Pflichtangabe der
// Nummer nr des § 33 bzw. § 34 Abs. 1 UStDV.
func (r invoiceRelief) norm(nr string) string {
	switch r {
	case reliefSmallAmount:
		return "§ 33 Satz 1 Nr. " + nr + " UStDV"
	case reliefTicket:
		return "§ 34 Abs. 1 Nr. " + nr + " UStDV"
	}
	return ""
}

// reliefOf bestimmt die Erleichterung eines Eingangsbelegs. Der Fahrausweis
// folgt der Belegart; die Kleinbetragsrechnung dem Gesamtbetrag in Euro bis
// zur Grenze am Rechnungsdatum (250 € seit 2017). booked ist der gebuchte
// Bruttobetrag, soweit es schon Buchungszeilen gibt; liegt er über der Grenze,
// gilt die Erleichterung auch dann nicht, wenn die Kopfdaten weniger nennen.
// Für Leistungen nach § 13b gilt § 33 nicht (Satz 3); dort hängt die Vorsteuer
// aber auch nicht an einer Rechnung nach § 14 UStG, und der Buchungsweg prüft
// sie nicht.
func reliefOf(receipt *domain.Receipt, booked domain.Cents) invoiceRelief {
	if receipt == nil {
		return reliefNone
	}
	switch receipt.Kind {
	case domain.ReceiptKindTicket:
		return reliefTicket
	case domain.ReceiptKindInvoice, "":
	default:
		return reliefNone
	}
	if currency := strings.ToUpper(strings.TrimSpace(receipt.Currency)); currency != "" && currency != "EUR" {
		return reliefNone
	}
	if receipt.GrossAmount <= 0 {
		return reliefNone
	}
	params, err := accounting.TaxParametersFor(receipt.DocumentDate)
	if err != nil || receipt.GrossAmount > params.SmallAmountInvoiceLimit || booked > params.SmallAmountInvoiceLimit {
		return reliefNone
	}
	return reliefSmallAmount
}

// bookedGross ist der Bruttobetrag eines Eingangsbelegs aus seinen Zeilen:
// die Summe der Sollseite.
func bookedGross(lines []domain.JournalLine) domain.Cents {
	var gross domain.Cents
	for _, l := range lines {
		if l.Side == domain.SideDebit {
			gross += l.Amount
		}
	}
	return gross
}

// Die Pflichtangaben der Rechnung gegen die Stammdaten (RECH-07 K2).
//
// Dieselbe Prüfung entscheidet an zwei Stellen: der Buchungsweg hält die
// Vorsteuer an, wenn eine Pflichtangabe fehlt (siehe inputTaxFindings), und die
// Klärungsliste zeigt denselben Mangel als Inhaltsfehler. Zwei Fassungen davon
// gingen auseinander — dann hielte der Buchungsweg eine Buchung an, die die
// Klärungsliste für beanstandungsfrei erklärt, oder umgekehrt. Deshalb steht
// die Regel hier einmal, und beide Seiten kleiden sie nur verschieden ein.

// masterDataDefect ist eine fehlende Pflichtangabe, gemessen an den Stammdaten.
type masterDataDefect struct {
	// Code ist die Regelkennung; sie ist an beiden Stellen dieselbe, damit ein
	// Befund der Klärungsliste und ein angehaltener Buchungsweg als derselbe
	// Mangel erkennbar sind.
	Code string
	// Title ist die Überschrift, Detail der Satz „es fehlt: …".
	Title  string
	Detail string
	// Norm ist die Fundstelle der Pflichtangabe.
	Norm string
	// Fixable sagt, ob eine Ergänzung der Stammdaten genügt — dann ist die
	// Übersteuerung nicht der erste Weg.
	Fixable bool
}

// issuerMasterDataDefects prüft die Angaben zum Aussteller (§ 14 Abs. 4 Nr. 1
// und 2 UStG) gegen den Kontakt. Unter einer Erleichterung der §§ 33, 34 UStDV
// bleiben Name und Anschrift verlangt, die Steuernummer nicht.
func issuerMasterDataDefects(contact *domain.Contact, relief invoiceRelief) []masterDataDefect {
	out := make([]masterDataDefect, 0, 3)
	if contact == nil {
		return out
	}
	norm := "§ 14 Abs. 4 Nr. 1 UStG"
	if relief != reliefNone {
		norm = relief.norm("1")
	}
	if strings.TrimSpace(contact.Name) == "" {
		out = append(out, masterDataDefect{
			Code:   findingSupplierName,
			Title:  "Der Name des Ausstellers fehlt",
			Detail: "der vollständige Name des leistenden Unternehmers (" + norm + ").",
			Norm:   norm,
		})
	}
	if !contact.HasCompleteAddress() {
		out = append(out, masterDataDefect{
			Code:    findingIssuerAddress,
			Title:   "Die Anschrift des Ausstellers ist unvollständig",
			Fixable: true,
			Detail: "die vollständige Anschrift von " + contactLabel(contact) +
				" — Straße, Postleitzahl und Ort (" + norm + ").",
			Norm: norm,
		})
	}
	if relief != reliefNone {
		return out
	}
	if strings.TrimSpace(contact.TaxID) == "" && strings.TrimSpace(contact.VatID) == "" {
		out = append(out, masterDataDefect{
			Code:    findingIssuerTaxNumber,
			Title:   "Steuernummer oder USt-IdNr. des Ausstellers fehlt",
			Fixable: true,
			Detail: "die vom Finanzamt erteilte Steuernummer oder die USt-IdNr. von " +
				contactLabel(contact) + " (§ 14 Abs. 4 Nr. 2 UStG).",
			Norm: "§ 14 Abs. 4 Nr. 2 UStG",
		})
	}
	return out
}

// recipientMasterDataDefects prüft die Angaben zum Leistungsempfänger
// (§ 14 Abs. 4 Nr. 1 UStG) gegen die eigenen Unternehmensdaten.
//
// Gegen die eigenen Stammdaten und nicht gegen das Dokument: was auf der
// Rechnung steht, kann Buchfink bei einem Papierscan nicht lesen. Sind die
// eigenen Angaben unvollständig, ist aber schon nicht feststellbar, ob die
// Rechnung auf den richtigen Empfänger lautet — und das ist der Mangel, den
// diese Prüfung meldet.
func recipientMasterDataDefects(company *domain.CompanySettings) []masterDataDefect {
	out := make([]masterDataDefect, 0, 1)
	if company == nil {
		return out
	}
	missing := make([]string, 0, 3)
	if strings.TrimSpace(company.CompanyName) == "" {
		missing = append(missing, "der Name")
	}
	if strings.TrimSpace(company.Street) == "" {
		missing = append(missing, "die Straße")
	}
	if strings.TrimSpace(company.PostalCode) == "" || strings.TrimSpace(company.City) == "" {
		missing = append(missing, "Postleitzahl und Ort")
	}
	if len(missing) == 0 {
		return out
	}
	out = append(out, masterDataDefect{
		Code:    "content_recipient_address",
		Title:   "Die eigene Anschrift ist unvollständig",
		Fixable: true,
		Detail: "die vollständige Anschrift des Leistungsempfängers in den Stammdaten (" +
			strings.Join(missing, ", ") + ") — ohne sie ist nicht festzustellen, ob die " +
			"Rechnung auf den richtigen Empfänger lautet (§ 14 Abs. 4 Nr. 1 UStG).",
		Norm: "§ 14 Abs. 4 Nr. 1 UStG",
	})
	return out
}

// contactLabel nennt den Kontakt beim Namen, ersatzweise als „dem Aussteller".
func contactLabel(contact *domain.Contact) string {
	if contact == nil || strings.TrimSpace(contact.Name) == "" {
		return "dem Aussteller"
	}
	return contact.Name
}
