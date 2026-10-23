package accounting

import (
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// subscriptionEntry ist eine Ausgangsrechnung über ein Jahresabo: 1.000 €
// netto, 190 € Steuer, Leistung vom 01.01. bis 31.12.2026, offen auf dem
// Debitorenkonto.
func subscriptionEntry(id uint) domain.JournalEntry {
	return domain.JournalEntry{
		ID: id, EntryNumber: "2026-000001", FiscalYear: 2026,
		BookingDate: "2026-01-05", DocumentDate: "2026-01-05",
		ServiceDateFrom: "2026-01-01", ServiceDateTo: "2026-12-31",
		Source: domain.EntrySourceInvoice,
		Lines: []domain.JournalLine{
			debitLine("10000", 119000, "", 0),
			creditLine("4400", 100000, "", 0),
			creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100000),
		},
	}
}

func paymentEntry(id uint, date string, amount domain.Cents) domain.JournalEntry {
	return domain.JournalEntry{
		ID: id, EntryNumber: fmt.Sprintf("2026-%06d", id), FiscalYear: 2026,
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Source: domain.EntrySourcePayment,
		Lines: []domain.JournalLine{
			debitLine(domain.AccountBank, amount, "", 0),
			creditLine("10000", amount, "", 0),
		},
	}
}

func returnFor(t *testing.T, key string, src VatReturnSource) *domain.VatReturn {
	t.Helper()
	period, err := ParseVatPeriodKey(key)
	if err != nil {
		t.Fatalf("Zeitraum %s: %v", key, err)
	}
	return BuildVatReturn(period, src)
}

// Drei Raten auf ein Jahresabo, zwei vor dem Leistungsende: die Steuer entsteht
// je Rate im Monat der Zahlung, der Rest bleibt im Dezember. Gerundet wird über
// die aufgelaufenen Anteile — die Teile ergeben zusammen genau die gebuchte
// Steuer.
func TestPrepaymentSplitsByInstalment(t *testing.T) {
	entries := []domain.JournalEntry{
		subscriptionEntry(1),
		paymentEntry(2, "2026-01-20", 39667),
		paymentEntry(3, "2026-02-20", 39666),
	}
	src := VatReturnSource{
		Entries: entries,
		Allocations: []domain.PaymentAllocation{
			{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: 39667, CashAmount: 39667},
			{OpenItemEntryID: 1, PaymentEntryID: 3, SettledAmount: 39666, CashAmount: 39666},
		},
	}
	jan := returnFor(t, "2026-01", src)
	feb := returnFor(t, "2026-02", src)
	dec := returnFor(t, "2026-12", src)

	// 396,67 € von 1.190 € = ein Drittel: 333,34 € Entgelt, 63,33 € Steuer.
	if jan.Tax("81") != 6333 || jan.Base("81") != 33300 {
		t.Errorf("Januar: Kz 81 = %s / %s, erwartet 333,00 / 63,33", jan.Base("81"), jan.Tax("81"))
	}
	total := jan.Tax("81") + feb.Tax("81") + dec.Tax("81")
	if total != 19000 {
		t.Errorf("Januar %s + Februar %s + Dezember %s = %s, erwartet 190,00",
			jan.Tax("81"), feb.Tax("81"), dec.Tax("81"), total)
	}
	if dec.Tax("81") <= 0 {
		t.Errorf("der unbezahlte Rest gehört in den Dezember, Kz 81 = %s", dec.Tax("81"))
	}
}

// Eine Zahlung nach dem Leistungsende verschiebt nichts: die Steuer ist dann
// schon mit der Leistung entstanden.
func TestPaymentAfterTheServiceChangesNothing(t *testing.T) {
	entry := subscriptionEntry(1)
	entry.ServiceDateFrom, entry.ServiceDateTo = "2026-01-05", "2026-01-05"
	src := VatReturnSource{
		Entries:     []domain.JournalEntry{entry, paymentEntry(2, "2026-02-20", 119000)},
		Allocations: []domain.PaymentAllocation{{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: 119000}},
	}
	if jan := returnFor(t, "2026-01", src); jan.Tax("81") != 19000 {
		t.Errorf("Januar: Kz 81 = %s, erwartet 190,00", jan.Tax("81"))
	}
	if feb := returnFor(t, "2026-02", src); feb.Tax("81") != 0 {
		t.Errorf("Februar: Kz 81 = %s, erwartet 0", feb.Tax("81"))
	}
}

