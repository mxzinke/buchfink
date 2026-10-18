package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestGoldenClosingSteps(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		year  int
		setup func(*testing.T, *testEnv, *closingModules)
	}{
		{"leeres_jahr_2025", 2025, nil},
		{"leeres_jahr_2027", 2027, nil},
		{"afa_faellig", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			assets := e.assets(t)
			m.steps.SetDepreciationSource(assets)
			m.steps.SetWriteUpSource(assets)
			_, err := assets.Save(ctx, &domain.FixedAsset{
				Name: "Maschine", Class: domain.AssetClassTangible,
				AcquisitionDate: "2026-01-15", AcquisitionCost: 1_200_000, UsefulLifeMonths: 120,
				Method: domain.DepreciationLinear, Account: "0440", DepreciationAccount: "6220",
			})
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"rueckstellungen", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			_, err := m.provisions.BookFormation(ctx, ProvisionRequest{
				FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
				Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
			})
			if err != nil {
				t.Fatal(err)
			}
			e.taxableProfit(t)
			if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{FiscalYear: 2026}); err != nil {
				t.Fatal(err)
			}
		}},
		{"uebersprungen_und_abgehakt", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			for _, step := range []domain.ClosingStep{
				{Year: 2026, Key: domain.ClosingStepProvisions, State: domain.ClosingStepSkipped, Reason: "Keine ungewissen Verbindlichkeiten", ChangedOn: "2027-03-01"},
				{Year: 2026, Key: domain.ClosingStepAccruals, State: domain.ClosingStepDone, ChangedOn: "2027-03-02"},
				{Year: 2026, Key: domain.ClosingStepDepreciation, State: domain.ClosingStepDone, ChangedOn: "2027-03-02"},
			} {
				if err := repository.NewClosingStepRepository(e.db).Save(ctx, &step); err != nil {
					t.Fatal(err)
				}
			}
			m.steps.SetDepreciationSource(depreciationStub{due: []DepreciationDue{{AssetID: 1, Due: 120_000}}})
		}},
		{"aufgestellt", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			if _, err := m.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-31", ""); err != nil {
				t.Fatal(err)
			}
		}},
		{"festgestellt", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			m.adopt(t, e, 2026, "2027-05-20")
		}},
		{"offengelegt", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			m.adopt(t, e, 2026, "2027-05-20")
			if _, err := m.closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearDisclosed, "2027-06-30", "Offenlegung"); err != nil {
				t.Fatal(err)
			}
		}},
		{"ug_mit_ruecklage", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			e.setSetting(t, "legal_form", domain.LegalFormUG)
			if _, err := e.journal.Post(ctx, datedEntry("2026-06-30", domain.AccountBank, "4400", 400_000)); err != nil {
				t.Fatal(err)
			}
			if _, err := m.closing.BookLegalReserve(ctx, 2026); err != nil {
				t.Fatal(err)
			}
			m.adopt(t, e, 2026, "2027-05-20")
			if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
				t.Fatal(err)
			}
			if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
				DecisionDate: "2027-05-20", Distribution: 200_000, Text: "Gesellschafterbeschluss",
			}); err != nil {
				t.Fatal(err)
			}
		}},
		{"rumpfjahr", 2026, func(t *testing.T, e *testEnv, m *closingModules) {
			if err := repository.NewFiscalYearRepository(e.db).Save(ctx, &domain.FiscalYear{
				Year: 2026, StartDate: "2026-03-15", EndDate: "2026-06-30", IsShort: true, Status: domain.FiscalYearOpen,
			}); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			m := e.closingModules(t)
			if tc.setup != nil {
				tc.setup(t, e, m)
			}
			steps, err := m.steps.Steps(ctx, tc.year)
			if err != nil {
				t.Fatal(err)
			}
			assertGolden(t, "closing_steps_"+tc.name, steps)
		})
	}
}
