package mcpserver

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Ziele der Aufgabenliste kommen aus der Oberfläche (Bereich, Reiter,
// Kennung); aufgaben nennt dafür das Werkzeug. Nur die Sicherung bleibt der App.
func TestTaskToolFollowsTaskTargets(t *testing.T) {
	for _, tc := range []struct {
		task  domain.Task
		tool  string
		param string
	}{
		{domain.Task{Key: domain.TaskKeyObligation + ".ustva.2026-Q1",
			Target: domain.TaskTarget{Area: domain.AreaClosings, Tab: "month", ID: "ustva.2026-Q1"}}, "fristen", "ustva.2026-Q1"},
		{domain.Task{Key: domain.TaskKeyObligation + ".gesellschafterliste.2026-06-01",
			Target: domain.TaskTarget{Area: domain.AreaCompany, Tab: "profile", ID: "gesellschafterliste.2026-06-01"}},
			"fristen", "gesellschafterliste.2026-06-01"},
		{domain.Task{Key: domain.TaskKeyReceiptsUnbooked, Target: domain.TaskTarget{Area: domain.AreaBooks, Tab: "receipts"}},
			"belege_suchen", "abgelegt"},
		{domain.Task{Key: domain.TaskKeyBankUnmatched, Target: domain.TaskTarget{Area: domain.AreaBooks, Tab: "bank"}},
			"bankumsaetze", "offen"},
		{domain.Task{Key: domain.TaskKeyOverdueItems, Target: domain.TaskTarget{Area: domain.AreaBooks, Tab: "invoices"}},
			"offene_posten", "forderung"},
		{domain.Task{Key: domain.TaskKeyCheckFindings + ".r1", Target: domain.TaskTarget{Area: domain.AreaSettings, Tab: "audit", ID: "r1"}},
			"auswertung", "pruefbericht"},
		{domain.Task{Key: domain.TaskKeyExemption, Target: domain.TaskTarget{Area: domain.AreaCompany, Tab: "contacts"}},
			"kontakte_suchen", ""},
		{domain.Task{Key: domain.TaskKeyCarryForward, Target: domain.TaskTarget{Area: domain.AreaClosings, Tab: "year"}},
			"abschluss_status", ""},
		{domain.Task{Key: domain.TaskKeyBackupMissing, Target: domain.TaskTarget{Area: domain.AreaSettings, Tab: "backup"}},
			"nur_app", ""},
	} {
		tool, params := taskTool(tc.task)
		if tool != tc.tool {
			t.Errorf("%s: Werkzeug %q, erwartet %q", tc.task.Key, tool, tc.tool)
			continue
		}
		if tc.param == "" {
			continue
		}
		found := false
		for _, v := range params {
			found = found || v == tc.param
		}
		if !found {
			t.Errorf("%s: Parameter %v ohne %q", tc.task.Key, params, tc.param)
		}
	}
}
