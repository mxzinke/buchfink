package wailsbridge

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Einrichtung eines Umsteigers legt den Umstieg fest; die Abstimmung nennt
// danach die fehlende Eröffnung, und eine Gründung ist ausgeschlossen.
func TestSetupOfATakeoverFixesTheStartOfTheBooks(t *testing.T) {
	b := newBridge(t)
	if err := b.SetupApplication(t.TempDir()+"/mandant", "", domain.CompanySettings{
		CompanyName: "Nordlicht Systeme GmbH", LegalForm: "GmbH", FiscalYear: 2026,
		FiscalYearStartMonth: 1, FederalState: "HH", VatPeriod: "unknown",
	}); err != nil {
		t.Fatalf("Einrichtung: %v", err)
	}
	if _, err := b.SetTakeoverDate("2026-02-01"); err == nil {
		t.Error("Umstieg im laufenden Geschäftsjahr festgelegt")
	}
	state, err := b.SetTakeoverDate("2026-01-01")
	if err != nil || state.Date != "2026-01-01" || state.FiscalYear != 2026 || state.Booked {
		t.Fatalf("Umstieg: %+v, %v", state, err)
	}
	rec, err := b.GetTakeover()
	if err != nil || rec.Reconciled || len(rec.Differences) != 1 || len(rec.Accounts) != 0 {
		t.Fatalf("Abstimmung vor der Eröffnung: %+v, %v", rec, err)
	}
	if _, err := b.PreviewTakeover(service.TakeoverRequest{}); err == nil ||
		!strings.Contains(err.Error(), "Schlussbilanz") {
		t.Errorf("Vorschau ohne Schlussbilanz: %v", err)
	}
	if _, err := b.BookTakeover(service.TakeoverRequest{}); err == nil {
		t.Error("Eröffnung ohne Schlussbilanz gebucht")
	}
	if _, err := b.SaveFoundation("2026-01-10", domain.Foundation{
		ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{{Name: "Mara Beispiel", ShareCapital: 2_500_000, PaidIn: 1_250_000}},
	}); err == nil || !strings.Contains(err.Error(), "Umstieg") {
		t.Errorf("Gründung neben dem Umstieg: %v", err)
	}
}
