package domain

import (
	"context"
	"fmt"
	"time"
)

// ProvisionKind ist der Rückstellungsgrund nach § 249 HGB.
//
// Die Art ist nicht bloß eine Beschriftung: § 249 Abs. 1 HGB zählt abschließend
// auf, wofür Rückstellungen gebildet werden *dürfen*, und Abs. 2 verbietet
// alles Übrige. Aus der Art folgen außerdem das Bilanz- und das Aufwandskonto,
// und bei der unterlassenen Instandhaltung folgt aus ihr eine Frist: nachgeholt
// werden muss in den ersten drei Monaten des folgenden Geschäftsjahres.
type ProvisionKind string

const (
	// ProvisionUncertainLiability ist die Rückstellung für ungewisse
	// Verbindlichkeiten (§ 249 Abs. 1 Satz 1 Alt. 1 HGB) — der Regelfall.
	ProvisionUncertainLiability ProvisionKind = "uncertain_liability"
	// ProvisionPendingLoss ist die Drohverlustrückstellung
	// (§ 249 Abs. 1 Satz 1 Alt. 2 HGB). Steuerlich ist sie nach
	// § 5 Abs. 4a EStG nicht anzusetzen — handelsrechtlich ist sie Pflicht.
	ProvisionPendingLoss ProvisionKind = "pending_loss"
	// ProvisionDeferredMaintenance ist die unterlassene Instandhaltung, die in
	// den ersten drei Monaten des Folgejahres nachgeholt wird
	// (§ 249 Abs. 1 Satz 2 Nr. 1 HGB).
	ProvisionDeferredMaintenance ProvisionKind = "deferred_maintenance"
	// ProvisionWarranty ist die Gewährleistung ohne rechtliche Verpflichtung
	// (§ 249 Abs. 1 Satz 2 Nr. 2 HGB) — Kulanz.
	ProvisionWarranty ProvisionKind = "warranty_without_obligation"
	// ProvisionTaxIncome ist die Steuerrückstellung für Körperschaftsteuer und
	// Solidaritätszuschlag.
	ProvisionTaxIncome ProvisionKind = "tax_income"
	// ProvisionTaxTrade ist die Gewerbesteuerrückstellung. Eigenes Konto, weil
	// § 4 Abs. 5b EStG die Gewerbesteuer vom Betriebsausgabenabzug ausnimmt und
	// sie deshalb in der Überleitung getrennt zu zeigen ist.
	ProvisionTaxTrade ProvisionKind = "tax_trade"
	// ProvisionClosingCosts sind die Kosten des Jahresabschlusses und seiner
	// Prüfung.
	ProvisionClosingCosts ProvisionKind = "closing_costs"
	// ProvisionRetentionCosts sind die Kosten der Aufbewahrung der
	// Geschäftsunterlagen (§ 257 HGB, § 147 AO).
	ProvisionRetentionCosts ProvisionKind = "retention_costs"
	// ProvisionPersonnel sind Urlaub, Tantiemen, Überstunden und Ähnliches.
	ProvisionPersonnel ProvisionKind = "personnel"
	// ProvisionPension ist die Pensionsrückstellung. Buchfink erfasst sie, aber
	// bewertet sie nicht: die Bewertung nach § 253 Abs. 1 Satz 2 und Abs. 2
	// HGB verlangt versicherungsmathematische Annahmen, die kein
	// Buchführungsprogramm setzen kann.
	ProvisionPension ProvisionKind = "pension"
)

// AllProvisionKinds liefert die Arten in der Reihenfolge, in der die Maske sie
// anbietet — die häufigen zuerst.
func AllProvisionKinds() []ProvisionKind {
	return []ProvisionKind{
		ProvisionUncertainLiability, ProvisionClosingCosts, ProvisionPersonnel,
		ProvisionWarranty, ProvisionDeferredMaintenance, ProvisionPendingLoss,
		ProvisionRetentionCosts, ProvisionTaxIncome, ProvisionTaxTrade, ProvisionPension,
	}
}

// Label benennt die Art in Klartext, ohne Paragraphen: die Auswahl ist für
// jemanden gemacht, der weiß, wofür er Geld zurücklegt, und nicht für jemanden,
// der § 249 HGB auswendig kann.
func (k ProvisionKind) Label() string {
	switch k {
	case ProvisionUncertainLiability:
		return "Offene Verpflichtung, Höhe noch unklar"
	case ProvisionPendingLoss:
		return "Drohender Verlust aus einem laufenden Geschäft"
	case ProvisionDeferredMaintenance:
		return "Aufgeschobene Instandhaltung (Nachholung bis 31. März)"
	case ProvisionWarranty:
		return "Gewährleistung aus Kulanz"
	case ProvisionTaxIncome:
		return "Körperschaftsteuer und Solidaritätszuschlag"
	case ProvisionTaxTrade:
		return "Gewerbesteuer"
	case ProvisionClosingCosts:
		return "Jahresabschluss- und Prüfungskosten"
	case ProvisionRetentionCosts:
		return "Aufbewahrung der Geschäftsunterlagen"
	case ProvisionPersonnel:
		return "Personalkosten (Urlaub, Tantiemen, Überstunden)"
	case ProvisionPension:
		return "Pensionszusage (nur Erfassung, keine Bewertung)"
	}
	return string(k)
}

