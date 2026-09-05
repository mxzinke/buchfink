package accounting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
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

// canonicalize renders an entry into an unambiguous byte sequence using the
// shared length-prefixed encoding (see canonicalWriter).
func canonicalize(e *domain.JournalEntry, prevHash string) []byte {
	var w canonicalWriter
	put, putInt := w.put, w.putInt

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
	// The Beleg is covered by one value, as the single file hash used to be —
	// only now that value stands for the whole ordered file list. ReceiptID is
	// deliberately excluded for the same reason the old DocumentPath was: moving
	// or re-importing data must not break the chain, and an id is a location
	// rather than content.
	put("receipt_hash", e.ReceiptHash)
	// The Steuerfall is part of the record, not just of the input. On the
	// incoming side it is the only thing separating an exempt purchase from one
	// at the Nullsteuersatz of § 12 Abs. 3 UStG.
	put("tax_treatment", string(e.TaxTreatment))
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

	// The Aufzeichnung for entertainment expenses is what the deduction hangs on
	// (§ 4 Abs. 5 Satz 1 Nr. 2 EStG), so it is covered like every other field
	// that carries accounting meaning. Absent, it contributes a single empty
	// marker rather than nothing, so "no record" and "an empty record" cannot
	// hash alike.
	if d := e.Entertainment; d != nil {
		putInt("entertainment", 1)
		put("entertainment_place", d.Place)
		put("entertainment_day", d.Day)
		put("entertainment_participants", d.Participants)
		put("entertainment_occasion", d.Occasion)
	} else {
		putInt("entertainment", 0)
	}

	return w.bytes()
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
//
// Die Prüfung hält beim ersten Bruch nicht an. Wer eine Zeile ändert, ändert
// meist mehrere; ein Ergebnis, das nur die erste nennt, führt zu einer
// Reparatur, die nach dem nächsten Lauf wieder von vorn beginnt. Nach einem
// Bruch setzt die Erwartung auf dem tatsächlichen Eigenhash der Buchung auf,
// damit nicht jede folgende Buchung als gebrochen gemeldet wird.
func (h *HashChain) VerifyChain(entries []domain.JournalEntry) domain.IntegrityCheckResult {
	result := h.verifyYear(0, entries)
	result.CheckedAt = time.Now().Format("02.01.2006 15:04:05")
	result.Message = chainMessage(result)
	result.EnsureLists()
	return result
}

// VerifyYears prüft mehrere Geschäftsjahre, jedes für sich.
//
// Die Kette beginnt je Jahr neu beim Genesis-Hash — der Jahreswechsel ist kein
// Bruch, sondern der Anfang einer neuen Kette. Eine Prüfung, die nur das aktive
// Jahr ansieht, meldet Unversehrtheit, während in einem festgeschriebenen Jahr
// eine Zeile verändert wurde; das ist genau der Fall, für den es die Kette gibt
// (§ 146 Abs. 4 AO, GoBD Rz. 107).
func (h *HashChain) VerifyYears(byYear map[int][]domain.JournalEntry) domain.IntegrityCheckResult {
	years := make([]int, 0, len(byYear))
	for year := range byYear {
		years = append(years, year)
	}
	sort.Ints(years)

	combined := domain.IntegrityCheckResult{
		IsValid:          true,
		LastVerifiedHash: domain.GenesisHash,
		FiscalYears:      years,
		Breaks:           make([]domain.IntegrityBreak, 0),
	}
	for _, year := range years {
		one := h.verifyYear(year, byYear[year])
		combined.TotalEntries += one.TotalEntries
		combined.CheckedEntries += one.CheckedEntries
		combined.Breaks = append(combined.Breaks, one.Breaks...)
		if !one.IsValid {
			combined.IsValid = false
			if combined.FirstBrokenID == nil {
				combined.FirstBrokenID = one.FirstBrokenID
			}
		}
		if one.TotalEntries > 0 {
			combined.LastVerifiedHash = one.LastVerifiedHash
		}
	}

	combined.CheckedAt = time.Now().Format("02.01.2006 15:04:05")
	combined.Message = chainMessage(combined)
	combined.EnsureLists()
	return combined
}

// verifyYear prüft eine Kette und sammelt jeden Bruch mit erwartetem und
// tatsächlichem Hash.
func (h *HashChain) verifyYear(year int, entries []domain.JournalEntry) domain.IntegrityCheckResult {
	result := domain.IntegrityCheckResult{
		IsValid:          true,
		TotalEntries:     len(entries),
		CheckedEntries:   len(entries),
		LastVerifiedHash: domain.GenesisHash,
		Breaks:           make([]domain.IntegrityBreak, 0),
	}
	if year > 0 {
		result.FiscalYears = []int{year}
	}

	expectedPrev := domain.GenesisHash
	for i := range entries {
		entry := &entries[i]

		if entry.PreviousHash != expectedPrev {
			result.IsValid = false
			if result.FirstBrokenID == nil {
				id := entry.ID
				result.FirstBrokenID = &id
			}
			result.Breaks = append(result.Breaks, domain.IntegrityBreak{
				FiscalYear:   entry.FiscalYear,
				EntryID:      entry.ID,
				EntryNumber:  entry.EntryNumber,
				Reason:       domain.IntegrityBreakLinkage,
				ExpectedHash: expectedPrev,
				ActualHash:   entry.PreviousHash,
				Message: fmt.Sprintf(
					"Buchung %s: Die Verkettung zur vorherigen Buchung weicht ab. Eine Buchung wurde nachträglich eingefügt oder entfernt.",
					entry.EntryNumber,
				),
			})
		}

		if computed := h.CalculateHash(entry, entry.PreviousHash); computed != entry.EntryHash {
			result.IsValid = false
			if result.FirstBrokenID == nil {
				id := entry.ID
				result.FirstBrokenID = &id
			}
			result.Breaks = append(result.Breaks, domain.IntegrityBreak{
				FiscalYear:   entry.FiscalYear,
				EntryID:      entry.ID,
				EntryNumber:  entry.EntryNumber,
				Reason:       domain.IntegrityBreakContent,
				ExpectedHash: computed,
				ActualHash:   entry.EntryHash,
				Message: fmt.Sprintf(
					"Buchung %s: Die Buchungsdaten wurden nach der Erfassung verändert.",
					entry.EntryNumber,
				),
			})
		}

		// Weiter geht es mit dem gespeicherten Eigenhash: sonst schleppte ein
		// einzelner Bruch sich durch den ganzen Rest des Jahres.
		expectedPrev = entry.EntryHash
	}

	result.LastVerifiedHash = expectedPrev
	return result
}

// chainMessage formuliert das Ergebnis in einem Satz.
func chainMessage(r domain.IntegrityCheckResult) string {
	switch {
	case r.TotalEntries == 0:
		return "Keine Buchungen vorhanden. Die Buchhaltung ist bereit."
	case r.IsValid && len(r.FiscalYears) > 1:
		return fmt.Sprintf("Alle %d Buchungen aus %d Geschäftsjahren sind vollständig und unverändert.",
			r.TotalEntries, len(r.FiscalYears))
	case r.IsValid:
		return fmt.Sprintf("Alle %d Buchungen sind vollständig und unverändert.", r.TotalEntries)
	case len(r.Breaks) == 1:
		return "Eine Unstimmigkeit gefunden: " + r.Breaks[0].Message
	default:
		return fmt.Sprintf("%d Unstimmigkeiten gefunden. Die erste: %s",
			len(r.Breaks), r.Breaks[0].Message)
	}
}
