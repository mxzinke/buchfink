package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func (env *testEnv) completeFoundationMasterData(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	repo := repository.NewSettingsRepository(env.db)
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.CompanyName, cfg.Seat = "Beispiel GmbH", ""
	cfg.Street, cfg.PostalCode, cfg.City = "Teststraße 1", "80331", "München"
	cfg.ManagingDirectors, cfg.TaxOffice, cfg.Employees = "", "", "unknown"
	cfg.TaxNumber, cfg.VatID, cfg.VatPeriod = "", "", "unknown"
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestFoundationMasterDataCompletionRequiresSavedFields(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	ctx := context.Background()
	f := env.saveFoundation(t, svc, gmbhFoundation())
	repo := repository.NewSettingsRepository(env.db)
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Street = ""
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	duty := func(key string) domain.FoundationDuty {
		t.Helper()
		state, err := env.foundations(t).GetState(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range state.Duties {
			if d.Key == key {
				return d
			}
		}
		t.Fatalf("Aufgabe fehlt: %s", key)
		return domain.FoundationDuty{}
	}
	if len(duty("stammdaten").MissingFields) == 0 {
		t.Fatal("Fehlende Stammdaten werden nicht genannt")
	}
	for _, complete := range []func() error{
		func() error { return svc.SetDutyStatus(ctx, "stammdaten", "done") },
		func() error { return svc.CompleteDuty(ctx, "stammdaten", "2026-08-20", "") },
	} {
		if err := complete(); err == nil || !strings.Contains(err.Error(), "Stammdaten ergänzen") {
			t.Fatalf("Unvollständige Stammdaten akzeptiert: %v", err)
		}
	}
	if err := repository.NewFoundationRepository(env.db).CompleteTask(ctx, &domain.FoundationTask{
		FoundationID: f.ID, Key: "stammdaten", DoneOn: "2026-08-20", Status: "done",
	}); err != nil {
		t.Fatal(err)
	}
	if duty("stammdaten").IsDone || !duty("fragebogen").IsPending {
		t.Fatal("Alter Haken umgeht die Vollständigkeitsprüfung")
	}
	if err := svc.SetDutyStatus(ctx, "stammdaten", "open"); err != nil {
		t.Fatal(err)
	}
	env.completeFoundationMasterData(t)
	if d := duty("stammdaten"); len(d.MissingFields) != 0 || d.IsDone {
		t.Fatalf("Vollständige Stammdaten ohne Steuer- und Registernummern: %+v", d)
	}
	if err := svc.SetDutyStatus(ctx, "stammdaten", "done"); err != nil {
		t.Fatal(err)
	}
	if !duty("stammdaten").IsDone || duty("fragebogen").IsPending {
		t.Fatal("Abhaken gibt Folgeaufgabe nicht frei")
	}
	cfg, err = repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Street = "  "
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if d := duty("stammdaten"); d.IsDone || len(d.MissingFields) != 1 || d.MissingFields[0] != "Straße und Hausnummer" {
		t.Fatalf("Gelöschte Stammdaten werden nicht erkannt: %+v", d)
	}
	if !duty("fragebogen").IsPending {
		t.Fatal("Folgeaufgabe bleibt ohne Stammdaten freigegeben")
	}
	cfg.Street, cfg.Shareholders = "Teststraße 1", []domain.CompanyShareholder{}
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if d := duty("stammdaten"); d.IsDone || len(d.MissingFields) != 1 || d.MissingFields[0] != "Gesellschafter" {
		t.Fatalf("Gelöschte Gesellschafter in den Stammdaten werden nicht erkannt: %+v", d)
	}
	if err := svc.SetDutyStatus(ctx, "stammdaten", "done"); err == nil {
		t.Fatal("Abhaken ohne Gesellschafter möglich")
	}
}

func TestFoundationMasterDataMinimum(t *testing.T) {
	minimum := domain.CompanySettings{
		CompanyName: "Beispiel GmbH", Street: "Teststraße 1", PostalCode: "80331", City: "München",
		Employees: "unknown", VatPeriod: "unknown", RegisteredOn: "2026-08-20",
		Shareholders: []domain.CompanyShareholder{{Name: "Anna Bauer", ShareCapital: 2500000}},
	}
	if missing := foundationMasterDataMissing(&minimum); len(missing) != 0 {
		t.Fatalf("Mindestangaben reichen ohne weitere Stammdaten nicht aus: %v", missing)
	}
	for _, tc := range []struct {
		name   string
		change func(*domain.CompanySettings)
	}{
		{"Firma", func(cfg *domain.CompanySettings) { cfg.CompanyName = " " }},
		{"Straße", func(cfg *domain.CompanySettings) { cfg.Street = " " }},
		{"Postleitzahl", func(cfg *domain.CompanySettings) { cfg.PostalCode = "" }},
		{"Ort", func(cfg *domain.CompanySettings) { cfg.City = "" }},
		{"Gesellschafter fehlen", func(cfg *domain.CompanySettings) { cfg.Shareholders = nil }},
		{"Gesellschafter ohne Namen", func(cfg *domain.CompanySettings) {
			cfg.Shareholders = []domain.CompanyShareholder{{Name: " ", ShareCapital: 2500000}}
		}},
		{"Gesellschafter ohne Anteil", func(cfg *domain.CompanySettings) {
			cfg.Shareholders = []domain.CompanyShareholder{{Name: "Anna Bauer"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := minimum
			tc.change(&cfg)
			if missing := foundationMasterDataMissing(&cfg); len(missing) != 1 {
				t.Fatalf("Fehlende Mindestangabe nicht erkannt: %v", missing)
			}
		})
	}
}
