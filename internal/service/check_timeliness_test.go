package service

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// --- Abstände als Kennzahlen des Prüflaufs ---------------------------------

// Die Abstände Belegdatum → Erfassung → Festschreibung stehen als Kennzahlen am
// Prüflauf. Der Journalexport beantwortet die Frage je Buchung; über den
// Zeitraum beantwortet sie nur der Lauf.
func TestCheckRunReportsCaptureAndCommitmentIntervals(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Drei Buchungen mit festen Abständen: 2, 4 und 30 Tage zwischen
	// Belegdatum und Erfassung. Der Median ist damit 4, das Maximum 30, und
	// über der Frist von zehn Tagen liegt genau eine.
	created := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	for _, documentDate := range []string{"2026-03-29", "2026-03-27", "2026-03-01"} {
		entry := &domain.JournalEntry{
			FiscalYear: 2026, BookingDate: "2026-03-31", DocumentDate: documentDate,
			ServiceDateFrom: documentDate, ServiceDateTo: documentDate,
			Description: "Wareneingang", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
			Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
			CreatedAt: created,
			Lines: []domain.JournalLine{
				{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
				{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
			},
		}
		if err := env.journalRepo.Append(ctx, entry, accounting.NewHashChain().CalculateHash); err != nil {
			t.Fatalf("Buchung anlegen: %v", err)
		}
	}

	// Eine davon ist festgeschrieben, und zwar sieben Tage nach der Erfassung.
	committed := created.AddDate(0, 0, 7)
	if _, err := env.journalRepo.MarkCommitted(ctx, 2026, "2026-03-31", 0, committed); err != nil {
		t.Fatalf("Festschreibungsstempel: %v", err)
	}

	checks := env.checks(t)
	run, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-03-31"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}

	tl := run.Timeliness
	if tl.MeasuredEntries != 3 {
		t.Fatalf("gemessene Buchungen %d, erwartet 3", tl.MeasuredEntries)
	}
	if tl.CaptureDaysMedian != 4 {
		t.Errorf("Median Belegdatum → Erfassung %d, erwartet 4", tl.CaptureDaysMedian)
	}
	if tl.CaptureDaysMax != 30 {
		t.Errorf("Maximum Belegdatum → Erfassung %d, erwartet 30", tl.CaptureDaysMax)
	}
	if tl.CaptureLimitDays != 10 || tl.LateEntries != 1 {
		t.Errorf("über der Frist von %d Tagen liegen %d Buchungen, erwartet 1 bei 10 Tagen",
			tl.CaptureLimitDays, tl.LateEntries)
	}
	if tl.CommittedEntries != 3 || tl.UncommittedEntries != 0 {
		t.Errorf("festgeschrieben %d, offen %d, erwartet 3 und 0",
			tl.CommittedEntries, tl.UncommittedEntries)
	}
	if tl.CommitDaysMedian != 7 || tl.CommitDaysMax != 7 {
		t.Errorf("Abstand Erfassung → Festschreibung %d/%d Tage, erwartet 7/7",
			tl.CommitDaysMedian, tl.CommitDaysMax)
	}
}

// Die Kennzahlen werden mit dem Lauf abgelegt und wieder gelesen: sie stehen als
// Spalten am Prüflauf, nicht nur im Speicher des Aufrufs.
func TestCheckRunTimelinessSurvivesTheRepository(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	checks := env.checks(t)
	if _, err := checks.Run(ctx, CheckRequest{CutoffDate: "2026-03-31"}); err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	runs, err := checks.Runs(ctx, 2026)
	if err != nil || len(runs) == 0 {
		t.Fatalf("Läufe lesen: %v (%d Läufe)", err, len(runs))
	}
	if runs[0].Timeliness.CaptureLimitDays != 10 {
		t.Errorf("die Erfassungsfrist überlebt die Ablage nicht: %+v", runs[0].Timeliness)
	}
}

// Mit Dauerfristverlängerung und Nachfrist meldet der Prüflauf einen Monat
// nicht vor dem Tag, an dem der Pflichtenkalender ihn führt.
//
// Sonst widersprächen sich beide: period_not_committed meldete den Januar ab
// dem 31. März, während CommitDueDate ihn erst zum 10. April fällig stellt —
// und commit_overdue, das zwischen beiden Tagen liegen soll, käme nie zum Zug.
func TestNotCommittedFollowsTheCommitDeadline(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.PermanentExtension = true
		c.CommitGraceDays = 10
	})

	// Die Frist des Januars: Ende März plus zehn Tage Nachfrist.
	cfg := companySettingsFor(t, env)
	january, err := accounting.ParseVatPeriodKey("2026-01")
	if err != nil {
		t.Fatalf("Zeitraum: %v", err)
	}
	due := CommitDueDate(january, cfg)
	if due != "2026-04-10" {
		t.Fatalf("Frist %s — erwartet den 10. April", due)
	}

	run, err := env.checksOn(t, "2026-04-05").Preview(ctx, CheckRequest{CutoffDate: "2026-04-05"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	for _, f := range findingsFor(run, domain.CheckRulePeriodNotCommitted) {
		if f.ObjectID == "2026-01" {
			t.Error("vor der Frist ist der Januar kein Rückstand")
		}
	}

	run, err = env.checksOn(t, "2026-04-11").Preview(ctx, CheckRequest{CutoffDate: "2026-04-11"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := false
	for _, f := range findingsFor(run, domain.CheckRulePeriodNotCommitted) {
		if f.ObjectID == "2026-01" {
			found = true
		}
	}
	if !found {
		t.Error("nach der Frist gehört der Januar in den Rückstand")
	}
}

// Der Prüflauf meldet einen Monat, der zwei Monate nach seinem Ende nicht
// festgeschrieben ist (UNV-02 K2).
func TestPeriodNotCommittedIsReportedAfterTwoMonths(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Ende März: der Januar ist überfällig, aber die Zweimonatsgrenze ist noch
	// nicht überschritten.
	run, err := env.checksOn(t, "2026-03-31").Preview(ctx, CheckRequest{CutoffDate: "2026-03-31"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	if len(findingsFor(run, domain.CheckRulePeriodNotCommitted)) != 0 {
		t.Error("am 31. März sind seit dem Ende des Januars erst zwei Monate vergangen")
	}

	run, err = env.checksOn(t, "2026-04-01").Preview(ctx, CheckRequest{CutoffDate: "2026-04-01"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := findingsFor(run, domain.CheckRulePeriodNotCommitted)
	if len(found) == 0 {
		t.Fatal("am 1. April ist der Januar zwei Monate nach seinem Ende nicht festgeschrieben")
	}
	if found[0].ObjectID != "2026-01" {
		t.Errorf("gemeldeter Zeitraum %q — erwartet den Januar", found[0].ObjectID)
	}
	if found[0].Severity != domain.CheckWarning {
		t.Errorf("Gewicht %q — erwartet einen Hinweis", found[0].Severity)
	}

	// Nach der Festschreibung ist die Meldung weg.
	env.commitUntil(t, "2026-01-31")
	run, err = env.checksOn(t, "2026-04-01").Preview(ctx, CheckRequest{CutoffDate: "2026-04-01"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	for _, f := range findingsFor(run, domain.CheckRulePeriodNotCommitted) {
		if f.ObjectID == "2026-01" {
			t.Error("der festgeschriebene Januar darf nicht mehr gemeldet werden")
		}
	}
}

// Die Frist zur Festschreibung folgt der Dauerfristverlängerung: wer einen
// Monat später anmelden darf, schreibt einen Monat später fest.
func TestCommitDeadlineFollowsThePermanentExtension(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	before, err := env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, ok := deadlineByKey(before, "festschreibung.2026-01")
	if !ok {
		t.Fatal("die Festschreibung des Januars fehlt")
	}
	if january.DueDate != "2026-02-28" {
		t.Fatalf("ohne Dauerfrist fällig am %s, erwartet 2026-02-28", january.DueDate)
	}

	env.setSettings(t, func(c *domain.CompanySettings) { c.PermanentExtension = true })
	after, err := env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, _ = deadlineByKey(after, "festschreibung.2026-01")
	if january.DueDate != "2026-03-31" {
		t.Errorf("mit Dauerfristverlängerung fällig am %s, erwartet 2026-03-31", january.DueDate)
	}
}
