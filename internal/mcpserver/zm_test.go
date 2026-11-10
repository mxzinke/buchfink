package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

func TestZMExportThroughHarness(t *testing.T) {
	h := newHarness(t)
	a := h.agent("")
	var id uint
	h.completeCompany()
	h.ui(func(ctx context.Context, s *core.Services) error {
		c := domain.Contact{Type: domain.ContactTypeCustomer, Name: "Client", CountryCode: "FR", VatID: "FR12345678901"}
		if err := s.Contacts.SaveContact(ctx, &c); err != nil {
			return err
		}
		_, err := s.Journal.Post(ctx, &domain.JournalEntry{BookingDate: "2026-02-10", DocumentDate: "2026-02-10", Description: "EU-Leistung", TaxTreatment: domain.TaxTreatmentReverseChargeSupply, ContactID: &c.ID,
			Lines: []domain.JournalLine{{Account: c.LedgerAccount, Side: domain.SideDebit, Amount: 123456}, {Account: "4337", Side: domain.SideCredit, Amount: 123456}}})
		if err != nil {
			return err
		}
		rec, err := s.ZM.Save(ctx, "2026-Q1")
		if err == nil {
			id = rec.ID
		}
		return err
	})
	var exp exportOut
	a.ok("erklaerung_exportieren", a.args(map[string]any{"art": "zm", "meldung_id": id}), &exp)
	if exp.Format != "csv" || filepath.Dir(exp.Pfad) != filepath.Join(h.tenant.DataDir, "Export") || len(exp.SHA256) != 64 {
		t.Fatalf("Export: %+v", exp)
	}
	data, err := os.ReadFile(exp.Pfad)
	if err != nil || !strings.Contains(string(data), "FR12345678901,1234,S\n") {
		t.Fatalf("CSV %q: %v", data, err)
	}
	var view erklaerungOut
	a.ok("erklaerung", map[string]any{"art": "zm", "zeitraum": "2026-Q1"}, &view)
	if view.MeldungID != id || view.Berichtigung || len(view.ZM) != 1 || view.ZM[0].UstID != "FR12345678901" {
		t.Fatalf("Berichtigung: %+v", view)
	}
}
