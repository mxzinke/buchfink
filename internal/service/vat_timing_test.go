package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// annualSubscription stellt ein Jahresabo über 1.200 € netto aus: Rechnung am
// 05.01.2026, Leistung vom 01.01. bis 31.12.2026.
func (e *testEnv) annualSubscription(t *testing.T, customerID uint) *domain.Invoice {
	t.Helper()
	inv := &domain.Invoice{
		ContactID: customerID, Date: "2026-01-05",
		ServiceDateFrom: "2026-01-01", ServiceDateTo: "2026-12-31",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Jahresabo 2026", QuantityMilli: 1000, UnitPrice: 120000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := e.invoices(t).Issue(context.Background(), inv); err != nil {
		t.Fatalf("Jahresabo ausstellen: %v", err)
	}
	if inv.JournalEntryID == nil {
		t.Fatal("die Rechnung hat keine Buchung")
	}
	return inv
}

func (e *testEnv) settle(t *testing.T, entryID uint, date string, amount domain.Cents) *domain.JournalEntry {
	t.Helper()
	payment, err := e.payments(t).Settle(context.Background(), PaymentRequest{
		PaymentAccount: domain.AccountBank, PaymentDate: date,
		Allocations: []AllocationRequest{{OpenItemEntryID: entryID, SettledAmount: amount}},
	})
	if err != nil {
		t.Fatalf("Zahlung vom %s: %v", date, err)
	}
	return payment
}

func (e *testEnv) vatDraft(t *testing.T, period string) *domain.VatReturn {
	t.Helper()
	ret, err := e.vatReturns(t).Draft(context.Background(), period)
	if err != nil {
		t.Fatalf("Voranmeldung %s: %v", period, err)
	}
	return ret
}

// Mindest-Istversteuerung (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG): das im
// Januar bezahlte Jahresabo gehört mit seiner ganzen Steuer ins erste Quartal,
// obwohl die Leistung erst am 31.12. endet.
func TestPaidAnnualSubscriptionTaxArisesWithPayment(t *testing.T) {
	env := newTestEnv(t)
	customer := env.customer(t, "Kunde", "DE", "")
	inv := env.annualSubscription(t, customer.ID)
	payment := env.settle(t, *inv.JournalEntryID, "2026-01-20", 142800)

	q1 := env.vatDraft(t, "2026-Q1")
	if q1.Base("81") != 120000 || q1.Tax("81") != 22800 {
		t.Errorf("Q1: Kz 81 = %s / %s, erwartet 1.200,00 / 228,00", q1.Base("81"), q1.Tax("81"))
	}
	ids := q1.EntryIDs("81")
	if len(ids) != 2 || !containsID(ids, *inv.JournalEntryID) || !containsID(ids, payment.ID) {
		t.Errorf("Drill-down Kz 81 = %v, erwartet Rechnung %d und Zahlung %d", ids, *inv.JournalEntryID, payment.ID)
	}
	q4 := env.vatDraft(t, "2026-Q4")
	if q4.Base("81") != 0 || q4.Tax("81") != 0 {
		t.Errorf("Q4: Kz 81 = %s / %s, erwartet nichts mehr", q4.Base("81"), q4.Tax("81"))
	}
}

// Ohne Zahlung bleibt es bei der Sollversteuerung: die Steuer entsteht mit dem
// Ende der Leistung im Dezember.
func TestUnpaidAnnualSubscriptionStaysWithTheService(t *testing.T) {
	env := newTestEnv(t)
	customer := env.customer(t, "Kunde", "DE", "")
	env.annualSubscription(t, customer.ID)

	if q1 := env.vatDraft(t, "2026-Q1"); q1.Tax("81") != 0 {
		t.Errorf("Q1: Kz 81 Steuer = %s, erwartet 0", q1.Tax("81"))
	}
	if q4 := env.vatDraft(t, "2026-Q4"); q4.Base("81") != 120000 || q4.Tax("81") != 22800 {
		t.Errorf("Q4: Kz 81 = %s / %s, erwartet 1.200,00 / 228,00", q4.Base("81"), q4.Tax("81"))
	}
}

