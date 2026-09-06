package accounting

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kein `null` in den Katalogen und Plänen, die an die Oberfläche gehen.
//
// Die Masken lesen diese Listen ohne Umweg — `groups.map`, `schedule.length`.
// Ein nicht belegter Go-Slice wird in JSON zu `null`, und `null.map` wirft im
// Render einen TypeError, der ohne ErrorBoundary den ganzen Baum mitnimmt.
// Betroffen wäre jeweils der Randfall, den niemand von Hand ausprobiert: die
// Richtung ohne Treffer, der Plan ohne Jahr, die große Gesellschaft.
func assertNotJSONNull(t *testing.T, label string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("%s als JSON: %v", label, err)
	}
	if string(raw) == "null" {
		t.Errorf("%s kommt als `null` in der Oberfläche an", label)
	}
}

// Die Auswahl der Buchungsgruppen fragt mit einer Richtung. Eine unbekannte
// liefert keine Gruppe — aber eine leere Liste.
func TestPostingGroupsUnknownDirectionIsEmptyList(t *testing.T) {
	assertNotJSONNull(t, "Buchungsgruppen (unbekannte Richtung)",
		PostingGroups(domain.Direction("keine")))
	if len(PostingGroups(domain.Direction("keine"))) != 0 {
		t.Error("eine unbekannte Richtung soll keine Gruppe liefern")
	}
	// Gegenprobe: die bekannten Richtungen liefern weiterhin Gruppen.
	if len(PostingGroups(domain.DirectionIncoming)) == 0 {
		t.Error("für Eingangsbelege fehlen die Buchungsgruppen")
	}
}

// Dieselbe Zusage für die Anlagekonten.
func TestAssetAccountsUnknownClassIsEmptyList(t *testing.T) {
	assertNotJSONNull(t, "Anlagekonten (unbekannte Klasse)",
		AssetAccounts(domain.AssetClass("keine")))
}

// Der AfA-Plan wird gerechnet, während die Maske noch ausgefüllt wird. Solange
// die Anschaffungskosten fehlen, gibt es kein Jahr — die Tabelle bleibt leer
// und die Antwort ist trotzdem eine Liste.
func TestAfAScheduleWithoutYearsIsEmptyList(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:  "2026-03-15",
		Cost:             0,
		UsefulLifeMonths: 36,
		Method:           domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("Plan ohne Anschaffungskosten: %v", err)
	}
	assertNotJSONNull(t, "AfA-Plan ohne Anschaffungskosten", rows)

	// Finanzanlagen werden nicht planmäßig abgeschrieben (§ 253 Abs. 3 HGB
	// kennt für sie nur die außerplanmäßige Abschreibung).
	none, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate: "2026-03-15",
		Cost:            500_000,
		Method:          domain.DepreciationNone,
	})
	if err != nil {
		t.Fatalf("Plan ohne planmäßige Abschreibung: %v", err)
	}
	assertNotJSONNull(t, "AfA-Plan ohne planmäßige Abschreibung", none)
}

// Die große Gesellschaft erfüllt kein Größenmerkmal. Genau dort stand bisher
// ein ausdrückliches nil.
func TestSizeAssessmentOfLargeCompanyHasEmptyMet(t *testing.T) {
	a, err := AssessSize(2026, "2026-12-31", "2026-01-01", domain.SizeCriteria{
		BalanceSheetTotal: 100_000_000_00,
		Revenue:           200_000_000_00,
		Employees:         900,
	})
	if err != nil {
		t.Fatalf("Größenklasse: %v", err)
	}
	if a.Class != domain.SizeLarge {
		t.Fatalf("für diesen Fall wird die große Gesellschaft erwartet, erhalten %q", a.Class)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Größenklasse als JSON: %v", err)
	}
	if strings.Contains(string(raw), `"met":null`) {
		t.Errorf("die erfüllten Merkmale kommen als `null` an: %s", raw)
	}
}