// Valid meldet, ob die Art eine der bekannten ist.
func (k ProvisionKind) Valid() bool {
	for _, known := range AllProvisionKinds() {
		if known == k {
			return true
		}
	}
	return false
}

// IsTax meldet, ob die Art eine Steuerrückstellung ist. Sie entsteht über den
// Abschlussbaustein und nicht von Hand.
func (k ProvisionKind) IsTax() bool {
	return k == ProvisionTaxIncome || k == ProvisionTaxTrade
}

// ProvisionMovementKind ist die Bewegungsart im Rückstellungsspiegel.
//
// Die fünf Arten sind genau die Spalten, die § 285 Nr. 12 HGB und die Praxis im
// Anhang erwarten. Sie zu einer einzigen Wertänderung zusammenzufassen wäre
// dasselbe wie beim Anlagenspiegel: aus dem Endbestand ließe sich die
// Entwicklung nicht mehr ablesen.
type ProvisionMovementKind string

const (
	// ProvisionFormation ist die Bildung.
	ProvisionFormation ProvisionMovementKind = "formation"
	// ProvisionIncrease ist die Zuführung zu einer bestehenden Rückstellung.
	ProvisionIncrease ProvisionMovementKind = "increase"
	// ProvisionConsumption ist der Verbrauch: die Verpflichtung wird erfüllt,
	// die Rechnung läuft gegen die Rückstellung und nicht gegen den Aufwand.
	ProvisionConsumption ProvisionMovementKind = "consumption"
	// ProvisionRelease ist die Auflösung: der Grund ist weggefallen
	// (§ 249 Abs. 2 Satz 2 HGB).
	ProvisionRelease ProvisionMovementKind = "release"
	// ProvisionUnwinding ist die Aufzinsung: der Barwert wächst mit dem
	// Näherrücken der Fälligkeit. Der Gegenposten ist Zinsaufwand, nicht der
	// ursprüngliche Aufwand — § 277 Abs. 5 Satz 1 HGB verlangt den Ausweis unter
	// „Zinsen und ähnliche Aufwendungen".
	ProvisionUnwinding ProvisionMovementKind = "unwinding"
)

// Label benennt die Bewegungsart.
func (k ProvisionMovementKind) Label() string {
	switch k {
	case ProvisionFormation:
		return "Bildung"
	case ProvisionIncrease:
		return "Zuführung"
	case ProvisionConsumption:
		return "Verbrauch"
	case ProvisionRelease:
		return "Auflösung"
	case ProvisionUnwinding:
		return "Aufzinsung"
	}
	return string(k)
}

// BalanceEffect ist die Wirkung der Bewegung auf den Rückstellungsbestand.
func (k ProvisionMovementKind) BalanceEffect() int {
	switch k {
	case ProvisionFormation, ProvisionIncrease, ProvisionUnwinding:
		return 1
	case ProvisionConsumption, ProvisionRelease:
		return -1
	}
	return 0
}

