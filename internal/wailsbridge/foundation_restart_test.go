package wailsbridge

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestFoundationRestartKeepsShortYearDeadlines(t *testing.T) {
	base := t.TempDir()
	config := repository.NewAppConfigRepository(filepath.Join(base, "config"))
	withConfig := func(o *core.HostOptions) { o.Config, o.BaseDir = config, base }
	b := newBridge(t, withConfig)
	if _, err := b.createTenant("August GmbH", filepath.Join(base, "company"), domain.CompanySettings{
		CompanyName: "August GmbH", LegalForm: "GmbH", FiscalYear: 2026, VatPeriod: "unknown",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveFoundation(domain.Foundation{
		NotarizedOn: "2026-08-15", RegisteredOn: "2026-08-25", ShareCapital: 2500000,
		Shareholders: []domain.Shareholder{{Name: "Anna", ShareCapital: 2500000, PaidIn: 1250000, Kind: domain.ContributionCash}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := b.host.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	restarted := newBridge(t, withConfig)
	state, err := restarted.GetFoundationState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.HasFoundation || state.Foundation.NotarizedOn != "2026-08-15" {
		t.Fatalf("Gründung nach Neustart: %+v", state)
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
