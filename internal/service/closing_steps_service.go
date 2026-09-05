package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// ClosingStepsService führt den Abschlussassistenten.
//
// Der Jahresabschluss besteht aus elf Arbeiten in fester Reihenfolge, und keine
// von ihnen ist neu — sie lagen bisher nur über die Oberfläche verstreut. Der
// Assistent fügt nichts hinzu, was nicht ohnehin zu tun wäre; er sagt, was
// getan ist und was nicht.
//
// Der Zustand ist zur Hälfte abgeleitet und zur Hälfte gespeichert. Abgeleitet
// ist, was sich aus den Daten ergibt: eine AfA, die nicht mehr offen ist, ist
// gebucht; ein festgestellter Abschluss ist festgestellt. Gespeichert ist, was
// sich daraus nicht ergibt — vor allem das ausdrückliche Übergehen mit Grund.
// Ein Jahr ohne Rückstellungen sieht sonst aus wie ein Jahr, in dem sie
// vergessen wurden.
type ClosingStepsService struct {
	stepRepo          domain.ClosingStepRepository
	accrualRepo       domain.AccrualRepository
	provisionRepo     domain.ProvisionRepository
	inventoryRepo     domain.InventoryRepository
	appropriationRepo domain.AppropriationRepository
	checkRunRepo      domain.CheckRunRepository
	journalRepo       domain.JournalRepository
	auditRepo         domain.AuditRepository
	closingSvc        *ClosingService
	depreciation      PendingDepreciationSource
	fiscalYear        int
	// Welle 5c: die beiden Bausteine, deren Zustand aus einer eigenen Kartei
	// folgt. Beide sind optional — ohne sie bleibt ihr Schritt schlicht offen.
	writeUps WriteUpSource
	inputTax InputTaxCorrectionSource
}

// NewClosingStepsService wires den Abschlussassistenten.
func NewClosingStepsService(
	stepRepo domain.ClosingStepRepository,
	accrualRepo domain.AccrualRepository,
	provisionRepo domain.ProvisionRepository,
	inventoryRepo domain.InventoryRepository,
	appropriationRepo domain.AppropriationRepository,
	checkRunRepo domain.CheckRunRepository,
	journalRepo domain.JournalRepository,
	auditRepo domain.AuditRepository,
	closingSvc *ClosingService,
	fiscalYear int,
) *ClosingStepsService {
	return &ClosingStepsService{
		stepRepo: stepRepo, accrualRepo: accrualRepo, provisionRepo: provisionRepo,
		inventoryRepo: inventoryRepo, appropriationRepo: appropriationRepo,
		checkRunRepo: checkRunRepo, journalRepo: journalRepo, auditRepo: auditRepo,
		closingSvc: closingSvc, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *ClosingStepsService) SetFiscalYear(year int) { s.fiscalYear = year }

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
}

// Steps stellt die Schrittliste eines Geschäftsjahres zusammen.
func (s *ClosingStepsService) Steps(ctx context.Context, year int) (*ClosingSteps, error) {
	if year == 0 {
		year = s.fiscalYear
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
			state, detail := s.derive(ctx, def.Key, year, fy)
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
		if view.State == domain.ClosingStepOpen {
			out.OpenCount++
		}
		out.Steps = append(out.Steps, view)
	}
	return out, nil
}