// ProvisionMovement ist eine Wertänderung einer Rückstellung.
type ProvisionMovement struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	ProvisionID uint                  `gorm:"index;not null" json:"provisionId"`
	Kind        ProvisionMovementKind `gorm:"size:20;not null;index" json:"kind"`

	Date       string `gorm:"size:10;not null;index" json:"date"`
	FiscalYear int    `gorm:"index;not null" json:"fiscalYear"`
	// Amount ist immer positiv; die Richtung folgt aus der Art.
	Amount Cents `gorm:"not null" json:"amount"`
	// Reason ist bei der Auflösung Pflicht: eine Rückstellung darf nur
	// aufgelöst werden, soweit der Grund entfallen ist (§ 249 Abs. 2 Satz 2
	// HGB). Ohne festgehaltenen Grund ist von außen nicht mehr zu erkennen, ob
	// aufgelöst oder das Ergebnis geglättet wurde.
	Reason string `gorm:"size:500;serializer:encrypted" json:"reason,omitempty"`

	JournalEntryID *uint  `gorm:"index" json:"journalEntryId,omitempty"`
	EntryNumber    string `gorm:"-" json:"entryNumber,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// Provision ist eine Rückstellung mit ihren Bewegungen.
type Provision struct {
	ID         uint          `gorm:"primaryKey" json:"id"`
	FiscalYear int           `gorm:"index;not null" json:"fiscalYear"`
	Kind       ProvisionKind `gorm:"size:30;not null;index" json:"kind"`
	Text       string        `gorm:"size:255;not null;serializer:encrypted" json:"text"`

	// SettlementAmount ist der Erfüllungsbetrag nach § 253 Abs. 1 Satz 2 HGB:
	// der Betrag, der nach vernünftiger kaufmännischer Beurteilung notwendig
	// ist — künftige Preis- und Kostensteigerungen eingeschlossen.
	SettlementAmount Cents `gorm:"not null" json:"settlementAmount"`
	// ExpectedDate ist der erwartete Erfüllungszeitpunkt. Er entscheidet über
	// die Abzinsung: § 253 Abs. 2 Satz 1 HGB verlangt sie erst bei einer
	// Restlaufzeit von mehr als einem Jahr.
	ExpectedDate string `gorm:"size:10;not null" json:"expectedDate"`

	// DiscountedAmount ist der abgezinste Wert zum Bilanzstichtag; gleich dem
	// Erfüllungsbetrag, wo nicht abgezinst wird. DiscountRateMicros hält den
	// verwendeten Satz fest, damit die Rechnung später nachvollziehbar bleibt,
	// auch wenn die Zinstabelle fortgeschrieben wurde.
	DiscountedAmount   Cents `gorm:"not null" json:"discountedAmount"`
	DiscountRateMicros int64 `gorm:"default:0" json:"discountRateMicros,omitempty"`

	BalanceAccount string `gorm:"size:10;not null" json:"balanceAccount"`
	ExpenseAccount string `gorm:"size:10;not null" json:"expenseAccount"`

	// Reason ist Pflicht. Eine Rückstellung ist eine Schätzung; ohne den Grund
	// und die Grundlage der Schätzung ist sie für einen Prüfer eine Zahl ohne
	// Herkunft.
	Reason string `gorm:"size:1000;not null;serializer:encrypted" json:"reason"`

	// SettledOn ist gesetzt, wenn die Rückstellung erledigt ist — verbraucht
	// oder aufgelöst.
	SettledOn string `gorm:"size:10" json:"settledOn,omitempty"`

	Movements []ProvisionMovement `gorm:"foreignKey:ProvisionID;constraint:OnDelete:CASCADE" json:"movements"`
	CreatedAt time.Time           `json:"createdAt"`
}

// Balance ist der Bestand aus allen Bewegungen.
func (p *Provision) Balance() Cents {
	var total Cents
	for _, m := range p.Movements {
		total += Cents(m.Kind.BalanceEffect()) * m.Amount
	}
	return total
}

// BalanceAt ist der Bestand am Ende eines Geschäftsjahres.
func (p *Provision) BalanceAt(fiscalYear int) Cents {
	var total Cents
	for _, m := range p.Movements {
		if m.FiscalYear > fiscalYear {
			continue
		}
		total += Cents(m.Kind.BalanceEffect()) * m.Amount
	}
	return total
}

// IsOpen meldet, ob die Rückstellung noch besteht.
func (p *Provision) IsOpen() bool { return p.SettledOn == "" && p.Balance() > 0 }

// Validate prüft die Invarianten einer Rückstellung.
func (p *Provision) Validate() error {
	if !p.Kind.Valid() {
		return fmt.Errorf("unbekannte Art der Rückstellung %q", p.Kind)
	}
	if p.Text == "" {
		return fmt.Errorf("zur Rückstellung gehört ein Text, der sagt, wofür sie gebildet wird")
	}
	if p.SettlementAmount <= 0 {
		return fmt.Errorf("der Erfüllungsbetrag der Rückstellung muss größer als null sein")
	}
	if p.ExpectedDate == "" {
		return fmt.Errorf("die Rückstellung braucht den erwarteten Erfüllungszeitpunkt: von ihm hängt ab, ob abgezinst wird (§ 253 Abs. 2 HGB)")
	}
	if p.Reason == "" {
		return fmt.Errorf("eine Rückstellung ist eine Schätzung und braucht ihre Begründung")
	}
	if p.BalanceAccount == "" || p.ExpenseAccount == "" {
		return fmt.Errorf("die Rückstellung braucht ein Bilanz- und ein Aufwandskonto")
	}
	return nil
}

// ProvisionMirrorRow ist eine Zeile des Rückstellungsspiegels.
//
// Die Spalten sind die des Anhangs: Anfangsbestand, Zuführung, Verbrauch,
// Auflösung, Aufzinsung, Endbestand. Sie gehen per Definition auf —
// Endbestand = Anfangsbestand + Zuführung + Aufzinsung − Verbrauch − Auflösung
// —, und genau das prüft der Test.
type ProvisionMirrorRow struct {
	Kind    ProvisionKind `json:"kind"`
	Label   string        `json:"label"`
	Account string        `json:"account"`

	Opening   Cents `json:"opening"`
	Additions Cents `json:"additions"`
	Used      Cents `json:"used"`
	Released  Cents `json:"released"`
	Unwinding Cents `json:"unwinding"`
	Closing   Cents `json:"closing"`
}

// ProvisionMirror ist der Rückstellungsspiegel eines Geschäftsjahres. Er ist
// Bestandteil des Anhangs (§ 285 HGB) und steht deshalb hier und nicht in der
// Auswertung: der Jahresabschluss trägt ihn mit.
type ProvisionMirror struct {
	FiscalYear int                  `json:"fiscalYear"`
	Rows       []ProvisionMirrorRow `json:"rows"`
	Total      ProvisionMirrorRow   `json:"total"`
}

// DiscountRate ist ein Satz der Abzinsungszinssatzverordnung, wie ihn die
// Deutsche Bundesbank monatlich veröffentlicht.
//
// Die Sätze stehen in einer pflegbaren Tabelle und nicht im Code. Sie ändern
// sich jeden Monat, und ein Programm, das sie mitbringt, wäre spätestens einen
// Monat nach seiner Auslieferung falsch — schlimmer noch: falsch, ohne es zu
// sagen. Fehlt der Satz, zinst Buchfink nicht ab und erzeugt einen Befund.
type DiscountRate struct {
	// Month ist der Monat der Veröffentlichung als JJJJ-MM.
	Month string `gorm:"primaryKey;size:7" json:"month"`
	// Years ist die Restlaufzeit in Jahren (1 bis 50).
	Years int `gorm:"primaryKey;autoIncrement:false" json:"years"`
	// RateMicros ist der Zinssatz in Millionsteln (1,50 % = 15000).
	RateMicros int64 `gorm:"not null" json:"rateMicros"`
	// Average ist die Mittelungsdauer: sieben Jahre für Rückstellungen nach
	// § 253 Abs. 2 Satz 1 HGB, zehn Jahre für Altersversorgungsverpflichtungen
	// nach Satz 2. Sie stehen nebeneinander, weil dieselbe Veröffentlichung
	// beide enthält und die Wahl vom Zweck der Rückstellung abhängt.
	Average   int       `gorm:"primaryKey;autoIncrement:false;default:7" json:"average"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate prüft einen Zinssatz.
