package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestProvisionContinuationMirrorAndReversal(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	p, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Abschluss",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot Kanzlei",
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err = m.provisions.BookIncrease(ctx, ProvisionRequest{
		ProvisionID: p.ID, FiscalYear: 2027, Date: "2027-03-31",
		Amount: 50_000, Reason: "Ergänztes Angebot",
	})
	if err != nil {
		t.Fatal(err)
	}
	req := ProvisionChangeRequest{ProvisionID: p.ID, Date: "2027-04-30", Amount: 100_000, Reason: "Leistungsumfang laut Kanzlei entfallen"}
	preview, err := m.provisions.PreviewChange(ctx, domain.ProvisionRelease, req)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Amount != 100_000 || preview.Entry.Lines[1].Account != "4930" {
		t.Fatalf("%+v", preview)
	}
	p, err = m.provisions.BookChange(ctx, domain.ProvisionRelease, req)
	if err != nil {
		t.Fatal(err)
	}
	first, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.provisions.Mirror(ctx, 2027)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total.Closing != 300_000 || second.Total.Opening != 300_000 ||
		second.Total.Additions != 50_000 || second.Total.Released != 100_000 || second.Total.Closing != 250_000 {
		t.Fatalf("Spiegel: %+v / %+v", first.Total, second.Total)
	}
	if _, err := env.journal.Reverse(ctx, *p.Movements[len(p.Movements)-1].JournalEntryID, "Umfang bleibt bestehen"); err != nil {
		t.Fatal(err)
	}
	second, err = m.provisions.Mirror(ctx, 2027)
	if err != nil {
		t.Fatal(err)
	}
	if second.Total.Released != 0 || second.Total.Closing != 350_000 {
		t.Fatalf("Storno: %+v", second.Total)
	}
}

func TestProvisionTwoYearsOfUnwinding(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	p, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Rückbau",
		Amount: 1_000_000, ExpectedOn: "2029-12-31", Reason: "Mietvertrag",
		RateEvidence: &ProvisionRateEvidence{
			Rate:   ProvisionRate{Month: "2026-12", Years: 3, Average: 7, RateMicros: 15_000},
			Source: "https://www.bundesbank.de/de/statistiken/zinssaetze",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		date    string
		balance domain.Cents
	}{
		{"2027-12-31", 970_662}, {"2028-12-31", 985_222},
	} {
		p, err = m.provisions.BookUnwinding(ctx, ProvisionChangeRequest{ProvisionID: p.ID, Date: tc.date, Reason: "Zeitablauf laut Mietvertrag"})
		if err != nil {
			t.Fatal(err)
		}
		if p.Balance() != tc.balance {
			t.Fatalf("%s: %d statt %d", tc.date, p.Balance(), tc.balance)
		}
		if _, err := m.provisions.BookUnwinding(ctx, ProvisionChangeRequest{ProvisionID: p.ID, Date: tc.date, Reason: "Doppelter Lauf"}); err == nil {
			t.Fatal("doppelte Aufzinsung")
		}
	}
}

func TestProvisionConsumptionInFollowingYear(t *testing.T) {
	for _, net := range []domain.Cents{200_000, 300_000, 350_000} {
		t.Run(net.String(), func(t *testing.T) {
			env := newTestEnv(t)
			m := env.closingModules(t)
			ctx := context.Background()
			vendor := env.vendor(t, "Kanzlei", "DE", "")
			p, err := m.provisions.BookFormation(ctx, ProvisionRequest{
				FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Abschluss",
				Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot",
			})
			if err != nil {
				t.Fatal(err)
			}
			req := env.receipt(t, vendor.ID, "", net, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
			req.BookingDate, req.DocumentDate = "2027-03-10", "2027-03-10"
			req.ServiceDateFrom, req.ServiceDateTo = "2027-03-01", "2027-03-10"
			req.ProvisionID = p.ID
			req.Positions = []ReceiptPosition{{Account: domain.AccountAbschlusskosten, Net: net, TaxRate: domain.TaxRateStandard}}
			entry, err := env.posting.PostIncomingReceipt(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			debit := map[string]domain.Cents{}
			for _, line := range entry.Lines {
				if line.Side == domain.SideDebit {
					debit[line.Account] += line.Amount
				}
			}
			covered := min(net, 300_000)
			if debit[domain.AccountRueckstellungAbschluss] != covered || debit[domain.AccountAbschlusskosten] != net-covered {
				t.Fatalf("Buchung: %+v", debit)
			}
			mirror, err := m.provisions.Mirror(ctx, 2027)
			if err != nil {
				t.Fatal(err)
			}
			if mirror.Total.Opening != 300_000 || mirror.Total.Used != covered || mirror.Total.Closing != 300_000-covered {
				t.Fatalf("Spiegel: %+v", mirror.Total)
			}
		})
	}
}

func TestProvisionMovementRejectsCommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	p, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Abschluss",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.NewFestschreibungRepository(env.db).Create(ctx, &domain.Festschreibung{
		FiscalYear: 2027, PeriodType: domain.PeriodTypeYear, PeriodLabel: "Geschäftsjahr 2027",
		CutoffDate: "2027-12-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.provisions.BookChange(ctx, domain.ProvisionRelease, ProvisionChangeRequest{
		ProvisionID: p.ID, Date: "2027-06-30", Reason: "Verpflichtung laut Kanzlei entfallen",
	}); err == nil {
		t.Fatal("Auflösung trotz Festschreibung")
	}
	mirror, err := m.provisions.Mirror(ctx, 2027)
	if err != nil {
		t.Fatal(err)
	}
	if mirror.Total.Closing != 300_000 || mirror.Total.Released != 0 {
		t.Fatalf("%+v", mirror.Total)
	}
}
