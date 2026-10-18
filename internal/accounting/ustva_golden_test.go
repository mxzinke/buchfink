package accounting

import (
	"fmt"
	"sort"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestGoldenBuildVatReturn(t *testing.T) {
	covered := map[string]bool{}
	check := func(name, period string, src VatReturnSource) {
		ret := returnFor(t, period, src)
		for _, line := range ret.Figures {
			if line.Base != 0 || line.Tax != 0 {
				covered[line.Code] = true
			}
		}
		t.Run(name, func(t *testing.T) {
			// IDs stammen aus festen Fixtures, nicht aus einer Datenbank. Die
			// Herkunftsverknüpfungen bleiben damit prüfbar. Nur Mengen werden sortiert;
			// Figuren und Befunde behalten ihre fachliche Reihenfolge.
			for i := range ret.Figures {
				sort.Slice(ret.Figures[i].EntryIDs, func(a, b int) bool { return ret.Figures[i].EntryIDs[a] < ret.Figures[i].EntryIDs[b] })
			}
			assertGolden(t, "vat_return/"+name, ret)
		})
	}
	check("leer", "2026-03", VatReturnSource{})
	// Dieselben Steuerzeilen wie TestVatCodeForEveryTaxKeyAndTreatment,
	// ergänzt um sämtliche Schlüssel des Katalogs, einschließlich Skonto.
	for _, key := range TaxKeyCatalog() {
		for _, skonto := range []bool{false, true} {
			taxKey, side, amount, base := key.Key, key.Side, domain.Cents(19009), domain.Cents(100049)
			if key.Rate != 0 {
				amount = key.Rate.Tax(base)
			}
			if key.Key == TaxKeyUnlawful || key.Key == TaxKeyInputTaxCorrection {
				base = 0
			}
			if skonto {
				taxKey = TaxKeySkontoPrefix + taxKey
				side = domain.SideDebit
				if key.Side == domain.SideDebit {
					side = domain.SideCredit
				}
			}
			line := domain.JournalLine{Account: key.Account, Side: side, Amount: amount, TaxBase: base, TaxKey: taxKey}
			check(taxKey, "2026-03", VatReturnSource{Entries: []domain.JournalEntry{march(1, "2026-000001", line)}})
		}
	}
	for _, tc := range []struct {
		name, account string
		eu, domestic  bool
	}{
		{"steuerfrei", "4150", false, false}, {"nullsatz", "4290", false, false},
		{"rc_leistender_eu", "4337", true, false}, {"rc_leistender_inland", "4337", false, true},
		{"rc_leistender_drittland", "4337", false, false}, {"nicht_steuerbar", "4400", false, false},
	} {
		entry := march(1, "2026-000001", creditLine(tc.account, 200049, "", 0))
		if tc.name == "nicht_steuerbar" {
			entry.TaxTreatment = domain.TaxTreatmentNotTaxable
		}
		contact := uint(7)
		entry.ContactID = &contact
		check(tc.name, "2026-03", VatReturnSource{Entries: []domain.JournalEntry{entry}, EURecipient: func(uint) bool { return tc.eu }, DomesticRecipient: func(uint) bool { return tc.domestic }})
	}
	original := march(1, "2026-000001", creditLine(domain.AccountUmsatzsteuer19, 19009, "UST19", 100049))
	input := march(2, "2026-000002", debitLine(domain.AccountVorsteuer19, 9500, "VST19", 50000))
	check("zahllast_sondervorauszahlung", "2026-03", VatReturnSource{Entries: []domain.JournalEntry{original, input}, SpecialPrepayment: 10000})
	for _, tc := range []struct {
		name  string
		lines []domain.JournalLine
	}{
		{"abweichung", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100049)}},
		{"rc_ohne_steuer", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer13b19, 0, "RCEU19_UST", 100000)}},
		{"rc_falscher_satz", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer13b19, 30000, "RC19_UST", 100000), debitLine(domain.AccountVorsteuer13b19, 30000, "RC19_VST", 100000)}},
		{"rc_vorsteuerueberhang", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer13b19, 19000, "RC19_UST", 100000), debitLine(domain.AccountVorsteuer13b19, 38000, "RC19_VST", 200000)}},
	} {
		check(tc.name, "2026-03", VatReturnSource{Entries: []domain.JournalEntry{march(1, "2026-000001", tc.lines...)}})
	}
	reversal := original
	reversal.ID = 3
	reversal.EntryNumber = "2026-000003"
	reversal.Kind = domain.EntryKindReversal
	reversal.BookingDate = "2026-06-01"
	reversal.Lines = []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, -19009, "UST19", -100049)}
	check("generalumkehr", "2026-03", VatReturnSource{Entries: []domain.JournalEntry{original, reversal}})
	reversal.Source = domain.EntrySourceInvoice
	check("rechnungsstorno", "2026-06", VatReturnSource{Entries: []domain.JournalEntry{original, reversal}})
	for _, source := range []domain.EntrySource{domain.EntrySourceOpening, domain.EntrySourceClosing} {
		entry := original
		entry.Source = source
		entry.Lines = append(append([]domain.JournalLine{}, entry.Lines...), creditLine(InputTaxCorrectionRepayableMovable, 9500, TaxKeyInputTaxCorrection, 0))
		check(string(source), "2026-03", VatReturnSource{Entries: []domain.JournalEntry{entry}})
	}
	late := original
	late.ID = 5
	late.EntryNumber = "2026-000005"
	late.DocumentDate = "2026-02-10"
	late.ServiceDateFrom = "2026-02-05"
	late.ServiceDateTo = "2026-02-05"
	for _, submitted := range []bool{false, true} {
		for _, reported := range []bool{false, true} {
			check(fmt.Sprintf("nachtrag_gemeldet_%t_enthalten_%t", submitted, reported), "2026-03", VatReturnSource{Entries: []domain.JournalEntry{late, original}, SubmittedPeriod: func(string) bool { return submitted }, ReportedEntry: func(string, uint) bool { return reported }})
		}
	}
	february := late
	february.ID, february.EntryNumber = 4, "2026-000004"
	check("berichtigung", "2026-02", VatReturnSource{Entries: []domain.JournalEntry{february, late}})
	receiptID := uint(9)
	input.ReceiptID = &receiptID
	for _, period := range []string{"2026-03", "2026-Q2", "2026"} {
		check("belegeingang_"+period, period, VatReturnSource{Entries: []domain.JournalEntry{input}, ReceivedAt: func(uint) string { return "2026-04-02" }})
	}
	src := VatReturnSource{
		Entries:     []domain.JournalEntry{subscriptionEntry(1), paymentEntry(2, "2026-01-20", 39667), paymentEntry(3, "2026-02-20", 39666)},
		Allocations: []domain.PaymentAllocation{{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: 39667, CashAmount: 39667}, {OpenItemEntryID: 1, PaymentEntryID: 3, SettledAmount: 39666, CashAmount: 39666}},
	}
	for _, period := range []string{"2026-01", "2026-02", "2026-12", "2026-Q1", "2026"} {
		check("raten_"+period, period, src)
	}
	src.SubmittedPeriod = func(date string) bool { return date < "2026-03-01" }
	src.ReportedEntry = func(_ string, id uint) bool { return id != 3 }
	check("zahlung_nachgemeldet", "2026-03", src)
	src.SubmittedPeriod, src.ReportedEntry = nil, nil
	revPayment := paymentEntry(4, "2026-02-25", 39666)
	revPayment.Kind = domain.EntryKindReversal
	paymentID := uint(3)
	revPayment.ReversalOfID = &paymentID
	src.Entries = append(src.Entries, revPayment)
	check("zahlung_storniert", "2026-02", src)
	// Kz 50 folgt der Zahlungszuordnung einer Ausbuchung, wie im Diensttest
	// zu uneinbringlichen Forderungen.
	writeOff := reversal
	writeOff.ID = 6
	writeOff.Source = domain.EntrySourceManual
	writeOff.Kind = domain.EntryKindNormal
	src = VatReturnSource{Entries: []domain.JournalEntry{original, writeOff}, Allocations: []domain.PaymentAllocation{{OpenItemEntryID: 1, PaymentEntryID: 6, SettledAmount: 119058, CashAmount: 0, DifferenceKind: domain.DifferenceWriteoff}}}
	check("uneinbringlich", "2026-03", src)
	advance := original
	advance.Source, advance.BookingDate = domain.EntrySourceAdvance, "2026-01-10"
	check("vereinnahmte_anzahlung", "2026-01", VatReturnSource{Entries: []domain.JournalEntry{advance}})
	for _, key := range []string{"VST19", "IG19_UST", "RCEU19_UST", "RC19_UST"} {
		entry := subscriptionEntry(1)
		entry.Source = domain.EntrySourceReceipt
		gross := domain.Cents(100000)
		if key == "VST19" {
			gross = 119000
		}
		entry.Lines = []domain.JournalLine{creditLine("70000", gross, "", 0)}
		for _, info := range TaxKeyCatalog() {
			if info.Key == key {
				entry.Lines = append(entry.Lines, domain.JournalLine{Account: info.Account, Side: info.Side, Amount: 19000, TaxKey: key, TaxBase: 100000})
			}
		}
		entry.ReceiptID = &receiptID
		source := VatReturnSource{Entries: []domain.JournalEntry{entry, paymentEntry(2, "2026-01-20", gross)}, Allocations: []domain.PaymentAllocation{{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: gross}}, ReceivedAt: func(uint) string { return "2026-02-01" }}
		for _, period := range []string{"2026-01", "2026-02", "2026-12"} {
			check("vorauszahlung_"+key+"_"+period, period, source)
		}
	}
	for _, key := range []string{"RCEU19_UST", "RC19_UST"} {
		entry := march(1, "2026-000001", creditLine(domain.AccountUmsatzsteuer13b19, 38000, key, 200000), debitLine(domain.AccountVorsteuer13b19, 38000, key[:len(key)-3]+"VST", 200000))
		entry.ServiceDateFrom, entry.ServiceDateTo, entry.DocumentDate = "2025-07-01", "2027-06-30", "2027-07-01"
		for _, period := range []string{"2025-12", "2026-12", "2027"} {
			check("dauerleistung_"+key+"_"+period, period, VatReturnSource{Entries: []domain.JournalEntry{entry}})
		}
	}
	check("vorsteueranteil", "2026-Q1", VatReturnSource{Entries: []domain.JournalEntry{march(1, "2026-000001", debitLine(domain.AccountVorsteuer19, 11400, "VST19", 60000))}})

	// Formularzeilen ohne Rechenzweig bleiben ausdrücklich außerhalb dieser
	// Liste; jede berechnete Kennzahl muss mindestens einmal ungleich null sein.
	for _, code := range []string{"81", "86", "87", "48", "89", "93", "46", "84", "60", "21", "45", "66", "61", "67", "64", "69", "39", "83", "50"} {
		if !covered[code] {
			t.Errorf("Kz %s hat kein Golden-Szenario mit einem Betrag ungleich null", code)
		}
	}
}
