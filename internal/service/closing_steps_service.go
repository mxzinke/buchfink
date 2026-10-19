package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// ClosingStepsService führt den Abschlussassistenten.
//
// Der Zustand ist zur Hälfte abgeleitet und zur Hälfte gespeichert. Abgeleitet
// ist, was sich aus den Daten ergibt: eine AfA, die nicht mehr offen ist, ist
// gebucht; ein festgestellter Abschluss ist festgestellt. Gespeichert ist, was
// sich daraus nicht ergibt — vor allem das ausdrückliche Übergehen mit Grund.
// Ein Jahr ohne Rückstellungen sieht sonst aus wie ein Jahr, in dem sie
// vergessen wurden.
type ClosingStepsService struct {
	serviceClock
	derivers map[domain.ClosingStepKey]closingStepDeriver
	auditor[int]
	stepRepo          domain.ClosingStepRepository
	accrualRepo       domain.AccrualRepository
	provisionRepo     domain.ProvisionRepository
	appropriationRepo domain.AppropriationRepository
	checkRunRepo      domain.CheckRunRepository
	journalRepo       domain.JournalRepository
	closingSvc        *ClosingService
	depreciation      PendingDepreciationSource
	// Die beiden Bausteine, deren Zustand aus einer eigenen Kartei
	// folgt. Beide sind optional — ohne sie bleibt ihr Schritt schlicht offen.
	writeUps WriteUpSource
	inputTax InputTaxCorrectionSource
}

// NewClosingStepsService wires den Abschlussassistenten.
func NewClosingStepsService(
	stepRepo domain.ClosingStepRepository,
	accrualRepo domain.AccrualRepository,
	provisionRepo domain.ProvisionRepository,
	appropriationRepo domain.AppropriationRepository,
	checkRunRepo domain.CheckRunRepository,
	journalRepo domain.JournalRepository,
	auditRepo domain.AuditRepository,
	closingSvc *ClosingService,
) *ClosingStepsService {
	s := &ClosingStepsService{
		stepRepo: stepRepo, accrualRepo: accrualRepo, provisionRepo: provisionRepo,
		appropriationRepo: appropriationRepo,
		checkRunRepo:      checkRunRepo, journalRepo: journalRepo, auditor: auditor[int]{auditRepo: auditRepo, entity: "CLOSING_STEP"},
		closingSvc: closingSvc,
	}
	s.derivers = s.stepDerivers()
	return s
}

// SetDepreciationSource gibt dem Assistenten die Anlagenkartei, soweit er sie
// braucht: er will nur wissen, ob die AfA des Jahres noch offen ist.
func (s *ClosingStepsService) SetDepreciationSource(src PendingDepreciationSource) {
	s.depreciation = src
}

// WriteUpSource ist der Ausschnitt der Anlagenkartei, den der Baustein
// „Wertaufholung prüfen" braucht: wie viele Anlagegüter mit außerplanmäßiger
// Abschreibung im Jahr noch unbeantwortet sind.
type WriteUpSource interface {
	WriteUpReport(ctx context.Context, year int) (*WriteUpReport, error)
}

// InputTaxCorrectionSource ist der Ausschnitt des Verzeichnisses nach § 15a
// UStG, den der Abschluss braucht.
type InputTaxCorrectionSource interface {
	Year(ctx context.Context, year int) (*InputTaxCorrectionYear, error)
}

// SetWriteUpSource koppelt die Wertaufholung an die Schrittliste.
func (s *ClosingStepsService) SetWriteUpSource(src WriteUpSource) { s.writeUps = src }

// SetInputTaxSource koppelt die Vorsteuerberichtigung an die Schrittliste.
func (s *ClosingStepsService) SetInputTaxSource(src InputTaxCorrectionSource) { s.inputTax = src }

// ClosingStepView ist ein Baustein mit seinem Zustand.
type ClosingStepView struct {
	domain.ClosingStepDefinition
	State     domain.ClosingStepState `json:"state"`
	Reason    string                  `json:"reason,omitempty"`
	ChangedOn string                  `json:"changedOn,omitempty"`
	// Detail sagt in einem Halbsatz, woran der Zustand liegt: „3 Posten
	// gebildet", „AfA für 2 Anlagegüter offen".
	Detail string `json:"detail,omitempty"`
}

