package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// march baut eine Buchung mit Leistungs- und Belegdatum im März 2026.
func march(id uint, number string, lines ...domain.JournalLine) domain.JournalEntry {
	return domain.JournalEntry{
		ID:          id,
		EntryNumber: number,
		FiscalYear:  2026,
		BookingDate: "2026-03-15", DocumentDate: "2026-03-15",
		ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
		Description: "Testbuchung " + number,
		Source:      domain.EntrySourceManual,
		Lines:       lines,
	}
}

func creditLine(account string, amount domain.Cents, taxKey string, base domain.Cents) domain.JournalLine {
	return domain.JournalLine{Side: domain.SideCredit, Account: account, Amount: amount, TaxKey: taxKey, TaxBase: base}
}

func debitLine(account string, amount domain.Cents, taxKey string, base domain.Cents) domain.JournalLine {
	return domain.JournalLine{Side: domain.SideDebit, Account: account, Amount: amount, TaxKey: taxKey, TaxBase: base}
}

func marchReturn(t *testing.T, src VatReturnSource) *domain.VatReturn {
	t.Helper()
	period, err := ParseVatPeriodKey("2026-03")
	if err != nil {
		t.Fatalf("Zeitraum: %v", err)
	}
	return BuildVatReturn(period, src)
}

// Jeder Steuerschlüssel und jeder Steuerfall hat genau eine Kennziffer. Fiele
// einer heraus, stünde sein Umsatz in keiner Zeile des Vordrucks — und die
// Anmeldung wäre unvollständig, ohne dass es auffiele.
func TestVatCodeForEveryTaxKeyAndTreatment(t *testing.T) {
	cases := []struct {
		name     string
		line     domain.JournalLine
		contact  *uint
		wantCode string
		wantBase domain.Cents
		wantTax  domain.Cents
	}{
		{"Umsatzsteuer 19 %", creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100000), nil, "81", 100000, 19000},
		{"Umsatzsteuer 7 %", creditLine(domain.AccountUmsatzsteuer7, 7000, "UST7", 100000), nil, "86", 100000, 7000},
		{"§ 14c-Steuer", creditLine(domain.AccountUmsatzsteuer14c, 19000, TaxKeyUnlawful, 0), nil, "69", 0, 19000},
		{"Erwerbsteuer 19 %", creditLine(domain.AccountUmsatzsteuerIG19, 19000, "IG19_UST", 100000), nil, "89", 100000, 19000},
		{"Erwerbsteuer 7 %", creditLine(domain.AccountUmsatzsteuerIG, 7000, "IG7_UST", 100000), nil, "93", 100000, 7000},
		{"§ 13b-Steuer als Empfänger", creditLine(domain.AccountUmsatzsteuer13b19, 19000, "RC19_UST", 100000), nil, "46", 100000, 19000},
		{"Vorsteuer aus Rechnungen", debitLine(domain.AccountVorsteuer19, 19000, "VST19", 100000), nil, "66", 0, 19000},
		{"Vorsteuer aus ig. Erwerb", debitLine(domain.AccountVorsteuerIG19, 19000, "IG19_VST", 100000), nil, "61", 0, 19000},
		{"Vorsteuer aus § 13b", debitLine(domain.AccountVorsteuer13b19, 19000, "RC19_VST", 100000), nil, "67", 0, 19000},

		{"Innergemeinschaftliche Lieferung", creditLine("4125", 500000, "", 0), nil, "41", 500000, 0},
		{"Ausfuhrlieferung", creditLine("4120", 300000, "", 0), nil, "43", 300000, 0},
		{"Steuerfreier Umsatz ohne Vorsteuerabzug", creditLine("4150", 200000, "", 0), nil, "48", 200000, 0},
		{"Nullsteuersatz § 12 Abs. 3 UStG", creditLine("4290", 1200000, "", 0), nil, "35", 1200000, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := march(1, "2026-000001", tc.line)
			ret := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{entry}})

			line, ok := ret.Line(tc.wantCode)
			if !ok {
				t.Fatalf("der Vordruck kennt die Kennziffer %s nicht", tc.wantCode)
			}
			if line.Base != tc.wantBase {
				t.Errorf("Kz %s Bemessungsgrundlage = %s, erwartet %s", tc.wantCode, line.Base, tc.wantBase)
			}
			if line.Tax != tc.wantTax {
				t.Errorf("Kz %s Steuer = %s, erwartet %s", tc.wantCode, line.Tax, tc.wantTax)
			}
			if len(line.EntryIDs) != 1 || line.EntryIDs[0] != 1 {
				t.Errorf("Kz %s trägt die Buchungen %v, erwartet [1] — ohne sie gibt es keinen Drill-down",
					tc.wantCode, line.EntryIDs)
			}
		})
	}
}

