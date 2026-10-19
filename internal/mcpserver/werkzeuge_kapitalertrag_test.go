package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// Eine Dividende kommt als Bankumsatz. Mit kapitalertrag verknüpft Buchfink
// sie mit der Beteiligung, bucht sie brutto auf das Ertragskonto des
// Anlagekontos und trennt Kapitalertragsteuer und Solidaritätszuschlag.
func TestCapitalIncomeIsBookedFromTheBankTransaction(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var assetID, txID uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		if err := s.Bank.ConfigureAccounts(ctx, []domain.BankAccount{
			{IBAN: ownIBAN, Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR"}}); err != nil {
			return err
		}
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Tochter GmbH", Class: domain.AssetClassFinancial, Account: "0850",
			AcquisitionDate: "2026-02-01", AcquisitionCost: 2_500_000, Method: domain.DepreciationNone,
			TaxPrivileged: true, HoldingPermille: 1000,
		})
		if err != nil {
			return err
		}
		assetID = asset.ID
		if _, err := s.Bank.ImportStatement(ctx, []byte(camt(ownIBAN, "", "", "", "",
			camtEntry{"736.25", "2026-05-15", "E2E-DIV", "Tochter GmbH", "Gewinnausschuettung 2025"})), "", nil); err != nil {
			return err
		}
		txs, err := s.Bank.GetTransactions(ctx, 2026)
		if err != nil || len(txs) != 1 {
			t.Fatalf("Bankumsätze: %+v (%v)", txs, err)
		}
		txID = txs[0].ID
		return nil
	})

	income := func(soli string) map[string]any {
		return a.args(map[string]any{"bankumsatz_id": txID, "kapitalertrag": map[string]any{
			"anlage_id": assetID, "brutto": "1000.00", "kapitalertragsteuer": "250.00", "solidaritaetszuschlag": soli,
		}})
	}
	e := a.fail("bankumsatz_buchen", income("10.00"), CodeRule)
	if !strings.Contains(e.Nachricht, "Bankumsatz") || !strings.Contains(e.Nachricht, "736,25") {
		t.Errorf("Abweichung vom Bankumsatz: %+v", e)
	}

	var booked struct {
		Buchung *struct {
			ID uint `json:"id"`
		} `json:"buchung"`
		Status string `json:"status"`
	}
	a.ok("bankumsatz_buchen", income("13.75"), &booked)
	if booked.Buchung == nil || booked.Status != "zugeordnet" {
		t.Fatalf("Kapitalertrag: %+v", booked)
	}
	var entry struct {
		Zeilen []buchungsZeile `json:"zeilen"`
	}
	a.ok("buchung_lesen", map[string]any{"buchung_id": booked.Buchung.ID}, &entry)
	lines := map[string]Betrag{}
	for _, l := range entry.Zeilen {
		lines[l.Seite+":"+l.Konto] = l.Betrag
	}
	for key, want := range map[string]Betrag{"H:7005": "1000.00", "S:7630": "250.00", "S:7633": "13.75", "S:1800": "736.25"} {
		if lines[key] != want {
			t.Errorf("Zeile %s = %q, erwartet %s (alle: %+v)", key, lines[key], want, lines)
		}
	}

	h.ui(func(ctx context.Context, s *core.Services) error {
		detail, err := s.Assets.Get(ctx, 2026, assetID)
		if err != nil {
			return err
		}
		for _, m := range detail.Movements {
			if m.Kind == domain.AssetMovementIncome && m.JournalEntryID != nil && *m.JournalEntryID == booked.Buchung.ID {
				return nil
			}
		}
		t.Errorf("der Ertrag fehlt am Anlagegut: %+v", detail.Movements)
		return nil
	})
}
