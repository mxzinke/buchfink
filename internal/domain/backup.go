package domain

import (
	"context"
	"time"
)

// BackupKind ist der Anlass eines Sicherungslaufs.
//
// Der Prüflauf steht bewusst neben der Sicherung und nicht in einem eigenen
// Protokoll: die Frage „habe ich eine brauchbare Sicherung" beantwortet nur,
// wer Erstellung und Prüfung nebeneinander sieht. Eine Sicherung, die nie
// zurückgespielt werden konnte, ist keine (AO § 147 Abs. 2, GoBD Rz. 103).
type BackupKind string

const (
	// BackupKindManual ist ein vom Anwender ausgelöster Lauf.
	BackupKindManual BackupKind = "manual"
	// BackupKindAutomatic ist der Lauf beim Start bzw. beim Beenden.
	BackupKindAutomatic BackupKind = "automatic"
	// BackupKindVerify ist der Wiederherstellungstest: eine vorhandene
	// Sicherung wird in einen Temporärordner entpackt, geprüft und wieder
	// gelöscht.
	BackupKindVerify BackupKind = "verify"
	// BackupKindRestore ist die tatsächliche Wiederherstellung in einen leeren
	// Zielordner.
	BackupKindRestore BackupKind = "restore"
)

// BackupRun ist ein Lauf: Zeitpunkt, Ziel, Umfang, Ergebnis.
//
// Er steht in der Datenbank des Mandanten, damit die Oberfläche die letzten
// Läufe zeigen kann, ohne den Sicherungsordner zu lesen — der liegt womöglich
// auf einem Wechseldatenträger, der gerade nicht steckt.
type BackupRun struct {
	ID   uint       `gorm:"primaryKey" json:"id"`
	Kind BackupKind `gorm:"size:20;not null;index" json:"kind"`

	StartedAt  time.Time `gorm:"not null;index" json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`

	// Target ist der Pfad der erzeugten oder geprüften ZIP-Datei.
	Target string `gorm:"size:500;serializer:encrypted" json:"target"`

	FileCount int   `json:"fileCount"`
	Bytes     int64 `json:"bytes"`

	Success bool   `gorm:"not null;default:false;index" json:"success"`
	Message string `gorm:"size:1000" json:"message"`

	// ProgramVersion hält fest, welche Fassung gesichert hat. Ohne sie ließe
	// sich später nicht sagen, welches Schema in der Sicherung liegt.
	ProgramVersion string `gorm:"size:40" json:"programVersion,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// Duration ist die Laufzeit in Sekunden, für die Anzeige.
func (r *BackupRun) Duration() float64 {
	if r.FinishedAt.IsZero() {
		return 0
	}
	return r.FinishedAt.Sub(r.StartedAt).Seconds()
}

// BackupRunRepository persistiert die Sicherungsläufe.
type BackupRunRepository interface {
	Create(ctx context.Context, run *BackupRun) error
	// FindRecent liefert die jüngsten Läufe, neueste zuerst.
	FindRecent(ctx context.Context, limit int) ([]BackupRun, error)
	// LatestSuccessful liefert die jüngste erfolgreiche Sicherung (keinen
	// Prüflauf) oder nil. Daran hängt der Hinweis „letzte Sicherung vor n
	// Tagen".
	LatestSuccessful(ctx context.Context) (*BackupRun, error)
}

// FileCheckIssue ist ein Befund des Belegprüflaufs.
type FileCheckIssue struct {
	// Kind ist "receipt" für eine Belegdatei und "document" für ein
	// Anlagendokument.
	Kind          string `json:"kind"`
	ReceiptNumber string `json:"receiptNumber,omitempty"`
	FileName      string `json:"fileName"`
	Path          string `json:"path"`
	// Reason ist entweder "missing" (Datei fehlt) oder "damaged" (Prüfsumme
	// stimmt nicht mehr).
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// FileCheckResult ist das Ergebnis eines Belegprüflaufs über alle Dateien.
//
// Die Hash-Chain sichert die Buchungen, nicht die Dateien: sie trägt nur den
// Beleg-Hash. Ob die Datei auf der Platte noch die ist, die gebucht wurde,
// beantwortet erst der Vergleich mit ihrer Prüfsumme (GoBD Rz. 110).
type FileCheckResult struct {
	Checked   int              `json:"checked"`
	Intact    int              `json:"intact"`
	Damaged   int              `json:"damaged"`
	Missing   int              `json:"missing"`
	Issues    []FileCheckIssue `json:"issues"`
	IsValid   bool             `json:"isValid"`
	Message   string           `json:"message"`
	CheckedAt string           `json:"checkedAt"`
}

// EnsureLists ersetzt eine nicht belegte Befundliste durch eine leere.
//
// Das Ergebnis geht als JSON an die Oberfläche, und ein nicht belegter Slice
// wird dort zu `null`. Betroffen wäre ausgerechnet der Regelfall: der Lauf ohne
// Befund.
func (r *FileCheckResult) EnsureLists() {
	if r.Issues == nil {
		r.Issues = make([]FileCheckIssue, 0)
	}
}
