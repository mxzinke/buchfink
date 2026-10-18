package accounting

import (
	"sort"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestGoldenVatPeriodFor(t *testing.T) {
	// VatPeriodFor kennt weder Besteuerungsart noch Zahlungszuordnungen. Der
	// JournalService sperrt IST; Zahlungen prüft TestGoldenBuildVatReturn.
	keys := []string{"", "EUSt", "UNBEKANNT", "UST_EXTRA", "VST_EXTRA", "IG_EXTRA", "RC_EXTRA", "RCEU_EXTRA"}
	for _, key := range TaxKeyCatalog() {
		keys = append(keys, key.Key, TaxKeySkontoPrefix+key.Key)
	}
	sort.Strings(keys)
	type dates struct{ Name, Booking, Document, From, To, Received string }
	inputs := []dates{
		{"monatswechsel", "2026-03-10", "2026-02-02", "2026-01-01", "2026-01-31", "2026-03-01"},
		{"quartalswechsel", "2026-05-10", "2026-04-02", "2026-03-01", "2026-03-31", "2026-05-01"},
		{"jahreswechsel", "2027-02-10", "2027-01-02", "2026-12-01", "2026-12-31", "2027-02-01"},
		{"rechnung_vor_leistung", "2026-05-10", "2026-03-01", "2026-03-01", "2026-04-30", "2026-03-02"},
		{"rechnung_nach_folgemonat", "2027-03-10", "2027-02-02", "2026-12-01", "2026-12-31", "2027-03-01"},
		{"schaltjahr_ohne_rechnung", "2024-03-10", "", "2024-01-01", "2024-01-31", ""},
		{"nur_leistungsbeginn", "2026-05-10", "2026-04-02", "2026-03-31", "", "2026-03-01"},
		{"ohne_leistung", "2027-02-10", "2026-12-31", "", "", "2027-01-02"},
		{"nur_buchung", "2026-03-10", "", "", "", ""},
		{"leer", "", "", "", "", ""},
	}
	type result struct {
		Input   dates
		TaxKey  string
		Source  domain.EntrySource
		Kind    domain.EntryKind
		Date    string
		Periods map[domain.VatPeriodType]string
	}
	results := map[string]result{}
	for _, key := range keys {
		for _, in := range inputs {
			for _, variant := range []struct {
				source domain.EntrySource
				kind   domain.EntryKind
			}{
				{domain.EntrySourceManual, domain.EntryKindNormal},
				{domain.EntrySourceAdvance, domain.EntryKindNormal},
				{domain.EntrySourceInvoice, domain.EntryKindReversal},
				{domain.EntrySourceReceipt, domain.EntryKindReversal},
			} {
				entry := &domain.JournalEntry{BookingDate: in.Booking, DocumentDate: in.Document, ServiceDateFrom: in.From, ServiceDateTo: in.To, Source: variant.source, Kind: variant.kind}
				date := VatPeriodFor(entry, domain.JournalLine{TaxKey: key}, in.Received)
				periods := map[domain.VatPeriodType]string{}
				if date != "" {
					for _, typ := range []domain.VatPeriodType{domain.VatPeriodMonth, domain.VatPeriodQuarter, domain.VatPeriodYear} {
						period, err := VatPeriodOf(date, typ)
						if err != nil {
							t.Fatal(err)
						}
						periods[typ] = period.Key
					}
				}
				scenario := key + "/" + string(variant.source) + "/" + string(variant.kind) + "/" + in.Name
				results[scenario] = result{in, key, variant.source, variant.kind, date, periods}
			}
		}
	}
	assertGolden(t, "vat_period/matrix", results)
	assertGolden(t, "vat_period/nil", VatPeriodFor(nil, domain.JournalLine{}, "2026-03-10"))
}
