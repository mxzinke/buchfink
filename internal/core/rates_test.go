package core

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// weekendRates stellt jeden Kurs am Freitag, dem 27.02.2026, fest.
type weekendRates struct{ calls atomic.Int32 }

func (r *weekendRates) RateAt(context.Context, string, string) (int64, string, string, error) {
	r.calls.Add(1)
	return 1_085_000, "2026-02-27", "EZB-Probe", nil
}

func storedRates(t *testing.T, tenant *Tenant) []domain.ExchangeRate {
	t.Helper()
	var rates []domain.ExchangeRate
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) (err error) {
		rates, err = s.Currency.Rates(ctx, "USD", "", "")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return rates
}

// Im Prüfermodus holt auch ein Lesevorgang keinen Kurs; sonst legte er ihn ab
// und änderte den Bestand.
func TestExaminerModeFetchesNoRates(t *testing.T) {
	rates := &weekendRates{}
	h := newHostWith(t, Network{Rates: rates})
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	until := today()
	if _, err := tenant.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}

	if err := tenant.Read(agentCtx(t), func(ctx context.Context, s *Services) error {
		_, err := s.Currency.RateAt(ctx, "USD", "2026-03-01")
		return err
	}); err == nil {
		t.Error("im Prüfermodus kam ein Kurs, obwohl keiner gespeichert ist")
	}
	if n := rates.calls.Load(); n != 0 {
		t.Errorf("der Kursdienst wurde im Prüfermodus %d-mal gefragt", n)
	}
	if got := storedRates(t, tenant); len(got) != 0 {
		t.Errorf("im Prüfermodus abgelegte Kurse: %+v", got)
	}
}

// Ein geholter Kurs ersetzt keinen von Hand erfassten desselben Tages.
func TestFetchedRateKeepsAManualRate(t *testing.T) {
	h := newHostWith(t, Network{Rates: &weekendRates{}})
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Currency.SaveRate(ctx, domain.ExchangeRate{
			Currency: "USD", Date: "2026-02-27", RateMicros: 1_100_000, Source: "Bankbeleg",
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := tenant.Read(agentCtx(t), func(ctx context.Context, s *Services) error {
		_, err := s.Currency.RateAt(ctx, "USD", "2026-03-01")
		return err
	}); err != nil {
		t.Fatalf("Kurs zum Sonntag: %v", err)
	}

	byDate := map[string]domain.ExchangeRate{}
	for _, r := range storedRates(t, tenant) {
		byDate[r.Date] = r
	}
	manual := byDate["2026-02-27"]
	if !manual.Manual || manual.Source != "Bankbeleg" || manual.RateMicros != 1_100_000 {
		t.Errorf("der Handkurs wurde ersetzt: %+v", manual)
	}
	if fetched, ok := byDate["2026-03-01"]; !ok || fetched.Manual {
		t.Errorf("der geholte Kurs zum angefragten Tag fehlt: %+v", fetched)
	}
}

// Ohne Protokolleintrag bleibt kein Kurs gespeichert, weder von Hand noch
// aus dem Import der Umsatzsteuer-Umrechnungskurse.
func TestRatesNeedTheirAuditEntry(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.db.Exec(`CREATE TRIGGER kein_protokoll BEFORE INSERT ON audit_log_entries
		BEGIN SELECT RAISE(ABORT, 'Protokoll nicht verfügbar'); END`).Error; err != nil {
		t.Fatal(err)
	}

	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Currency.SaveRate(ctx, domain.ExchangeRate{
			Currency: "USD", Date: "2026-02-03", RateMicros: 1_100_000, Source: "Bankbeleg",
		})
		return err
	}); err == nil {
		t.Error("Kurs von Hand ohne Protokolleintrag gespeichert")
	}
	if got := storedRates(t, tenant); len(got) != 0 {
		t.Errorf("Kurse ohne Protokolleintrag: %+v", got)
	}

	csv := filepath.Join(t.TempDir(), "kurse.csv")
	if err := os.WriteFile(csv, []byte("Monat;Währung;Kurs\n2026-02;USD;1,0850\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Currency.ImportVatRatesCSV(ctx, csv)
		return err
	}); err == nil {
		t.Error("Import der Umrechnungskurse ohne Protokolleintrag")
	}
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		vat, err := s.Currency.VatRates(ctx, "", "")
		if err == nil && len(vat) != 0 {
			t.Errorf("Umrechnungskurse ohne Protokolleintrag: %+v", vat)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
