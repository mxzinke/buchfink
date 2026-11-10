package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

type stubRevenue map[int]domain.Cents

func (s stubRevenue) RevenueOf(_ context.Context, year int) (domain.Cents, error) {
	return s[year], nil
}

func openReceivable(id uint, open domain.Cents, due string) domain.OpenItem {
	return domain.OpenItem{
		Source: domain.OpenItemSourceJournal, EntryID: id, EntryNumber: "B-" + due,
		ContactType: domain.ContactTypeCustomer, ContactName: "Kunde GmbH", DueDate: due,
		GrossAmount: open, OpenAmount: open,
		TaxRate: domain.TaxRateStandard, TaxTreatment: domain.TaxTreatmentDomestic,
	}
}

// Der Baustein bucht die Veränderung auf den Sollbestand: 2026 die Bildung,
// 2027 nach dem Saldenvortrag nur die Differenz.
//
//	2026: A 11.900 € seit 138 Tagen überfällig, EWB 50 % von 10.000 € = 5.000 €;
//	      B 5.950 € nicht fällig, PWB 1 % von 5.000 € = 50 €.
//	2027: A fällt ganz aus (EWB 100 % = 10.000 €, Zuführung 5.000 €);
//	      B ist bezahlt, PWB 0 € (Herabsetzung 50 €).
func TestReceivableAllowanceBooksTheChangeOnTheStock(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	// Die Ausfälle des Vorjahres: 300 € auf 100.000 € Umsatz → 3 ‰.
	env.post(t, "2025-06-30", "6936", "1800", 30_000)
	m.bookings.SetRevenueSource(stubRevenue{2025: 10_000_000})
	m.bookings.SetOpenItemSource(stubOpenItems{
		openReceivable(1, 1_190_000, "2026-08-15"),
		openReceivable(2, 595_000, "2027-01-31"),
	})

	req := ReceivableAllowanceRequest{
		FiscalYear: 2026, GeneralPermille: 10,
		Specific: []accounting.SpecificAllowance{{EntryID: 1, Percent: 50, Reason: "Mahnbescheid ohne Zahlung"}},
	}
	preview, err := m.bookings.PreviewReceivableAllowance(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ProposedPermille != 3 || len(preview.LossHistory) != 1 {
		t.Errorf("vorgeschlagene Quote %d ‰ aus %+v", preview.ProposedPermille, preview.LossHistory)
	}
	if len(preview.Lines) != 4 {
		t.Fatalf("Buchungssatz: %+v", preview.Lines)
	}
	if _, err := m.bookings.BookReceivableAllowance(ctx, ReceivableAllowanceRequest{
		FiscalYear: 2026, Specific: []accounting.SpecificAllowance{{EntryID: 1, Percent: 50}},
	}); err == nil || !strings.Contains(err.Error(), "Grund") {
		t.Errorf("eine Einzelwertberichtigung ohne Grund: %v", err)
	}
	if _, err := m.bookings.BookReceivableAllowance(ctx, req); err != nil {
		t.Fatal(err)
	}
	after := balances(t, env, 2026)
	if after[accounting.AccountEWBShortTerm] != -500_000 || after[accounting.AccountPWBShortTerm] != -5_000 ||
		after[accounting.AccountEWBExpense] != 500_000 || after[accounting.AccountPWBExpense] != 5_000 {
		t.Errorf("nach der Bildung: %v", after)
	}
	if _, err := m.bookings.BookReceivableAllowance(ctx, req); err == nil {
		t.Error("ein zweiter Lauf hätte die Wertberichtigung ein zweites Mal gebucht")
	}
	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps.Steps {
		if step.Key == domain.ClosingStepReceivableAllowance && step.State != domain.ClosingStepDone {
			t.Errorf("Schritt Wertberichtigung: %s", step.State)
		}
	}

	// 2027: ohne Saldenvortrag ist der Bestand nicht bekannt.
	m.bookings.SetOpenItemSource(stubOpenItems{openReceivable(1, 1_190_000, "2026-08-15")})
	next := ReceivableAllowanceRequest{
		FiscalYear: 2027, GeneralPermille: 10,
		Specific: []accounting.SpecificAllowance{{EntryID: 1, Percent: 100, Reason: "Insolvenzantrag"}},
	}
	if _, err := m.bookings.PreviewReceivableAllowance(ctx, next); err == nil ||
		!strings.Contains(err.Error(), "Saldenvortrag") {
		t.Fatalf("ohne Saldenvortrag: %v", err)
	}
	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	if _, err := m.bookings.BookReceivableAllowance(ctx, next); err != nil {
		t.Fatal(err)
	}
	later := balances(t, env, 2027)
	if later[accounting.AccountEWBShortTerm] != -1_000_000 || later[accounting.AccountPWBShortTerm] != 0 {
		t.Errorf("Bestände 2027: 1246 %s €, 1248 %s €",
			later[accounting.AccountEWBShortTerm], later[accounting.AccountPWBShortTerm])
	}
	if later[accounting.AccountEWBExpense] != 500_000 || later[accounting.AccountPWBIncome] != -5_000 {
		t.Errorf("Veränderung 2027: 6923 %s €, 4920 %s €",
			later[accounting.AccountEWBExpense], later[accounting.AccountPWBIncome])
	}
}
