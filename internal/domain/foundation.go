package domain

import (
	"context"
	"time"
)

// Die Gründung einer Kapitalgesellschaft, als Datensatz.
//
// Zwischen dem Notartermin und der Eintragung ins Handelsregister liegt eine
// eigene Rechtsform: die Vorgesellschaft. Sie ist bereits buchführungspflichtig
// und mit der späteren GmbH identisch, aber die Haftungsbeschränkung greift noch
// nicht (§ 11 Abs. 1 GmbHG). Zwei Folgen daraus muss eine Buchhaltung kennen:
//
//  1. Wer vor der Eintragung im Namen der Gesellschaft handelt, haftet
//     persönlich und solidarisch (§ 11 Abs. 2 GmbHG). Das endet mit der
//     Eintragung — deshalb ist ihr Datum eine Angabe, keine Randnotiz.
//  2. Ist das Reinvermögen am Tag der Eintragung kleiner als das Stammkapital,
//     schulden die Gesellschafter die Differenz. Die Zahlen dafür stehen bereits
//     im Journal; was fehlte, war die Rechnung.
//
// Gespeichert wird deshalb nur, was nicht aus den Buchungen folgt: die Daten der
// Gründung, das gezeichnete Kapital und wer es übernommen hat.

// FoundationStage ist die Phase der Gründung.
//
// Sie wird nicht gespeichert, sondern aus dem Eintragungsdatum abgeleitet. Ein
// Status, den man unabhängig davon setzen kann, geht früher oder später mit dem
// Datum auseinander — und dann steht in der Oberfläche etwas anderes als in der
// Rechnung.
type FoundationStage string

const (
	// FoundationStageVorgesellschaft läuft von der Beurkundung des
	// Gesellschaftsvertrags (§ 2 GmbHG) bis zur Eintragung.
	FoundationStageVorgesellschaft FoundationStage = "vorgesellschaft"
	// FoundationStageEingetragen beginnt mit der Eintragung ins Handelsregister.
	// Ab hier besteht die Gesellschaft als juristische Person (§ 11 Abs. 1 GmbHG).
	FoundationStageEingetragen FoundationStage = "eingetragen"
)

// Label renders the stage for the UI.
func (s FoundationStage) Label() string {
	switch s {
	case FoundationStageEingetragen:
		return "Eingetragen"
	default:
		return "In Gründung"
	}
}

// ContributionKind unterscheidet Bar- von Sacheinlage.
//
// Der Unterschied ist keine Formalie: eine Sacheinlage ist vor der Anmeldung
// vollständig zu bewirken (§ 7 Abs. 3 GmbHG), zählt bei der Mindesteinzahlung
// mit ihrem Nennbetrag statt mit dem eingezahlten Geld (§ 7 Abs. 2 Satz 2), und
// bei der UG ist sie ausgeschlossen (§ 5a Abs. 2 Satz 2 GmbHG).
type ContributionKind string

const (
	ContributionCash   ContributionKind = "cash"
	ContributionInKind ContributionKind = "kind"
)

// Label renders the contribution kind for the UI.
func (k ContributionKind) Label() string {
	if k == ContributionInKind {
		return "Sacheinlage"
	}
	return "Bareinlage"
}

// Valid reports whether the kind is one of the known ones.
func (k ContributionKind) Valid() bool {
	return k == ContributionCash || k == ContributionInKind
}

