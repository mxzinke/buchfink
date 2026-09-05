package domain

import (
	"fmt"
	"strings"
)

// GiftRecord ist die Aufzeichnung eines Geschenks an einen Geschäftspartner.
//
// § 4 Abs. 7 EStG lässt den Abzug der in § 4 Abs. 5 Satz 1 Nr. 1 bis 4 und 6b
// EStG genannten Aufwendungen nur zu, wenn sie einzeln und getrennt von den
// übrigen Betriebsausgaben aufgezeichnet sind. Für die Bewirtung tut das
// EntertainmentDetail; das Geschenk braucht dasselbe, und zusätzlich den
// Empfänger — ohne ihn ließe sich die Freigrenze je Empfänger und
// Wirtschaftsjahr nicht führen, und ohne die Freigrenze wäre die Angabe
// „abziehbar" eine Behauptung.
//
// Wie die Bewirtungsaufzeichnung hängt sie an der Buchung und nicht am Beleg:
// der Beleg-Hash deckt eine Dateiliste ab, und eine Aufzeichnung, an der der
// Abzug hängt, darf nicht ungedeckt daneben liegen.
type GiftRecord struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	EntryID uint `gorm:"index;not null" json:"entryId"`

	// FiscalYear ist das Wirtschaftsjahr, in dem die Freigrenze läuft. Es wird
	// gespeichert, weil sich der Beginn des Geschäftsjahres ändern kann und die
	// Zuordnung einer alten Buchung davon nicht berührt werden darf.
	FiscalYear int    `gorm:"index;not null" json:"fiscalYear"`
	Date       string `gorm:"size:10;not null" json:"date"`

	// RecipientContactID benennt den Empfänger, wo er ein erfasster
	// Geschäftspartner ist. RecipientName ist der Name — bei einem erfassten
	// Kontakt seiner, sonst der freie Text.
	RecipientContactID *uint  `gorm:"index" json:"recipientContactId,omitempty"`
	RecipientName      string `gorm:"size:255;not null;serializer:encrypted" json:"recipientName"`

	// Occasion ist der Anlass. Er gehört zur Aufzeichnung wie bei der Bewirtung:
	// ein Geschenk ohne betrieblichen Anlass ist keine Betriebsausgabe.
	Occasion string `gorm:"size:500;serializer:encrypted" json:"occasion,omitempty"`

	// NetAmount ist der Nettobetrag, an dem die Freigrenze gemessen wird.
	//
	// Netto, weil die Vorsteuer abziehbar ist, solange das Geschenk abziehbar
	// ist (§ 15 Abs. 1a UStG im Umkehrschluss). Wo sie es nicht ist, zählt der
	// Bruttobetrag — dann ist die Grenze aber ohnehin schon überschritten.
	NetAmount Cents `gorm:"not null" json:"netAmount"`
	// NonDeductible sagt, dass das Geschenk als nicht abziehbar gebucht wurde —
	// weil die Freigrenze mit ihm gerissen ist.
	//
	// Negativ formuliert wie IsPrivate am Kontakt, und aus demselben Grund: GORM
	// lässt bei einem Feld mit `default:true` den Nullwert aus dem INSERT
	// heraus, damit die Datenbankvorgabe greift. Ein `false`, das aus einem
	// Struct kommt, käme deshalb als `true` in der Spalte an — das abziehbar
	// gebuchte Geschenk und das nicht abziehbare wären nicht mehr zu
	// unterscheiden. Der Regelfall muss der Nullwert sein.
	NonDeductible bool `gorm:"not null;default:false" json:"nonDeductible,omitempty"`
	// Account ist das Konto, auf das gebucht wurde.
	Account string `gorm:"size:10;index" json:"account,omitempty"`
}

// Ob ein zunächst abziehbar gebuchtes Geschenk inzwischen umgebucht wurde, steht
// nicht an dieser Aufzeichnung. Sie ist Teil der Buchung und von der Hashkette
// gedeckt; ein Feld, das nachträglich gesetzt würde, bräche sie. Die Antwort
// steht dort, wo sie hingehört: die Umbuchung ist eine Generalumkehr der
// ursprünglichen Buchung, und ob es eine gibt, weiß das Journal.

// Deductible meldet, ob das Geschenk abziehbar gebucht wurde.
func (g *GiftRecord) Deductible() bool { return !g.NonDeductible }

// RecipientKey ist der Schlüssel, unter dem die Freigrenze geführt wird.
//
// Der erfasste Kontakt geht vor: zwei Schreibweisen desselben Namens sind ein
// Empfänger, und wer die Freigrenze über den Namen führte, könnte sie mit einem
// Leerzeichen umgehen.
func (g *GiftRecord) RecipientKey() string {
	if g.RecipientContactID != nil && *g.RecipientContactID != 0 {
		return fmt.Sprintf("contact:%d", *g.RecipientContactID)
	}
	return "name:" + strings.ToLower(strings.Join(strings.Fields(g.RecipientName), " "))
}

// Validate prüft die Aufzeichnung.
func (g *GiftRecord) Validate() error {
	if strings.TrimSpace(g.RecipientName) == "" {
		return fmt.Errorf(
			"zu einem Geschenk gehört der Empfänger (§ 4 Abs. 7 EStG). Ohne ihn ist der Abzug " +
				"verloren, und die Freigrenze je Empfänger und Wirtschaftsjahr lässt sich nicht führen")
	}
	if len(g.Date) != 10 {
		return fmt.Errorf("das Datum des Geschenks fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if g.NetAmount <= 0 {
		return fmt.Errorf("der Betrag des Geschenks muss größer als null sein")
	}
	return nil
}