// Eine Teilzahlung zieht nur ihren Anteil vor („insoweit"): 714 € von 1.428 €
// sind die Hälfte, also 600 € Entgelt und 114 € Steuer im ersten Quartal.
func TestPartialPrepaymentMovesOnlyItsShare(t *testing.T) {
	env := newTestEnv(t)
	customer := env.customer(t, "Kunde", "DE", "")
	inv := env.annualSubscription(t, customer.ID)
	env.settle(t, *inv.JournalEntryID, "2026-02-10", 71400)

	q1 := env.vatDraft(t, "2026-Q1")
	if q1.Base("81") != 60000 || q1.Tax("81") != 11400 {
		t.Errorf("Q1: Kz 81 = %s / %s, erwartet 600,00 / 114,00", q1.Base("81"), q1.Tax("81"))
	}
	q4 := env.vatDraft(t, "2026-Q4")
	if q4.Base("81") != 60000 || q4.Tax("81") != 11400 {
		t.Errorf("Q4: Kz 81 = %s / %s, erwartet 600,00 / 114,00", q4.Base("81"), q4.Tax("81"))
	}
}

// Die Vorsteuer aus der Vorausrechnung eines Lieferanten ist abziehbar, sobald
// Rechnung und Zahlung vorliegen (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG).
func TestPrepaidSupplierInvoiceDeductsInputTaxWithPayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Hosting GmbH", "DE", "")

	open := env.receipt(t, vendor.ID, "software", 240000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	open.DocumentDate, open.BookingDate = "2026-01-08", "2026-01-08"
	open.ServiceDateFrom, open.ServiceDateTo = "2026-01-01", "2026-12-31"
	env.setReceivedAt(t, open.ReceiptID, "2026-01-08")
	entry, err := env.posting.PostIncomingReceipt(ctx, open)
	if err != nil {
		t.Fatalf("Vorausrechnung buchen: %v", err)
	}
	if q1 := env.vatDraft(t, "2026-Q1"); q1.Tax("66") != 0 {
		t.Errorf("unbezahlt: Kz 66 in Q1 = %s, erwartet 0", q1.Tax("66"))
	}

	env.settle(t, entry.ID, "2026-02-01", 285600)
	if q1 := env.vatDraft(t, "2026-Q1"); q1.Tax("66") != 45600 {
		t.Errorf("bezahlt: Kz 66 in Q1 = %s, erwartet 456,00", q1.Tax("66"))
	}
	if q4 := env.vatDraft(t, "2026-Q4"); q4.Tax("66") != 0 {
		t.Errorf("bezahlt: Kz 66 in Q4 = %s, erwartet 0", q4.Tax("66"))
	}
}

// Ein sofort bezahlter Beleg ist am Buchungstag bezahlt; die Vorsteuer aus dem
// Jahresvertrag gehört dann in diesen Zeitraum.
func TestImmediatelyPaidReceiptDeductsInputTaxWithPayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Software GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "software", 120000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.DocumentDate, req.BookingDate = "2026-01-08", "2026-01-08"
	req.ServiceDateFrom, req.ServiceDateTo = "2026-01-01", "2026-12-31"
	req.Settlement, req.PaymentAccount = SettlementPaid, domain.AccountBank
	env.setReceivedAt(t, req.ReceiptID, "2026-01-08")
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	if q1 := env.vatDraft(t, "2026-Q1"); q1.Tax("66") != 22800 {
		t.Errorf("Kz 66 in Q1 = %s, erwartet 228,00", q1.Tax("66"))
	}
}