// Die Leistung mit Steuerschuld des Empfängers ist im übrigen
// Gemeinschaftsgebiet eine nicht steuerbare sonstige Leistung nach § 18b Satz 1
// Nr. 2 UStG (Kz 21) und sonst ein übriger nicht steuerbarer Umsatz (Kz 45).
// Derselbe Buchungssatz, zwei Kennziffern — der Unterschied steht an den
// Stammdaten des Empfängers.
func TestReverseChargeSupplySplitsByRecipient(t *testing.T) {
	contactID := uint(7)
	entry := march(1, "2026-000001", creditLine("4337", 400000, "", 0))
	entry.ContactID = &contactID

	eu := marchReturn(t, VatReturnSource{
		Entries:     []domain.JournalEntry{entry},
		EURecipient: func(uint) bool { return true },
	})
	if eu.Base("21") != 400000 || eu.Base("45") != 0 {
		t.Errorf("EU-Empfänger: Kz 21 = %s, Kz 45 = %s, erwartet 4.000,00 in Kz 21", eu.Base("21"), eu.Base("45"))
	}

	third := marchReturn(t, VatReturnSource{
		Entries:     []domain.JournalEntry{entry},
		EURecipient: func(uint) bool { return false },
	})
	if third.Base("45") != 400000 || third.Base("21") != 0 {
		t.Errorf("Drittland: Kz 21 = %s, Kz 45 = %s, erwartet 4.000,00 in Kz 45", third.Base("21"), third.Base("45"))
	}
}

// Der Vordruck führt die Bemessungsgrundlagen in vollen Euro. Die Steuer bleibt
// auf den Cent genau — sie ist die gebuchte und nicht die nachgerechnete.
func TestBasesAreTruncatedToWholeEuros(t *testing.T) {
	entry := march(1, "2026-000001", creditLine(domain.AccountUmsatzsteuer19, 19009, "UST19", 100049))
	ret := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{entry}})

	line, _ := ret.Line("81")
	if line.Base != 100000 {
		t.Errorf("Bemessungsgrundlage = %s, erwartet 1.000,00 (abgerundet von 1.000,49)", line.Base)
	}
	if line.Tax != 19009 {
		t.Errorf("Steuer = %s, erwartet 190,09 — gebucht wird die Steuer der Rechnung", line.Tax)
	}
	// Die rechnerische Steuer folgt der ungerundeten Grundlage: 19 % von
	// 1.000,49 sind 190,09, und genau das ist gebucht. Die Abrundung des Blattes
	// darf keine Abweichung erzeugen, sonst schlüge die Anzeige bei jeder
	// krummen Rechnung aus und niemand sähe die echte mehr.
	if line.ExpectedTax != 19009 {
		t.Errorf("rechnerische Steuer = %s, erwartet 190,09", line.ExpectedTax)
	}
	if line.Deviation() != 0 {
		t.Errorf("Abweichung = %s, erwartet 0,00 — die Steuer ist richtig gebucht", line.Deviation())
	}
}

// Die Abweichungsanzeige bleibt scharf: eine falsch gebuchte Steuer fällt auf.
func TestDeviationShowsWronglyBookedTax(t *testing.T) {
	entry := march(1, "2026-000001", creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100049))
	ret := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{entry}})

	line, _ := ret.Line("81")
	if line.Deviation() != -9 {
		t.Errorf("Abweichung = %s, erwartet −0,09 (gebucht 190,00, rechnerisch 190,09)", line.Deviation())
	}
}

