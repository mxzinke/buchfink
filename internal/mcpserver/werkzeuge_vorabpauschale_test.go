package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// Ein ETF kommt mit seiner Fondsart in die Kartei. Die Vorabpauschale
// rechnet Buchfink aus den Rücknahmepreisen und dem hinterlegten Basiszins und
// hält sie ohne Buchung am Anlagegut fest, einmal je Kalenderjahr.
func TestVorabpauschaleIsRecordedAtTheFund(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var purchase uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		entry, err := s.Journal.Post(ctx, &domain.JournalEntry{BookingDate: "2026-01-15", DocumentDate: "2026-01-15",
			ServiceDateFrom: "2026-01-15", ServiceDateTo: "2026-01-15", Description: "Kauf Aktien-ETF Welt",
			Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "0900", Amount: 5_000_000},
				{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 5_000_000}}})
		if err != nil {
			return err
		}
		purchase = entry.ID
		return nil
	})

	run := func(key string, preview bool, in map[string]any) map[string]any {
		args := a.args(map[string]any{"art": key, "geschaeftsjahr": 2026, "idempotenz_schluessel": key + "-" + in["schluessel"].(string)})
		delete(in, "schluessel")
		for k, v := range in {
			args[k] = v
		}
		if preview {
			args["nur_vorschau"] = true
		}
		return args
	}
	var activated abschlussAntwort
	a.ok("abschlussbuchung", run("anlage_aktivieren", false, map[string]any{"schluessel": "etf", "anlage": map[string]any{
		"buchung_id": purchase, "bezeichnung": "Aktien-ETF Welt", "klasse": "finanzanlage", "methode": "keine",
		"fondsart": "aktienfonds"}}), &activated)
	if activated.AnlageID == nil {
		t.Fatalf("anlage_aktivieren: %+v", activated)
	}

	figures := func() map[string]any {
		return map[string]any{"anlage_id": *activated.AnlageID, "kalenderjahr": 2026,
			"wert_jahresbeginn": "50000.00", "wert_jahresende": "54000.00"}
	}
	var preview, recorded abschlussAntwort
	a.ok("abschlussbuchung", run("vorabpauschale", true, map[string]any{"schluessel": "2026", "vorabpauschale": figures()}), &preview)
	// 50.000,00 € × 70 % × 3,20 % = 1.120,00 €; der Wertzuwachs von 4.000,00 € begrenzt nicht.
	if !preview.Vorschau || preview.Betrag == nil || *preview.Betrag != "1120.00" || len(preview.BuchungIDs) != 0 {
		t.Fatalf("Vorschau: %+v", preview)
	}
	a.ok("abschlussbuchung", run("vorabpauschale", false, map[string]any{"schluessel": "2026", "vorabpauschale": figures()}), &recorded)
	if recorded.Vorschau || recorded.Betrag == nil || *recorded.Betrag != "1120.00" || len(recorded.BuchungIDs) != 0 {
		t.Fatalf("festgehalten: %+v", recorded)
	}
	e := a.fail("abschlussbuchung", run("vorabpauschale", false, map[string]any{"schluessel": "2026-zweimal", "vorabpauschale": figures()}), CodeRule)
	if !strings.Contains(e.Nachricht, "bereits") {
		t.Errorf("zweite Vorabpauschale desselben Jahres: %+v", e)
	}
	before := figures()
	before["kalenderjahr"] = 2025
	if e := a.fail("abschlussbuchung", run("vorabpauschale", true, map[string]any{"schluessel": "2025", "vorabpauschale": before}), CodeRule); !strings.Contains(e.Nachricht, "2025") {
		t.Errorf("Vorabpauschale vor dem Erwerb: %+v", e)
	}

	h.ui(func(ctx context.Context, s *core.Services) error {
		detail, err := s.Assets.Get(ctx, 2026, *activated.AnlageID)
		if err != nil {
			return err
		}
		found := false
		for _, m := range detail.Movements {
			found = found || (m.Kind == domain.AssetMovementVorabpauschale && m.TaxAmount == 112_000 && m.JournalEntryID == nil)
		}
		if detail.Asset.FundClass != "equity" || !found {
			t.Errorf("Fondsart %q, Bewegungen %+v", detail.Asset.FundClass, detail.Movements)
		}
		return nil
	})
}