// Mit der Firmenkreditkarte oder als Auslage ist ein Beleg ebenso am
// Buchungstag bezahlt wie über die Bank: die Vorsteuer aus dem Jahresabo und
// beim EU-Anbieter die Steuer nach § 13b Abs. 4 Satz 2 UStG gehören in den
// Zeitraum der Zahlung, nicht an das Leistungsende im Dezember.
func TestReceiptPaidByCardOrAsExpenseCountsAsPaid(t *testing.T) {
	ways := []struct {
		name   string
		settle func(env *testEnv, req *ReceiptRequest)
	}{
		{"Kreditkarte", func(_ *testEnv, req *ReceiptRequest) { req.Settlement = SettlementCreditCard }},
		{"Auslage", func(env *testEnv, req *ReceiptRequest) {
			env.withShareholders(t)
			req.Settlement = SettlementShareholderExpense
			req.CreditorContactID = env.person(t, "Mara Beispiel").ID
		}},
	}
	for _, way := range ways {
		t.Run(way.name+"/Inland", func(t *testing.T) {
			env := newTestEnv(t)
			vendor := env.vendor(t, "Software GmbH", "DE", "")
			req := env.receipt(t, vendor.ID, "software", 120000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
			req.DocumentDate, req.BookingDate = "2026-01-08", "2026-01-08"
			req.ServiceDateFrom, req.ServiceDateTo = "2026-01-01", "2026-12-31"
			way.settle(env, &req)
			env.setReceivedAt(t, req.ReceiptID, "2026-01-08")
			if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err != nil {
				t.Fatalf("Beleg buchen: %v", err)
			}
			if q1 := env.vatDraft(t, "2026-Q1"); q1.Tax("66") != 22800 {
				t.Errorf("Kz 66 in Q1 = %s, erwartet 228,00", q1.Tax("66"))
			}
			if q4 := env.vatDraft(t, "2026-Q4"); q4.Tax("66") != 0 {
				t.Errorf("Kz 66 in Q4 = %s, erwartet 0", q4.Tax("66"))
			}
		})
		t.Run(way.name+"/EU", func(t *testing.T) {
			env := newTestEnv(t)
			vendor := env.vendor(t, "Cloud Ireland Ltd.", "IE", "IE6388047V")
			req := env.receipt(t, vendor.ID, "software", 120000, domain.TaxRateStandard, domain.TaxTreatmentReverseChargeEU)
			req.DocumentDate, req.BookingDate = "2026-01-08", "2026-01-08"
			req.ServiceDateFrom, req.ServiceDateTo = "2026-01-01", "2026-12-31"
			way.settle(env, &req)
			if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err != nil {
				t.Fatalf("Beleg buchen: %v", err)
			}
			q1 := env.vatDraft(t, "2026-Q1")
			if q1.Base("46") != 120000 || q1.Tax("46") != 22800 || q1.Tax("67") != 22800 {
				t.Errorf("Q1: Kz 46 = %s / %s, Kz 67 = %s, erwartet 1.200,00 / 228,00 / 228,00",
					q1.Base("46"), q1.Tax("46"), q1.Tax("67"))
			}
			if q4 := env.vatDraft(t, "2026-Q4"); q4.Tax("46") != 0 || q4.Tax("67") != 0 {
				t.Errorf("Q4: Kz 46 = %s, Kz 67 = %s, erwartet nichts mehr", q4.Tax("46"), q4.Tax("67"))
			}
		})
	}
}

// § 13b Abs. 1 UStG: die Steuer auf die sonstige Leistung eines EU-Unternehmers
// entsteht im Zeitraum der Leistung — die im April datierte Rechnung über eine
// Märzleistung gehört ins erste Quartal, in Kz 46/47.
func TestReverseChargeEUServiceFollowsTheService(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Ads Ireland Ltd.", "IE", "IE6388047V")

	req := env.receipt(t, vendor.ID, "werbung", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseChargeEU)
	req.DocumentDate, req.BookingDate = "2026-04-03", "2026-04-03"
	req.ServiceDateFrom, req.ServiceDateTo = "2026-03-01", "2026-03-31"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	q1 := env.vatDraft(t, "2026-Q1")
	if q1.Base("46") != 100000 || q1.Tax("46") != 19000 || q1.Tax("67") != 19000 {
		t.Errorf("Q1: Kz 46 = %s / %s, Kz 67 = %s, erwartet 1.000,00 / 190,00 / 190,00",
			q1.Base("46"), q1.Tax("46"), q1.Tax("67"))
	}
	if q1.Base("84") != 0 {
		t.Errorf("Kz 84 = %s, die EU-Leistung gehört nicht dorthin", q1.Base("84"))
	}
	if q2 := env.vatDraft(t, "2026-Q2"); q2.Tax("46") != 0 {
		t.Errorf("Q2: Kz 46 = %s, das Rechnungsdatum entscheidet bei Abs. 1 nicht", q2.Tax("46"))
	}
}

// § 13b Abs. 3 UStG: die sonstige Leistung eines EU-Unternehmers vom
// 01.07.2025 bis 30.06.2027 (730 Tage, 24.000 € netto) läuft über mehr als ein
// Jahr. Die Steuer entsteht spätestens mit Ablauf jedes Kalenderjahres für den
// darin erbrachten Teil: 184 Tage in 2025 (6.049,32 €, Steuer 1.149,37 €),
// 365 Tage in 2026 (12.000,00 €, 2.280,00 €), der Rest mit dem Leistungsende
// (5.950,68 €, 1.130,63 €). Die Bemessungsgrundlage steht in vollen Euro im
// Vordruck. Gebucht ist der Vertrag 2025; die Anmeldung 2027 sieht ihn
// trotzdem.
func TestLongTermReverseChargeArisesAtEachYearEnd(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Cloud Ireland Ltd.", "IE", "IE6388047V")
	req := env.receipt(t, vendor.ID, "software", 2_400_000, domain.TaxRateStandard, domain.TaxTreatmentReverseChargeEU)
	req.DocumentDate, req.BookingDate = "2025-07-01", "2025-07-01"
	req.ServiceDateFrom, req.ServiceDateTo = "2025-07-01", "2027-06-30"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	for _, c := range []struct {
		period    string
		base, tax domain.Cents
	}{
		{"2025-Q3", 0, 0},
		{"2025-Q4", 604_900, 114_937},
		{"2026-Q1", 0, 0},
		{"2026-Q4", 1_200_000, 228_000},
		{"2027-Q2", 595_000, 113_063},
	} {
		ret := env.vatDraft(t, c.period)
		if ret.Base("46") != c.base || ret.Tax("46") != c.tax || ret.Tax("67") != c.tax {
			t.Errorf("%s: Kz 46 = %s / %s, Kz 67 = %s, erwartet %s / %s / %s", c.period,
				ret.Base("46"), ret.Tax("46"), ret.Tax("67"), c.base, c.tax, c.tax)
		}
	}
}

// Bezahlt die GmbH den Zweijahresvertrag im August 2025 vorab, entsteht die
// ganze Steuer im dritten Quartal 2025 (§ 13b Abs. 4 Satz 2 UStG). Spätere
// Jahre sehen die Zahlung aus 2025 und rechnen nichts ein zweites Mal.
func TestPrepaidLongTermReverseChargeArisesWithThePayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Cloud Ireland Ltd.", "IE", "IE6388047V")
	req := env.receipt(t, vendor.ID, "software", 2_400_000, domain.TaxRateStandard, domain.TaxTreatmentReverseChargeEU)
	req.DocumentDate, req.BookingDate = "2025-07-01", "2025-07-01"
	req.ServiceDateFrom, req.ServiceDateTo = "2025-07-01", "2027-06-30"
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	env.settle(t, entry.ID, "2025-08-01", 2_400_000)

	for period, tax := range map[string]domain.Cents{"2025-Q3": 456_000, "2025-Q4": 0, "2026-Q4": 0, "2027-Q2": 0} {
		if ret := env.vatDraft(t, period); ret.Tax("46") != tax {
			t.Errorf("%s: Kz 46 Steuer = %s, erwartet %s", period, ret.Tax("46"), tax)
		}
	}
}

