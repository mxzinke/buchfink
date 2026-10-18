package service

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Monatsabschluss ------------------------------------------------------

// vatPeriodFor liest einen Zeitraumschlüssel („2026-03", „2026-Q1").
func vatPeriodFor(t *testing.T, key string) accounting.VatPeriod {
	t.Helper()
	period, err := accounting.ParseVatPeriodKey(key)
	if err != nil {
		t.Fatalf("Zeitraum %q: %v", key, err)
	}
	return period
}

type stubVatSource struct{ periods []VatPeriodStatus }

func (s stubVatSource) Periods(context.Context, int) ([]VatPeriodStatus, error) {
	return s.periods, nil
}

// Der Zustand eines Monats folgt aus Prüflauf, Festschreibung und Voranmeldung —
// und die Bestätigung der Voranmeldung ist gesperrt, solange nicht
// festgeschrieben ist (Architektur 6.2).
func TestMonthCloseStateFollowsTheThreeSteps(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	festschreibung := repository.NewFestschreibungRepository(env.db)

	periods := []VatPeriodStatus{{
		VatPeriod: vatPeriodFor(t, "2026-03"),
		DueDate:   "2026-04-10", Status: domain.VatReturnDraft,
	}}
	svc := NewMonthCloseService(
		stubCheckSource{run: &domain.CheckRun{Findings: []domain.CheckFinding{
			{Rule: domain.CheckRuleEntryWithoutReceipt, Severity: domain.CheckBlocking,
				Message: "Buchung ohne Beleg"},
		}}},
		stubVatSource{periods: periods}, festschreibung)

	state, err := svc.State(ctx, "2026-03")
	if err != nil {
		t.Fatalf("Monatsstand: %v", err)
	}
	if state.From != "2026-03-01" || state.To != "2026-03-31" {
		t.Errorf("Monatsgrenzen %s bis %s — erwartet 2026-03-01 bis 2026-03-31", state.From, state.To)
	}
	if state.Label != "März 2026" {
		t.Errorf("Bezeichnung = %q, erwartet \"März 2026\"", state.Label)
	}
	if len(state.Steps) != 3 {
		t.Fatalf("erwartet drei Schritte, erhalten %d", len(state.Steps))
	}
	if state.Blocking != 1 || state.Steps[0].State != MonthStepBlocked {
		t.Errorf("Schritt 1 = %q bei %d blockierenden Befunden — erwartet blocked",
			state.Steps[0].State, state.Blocking)
	}
	if state.Steps[1].State != MonthStepBlocked {
		t.Errorf("Schritt 2 = %q — erwartet blocked, solange Befunde offen sind", state.Steps[1].State)
	}
	if state.Steps[2].State != MonthStepBlocked {
		t.Errorf("Schritt 3 = %q — die Bestätigung setzt die Festschreibung voraus", state.Steps[2].State)
	}

	// Ohne Befund: Schritt 1 erledigt, Schritt 2 offen, Schritt 3 weiter gesperrt.
	clean := NewMonthCloseService(
		stubCheckSource{run: &domain.CheckRun{}}, stubVatSource{periods: periods},
		festschreibung)
	state, err = clean.State(ctx, "2026-03")
	if err != nil {
		t.Fatalf("Monatsstand: %v", err)
	}
	if state.Steps[0].State != MonthStepDone || state.Steps[1].State != MonthStepOpen {
		t.Errorf("ohne Befund erwartet done/open, erhalten %q/%q",
			state.Steps[0].State, state.Steps[1].State)
	}
	if state.Steps[2].State != MonthStepBlocked {
		t.Errorf("Schritt 3 = %q — erwartet blocked vor der Festschreibung", state.Steps[2].State)
	}

	// Nach der Festschreibung ist Schritt 3 offen — und nach der Übermittlung
	// erledigt.
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "month", PeriodLabel: "März 2026",
		CutoffDate: "2026-03-31", ChainHead: domain.GenesisHash, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	state, err = clean.State(ctx, "2026-03")
	if err != nil {
		t.Fatalf("Monatsstand: %v", err)
	}
	if !state.Committed || state.Steps[1].State != MonthStepDone {
		t.Errorf("nach der Festschreibung erwartet done, erhalten %q", state.Steps[1].State)
	}
	if state.Steps[2].State != MonthStepOpen {
		t.Errorf("Schritt 3 = %q — erwartet open nach der Festschreibung", state.Steps[2].State)
	}

	submitted := []VatPeriodStatus{{
		VatPeriod: vatPeriodFor(t, "2026-03"),
		DueDate:   "2026-04-10", Status: domain.VatReturnSubmitted, SubmittedAt: "2026-04-08",
	}}
	done := NewMonthCloseService(
		stubCheckSource{run: &domain.CheckRun{}}, stubVatSource{periods: submitted},
		festschreibung)
	state, err = done.State(ctx, "2026-03")
	if err != nil {
		t.Fatalf("Monatsstand: %v", err)
	}
	if state.Steps[2].State != MonthStepDone || state.VatSubmittedAt != "2026-04-08" {
		t.Errorf("Schritt 3 = %q (übermittelt am %q) — erwartet done",
			state.Steps[2].State, state.VatSubmittedAt)
	}
}

// Ein Monat, der kein Ende eines Voranmeldungszeitraums ist, endet nach
// Schritt 2.
func TestMonthWithoutVatPeriodEndsAfterTheCommitment(t *testing.T) {
	env := newTestEnv(t)
	svc := NewMonthCloseService(
		stubCheckSource{run: &domain.CheckRun{}},
		stubVatSource{periods: []VatPeriodStatus{{
			VatPeriod: vatPeriodFor(t, "2026-Q1"), DueDate: "2026-04-10",
		}}},
		repository.NewFestschreibungRepository(env.db))

	state, err := svc.State(context.Background(), "2026-01")
	if err != nil {
		t.Fatalf("Monatsstand: %v", err)
	}
	if state.VatApplies {
		t.Error("im Januar eines Quartalszahlers ist keine Voranmeldung fällig")
	}
	if state.Steps[2].State != MonthStepNotApplicable {
		t.Errorf("Schritt 3 = %q — erwartet not_applicable", state.Steps[2].State)
	}
}

