package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// -------------------------------------------------------------------------
// Stichtagsbewertung: Bankkonten und die Auflösung beim Saldenvortrag
// -------------------------------------------------------------------------

// Ein Dollarkonto gehört in die Stichtagsbewertung: § 256a HGB nennt „auf
// Fremdwährung lautende Vermögensgegenstände" und meint nicht nur die offenen
// Posten.
func TestCurrencyValuationCoversForeignBankAccounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	svc := env.currency(t, server.URL)
	svc.SetPeriodSource(env.closing(t))
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db)))

	// 11.000 USD zum Buchkurs 1,10 sind 10.000 € auf dem Dollarkonto. Zum
	// Stichtagskurs 1,00 sind sie 11.000 € wert: 1.000 € Kursgewinn.
	if _, err := env.journal.Post(ctx, &domain.JournalEntry{
		BookingDate: "2026-05-02", DocumentDate: "2026-05-02",
		ServiceDateFrom: "2026-05-02", ServiceDateTo: "2026-05-02",
		Description: "Zahlungseingang auf dem Dollarkonto", Source: domain.EntrySourceManual,
		DocumentNumber: "BK-USD-1", TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency: "USD", ExchangeRateMicros: 1_100_000,
		ExchangeRateSource: "EZB-Referenzkurs", ExchangeRateDate: "2026-05-02",
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "1800", Amount: 1_000_000, ForeignAmount: 1_100_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 1_000_000, ForeignAmount: 1_100_000},
		},
	}); err != nil {
		t.Fatalf("Bankbuchung: %v", err)
	}

	preview, err := svc.PreviewCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	var bank *ForeignCurrencyValuationItem
	for i := range preview.Items {
		if preview.Items[i].Kind == "bank" {
			bank = &preview.Items[i]
		}
	}
	if bank == nil {
		t.Fatalf("das Fremdwährungskonto fehlt in der Bewertung: %+v", preview.Items)
	}
	if bank.Account != "1800" || bank.Currency != "USD" || bank.ForeignAmount != 1_100_000 {
		t.Errorf("Bankzeile %+v — erwartet 11.000,00 USD auf 1800", bank)
	}
	if !bank.Gain || !bank.Recognised || bank.Amount != 100_000 {
		t.Errorf("Bankzeile: Gewinn=%v erfasst=%v %s € — ein Guthaben ist jederzeit fällig und "+
			"damit stets kurzfristig (%s)", bank.Gain, bank.Recognised, bank.Amount, bank.Reason)
	}
	if preview.TotalGain != 100_000 {
		t.Errorf("Ertrag %s € — erwartet 1.000,00 €", preview.TotalGain)
	}
}

// Die Auflösung der Bewertung wird nicht mehr sofort ins Folgejahr geschrieben,
// sondern vom Saldenvortrag gebucht — wie die Auflösung der Abgrenzung. Und der
// Vortrag ist zugleich der Nachholweg: er fragt das Journal und kein Merkzeichen.
func TestCurrencyValuationIsReversedByTheCarryForward(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	closing := env.closing(t)
	svc := env.currency(t, server.URL)
	svc.SetPeriodSource(closing)
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db)))
	closing.SetCurrencyReverser(svc)

	customer := env.customer(t, "Client Inc.", "US", "")
	customer.PaymentTermsDays = 30
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde: %v", err)
	}
	env.foreignOpenItem(t, customer, "RE-USD-1", 1_000_000, 1_100_000)

	booked, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if booked.EntryNumber == "" {
		t.Fatal("die Bewertungsbuchung fehlt")
	}
	// Die Auflösung steht noch nicht: sie kommt mit dem Vortrag.
	if booked.ReversalEntryNumber != "" {
		t.Errorf("Auflösung %q — sie gehört an den Saldenvortrag und nicht in denselben Zug",
			booked.ReversalEntryNumber)
	}
	entries2027, err := env.journalRepo.FindAll(ctx, 2027)
	if err != nil {
		t.Fatalf("Journal 2027: %v", err)
	}
	if len(entries2027) != 0 {
		t.Fatalf("im Folgejahr steht schon etwas: %d Buchungen", len(entries2027))
	}

	// Der Vortrag bucht sie.
	reversals, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung: %v", err)
	}
	if len(reversals) != 1 {
		t.Fatalf("%d Auflösungsbuchungen — erwartet eine", len(reversals))
	}
	reversal := reversals[0]
	if reversal.BookingDate != "2027-01-01" {
		t.Errorf("Auflösung am %s — erwartet den ersten Tag des Folgejahres", reversal.BookingDate)
	}
	if got := debitOn(&reversal, accounting.CurrencyGainAccount); got != 100_000 {
		t.Errorf("die Auflösung muss den Ertrag zurücknehmen: %s € auf %s",
			got, accounting.CurrencyGainAccount)
	}

	// Ein zweiter Lauf bucht sie nicht noch einmal — und wäre der Nachholweg,
	// wenn der erste ausgefallen wäre.
	again, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("%d weitere Auflösungen — eine Bewertung wird einmal aufgelöst", len(again))
	}
}

// -------------------------------------------------------------------------
// Fremdwährungsbewertung: nach dem Storno wird erneut bewertet
// -------------------------------------------------------------------------

// Dieselbe Sackgasse an der Stichtagsbewertung: die Schrittliste meldete nach
// der Generalumkehr richtig „storniert, offen", und der Baustein ließ sich
// trotzdem nicht wiederholen.
func TestCurrencyValuationCanBeBookedAgainAfterAReversal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	svc := env.currency(t, server.URL)
	svc.SetPeriodSource(env.closing(t))
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db)))

	customer := env.customer(t, "Client Inc.", "US", "")
	customer.PaymentTermsDays = 30
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde: %v", err)
	}
	env.foreignOpenItem(t, customer, "RE-USD-1", 1_000_000, 1_100_000)

	booked, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("erste Bewertung: %v", err)
	}
	first := env.entryByNumber(t, 2026, booked.EntryNumber)

	// Ein zweiter Lauf wird abgewiesen, solange die Bewertung steht.
	if _, err := svc.BookCurrencyValuation(ctx, 2026); err == nil {
		t.Fatal("eine stehende Bewertung wird nicht ein zweites Mal gebucht")
	}

	if _, err := env.journal.Reverse(ctx, first.ID,
		"Stichtagskurs war der falsche"); err != nil {
		t.Fatalf("Generalumkehr: %v", err)
	}

	again, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("nach dem Storno muss die Bewertung erneut buchbar sein: %v", err)
	}
	if again.EntryNumber == "" || again.EntryNumber == booked.EntryNumber {
		t.Fatalf("Buchung %q — erwartet eine neue Bewertungsbuchung", again.EntryNumber)
	}

	// Und aufgelöst wird nur die stehende: die stornierte ist bereits
	// zurückgenommen, eine Auflösung dazu wäre eine Wertänderung ohne Grund.
	reversals, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung: %v", err)
	}
	if len(reversals) != 1 {
		t.Fatalf("%d Auflösungsbuchungen — erwartet genau eine für die stehende Bewertung",
			len(reversals))
	}
}
