package domain

import (
	"context"
	"time"
)

// Konten des Lohnjournals (SKR04-Katalog internal/accounting/skr04_2026.json).
const (
	AccountGehaelter                  = "6020" // Gehälter
	AccountGFGehaelterGesellschafter  = "6024" // Geschäftsführergehälter der GmbH-Gesellschafter
	AccountGFGehaelter                = "6027" // Geschäftsführergehälter
	AccountLoehneMinijobs             = "6035" // Löhne für Minijobs
	AccountPauschsteuerMinijobs       = "6036" // Pauschale Steuer für Minijobber
	AccountPauschsteuerGesellschafter = "6037" // Pauschale Steuer für Gesellschafter-Geschäftsführer
	AccountPauschsteuerArbeitnehmer   = "6039" // Pauschale Steuer für Arbeitnehmer
	AccountGesetzlicheSozialeAufw     = "6110" // Gesetzliche soziale Aufwendungen
	AccountSozialeAbgabenMinijobs     = "6171" // Soziale Abgaben für Minijobber
	AccountForderungenPersonal        = "1340" // Forderungen gegen Personal aus Lohn- und Gehaltsabrechnung
	AccountVerbLohnGehalt             = "3720" // Verbindlichkeiten aus Lohn und Gehalt
	AccountVerbEinbehaltungen         = "3725" // Verbindlichkeiten für Einbehaltungen von Arbeitnehmern
	AccountVerbLohnKirchensteuer      = "3730" // Verbindlichkeiten aus Lohn- und Kirchensteuer
	AccountVerbSozialeSicherheit      = "3740" // Verbindlichkeiten im Rahmen der sozialen Sicherheit
	AccountBeitragsschuldSV           = "3759" // Voraussichtliche Beitragsschuld gegenüber den Sozialversicherungsträgern
	AccountLohnGehaltVerrechnung      = "3790" // Lohn- und Gehaltsverrechnungskonto
)

// PayrollRun ist ein gebuchtes Lohnjournal eines Abrechnungsmonats.
type PayrollRun struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Month     string `gorm:"size:7;not null;index" json:"month"` // JJJJ-MM
	EntryID   uint   `gorm:"not null;index" json:"entryId"`
	ReceiptID uint   `gorm:"not null" json:"receiptId"`
	Gross     Cents  `gorm:"not null" json:"gross"`
	Net       Cents  `gorm:"not null" json:"net"`
	// WageTaxes ist, was an das Finanzamt geht: Lohn- und Kirchensteuer,
	// Solidaritätszuschlag und pauschale Lohnsteuer außer der der Minijobs.
	WageTaxes Cents `gorm:"not null" json:"wageTaxes"`
	// SocialSecurity ist, was an die Einzugsstellen geht, samt Umlagen und
	// den Abgaben an die Minijob-Zentrale.
	SocialSecurity Cents     `gorm:"not null" json:"socialSecurity"`
	CreatedAt      time.Time `json:"createdAt"`
}

// PayrollRepository speichert die gebuchten Lohnjournale.
type PayrollRepository interface {
	Save(ctx context.Context, run *PayrollRun) error
	FindByMonth(ctx context.Context, month string) ([]PayrollRun, error)
	FindAll(ctx context.Context) ([]PayrollRun, error)
}