// Eine stornierte Zahlung hat nichts vereinnahmt.
func TestReversedPaymentDoesNotPullTheTaxForward(t *testing.T) {
	reversal := paymentEntry(3, "2026-01-25", 119000)
	reversal.Kind = domain.EntryKindReversal
	original := uint(2)
	reversal.ReversalOfID = &original
	src := VatReturnSource{
		Entries:     []domain.JournalEntry{subscriptionEntry(1), paymentEntry(2, "2026-01-20", 119000), reversal},
		Allocations: []domain.PaymentAllocation{{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: 119000}},
	}
	if jan := returnFor(t, "2026-01", src); jan.Tax("81") != 0 {
		t.Errorf("Januar: Kz 81 = %s, erwartet 0 — die Zahlung ist storniert", jan.Tax("81"))
	}
	if dec := returnFor(t, "2026-12", src); dec.Tax("81") != 19000 {
		t.Errorf("Dezember: Kz 81 = %s, erwartet 190,00", dec.Tax("81"))
	}
}

// Die Generalumkehr eines bezahlten Eingangsbelegs folgt den Zahlungen ihres
// Originals und hebt die Vorsteuer in denselben Zeiträumen auf.
func TestReversalOfPrepaidReceiptNetsOutInThePaymentPeriod(t *testing.T) {
	receipt := domain.JournalEntry{
		ID: 1, EntryNumber: "2026-000001", FiscalYear: 2026,
		BookingDate: "2026-01-08", DocumentDate: "2026-01-08",
		ServiceDateFrom: "2026-01-01", ServiceDateTo: "2026-12-31",
		Source: domain.EntrySourceReceipt,
		Lines: []domain.JournalLine{
			debitLine("6495", 100000, "", 0),
			debitLine(domain.AccountVorsteuer19, 19000, "VST19", 100000),
			creditLine(domain.AccountBank, 119000, "", 0),
		},
	}
	reversal := receipt
	reversal.ID, reversal.EntryNumber = 2, "2026-000002"
	reversal.Kind, reversal.BookingDate = domain.EntryKindReversal, "2026-05-02"
	original := uint(1)
	reversal.ReversalOfID = &original
	reversal.Lines = []domain.JournalLine{
		debitLine("6495", -100000, "", 0),
		debitLine(domain.AccountVorsteuer19, -19000, "VST19", -100000),
		creditLine(domain.AccountBank, -119000, "", 0),
	}

	src := VatReturnSource{Entries: []domain.JournalEntry{receipt, reversal}}
	if jan := returnFor(t, "2026-01", src); jan.Tax("66") != 0 {
		t.Errorf("Januar: Kz 66 = %s, erwartet 0 — Beleg und Umkehr heben sich auf", jan.Tax("66"))
	}
	only := VatReturnSource{Entries: []domain.JournalEntry{receipt}}
	if jan := returnFor(t, "2026-01", only); jan.Tax("66") != 19000 {
		t.Errorf("ohne Umkehr: Kz 66 im Januar = %s, erwartet 190,00", jan.Tax("66"))
	}
	if dec := returnFor(t, "2026-12", src); dec.Tax("66") != 0 {
		t.Errorf("Dezember: Kz 66 = %s, erwartet 0", dec.Tax("66"))
	}
}

// Die Erwerbsteuer kennt keine Mindest-Istversteuerung: Anzahlungen auf einen
// innergemeinschaftlichen Erwerb werden nicht angemeldet.
func TestAcquisitionIgnoresPrepayment(t *testing.T) {
	entry := domain.JournalEntry{
		ID: 1, EntryNumber: "2026-000001", FiscalYear: 2026,
		BookingDate: "2026-03-02", DocumentDate: "2026-03-02",
		ServiceDateFrom: "2026-03-20", ServiceDateTo: "2026-03-20",
		Source: domain.EntrySourceReceipt,
		Lines: []domain.JournalLine{
			debitLine("5425", 100000, "", 0),
			debitLine(domain.AccountVorsteuerIG19, 19000, "IG19_VST", 100000),
			creditLine(domain.AccountUmsatzsteuerIG19, 19000, "IG19_UST", 100000),
			creditLine("70000", 100000, "", 0),
		},
	}
	src := VatReturnSource{
		Entries:     []domain.JournalEntry{entry, paymentEntry(2, "2026-01-15", 100000)},
		Allocations: []domain.PaymentAllocation{{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: 100000}},
	}
	if jan := returnFor(t, "2026-01", src); jan.Tax("89") != 0 {
		t.Errorf("Januar: Kz 89 = %s, erwartet 0", jan.Tax("89"))
	}
	if mar := returnFor(t, "2026-03", src); mar.Tax("89") != 19000 {
		t.Errorf("März: Kz 89 = %s, erwartet 190,00", mar.Tax("89"))
	}
}

