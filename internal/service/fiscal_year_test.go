package service_test

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

func TestFiscalYear_DeviatingAndAutomaticAssignment(t *testing.T) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	accRepo := repository.NewAccountRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// Set deviating fiscal year starting in July (startMonth = 7)
	settings := &domain.CompanySettings{
		CompanyName:          "Deviating FY Corp",
		FiscalYearStartMonth: 7,
		FiscalYear:           2025,
		SKR:                  "SKR04",
	}
	if err := settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}

	accSvc := service.NewAccountingService(accRepo, bookingRepo, settingsRepo, auditRepo, 2025)

	// Test 1: Date in August 2025 -> should belong to FY 2025 (2025/2026)
	b1 := &domain.BookingEntry{
		Date:          "2025-08-15",
		Description:   "Erlöse August 2025",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        1000.0,
	}
	created1, err := accSvc.CreateBooking(ctx, b1)
	if err != nil {
		t.Fatalf("failed to create booking 1: %v", err)
	}
	if created1.FiscalYear != 2025 {
		t.Fatalf("expected fiscal year 2025, got %d", created1.FiscalYear)
	}

	// Test 2: Date in March 2026 -> with startMonth=7, month 3 < 7 -> belongs to FY 2025 (2025/2026)
	b2 := &domain.BookingEntry{
		Date:          "2026-03-10",
		Description:   "Erlöse März 2026",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        2000.0,
	}
	created2, err := accSvc.CreateBooking(ctx, b2)
	if err != nil {
		t.Fatalf("failed to create booking 2: %v", err)
	}
	if created2.FiscalYear != 2025 {
		t.Fatalf("expected fiscal year 2025 for March 2026 (startMonth 7), got %d", created2.FiscalYear)
	}

	// Test 3: Date in July 2026 -> month 7 >= 7 -> belongs to FY 2026 (2026/2027)
	b3 := &domain.BookingEntry{
		Date:          "2026-07-01",
		Description:   "Erlöse Juli 2026",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        3000.0,
	}
	created3, err := accSvc.CreateBooking(ctx, b3)
	if err != nil {
		t.Fatalf("failed to create booking 3: %v", err)
	}
	if created3.FiscalYear != 2026 {
		t.Fatalf("expected fiscal year 2026 for July 2026, got %d", created3.FiscalYear)
	}

	// Test 4: Dynamic Discovery of Fiscal Years from bookings
	years := accSvc.GetAvailableFiscalYears(ctx)
	if len(years) < 2 {
		t.Fatalf("expected at least 2 available fiscal years, got %v", years)
	}
	has2025 := false
	has2026 := false
	for _, y := range years {
		if y == 2025 {
			has2025 = true
		}
		if y == 2026 {
			has2026 = true
		}
	}
	if !has2025 || !has2026 {
		t.Fatalf("expected years to include 2025 and 2026, got %v", years)
	}

	// Test 5: Filtering bookings by fiscal year
	accSvc.SetFiscalYear(2025)
	bks2025, err := accSvc.GetBookings(ctx)
	if err != nil || len(bks2025) != 2 {
		t.Fatalf("expected 2 bookings in FY 2025, got %d", len(bks2025))
	}

	accSvc.SetFiscalYear(2026)
	bks2026, err := accSvc.GetBookings(ctx)
	if err != nil || len(bks2026) != 1 {
		t.Fatalf("expected 1 booking in FY 2026, got %d", len(bks2026))
	}
}

func TestFiscalYear_TransitionPeriodOverride(t *testing.T) {
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	accRepo := repository.NewAccountRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	accSvc := service.NewAccountingService(accRepo, bookingRepo, settingsRepo, auditRepo, 2026)

	// Booking created in January 2026 but user explicitly overrides fiscal year to 2025 (Abschlussbuchung)
	b := &domain.BookingEntry{
		Date:          "2026-01-10",
		FiscalYear:    2025, // explicit user override
		Description:   "Rechnungsabgrenzung Vorjahr",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        500.0,
	}

	created, err := accSvc.CreateBooking(ctx, b)
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}
	if created.FiscalYear != 2025 {
		t.Fatalf("expected explicit fiscal year 2025 to be preserved, got %d", created.FiscalYear)
	}
}