// ClosingSteps ist die Schrittliste eines Geschäftsjahres.
type ClosingSteps struct {
	FiscalYear int               `json:"fiscalYear"`
	Cutoff     string            `json:"cutoff"`
	Steps      []ClosingStepView `json:"steps"`
	// OpenCount ist die Zahl der Schritte, die weder erledigt noch übersprungen
	// sind — die Zahl, die auf der Seite steht.
	OpenCount int `json:"openCount"`
	// DoneCount, SkippedCount und Total sind der Fortschritt des geführten
	// Weges („Schritt 4 von 11", Architektur 6.3).
	//
	// Gerechnet und nicht von der Ansicht gezählt: der übersprungene Schritt
	// ist erledigt, aber nicht getan, und wer das in der Oberfläche noch einmal
	// entscheidet, entscheidet es beim nächsten Mal anders.
	DoneCount    int `json:"doneCount"`
	SkippedCount int `json:"skippedCount"`
	Total        int `json:"total"`
	// Reopenable sagt, ob eine Entscheidung des geführten Weges noch
	// zurückzunehmen ist — ein übersprungener Baustein also wieder offen werden
	// darf (Architektur 6.3: „Zurück ist möglich, solange nicht
	// festgeschrieben; danach nur Storno"). ReopenBlocker nennt den Grund,
	// wenn nicht; die Ansicht schreibt ihn an den gesperrten Knopf, statt ihn
	// zu verschweigen.
	Reopenable    bool   `json:"reopenable"`
	ReopenBlocker string `json:"reopenBlocker,omitempty"`
}

// Progress ist die Zahl der abgeschlossenen Schritte — erledigte und
// übersprungene zusammen. Sie ist der Zähler der Fortschrittsanzeige.
func (s *ClosingSteps) Progress() int { return s.DoneCount + s.SkippedCount }

// EnsureLists ersetzt eine nicht belegte Schrittliste durch eine leere.
func (s *ClosingSteps) EnsureLists() {
	if s.Steps == nil {
		s.Steps = make([]ClosingStepView, 0)
	}
}

// Steps stellt die Schrittliste eines Geschäftsjahres zusammen.
func (s *ClosingStepsService) Steps(ctx context.Context, year int) (*ClosingSteps, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	stored, err := s.stepRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	byKey := map[domain.ClosingStepKey]domain.ClosingStep{}
	for _, step := range stored {
		byKey[step.Key] = step
	}

	out := &ClosingSteps{FiscalYear: year, Cutoff: fy.EndDate, Steps: make([]ClosingStepView, 0, 11)}
	for _, def := range domain.AllClosingSteps() {
		view := ClosingStepView{ClosingStepDefinition: def, State: domain.ClosingStepOpen}
		if step, ok := byKey[def.Key]; ok {
			view.State = step.State
			view.Reason = step.Reason
			view.ChangedOn = step.ChangedOn
		}
		// Ein übersprungener Schritt bleibt übersprungen; sonst entscheiden die
		// Daten.
		//
		// Zwischen dem Haken des Anwenders und der Ableitung gilt: die
		// Ableitung gewinnt, sobald sie etwas gefunden hat. „Offen" heißt bei
		// ihr zweierlei — entweder sie hat einen Rest gefunden (AfA für zwei
		// Anlagegüter, ein blockierender Befund im Prüflauf) oder sie hat
		// nichts gefunden, woraus „erledigt" folgte. Nur im ersten Fall steht
		// dem Haken etwas entgegen, und dann verdeckt er einen Rest, der später
		// auffiel; im zweiten Fall ist der Haken die einzige Auskunft, die es
		// gibt — ein Jahr ohne Rückstellungen sähe sonst für immer unerledigt
		// aus. Der gefundene Rest steht im Detail, und daran ist er zu
		// erkennen.
		if view.State != domain.ClosingStepSkipped {
			state, detail := s.derivers[def.Key](ctx, year, fy)
			view.Detail = detail
			switch {
			case state == domain.ClosingStepDone, view.State == domain.ClosingStepOpen, detail != "":
				view.State = state
			default:
				view.Detail = "von Hand abgehakt"
				if view.ChangedOn != "" {
					view.Detail += " am " + view.ChangedOn
				}
			}
		}
		switch view.State {
		case domain.ClosingStepOpen:
			out.OpenCount++
		case domain.ClosingStepSkipped:
			out.SkippedCount++
		default:
			out.DoneCount++
		}
		out.Steps = append(out.Steps, view)
	}
	out.Total = len(out.Steps)
	out.Reopenable, out.ReopenBlocker = s.reopenState(ctx, year, fy)
	return out, nil
}

