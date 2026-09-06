package domain

import (
	"context"
	"time"
)

// AuditAction defines the type of event logged in the GoBD audit trail.
type AuditAction string

const (
	AuditActionCreate         AuditAction = "CREATE"
	AuditActionUpdate         AuditAction = "UPDATE"
	AuditActionStorno         AuditAction = "STORNO"
	AuditActionImport         AuditAction = "IMPORT"
	AuditActionIntegrityCheck AuditAction = "INTEGRITY_CHECK"
	AuditActionExport         AuditAction = "EXPORT"
)

// AuditEntityReadOnly ist die Entität der Prüfermodus-Einträge. Sie steht als
// Konstante, weil der Zugriffsfilter sie kennen muss (siehe AuditFilter.Access).
const AuditEntityReadOnly = "READ_ONLY"

// AuditLogEntry represents an immutable entry in the GoBD compliance audit trail.
type AuditLogEntry struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Timestamp steht in UTC. Eine Ortszeit ohne Zone ist zweimal im Jahr
	// mehrdeutig — in der Nacht der Zeitumstellung gibt es 02:30 Uhr zweimal —,
	// und ein Protokoll, dessen Reihenfolge sich nicht aus den Zeitpunkten
	// ergibt, beantwortet die Frage nicht, für die es geführt wird.
	Timestamp  time.Time   `gorm:"not null;index" json:"timestamp"`
	Action     AuditAction `gorm:"size:30;not null;index" json:"action"`
	EntityType string      `gorm:"size:50;not null;index" json:"entityType"` // "BOOKING", "ACCOUNT", "BANK_TX", "INVOICE", "SETTINGS"
	EntityID   string      `gorm:"size:50;not null" json:"entityId"`
	Details    string      `gorm:"type:text;not null" json:"details"`

	// Before und After sind die geänderten Felder als JSON-Objekt, und zwar nur
	// die geänderten.
	//
	// GoBD Rz. 34 verlangt, dass eine Änderung in ihrem ursprünglichen Inhalt
	// erkennbar bleibt. Ein Protokollsatz „Kontakt 12 geändert" leistet das
	// nicht: er sagt, dass etwas geschah, aber nicht was. Der vollständige
	// Datensatz wäre die andere Übertreibung — er verbärge die eine geänderte
	// Zeile zwischen dreißig unveränderten.
	//
	// Verschlüsselt, weil Stammdaten personenbezogene Daten tragen: Name,
	// Anschrift, Bankverbindung eines Geschäftspartners stünden sonst im
	// Protokoll im Klartext, während sie in ihrer eigenen Tabelle verschlüsselt
	// liegen.
	Before string `gorm:"type:text;serializer:encrypted" json:"before,omitempty"`
	After  string `gorm:"type:text;serializer:encrypted" json:"after,omitempty"`

	// Actor ist die Bearbeiterkennung (internal/actor), AppVersion die Fassung
	// des Programms, die gehandelt hat. Beide beantworten die Fragen „wer" und
	// „womit", die die Nachvollziehbarkeit neben „was" und „wann" stellt.
	Actor      string `gorm:"size:120;index" json:"actor,omitempty"`
	AppVersion string `gorm:"size:60" json:"appVersion,omitempty"`

	PreviousHash string `gorm:"size:64" json:"previousHash,omitempty"`
	EntryHash    string `gorm:"size:64" json:"entryHash,omitempty"`

	// TODO: Add cryptographic signature per audit entry (e.g. Ed25519) if required for extended certifications
}

// AuditFilter schränkt die Protokollabfrage ein. Leere Felder heißen: alles.
//
// Das Protokoll wächst mit jeder Buchung, jedem Export und jeder Sicherung. Wer
// nachsehen will, wer einen bestimmten Kontakt geändert hat, kann das nicht,
// indem er zehntausend Zeilen in die Oberfläche lädt und dort sucht.
type AuditFilter struct {
	Action     AuditAction `json:"action,omitempty"`
	EntityType string      `json:"entityType,omitempty"`
	EntityID   string      `json:"entityId,omitempty"`
	Actor      string      `json:"actor,omitempty"`
	// Access schränkt auf die Lesezugriffe auf personenbezogene Daten ein
	// (QUE-02 K2): Datenüberlassung, Prüferpaket, Herausgabe einzelner Dateien
	// — sie stehen als EXPORT — und das Ein- und Ausschalten des Prüfermodus,
	// das als Änderung an der Entität READ_ONLY protokolliert wird. Beides in
	// einer Abfrage, weil „Zugriffe" eine Kategorie ist und keine Aktion: über
	// Action allein ließe sie sich nicht abbilden, und zwei Abfragen in der
	// Oberfläche zusammenzuführen hieße, die Reihenfolge des Protokolls dort
	// neu zu erfinden.
	Access bool `json:"access,omitempty"`
	// From und To sind Tagesgrenzen im Format YYYY-MM-DD, beide einschließlich.
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// IsEmpty meldet, ob der Filter nichts einschränkt.
func (f AuditFilter) IsEmpty() bool {
	return f.Action == "" && f.EntityType == "" && f.EntityID == "" &&
		f.Actor == "" && f.From == "" && f.To == "" && !f.Access
}

// AuditEntryHashFunc verkettet einen Protokolleintrag mit seinem Vorgänger.
type AuditEntryHashFunc func(e *AuditLogEntry, prevHash string) string

// AuditRepository defines persistence operations for audit logs.
type AuditRepository interface {
	// Log ist die Kurzform ohne Vorher/Nachher: für Vorgänge, die nichts
	// ändern — ein Export, eine Integritätsprüfung, ein Sicherungslauf.
	Log(ctx context.Context, action AuditAction, entityType, entityID, details string) error
	// LogChange ist die Langform. before und after sind die beiden Stände des
	// geänderten Objekts; gespeichert wird nur, was sich zwischen ihnen
	// unterscheidet (siehe ChangedFields). Ein nil-before heißt: neu angelegt.
	LogChange(ctx context.Context, action AuditAction, entityType, entityID, details string, before, after any) error
	FindAll(ctx context.Context, limit int) ([]AuditLogEntry, error)
	// FindFiltered liefert dieselbe Liste, eingeschränkt durch den Filter.
	FindFiltered(ctx context.Context, limit int, filter AuditFilter) ([]AuditLogEntry, error)
	// FindAllAscending liefert das Protokoll in Schreibreihenfolge. Die
	// Kettenprüfung braucht genau diese Richtung — sie läuft vom ersten Eintrag
	// vorwärts —, während die Anzeige die umgekehrte will.
	FindAllAscending(ctx context.Context) ([]AuditLogEntry, error)
	Count(ctx context.Context) (int64, error)
}
