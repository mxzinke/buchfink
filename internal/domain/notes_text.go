package domain

import (
	"context"
	"fmt"
	"time"
)

// NotesSection ist ein Abschnitt des Anhangs.
//
// Der Anhang besteht zum großen Teil aus Text, den kein Programm errechnen
// kann: die angewandten Bilanzierungs- und Bewertungsmethoden (§ 284 Abs. 2
// Nr. 1 HGB), Haftungsverhältnisse (§ 268 Abs. 7 HGB), der Nachtragsbericht
// (§ 285 Nr. 33 HGB). Buchfink hält ihn als Freitext je Abschnitt vor, weil ein
// einziges großes Textfeld weder gegliedert noch ins Folgejahr übernehmbar
// wäre.
type NotesSection string

const (
	NotesSectionMethods       NotesSection = "methods"       // Bilanzierungs- und Bewertungsmethoden
	NotesSectionBoard         NotesSection = "board"         // Organbezüge
	NotesSectionSubsequent    NotesSection = "subsequent"    // Nachtragsbericht
	NotesSectionCommitments   NotesSection = "commitments"   // Sonstige finanzielle Verpflichtungen
	NotesSectionContingent    NotesSection = "contingent"    // Haftungsverhältnisse
	NotesSectionInvestments   NotesSection = "investments"   // Beteiligungen
	NotesSectionAppropriation NotesSection = "appropriation" // Vorschlag zur Ergebnisverwendung
)

// NotesSectionDefinition beschreibt einen Abschnitt.
type NotesSectionDefinition struct {
	Section NotesSection `json:"section"`
	Label   string       `json:"label"`
	Hint    string       `json:"hint"`
	// Basis nennt die Vorschrift, aus der die Angabe folgt.
	Basis string `json:"basis"`
}

// AllNotesSections liefert die Abschnitte in der Reihenfolge des Anhangs.
func AllNotesSections() []NotesSectionDefinition {
	return []NotesSectionDefinition{
		{NotesSectionMethods, "Bilanzierungs- und Bewertungsmethoden",
			"Welche Methoden angewandt wurden — Abschreibungsverfahren, Bewertung der Vorräte, Abzinsung der Rückstellungen.",
			"§ 284 Abs. 2 Nr. 1 HGB"},
		{NotesSectionBoard, "Organbezüge",
			"Die Gesamtbezüge der Geschäftsführung. Kleine Gesellschaften sind davon befreit.",
			"§ 285 Nr. 9 HGB, § 288 Abs. 1 HGB"},
		{NotesSectionSubsequent, "Nachtragsbericht",
			"Vorgänge von besonderer Bedeutung nach dem Bilanzstichtag.",
			"§ 285 Nr. 33 HGB"},
		{NotesSectionCommitments, "Sonstige finanzielle Verpflichtungen",
			"Miet-, Leasing- und Abnahmeverpflichtungen, die nicht in der Bilanz stehen.",
			"§ 285 Nr. 3a HGB"},
		{NotesSectionContingent, "Haftungsverhältnisse",
			"Bürgschaften, Garantien und Sicherheiten für fremde Verbindlichkeiten.",
			"§ 251, § 268 Abs. 7 HGB"},
		{NotesSectionInvestments, "Beteiligungen",
			"Name, Sitz, Anteil, Eigenkapital und Ergebnis der Beteiligungsunternehmen.",
			"§ 285 Nr. 11 HGB"},
		{NotesSectionAppropriation, "Vorschlag zur Ergebnisverwendung",
			"Der Vorschlag der Geschäftsführung an die Gesellschafter.",
			"§ 285 Nr. 34 HGB"},
	}
}

// NotesSectionDefinitionFor sucht die Beschreibung zu einem Abschnitt.
func NotesSectionDefinitionFor(section NotesSection) (NotesSectionDefinition, bool) {
	for _, def := range AllNotesSections() {
		if def.Section == section {
			return def, true
		}
	}
	return NotesSectionDefinition{}, false
}

// NotesText ist der Freitext eines Anhangabschnitts in einem Geschäftsjahr.
type NotesText struct {
	Year    int          `gorm:"primaryKey;autoIncrement:false" json:"year"`
	Section NotesSection `gorm:"primaryKey;size:30" json:"section"`
	Text    string       `gorm:"type:text;serializer:encrypted" json:"text"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate prüft den Abschnitt.
func (n *NotesText) Validate() error {
	if n.Year <= 0 {
		return fmt.Errorf("zum Anhangtext gehört ein Geschäftsjahr")
	}
	if _, ok := NotesSectionDefinitionFor(n.Section); !ok {
		return fmt.Errorf("unbekannter Anhangabschnitt %q", n.Section)
	}
	return nil
}

// NotesTextRepository persistiert die Anhangtexte.
type NotesTextRepository interface {
	FindByYear(ctx context.Context, year int) ([]NotesText, error)
	Save(ctx context.Context, text *NotesText) error
}
