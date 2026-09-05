package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Rücknahme einer Ausgangsrechnung wirkt im Zeitraum der Berichtigung.
//
// § 17 Abs. 1 Satz 8 UStG legt die Berichtigung in den Zeitraum, in dem die
// Änderung eingetreten ist; für die zu hoch oder zu Unrecht ausgewiesene Steuer
// sagt Abschn. 14c.1 Abs. 5 UStAE dasselbe. Beides ist der Tag des
// Stornodokuments — und der steht als Buchungsdatum an der Generalumkehr,
// während Beleg- und Leistungsdatum weiter den zurückgenommenen Vorgang nennen.
func TestVatPeriodOfInvoiceReversalIsTheCorrectionDate(t *testing.T) {
	reversal := &domain.JournalEntry{
		Kind:            domain.EntryKindReversal,
		Source:          domain.EntrySourceInvoice,
		BookingDate:     "2026-06-15",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
	}

	for _, taxKey := range []string{"UST19", ""} {
		got := VatPeriodFor(reversal, domain.JournalLine{TaxKey: taxKey}, "")
		if got != "2026-06-15" {
			t.Errorf("Steuerschlüssel %q: Zeitraum = %q, erwartet den Tag der Berichtigung 2026-06-15",
				taxKey, got)
		}
	}

	// Gegenprobe: die Ursprungsbuchung selbst bleibt im Leistungszeitraum. Ohne
	// sie prüfte der Test nur, dass irgendein Datum herauskommt.
	original := &domain.JournalEntry{
		Source:          domain.EntrySourceInvoice,
		BookingDate:     "2026-04-02",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
	}
	if got := VatPeriodFor(original, domain.JournalLine{TaxKey: "UST19"}, ""); got != "2026-03-01" {
		t.Errorf("die Ursprungsbuchung gehört in den Leistungszeitraum, ergibt aber %q", got)
	}

	// Und die Generalumkehr eines Eingangsbelegs bleibt, wo sie war: die Regel
	// gilt der Rechnungsberichtigung des Leistenden und nicht jeder Umkehrung.
	inputReversal := &domain.JournalEntry{
		Kind:            domain.EntryKindReversal,
		Source:          domain.EntrySourceReceipt,
		BookingDate:     "2026-06-15",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
	}
	if got := VatPeriodFor(inputReversal, domain.JournalLine{TaxKey: "VST19"}, ""); got != "2026-03-01" {
		t.Errorf("die Umkehrung einer Eingangsbuchung ergibt %q, erwartet 2026-03-01", got)
	}
}
