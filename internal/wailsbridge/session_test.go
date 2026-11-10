package wailsbridge

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Was über die Bridge kommt, trägt den Erfassungsweg der App.
func TestBridgeRecordsTheAppChannel(t *testing.T) {
	b := testBridge(t)
	entry, err := postManualEntry(b, manualEntryRequest(t, b))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if entry.Channel != string(actor.ChannelUI) {
		t.Errorf("die Buchung trägt %q, erwartet ui", entry.Channel)
	}
	logged := auditEntries(t, b, domain.AuditFilter{EntityType: "JOURNAL"})
	if len(logged) == 0 {
		t.Fatal("die Buchung steht nicht im Protokoll")
	}
	for _, e := range logged {
		if e.Channel != string(actor.ChannelUI) {
			t.Errorf("Protokolleintrag %d trägt %q, erwartet ui", e.ID, e.Channel)
		}
	}
}

// hangingRates hält den ersten Kursabruf an, bis der Test ihn freigibt.
type hangingRates struct {
	asked, release chan struct{}
	once           sync.Once
}

func (r *hangingRates) RateAt(ctx context.Context, _, date string) (int64, string, string, error) {
	r.once.Do(func() {
		close(r.asked)
		<-r.release
	})
	return 1_085_000, date, "Testkurs", nil
}

// Ein Vorgang mit Vorabruf schreibt auf dem Mandanten, für den er vorab
// geholt hat. Wechselt die App währenddessen, endet er, statt im neuen
// Mandanten mit Angaben aus dem alten zu buchen.
func TestPrefetchedWriteStaysWithItsTenant(t *testing.T) {
	rates := &hangingRates{asked: make(chan struct{}), release: make(chan struct{})}
	b := testBridge(t, func(o *core.HostOptions) { o.Network.Rates = rates })
	first, err := b.GetActiveTenant()
	if err != nil || first == nil {
		t.Fatalf("erster Mandant: %+v, %v", first, err)
	}
	second, err := b.createTenant("Zweite GmbH", filepath.Join(t.TempDir(), "zweiter"),
		domain.CompanySettings{CompanyName: "Zweite GmbH", FiscalYear: 2026})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.SwitchTenant(first.ID); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := b.PostIncomingReceipt(service.ReceiptRequest{ReceiptID: 1, Currency: "USD", DocumentDate: "2026-03-02"})
		done <- err
	}()
	<-rates.asked
	if err := b.SwitchTenant(second.ID); err != nil {
		t.Fatal(err)
	}
	close(rates.release)
	if err := <-done; !errors.Is(err, core.ErrTenantClosing) {
		t.Errorf("Vorgang nach dem Wechsel: %v, erwartet ErrTenantClosing", err)
	}

	entries, err := b.GetAllJournalEntries()
	if err != nil {
		t.Fatal(err)
	}
	if active, _ := b.GetActiveTenant(); active == nil || active.ID != second.ID || len(entries) != 0 {
		t.Errorf("im neuen Mandanten %+v entstanden %d Buchungen", active, len(entries))
	}
}
