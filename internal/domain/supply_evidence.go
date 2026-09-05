package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Der Belegnachweis der innergemeinschaftlichen Lieferung.
//
// Die Belege hängen an der Rechnung und nicht am Beleg der Rechnung: der
// Beleg-Hash deckt eine feste Dateiliste ab und wird beim Buchen versiegelt,
// während ein Frachtbrief regelmäßig erst Tage später eintrifft. Ein Nachweis,
// der nur vor dem Buchen abgelegt werden kann, ist keiner — es ist der Normalfall,
// dass die Rechnung vor dem Frachtbrief da ist.

// SupplyEvidence ist ein abgelegter Nachweisbeleg zu einer Rechnung.
type SupplyEvidence struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	InvoiceID uint `gorm:"index;not null" json:"invoiceId"`

	// Kind ist die Belegart (siehe accounting.EvidenceKinds).
	Kind string `gorm:"size:40;not null;index" json:"kind"`
	// Issuer ist der Aussteller. Er entscheidet über die Unabhängigkeit: zwei
	// Belege desselben Spediteurs sind ein Beleg mit zwei Blättern.
	Issuer string `gorm:"size:255;not null;serializer:encrypted" json:"issuer"`
	// Independent sagt, dass der Aussteller weder der Lieferer noch der Erwerber
	// ist — die Bedingung, die Art. 45a MwStVO an die beiden Belege knüpft.
	//
	// Die Datenbankvorgabe ist bewusst `false` und nicht `true`: GORM lässt bei
	// einem Feld mit `default:true` den Nullwert aus dem INSERT heraus, und ein
	// abhängiger Aussteller käme dann als unabhängiger in der Spalte an. Die
	// Vermutung des § 17a UStDV hinge damit an einem Beleg, der sie nicht
	// trägt. Die Maske schlägt „unabhängig" vor; gespeichert wird, was dort
	// steht.
	Independent bool `gorm:"not null;default:false" json:"independent"`

	// Date ist das Datum des Belegs.
	Date string `gorm:"size:10;not null" json:"date"`

	// ReceiptID verweist auf den Beleg, unter dem die Datei abgelegt ist. Die
	// Datei selbst liegt im Belegspeicher wie jede andere — ein zweiter
	// Ablageweg wäre ein zweiter Ort, an dem die Aufbewahrungsfrist einzuhalten
	// wäre.
	ReceiptID *uint `gorm:"index" json:"receiptId,omitempty"`

	Note      string    `gorm:"size:500;serializer:encrypted" json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Validate prüft, was ein Nachweisbeleg tragen muss.
func (e *SupplyEvidence) Validate() error {
	if e.InvoiceID == 0 {
		return fmt.Errorf("der Nachweis gehört zu einer Rechnung")
	}
	if strings.TrimSpace(e.Kind) == "" {
		return fmt.Errorf("die Belegart fehlt")
	}
	if strings.TrimSpace(e.Issuer) == "" {
		return fmt.Errorf(
			"der Aussteller fehlt. Er ist keine Formalie: Art. 45a MwStVO verlangt Belege von zwei " +
				"voneinander unabhängigen Parteien, und ohne den Aussteller lässt sich das nicht zählen")
	}
	if len(e.Date) != 10 {
		return fmt.Errorf("das Belegdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	return nil
}

// SupplyEvidenceRepository persistiert die Nachweisbelege.
type SupplyEvidenceRepository interface {
	FindByInvoice(ctx context.Context, invoiceID uint) ([]SupplyEvidence, error)
	// FindByInvoices liefert die Nachweise mehrerer Rechnungen in einem Zug. Der
	// Bericht über die Lieferungen eines Jahres braucht sie so — je Rechnung
	// einzeln zu fragen liest die Tabelle n-mal.
	FindByInvoices(ctx context.Context, invoiceIDs []uint) (map[uint][]SupplyEvidence, error)
	Save(ctx context.Context, evidence *SupplyEvidence) error
	Delete(ctx context.Context, id uint) error
}
