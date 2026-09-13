package wailsbridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/zalando/go-keyring"
)

func TestFoundationProofCompletesTaskAtomicallyAndSurvivesRestart(t *testing.T) {
	keyring.MockInit()
	t.Cleanup(func() { repository.SetActiveVault(nil) })
	base := t.TempDir()
	config := repository.NewAppConfigRepository(filepath.Join(base, "config"))
	b := &BuchfinkBridge{appCfgRepo: config, currentYear: 2026}
	tenant, err := b.CreateTenant("Nachweis GmbH", filepath.Join(base, "company"), domain.CompanySettings{
		CompanyName: "Nachweis GmbH", LegalForm: "GmbH", FiscalYear: 2026, VatPeriod: "unknown",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.CloseDB(b.db) })
	if _, err := b.SaveFoundation(domain.Foundation{
		NotarizedOn: "2026-08-15", ShareCapital: 2500000,
		Shareholders: []domain.Shareholder{{Name: "Anna", ShareCapital: 2500000, PaidIn: 1250000, Kind: domain.ContributionCash}},
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "IHK.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4 Testnachweis"), 0600); err != nil {
		t.Fatal(err)
	}
	req := service.DocumentRequest{Kind: domain.DocBehoerde, Path: path, DutyKey: "ihk", DocumentDate: "2026-08-03"}
	check := func(key string, count int, done bool) domain.FoundationDuty {
		t.Helper()
		state, err := b.GetFoundationState()
		if err != nil {
			t.Fatal(err)
		}
		for _, duty := range state.Duties {
			if duty.Key == key {
				if len(duty.Proof) != count || duty.IsDone != done {
					t.Fatalf("Aufgabe %s: %+v", key, duty)
				}
				return duty
			}
		}
		t.Fatalf("Aufgabe fehlt: %s", key)
		return domain.FoundationDuty{}
	}
	for _, key := range []string{"stammdaten", "geschaeftsbriefe", "lohnabrechnung", "fragebogen", "betriebsnummer"} {
		invalid := req
		invalid.DutyKey = key
		if _, err := b.AttachDocument(invalid); err == nil {
			t.Fatalf("Nachweis für gesperrte Aufgabe %s akzeptiert", key)
		}
		check(key, 0, false)
	}
	missing := req
	missing.Path = filepath.Join(base, "missing.pdf")
	if _, err := b.AttachDocument(missing); err == nil {
		t.Fatal("Fehlende Datei akzeptiert")
	}
	check("ihk", 0, false)
	if err := b.db.Exec("CREATE TRIGGER reject_task BEFORE INSERT ON foundation_tasks BEGIN SELECT RAISE(ABORT, 'test task failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := b.AttachDocument(req); err == nil {
		t.Fatal("Fehler beim Abhaken nicht weitergegeben")
	}
	check("ihk", 0, false)
	if err := b.db.Exec("DROP TRIGGER reject_task").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := b.AttachDocument(req); err != nil {
		t.Fatal(err)
	}
	check("ihk", 1, true)
	if err := b.CompleteFoundationDuty("ihk", "2026-08-20", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AttachDocument(req); err != nil {
		t.Fatal(err)
	}
	if duty := check("ihk", 2, true); duty.DoneOn != "2026-08-20" {
		t.Fatal("Weiterer Nachweis verändert das Erledigungsdatum")
	}
	if err := repository.CloseDB(b.db); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	b = &BuchfinkBridge{appCfgRepo: config, appConfig: *cfg, currentYear: 2026}
	if err := b.initTenant(tenant); err != nil {
		t.Fatal(err)
	}
	check("ihk", 2, true)
}
