package domain

import (
	"context"
	"time"
)

// Der Pflichtenkalender führt jede Pflicht der Gesellschaft als Instanz einer
// Pflichtart mit Zeitraum, Fälligkeit, Stand und Nachweis. Die Instanzen
// entstehen aus Stammdaten und Buchführung; gespeichert wird nur, was ein
// Mensch bestätigt, ein Agent vorbereitet oder ein Anlass ausgelöst hat.

// ObligationKind ist der stabile Schlüssel einer Pflichtart. Er steht in den
// Schlüsseln der Pflichten und ändert sich deshalb nicht. Den Abschnitt in
// buchfink-grundlagen/references/pflichten.md nennt Anchor.
type ObligationKind string

const (
	ObligationVatReturn          ObligationKind = "ustva"
	ObligationSpecialPrepayment  ObligationKind = "sondervorauszahlung"
	ObligationECSalesList        ObligationKind = "zm"
	ObligationCorporatePrepay    ObligationKind = "kst-vorauszahlung"
	ObligationTradePrepay        ObligationKind = "gewst-vorauszahlung"
	ObligationPayrollTax         ObligationKind = "lohnsteuer-anmeldung"
	ObligationMonthCommit        ObligationKind = "festschreibung"
	ObligationClosingBookings    ObligationKind = "abschlussbuchungen"
	ObligationStatementPrep      ObligationKind = "aufstellung"
	ObligationAdoption           ObligationKind = "feststellung"
	ObligationAppropriation      ObligationKind = "ergebnisverwendung"
	ObligationEBilanz            ObligationKind = "e-bilanz"
	ObligationCorporateTaxReturn ObligationKind = "kst-erklaerung"
	ObligationTradeTaxReturn     ObligationKind = "gewst-erklaerung"
	ObligationAnnualVatReturn    ObligationKind = "ust-erklaerung"
	ObligationDisclosure         ObligationKind = "offenlegung"
	ObligationTransparency       ObligationKind = "transparenzregister"
	ObligationShareholderList    ObligationKind = "gesellschafterliste"
	ObligationTaxRegistration    ObligationKind = "steuerliche-erfassung"
	ObligationTradeRegistration  ObligationKind = "gewerbeanmeldung"
	ObligationOpeningBalance     ObligationKind = "eroeffnungsbilanz"
	ObligationOpeningEBilanz     ObligationKind = "e-bilanz-eroeffnung"
	ObligationWithholdingTax     ObligationKind = "kapitalertragsteuer"
)

// ObligationSkillFile ist die Skill-Referenz, deren Anker die Pflichtarten sind.
const ObligationSkillFile = "buchfink-grundlagen/references/pflichten.md"

// obligationAnchors sind die Pflichtarten, deren Abschnitt in der
// Skill-Referenz anders heißt als ihr Schlüssel.
var obligationAnchors = map[ObligationKind]string{
	ObligationMonthCommit:     "monatsfestschreibung",
	ObligationClosingBookings: "bilanzstichtag",
	ObligationEBilanz:         "ebilanz",
	ObligationOpeningEBilanz:  "ebilanz-eroeffnung",
	ObligationTaxRegistration: "fragebogen-steuerliche-erfassung",
}

// ObligationKinds sind alle Pflichtarten.
func ObligationKinds() []ObligationKind {
	return []ObligationKind{ObligationVatReturn, ObligationSpecialPrepayment, ObligationECSalesList,
		ObligationCorporatePrepay, ObligationTradePrepay, ObligationPayrollTax, ObligationMonthCommit,
		ObligationClosingBookings, ObligationStatementPrep, ObligationAdoption, ObligationAppropriation,
		ObligationEBilanz, ObligationCorporateTaxReturn, ObligationTradeTaxReturn, ObligationAnnualVatReturn,
		ObligationDisclosure, ObligationTransparency, ObligationShareholderList, ObligationTaxRegistration,
		ObligationTradeRegistration, ObligationOpeningBalance, ObligationOpeningEBilanz, ObligationWithholdingTax}
}

