package accounting_test

import (
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/models"
)

func TestHashChainIntegrity(t *testing.T) {
	// Genesis booking
	b1 := models.BookingEntry{
		ID:            1,
		BookingNumber: "B-2024-0001",
		Date:          "2024-01-15",
		ValueDate:     "2024-01-15",
		Description:   "Zahlungseingang Kunde",
		DebitAccount:  "1800",
		CreditAccount: "4400",
		Amount:        5950.00,
		Currency:      "EUR",
		TaxCode:       "UST19",
		ReceiptHash:   "abc123hash",
		PreviousHash:  accounting.GenesisHash,
		CreatedAt:     time.Now(),
	}
	b1.EntryHash = accounting.CalculateEntryHash(&b1, b1.PreviousHash)

	// Second booking chained to b1
	b2 := models.BookingEntry{
		ID:            2,
		BookingNumber: "B-2024-0002",
		Date:          "2024-01-20",
		ValueDate:     "2024-01-20",
		Description:   "Büromiete Januar",
		DebitAccount:  "6500",
		CreditAccount: "1800",
		Amount:        650.00,
		Currency:      "EUR",
		TaxCode:       "NONE",
		ReceiptHash:   "def456hash",
		PreviousHash:  b1.EntryHash,
		CreatedAt:     time.Now(),
	}
	b2.EntryHash = accounting.CalculateEntryHash(&b2, b2.PreviousHash)

	entries := []models.BookingEntry{b1, b2}

	// 1. Verify valid chain
	res := accounting.VerifyChain(entries)
	if !res.IsValid {
		t.Fatalf("expected valid chain, got error: %s", res.Message)
	}
	if res.CheckedEntries != 2 {
		t.Fatalf("expected 2 checked entries, got %d", res.CheckedEntries)
	}

	// 2. Tamper with amount of b1 (simulation of database manipulation)
	tamperedEntries := []models.BookingEntry{b1, b2}
	tamperedEntries[0].Amount = 1000.00 // Changed amount!

	tamperedRes := accounting.VerifyChain(tamperedEntries)
	if tamperedRes.IsValid {
		t.Fatalf("expected tampered chain to be invalid, but got valid")
	}
	if tamperedRes.FirstBrokenID == nil || *tamperedRes.FirstBrokenID != 1 {
		t.Fatalf("expected first broken ID to be 1, got %v", tamperedRes.FirstBrokenID)
	}
}

func TestSKR04Defaults(t *testing.T) {
	accounts := accounting.DefaultSKR04Accounts()
	if len(accounts) == 0 {
		t.Fatal("expected non-empty default SKR04 accounts")
	}

	foundBank := false
	for _, a := range accounts {
		if a.Number == "1800" {
			foundBank = true
			if a.Type != "asset" {
				t.Fatalf("expected bank to be asset, got %s", a.Type)
			}
		}
	}

	if !foundBank {
		t.Fatal("expected account 1800 (Bank) in SKR04 defaults")
	}
}