// derive liest den Zustand eines Bausteins aus den Daten.
func (s *ClosingStepsService) derive(
	ctx context.Context, key domain.ClosingStepKey, year int, fy *domain.FiscalYear,
) (domain.ClosingStepState, string) {
	switch key {
	case domain.ClosingStepDepreciation:
		if s.depreciation == nil {
			return domain.ClosingStepOpen, ""
		}
		due, err := s.depreciation.PendingDepreciation(ctx, year)
		if err != nil {
			return domain.ClosingStepOpen, ""
		}
		if len(due) == 0 {
			return domain.ClosingStepDone, "keine Abschreibung offen"
		}
		return domain.ClosingStepOpen, fmt.Sprintf("AfA für %d Anlagegüter offen", len(due))

	case domain.ClosingStepWriteUp:
		if s.writeUps == nil {
			return domain.ClosingStepOpen, ""
		}
		report, err := s.writeUps.WriteUpReport(ctx, year)
		if err != nil {
			return domain.ClosingStepOpen, ""
		}
		if len(report.Candidates) == 0 {
			return domain.ClosingStepDone, "kein Anlagegut mit außerplanmäßiger Abschreibung"
		}
		if report.Open == 0 {
			return domain.ClosingStepDone, fmt.Sprintf(
				"%d Anlagegüter geprüft", len(report.Candidates))
		}
		return domain.ClosingStepOpen, fmt.Sprintf(
			"%d von %d Anlagegütern noch nicht beantwortet", report.Open, len(report.Candidates))

	case domain.ClosingStepCurrencyValuation:
		// Die Bewertung wird an ihrer Belegnummer erkannt und nicht an einer
		// eigenen Kartei: sie hinterlässt genau zwei Buchungen, und ob sie
		// stehen, weiß das Journal.
		standing, reversed := s.hasClosingReference(ctx, year, foreignCurrencyDocument(year))
		if standing {
			return domain.ClosingStepDone, "Fremdwährungsposten bewertet"
		}
		if reversed {
			return domain.ClosingStepOpen, "die Bewertungsbuchung wurde storniert"
		}
		return domain.ClosingStepOpen, ""

	case domain.ClosingStepInputTaxCorrection:
		standing, reversed := s.hasClosingReference(ctx, year, inputTaxCorrectionDocument(year))
		if standing {
			return domain.ClosingStepDone, "Vorsteuerberichtigung gebucht"
		}
		if reversed {
			return domain.ClosingStepOpen, "die Berichtigungsbuchung wurde storniert"
		}
		if s.inputTax == nil {
			return domain.ClosingStepOpen, ""
		}
		view, err := s.inputTax.Year(ctx, year)
		if err != nil {
			return domain.ClosingStepOpen, ""
		}
		pending := 0
		for _, row := range view.Rows {
			if row.InPeriod && row.Assessment.Required && !row.Booked {
				pending++
			}
		}
		switch {
		case view.Unconfirmed > 0:
			return domain.ClosingStepOpen, fmt.Sprintf(
				"%d Verwendungsanteile noch nicht bestätigt", view.Unconfirmed)
		case pending == 0:
			return domain.ClosingStepDone, "nichts zu berichtigen"
		default:
			return domain.ClosingStepOpen, fmt.Sprintf("%d Berichtigungen offen", pending)
		}

	case domain.ClosingStepAccruals:
		// Gezählt wird, was noch steht: ein Posten, dessen Bildungsbuchung per
		// Generalumkehr zurückgenommen wurde, ist nicht gebildet, und der
		// Schritt wäre sonst durch eine Buchung erledigt, die es nicht mehr
		// gibt.
		accruals, err := s.liveAccruals(ctx, year)
		if err != nil || len(accruals) == 0 {
			return domain.ClosingStepOpen, ""
		}
		return domain.ClosingStepDone, fmt.Sprintf("%d Abgrenzungsposten gebildet", len(accruals))

	case domain.ClosingStepProvisions:
		provisions, err := s.liveProvisions(ctx, year)
		if err != nil {
			return domain.ClosingStepOpen, ""
		}
		// Die Steuerrückstellung ist ein eigener Schritt. Zählte sie hier mit,
		// wäre der Schritt „Rückstellungen" allein durch sie erledigt — obwohl
		// niemand nach ungewissen Verbindlichkeiten gesehen hat.
		count := 0
		for _, p := range provisions {
			if !p.Kind.IsTax() {
				count++
			}
		}
		if count == 0 {
			return domain.ClosingStepOpen, ""
		}
		return domain.ClosingStepDone, fmt.Sprintf("%d Rückstellungen erfasst", count)

	case domain.ClosingStepInventory:
		counts, err := s.inventoryRepo.FindByYear(ctx, year)
		if err != nil || len(counts) == 0 {
			return domain.ClosingStepOpen, ""
		}
		open := 0
		for i := range counts {
			voided, err := entryIsReversed(ctx, s.journalRepo, counts[i].JournalEntryID)
			if err != nil {
				return domain.ClosingStepOpen, ""
			}
			if !voided {
				open++
			}
		}
		if open == 0 {
			return domain.ClosingStepOpen, ""
		}
		return domain.ClosingStepDone, fmt.Sprintf("%d Vorratskonten aufgenommen", open)

	case domain.ClosingStepVatSettlement:
		standing, reversed := s.hasClosingReference(ctx, year, VatSettlementReference(year))
		if standing {
			return domain.ClosingStepDone, "Steuerkonten verrechnet"
		}
		if reversed {
			return domain.ClosingStepOpen, "die Verrechnungsbuchung wurde storniert"
		}
		return domain.ClosingStepOpen, ""

	case domain.ClosingStepTaxProvision:
		provisions, err := s.liveProvisions(ctx, year)
		if err != nil {
			return domain.ClosingStepOpen, ""
		}
		for _, p := range provisions {
			if p.Kind.IsTax() {
				return domain.ClosingStepDone, "Steuerrückstellung gebildet"
			}
		}
		return domain.ClosingStepOpen, ""

	case domain.ClosingStepCheckRun:
		if s.checkRunRepo == nil {
			return domain.ClosingStepOpen, ""
		}
		run, err := s.checkRunRepo.Latest(ctx, year)
		if err != nil || run == nil {
			return domain.ClosingStepOpen, ""
		}
		// Ein Prüflauf mit blockierendem Befund ist gelaufen, aber nicht
		// erledigt: der Befund steht der Festschreibung im Weg, und ein Haken
		// an dieser Stelle führte den Anwender an ihm vorbei.
		if blocking := len(run.Blocking()); blocking > 0 {
			return domain.ClosingStepOpen, fmt.Sprintf(
				"Prüflauf vom %s: %d blockierende Befunde", run.CutoffDate, blocking)
		}
		return domain.ClosingStepDone, fmt.Sprintf(
			"Prüflauf vom %s ohne blockierenden Befund", run.CutoffDate)

	case domain.ClosingStepStatement, domain.ClosingStepAdoption:
		if fy.IsAdopted() {
			return domain.ClosingStepDone, "festgestellt am " + fy.AdoptedOn
		}
		if fy.Status == domain.FiscalYearPrepared && key == domain.ClosingStepStatement {
			return domain.ClosingStepDone, "aufgestellt am " + fy.PreparedOn
		}
		return domain.ClosingStepOpen, ""

	case domain.ClosingStepDisclosure:
		if fy.Status == domain.FiscalYearDisclosed {
			return domain.ClosingStepDone, "offengelegt am " + fy.DisclosedOn
		}
		return domain.ClosingStepOpen, ""

	case domain.ClosingStepAppropriation:
		// Der Schritt hat zwei Hälften: den Saldenvortrag ins Folgejahr und den
		// Beschluss über das Ergebnis. Erledigt ist er erst mit beiden — ein
		// Beschluss ohne Vortrag hätte kein Vortragskonto, aus dem er verteilt.
		carried := false
		if preview, err := s.closingSvc.CarryForwardState(ctx, year+1); err == nil && preview != nil {
			carried = preview.AlreadyCarried
		}
		appropriation, err := s.appropriationRepo.FindByYear(ctx, year)
		decided := err == nil && appropriation != nil
		if decided {
			// Ein Beschluss, dessen Buchung storniert wurde, ist wieder offen.
			voided, err := entryIsReversed(ctx, s.journalRepo, appropriation.JournalEntryID)
			decided = err == nil && !voided
		}
		switch {
		case carried && decided:
			return domain.ClosingStepDone, fmt.Sprintf(
				"vorgetragen nach %d, beschlossen am %s", year+1, appropriation.DecisionDate)
		case decided:
			return domain.ClosingStepOpen, fmt.Sprintf(
				"beschlossen am %s; der Saldenvortrag nach %d fehlt noch",
				appropriation.DecisionDate, year+1)
		case carried:
			return domain.ClosingStepOpen, fmt.Sprintf(
				"nach %d vorgetragen; der Beschluss über das Ergebnis fehlt noch", year+1)
		}
		return domain.ClosingStepOpen, ""
	}
	return domain.ClosingStepOpen, ""
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
// Zwei Buchungen tragen die Belegnummer: die Verrechnung selbst und ihre
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
	if year == 0 {
		year = s.fiscalYear
	}
	step := &domain.ClosingStep{
		Year: year, Key: key, State: state, Reason: reason,
		ChangedOn: time.Now().Format("2006-01-02"),
	}
	if err := step.Validate(); err != nil {
		return nil, err
	}
	if err := s.stepRepo.Save(ctx, step); err != nil {
		return nil, err
	}
	def, _ := domain.ClosingStepDefinitionFor(key)
	s.audit(ctx, year, fmt.Sprintf("Abschlussschritt %q im Geschäftsjahr %d auf %q gesetzt: %s",
		def.Label, year, state, reason))
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
	if year == 0 {
		year = s.fiscalYear
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

func (s *ClosingStepsService) audit(ctx context.Context, year int, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "CLOSING_STEP", fmt.Sprintf("%d", year), details)
}