// ObligationKindOf liest eine Pflichtart, auch unter dem Anker ihres
// Abschnitts in der Skill-Referenz („ebilanz" für „e-bilanz").
func ObligationKindOf(s string) (ObligationKind, bool) {
	for _, k := range ObligationKinds() {
		if string(k) == s || k.Anchor() == s {
			return k, true
		}
	}
	return "", false
}

// Anchor ist der Anker des Abschnitts in der Skill-Referenz.
func (k ObligationKind) Anchor() string {
	if a, ok := obligationAnchors[k]; ok {
		return a
	}
	return string(k)
}

// SkillRef ist der Verweis auf die Skill-Referenz der Pflichtart.
func (k ObligationKind) SkillRef() string { return ObligationSkillFile + "#" + k.Anchor() }

// ObligationStatus ist der Stand einer Pflicht.
type ObligationStatus string

const (
	ObligationOpen     ObligationStatus = "open"
	ObligationPrepared ObligationStatus = "prepared"
	ObligationDone     ObligationStatus = "done"
	ObligationOverdue  ObligationStatus = "overdue"
	ObligationWaived   ObligationStatus = "waived"
)

// Valid meldet einen bekannten Stand.
func (s ObligationStatus) Valid() bool {
	switch s {
	case ObligationOpen, ObligationPrepared, ObligationDone, ObligationOverdue, ObligationWaived:
		return true
	}
	return false
}

// Closed meldet, ob die Pflicht nichts mehr verlangt.
func (s ObligationStatus) Closed() bool { return s == ObligationDone || s == ObligationWaived }

// Label ist der Stand in der Sprache der Oberfläche.
func (s ObligationStatus) Label() string {
	switch s {
	case ObligationPrepared:
		return "vorbereitet"
	case ObligationDone:
		return "erledigt"
	case ObligationOverdue:
		return "überfällig"
	case ObligationWaived:
		return "entfällt"
	}
	return "offen"
}

// ObligationChannel ist der Weg, auf dem die Pflicht erfüllt wird.
type ObligationChannel string

const (
	ChannelElster               ObligationChannel = "elster"
	ChannelCompanyRegister      ObligationChannel = "unternehmensregister"
	ChannelBZSt                 ObligationChannel = "bzst"
	ChannelTransparencyRegister ObligationChannel = "transparenzregister"
	ChannelNotary               ObligationChannel = "notar"
	ChannelTradeOffice          ObligationChannel = "gewerbeamt"
	ChannelPayrollProvider      ObligationChannel = "lohnanbieter"
	ChannelInternal             ObligationChannel = "intern"
)

// ObligationResponsibility sagt, wer die Pflicht trägt.
type ObligationResponsibility string

const (
	// ResponsibilityAgentPrepares: der Agent bereitet vor, der Mensch löst aus
	// und bestätigt.
	ResponsibilityAgentPrepares ObligationResponsibility = "agent_prepares"
	// ResponsibilityHumanConfirms: der Mensch handelt selbst und bestätigt.
	ResponsibilityHumanConfirms ObligationResponsibility = "human_confirms"
	// ResponsibilityTaxAdvisor: der beauftragte Steuerberater erstellt.
	ResponsibilityTaxAdvisor ObligationResponsibility = "tax_advisor"
)

// EvidenceSource sagt, woher der Stand einer Pflichtart kommt.
type EvidenceSource string

const (
	// EvidenceFromData: allein aus den Daten (Transferticket, Festschreibung,
	// Abschlussstand). Ein Mensch bestätigt sie nicht daneben.
	EvidenceFromData EvidenceSource = "data"
	// EvidenceFromHuman: allein durch Bestätigung mit Nachweis.
	EvidenceFromHuman EvidenceSource = "human"
	// EvidenceFromEither: aus den Daten oder durch Bestätigung.
	EvidenceFromEither EvidenceSource = "either"
)

// EvidenceKind ist die Art eines Nachweises.
type EvidenceKind string

const (
	EvidenceTransferTicket EvidenceKind = "transfer_ticket"
	EvidenceFileNumber     EvidenceKind = "file_number"
	EvidenceOrderNumber    EvidenceKind = "order_number"
	EvidenceBooking        EvidenceKind = "booking"
	EvidenceCommitment     EvidenceKind = "commitment"
	EvidenceResolution     EvidenceKind = "resolution"
	EvidenceStatus         EvidenceKind = "status"
	EvidenceConfirmation   EvidenceKind = "confirmation"
)

