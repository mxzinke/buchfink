package service

import (
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

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
// und 2 UStG) gegen den Kontakt.
func issuerMasterDataDefects(contact *domain.Contact) []masterDataDefect {
	out := make([]masterDataDefect, 0, 3)
	if contact == nil {
		return out
	}
	if strings.TrimSpace(contact.Name) == "" {
		out = append(out, masterDataDefect{
			Code:   findingSupplierName,
			Title:  "Der Name des Ausstellers fehlt",
			Detail: "der vollständige Name des leistenden Unternehmers (§ 14 Abs. 4 Nr. 1 UStG).",
			Norm:   "§ 14 Abs. 4 Nr. 1 UStG",
		})
	}
	if !contact.HasCompleteAddress() {
		out = append(out, masterDataDefect{
			Code:    findingIssuerAddress,
			Title:   "Die Anschrift des Ausstellers ist unvollständig",
			Fixable: true,
			Detail: "die vollständige Anschrift von " + contactLabel(contact) +
				" — Straße, Postleitzahl und Ort (§ 14 Abs. 4 Nr. 1 UStG).",
			Norm: "§ 14 Abs. 4 Nr. 1 UStG",
		})
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
	if strings.TrimSpace(company.ZipCity) == "" {
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
