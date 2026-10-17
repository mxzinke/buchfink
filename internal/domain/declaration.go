package domain

import (
	"context"
	"time"
)

// DeclarationInput ist eine Angabe zu einer Jahressteuererklärung, die nicht
// in den Büchern steht: ein Wert aus einem Bescheid, eine Entscheidung des
// Menschen, eine Beteiligungsquote. Je Erklärung, Jahr und Zeile gibt es
// höchstens eine; ihre Geschichte steht im Änderungsprotokoll.
type DeclarationInput struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Kind string `gorm:"size:20;not null;uniqueIndex:idx_declaration_input" json:"kind"`
	Year int    `gorm:"not null;uniqueIndex:idx_declaration_input" json:"year"`
	Key  string `gorm:"size:100;not null;uniqueIndex:idx_declaration_input" json:"key"`
	// ValueJSON ist der Wert im Format der Erklärung (Betrag, Text, ja/nein,
	// Datum).
	ValueJSON string `gorm:"type:text;not null" json:"value"`
	Source    string `gorm:"size:20;not null" json:"source"`
	ReceiptID *uint  `gorm:"index" json:"receiptId,omitempty"`
	Note      string `gorm:"size:500" json:"note,omitempty"`
	// Channel ist der Erfassungsweg, Actor die Bearbeiterkennung.
	Channel   string    `gorm:"size:60;not null" json:"channel"`
	Actor     string    `gorm:"size:120" json:"actor"`
	EnteredAt time.Time `gorm:"not null" json:"enteredAt"`
	// Confirmed: der Wert gilt; ein unbestätigter ist ein Vorschlag, den ein
	// Mensch in der App bestätigt.
	Confirmed   bool       `gorm:"not null;default:false" json:"confirmed"`
	ConfirmedBy string     `gorm:"size:120" json:"confirmedBy,omitempty"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
}

// DeclarationRecord ist ein festgehaltener Stand einer Erklärung: ein Export
// oder die in der App erfasste Übermittlung. Er wird nie geändert; ein neuer
// Stand ist ein neuer Eintrag.
type DeclarationRecord struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Kind string `gorm:"size:20;not null;index:idx_declaration_record" json:"kind"`
	Year int    `gorm:"not null;index:idx_declaration_record" json:"year"`
	// Status ist „exportiert“ oder „uebermittelt“.
	Status         string `gorm:"size:20;not null" json:"status"`
	FiguresHash    string `gorm:"size:64;not null" json:"figuresHash"`
	FormYear       int    `gorm:"not null" json:"formYear"`
	CatalogVersion string `gorm:"size:40" json:"catalogVersion"`
	ProgramVersion string `gorm:"size:60" json:"programVersion"`
	JournalHash    string `gorm:"size:64" json:"journalHash"`
	// SnapshotJSON ist die Erklärung, wie sie exportiert oder übermittelt
	// wurde.
	SnapshotJSON string `gorm:"type:text;not null" json:"-"`
	ExportFile   string `gorm:"size:500" json:"exportFile,omitempty"`
	ExportSHA256 string `gorm:"size:64" json:"exportSha256,omitempty"`
	// SubmittedAt und TransferTicket belegen die Übermittlung in Mein ELSTER.
	SubmittedAt    string `gorm:"size:10" json:"submittedAt,omitempty"`
	TransferTicket string `gorm:"size:100" json:"transferTicket,omitempty"`
	// OverrideReason begründet eine Übermittlung trotz blockierender Befunde.
	OverrideReason string    `gorm:"size:1000" json:"overrideReason,omitempty"`
	Channel        string    `gorm:"size:60;not null" json:"channel"`
	Actor          string    `gorm:"size:120" json:"actor"`
	CreatedAt      time.Time `json:"createdAt"`
}

// DeclarationRepository legt Eingaben und Stände der Erklärungen ab.
type DeclarationRepository interface {
	FindInputs(ctx context.Context, kind string, year int) ([]DeclarationInput, error)
	// FindInput liefert nil ohne Fehler, wenn es keine Eingabe gibt.
	FindInput(ctx context.Context, kind string, year int, key string) (*DeclarationInput, error)
	SaveInput(ctx context.Context, input *DeclarationInput) error
	DeleteInput(ctx context.Context, id uint) error
	CreateRecord(ctx context.Context, record *DeclarationRecord) error
	// FindRecords liefert die Stände eines Jahres, älteste zuerst.
	FindRecords(ctx context.Context, kind string, year int) ([]DeclarationRecord, error)
}