// Eine Leistung über genau ein Jahr fällt nicht unter § 13b Abs. 3 UStG: das
// Jahresabo vom 01.07.2026 bis 30.06.2027 gehört mit seiner ganzen Steuer ans
// Leistungsende.
func TestOneYearReverseChargeStaysWithTheServiceEnd(t *testing.T) {
	env := newTestEnv(t)
	vendor := env.vendor(t, "Cloud Ireland Ltd.", "IE", "IE6388047V")
	req := env.receipt(t, vendor.ID, "software", 120_000, domain.TaxRateStandard, domain.TaxTreatmentReverseChargeEU)
	req.DocumentDate, req.BookingDate = "2026-07-01", "2026-07-01"
	req.ServiceDateFrom, req.ServiceDateTo = "2026-07-01", "2027-06-30"
	if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	if q4 := env.vatDraft(t, "2026-Q4"); q4.Tax("46") != 0 {
		t.Errorf("2026-Q4: Kz 46 = %s, erwartet nichts", q4.Tax("46"))
	}
	if q2 := env.vatDraft(t, "2027-Q2"); q2.Base("46") != 120_000 || q2.Tax("46") != 22_800 {
		t.Errorf("2027-Q2: Kz 46 = %s / %s, erwartet 1.200,00 / 228,00", q2.Base("46"), q2.Tax("46"))
	}
}

