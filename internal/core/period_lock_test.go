package core

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// bookOn bucht die Testbuchung zu einem Tag.
func bookOn(ctx context.Context, tenant *Tenant, date string) (*domain.JournalEntry, error) {
	var created *domain.JournalEntry
	err := tenant.Write(ctx, func(ctx context.Context, s *Services) error {
		req, err := receiptAndEntry(ctx, s)
		if err != nil {
			return err
		}
		e := &req.Entry
		e.BookingDate, e.DocumentDate, e.ServiceDateFrom, e.ServiceDateTo = date, date, date, date
		created, err = s.Posting.PostManualEntry(ctx, req)
		return err
	})
	return created, err
}

// Ein geänderter Beginnmonat ordnet einen Tag einem anderen Geschäftsjahr zu.
// Die Festschreibung hält den Tag trotzdem, und der Beginnmonat selbst lässt
// sich nach der ersten Buchung auf keinem Weg mehr ändern.
func TestCommittedPeriodSurvivesAChangedStartMonth(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)

	if _, err := bookOn(uiCtx, tenant, "2026-02-01"); err != nil {
		t.Fatalf("Buchung vor der Festschreibung: %v", err)
	}
	if _, err := tenant.CommitPeriod(uiCtx, CommitRequest{
		PeriodType: "month", PeriodLabel: "Juni 2026", CutoffDate: "2026-06-30",
		OverrideReason: "Testlauf ohne Belegprüfung",
	}); err != nil {
		t.Fatalf("Festschreibung bis Juni: %v", err)
	}
	if _, err := bookOn(agent, tenant, "2026-03-15"); err == nil {
		t.Fatal("Kontrollbuchung in den festgeschriebenen März angenommen")
	}

	// Kein Weg ändert den Beginnmonat: weder der einzelne Schlüssel noch die
	// Unternehmensdaten, auch nicht aus der App.
	if err := tenant.Write(agent, func(ctx context.Context, s *Services) error {
		return s.Settings.SetValue(ctx, "fiscal_year_start_month", "7", "")
	}); err == nil {
		t.Error("ein Agent setzt den Beginnmonat über SetValue")
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		settings, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		settings.FiscalYearStartMonth = 7
		return s.Settings.UpdateCompanySettings(ctx, settings)
	}, AppOnly()); err == nil || !strings.Contains(err.Error(), "Beginn des Geschäftsjahres") {
		t.Errorf("die App ändert den Beginnmonat nach der ersten Buchung: %v", err)
	}

	// Auch ein Beginnmonat, der an allen Prüfungen vorbei in die Datei kam,
	// öffnet den festgeschriebenen Zeitraum nicht.
	if err := repository.NewSettingsRepository(tenant.db).Set(context.Background(), "fiscal_year_start_month", "7"); err != nil {
		t.Fatal(err)
	}
	if year, err := tenant.FiscalYearOf(uiCtx, "2026-03-15"); err != nil || year != 2025 {
		t.Fatalf("Vorbedingung: der 15.03.2026 fällt jetzt in das Jahr %d (%v), erwartet 2025", year, err)
	}
	if entry, err := bookOn(agent, tenant, "2026-03-15"); err == nil || !strings.Contains(err.Error(), "festgeschrieben") {
		t.Errorf("Buchung in den festgeschriebenen März nach geändertem Beginnmonat: %+v, %v", entry, err)
	}
}

// Ein festgestelltes Jahr bleibt gesperrt, auch wenn ein geänderter Beginnmonat
// den Tag einer anderen Jahresnummer zuschlägt.
func TestAdoptedYearSurvivesAChangedStartMonth(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	ctx := context.Background()

	years := repository.NewFiscalYearRepository(tenant.db)
	fy, err := years.FindByYear(ctx, 2026)
	if err != nil || fy == nil {
		t.Fatalf("Geschäftsjahr 2026: %+v, %v", fy, err)
	}
	fy.Status, fy.PreparedOn, fy.AdoptedOn = domain.FiscalYearAdopted, "2026-01-02", "2026-01-03"
	if err := years.Save(ctx, fy); err != nil {
		t.Fatal(err)
	}
	if err := repository.NewSettingsRepository(tenant.db).Set(ctx, "fiscal_year_start_month", "7"); err != nil {
		t.Fatal(err)
	}
	if _, err := bookOn(agentCtx(t), tenant, "2026-03-15"); err == nil || !strings.Contains(err.Error(), "festgestellt") {
		t.Errorf("Buchung in das festgestellte Jahr nach geändertem Beginnmonat: %v", err)
	}
}

// Ohne Buchung und Festschreibung bleibt der Beginnmonat änderbar: so richtet
// die App einen Mandanten mit abweichendem Wirtschaftsjahr ein.
func TestStartMonthChangesBeforeTheFirstBooking(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		settings, err := s.Settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		settings.FiscalYearStartMonth = 7
		return s.Settings.UpdateCompanySettings(ctx, settings)
	}, AppOnly()); err != nil {
		t.Fatalf("Beginnmonat vor der ersten Buchung: %v", err)
	}
}