func (r *DiscountRate) Validate() error {
	if len(r.Month) != 7 || r.Month[4] != '-' {
		return fmt.Errorf("der Monat %q ist kein Monat im Format JJJJ-MM", r.Month)
	}
	if r.Years < 1 || r.Years > 50 {
		return fmt.Errorf("die Restlaufzeit %d liegt außerhalb von 1 bis 50 Jahren", r.Years)
	}
	if r.Average != 7 && r.Average != 10 {
		return fmt.Errorf("die Mittelungsdauer %d gibt es nicht; die Verordnung kennt sieben und zehn Jahre", r.Average)
	}
	if r.RateMicros < 0 {
		return fmt.Errorf("ein negativer Abzinsungssatz ist nicht vorgesehen")
	}
	return nil
}

// ProvisionRepository persistiert die Rückstellungen.
type ProvisionRepository interface {
	FindAll(ctx context.Context) ([]Provision, error)
	FindByYear(ctx context.Context, fiscalYear int) ([]Provision, error)
	FindByID(ctx context.Context, id uint) (*Provision, error)
	Save(ctx context.Context, provision *Provision) error
	AddMovement(ctx context.Context, movement *ProvisionMovement) error
	Delete(ctx context.Context, id uint) error
}

// DiscountRateRepository persistiert die Zinssätze der Bundesbank.
type DiscountRateRepository interface {
	FindByMonth(ctx context.Context, month string) ([]DiscountRate, error)
	// FindLatestUpTo liefert die Sätze des jüngsten Monats, der nicht nach dem
	// gesuchten liegt. Die Veröffentlichung eines Monats erscheint erst danach;
	// zum Bilanzstichtag zählt der Satz des Stichtagsmonats, und wer eine
	// ältere Tabelle gepflegt hat, soll damit rechnen können statt gar nicht.
	FindLatestUpTo(ctx context.Context, month string) ([]DiscountRate, error)
	Months(ctx context.Context) ([]string, error)
	Save(ctx context.Context, rates []DiscountRate) error
}
