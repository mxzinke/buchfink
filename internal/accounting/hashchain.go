package accounting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/models"
)

// GenesisHash is the root anchor for the first booking entry in a fiscal year.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// CalculateEntryHash generates an immutable cryptographic SHA256 hash
// linking the current transaction data with the previous hash.
func CalculateEntryHash(b *models.BookingEntry, prevHash string) string {
	payload := fmt.Sprintf(
		"NR:%s|D:%s|VD:%s|DEB:%s|CRE:%s|AMT:%.2f|CUR:%s|TX:%s|RH:%s|PREV:%s|ST:%t|STID:%v",
		b.BookingNumber,
		b.Date,
		b.ValueDate,
		b.DebitAccount,
		b.CreditAccount,
		b.Amount,
		b.Currency,
		b.TaxCode,
		b.ReceiptHash,
		prevHash,
		b.IsStorno,
		b.StornoForID,
	)
	hashBytes := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hashBytes[:])
}

// VerifyChain checks the cryptographic integrity of a list of booking entries.
func VerifyChain(entries []models.BookingEntry) models.IntegrityCheckResult {
	if len(entries) == 0 {
		return models.IntegrityCheckResult{
			IsValid:          true,
			TotalEntries:     0,
			CheckedEntries:   0,
			Message:          "Keine Buchungen vorhanden. Die Buchhaltung ist bereit.",
			LastVerifiedHash: GenesisHash,
			CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
		}
	}

	currentExpectedPrev := GenesisHash

	for i, entry := range entries {
		// Check 1: Does the entry reference the correct previous hash?
		if entry.PreviousHash != currentExpectedPrev {
			brokenID := entry.ID
			return models.IntegrityCheckResult{
				IsValid:        false,
				TotalEntries:   len(entries),
				CheckedEntries: i,
				FirstBrokenID:  &brokenID,
				Message: fmt.Sprintf(
					"Unstimmigkeit bei Buchung %s (ID %d): Vorgänger-Referenz weicht ab.",
					entry.BookingNumber, entry.ID,
				),
				LastVerifiedHash: currentExpectedPrev,
				CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
			}
		}

		// Check 2: Does recalculating the hash match the stored entryHash?
		calculated := CalculateEntryHash(&entry, entry.PreviousHash)
		if calculated != entry.EntryHash {
			brokenID := entry.ID
			return models.IntegrityCheckResult{
				IsValid:        false,
				TotalEntries:   len(entries),
				CheckedEntries: i,
				FirstBrokenID:  &brokenID,
				Message: fmt.Sprintf(
					"Unstimmigkeit bei Buchung %s (ID %d): Daten wurden nach der Buchung verändert.",
					entry.BookingNumber, entry.ID,
				),
				LastVerifiedHash: currentExpectedPrev,
				CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
			}
		}

		currentExpectedPrev = entry.EntryHash
	}

	return models.IntegrityCheckResult{
		IsValid:          true,
		TotalEntries:     len(entries),
		CheckedEntries:   len(entries),
		Message:          fmt.Sprintf("Alle %d Buchungen sind vollständig und unverändert.", len(entries)),
		LastVerifiedHash: currentExpectedPrev,
		CheckedAt:        time.Now().Format("02.01.2006 15:04:05"),
	}
}