// Kennziffer 83 ist die Zahllast: geschuldete Steuer abzüglich Vorsteuer und
// abzüglich der angerechneten Sondervorauszahlung.
func TestPayableIsOwedMinusInputTaxMinusPrepayment(t *testing.T) {
	entries := []domain.JournalEntry{
		march(1, "2026-000001", creditLine(domain.AccountUmsatzsteuer19, 38000, "UST19", 200000)),
		march(2, "2026-000002", debitLine(domain.AccountVorsteuer19, 19000, "VST19", 100000)),
	}
	ret := marchReturn(t, VatReturnSource{Entries: entries, SpecialPrepayment: 10000})

	if got := ret.Tax("39"); got != 10000 {
		t.Errorf("Kz 39 = %s, erwartet 100,00", got)
	}
	if ret.Payable != 9000 {
		t.Errorf("Zahllast = %s, erwartet 90,00 (380,00 − 190,00 − 100,00)", ret.Payable)
	}
	if ret.Tax("83") != ret.Payable {
		t.Errorf("Kz 83 = %s, Zahllast am Kopf = %s — beide müssen dasselbe sagen", ret.Tax("83"), ret.Payable)
	}
}

// Das Blatt ist vollständig: auch die Kennziffern, die Buchfink nicht befüllt,
// stehen darauf. Wer nur die befüllten Zeilen sähe, übersähe die Zeile, die er
// von Hand hätte füllen müssen.
func TestSheetCarriesEveryCodeOfTheForm(t *testing.T) {
	ret := marchReturn(t, VatReturnSource{})
	if len(ret.Figures) != len(vatCodes) {
		t.Fatalf("das Blatt hat %d Zeilen, der Vordruck %d", len(ret.Figures), len(vatCodes))
	}
	for _, code := range []string{"41", "43", "48", "81", "86", "35", "89", "93", "46", "21", "45", "66", "61", "62", "67", "64", "69", "39", "83"} {
		if _, ok := ret.Line(code); !ok {
			t.Errorf("Kennziffer %s fehlt auf dem Blatt", code)
		}
	}
}

// Eine Buchung, deren Zeitraum bereits übermittelt ist, wandert nicht
// stillschweigend in den laufenden Zeitraum, sondern erscheint als Nachtrag.
func TestLateEntryIsNotSilentlyMovedIntoTheCurrentPeriod(t *testing.T) {
	late := domain.JournalEntry{
		ID: 5, EntryNumber: "2026-000005", FiscalYear: 2026,
		BookingDate: "2026-03-20", DocumentDate: "2026-02-10",
		ServiceDateFrom: "2026-02-05", ServiceDateTo: "2026-02-05",
		Description: "Nachgereichte Februarrechnung",
		Source:      domain.EntrySourceManual,
		Lines:       []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100000)},
	}

	// Februar ist übermittelt.
	ret := marchReturn(t, VatReturnSource{
		Entries:         []domain.JournalEntry{late},
		SubmittedPeriod: func(date string) bool { return date < "2026-03-01" },
	})
	if ret.Base("81") != 0 {
		t.Errorf("Kz 81 des Märzblatts = %s, erwartet 0 — der Umsatz gehört in den Februar", ret.Base("81"))
	}
	if len(ret.LateEntries) != 1 {
		t.Fatalf("erwartet einen Nachtrag, erhalten %d", len(ret.LateEntries))
	}
	nachtrag := ret.LateEntries[0]
	if nachtrag.PeriodKey != "2026-02" || nachtrag.Code != "81" || nachtrag.Tax != 19000 {
		t.Errorf("Nachtrag = %s / Kz %s / %s, erwartet 2026-02 / Kz 81 / 190,00",
			nachtrag.PeriodKey, nachtrag.Code, nachtrag.Tax)
	}

	// Solange der Februar nicht übermittelt ist, ist es kein Nachtrag: die
	// Buchung gehört dann in die noch offene Februaranmeldung.
	open := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{late}})
	if len(open.LateEntries) != 0 {
		t.Errorf("ohne übermittelten Februar gibt es keinen Nachtrag, erhalten %d", len(open.LateEntries))
	}
	if open.Base("81") != 0 {
		t.Errorf("die Buchung darf auch dann nicht im März stehen, Kz 81 = %s", open.Base("81"))
	}
}

