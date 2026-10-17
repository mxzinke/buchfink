package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Datenordner hat einen Schreiber: ein zweites Öffnen, etwa aus einer
// zweiten App-Instanz, wird abgewiesen, bis das erste geschlossen ist.
func TestDataDirHasOneWriter(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	if _, err := Open(systemCtx, tc.DataDir, Options{TenantID: tc.ID}); !errors.Is(err, ErrTenantInUse) {
		t.Fatalf("zweites Öffnen desselben Ordners: %v", err)
	}
	if err := h.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	again, err := Open(systemCtx, tc.DataDir, Options{TenantID: tc.ID})
	if err != nil {
		t.Fatalf("Öffnen nach dem Schließen: %v", err)
	}
	_ = again.Close(context.Background())
}

// Wer zu einem Mandanten zurückwechselt, der nach dem Wechsel noch auf einen
// laufenden Vorgang wartet, wartet mit ihm und öffnet danach.
func TestSwitchBackWaitsForTheClosingTenant(t *testing.T) {
	h := newHost(t)
	first := newTenant(t, h, "Erste GmbH")
	_, release, err := h.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	newTenant(t, h, "Zweite GmbH")

	done := make(chan error, 1)
	go func() {
		_, err := h.Switch(uiCtx, first.ID)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("zurückgewechselt, während der Mandant noch schließt: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	release()
	if err := <-done; err != nil {
		t.Fatalf("Wechsel zurück: %v", err)
	}
	if got := openTenant(t, h); got.ID() != first.ID {
		t.Errorf("offen ist %s", got.ID())
	}
}

// Nach dem Beenden öffnet der Host keinen Mandanten mehr.
func TestClosedHostOpensNothing(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	if err := h.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Switch(uiCtx, tc.ID); !errors.Is(err, ErrHostClosed) {
		t.Errorf("Wechsel nach dem Beenden: %v", err)
	}
	if _, err := h.CreateTenant(uiCtx, "Neu", filepath.Join(t.TempDir(), "neu"), domain.CompanySettings{FiscalYear: 2026}); !errors.Is(err, ErrHostClosed) {
		t.Errorf("Anlegen nach dem Beenden: %v", err)
	}
	if _, err := h.RegisterTenant(uiCtx, filepath.Join(tc.DataDir, "buchfink.sqlite"), "", domain.MigrationKindOpen); !errors.Is(err, ErrHostClosed) {
		t.Errorf("Anmelden nach dem Beenden: %v", err)
	}
	if err := h.RemoveTenant(uiCtx, tc.ID); !errors.Is(err, ErrHostClosed) {
		t.Errorf("Entfernen nach dem Beenden: %v", err)
	}
	if err := h.Close(context.Background()); err != nil {
		t.Errorf("zweites Beenden: %v", err)
	}
}

// Die Transaktion im Kontext gehört zu ihrem Mandanten. Ein anderer Mandant
// läuft nicht in ihr weiter, sondern weist den Vorgang ab.
func TestForeignTenantContextIsRefused(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Erste GmbH")
	first := openTenant(t, h)
	second, err := Open(systemCtx, filepath.Join(t.TempDir(), "zweiter"), Options{TenantID: "zweiter"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close(context.Background()) })
	noop := func(context.Context, *Services) error { return nil }

	if err := first.Write(uiCtx, func(ctx context.Context, _ *Services) error {
		return second.Write(ctx, noop)
	}); !errors.Is(err, ErrForeignTenant) {
		t.Errorf("Write des zweiten Mandanten im Write des ersten: %v", err)
	}
	if err := first.Write(uiCtx, func(ctx context.Context, _ *Services) error {
		return second.Read(ctx, noop)
	}); !errors.Is(err, ErrForeignTenant) {
		t.Errorf("Read des zweiten Mandanten im Write des ersten: %v", err)
	}
	if err := first.Read(uiCtx, func(ctx context.Context, _ *Services) error {
		return second.Read(ctx, noop)
	}); !errors.Is(err, ErrForeignTenant) {
		t.Errorf("Read des zweiten Mandanten im Read des ersten: %v", err)
	}
	// Derselbe Mandant läuft in seiner Transaktion weiter.
	if err := first.Write(uiCtx, func(ctx context.Context, _ *Services) error {
		if err := first.Read(ctx, noop); err != nil {
			return err
		}
		return first.Write(ctx, noop)
	}); err != nil {
		t.Errorf("verschachtelter Vorgang desselben Mandanten: %v", err)
	}
}
