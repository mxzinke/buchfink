package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// withShareholders trägt Gesellschafterliste und Geschäftsführung ein: Mara
// Beispiel hält 15.000 von 25.000 € (60 %) und führt die Geschäfte, die Muster
// Holding GmbH hält 10.000 € (40 %), Jonas Fremd ist Geschäftsführer ohne
// Anteil.
func (e *testEnv) withShareholders(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	settings := repository.NewSettingsRepository(e.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ShareCapital = 2_500_000
	cfg.Shareholders = []domain.CompanyShareholder{
		{Name: "Mara Beispiel", ShareCapital: 1_500_000},
		{Name: "Muster Holding GmbH", ShareCapital: 1_000_000},
	}
	cfg.ManagingDirectors = "Mara Beispiel, Jonas Fremd"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	e.posting.SetSettingsSource(settings)
}

// person legt einen Kontakt an, etwa einen Gesellschafter.
func (e *testEnv) person(t *testing.T, name string) *domain.Contact {
	t.Helper()
	c := &domain.Contact{Type: domain.ContactTypeVendor, Name: name, CountryCode: "DE", IsPrivate: true,
		Street: "Gartenweg 2", PostalCode: "80331", City: "München"}
	if err := e.contacts.SaveContact(context.Background(), c); err != nil {
		t.Fatalf("Kontakt %s: %v", name, err)
	}
	return c
}

// Firmenkreditkarte: der Beleg steht gegen 3610, ein offener Posten beim
// Lieferanten entsteht nicht.
func TestCreditCardReceiptBooksAgainstCardAccount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Software AG", "DE", "")
	req := env.receipt(t, vendor.ID, "software", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Settlement = SettlementCreditCard

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Kartenbeleg: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "6495", 10000},
		{domain.SideDebit, domain.AccountVorsteuer19, 1900},
		{domain.SideCredit, domain.AccountKreditkartenabrechnung, 11900},
	})
	if entry.ContactID == nil || *entry.ContactID != vendor.ID {
		t.Error("der Lieferant bleibt Kontakt der Buchung")
	}
	items, err := env.payments(t).OpenItems(ctx)
	if err != nil || len(items) != 0 {
		t.Errorf("ein Kartenbeleg hinterlässt keinen offenen Posten: %d, %v", len(items), err)
	}
}

