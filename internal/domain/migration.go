package domain

import (
	"context"
	"time"
)

// SchemaMigration ist ein Eintrag des Migrationsprotokolls über das
// Datenbankschema.
//
// Bisher lief die Schemaanpassung bei jedem Start und hinterließ keine Spur.
// Für die Verfahrensdokumentation ist genau das die Lücke (GoBD Rz. 34, 145):
// wer wissen will, ob die Datei, die vor einem Jahr gesichert wurde, mit dem
// heutigen Programm noch dasselbe bedeutet, findet ohne Protokoll nichts vor.
type SchemaMigration struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// RunAt ist der Zeitpunkt des Laufs in UTC.
	RunAt time.Time `gorm:"not null;index" json:"runAt"`
	// AppVersion ist die Programmfassung, die migriert hat.
	AppVersion string `gorm:"size:60;not null" json:"appVersion"`
	// FromVersion und ToVersion sind die Schemaversionen vor und nach dem Lauf.
	// FromVersion 0 heißt: eine neue oder eine noch nicht protokollierte Datei.
	FromVersion int `gorm:"not null" json:"fromVersion"`
	ToVersion   int `gorm:"not null" json:"toVersion"`
	// Tables sind die betroffenen Tabellen, durch Komma getrennt.
	Tables string `gorm:"type:text" json:"tables"`
	// Result ist "ok" oder "fehlgeschlagen", Message die Erläuterung.
	Result  string `gorm:"size:20;not null" json:"result"`
	Message string `gorm:"type:text" json:"message"`
	Actor   string `gorm:"size:120" json:"actor,omitempty"`
}

// SchemaMigrationResultOK und SchemaMigrationResultFailed sind die beiden
// Ergebnisse eines Laufs.
const (
	SchemaMigrationResultOK     = "ok"
	SchemaMigrationResultFailed = "fehlgeschlagen"
)

// MigrationKind sagt, auf welchem Weg Daten in diese Installation gekommen
// sind.
type MigrationKind string

const (
	// MigrationKindImport ist die Übernahme einer fremden Datenbankdatei.
	MigrationKindImport MigrationKind = "import"
	// MigrationKindRestore ist die Wiederherstellung aus einer Sicherung.
	MigrationKindRestore MigrationKind = "restore"
	// MigrationKindOpen ist das Öffnen einer vorhandenen Datei.
	MigrationKindOpen MigrationKind = "open"
)

// MigrationCounts sind die Zählungen, mit denen sich eine Datenübernahme
// abstimmen lässt.
//
// Eine Übernahme ohne Zählung ist keine Übernahme, sondern eine Hoffnung: ohne
// Soll-Ist-Abstimmung fällt eine halb kopierte Datei erst auf, wenn die Bilanz
// nicht mehr aufgeht (ARC-05, GoBD Rz. 136).
type MigrationCounts struct {
	JournalEntries int   `json:"journalEntries"`
	JournalLines   int   `json:"journalLines"`
	Receipts       int   `json:"receipts"`
	Contacts       int   `json:"contacts"`
	Accounts       int   `json:"accounts"`
	Invoices       int   `json:"invoices"`
	FixedAssets    int   `json:"fixedAssets"`
	AuditEntries   int   `json:"auditEntries"`
	DebitTotal     Cents `json:"debitTotal"`
	CreditTotal    Cents `json:"creditTotal"`
}

// IsBalanced meldet, ob Soll und Haben über alle übernommenen Buchungen
// übereinstimmen. Tun sie es nicht, fehlt etwas.
func (c MigrationCounts) IsBalanced() bool { return c.DebitTotal == c.CreditTotal }

// MigrationRecord protokolliert eine Datenübernahme.
type MigrationRecord struct {
	ID   uint          `gorm:"primaryKey" json:"id"`
	Kind MigrationKind `gorm:"size:20;not null;index" json:"kind"`
	// Source ist die Herkunft (Dateiname der übernommenen Datei), Target das
	// Ziel (Mandantenkennung).
	Source     string    `gorm:"size:500;serializer:encrypted" json:"source"`
	Target     string    `gorm:"size:120" json:"target"`
	RunAt      time.Time `gorm:"not null;index" json:"runAt"`
	AppVersion string    `gorm:"size:60" json:"appVersion"`
	Actor      string    `gorm:"size:120" json:"actor"`

	// Die Zählungen liegen als Spalten und nicht als JSON-Blob: sie sind das,
	// wonach ein Prüfer fragt, und ein Blob ließe sich weder sortieren noch in
	// den Datenexport nehmen.
	JournalEntries int   `json:"journalEntries"`
	JournalLines   int   `json:"journalLines"`
	Receipts       int   `json:"receipts"`
	Contacts       int   `json:"contacts"`
	Accounts       int   `json:"accounts"`
	Invoices       int   `json:"invoices"`
	FixedAssets    int   `json:"fixedAssets"`
	AuditEntries   int   `json:"auditEntries"`
	DebitTotal     Cents `json:"debitTotal"`
	CreditTotal    Cents `json:"creditTotal"`

	// ChainValid und ChainMessage sind das Ergebnis der Kettenprüfung
	// unmittelbar nach der Übernahme. Eine übernommene Buchhaltung, deren Kette
	// schon beim Ankommen gebrochen ist, muss das sagen — sonst sieht es
	// später aus, als sei sie hier gebrochen worden.
	ChainValid   bool   `json:"chainValid"`
	ChainMessage string `gorm:"type:text" json:"chainMessage"`
}

// Counts bündelt die Zählungen eines Protokolleintrags.
func (r *MigrationRecord) Counts() MigrationCounts {
	return MigrationCounts{
		JournalEntries: r.JournalEntries,
		JournalLines:   r.JournalLines,
		Receipts:       r.Receipts,
		Contacts:       r.Contacts,
		Accounts:       r.Accounts,
		Invoices:       r.Invoices,
		FixedAssets:    r.FixedAssets,
		AuditEntries:   r.AuditEntries,
		DebitTotal:     r.DebitTotal,
		CreditTotal:    r.CreditTotal,
	}
}

// SetCounts überträgt die Zählungen in den Protokolleintrag.
func (r *MigrationRecord) SetCounts(c MigrationCounts) {
	r.JournalEntries = c.JournalEntries
	r.JournalLines = c.JournalLines
	r.Receipts = c.Receipts
	r.Contacts = c.Contacts
	r.Accounts = c.Accounts
	r.Invoices = c.Invoices
	r.FixedAssets = c.FixedAssets
	r.AuditEntries = c.AuditEntries
	r.DebitTotal = c.DebitTotal
	r.CreditTotal = c.CreditTotal
}

// MigrationRepository persistiert beide Protokolle.
type MigrationRepository interface {
	FindSchemaMigrations(ctx context.Context) ([]SchemaMigration, error)
	CreateMigrationRecord(ctx context.Context, rec *MigrationRecord) error
	FindMigrationRecords(ctx context.Context) ([]MigrationRecord, error)
}