// reopenState beantwortet, ob eine Entscheidung des geführten Weges noch
// zurückzunehmen ist.
//
// Zwei Sperren, in der Reihenfolge, in der sie eintreten: die Jahres-
// Festschreibung macht die Buchungen des Jahres unveränderlich (§ 146 Abs. 4
// AO), und die Feststellung schließt den Abschluss ab (§ 42a Abs. 1 GmbHG).
// Danach führt der Weg zurück über die Rücksetzung der Feststellung
// beziehungsweise den Storno der Buchung — beides gibt es an anderer Stelle
// und mit Grund im Protokoll. Eine Entscheidung selbst nimmt danach niemand
// mehr direkt zurück.
func (s *ClosingStepsService) reopenState(
	ctx context.Context, year int, fy *domain.FiscalYear,
) (bool, string) {
	if fy != nil && fy.IsAdopted() {
		return false, fmt.Sprintf(
			"Der Abschluss %d ist festgestellt. Eine Entscheidung des Abschlussweges lässt sich "+
				"erst nach der Rücksetzung der Feststellung zurücknehmen.", year)
	}
	if s.closingSvc == nil {
		return true, ""
	}
	committed, _, err := s.closingSvc.HasYearCommitment(ctx, year)
	if err != nil {
		// Die Auskunft fehlt, also gilt die vorsichtige Antwort: lieber ein
		// gesperrter Knopf mit Grund als eine Rücknahme in einem Jahr, dessen
		// Festschreibung nicht gelesen werden konnte.
		return false, fmt.Sprintf(
			"Ob %d festgeschrieben ist, konnte nicht gelesen werden: %v", year, err)
	}
	if committed {
		return false, fmt.Sprintf(
			"Das Geschäftsjahr %d ist festgeschrieben. Was danach zu ändern ist, wird storniert "+
				"und neu gebucht.", year)
	}
	return true, ""
}

// liveAccruals sind die Abgrenzungsposten eines Jahres, deren Bildungsbuchung
// noch steht.
func (s *ClosingStepsService) liveAccruals(ctx context.Context, year int) ([]domain.Accrual, error) {
	accruals, err := s.accrualRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	return liveAccruals(ctx, s.journalRepo, accruals)
}

// liveProvisions sind die Rückstellungen eines Jahres ohne die Bewegungen, deren
// Buchung zurückgenommen wurde.
func (s *ClosingStepsService) liveProvisions(ctx context.Context, year int) ([]domain.Provision, error) {
	provisions, err := s.provisionRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	return liveProvisions(ctx, s.journalRepo, provisions)
}

// hasClosingReference meldet, ob im Jahr eine nicht stornierte Abschlussbuchung
// mit dieser Belegnummer steht.
//
// Zwei Buchungen haben die Belegnummer: die Verrechnung selbst und ihre
// Generalumkehr, denn der Storno übernimmt Quelle und Belegnummer der
// Ursprungsbuchung. Die Umkehr auszulassen genügt deshalb nicht — gefragt ist,
// ob die Ursprungsbuchung noch steht. Ohne diese zweite Frage bliebe der
// Schritt „Umsatzsteuer-Verrechnung" nach dem Storno auf „erledigt", obwohl die
// Steuerkonten wieder Salden tragen und die Verrechnung erneut zu buchen ist.
// Zurück kommt neben der Antwort, ob eine solche Buchung einmal bestand: ein
// Schritt, dessen Buchung storniert wurde, ist wieder offen und sagt das auch —
// sonst verdeckte ein von Hand gesetzter Haken den Storno.
func (s *ClosingStepsService) hasClosingReference(
	ctx context.Context, year int, reference string,
) (standing, reversed bool) {
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return false, false
	}
	index := newReversalIndex(s.journalRepo)
	for i := range entries {
		e := &entries[i]
		if e.Source != domain.EntrySourceClosing || e.DocumentNumber != reference ||
			e.Kind == domain.EntryKindReversal {
			continue
		}
		voided, err := index.reversed(ctx, &e.ID)
		if err != nil {
			return false, false
		}
		if !voided {
			return true, false
		}
		reversed = true
	}
	return false, reversed
}

// SetStep setzt einen Baustein auf erledigt oder übersprungen.
func (s *ClosingStepsService) SetStep(
	ctx context.Context, year int, key domain.ClosingStepKey, state domain.ClosingStepState, reason string,
) (*ClosingSteps, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	step := &domain.ClosingStep{
		Year: year, Key: key, State: state, Reason: reason,
		ChangedOn: s.today(),
	}
	if err := step.Validate(); err != nil {
		return nil, err
	}
	if err := s.stepRepo.Save(ctx, step); err != nil {
		return nil, err
	}
	def, _ := domain.ClosingStepDefinitionFor(key)
	if err := s.audit(ctx, domain.AuditActionUpdate, year, fmt.Sprintf("Abschlussschritt %q im Geschäftsjahr %d auf %q gesetzt: %s",
		def.Label, year, state, reason)); err != nil {
		return nil, err
	}
	return s.Steps(ctx, year)
}

// SkippedClosingStep ist ein übersprungener Baustein mit seinem Grund.
type SkippedClosingStep struct {
	Key       domain.ClosingStepKey
	Label     string
	Reason    string
	ChangedOn string
}