// Foundation is the Gründung of the tenant — at most one row per Mandant.
//
// Es ist bewusst eine Tabelle und kein Bündel von SettingItem-Zeilen: an der
// Gründung hängt eine Gesellschafterliste, und ein Schlüssel-Wert-Speicher
// bildet eine Liste nur als serialisierter Klumpen ab, der sich weder abfragen
// noch mit einem Fremdschlüssel verbinden lässt.
type Foundation struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// NotarizedOn ist der Tag der notariellen Beurkundung des
	// Gesellschaftsvertrags (§ 2 Abs. 1 GmbHG). Mit ihm entsteht die
	// Vorgesellschaft, beginnt die Buchführungspflicht und laufen alle Fristen
	// der Gründung an.
	NotarizedOn string `gorm:"size:10;not null" json:"notarizedOn"` // YYYY-MM-DD

	// RegisteredOn ist der Tag der Eintragung ins Handelsregister. Leer heißt:
	// die Gesellschaft ist noch Vorgesellschaft. Auf dieses Datum wird die
	// Unterbilanz festgestellt.
	RegisteredOn   string `gorm:"size:10" json:"registeredOn"`
	RegisterCourt  string `gorm:"size:120" json:"registerCourt"` // Amtsgericht
	RegisterNumber string `gorm:"size:40" json:"registerNumber"` // z. B. "HRB 123456"

	// ShareCapital ist das Stammkapital laut Gesellschaftsvertrag (bei der AG das
	// Grundkapital). Die Summe der übernommenen Einlagen muss ihm entsprechen
	// (§ 5 Abs. 3 Satz 2 GmbHG).
	ShareCapital Cents `gorm:"not null" json:"shareCapital"`

	// FoundationCostCap ist der Gründungsaufwand, den der Gesellschaftsvertrag
	// der Gesellschaft auferlegt — betragsmäßig, sonst trägt ihn das Registerrecht
	// nicht.
	//
	// Er gehört hierher, weil er die Vorbelastungshaftung begrenzt: Aufwand, den
	// die Satzung der Gesellschaft zuweist, ist zulässig getragen und wird den
	// Gesellschaftern nicht noch einmal in Rechnung gestellt. Ohne eine solche
	// Klausel ist der gesamte Gründungsaufwand Vorbelastung — und weil § 248
	// Abs. 1 Nr. 1 HGB verbietet, ihn zu aktivieren, schlägt er ungebremst durch.
	FoundationCostCap Cents `json:"foundationCostCap"`

	Shareholders []Shareholder `gorm:"foreignKey:FoundationID;constraint:OnDelete:CASCADE" json:"shareholders"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Stage derives the phase from the registration date.
func (f *Foundation) Stage() FoundationStage {
	if f != nil && f.RegisteredOn != "" {
		return FoundationStageEingetragen
	}
	return FoundationStageVorgesellschaft
}

// IsRegistered reports whether the company is entered in the Handelsregister.
func (f *Foundation) IsRegistered() bool {
	return f != nil && f.RegisteredOn != ""
}

// SubscribedCapital is the sum of the shares the shareholders took over. It must
// equal ShareCapital (§ 5 Abs. 3 Satz 2 GmbHG).
func (f *Foundation) SubscribedCapital() Cents {
	var sum Cents
	for _, s := range f.Shareholders {
		sum += s.ShareCapital
	}
	return sum
}

// PaidInCapital is the sum actually contributed so far.
func (f *Foundation) PaidInCapital() Cents {
	var sum Cents
	for _, s := range f.Shareholders {
		sum += s.PaidIn
	}
	return sum
}

// Shareholder is one Gesellschafter and the Geschäftsanteil they took over.
type Shareholder struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	FoundationID uint `gorm:"index;not null" json:"foundationId"`

	// Name ist personenbezogen und liegt deshalb verschlüsselt, wie jedes andere
	// Namensfeld der Anwendung.
	Name string `gorm:"size:255;not null;serializer:encrypted" json:"name"`

	// ShareCapital ist der Nennbetrag des übernommenen Geschäftsanteils,
	// PaidIn das darauf tatsächlich Geleistete.
	ShareCapital Cents `gorm:"not null" json:"shareCapital"`
	PaidIn       Cents `json:"paidIn"`

	Kind ContributionKind `gorm:"size:10;not null;default:'cash'" json:"kind"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FoundationTask records that a Gründungspflicht was fulfilled — with the day it
