package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Abschluss-Einstellungen kommen mit den Voreinstellungen der Bausteine
// zurück, lassen sich speichern und werden vor dem Speichern geprüft.
func TestClosingSettingsRoundTrip(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewClosingSettingsService(repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db))

	got, err := svc.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.TradeTaxRatePercent != 400 || got.AccrualMethod != "monthly" || got.AccrualThreshold != domain.DefaultAccrualThreshold {
		t.Fatalf("unerwartete Voreinstellungen: %+v", got)
	}

	saved, err := svc.Save(ctx, ClosingSettings{TradeTaxRatePercent: 470, AccrualMethod: "daily", AccrualThreshold: 50_000, AccrualRelease: "monthly"})
	if err != nil {
		t.Fatal(err)
	}
	if saved.TradeTaxRatePercent != 470 || saved.AccrualMethod != "daily" || saved.AccrualThreshold != 50_000 || saved.AccrualRelease != "monthly" {
		t.Errorf("gespeichert wurde %+v", saved)
	}
	if v, _ := repository.NewSettingsRepository(env.db).Get(ctx, domain.SettingTradeTaxRate); v != "470" {
		t.Errorf("Hebesatz liegt als %q in der Einstellungstabelle, erwartet 470", v)
	}

	if _, err := svc.Save(ctx, ClosingSettings{TradeTaxRatePercent: 150, AccrualMethod: "monthly", AccrualRelease: "yearly"}); err == nil {
		t.Error("ein Hebesatz unter 200 % wurde angenommen")
	}
	if _, err := svc.Save(ctx, ClosingSettings{TradeTaxRatePercent: 400, AccrualMethod: "weekly", AccrualRelease: "yearly"}); err == nil {
		t.Error("eine unbekannte Abgrenzungsmethode wurde angenommen")
	}
}
