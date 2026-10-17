package domain

import "time"

// RetentionClass ist die Aufbewahrungsklasse eines Objekts.
//
// Drei Klassen und nicht eine Frist je Objektart: § 257 Abs. 4 HGB und § 147
// Abs. 3 AO kennen genau diese Staffelung, und wer sie an jeder Objektart
// einzeln hinterlegte, hätte dieselbe Regel an zwanzig Stellen — und beim
// nächsten Gesetz zwanzig Stellen zu ändern.
type RetentionClass string

const (
	// RetentionClassBooks sind die Handelsbücher, Inventare, Eröffnungsbilanzen,
	// Jahresabschlüsse, Lageberichte und die zu ihrem Verständnis erforderlichen
	// Arbeitsanweisungen und Organisationsunterlagen: zehn Jahre (§ 257 Abs. 4
	// i. V. m. Abs. 1 Nr. 1 HGB, § 147 Abs. 3 Satz 1 i. V. m. Abs. 1 Nr. 1 AO).
	RetentionClassBooks RetentionClass = "books"
	// RetentionClassVouchers sind die Buchungsbelege und die empfangenen und
	// abgesandten Rechnungen: acht Jahre seit dem Vierten
	// Bürokratieentlastungsgesetz (§ 257 Abs. 4 HGB, § 147 Abs. 3 Satz 1 AO in
	// der Fassung ab 1.1.2025).
	RetentionClassVouchers RetentionClass = "vouchers"
	// RetentionClassLetters sind die empfangenen und abgesandten Handelsbriefe
	// und die sonstigen Unterlagen, soweit sie für die Besteuerung von Bedeutung
	// sind: sechs Jahre (§ 257 Abs. 4 HGB, § 147 Abs. 3 Satz 1 AO).
	RetentionClassLetters RetentionClass = "letters"
	// RetentionClassNone steht an Objekten, für die keine Frist gilt oder deren
	// Klasse (noch) nicht bestimmt ist.
	RetentionClassNone RetentionClass = ""
)

// Label ist der Klartext für die Oberfläche und das Schlüsselverzeichnis.
func (c RetentionClass) Label() string {
	switch c {
	case RetentionClassBooks:
		return "Handelsbücher und Abschlüsse (10 Jahre)"
	case RetentionClassVouchers:
		return "Buchungsbelege und Rechnungen (8 Jahre)"
	case RetentionClassLetters:
		return "Handelsbriefe und sonstige Unterlagen (6 Jahre)"
	default:
		return "Ohne Aufbewahrungsklasse"
	}
}

// AllRetentionClasses listet die Klassen in fester Reihenfolge.
func AllRetentionClasses() []RetentionClass {
	return []RetentionClass{RetentionClassBooks, RetentionClassVouchers, RetentionClassLetters}
}

// RetentionKind ist die Art des aufzubewahrenden Objekts. Aus ihr folgt die
// Klasse.
type RetentionKind string

const (
	RetentionKindJournal          RetentionKind = "journal"
	RetentionKindFestschreibung   RetentionKind = "festschreibung"
	RetentionKindClosing          RetentionKind = "closing"
	RetentionKindVatReturn        RetentionKind = "vat_return"
	RetentionKindReceiptInvoice   RetentionKind = "receipt_invoice"
	RetentionKindReceiptStatement RetentionKind = "receipt_statement"
	RetentionKindReceiptSelfIssue RetentionKind = "receipt_self_issued"
	RetentionKindReceiptLetter    RetentionKind = "receipt_letter"
	RetentionKindReceiptOther     RetentionKind = "receipt_other"
)

// RetentionKindOf ordnet einer Belegart ihre Aufbewahrungsart zu.
func RetentionKindOf(k ReceiptKind) RetentionKind {
	switch k {
	case ReceiptKindStatement:
		return RetentionKindReceiptStatement
	case ReceiptKindSelfIssued:
		return RetentionKindReceiptSelfIssue
	case ReceiptKindLetter:
		return RetentionKindReceiptLetter
	case ReceiptKindOther:
		return RetentionKindReceiptOther
	default:
		return RetentionKindReceiptInvoice
	}
}

// RetentionInfo ist die Frist eines einzelnen Objekts.
type RetentionInfo struct {
	Kind  RetentionKind  `json:"kind"`
	Class RetentionClass `json:"class"`
	// Years ist die Frist in Jahren, OriginYear das Entstehungsjahr.
	Years      int `json:"years"`
	OriginYear int `json:"originYear"`
	// EarliestDeletion ist der erste Tag, an dem gelöscht werden darf:
	// der 31.12. des letzten Aufbewahrungsjahres ist der letzte Tag der Frist,
	// gelöscht werden darf ab dem Tag danach. Format YYYY-MM-DD.
	//
	// Der Fristbeginn ist der Schluss des Kalenderjahres, in dem die letzte
	// Eintragung gemacht oder der Beleg entstanden ist (§ 257 Abs. 5 HGB, § 147
	// Abs. 4 AO) — deshalb wird ab dem 31.12. des Entstehungsjahres gerechnet
	// und nicht ab dem Belegdatum.
	EarliestDeletion string `json:"earliestDeletion"`
	// RetentionEnd ist der letzte Tag, an dem aufzubewahren ist.
	RetentionEnd string `json:"retentionEnd"`
	LegalBasis   string `json:"legalBasis"`
	Note         string `json:"note"`
}

// EarliestDeletionAfter liefert den ersten Tag, an dem gelöscht werden darf,
// aus dem letzten Aufbewahrungstag.
//
// Die beiden Tage liegen genau einen Tag auseinander, und die Verwechslung ist
// die naheliegendste in diesem ganzen Bereich: „aufzubewahren bis 31.12.2033"
// und „löschbar ab 01.01.2034" sind dieselbe Frist, aber „löschbar ab
// 31.12.2033" wäre eine Löschung einen Tag zu früh. Statt sich darauf zu
// verlassen, dass jede Anzeige richtig beschriftet, wird der zweite Tag
// mitgeliefert.
//
// Leerer oder unlesbarer Eingabewert ergibt einen leeren Ausgabewert: ein
// erfundenes Datum wäre eine Aussage über eine Frist, die niemand bestimmt hat.
func EarliestDeletionAfter(retentionUntil string) string {
	day, err := time.Parse("2006-01-02", retentionUntil)
	if err != nil {
		return ""
	}
	return day.AddDate(0, 0, 1).Format("2006-01-02")
}