// § 13b Abs. 2 UStG: die Leistung aus dem Drittland gehört in Kz 84/85, und die
// Steuer entsteht mit der Rechnung im April.
func TestReverseChargeThirdCountryFollowsTheInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "SaaS Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge)
	req.DocumentDate, req.BookingDate = "2026-04-03", "2026-04-03"
	req.ServiceDateFrom, req.ServiceDateTo = "2026-03-01", "2026-03-31"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	if q1 := env.vatDraft(t, "2026-Q1"); q1.Base("84") != 0 || q1.Tax("84") != 0 {
		t.Errorf("Q1: Kz 84 = %s, erwartet nichts", q1.Tax("84"))
	}
	q2 := env.vatDraft(t, "2026-Q2")
	if q2.Base("84") != 100000 || q2.Tax("84") != 19000 || q2.Tax("67") != 19000 {
		t.Errorf("Q2: Kz 84 = %s / %s, Kz 67 = %s, erwartet 1.000,00 / 190,00 / 190,00",
			q2.Base("84"), q2.Tax("84"), q2.Tax("67"))
	}
	if q2.Base("46") != 0 {
		t.Errorf("Kz 46 = %s, die Drittlandsleistung gehört nicht dorthin", q2.Base("46"))
	}
}

// § 13b Abs. 1 setzt einen Unternehmer aus einem anderen EU-Staat voraus; für
// einen EU-Lieferanten unter Abs. 2 weist die Vorschau auf Abs. 1 hin.
func TestReverseChargeParagraphFollowsTheSupplierSeat(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	us := env.vendor(t, "SaaS Inc.", "US", "")
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, us.ID, "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseChargeEU)); err == nil {
		t.Fatal("§ 13b Abs. 1 für einen Lieferanten aus dem Drittland muss abgewiesen werden")
	} else if !strings.Contains(err.Error(), "§ 13b Abs. 2") {
		t.Errorf("die Meldung sollte auf § 13b Abs. 2 verweisen: %v", err)
	}

	ie := env.vendor(t, "Cloud Ireland Ltd.", "IE", "IE6388047V")
	preview, err := env.posting.PreviewIncomingReceipt(ctx,
		env.receipt(t, ie.ID, "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge))
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	found := false
	for _, w := range preview.Warnings {
		found = found || w.Code == warningReverseChargeEU
	}
	if !found {
		t.Errorf("die Vorschau sollte bei einem EU-Lieferanten unter Abs. 2 auf Abs. 1 hinweisen: %+v", preview.Warnings)
	}
}

// Die Ausbuchung einer uneinbringlichen Forderung mindert Kz 81 im Zeitraum der
// Uneinbringlichkeit und steht zusätzlich in Kz 50 (§ 17 Abs. 2 Nr. 1 UStG).
func TestWriteOffShowsInCode50(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	inv := env.domesticInvoice(t, customer.ID, "2026-02-10", 100000)

	if _, err := env.payments(t).WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: *inv.JournalEntryID, Date: "2026-05-20", Reason: "Insolvenz des Kunden",
	}); err != nil {
		t.Fatalf("ausbuchen: %v", err)
	}

	q2 := env.vatDraft(t, "2026-Q2")
	if q2.Base("81") != -100000 || q2.Tax("81") != -19000 {
		t.Errorf("Q2: Kz 81 = %s / %s, erwartet -1.000,00 / -190,00", q2.Base("81"), q2.Tax("81"))
	}
	if q2.Base("50") != 100000 {
		t.Errorf("Q2: Kz 50 = %s, erwartet 1.000,00", q2.Base("50"))
	}
	if q1 := env.vatDraft(t, "2026-Q1"); q1.Base("50") != 0 || q1.Base("81") != 100000 {
		t.Errorf("Q1: Kz 81 = %s, Kz 50 = %s, erwartet 1.000,00 und 0", q1.Base("81"), q1.Base("50"))
	}
}