// Ein unlesbarer Monat ist ein Fehler und keine leere Antwort.
func TestMonthCloseRefusesAnUnreadableMonth(t *testing.T) {
	env := newTestEnv(t)
	svc := NewMonthCloseService(nil, nil, repository.NewFestschreibungRepository(env.db))
	if _, err := svc.State(context.Background(), "März"); err == nil {
		t.Error("ein unlesbarer Monat muss abgewiesen werden")
	}
}

// --- Der Monat und sein Geschäftsjahr -------------------------------------

// stubYearVatSource merkt sich, für welches Jahr die Zeiträume abgefragt wurden.
type stubYearVatSource struct {
	periods   []VatPeriodStatus
	askedYear int
}

func (s *stubYearVatSource) Periods(_ context.Context, year int) ([]VatPeriodStatus, error) {
	s.askedYear = year
	return s.periods, nil
}

// stubFiscalYears beschreibt ein einzelnes Geschäftsjahr.
type stubFiscalYears struct{ year domain.FiscalYear }

func (s stubFiscalYears) FindByYear(_ context.Context, year int) (*domain.FiscalYear, error) {
	if s.year.Year == year {
		fy := s.year
		return &fy, nil
	}
	return nil, nil
}

// Ein Monat zählt zu dem Geschäftsjahr, in dem er liegt, gleich welches Jahr
// gerade angesehen wird.
//
// Der Fall ist der Regelfall am Jahreswechsel: die Voranmeldung für Dezember ist
// am 10. Januar fällig und steht als Frist in der Aufgabenliste. Festschreibung,
// Prüfbericht und Voranmeldung des Dezembers gehören zum alten Jahr.
func TestMonthCloseUsesTheFiscalYearOfTheMonth(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2025, PeriodType: "month", PeriodLabel: "Dezember 2025",
		CutoffDate: "2025-12-31", ChainHead: domain.GenesisHash, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	vat := &stubYearVatSource{periods: []VatPeriodStatus{{
		VatPeriod: vatPeriodFor(t, "2025-12"),
		DueDate:   "2026-01-10", Status: domain.VatReturnSubmitted, SubmittedAt: "2026-01-09",
	}}}
	checks := &recordingCheckSource{run: &domain.CheckRun{}}
	svc := NewMonthCloseService(checks, vat, festschreibung)

	state, err := svc.State(ctx, "2025-12")
	if err != nil {
		t.Fatalf("Stand Dezember 2025: %v", err)
	}
	if state.FiscalYear != 2025 || !state.Committed || state.VatStatus != domain.VatReturnSubmitted {
		t.Errorf("der Dezember 2025 steht nicht im Geschäftsjahr 2025: %+v", state)
	}
	if vat.askedYear != 2025 {
		t.Errorf("die Voranmeldungszeiträume wurden für %d abgefragt, erwartet 2025", vat.askedYear)
	}
	if checks.req.FiscalYear != 2025 {
		t.Errorf("der Prüfbericht lief für %d, erwartet 2025", checks.req.FiscalYear)
	}
}

// recordingCheckSource merkt sich den Auftrag des Prüflaufs.
type recordingCheckSource struct {
	run *domain.CheckRun
	req CheckRequest
}

func (s *recordingCheckSource) Preview(_ context.Context, req CheckRequest) (*domain.CheckRun, error) {
	s.req = req
	return s.run, nil
}

// Bei abweichendem Geschäftsjahr entscheidet dessen Spanne und nicht das
// Kalenderjahr — und die Voranmeldung folgt dem Kalenderjahr des Monats
// (§ 18 UStG kennt nur Kalenderzeiträume).
func TestMonthCloseFollowsAShiftedFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	years := stubFiscalYears{year: domain.FiscalYear{
		Year: 2026, StartDate: "2026-07-01", EndDate: "2027-06-30",
	}}
	vat := &stubYearVatSource{periods: []VatPeriodStatus{{
		VatPeriod: vatPeriodFor(t, "2027-01"), DueDate: "2027-02-10",
	}}}
	svc := NewMonthCloseService(
		stubCheckSource{run: &domain.CheckRun{}}, vat,
		repository.NewFestschreibungRepository(env.db))
	svc.SetFiscalYearSource(years)

	state, err := svc.State(ctx, "2027-01")
	if err != nil {
		t.Fatalf("der Januar 2027 gehört zum Geschäftsjahr 2026: %v", err)
	}
	if state.FiscalYear != 2026 {
		t.Errorf("Geschäftsjahr = %d, erwartet 2026", state.FiscalYear)
	}
	if vat.askedYear != 2027 {
		t.Errorf("die Voranmeldungszeiträume wurden für %d abgefragt, erwartet 2027", vat.askedYear)
	}
	if !state.VatApplies || state.Steps[2].State == MonthStepNotApplicable {
		t.Errorf("die Voranmeldung Januar 2027 fehlt im Stand: %+v", state.Steps[2])
	}

	// Der Januar 2026 trägt zwar dieselbe Jahreszahl wie das Geschäftsjahr,
	// liegt aber vor seinem Beginn und in keinem angelegten Jahr.
	if _, err := svc.State(ctx, "2026-01"); err == nil {
		t.Error("der Januar 2026 liegt in keinem angelegten Geschäftsjahr und darf keinen Stand liefern")
	}
}
