package domain

import (
	"context"
	"fmt"
	"time"
)

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
	RetentionKindInventory        RetentionKind = "inventory"
	RetentionKindAssetDocument    RetentionKind = "asset_document"
	RetentionKindOrganisation     RetentionKind = "organisation"
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

// IsExpired meldet, ob die Frist am Stichtag abgelaufen ist. Leerer Stichtag
// heißt: heute.
func (r RetentionInfo) IsExpired(today string) bool {
	if r.EarliestDeletion == "" {
		return false
	}
	if today == "" {
		today = time.Now().UTC().Format("2006-01-02")
	}
	return today >= r.EarliestDeletion
}

// RetentionHoldReason ist der Grund, aus dem eine Frist ausgesetzt ist.
type RetentionHoldReason string

const (
	// RetentionHoldAudit ist die Außenprüfung: § 147 Abs. 3 Satz 5 AO lässt die
	// Frist nicht ablaufen, solange die Unterlagen für begonnene Außenprüfungen
	// von Bedeutung sind.
	RetentionHoldAudit RetentionHoldReason = "audit"
	// RetentionHoldAppeal ist das anhängige Rechtsbehelfsverfahren.
	RetentionHoldAppeal RetentionHoldReason = "appeal"
	// RetentionHoldOther ist jeder andere Grund; er ist zu beschreiben.
	RetentionHoldOther RetentionHoldReason = "other"
)

// Label ist der Klartext für die Oberfläche.
func (r RetentionHoldReason) Label() string {
	switch r {
	case RetentionHoldAudit:
		return "Außenprüfung"
	case RetentionHoldAppeal:
		return "Rechtsbehelfsverfahren"
	default:
		return "Sonstiger Grund"
	}
}

// RetentionHold setzt die Aufbewahrungsfrist eines Geschäftsjahres aus.
//
// Ohne ihn wäre die Löschfunktion gefährlich: § 147 Abs. 3 Satz 5 AO lässt die
// Frist nicht ablaufen, solange die Unterlagen für eine begonnene Außenprüfung,
// eine vorläufige Steuerfestsetzung, ein anhängiges Rechtsbehelfsverfahren oder
// eine Straf- oder Bußgeldsache von Bedeutung sind — und das weiß nur der
// Unternehmer, nicht das Programm. Der Hold ist die Stelle, an der er es sagt.
type RetentionHold struct {
	ID         uint                `gorm:"primaryKey" json:"id"`
	FiscalYear int                 `gorm:"index;not null" json:"fiscalYear"`
	Reason     RetentionHoldReason `gorm:"size:20;not null" json:"reason"`
	// Description ist die Erläuterung, bei „sonstiger Grund" Pflicht.
	Description string `gorm:"size:500;serializer:encrypted" json:"description"`

	SetAt time.Time `gorm:"not null" json:"setAt"`
	SetBy string    `gorm:"size:120;not null" json:"setBy"`

	// ReleasedAt bleibt leer, solange der Hold gilt. Ein aufgehobener Hold wird
	// nicht gelöscht: dass eine Frist einmal ausgesetzt war, gehört zur
	// Geschichte der Daten.
	ReleasedAt    *time.Time `json:"releasedAt,omitempty"`
	ReleasedBy    string     `gorm:"size:120" json:"releasedBy,omitempty"`
	ReleaseReason string     `gorm:"size:500;serializer:encrypted" json:"releaseReason,omitempty"`
	AppVersion    string     `gorm:"size:60" json:"appVersion,omitempty"`
	// AffectedNote hält in einem Satz fest, worauf der Hold beim Setzen wirkte:
	// wie viele Buchungen, Belege und Dateien des Jahres seine Frist aussetzt.
	// Ein Satz und keine Zählstruktur — die Zahlen des Berichts kommen bei
	// jedem Aufruf frisch aus der Datenbank (siehe RetentionService), während
	// diese hier den Stand zum Zeitpunkt der Anordnung festhält.
	AffectedNote string `gorm:"size:500" json:"affectedNote,omitempty"`
}

// IsActive meldet, ob der Hold noch gilt.
func (h *RetentionHold) IsActive() bool { return h.ReleasedAt == nil }

// Validate prüft, was ein Hold mindestens braucht.
func (h *RetentionHold) Validate() error {
	if h.FiscalYear < 1900 || h.FiscalYear > 2200 {
		return fmt.Errorf("ungültiges Geschäftsjahr %d", h.FiscalYear)
	}
	switch h.Reason {
	case RetentionHoldAudit, RetentionHoldAppeal:
	case RetentionHoldOther:
		if h.Description == "" {
			return fmt.Errorf("ein sonstiger Grund ist zu beschreiben — sonst lässt sich später nicht beurteilen, ob er noch gilt")
		}
	default:
		return fmt.Errorf("unbekannter Grund %q für die Aussetzung der Frist", h.Reason)
	}
	return nil
}

