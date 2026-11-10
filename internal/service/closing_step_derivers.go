package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

type closingStepDeriver func(context.Context, int, *domain.FiscalYear) (domain.ClosingStepState, string)

func (s *ClosingStepsService) stepDerivers() map[domain.ClosingStepKey]closingStepDeriver {
	return map[domain.ClosingStepKey]closingStepDeriver{
		domain.ClosingStepDepreciation:        s.deriveDepreciation,
		domain.ClosingStepWriteUp:             s.deriveWriteUp,
		domain.ClosingStepCurrencyValuation:   s.deriveCurrencyValuation,
		domain.ClosingStepReceivableAllowance: s.deriveReceivableAllowance,
		domain.ClosingStepWorkInProgress:      s.deriveWorkInProgress,
		domain.ClosingStepInputTaxCorrection:  s.deriveInputTaxCorrection,
		domain.ClosingStepAccruals:            s.deriveAccruals,
		domain.ClosingStepProvisions:          s.deriveProvisions,
		domain.ClosingStepVatSettlement:       s.deriveVatSettlement,
		domain.ClosingStepTaxProvision:        s.deriveTaxProvision,
		domain.ClosingStepCheckRun:            s.deriveCheckRun,
		domain.ClosingStepStatement:           s.deriveStatement,
		domain.ClosingStepAdoption:            s.deriveAdoption,
		domain.ClosingStepDisclosure:          s.deriveDisclosure,
		domain.ClosingStepAppropriation:       s.deriveAppropriation,
	}
}

func (s *ClosingStepsService) deriveDepreciation(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
}

func (s *ClosingStepsService) deriveWriteUp(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
}

func (s *ClosingStepsService) deriveCurrencyValuation(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
}

func (s *ClosingStepsService) deriveReceivableAllowance(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	standing, reversed := s.hasClosingReference(ctx, year, ReceivableAllowanceReference(year))
	if standing {
		return domain.ClosingStepDone, "Wertberichtigung gebucht"
	}
	if reversed {
		return domain.ClosingStepOpen, "die Buchung der Wertberichtigung wurde storniert"
	}
	return domain.ClosingStepOpen, ""
}

func (s *ClosingStepsService) deriveWorkInProgress(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	standing, reversed := s.hasClosingReference(ctx, year, WorkInProgressReference(year))
	if standing {
		return domain.ClosingStepDone, "unfertige Leistungen aktiviert"
	}
	if reversed {
		return domain.ClosingStepOpen, "die Buchung der unfertigen Leistungen wurde storniert"
	}
	return domain.ClosingStepOpen, ""
}

func (s *ClosingStepsService) deriveInputTaxCorrection(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
}

func (s *ClosingStepsService) deriveAccruals(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	// Gezählt wird, was noch steht: ein Posten, dessen Bildungsbuchung per
	// Generalumkehr zurückgenommen wurde, ist nicht gebildet, und der
	// Schritt wäre sonst durch eine Buchung erledigt, die es nicht mehr
	// gibt.
	accruals, err := s.liveAccruals(ctx, year)
	if err != nil || len(accruals) == 0 {
		return domain.ClosingStepOpen, ""
	}
	return domain.ClosingStepDone, fmt.Sprintf("%d Abgrenzungsposten gebildet", len(accruals))
}

func (s *ClosingStepsService) deriveProvisions(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
}

func (s *ClosingStepsService) deriveVatSettlement(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	standing, reversed := s.hasClosingReference(ctx, year, VatSettlementReference(year))
	if standing {
		return domain.ClosingStepDone, "Steuerkonten verrechnet"
	}
	if reversed {
		return domain.ClosingStepOpen, "die Verrechnungsbuchung wurde storniert"
	}
	return domain.ClosingStepOpen, ""
}

func (s *ClosingStepsService) deriveTaxProvision(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	// Ein Jahr mit Überzahlung bucht nur Erstattungsansprüche und keine
	// Rückstellung; erledigt ist der Schritt mit der Buchung.
	if standing, _ := s.hasClosingReference(ctx, year, TaxProvisionReference(year)); standing {
		return domain.ClosingStepDone, "Steuerrückstellung und Erstattungsansprüche gebucht"
	}
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
}

func (s *ClosingStepsService) deriveCheckRun(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
}

func (s *ClosingStepsService) deriveStatement(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	if fy.IsAdopted() {
		return domain.ClosingStepDone, "festgestellt am " + fy.AdoptedOn
	}
	if fy.Status == domain.FiscalYearPrepared {
		return domain.ClosingStepDone, "aufgestellt am " + fy.PreparedOn
	}
	return domain.ClosingStepOpen, ""
}

func (s *ClosingStepsService) deriveAdoption(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	if fy.IsAdopted() {
		return domain.ClosingStepDone, "festgestellt am " + fy.AdoptedOn
	}
	return domain.ClosingStepOpen, ""
}

func (s *ClosingStepsService) deriveDisclosure(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
	if fy.Status == domain.FiscalYearDisclosed {
		return domain.ClosingStepDone, "offengelegt am " + fy.DisclosedOn
	}
	return domain.ClosingStepOpen, ""
}

func (s *ClosingStepsService) deriveAppropriation(ctx context.Context, year int, fy *domain.FiscalYear) (domain.ClosingStepState, string) {
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
