package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Monatsabschluss in drei Schritten (Architektur 6.2).
//
// Die Reihenfolge ist der Inhalt dieses Dienstes: erst der Prüfbericht, dann die
// Festschreibung, dann die Bestätigung der Voranmeldung. Sie ist nicht
// willkürlich — was gemeldet wird, muss vorher unveränderbar sein, sonst weicht
// die Buchführung später von der Meldung ab, und die Abweichung fällt erst dem
// Prüfer auf.
//
// Der Dienst bucht nichts und schreibt nichts fest: er sagt, wo der Monat steht.
// Festgeschrieben wird über die Bridge (CommitPeriod), bestätigt über den
// Voranmeldungsdienst — beides Vorgänge, die eine Bestätigung verlangen.

// MonthCloseStepState ist der Stand eines der drei Schritte.
type MonthCloseStepState string

const (
	// MonthStepDone ist erledigt.
	MonthStepDone MonthCloseStepState = "done"
	// MonthStepOpen steht an.
	MonthStepOpen MonthCloseStepState = "open"
	// MonthStepBlocked kann nicht ausgeführt werden, solange etwas anderes
	// offen ist — ein blockierender Befund oder die fehlende Festschreibung.
	MonthStepBlocked MonthCloseStepState = "blocked"
	// MonthStepNotApplicable entfällt in diesem Monat: der Zeitraum der
	// Voranmeldung ist das Quartal, und dann endet der Monat nach Schritt 2.
	MonthStepNotApplicable MonthCloseStepState = "not_applicable"
)

// MonthCloseStep ist ein Schritt des Monatsabschlusses.
type MonthCloseStep struct {
	Number int                 `json:"number"`
	Key    string              `json:"key"`
	Title  string              `json:"title"`
	State  MonthCloseStepState `json:"state"`
	Note   string              `json:"note"`
}