// RetentionCounts sind die Objekte eines Geschäftsjahres, auf die sich eine
// Aufbewahrungsfrist oder eine Löschung bezieht.
type RetentionCounts struct {
	JournalEntries   int `json:"journalEntries"`
	JournalLines     int `json:"journalLines"`
	Receipts         int `json:"receipts"`
	ReceiptFiles     int `json:"receiptFiles"`
	Festschreibungen int `json:"festschreibungen"`
	CheckRuns        int `json:"checkRuns"`
	VatReturns       int `json:"vatReturns"`
	// Die übrigen Objektarten des Jahres. Sie stehen einzeln und nicht in einer
	// Summe, weil das Löschprotokoll sagen muss, was verschwunden ist: „ein
	// Geschäftsjahr gelöscht" ist keine Auskunft, „14 Rechnungen, 212
	// Bankumsätze, 3 Anlagenbewegungen" ist eine.
	Invoices         int `json:"invoices"`
	BankTransactions int `json:"bankTransactions"`
	AssetMovements   int `json:"assetMovements"`
	Accruals         int `json:"accruals"`
	Provisions       int `json:"provisions"`
	InventoryCounts  int `json:"inventoryCounts"`
	ZMReturns        int `json:"zmReturns"`
	InputTaxUsages   int `json:"inputTaxUsages"`
	NumberGaps       int `json:"numberGaps"`
	// ClearedReferences zählt die Verweise, die beim Löschen genullt wurden:
	// Objekte, die das Jahr überdauern — ein Anlagegut, ein noch laufender
	// Eintrag des § 15a-Verzeichnisses, eine Rechnung eines anderen Jahres —,
	// aber auf eine gelöschte Buchung oder einen gelöschten Beleg zeigten. Sie
	// gehören ins Protokoll: ein genullter Verweis ist eine Änderung an Daten,
	// die bleiben.
	ClearedReferences int `json:"clearedReferences"`
}

// IsEmpty meldet, ob das Jahr überhaupt Daten trägt.
func (c RetentionCounts) IsEmpty() bool {
	return c.JournalEntries == 0 && c.Receipts == 0 &&
		c.Festschreibungen == 0 && c.CheckRuns == 0 && c.VatReturns == 0 &&
		c.Invoices == 0 && c.BankTransactions == 0 && c.AssetMovements == 0 &&
		c.Accruals == 0 && c.Provisions == 0 && c.InventoryCounts == 0 &&
		c.ZMReturns == 0
}

// RetentionRepository persistiert die Aussetzungen und beantwortet, was ein
// Geschäftsjahr an aufzubewahrenden Daten trägt.
type RetentionRepository interface {
	CreateHold(ctx context.Context, hold *RetentionHold) error
	ReleaseHold(ctx context.Context, id uint, releasedBy, reason string, at time.Time) error
	FindHolds(ctx context.Context) ([]RetentionHold, error)
	// FindActiveHold liefert die geltende Aussetzung eines Jahres oder nil.
	FindActiveHold(ctx context.Context, fiscalYear int) (*RetentionHold, error)
	// CountObjects zählt, was ein Geschäftsjahr trägt.
	CountObjects(ctx context.Context, fiscalYear int) (RetentionCounts, error)
	// FiscalYearsWithObjects nennt aufsteigend jedes Geschäftsjahr, das
	// überhaupt aufzubewahrende Objekte trägt.
	//
	// Nicht nur die Jahre mit Buchungen: ein Jahr, in dem Belege abgelegt und
	// Handelsbriefe verwahrt, aber (noch) keine Buchungen erfasst wurden, hat
	// eine Aufbewahrungsfrist wie jedes andere. Fehlte es in der Übersicht,
	// liefe seine Frist unbemerkt, und der Bericht über abgelaufene Objekte
	// verschwiege genau die Objekte, für die er da ist.
	FiscalYearsWithObjects(ctx context.Context) ([]int, error)
	// DeleteFiscalYear löscht die Daten eines Geschäftsjahres in einer
	// Transaktion und liefert die gelöschten Zahlen sowie die Ablagepfade der
	// Belegdateien, auf die danach kein Beleg mehr zeigt.
	//
	// Die verwaisten Pfade werden zurückgegeben statt selbst gelöscht: der
	// Belegspeicher ist inhaltsadressiert, dieselbe Datei kann zu mehreren
	// Belegen gehören, und die Entscheidung, eine Datei von der Platte zu
	// nehmen, gehört in die Schicht, die den Speicher kennt.
	DeleteFiscalYear(ctx context.Context, fiscalYear int) (RetentionCounts, []string, error)
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
