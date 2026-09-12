package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestUGReserveIsInProfitYearAndCarriedWithoutReducingProfit(t *testing.T) {
	e := newTestEnv(t)
	m := e.closingModules(t)
	ctx := context.Background()
	e.setSetting(t, "legal_form", domain.LegalFormUG)
	e.post(t, "2026-01-15", "1800", "2900", 500000)
	e.post(t, "2026-06-30", "1800", "4400", 678212)
	before, err := m.closing.LegalReserve(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if before.Required != 169553 || before.Difference != 169553 {
		t.Fatalf("reserve: %+v", before)
	}
	entry, err := m.closing.BookLegalReserve(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if entry.BookingDate != "2026-12-31" || entry.FiscalYear != 2026 || entry.ReceiptID == nil {
		t.Fatalf("reserve date or evidence: %+v", entry)
	}
	if _, err := m.closing.BookLegalReserve(ctx, 2026); err == nil {
		t.Fatal("reserve booked twice")
	}
	fs, err := e.statements(t).Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if fs.Statement.NetIncome != 678212 || fs.Statement.TotalAssets != fs.Statement.TotalLiabilities {
		t.Fatalf("profit or balance changed: %+v", fs.Statement)
	}
	line := fs.Statement.Line("passiva.A.bilanzgewinn")
	if line == nil || line.Amount != 508659 || line.Omitted {
		t.Fatalf("Bilanzgewinn: %+v", line)
	}
	if got := balances(t, e, 2026)[domain.AccountGesetzlicheRuecklage]; got != -169553 {
		t.Fatalf("reserve absent in profit year: %d", got)
	}
	summary, err := e.accounting.GetFinancialSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.NetIncome != 678212 {
		t.Fatalf("dashboard profit reduced by reserve: %+v", summary)
	}
	carry, err := m.closing.CarryForwardState(ctx, 2027)
	if err != nil || carry.ResultToCarry != 508659 {
		t.Fatalf("remaining result in carry-forward preview: %+v, %v", carry, err)
	}
	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatal(err)
	}
	carried := balances(t, e, 2027)
	if carried[domain.AccountGesetzlicheRuecklage] != -169553 || carried[domain.AccountGewinnvortrag] != -508659 || carried[legalReserveExpenseAccount] != 0 {
		t.Fatalf("wrong opening balances: %+v", carried)
	}
	preview, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{DecisionDate: "2027-05-20"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.RequiredLegalReserve != 0 || preview.ReservedInClosing != 169553 || preview.NetIncome != 508659 {
		t.Fatalf("double allocation: %+v", preview)
	}
}

func TestUGReserveLossCarryforwardAndCapitalThreshold(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		capital, loss, income, want domain.Cents
	}{
		{"loss carryforward", 500000, 200000, 678212, 119553},
		{"loss exceeds profit", 500000, 700000, 678212, 0},
		{"no profit", 500000, 0, 0, 0},
		{"capital reaches threshold", 2500000, 0, 678212, 0},
		{"cent rounding", 500000, 0, 5, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			m := e.closingModules(t)
			ctx := context.Background()
			e.setSetting(t, "legal_form", domain.LegalFormUG)
			e.post(t, "2026-01-01", "1800", "2900", tc.capital)
			if tc.loss > 0 {
				e.post(t, "2026-01-01", domain.AccountVerlustvortrag, "1800", tc.loss)
			}
			if tc.income > 0 {
				e.post(t, "2026-06-30", "1800", "4400", tc.income)
			}
			r, err := m.closing.LegalReserve(ctx, 2026)
			if err != nil {
				t.Fatal(err)
			}
			if r.Required != tc.want {
				t.Fatalf("got %d, want %d: %+v", r.Required, tc.want, r)
			}
		})
	}
}
