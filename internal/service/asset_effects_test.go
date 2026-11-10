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