// SkippedSteps nennt die Bausteine, die ein Jahr ausdrücklich übergeht.
//
// Der Prüflauf liest sie und führt sie als Hinweis auf: das Übergehen ist die
// Auskunft, die sonst nirgends steht — an einem Jahr ohne Rückstellungen ist
// nicht zu sehen, ob niemand welche brauchte oder niemand hinsah.
func (s *ClosingStepsService) SkippedSteps(ctx context.Context, year int) ([]SkippedClosingStep, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	stored, err := s.stepRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	out := make([]SkippedClosingStep, 0, len(stored))
	for _, def := range domain.AllClosingSteps() {
		for _, step := range stored {
			if step.Key != def.Key || step.State != domain.ClosingStepSkipped {
				continue
			}
			out = append(out, SkippedClosingStep{
				Key: def.Key, Label: def.Label, Reason: step.Reason, ChangedOn: step.ChangedOn,
			})
		}
	}
	return out, nil
}

// SkipStep übergeht einen Baustein mit Grund.
func (s *ClosingStepsService) SkipStep(
	ctx context.Context, year int, key domain.ClosingStepKey, reason string,
) (*ClosingSteps, error) {
	if reason == "" {
		return nil, fmt.Errorf(
			"einen Abschlussschritt zu überspringen verlangt eine Begründung: sie steht später im " +
				"Prüfbericht und erklärt dem Prüfer, warum an dieser Stelle nichts gebucht wurde")
	}
	return s.SetStep(ctx, year, key, domain.ClosingStepSkipped, reason)
}

// ReopenStep nimmt eine Entscheidung des geführten Weges zurück: der
// übersprungene Baustein steht wieder offen.
//
// Der Weg zurück gehört zum geführten Weg (Architektur 6.3). Ohne ihn wäre das
// Überspringen die einzige Entscheidung des Programms, die niemand mehr
// korrigieren kann — und ein Grund wie „im ersten Jahr nicht einschlägig", der
// sich als falsch herausstellt, bliebe für immer im Prüfbericht stehen.
//
// Zwei Bedingungen: Solange das Jahr festgeschrieben oder festgestellt ist,
// bleibt es dabei (siehe reopenState). Und die Rücknahme verlangt selbst einen
// Grund — sie überschreibt eine begründete Entscheidung, und der Prüfbericht
// hätte sonst eine Lücke: erst stand dort ein Grund, dann nichts mehr.
//
// Zurückgenommen wird nur das Überspringen. Ein erledigter Baustein ist
// entweder abgeleitet (dann folgt er den Buchungen und wird durch den Storno
// wieder offen) oder von Hand abgehakt (dann ist das Abhaken jederzeit erneut
// möglich); beides braucht diese Tür nicht.
func (s *ClosingStepsService) ReopenStep(
	ctx context.Context, year int, key domain.ClosingStepKey, reason string,
) (*ClosingSteps, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf(
			"eine übersprungene Arbeit wieder aufzunehmen verlangt eine Begründung: sie tritt an " +
				"die Stelle des Grundes, mit dem der Schritt übergangen wurde")
	}
	def, ok := domain.ClosingStepDefinitionFor(key)
	if !ok {
		return nil, fmt.Errorf("unbekannter Abschlussschritt %q", key)
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	if allowed, blocker := s.reopenState(ctx, year, fy); !allowed {
		return nil, fmt.Errorf("%s", blocker)
	}

	stored, err := s.stepRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	previous := ""
	found := false
	for _, step := range stored {
		if step.Key != key {
			continue
		}
		found = step.State == domain.ClosingStepSkipped
		previous = step.Reason
	}
	if !found {
		return nil, fmt.Errorf(
			"der Baustein %q ist nicht übersprungen; zurückzunehmen ist nur eine Entscheidung, "+
				"die getroffen wurde", def.Label)
	}

	// Geschrieben wird hier und nicht über SetStep: der Protokolleintrag soll
	// beide Gründe nennen — den, mit dem übergangen wurde, und den, mit dem die
	// Arbeit wieder aufgenommen wird. Wer später prüft, liest sonst nur, dass
	// ein Zustand umsprang.
	step := &domain.ClosingStep{
		Year: year, Key: key, State: domain.ClosingStepOpen, ChangedOn: s.today(),
	}
	if err := step.Validate(); err != nil {
		return nil, err
	}
	if err := s.stepRepo.Save(ctx, step); err != nil {
		return nil, err
	}
	if err := s.audit(ctx, domain.AuditActionUpdate, year, fmt.Sprintf(
		"Abschlussschritt %q im Geschäftsjahr %d wieder aufgenommen: %s (übersprungen war er mit: %s)",
		def.Label, year, strings.TrimSpace(reason), previous)); err != nil {
		return nil, err
	}
	return s.Steps(ctx, year)
}
