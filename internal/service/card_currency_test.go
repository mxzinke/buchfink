package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// cardReceipt bucht 84 USD für Software über die Firmenkarte (§ 13b Abs. 2
// UStG), zum EZB-Kurs 1,1405 also 73,65 €.
func (e *testEnv) cardReceipt(t *testing.T, charged domain.Cents) *domain.JournalEntry {
	t.Helper()
	vendor := e.vendor(t, "Codeforge Inc.", "US", "")
	req := e.receipt(t, vendor.ID, "software", 8_400, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge)
	req.Currency, req.ForeignAmount, req.ChargedAmount = "USD", 8_400, charged
	req.Settlement = SettlementCreditCard
	req.BookingDate, req.DocumentDate, req.ServiceDateFrom, req.ServiceDateTo = "2026-03-10", "2026-03-10", "2026-03-10", "2026-03-10"
	entry, err := e.posting.PostIncomingReceipt(context.Background(), req)
	if err != nil {
		t.Fatalf("Kartenbeleg: %v", err)
	}
	return entry
}

// Ist die Belastung der Karte bekannt, steht sie im Aufwand und auf 3610; die
// Umsatzsteuer nach § 13b rechnet weiter mit dem EZB-Kurs, nicht mit dem der
// Karte. Beim Ausgleich der Abrechnung bleibt keine Differenz.
func TestCardReceiptBooksTheChargedEuroAmount(t *testing.T) {
	env := newTestEnv(t)
	server, _ := ecbServer(t, "1.1405", "2026-03-10")
	env.currencyPosting(t, server.URL)

	entry := env.cardReceipt(t, 7_229)
	if debitOn(entry, "6495") != 7_229 || creditOn(entry, domain.AccountKreditkartenabrechnung) != 7_229 {
		t.Errorf("Aufwand %s, Karte %s; erwartet die Belastung 72,29 €",
			debitOn(entry, "6495"), creditOn(entry, domain.AccountKreditkartenabrechnung))
	}
	if debitOn(entry, "1407") != 1_399 {
		t.Errorf("Vorsteuer § 13b %s, erwartet 13,99 € aus 73,65 € zum EZB-Kurs", debitOn(entry, "1407"))
	}
}

// Ohne bekannte Belastung bucht der Beleg zum EZB-Kurs. Weicht die Abbuchung
// der Karte im Rahmen des Kurses ab, nimmt Buchfink die Abweichung beim
// Ausgleich als Kursdifferenz an.
func TestCardSettlementAssumesTheCurrencyDifference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.1405", "2026-03-10")
	env.currencyPosting(t, server.URL)
	payments := env.paymentsWithBank(t)

	entry := env.cardReceipt(t, 0)
	if creditOn(entry, domain.AccountKreditkartenabrechnung) != 7_365 {
		t.Fatalf("Karte %s, erwartet 73,65 € zum EZB-Kurs", creditOn(entry, domain.AccountKreditkartenabrechnung))
	}
	date, err := payments.CreditCardStatementDate(ctx, 7_229, "2026-04-05")
	if err != nil || date != "2026-03-10" {
		t.Fatalf("Stichtag %q, %v; erwartet 10.03. trotz Kursabweichung", date, err)
	}
	id := env.bankLine(t, "2026-04-05", -7_229, "Kartenbank", "Abrechnung 03/2026")
	settled, err := payments.SettleCreditCard(ctx, CreditCardSettlementRequest{BankTxID: id, StatementDate: date})
	if err != nil {
		t.Fatalf("Ausgleich: %v", err)
	}
	assertLines(t, settled, []bookedLine{
		{domain.SideDebit, domain.AccountKreditkartenabrechnung, 7_365},
		{domain.SideCredit, domain.AccountBank, 7_229},
		{domain.SideCredit, "4840", 136},
	})
}