// Die berichtigte Anmeldung ist eine vollständige Neuanmeldung des Zeitraums:
// die ursprünglichen Werte plus das, was seither dazugekommen ist. Eine
// Differenzmeldung wäre etwas anderes — das Finanzamt ersetzt die alte Anmeldung
// durch die neue.
func TestCorrectionRestatesTheWholePeriod(t *testing.T) {
	original := domain.JournalEntry{
		ID: 1, EntryNumber: "2026-000001", FiscalYear: 2026,
		BookingDate: "2026-02-10", DocumentDate: "2026-02-10",
		ServiceDateFrom: "2026-02-05", ServiceDateTo: "2026-02-05",
		Description: "Februarrechnung", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100000)},
	}
	late := domain.JournalEntry{
		ID: 5, EntryNumber: "2026-000005", FiscalYear: 2026,
		BookingDate: "2026-03-20", DocumentDate: "2026-02-20",
		ServiceDateFrom: "2026-02-18", ServiceDateTo: "2026-02-18",
		Description: "Nachgereichte Februarrechnung", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, 9500, "UST19", 50000)},
	}

	february, err := ParseVatPeriodKey("2026-02")
	if err != nil {
		t.Fatal(err)
	}
	correction := BuildVatReturn(february, VatReturnSource{
		Entries: []domain.JournalEntry{original, late},
	})

	if correction.Base("81") != 150000 {
		t.Errorf("Kz 81 der Berichtigung = %s, erwartet 1.500,00 (1.000,00 + 500,00)", correction.Base("81"))
	}
	if correction.Tax("81") != 28500 {
		t.Errorf("Steuer der Berichtigung = %s, erwartet 285,00", correction.Tax("81"))
	}
	if len(correction.EntryIDs("81")) != 2 {
		t.Errorf("die Berichtigung führt %d Buchungen unter Kz 81, erwartet 2", len(correction.EntryIDs("81")))
	}
	if len(correction.LateEntries) != 0 {
		t.Errorf("in der Berichtigung des eigenen Zeitraums gibt es keine Nachträge, erhalten %d", len(correction.LateEntries))
	}
}

// Die Generalumkehr nimmt den Umsatz in dem Zeitraum zurück, in dem er stand —
// nicht in dem, in dem storniert wurde.
func TestReversalNetsOutInTheOriginalPeriod(t *testing.T) {
	original := march(1, "2026-000001", creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100000))
	reversal := domain.JournalEntry{
		ID: 2, EntryNumber: "2026-000002", FiscalYear: 2026,
		Kind: domain.EntryKindReversal, BookingDate: "2026-06-01",
		DocumentDate:    original.DocumentDate,
		ServiceDateFrom: original.ServiceDateFrom, ServiceDateTo: original.ServiceDateTo,
		Description: "Storno", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, -19000, "UST19", -100000)},
	}

	ret := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{original, reversal}})
	if ret.Base("81") != 0 || ret.Tax("81") != 0 {
		t.Errorf("nach der Generalumkehr steht in Kz 81 noch %s / %s, erwartet null",
			ret.Base("81"), ret.Tax("81"))
	}
	if ret.Payable != 0 {
		t.Errorf("Zahllast = %s, erwartet null", ret.Payable)
	}
}

// Der Saldenvortrag bringt den Bestand der Steuerkonten ins neue Jahr. Er ist
// kein Umsatz dieses Zeitraums — sonst stünde die Vorsteuer des Vorjahres ein
// zweites Mal in der Anmeldung des Januars.
func TestOpeningEntriesStayOutOfTheReturn(t *testing.T) {
	opening := march(1, "2026-000001", debitLine(domain.AccountVorsteuer19, 50000, "", 0))
	opening.Source = domain.EntrySourceOpening

	ret := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{opening}})
	if ret.Tax("66") != 0 {
		t.Errorf("Kz 66 = %s, erwartet null — der Vortrag ist keine Vorsteuer dieses Zeitraums", ret.Tax("66"))
	}
}
