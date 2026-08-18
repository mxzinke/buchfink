package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestEBilanzService_ExportXBRL(t *testing.T) {
	ctx := context.Background()
	accSvc, _, _, ebilanzSvc, _ := setupTestServices(t)

	// Create a booking so accounts have non-zero balance
	_, err := accSvc.CreateBooking(ctx, &domain.BookingEntry{
		Description:   "Umsatzerlöse",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        5000.0,
		Currency:      "EUR",
	})
	if err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	xbrl, err := ebilanzSvc.ExportXBRL(ctx)
	if err != nil {
		t.Fatalf("failed to export XBRL: %v", err)
	}

	if !strings.Contains(xbrl, "xbrli:xbrl") {
		t.Fatalf("expected xbrl root tag, got: %s", xbrl)
	}
	if !strings.Contains(xbrl, "de-gcd:genInfo.report.accountScheme") {
		t.Fatalf("expected GCD metadata in XBRL, got: %s", xbrl)
	}
	if !strings.Contains(xbrl, "accountAuditProof") {
		t.Fatalf("expected Kontennachweis audit proof in XBRL, got: %s", xbrl)
	}
}
