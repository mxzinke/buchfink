// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package accounting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// HashChain computes and verifies the GoBD hash chain over journal entries.
//
// The chain only proves what it covers. The previous implementation hashed a
// subset of fields, which left the Buchungstext and the tax amount silently
// editable — the Unveränderbarkeit claim did not actually hold. This version
// covers every field that carries accounting meaning, including all lines.
type HashChain struct{}

// NewHashChain returns the hash chain implementation.
func NewHashChain() *HashChain { return &HashChain{} }

// canonicalize renders an entry into an unambiguous byte sequence.
//
// Each field is written as name, byte length and value. The explicit length
// makes the encoding injection-proof: no value can be crafted to look like a
// field boundary, which a plain delimiter-joined string would allow.
func canonicalize(e *domain.JournalEntry, prevHash string) []byte {
	var b strings.Builder

	put := func(name, value string) {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(len(value)))
		b.WriteByte(':')
		b.WriteString(value)
		b.WriteByte('\n')
	}
	putInt := func(name string, value int64) { put(name, strconv.FormatInt(value, 10)) }

	put("prev", prevHash)
	put("number", e.EntryNumber)
	putInt("fy", int64(e.FiscalYear))
	put("booking_date", e.BookingDate)
	put("document_date", e.DocumentDate)
	put("service_from", e.ServiceDateFrom)
	put("service_to", e.ServiceDateTo)
	put("value_date", e.ValueDate)
	put("description", e.Description)
	put("source", string(e.Source))
	put("document_number", e.DocumentNumber)
	// DocumentPath is deliberately excluded: moving the data directory must not
	// break the chain. DocumentHash pins the file content instead.
	put("document_hash", e.DocumentHash)
	put("contact", optUint(e.ContactID))
	put("bank_tx", optUint(e.BankTxID))
	put("kind", string(e.Kind))
	put("reversal_of", optUint(e.ReversalOfID))
	put("reversal_reason", e.ReversalReason)
	put("currency", e.Currency)
	putInt("rate_micros", e.ExchangeRateMicros)
	put("rate_source", e.ExchangeRateSource)
	put("rate_date", e.ExchangeRateDate)
	put("rule_version", e.PostingRuleVersion)
	put("created_at", e.CreatedAt.UTC().Format(time.RFC3339))

	// Lines are hashed in a stable order so that a differently ordered read from
	// the database cannot change the result.
	lines := make([]domain.JournalLine, len(e.Lines))
	copy(lines, e.Lines)
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Position < lines[j].Position })

	putInt("lines", int64(len(lines)))
	for _, l := range lines {
		putInt("line_pos", int64(l.Position))
		put("line_side", string(l.Side))
		put("line_account", l.Account)
		putInt("line_amount", int64(l.Amount))
		put("line_contact", optUint(l.ContactID))
		put("line_tax_key", l.TaxKey)
		putInt("line_tax_base", int64(l.TaxBase))
		put("line_text", l.Text)
	}

	return []byte(b.String())
}

func optUint(v *uint) string {
	if v == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*v), 10)
}

// CalculateHash returns the SHA256 digest anchoring an entry to its predecessor.
func (h *HashChain) CalculateHash(e *domain.JournalEntry, prevHash string) string {
	sum := sha256.Sum256(canonicalize(e, prevHash))
	return hex.EncodeToString(sum[:])
}

// VerifyChain walks the entries in journal order and checks both the linkage to
// the predecessor and that each entry still hashes to its stored digest.
func (h *HashChain) VerifyChain(entries []domain.JournalEntry) domain.IntegrityCheckResult {
	checkedAt := time.Now().Format("02.01.2006 15:04:05")

	if len(entries) == 0 {
		return domain.IntegrityCheckResult{
			IsValid:          true,
			Message:          "Keine Buchungen vorhanden. Die Buchhaltung ist bereit.",
			LastVerifiedHash: domain.GenesisHash,
			CheckedAt:        checkedAt,
		}
	}

	expectedPrev := domain.GenesisHash
	for i := range entries {
		entry := &entries[i]

		if entry.PreviousHash != expectedPrev {
			id := entry.ID
			return domain.IntegrityCheckResult{
				IsValid:        false,
				TotalEntries:   len(entries),
				CheckedEntries: i,
				FirstBrokenID:  &id,
				Message: fmt.Sprintf(
					"Unstimmigkeit bei Buchung %s: Die Verkettung zur vorherigen Buchung weicht ab. Eine Buchung wurde nachträglich eingefügt oder entfernt.",
					entry.EntryNumber,
				),
				LastVerifiedHash: expectedPrev,
				CheckedAt:        checkedAt,
			}
		}

		if h.CalculateHash(entry, entry.PreviousHash) != entry.EntryHash {
			id := entry.ID
			return domain.IntegrityCheckResult{
				IsValid:        false,
				TotalEntries:   len(entries),
				CheckedEntries: i,
				FirstBrokenID:  &id,
				Message: fmt.Sprintf(
					"Unstimmigkeit bei Buchung %s: Die Buchungsdaten wurden nach der Erfassung verändert.",
					entry.EntryNumber,
				),
				LastVerifiedHash: expectedPrev,
				CheckedAt:        checkedAt,
			}
		}

		expectedPrev = entry.EntryHash
	}

	return domain.IntegrityCheckResult{
		IsValid:          true,
		TotalEntries:     len(entries),
		CheckedEntries:   len(entries),
		Message:          fmt.Sprintf("Alle %d Buchungen sind vollständig und unverändert.", len(entries)),
		LastVerifiedHash: expectedPrev,
		CheckedAt:        checkedAt,
	}
}
