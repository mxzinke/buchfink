package wailsbridge

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

func TestAdapterAssetRuleValues(t *testing.T) {
	b := testBridge(t)
	rules, err := b.GetAssetRules()
	if err != nil {
		t.Fatal(err)
	}
	got := []int64{int64(rules.FiscalYear), int64(rules.GWGImmediateLimit), int64(rules.GWGRecordFrom), int64(rules.PoolLowerLimit), int64(rules.PoolUpperLimit), int64(rules.PoolYears), int64(rules.SpecialMaxPermille), int64(rules.SpecialPeriodYears)}
	want := []int64{2026, 80000, 25000, 25000, 100000, 5, 400, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Anlagenregeln: %v, erwartet %v", got, want)
	}
	if !reflect.DeepEqual(rules.DegressiveWindows, accounting.DegressiveWindows()) || !reflect.DeepEqual(rules.ElectricVehicleWindows, accounting.AfARuleSet().ElectricVehicle) {
		t.Fatal("Zeitfenster weichen ab")
	}
}

func TestAdapterInvestmentRulesWithoutTenant(t *testing.T) {
	b := newBridge(t)
	rules, err := b.GetInvestmentRules()
	if err != nil {
		t.Fatal(err)
	}
	if rules.InvestorType != "" || rules.InvestorReason != "" || rules.Exemptions == nil || len(rules.Exemptions) != 0 || len(rules.FundClasses) != len(accounting.AllFundClasses()) {
		t.Fatalf("Investmentregeln ohne Mandant: %+v", rules)
	}
}

func TestAdapterComplianceSystemChange(t *testing.T) {
	b := testBridge(t)
	for _, date := range []string{"", "2000-01-01", "2090-06-30", "ungueltig"} {
		testWrite(t, b, func(ctx context.Context, s *core.Services) error {
			return s.Settings.SetValue(ctx, domain.SettingSystemChangeDate, date, "Testdatum")
		})
		got, err := b.GetComplianceHints()
		if err != nil {
			t.Fatal(err)
		}
		want := &ComplianceHints{DataDir: b.host.Config().DataDir, TaxCaseHints: domain.TaxCaseHints(), SystemChangeDate: date, SystemChangeNote: accounting.SystemChangeNote(date, "")}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Hinweise für %q: %+v, erwartet %+v", date, got, want)
		}
	}
}

func TestAdapterOpenItemsAgingJSON(t *testing.T) {
	for i, b := range []*BuchfinkBridge{newBridge(t), testBridge(t)} {
		got, err := b.GetOpenItemsAging("2026-06-30")
		if err != nil {
			t.Fatal(err)
		}
		want := accounting.AgeOpenItems(nil, "2026-06-30")
		// Ohne Mandant liefert die Bridge nur Stichtag und leere Listen.
		if i == 0 {
			want = domain.OpenItemsAging{Cutoff: "2026-06-30"}
		}
		want.EnsureLists()
		rawGot, _ := json.Marshal(got)
		rawWant, _ := json.Marshal(want)
		if string(rawGot) != string(rawWant) {
			t.Fatalf("Altersstruktur: %s, erwartet %s", rawGot, rawWant)
		}
	}
}