// Die eigene Leistung an einen inländischen Unternehmer, der die Steuer nach
// § 13b Abs. 5 UStG schuldet, gehört in Kz 60 und nicht zu den nicht
// steuerbaren Umsätzen.
func TestDomesticReverseChargeSupplyIsCode60(t *testing.T) {
	env := newTestEnv(t)
	customer := env.customer(t, "Reinigungsbetrieb GmbH", "DE", "DE123456789")
	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-02-10",
		ServiceDateFrom: "2026-02-10", ServiceDateTo: "2026-02-10",
		TaxTreatment: domain.TaxTreatmentReverseChargeSupply,
		Items: []domain.InvoiceItem{{
			Description: "Leistung", QuantityMilli: 1000, UnitPrice: 50000, TaxRate: domain.TaxRateNone,
		}},
	}
	if err := env.invoices(t).Issue(context.Background(), inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}
	q1 := env.vatDraft(t, "2026-Q1")
	if q1.Base("60") != 50000 || q1.Base("45") != 0 || q1.Base("21") != 0 {
		t.Errorf("Kz 60 = %s, Kz 45 = %s, Kz 21 = %s, erwartet 500,00 in Kz 60",
			q1.Base("60"), q1.Base("45"), q1.Base("21"))
	}
}

// Das Skonto auf eine EU-Dienstleistung mindert Kz 21 und die Zusammenfassende
// Meldung im Zeitraum der Zahlung (§ 17 Abs. 1 Satz 8 UStG, § 18a Abs. 7
// Satz 2 UStG); beide Meldungen bleiben abgestimmt.
func TestSkontoOnEUServiceReducesCode21AndZM(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client GmbH", "AT", "ATU12345678")
	inv := env.euService(t, customer.ID, "2026-03-10", 100000)

	payment, err := env.payments(t).Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank, PaymentDate: "2026-04-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID: *inv.JournalEntryID, SettledAmount: 100000,
			DifferenceKind: domain.DifferenceSkonto, DifferenceAmount: 2000,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Skonto: %v", err)
	}
	skonto := false
	for _, l := range payment.Lines {
		skonto = skonto || (l.Account == "4741" && l.Amount == 2000)
	}
	if !skonto {
		t.Errorf("das Skonto gehört auf 4741, Zeilen %+v", payment.Lines)
	}

	q1 := env.vatDraft(t, "2026-Q1")
	q2 := env.vatDraft(t, "2026-Q2")
	if q1.Base("21") != 100000 || q2.Base("21") != -2000 {
		t.Errorf("Kz 21: Q1 = %s, Q2 = %s, erwartet 1.000,00 und -20,00", q1.Base("21"), q2.Base("21"))
	}
	if _, err := env.vatReturns(t).Save(ctx, "2026-Q2"); err != nil {
		t.Fatalf("Voranmeldung Q2 speichern: %v", err)
	}
	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q2")
	if err != nil {
		t.Fatalf("ZM Q2: %v", err)
	}
	if zm.TotalServices != -2000 {
		t.Errorf("ZM Q2 = %s, erwartet -20,00", zm.TotalServices)
	}
	if zm.Reconciliation == nil || zm.Reconciliation.ServicesDifference() != 0 {
		t.Errorf("ZM und Kz 21 gehen auseinander: %+v", zm.Reconciliation)
	}
}

func containsID(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