// Valid meldet eine bekannte Nachweisart.
func (k EvidenceKind) Valid() bool {
	switch k {
	case EvidenceTransferTicket, EvidenceFileNumber, EvidenceOrderNumber, EvidenceBooking,
		EvidenceCommitment, EvidenceResolution, EvidenceStatus, EvidenceConfirmation:
		return true
	}
	return false
}

// ObligationType ist ein Eintrag im Katalog der Pflichtarten.
type ObligationType struct {
	Kind           ObligationKind           `json:"kind"`
	Title          string                   `json:"title"`
	Basis          string                   `json:"basis"`
	Rhythm         string                   `json:"rhythm"`
	Channel        ObligationChannel        `json:"channel"`
	Responsibility ObligationResponsibility `json:"responsibility"`
	Evidence       EvidenceSource           `json:"evidence"`
	// ShiftsToWorkday: das Fristende verschiebt sich nach § 108 Abs. 3 AO.
	// Nicht für handelsrechtliche Fristen, die Gewerbeanzeige, interne
	// Pflichten und Zahltermine.
	ShiftsToWorkday bool   `json:"shiftsToWorkday"`
	Description     string `json:"description"`
	SkillRef        string `json:"skillRef"`
}

// ObligationEvidence ist der Nachweis einer erledigten Pflicht.
type ObligationEvidence struct {
	Date      string       `json:"date"`
	Kind      EvidenceKind `json:"kind"`
	Reference string       `json:"reference,omitempty"`
	Note      string       `json:"note,omitempty"`
	// Automatic heißt: aus den Daten abgeleitet, nicht bestätigt.
	Automatic bool   `json:"automatic"`
	Channel   string `json:"channel,omitempty"`
	Actor     string `json:"actor,omitempty"`
}

// ObligationPreparation ist die Meldung, dass eine Pflicht vorbereitet ist.
type ObligationPreparation struct {
	At      string `json:"at"`
	Note    string `json:"note,omitempty"`
	Channel string `json:"channel"`
	Actor   string `json:"actor"`
}

// Obligation ist eine Pflicht-Instanz: eine Pflichtart für einen Zeitraum.
type Obligation struct {
	// Key ist Pflichtart und Zeitraum, etwa „ustva.2026-Q1".
	Key            string                   `json:"key"`
	Kind           ObligationKind           `json:"kind"`
	Title          string                   `json:"title"`
	Description    string                   `json:"description"`
	Basis          string                   `json:"basis"`
	Channel        ObligationChannel        `json:"channel"`
	Responsibility ObligationResponsibility `json:"responsibility"`
	Evidence       EvidenceSource           `json:"evidenceSource"`
	SkillRef       string                   `json:"skillRef"`
	PeriodKey      string                   `json:"periodKey"`
	PeriodLabel    string                   `json:"periodLabel"`
	PeriodFrom     string                   `json:"periodFrom,omitempty"`
	PeriodTo       string                   `json:"periodTo,omitempty"`
	FiscalYear     int                      `json:"fiscalYear"`
	// StatutoryDate ist das gesetzliche Fristende, DueDate das nach
	// § 108 Abs. 3 AO verschobene; ShiftReason nennt den Grund.
	StatutoryDate string `json:"statutoryDate"`
	DueDate       string `json:"dueDate"`
	ShiftReason   string `json:"shiftReason,omitempty"`
	// Immediate heißt „unverzüglich": die Frist beginnt am Fälligkeitstag.
	Immediate bool             `json:"immediate,omitempty"`
	Status    ObligationStatus `json:"status"`
	// Proof ist der Nachweis einer erledigten Pflicht.
	Proof        *ObligationEvidence    `json:"proof,omitempty"`
	Preparation  *ObligationPreparation `json:"preparation,omitempty"`
	WaivedReason string                 `json:"waivedReason,omitempty"`
	Hints        []string               `json:"hints"`
}