// happened.
//
// Die Steuerfristen der laufenden Buchhaltung hakt Buchfink im `localStorage` ab;
// das ist eine Merkhilfe und darf eine sein. Eine Gründungspflicht ist etwas
// anderes: dass der Fragebogen zur steuerlichen Erfassung am 12. Oktober
// übermittelt wurde, ist eine Tatsache über das Unternehmen und gehört in die
// Datenbank — mit Datum, nicht als Haken, und nicht in den Browserspeicher.
type FoundationTask struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	FoundationID uint   `gorm:"index;not null" json:"foundationId"`
	Key          string `gorm:"size:40;not null;index" json:"key"`
	DoneOn       string `gorm:"size:10;not null" json:"doneOn"` // YYYY-MM-DD
	Note         string `gorm:"size:500;serializer:encrypted" json:"note,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// -------------------------------------------------------------
// Ergebnisse der Rechnung
// -------------------------------------------------------------

// Unterbilanz is the Vorbelastungsrechnung: what the shareholders owe the
// company because its net assets fall short of the Stammkapital.
//
// Die Haftung ist Richterrecht, kein Paragraf: Die Gesellschafter haften
// anteilig für die Differenz zwischen dem Stammkapital und dem Reinvermögen im
// Zeitpunkt der Eintragung. Solange die Eintragung fehlt, ist jede Zahl
// vorläufig — sie wächst mit jeder Buchung weiter. Deshalb trägt das Ergebnis
// seinen Stichtag und sagt, ob er der endgültige ist.
type Unterbilanz struct {
	AsOf    string `json:"asOf"`    // YYYY-MM-DD
	IsFinal bool   `json:"isFinal"` // true ab der Eintragung

	ShareCapital Cents `json:"shareCapital"`
	Assets       Cents `json:"assets"`      // Summe der Aktiva
	Liabilities  Cents `json:"liabilities"` // Summe der Schulden
	NetAssets    Cents `json:"netAssets"`   // Reinvermögen = Aktiva − Schulden

	// Shortfall ist die rohe Differenz, Covered der davon durch die
	// Gründungsaufwandsklausel gedeckte Teil, Amount der Rest — und nur der ist
	// die Haftung.
	Shortfall Cents `json:"shortfall"`
	Covered   Cents `json:"covered"`
	Amount    Cents `json:"amount"`

	Shares []UnterbilanzShare `json:"shares"`
}

// UnterbilanzShare is one shareholder's part of the liability.
type UnterbilanzShare struct {
	ShareholderID uint   `json:"shareholderId"`
	Name          string `json:"name"`
	ShareCapital  Cents  `json:"shareCapital"`
	Amount        Cents  `json:"amount"`
}

// AnmeldungCheck answers whether enough has been contributed for the
// Handelsregisteranmeldung.
type AnmeldungCheck struct {
	LegalForm       string `json:"legalForm"`
	MinShareCapital Cents  `json:"minShareCapital"`
	ShareCapital    Cents  `json:"shareCapital"`
	RequiredPaidIn  Cents  `json:"requiredPaidIn"`
	ActualPaidIn    Cents  `json:"actualPaidIn"`
	IsSatisfied     bool   `json:"isSatisfied"`

	// Findings nennen im Klartext, was fehlt — je Befund ein Satz mit seiner
	// Fundstelle. Leer heißt: die Anmeldung kann erfolgen.
	Findings []string `json:"findings"`
	// Reference ist die Fundstelle der Einzahlungsregel dieser Rechtsform.
	Reference string `json:"reference"`

	Shareholders []ShareholderCheck `json:"shareholders"`
}

// ShareholderCheck is the per-share view of the Anmeldung requirement.
type ShareholderCheck struct {
	ShareholderID  uint             `json:"shareholderId"`
	Name           string           `json:"name"`
	Kind           ContributionKind `json:"kind"`
	ShareCapital   Cents            `json:"shareCapital"`
	RequiredPaidIn Cents            `json:"requiredPaidIn"`
	PaidIn         Cents            `json:"paidIn"`
	IsSatisfied    bool             `json:"isSatisfied"`
}

// FoundationDuty is one obligation arising from the founding, with the day it is
// due and whether it has been fulfilled.
type FoundationDuty struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	// DueDate ist leer, wo das Gesetz keine Frist in Tagen nennt, sondern
	// „unverzüglich" sagt. Dann trägt Deadline den Wortlaut. Eine erfundene
	// Tagesfrist wäre bequemer und falsch.
	DueDate     string `json:"dueDate"`
	Deadline    string `json:"deadline"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
	DoneOn      string `json:"doneOn"`
	IsDone      bool   `json:"isDone"`
}

// FoundationRepository persists the Gründung of a tenant.
type FoundationRepository interface {
	// Get returns the Gründung including shareholders, or nil when the tenant is
	// not a founding case.
	Get(ctx context.Context) (*Foundation, error)
	// Save writes the Gründung and replaces its shareholder list.
	Save(ctx context.Context, f *Foundation) error
	Tasks(ctx context.Context, foundationID uint) ([]FoundationTask, error)
	// CompleteTask records a fulfilled duty, replacing an earlier record of the
	// same key.
	CompleteTask(ctx context.Context, task *FoundationTask) error
	// ClearTask takes a duty back to open.
	ClearTask(ctx context.Context, foundationID uint, key string) error
}
