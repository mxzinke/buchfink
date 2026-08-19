package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

func newBooking(date string) *domain.BookingEntry {
	return &domain.BookingEntry{
		FiscalYear:    2026,
		Date:          date,
		Description:   "Test",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        100,
	}
}

func TestFestschreibungBlocksBackdatedBookings(t *testing.T) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("in-memory db: %v", err)
	}
	bookingRepo := repository.NewBookingRepository(db)
	fsRepo := repository.NewFestschreibungRepository(db)

	svc := service.NewAccountingService(nil, bookingRepo, nil, nil, 2026)
	svc.SetFestschreibungRepo(fsRepo)

	// A booking in January, entered before the period is committed.
	if _, err := svc.CreateBooking(ctx, newBooking("2026-01-15")); err != nil {
		t.Fatalf("initial booking should succeed: %v", err)
	}

	// Commit (festschreiben) January.
	if err := fsRepo.Create(ctx, &domain.Festschreibung{
		FiscalYear:      2026,
		PeriodType:      "month",
		PeriodLabel:     "Januar 2026",
		CutoffDate:      "2026-01-31",
		ChainHead:       "dummy",
		TimestampStatus: "pending",
		CreatedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("commit period: %v", err)
	}

	// A new booking backdated into the committed period must be rejected.
	if _, err := svc.CreateBooking(ctx, newBooking("2026-01-20")); err == nil {
		t.Fatal("expected backdated booking into committed period to be rejected")
	}

	// A booking after the cutoff is still allowed.
	if _, err := svc.CreateBooking(ctx, newBooking("2026-02-05")); err != nil {
		t.Fatalf("booking after cutoff should succeed: %v", err)
	}

	// Storno of the original January booking must stay allowed (dated today, the
	// legal correction path for closed periods).
	if _, err := svc.StornoBooking(ctx, 1, "Korrektur"); err != nil {
		t.Fatalf("storno of a committed-period booking should be allowed: %v", err)
	}
}

func TestNoFestschreibungAllowsAllDates(t *testing.T) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("in-memory db: %v", err)
	}
	svc := service.NewAccountingService(nil, repository.NewBookingRepository(db), nil, nil, 2026)
	svc.SetFestschreibungRepo(repository.NewFestschreibungRepository(db))

	// Without any commitment, any date is fine.
	if _, err := svc.CreateBooking(ctx, newBooking("2026-01-02")); err != nil {
		t.Fatalf("booking without commitment should succeed: %v", err)
	}
}