// Die Monatsabrechnung: zwei Kartenbelege bis zum Stichtag 20.03., ein dritter
// danach. Die Abbuchung am 02.04. trifft den Saldo zum Stichtag, Buchfink
// schlägt die Abrechnung vor und gleicht 3610 aus.
func TestCreditCardStatementIsMatchedAndSettled(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.paymentsWithBank(t)
	vendor := env.vendor(t, "Software AG", "DE", "")
	for _, c := range []struct {
		date string
		net  domain.Cents
	}{{"2026-03-05", 10000}, {"2026-03-20", 5000}, {"2026-03-28", 2000}} {
		req := env.receipt(t, vendor.ID, "software", c.net, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		req.Settlement = SettlementCreditCard
		req.BookingDate, req.DocumentDate, req.ServiceDateFrom, req.ServiceDateTo = c.date, c.date, c.date, c.date
		if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
			t.Fatalf("Kartenbeleg %s: %v", c.date, err)
		}
	}
	id := env.bankLine(t, "2026-04-02", -17850, "Kartenbank", "Abrechnung 03/2026")

	bank := env.banks(t)
	bank.SetCreditCardSource(payments)
	suggestions, err := bank.Suggest(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var card *BankSuggestion
	for i := range suggestions.Suggestions {
		if suggestions.Suggestions[i].Kind == SuggestionCreditCard {
			card = &suggestions.Suggestions[i]
		}
	}
	if card == nil || card.StatementDate != "2026-03-20" {
		t.Fatalf("erwartet Vorschlag Kartenabrechnung zum 20.03., erhalten %+v", suggestions.Suggestions)
	}

	if _, err := payments.SettleCreditCard(ctx, CreditCardSettlementRequest{BankTxID: id, StatementDate: "2026-03-28"}); err == nil ||
		!strings.Contains(err.Error(), "Differenz") {
		t.Errorf("ein Stichtag mit anderem Saldo muss abgewiesen werden, erhalten %v", err)
	}
	entry, err := payments.SettleCreditCard(ctx, CreditCardSettlementRequest{BankTxID: id, StatementDate: card.StatementDate})
	if err != nil {
		t.Fatalf("Ausgleich: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, domain.AccountKreditkartenabrechnung, 17850},
		{domain.SideCredit, domain.AccountBank, 17850},
	})
	env.assertMatch(t, id, domain.MatchStatusMatched)
	balance, err := payments.CreditCardBalance(ctx, "2026-04-30")
	if err != nil || balance != 2380 {
		t.Errorf("nach der Abrechnung steht nur der Beleg vom 28.03. offen (23,80 €), erhalten %s €, %v", balance, err)
	}
}

// Auslage: die Gesellschafterin bezahlt eine Rechnung der GmbH privat. Der
// Lieferant ist Aussteller und Kontakt der Buchung, die Erstattungsschuld steht
// auf 3640 bei ihr; die Vorsteuer zieht die GmbH.
func TestShareholderExpenseCreditsTheShareholder(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	vendor := env.vendor(t, "Bürohaus GmbH", "DE", "")
	mara := env.person(t, "Mara Beispiel")

	req := env.receipt(t, vendor.ID, "buerobedarf", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Settlement = SettlementShareholderExpense
	req.CreditorContactID = mara.ID
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Auslage: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "6815", 10000},
		{domain.SideDebit, domain.AccountVorsteuer19, 1900},
		{domain.SideCredit, domain.AccountVerbindlichkeitenGesellschafter, 11900},
	})
	if entry.ContactID == nil || *entry.ContactID != vendor.ID {
		t.Error("Kontakt der Buchung ist der Aussteller")
	}
	for _, l := range entry.Lines {
		if l.Account == domain.AccountVerbindlichkeitenGesellschafter && (l.ContactID == nil || *l.ContactID != mara.ID) {
			t.Error("die Erstattungsschuld steht nicht bei der Gesellschafterin")
		}
	}

	// Die Erstattung über die Bank gleicht ihr Verrechnungskonto aus.
	id := env.bankLine(t, "2026-03-20", -11900, "Mara Beispiel", "Erstattung Auslage")
	if _, err := env.paymentsWithBank(t).BookBankSplit(ctx, BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{
		{Account: domain.AccountVerbindlichkeitenGesellschafter, ContactID: mara.ID, Amount: 11900, Text: "Erstattung Auslage"},
	}}); err != nil {
		t.Fatalf("Erstattung: %v", err)
	}
	if balance := env.accountBalance(t, domain.AccountVerbindlichkeitenGesellschafter); balance != 0 {
		t.Errorf("nach der Erstattung steht auf 3640 noch %s €", balance)
	}
}

// Ein Geschäftsführer ohne Anteil bekommt seine Auslage als sonstige
// Verbindlichkeit; wer weder Gesellschafter noch Geschäftsführer ist, keine.
func TestExpenseCreditorMustBelongToTheCompany(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withShareholders(t)
	vendor := env.vendor(t, "Bürohaus GmbH", "DE", "")
	jonas := env.person(t, "Jonas Fremd")
	stranger := env.person(t, "Erika Mustermann")

	req := env.receipt(t, vendor.ID, "buerobedarf", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Settlement = SettlementShareholderExpense
	req.CreditorContactID = jonas.ID
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Auslage des Geschäftsführers: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "6815", 10000},
		{domain.SideDebit, domain.AccountVorsteuer19, 1900},
		{domain.SideCredit, domain.AccountSonstigeVerbindlichkeiten, 11900},
	})

	other := env.receipt(t, vendor.ID, "buerobedarf", 10000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	other.Settlement = SettlementShareholderExpense
	other.CreditorContactID = stranger.ID
	if _, err := env.posting.PostIncomingReceipt(ctx, other); err == nil ||
		!strings.Contains(err.Error(), "weder in der Gesellschafterliste") {
		t.Errorf("erwartet Abweisung, erhalten %v", err)
	}
	other.CreditorContactID = 0
	if _, err := env.posting.PostIncomingReceipt(ctx, other); err == nil {
		t.Error("eine Auslage ohne Gläubiger wurde gebucht")
	}
}
