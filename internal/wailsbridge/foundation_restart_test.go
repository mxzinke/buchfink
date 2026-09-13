package wailsbridge

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/zalando/go-keyring"
)

func TestFoundationRestartKeepsProgressAndShortYearDeadlines(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { repository.SetActiveVault(nil) })
	base := t.TempDir()
	config := repository.NewAppConfigRepository(filepath.Join(base, "config"))
	b := &BuchfinkBridge{appCfgRepo: config, currentYear: 2026}
	tenant, err := b.CreateTenant("August GmbH", filepath.Join(base, "company"), domain.CompanySettings{
		CompanyName: "August GmbH", LegalForm: "GmbH", FiscalYear: 2026, VatPeriod: "unknown",
	})
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := b.SaveFoundation(domain.Foundation{
		NotarizedOn: "2026-08-15", RegisteredOn: "2026-08-25", ShareCapital: 2500000,
		Shareholders: []domain.Shareholder{{Name: "Anna", ShareCapital: 2500000, PaidIn: 1250000, Kind: domain.ContributionCash}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.CompleteFoundationDuty("fragebogen", "2026-08-28", "Übermittelt"); err != nil {
		t.Fatal(err)
	}
	if err := b.CompleteFoundationDuty("betriebsnummer", "2026-08-28", "Nicht zutreffend"); err != nil {
		t.Fatal(err)
	}
	settings, err := b.GetCompanySettings()
	if err != nil {
		t.Fatal(err)
	}
	settings.Employees = "none"
	if err := b.UpdateCompanySettings(*settings); err != nil {
		t.Fatal(err)
	}
	if err := repository.NewFoundationRepository(b.db).CompleteTask(context.Background(), &domain.FoundationTask{FoundationID: foundation.ID, Key: "ust_id", Status: "deferred", DoneOn: "2026-08-28"}); err != nil {
		t.Fatal(err)
	}
	before, err := b.GetFoundationState()
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CloseDB(b.db); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	restarted := &BuchfinkBridge{appCfgRepo: config, appConfig: *cfg, currentYear: 2026}
	if err := restarted.initTenant(tenant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.CloseDB(restarted.db) })
	state, err := restarted.GetFoundationState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.HasFoundation || state.Guide != before.Guide || state.Guide.Done >= state.Guide.Total {
		t.Fatalf("Gründungsfortschritt nach Neustart: %+v", state)
	}
	savedSettings, err := restarted.GetCompanySettings()
	if err != nil {
		t.Fatal(err)
	}
	if savedSettings.Employees != "none" {
		t.Error("Beschäftigtenangabe geht beim Neustart verloren")
	}
	for _, duty := range state.Duties {
		if duty.Key == "ust_id" && (duty.IsDone || duty.IsNotApplicable || duty.IsPending) {
			t.Error("zuvor zurückgestellte Aufgabe muss wieder offen sein")
		}
		if duty.Key == "betriebsnummer" && !duty.IsNotApplicable {
			t.Error("Nicht zutreffend geht beim Neustart verloren")
		}
		if duty.Key == "handelsregister" || duty.Key == "offenlegung" || duty.Key == "ruecklage" {
			t.Errorf("Aufgabe außerhalb der Gründungscheckliste: %s", duty.Key)
		}
	}
	years, err := restarted.GetFiscalYears()
	if err != nil {
		t.Fatal(err)
	}
	if len(years) != 1 || years[0].StartDate != "2026-08-15" || !years[0].IsShort {
		t.Fatalf("Rumpfjahr: %+v", years)
	}
	deadlines, err := restarted.GetDeadlines(2026)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, d := range deadlines {
		if strings.HasPrefix(d.Key, "festschreibung.") {
			count++
			if d.Key < "festschreibung.2026-08" {
				t.Errorf("Frist vor Gründung: %+v", d)
			}
		}
	}
	if count != 5 {
		t.Fatalf("%d Festschreibungsfristen, erwartet August bis Dezember: %+v", count, deadlines)
	}
	august, err := restarted.GetMonthCloseState("2026-08")
	if err != nil {
		t.Fatalf("Angefangener Gründungsmonat nicht erreichbar: %v", err)
	}
	for _, finding := range august.Findings {
		if strings.HasPrefix(finding.ObjectID, "2026-") && finding.ObjectID < "2026-08" {
			t.Errorf("Befund vor Gründung: %+v", finding)
		}
	}
	if _, err := restarted.GetMonthCloseState("2026-07"); err == nil {
		t.Error("Monat vor Gründung akzeptiert")
	}
}