// ObligationRecord hält fest, was ein Mensch bestätigt oder ein Agent
// vorbereitet hat. Je Instanz eine Zeile; die Geschichte steht im
// Änderungsprotokoll.
type ObligationRecord struct {
	Key  string         `gorm:"primaryKey;size:120" json:"key"`
	Kind ObligationKind `gorm:"size:40;not null;index" json:"kind"`
	// Status ist done oder waived, leer solange nur eine Vorbereitung vorliegt.
	Status            ObligationStatus `gorm:"size:20" json:"status"`
	EvidenceDate      string           `gorm:"size:10" json:"evidenceDate,omitempty"`
	EvidenceKind      EvidenceKind     `gorm:"size:30" json:"evidenceKind,omitempty"`
	EvidenceReference string           `gorm:"size:200" json:"evidenceReference,omitempty"`
	EvidenceNote      string           `gorm:"size:1000" json:"evidenceNote,omitempty"`
	WaivedReason      string           `gorm:"size:1000" json:"waivedReason,omitempty"`
	DecidedChannel    string           `gorm:"size:60" json:"decidedChannel,omitempty"`
	DecidedActor      string           `gorm:"size:200" json:"decidedActor,omitempty"`
	DecidedAt         *time.Time       `json:"decidedAt,omitempty"`
	PreparedNote      string           `gorm:"size:1000" json:"preparedNote,omitempty"`
	PreparedChannel   string           `gorm:"size:60" json:"preparedChannel,omitempty"`
	PreparedActor     string           `gorm:"size:200" json:"preparedActor,omitempty"`
	PreparedAt        *time.Time       `json:"preparedAt,omitempty"`
	UpdatedAt         time.Time        `json:"updatedAt"`
}

// ObligationEventKind ist ein Anlass, der Pflichten auslöst.
type ObligationEventKind string

const (
	// EventShareholderChange: Veränderung in den Personen der Gesellschafter
	// oder im Umfang ihrer Beteiligung (§ 40 Abs. 1 GmbHG, § 20 Abs. 1 GwG).
	EventShareholderChange ObligationEventKind = "gesellschafterwechsel"
	// EventBeneficialOwnerChange: andere Änderung bei den wirtschaftlich
	// Berechtigten (§ 20 Abs. 1 GwG).
	EventBeneficialOwnerChange ObligationEventKind = "wirtschaftlich-berechtigte"
	// EventCorporatePrepaymentNotice: Vorauszahlungsbescheid zur
	// Körperschaftsteuer (§ 37 EStG, § 31 KStG).
	EventCorporatePrepaymentNotice ObligationEventKind = "vorauszahlungsbescheid-kst"
	// EventTradePrepaymentNotice: Vorauszahlungsbescheid zur Gewerbesteuer
	// (§ 19 GewStG).
	EventTradePrepaymentNotice ObligationEventKind = "vorauszahlungsbescheid-gewst"
)

// Valid meldet einen bekannten Anlass.
func (k ObligationEventKind) Valid() bool {
	switch k {
	case EventShareholderChange, EventBeneficialOwnerChange,
		EventCorporatePrepaymentNotice, EventTradePrepaymentNotice:
		return true
	}
	return false
}

// ObligationEvent ist ein erfasster Anlass.
type ObligationEvent struct {
	ID        uint                `gorm:"primaryKey" json:"id"`
	Kind      ObligationEventKind `gorm:"size:40;not null;index" json:"kind"`
	Date      string              `gorm:"size:10;not null" json:"date"`
	Note      string              `gorm:"size:1000" json:"note,omitempty"`
	Channel   string              `gorm:"size:60" json:"channel"`
	Actor     string              `gorm:"size:200" json:"actor"`
	CreatedAt time.Time           `json:"createdAt"`
}

// ObligationRepository speichert Bestätigungen, Vorbereitungen und Anlässe.
type ObligationRepository interface {
	FindRecords(ctx context.Context) ([]ObligationRecord, error)
	// FindRecord liefert nil ohne Fehler, wenn es keinen Eintrag gibt.
	FindRecord(ctx context.Context, key string) (*ObligationRecord, error)
	SaveRecord(ctx context.Context, rec *ObligationRecord) error
	FindEvents(ctx context.Context) ([]ObligationEvent, error)
	CreateEvent(ctx context.Context, ev *ObligationEvent) error
}