// MonthCloseState ist der Stand eines Monats.
type MonthCloseState struct {
	// Month ist der Monat als "JJJJ-MM", From und To seine Grenzen.
	Month      string `json:"month"`
	Label      string `json:"label"`
	From       string `json:"from"`
	To         string `json:"to"`
	FiscalYear int    `json:"fiscalYear"`

	Steps []MonthCloseStep `json:"steps"`

	// Findings sind die Befunde des Prüflaufs bis zum Monatsende, Blocking die
	// Zahl derer, die die Festschreibung verhindern.
	Findings []domain.CheckFinding `json:"findings"`
	Blocking int                   `json:"blocking"`

	Committed   bool   `json:"committed"`
	CommittedTo string `json:"committedTo,omitempty"`

	// Die Voranmeldung des Zeitraums, in dem der Monat endet.
	VatApplies     bool                   `json:"vatApplies"`
	VatPeriodKey   string                 `json:"vatPeriodKey,omitempty"`
	VatPeriodLabel string                 `json:"vatPeriodLabel,omitempty"`
	VatStatus      domain.VatReturnStatus `json:"vatStatus,omitempty"`
	VatDueDate     string                 `json:"vatDueDate,omitempty"`
	VatSubmittedAt string                 `json:"vatSubmittedAt,omitempty"`
	VatReturnID    uint                   `json:"vatReturnId,omitempty"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (s *MonthCloseState) EnsureLists() {
	if s.Steps == nil {
		s.Steps = make([]MonthCloseStep, 0)
	}
	if s.Findings == nil {
		s.Findings = make([]domain.CheckFinding, 0)
	}
}

// MonthVatSource liefert die Zeiträume der Voranmeldung mit ihrem Stand.
type MonthVatSource interface {
	Periods(ctx context.Context, year int) ([]VatPeriodStatus, error)
}

// MonthFiscalYearSource liefert die Spanne eines Geschäftsjahres.
//
// Sie wird gebraucht, um zu entscheiden, ob ein Monat überhaupt zum aktiven
// Geschäftsjahr gehört. Ohne sie gilt der Regelfall des Kalenderjahres; mit ihr
// stimmt die Antwort auch für ein abweichendes oder ein Rumpfgeschäftsjahr
// (§ 8b EStDV).
type MonthFiscalYearSource interface {
	FindByYear(ctx context.Context, year int) (*domain.FiscalYear, error)
}

// MonthCloseService beantwortet, wo ein Monat im Abschluss steht.
type MonthCloseService struct {
	checks             TaskCheckSource
	vat                MonthVatSource
	festschreibungRepo domain.FestschreibungRepository
	years              MonthFiscalYearSource
	fiscalYear         int
}

// NewMonthCloseService wires the three steps.
func NewMonthCloseService(
	checks TaskCheckSource,
	vat MonthVatSource,
	festschreibungRepo domain.FestschreibungRepository,
	fiscalYear int,
) *MonthCloseService {
	return &MonthCloseService{
		checks: checks, vat: vat,
		festschreibungRepo: festschreibungRepo, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear updates the active fiscal year.
func (s *MonthCloseService) SetFiscalYear(year int) { s.fiscalYear = year }

// SetFiscalYearSource hängt die Geschäftsjahre an. Ohne sie gilt ein Monat als
// zum Geschäftsjahr gehörig, wenn sein Kalenderjahr das des Geschäftsjahres ist.
func (s *MonthCloseService) SetFiscalYearSource(src MonthFiscalYearSource) { s.years = src }

// State liefert den Stand eines Monats ("JJJJ-MM").
func (s *MonthCloseService) State(ctx context.Context, month string) (*MonthCloseState, error) {
	from, to, label, err := monthBounds(month)
	if err != nil {
		return nil, err
	}
	// Der Monat muss zum aktiven Geschäftsjahr gehören.
	//
	// Am Jahreswechsel ist das der Regelfall und nicht die Ausnahme: die
	// Voranmeldung für Dezember ist am 10. Januar fällig und steht dann als
	// Frist in der Aufgabenliste. Wer sie im Geschäftsjahr 2026 anklickte, bekam
	// bisher den Dezember 2025 gegen die Bücher von 2026 gerechnet — eine leere
	// Festschreibungstabelle („nicht festgeschrieben"), einen Prüfbericht über
	// die Buchungen des falschen Jahres und „keine Voranmeldung abzugeben",
	// weil die Zeiträume 2026 den Dezember 2025 nicht enthalten. Drei Haken,
	// alle drei falsch. Da sich der Prüfbericht (CheckService) nach dem aktiven
	// Geschäftsjahr richtet, ist die ehrliche Antwort die Abweisung mit dem
	// Hinweis, das Geschäftsjahr zu wechseln.
	if err := s.requireActiveYear(ctx, month, label, from); err != nil {
		return nil, err
	}
	// Das Kalenderjahr des Monats bestimmt die Voranmeldung: die Zeiträume des
	// § 18 UStG sind Kalenderzeiträume. Im Regelfall (Geschäftsjahr =
	// Kalenderjahr) ist es dasselbe Jahr; bei abweichendem Geschäftsjahr fragt
	// der Januar zu Recht die Zeiträume seines Kalenderjahres ab.
	periodYear := yearOfDate(from)

	state := &MonthCloseState{
		Month: month, Label: label, From: from, To: to, FiscalYear: s.fiscalYear,
	}
	state.EnsureLists()

	// Schritt 2 zuerst ermitteln: die Festschreibung entscheidet mit darüber,
	// wie Schritt 1 und Schritt 3 dastehen.
	if s.festschreibungRepo != nil {
		// Nach dem Geschäftsjahr und nicht nach dem Kalenderjahr: die
		// Festschreibungen sind je Geschäftsjahr geführt.
		cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, s.fiscalYear)
		if err != nil {
			return nil, fmt.Errorf("der Stand der Festschreibung ließ sich nicht lesen: %w", err)
		}
		state.CommittedTo = cutoff
		state.Committed = cutoff != "" && cutoff >= to
	}

	checkState, checkNote := MonthStepOpen, "Der Prüfbericht wurde für diesen Monat noch nicht gerechnet."
	if s.checks != nil {
		run, err := s.checks.Preview(ctx, CheckRequest{CutoffDate: to, PeriodType: "month"})
		if err != nil {
			return nil, fmt.Errorf("der Prüfbericht ließ sich nicht rechnen: %w", err)
		}
		run.EnsureLists()
		state.Findings = run.Findings
		state.Blocking = len(run.Blocking())
		switch {
		case state.Blocking > 0:
			checkState = MonthStepBlocked
			checkNote = fmt.Sprintf(
				"%d Befunde verhindern die Festschreibung. Klären oder mit Begründung übergehen.",
				state.Blocking)
		case len(run.Findings) > 0:
			checkState = MonthStepOpen
			checkNote = fmt.Sprintf("%d Hinweise, die die Festschreibung nicht verhindern.", len(run.Findings))
		default:
			checkState = MonthStepDone
			checkNote = "Der Prüfbericht ist ohne Befund."
		}
	}
	// Ein festgeschriebener Monat hat seinen Prüfbericht hinter sich: die
	// Festschreibung setzt ihn voraus. Ein Befund, der jetzt noch auftaucht,
	// betrifft einen Zeitraum, der nicht mehr zu ändern ist — er bleibt
	// sichtbar, macht den Schritt aber nicht wieder offen.
	if state.Committed && checkState != MonthStepDone {
		checkState = MonthStepDone
		checkNote = "Der Monat ist festgeschrieben; der Prüfbericht lief davor."
	}

	commitState, commitNote := MonthStepOpen, "Der Monat ist noch nicht festgeschrieben."
	switch {
	case state.Committed:
		commitState, commitNote = MonthStepDone,
			fmt.Sprintf("Festgeschrieben bis zum %s.", state.CommittedTo)
	case checkState == MonthStepBlocked:
		commitState, commitNote = MonthStepBlocked,
			"Erst die blockierenden Befunde klären oder mit Begründung übergehen."
	}

	vatStep := s.vatStep(ctx, state, periodYear)

	state.Steps = []MonthCloseStep{
		{Number: 1, Key: "check", Title: "Prüfbericht", State: checkState, Note: checkNote},
		{Number: 2, Key: "commit", Title: "Festschreiben", State: commitState, Note: commitNote},
		vatStep,
	}
	state.EnsureLists()
	return state, nil
}

// vatStep bestimmt den dritten Schritt und füllt die Angaben zur Voranmeldung.
//
// Der Schritt entfällt in einem Monat, der kein Zeitraumende ist: wer
// vierteljährlich voranmeldet, ist im Januar nach Schritt 2 fertig, und am
// Quartalsende folgt eine Meldung über drei Monate.
func (s *MonthCloseService) vatStep(
	ctx context.Context, state *MonthCloseState, periodYear int,
) MonthCloseStep {
	step := MonthCloseStep{
		Number: 3, Key: "vat", Title: "Voranmeldung bestätigen",
		State: MonthStepNotApplicable,
		Note:  "In diesem Monat ist keine Voranmeldung abzugeben.",
	}
	if s.vat == nil {
		return step
	}
	periods, err := s.vat.Periods(ctx, periodYear)
	if err != nil {
		return step
	}
	var period *VatPeriodStatus
	for i := range periods {
		// Der Zeitraum, der mit diesem Monat endet — nicht der, in den der
		// Monat fällt: die Meldung ist am Ende des Zeitraums fällig.
		if periods[i].To == state.To {
			period = &periods[i]
			break
		}
	}
	if period == nil {
		return step
	}

	state.VatApplies = true
	state.VatPeriodKey = period.Key
	state.VatPeriodLabel = period.Label
	state.VatStatus = period.Status
	state.VatDueDate = period.DueDate
	state.VatSubmittedAt = period.SubmittedAt
	state.VatReturnID = period.ReturnID

	step.Title = fmt.Sprintf("Voranmeldung %s bestätigen", period.Label)
	switch {
	case period.Status == domain.VatReturnSubmitted:
		step.State = MonthStepDone
		step.Note = fmt.Sprintf("Am %s übermittelt.", period.SubmittedAt)
	case !state.Committed:
		// Das Kennziffernblatt ist vorher schon sichtbar — bestätigt werden
		// kann erst, was unveränderbar ist.
		step.State = MonthStepBlocked
		step.Note = "Das Kennziffernblatt steht bereit. Bestätigen lässt sich die Übermittlung erst nach der Festschreibung."
	default:
		step.State = MonthStepOpen
		step.Note = fmt.Sprintf(
			"Werte in Mein ELSTER eintragen, danach Datum und Transferticket erfassen. Fällig am %s.",
			period.DueDate)
	}
	return step
}

// requireActiveYear weist einen Monat ab, der nicht zum aktiven Geschäftsjahr
// gehört.
//
// Gefragt wird zuerst das Geschäftsjahr selbst: es kann vom Kalenderjahr
// abweichen, und dann liegt sein Januar im Folgejahr. Ist es nicht hinterlegt —
// in einer gewachsenen Datenbank der Normalfall —, gilt das Kalenderjahr.
func (s *MonthCloseService) requireActiveYear(ctx context.Context, month, label, from string) error {
	if s.years != nil {
		fy, err := s.years.FindByYear(ctx, s.fiscalYear)
		if err == nil && fy != nil && fy.StartDate != "" && fy.EndDate != "" {
			if from >= fy.StartDate && from <= fy.EndDate {
				return nil
			}
			return fmt.Errorf(
				"%s (%s) gehört nicht zum Geschäftsjahr %d (%s bis %s) — wechsle zuerst das "+
					"Geschäftsjahr, sonst stünde der Monat gegen die Bücher eines anderen Jahres",
				label, month, s.fiscalYear, fy.StartDate, fy.EndDate)
		}
	}
	if year := yearOfDate(from); year != s.fiscalYear {
		return fmt.Errorf(
			"%s (%s) gehört zum Geschäftsjahr %d — wechsle zuerst das Geschäftsjahr, sonst "+
				"stünde der Monat gegen die Bücher des Jahres %d",
			label, month, year, s.fiscalYear)
	}
	return nil
}

// yearOfDate liest das Jahr aus einem Datum "JJJJ-MM-TT" oder "JJJJ-MM".
func yearOfDate(date string) int {
	if len(date) < 4 {
		return 0
	}
	year := 0
	for i := 0; i < 4; i++ {
		if date[i] < '0' || date[i] > '9' {
			return 0
		}
		year = year*10 + int(date[i]-'0')
	}
	return year
}

// monthBounds zerlegt "JJJJ-MM" in seine Grenzen und seine Bezeichnung.
func monthBounds(month string) (from, to, label string, err error) {
	start, parseErr := time.Parse("2006-01", month)
	if parseErr != nil {
		return "", "", "", fmt.Errorf("%q ist kein Monat (erwartet JJJJ-MM)", month)
	}
	end := start.AddDate(0, 1, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02"),
		fmt.Sprintf("%s %d", germanMonthName(int(start.Month())), start.Year()), nil
}

// germanMonthName benennt den Monat. Er steht in der Überschrift des Dialogs,
// und „2026-03" ist kein Monatsname.
func germanMonthName(month int) string {
	names := [...]string{
		"Januar", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember",
	}
	if month < 1 || month > 12 {
		return ""
	}
	return names[month-1]
}
