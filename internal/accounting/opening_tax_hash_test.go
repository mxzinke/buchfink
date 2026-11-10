package accounting

import (
	"bytes"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestOpeningTaxRateIsCoveredByJournalHash(t *testing.T) {
	entry := &domain.JournalEntry{}
	unknown := canonicalize(entry, domain.GenesisHash)
	rate := domain.TaxRateNone
	entry.OpeningTaxRate = &rate
	zero := canonicalize(entry, domain.GenesisHash)
	if bytes.Equal(unknown, zero) {
		t.Fatal("unbekannter Steuerfall und steuerloser Posten haben dieselbe kanonische Form")
	}
	rate = domain.TaxRateStandard
	if bytes.Equal(zero, canonicalize(entry, domain.GenesisHash)) {
		t.Fatal("Steuersatz ist nicht durch die Hashkette geschützt")
	}
	entry.OpeningTaxRate = nil
	if !bytes.Equal(unknown, canonicalize(entry, domain.GenesisHash)) {
		t.Fatal("alte Buchungen ändern ihre kanonische Form")
	}
}
