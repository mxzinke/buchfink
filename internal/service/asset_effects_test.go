package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestDepreciationReversalRestoresRegister(t *testing.T) {
	for _, locked := range []bool{false, true} {
		name := "offen"
		if locked {
			name = "festgeschrieben"
		}
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			svc := env.assets(t)
			env.journal.AddEntryEffects(svc.EntryEffect())
			asset := env.machine(t, svc)
			ctx := context.Background()
			booked, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026})
			if err != nil {
				t.Fatal(err)
			}
			if booked.Total != 300_000 {
				t.Fatalf("AfA: %v", booked.Total)
			}
			if locked {
				repo := repository.NewFestschreibungRepository(env.db)
				env.journal.SetFestschreibungRepo(repo)
				if err := repo.Create(ctx, &domain.Festschreibung{
					FiscalYear: 2026, PeriodType: "year", PeriodLabel: "2026",
					CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash,
				}); err != nil {
					t.Fatal(err)
				}
			}
			_, err = env.journal.Reverse(ctx, booked.Entries[0].ID, "AfA berichtigen")
			if locked && err == nil {
				t.Fatal("Storno trotz Festschreibung")
			}
			if !locked && err != nil {
				t.Fatal(err)
			}
			detail, err := svc.Get(ctx, 2026, asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			wantValue, wantDue := domain.Cents(1_200_000), domain.Cents(300_000)
			if locked {
				wantValue, wantDue = 900_000, 0
			}
			if detail.Asset.BookValue != wantValue || detail.Asset.DueAmount != wantDue {
				t.Fatalf("Buchwert %v, offen %v", detail.Asset.BookValue, detail.Asset.DueAmount)
			}
			if !locked {
				if len(detail.Movements) != 3 {
					t.Fatalf("Bewegungen: %d", len(detail.Movements))
				}
				again, err := svc.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026})
				if err != nil || again.Total != 300_000 {
					t.Fatalf("erneute AfA: %+v, %v", again, err)
				}
			}
		})
	}
}

func TestAssetValuationRejectsCommittedPeriod(t *testing.T) {
	for _, kind := range []string{"impairment", "writeup", "confirm"} {
		t.Run(kind, func(t *testing.T) {
			env := newTestEnv(t)
			svc := env.assets(t)
			ctx := context.Background()
			asset, err := svc.Save(ctx, &domain.FixedAsset{
				Name: "Beteiligung", Class: domain.AssetClassFinancial, Account: "0850",
				AcquisitionDate: "2026-01-01", AcquisitionCost: 1_000_000, Method: domain.DepreciationNone,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.BookImpairment(ctx, ImpairmentRequest{
				AssetID: asset.ID, Date: "2026-09-30", Amount: 200_000, Reason: "Wertverlust", Evidence: "Gutachten",
			}); err != nil {
				t.Fatal(err)
			}
			repo := repository.NewFestschreibungRepository(env.db)
			env.journal.SetFestschreibungRepo(repo)
			if err := repo.Create(ctx, &domain.Festschreibung{
				FiscalYear: 2026, PeriodType: "year", PeriodLabel: "2026", CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash,
			}); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "impairment":
				_, err = svc.BookImpairment(ctx, ImpairmentRequest{
					AssetID: asset.ID, Date: "2026-12-31", Amount: 100_000, Reason: "Wertverlust", Evidence: "Gutachten",
				})
			case "writeup":
				_, err = svc.BookWriteUp(ctx, WriteUpRequest{
					AssetID: asset.ID, Date: "2026-12-31", Amount: 100_000, Reason: "Erholung", Evidence: "Gutachten",
				})
			case "confirm":
				_, err = svc.ConfirmImpairmentPersists(ctx, asset.ID, 2026, "Unverändert", "Gutachten")
			}
			if err == nil {
				t.Fatal("Bewertung trotz Festschreibung")
			}
			detail, err := svc.Get(ctx, 2026, asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			if detail.Asset.BookValue != 800_000 || len(detail.Movements) != 2 {
				t.Fatalf("Kartei verändert: %+v", detail.Asset)
			}
		})
	}
}
