package domain

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Das Mahnwesen (QUE-05).
//
// Eine überfällige Forderung ist kein Buchungsfall, sondern ein Vorgang: der
// Schuldner kommt spätestens dreißig Tage nach Fälligkeit und Zugang der
// Rechnung in Verzug (§ 286 Abs. 3 BGB), von da an laufen Verzugszinsen
// (§ 288 BGB), und gegenüber einem Unternehmer kommt die Pauschale des § 288
// Abs. 5 BGB dazu. Buchfink rechnet das aus, schreibt das Schreiben und legt es
// ab — gebucht wird davon nichts: Zinsen und Gebühren sind erst mit der Zahlung
// Ertrag, und eine Forderung, die nie gezahlt wird, wäre sonst zweimal
// abzuschreiben.

// DunningLevel ist eine Mahnstufe.
//
// Die Stufen sind eine Vereinbarung des Unternehmens mit sich selbst und keine
// Vorschrift: das Gesetz kennt keine „erste Mahnung". Deshalb sind sie
// einstellbar, und deshalb steht die Gebühr als Vorschlag darin — sie ist
// ersatzfähig nur, soweit sie tatsächlich entstandener Verzugsschaden ist.
type DunningLevel struct {
	// Level ist die Stufe, beginnend bei 1.
	Level int `json:"level"`
	// Label ist die Bezeichnung im Schreiben („Zahlungserinnerung").
	Label string `json:"label"`
	// DaysAfterDue ist der Abstand zur Fälligkeit, ab dem diese Stufe
	// vorgeschlagen wird.
	DaysAfterDue int `json:"daysAfterDue"`
	// Fee ist die vorgeschlagene Mahngebühr.
	Fee Cents `json:"fee"`
}

// DefaultDunningLevels ist die Voreinstellung der Mahnstufen.
//
// Zahlungserinnerung nach einer Woche ohne Gebühr — sie ist eine Erinnerung und
// kein Mahnschreiben; die beiden Mahnungen danach mit einer geringen Gebühr, wie
// sie in der Rechtsprechung als Verzugsschaden anerkannt wird. Wer höhere
// Gebühren ansetzt, trägt die Begründung selbst.
func DefaultDunningLevels() []DunningLevel {
	return []DunningLevel{
		{Level: 1, Label: "Zahlungserinnerung", DaysAfterDue: 7, Fee: 0},
		{Level: 2, Label: "1. Mahnung", DaysAfterDue: 21, Fee: 500},
		{Level: 3, Label: "2. Mahnung", DaysAfterDue: 35, Fee: 1000},
	}
}

// NormalizeDunningLevels bringt eine eingestellte Stufenfolge in Form.
//
// Leer heißt: die Voreinstellung. Sonst wird nach dem Abstand zur Fälligkeit
// sortiert und durchnummeriert — die Stufe folgt aus der Reihenfolge und nicht
// aus einer Zahl, die jemand eingetragen hat: zwei Stufen mit der Nummer 2
// wären sonst möglich, und der Mahnlauf wüsste nicht, welche als Nächste kommt.
func NormalizeDunningLevels(levels []DunningLevel) []DunningLevel {
	cleaned := make([]DunningLevel, 0, len(levels))
	for _, l := range levels {
		if l.Label == "" || l.DaysAfterDue < 0 {
			continue
		}
		if l.Fee < 0 {
			l.Fee = 0
		}
		cleaned = append(cleaned, l)
	}
	if len(cleaned) == 0 {
		return DefaultDunningLevels()
	}
	sort.SliceStable(cleaned, func(i, j int) bool {
		return cleaned[i].DaysAfterDue < cleaned[j].DaysAfterDue
	})
	for i := range cleaned {
		cleaned[i].Level = i + 1
	}
	return cleaned
}

