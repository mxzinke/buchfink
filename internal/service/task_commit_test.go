package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// „Monat festschreiben" steht ab dem 10. des Folgemonats als offene Aufgabe und
// ab dem Tag nach der Frist als überfällige (UNV-02 K2, Entscheidung 5).
func TestCommitTaskIsOpenFromTheTenthOfTheNextMonth(t *testing.T) {
	env := newTestEnv(t)

	// Die Frist der Januarfestschreibung mit Dauerfristverlängerung: Ende März.
	// Sie liegt damit weiter als der Vorlauf von dreißig Tagen entfernt, und
	// die Aufgabe muss trotzdem erscheinen.
	deadline := domain.Obligation{
		Key: string(domain.ObligationMonthCommit) + ".2026-01", Kind: domain.ObligationMonthCommit,
		Title: "Festschreibung Januar 2026", DueDate: "2026-03-31", Basis: "GoBD Rz. 107",
		PeriodFrom: "2026-01-01", PeriodTo: "2026-01-31", Status: domain.ObligationOpen,
		Description: "Nach der Festschreibung nimmt der Monat keine Buchung mehr auf.",
	}

	groupOf := func(today string) (domain.TaskGroup, bool) {
		t.Helper()
		svc := NewTaskService(repository.NewSettingsRepository(env.db))
		svc.SetObligationSource(stubObligationSource{obligations: []domain.Obligation{deadline}})
		list, err := svc.Tasks(context.Background(), TaskOptions{Today: today})
		if err != nil {
			t.Fatalf("Aufgabenliste zum %s: %v", today, err)
		}
		for _, task := range list.Overdue {
			if task.Title == deadline.Title {
				return domain.TaskGroupOverdue, true
			}
		}
		for _, task := range list.Open {
			if task.Title == deadline.Title {
				return domain.TaskGroupOpen, true
			}
		}
		for _, task := range list.Upcoming {
			if task.Title == deadline.Title {
				return domain.TaskGroupUpcoming, true
			}
		}
		return "", false
	}

	// Am 9. Februar ist die Voranmeldung des Januars noch nicht abzugeben: die
	// Festschreibung ist noch nicht dran und keine offene Aufgabe.
	if group, ok := groupOf("2026-02-09"); ok && group == domain.TaskGroupOpen {
		t.Error("vor dem 10. des Folgemonats ist die Festschreibung noch nicht offen")
	}
	if group, ok := groupOf("2026-02-10"); !ok || group != domain.TaskGroupOpen {
		t.Errorf("am 10. des Folgemonats gehört die Festschreibung des Januars in die offenen "+
			"Aufgaben, erhalten %q (gefunden: %v)", group, ok)
	}
	if group, ok := groupOf("2026-04-01"); !ok || group != domain.TaskGroupOverdue {
		t.Errorf("nach der Frist ist die Festschreibung überfällig, erhalten %q (gefunden: %v)",
			group, ok)
	}
}

// Die Festschreibung eines Monats steht einmal in der Aufgabenliste: als
// Fristzeile mit ihrem Monat und nicht zusätzlich als Regelzeile des Prüflaufs.
func TestCommitTaskAppearsOnlyOnce(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetObligationSource(stubObligationSource{obligations: []domain.Obligation{{
		Key: string(domain.ObligationMonthCommit) + ".2026-01", Kind: domain.ObligationMonthCommit,
		Title: "Festschreibung Januar 2026", DueDate: "2026-02-28", Basis: "GoBD Rz. 107",
		PeriodFrom: "2026-01-01", PeriodTo: "2026-01-31", Status: domain.ObligationOverdue,
		Description: "Nach der Festschreibung nimmt der Monat keine Buchung mehr auf.",
	}}})
	svc.SetCheckSource(stubCheckSource{run: &domain.CheckRun{Findings: []domain.CheckFinding{{
		Rule: domain.CheckRulePeriodNotCommitted, Severity: domain.CheckWarning,
		ObjectID: "2026-01", ObjectName: "Januar 2026",
		Message: "Januar 2026 ist seit dem 2026-03-31 nicht festgeschrieben",
	}}}})

	list, err := svc.Tasks(context.Background(), TaskOptions{Today: "2026-04-01"})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	commit := 0
	for _, group := range [][]domain.Task{list.Overdue, list.Open, list.Upcoming} {
		for _, task := range group {
			if strings.Contains(task.Title, "Festschreibung") || strings.Contains(task.Title, "festschreiben") {
				commit++
			}
		}
	}
	if commit != 1 {
		t.Errorf("%d Zeilen zur Festschreibung des Januars — dieselbe Arbeit steht einmal in der "+
			"Liste: %+v", commit, list)
	}
}

// Der Befund über den nicht festgeschriebenen Monat wird zur Aufgabe „Monat
// festschreiben" (UNV-02 K2).
func TestNotCommittedPeriodBecomesATask(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetCheckSource(stubCheckSource{run: &domain.CheckRun{Findings: []domain.CheckFinding{{
		Rule: domain.CheckRulePeriodNotCommitted, Severity: domain.CheckWarning,
		ObjectID: "2026-01", ObjectName: "Januar 2026",
		Message: "Januar 2026 ist seit dem 2026-03-31 nicht festgeschrieben",
	}}}})

	list, err := svc.Tasks(context.Background(), TaskOptions{Today: "2026-04-01"})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	found := false
	for _, group := range [][]domain.Task{list.Overdue, list.Open, list.Upcoming} {
		for _, task := range group {
			if task.Title == "Monat festschreiben" {
				found = true
				if task.Reference == "" {
					t.Error("der Aufgabe fehlt ihre Fundstelle")
				}
			}
		}
	}
	if !found {
		t.Errorf("die Aufgabe „Monat festschreiben\" fehlt: %+v", list)
	}
}
