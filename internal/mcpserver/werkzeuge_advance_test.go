package mcpserver

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

func TestAdvanceBankTools(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var inv *domain.Invoice
	h.ui(func(ctx context.Context, s *core.Services) error {
		c := domain.Contact{Type: domain.ContactTypeCustomer, Name: "Abschlag GmbH", Street: "Testweg 1", PostalCode: "10115", City: "Berlin", CountryCode: "DE"}
		if err := s.Contacts.SaveContact(ctx, &c); err != nil {
			return err
		}
		g, err := s.Invoices.CreateInvoiceGroup(ctx, service.AdvanceGroupRequest{Date: "2026-03-01", ContactID: c.ID, Title: "Auftrag", TotalNet: 20000, TaxRate: domain.TaxRateStandard})
		if err != nil {
			return err
		}
		inv, err = s.Invoices.IssueAdvanceInvoice(ctx, service.AdvanceInvoiceRequest{GroupID: g.ID, Date: "2026-03-01", Net: 10000})
		if err != nil {
			return err
		}
		if err := s.Bank.ConfigureAccounts(ctx, []domain.BankAccount{{IBAN: ownIBAN, Name: "Bank", LedgerAccount: "1800", Currency: "EUR"}}); err != nil {
			return err
		}
		_, err = s.Bank.ImportStatement(ctx, []byte(camt(ownIBAN, "", "", "", "", camtEntry{"119.00", "2026-04-15", "advance-receive", c.Name, inv.InvoiceNumber}, camtEntry{"-119.00", "2026-05-15", "advance-refund", c.Name, inv.InvoiceNumber})), "", nil)
		return err
	})
	var open offenePostenOut
	a.ok("offene_posten", nil, &open)
	if len(open.Posten) != 1 || open.Posten[0].AbschlagID != inv.ID {
		t.Fatalf("Abschlag fehlt: %+v", open)
	}
	var bank bankumsaetzeOut
	a.ok("bankumsaetze", map[string]any{"geschaeftsjahr": 2026}, &bank)
	var incoming, outgoing uint
	for _, tx := range bank.Umsaetze {
		if tx.Betrag == Betrag("119.00") {
			incoming = tx.ID
			if len(tx.Vorschlaege) == 0 || tx.Vorschlaege[0].Einstufung != "direkt" {
				t.Fatalf("Routine: %+v", tx)
			}
		} else {
			outgoing = tx.ID
		}
	}
	args := a.args(map[string]any{"bankumsatz_id": incoming, "abschlag_id": inv.ID, "idempotenz_schluessel": "advance-routine"})
	var result buchungOut
	a.ok("bankumsatz_zuordnen", args, &result)
	first := result.Buchung.ID
	a.ok("bankumsatz_zuordnen", args, &result)
	if result.Buchung.ID != first {
		t.Fatal("Wiederholung bucht erneut")
	}
	a.ok("bankumsaetze", map[string]any{"bankumsatz_id": outgoing}, &bank)
	if len(bank.Umsaetze) != 1 || len(bank.Umsaetze[0].Vorschlaege) == 0 || bank.Umsaetze[0].Vorschlaege[0].Einstufung != "rueckfrage" {
		t.Fatalf("Rückzahlung: %+v", bank)
	}
	a.fail("bankumsatz_zuordnen", a.args(map[string]any{"bankumsatz_id": outgoing, "abschlag_id": inv.ID}), CodeApprovalNeeded)
	refund := a.args(map[string]any{"bankumsatz_id": outgoing, "idempotenz_schluessel": "advance-refund", "aufteilung": []map[string]any{{"abschlag_id": inv.ID, "ausgleich": "119.00", "text": "Auftrag entfällt, keine Leistung"}}})
	a.ok("bankumsatz_buchen", refund, nil)
	a.ok("bankumsatz_buchen", refund, nil)
	a.ok("offene_posten", nil, &open)
	if len(open.Posten) != 1 || open.Posten[0].Offen != Betrag("119.00") {
		t.Fatalf("nach Rückzahlung: %+v", open)
	}
	var partialID uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Bank.ImportStatement(ctx, []byte(camt(ownIBAN, "", "", "", "", camtEntry{"59.50", "2026-06-15", "advance-partial", "Abschlag GmbH", inv.InvoiceNumber})), "", nil)
		if err != nil {
			return err
		}
		txs, err := s.Bank.GetTransactions(ctx, 2026)
		if err != nil {
			return err
		}
		for _, tx := range txs {
			if tx.Amount == 5950 {
				partialID = tx.ID
			}
		}
		return nil
	})
	a.fail("bankumsatz_zuordnen", a.args(map[string]any{"bankumsatz_id": partialID, "abschlag_id": inv.ID}), CodeApprovalNeeded)
	partial := a.args(map[string]any{"bankumsatz_id": partialID, "idempotenz_schluessel": "advance-partial", "aufteilung": []map[string]any{{"abschlag_id": inv.ID, "ausgleich": "59.50"}}})
	a.ok("bankumsatz_buchen", partial, nil)
	a.ok("bankumsatz_buchen", partial, nil)
	a.ok("offene_posten", nil, &open)
	if len(open.Posten) != 1 || open.Posten[0].Offen != Betrag("59.50") {
		t.Fatalf("Teilzahlung: %+v", open)
	}

	var restID uint
	h.ui(func(ctx context.Context, s *core.Services) error {
		reference := "Rest " + inv.InvoiceNumber
		_, err := s.Bank.ImportStatement(ctx, []byte(camt(ownIBAN, "", "", "", "", camtEntry{"59.50", "2026-07-15", "advance-rest", "Abschlag GmbH", reference})), "", nil)
		if err != nil {
			return err
		}
		txs, err := s.Bank.GetTransactions(ctx, 2026)
		if err != nil {
			return err
		}
		for _, tx := range txs {
			if tx.RemittanceInfo == reference {
				restID = tx.ID
			}
		}
		return nil
	})
	a.ok("bankumsatz_zuordnen", a.args(map[string]any{"bankumsatz_id": restID, "abschlag_id": inv.ID}), nil)
	a.ok("offene_posten", nil, &open)
	if len(open.Posten) != 0 {
		t.Fatalf("nach Restzahlung: %+v", open)
	}

}