// Eine Zahlung, die nach der Übermittlung des Zahlungsmonats gebucht wird, ist
// ein Nachtrag — auch wenn die Rechnung selbst in dieser Anmeldung schon stand.
func TestLatePaymentOnReportedInvoiceIsALateEntry(t *testing.T) {
	src := VatReturnSource{
		Entries: []domain.JournalEntry{
			subscriptionEntry(1),
			paymentEntry(2, "2026-01-10", 59500),
			paymentEntry(3, "2026-01-28", 59500),
		},
		Allocations: []domain.PaymentAllocation{
			{OpenItemEntryID: 1, PaymentEntryID: 2, SettledAmount: 59500},
			{OpenItemEntryID: 1, PaymentEntryID: 3, SettledAmount: 59500},
		},
		SubmittedPeriod: func(date string) bool { return date < "2026-02-01" },
		// Gemeldet waren im Januar die Rechnung und die erste Zahlung.
		ReportedEntry: func(date string, id uint) bool { return date < "2026-02-01" && (id == 1 || id == 2) },
	}
	feb := returnFor(t, "2026-02", src)
	if len(feb.LateEntries) != 1 {
		t.Fatalf("erwartet ein Nachtrag aus der zweiten Zahlung, erhalten %d", len(feb.LateEntries))
	}
	if late := feb.LateEntries[0]; late.PeriodKey != "2026-01" || late.Tax != 9500 {
		t.Errorf("Nachtrag = %s / %s, erwartet 2026-01 / 95,00", late.PeriodKey, late.Tax)
	}
}

// Die Paarprüfungen: Grundlage ohne Steuer, Steuer neben dem Satz, Vorsteuer
// aus § 13b über der geschuldeten Steuer.
func TestVatReturnPairChecks(t *testing.T) {
	cases := []struct {
		name  string
		lines []domain.JournalLine
		want  string
	}{
		{"stimmig", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, 19000, "UST19", 100000)}, ""},
		{"Steuer weicht ab", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer19, 21000, "UST19", 100000)}, "Kz 81"},
		{"§ 13b ohne Steuer", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer13b19, 0, "RCEU19_UST", 100000)}, "Kz 46"},
		{"§ 13b-Satz außerhalb 7 bis 19 %", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer13b19, 30000, "RC19_UST", 100000),
			debitLine(domain.AccountVorsteuer13b19, 30000, "RC19_VST", 100000)}, "Kz 85"},
		{"Vorsteuer über der Steuerschuld", []domain.JournalLine{creditLine(domain.AccountUmsatzsteuer13b19, 19000, "RC19_UST", 100000),
			debitLine(domain.AccountVorsteuer13b19, 38000, "RC19_VST", 200000)}, "Kz 67"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ret := marchReturn(t, VatReturnSource{Entries: []domain.JournalEntry{march(1, "2026-000001", tc.lines...)}})
			if tc.want == "" {
				if len(ret.Findings) != 0 {
					t.Errorf("keine Befunde erwartet, erhalten %v", ret.Findings)
				}
				return
			}
			found := false
			for _, f := range ret.Findings {
				found = found || strings.Contains(f, tc.want)
			}
			if !found {
				t.Errorf("erwartet einen Befund zu %s, erhalten %v", tc.want, ret.Findings)
			}
		})
	}
}

// Der Vordruck 2026 führt die neuen Kennzahlen; jede steht auf dem Blatt, auch
// wenn Buchfink sie nicht füllt.
func TestSheetFollowsTheForm2026(t *testing.T) {
	ret := marchReturn(t, VatReturnSource{})
	want := []string{"81", "86", "87", "35", "77", "76", "41", "44", "49", "43", "48", "91", "89", "93", "90",
		"95", "94", "46", "73", "84", "42", "60", "21", "45", "66", "61", "62", "67", "63", "59", "64", "65",
		"69", "39", "83", "50", "37"}
	if len(ret.Figures) != len(want) {
		t.Fatalf("das Blatt hat %d Zeilen, erwartet %d", len(ret.Figures), len(want))
	}
	for i, code := range want {
		if ret.Figures[i].Code != code {
			t.Errorf("Zeile %d ist Kz %s, erwartet Kz %s (Reihenfolge des Vordrucks)", i+1, ret.Figures[i].Code, code)
		}
	}
	if l, _ := ret.Line("73"); l.Reference != "§ 13b Abs. 2 Nr. 3 UStG" {
		t.Errorf("Kz 73 ist die Zeile der GrEStG-Umsätze, Fundstelle %q", l.Reference)
	}
}
