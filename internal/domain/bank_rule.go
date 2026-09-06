package domain

import (
	"context"
	"time"
)

// BankRule ist eine gelernte Zuordnung: Muster → Buchungsgruppe.
//
// Miete, Kontoführungsentgelt, Gehalt — dieselben Umsätze kommen jeden Monat
// wieder und haben keinen offenen Posten, gegen den sie liefen. Buchfink merkt
// sich, wogegen ein solcher Umsatz zuletzt gebucht wurde, und schlägt es beim
// nächsten Mal vor.
//
// Vorschlagen, nicht buchen: die Regel entsteht aus einer bestätigten
// Zuordnung, und die nächste Zuordnung bestätigt der Anwender wieder. Eine
// Regel, die selbst bucht, würde aus einem einmaligen Griff eine Gewohnheit des
// Programms machen, die niemand mehr überprüft — und sie stünde im Widerspruch
// zum Versprechen, dass Buchfink nie ohne Bestätigung bucht. Sie ist deshalb in
// den Einstellungen einsehbar und löschbar.
type BankRule struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Pattern ist das normalisierte Muster aus Zahlungspartner und
	// Verwendungszweck (siehe accounting.BankRulePattern). Es ist der
	// Schlüssel: zwei Umsätze mit demselben Muster gelten als derselbe Vorgang.
	Pattern string `gorm:"size:200;uniqueIndex;not null" json:"pattern"`
	// Label ist das Muster in lesbarer Form, wie es in den Einstellungen steht.
	Label string `gorm:"size:255" json:"label"`
	// CounterAccount ist das zuletzt bestätigte Gegenkonto, PostingGroup die
	// Buchungsgruppe dazu, soweit sie sich zuordnen ließ. Die Gruppe ist das,
	// was die Oberfläche anbietet; das Konto ist das, was gebucht wird.
	CounterAccount string `gorm:"size:10;not null" json:"counterAccount"`
	PostingGroup   string `gorm:"size:60" json:"postingGroup,omitempty"`
	// MoneyIn sagt, ob das Muster aus einem Geldeingang gelernt wurde.
	//
	// Ein eigenes Kennzeichen und nicht die Belegrichtung: hier geht es um die
	// Richtung des Geldes auf dem Konto und nicht um die eines Dokuments.
	// Dasselbe Gegenkonto kann in beide Richtungen laufen — die Rückerstattung
	// einer Versicherung trifft dasselbe Konto wie ihr Beitrag —, und ein
	// Vorschlag, der die Richtung vertauscht, ist keiner.
	MoneyIn bool `gorm:"not null;default:false" json:"moneyIn"`
	// Hits zählt die Bestätigungen. Sie ist kein Rang, sondern eine Auskunft:
	// eine Regel, die einmal entstand und nie wieder passte, sieht man daran.
	Hits       int       `gorm:"not null;default:1" json:"hits"`
	LastUsedAt string    `gorm:"size:10" json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// BankRuleRepository persistiert die gelernten Zuordnungen.
type BankRuleRepository interface {
	FindAll(ctx context.Context) ([]BankRule, error)
	// FindByPattern liefert die Regel zu einem Muster oder nil.
	FindByPattern(ctx context.Context, pattern string) (*BankRule, error)
	// Save legt die Regel an oder zählt sie hoch.
	Save(ctx context.Context, rule *BankRule) error
	Delete(ctx context.Context, id uint) error
}