// DunningNotice ist ein erzeugtes Mahnschreiben.
//
// Es steht als eigener Datensatz und nicht als Vermerk am offenen Posten: ein
// Schreiben kann mehrere Posten desselben Kunden umfassen (und tut es in der
// Regel), und welche Stufe ein Posten erreicht hat, ergibt sich aus den
// Schreiben, in denen er stand.
type DunningNotice struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	FiscalYear  int    `gorm:"index;not null" json:"fiscalYear"`
	ContactID   uint   `gorm:"index;not null" json:"contactId"`
	ContactName string `gorm:"size:255;serializer:encrypted" json:"contactName"`
	// IsConsumer hält fest, ob der Empfänger beim Erzeugen als Verbraucher
	// geführt wurde. Er entscheidet über den Zinssatz (§ 288 Abs. 1 und 2 BGB)
	// und über die Pauschale (§ 288 Abs. 5 BGB); ändern sich die Stammdaten
	// später, bleibt das Schreiben trotzdem erklärbar.
	IsConsumer bool `gorm:"not null;default:false" json:"isConsumer"`

	Level      int    `gorm:"not null;index" json:"level"`
	LevelLabel string `gorm:"size:60;not null" json:"levelLabel"`
	// NoticeDate ist das Datum des Schreibens, DueDate die darin gesetzte
	// Zahlungsfrist.
	NoticeDate string `gorm:"size:10;not null;index" json:"noticeDate"`
	DueDate    string `gorm:"size:10" json:"dueDate"`

	PrincipalAmount Cents `gorm:"not null" json:"principalAmount"`
	InterestAmount  Cents `gorm:"not null" json:"interestAmount"`
	FeeAmount       Cents `gorm:"not null" json:"feeAmount"`
	// LumpSumAmount ist die Pauschale des § 288 Abs. 5 BGB (40 €). Sie fällt nur
	// an, wenn der Schuldner kein Verbraucher ist, und nur einmal je Forderung.
	LumpSumAmount Cents `gorm:"not null" json:"lumpSumAmount"`
	TotalAmount   Cents `gorm:"not null" json:"totalAmount"`

	// Das abgelegte Schreiben im Belegspeicher (Zweig dokumente/mahnungen).
	DocumentName   string `gorm:"size:255" json:"documentName,omitempty"`
	DocumentPath   string `gorm:"size:500" json:"documentPath,omitempty"`
	DocumentSHA256 string `gorm:"size:64" json:"documentSha256,omitempty"`
	// DocumentNote nennt den Grund, aus dem kein PDF entstanden ist. Das
	// Schreiben bleibt trotzdem verzeichnet: die Forderung ist gemahnt, auch
	// wenn der Setzer nicht lief.
	DocumentNote string `gorm:"size:255" json:"documentNote,omitempty"`

	Items     []DunningNoticeItem `gorm:"foreignKey:DunningNoticeID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt time.Time           `json:"createdAt"`
}

// DunningNoticeItem ist ein gemahnter Posten mit seiner Stufe und seinen Zinsen.
type DunningNoticeItem struct {
	ID              uint `gorm:"primaryKey" json:"id"`
	DunningNoticeID uint `gorm:"index;not null" json:"dunningNoticeId"`
	// OpenItemEntryID ist die Buchung, die den offenen Posten trägt.
	OpenItemEntryID uint   `gorm:"index;not null" json:"openItemEntryId"`
	DocumentNumber  string `gorm:"size:60" json:"documentNumber"`
	DocumentDate    string `gorm:"size:10" json:"documentDate"`
	DueDate         string `gorm:"size:10" json:"dueDate"`
	OpenAmount      Cents  `gorm:"not null" json:"openAmount"`
	// DefaultFrom ist der Tag, ab dem für diesen Posten Verzugszinsen laufen,
	// InterestDays die gerechneten Tage bis zum Datum des Schreibens.
	DefaultFrom    string `gorm:"size:10" json:"defaultFrom"`
	InterestDays   int    `json:"interestDays"`
	InterestAmount Cents  `gorm:"not null" json:"interestAmount"`
	// LumpSumAmount ist die Pauschale des § 288 Abs. 5 BGB, soweit sie auf
	// diesen Posten entfällt.
	//
	// Am Posten und nicht nur in der Summe des Schreibens: die Pauschale fällt
	// je Forderung einmal an, und die Frage, ob sie für diese Forderung schon
	// angesetzt wurde, lässt sich nur hier beantworten. Stünde sie allein am
	// Schreiben, wüsste der nächste Mahnlauf bei einem Schreiben über drei
	// Posten nicht, welcher der drei sie getragen hat.
	LumpSumAmount Cents `gorm:"not null;default:0" json:"lumpSumAmount"`
	// Level ist die Stufe, die dieser Posten mit diesem Schreiben erreicht.
	Level int `gorm:"not null" json:"level"`
}

// EnsureLists ersetzt eine nicht belegte Postenliste durch eine leere.
func (n *DunningNotice) EnsureLists() {
	if n.Items == nil {
		n.Items = make([]DunningNoticeItem, 0)
	}
}

// DunningRepository persistiert die Mahnschreiben.
type DunningRepository interface {
	Create(ctx context.Context, notice *DunningNotice) error
	// FindByContact liefert die Schreiben eines Kunden, das jüngste zuerst.
	// contactID 0 heißt: alle.
	FindByContact(ctx context.Context, contactID uint) ([]DunningNotice, error)
	// LevelByOpenItem liefert je offenem Posten die höchste bisher erreichte
	// Mahnstufe. Der Mahnlauf braucht sie, um die nächste vorzuschlagen — ohne
	// sie stünde jeder Posten ewig auf der ersten Stufe.
	LevelByOpenItem(ctx context.Context) (map[uint]int, error)
	// LastNoticeByOpenItem liefert je Posten das Datum des jüngsten Schreibens.
	LastNoticeByOpenItem(ctx context.Context) (map[uint]string, error)
	// LumpSumChargedByOpenItem meldet je Posten, ob die Pauschale des § 288
	// Abs. 5 BGB für ihn schon in einem Schreiben stand. Sie fällt je Forderung
	// einmal an; ohne diese Auskunft stünde sie in jedem Schreiben erneut oder
	// — je nach Bedingung — in keinem.
	LumpSumChargedByOpenItem(ctx context.Context) (map[uint]bool, error)
}

// BaseRate ist der Basiszinssatz nach § 247 BGB ab einem Stichtag.
//
// Als gepflegte Tabelle und nicht als Konstante: die Bundesbank setzt ihn zum
// 1. Januar und zum 1. Juli neu fest und gibt ihn im Bundesanzeiger bekannt.
// Ein Programm, das nur den Wert seiner Auslieferung kennt, rechnet ein halbes
// Jahr später falsch — und eine Zinsrechnung, die falsch ist, ist eine
// unberechtigte Forderung.
type BaseRate struct {
	// ValidFrom ist der erste Tag, für den dieser Satz gilt.
	ValidFrom string `gorm:"primaryKey;size:10" json:"validFrom"`
	// BasisPoints ist der Satz in Hundertsteln eines Prozentpunktes:
	// 127 sind 1,27 %. Ganzzahlig, weil die Bekanntgabe zwei Nachkommastellen
	// hat und ein Gleitkommawert die Zinsrechnung um Cents verschieben würde.
	BasisPoints int    `gorm:"not null" json:"basisPoints"`
	Source      string `gorm:"size:255" json:"source,omitempty"`
	// Provisional markiert einen fortgeschriebenen Wert, der noch nicht aus der
	// Bekanntgabe stammt. Die Einstellungsseite zeigt ihn als „zu prüfen".
	Provisional bool      `gorm:"not null;default:false" json:"provisional"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Percent gibt den Satz als Prozentzeichenfolge aus („1,27 %").
func (r BaseRate) Percent() string {
	sign := ""
	points := r.BasisPoints
	if points < 0 {
		sign = "-"
		points = -points
	}
	return fmt.Sprintf("%s%d,%02d %%", sign, points/100, points%100)
}

// BaseRateRepository persistiert die Basiszinssätze.
type BaseRateRepository interface {
	// FindAll liefert die Sätze, der älteste zuerst.
	FindAll(ctx context.Context) ([]BaseRate, error)
	Save(ctx context.Context, rate *BaseRate) error
}
