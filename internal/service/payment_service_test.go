package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) payments(t *testing.T) *PaymentService {
	t.Helper()
	return NewPaymentService(
		e.journal,
		e.journalRepo,
		repository.NewPaymentAllocationRepository(e.db),
		e.contactRepo,
		repository.NewBankRepository(e.db),
		e.fiscalYear,
	)
}

// openPayable books a supplier invoice on account and returns its open item.
func (e *testEnv) openPayable(t *testing.T, vendorID uint, net domain.Cents, rate domain.TaxRate) *domain.JournalEntry {
	t.Helper()
	entry, err := e.posting.PostIncomingReceipt(context.Background(), ReceiptRequest{
		ContactID:       vendorID,
		BookingDate:     "2026-03-01",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
		DocumentNumber:  "ER-2026-0001",
		Description:     "Lieferantenrechnung",
		TaxTreatment:    domain.TaxTreatmentDomestic,
		Positions:       []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: net, TaxRate: rate}},
		Settlement:      SettlementOpen,
	})
	if err != nil {
		t.Fatalf("Eingangsrechnung buchen: %v", err)
	}
	return entry
}

func TestOpenItemsReflectTheJournal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")
	env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	items, err := env.payments(t).OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("erwartet 1 offener Posten, erhalten %d", len(items))
	}
	item := items[0]
	if item.OpenAmount != 119000 {
		t.Errorf("offener Betrag = %s €, erwartet 1.190,00", item.OpenAmount)
	}
	if item.Status() != "offen" {
		t.Errorf("Status = %q, erwartet \"offen\"", item.Status())
	}
	if item.TaxRate != domain.TaxRateStandard {
		t.Errorf("der Steuersatz des Belegs muss aus der Buchung ableitbar sein, erhalten %s", item.TaxRate.Label())
	}
}

// Vollzahlung: Verbindlichkeit an Bank, der Posten schließt.
func TestFullPaymentClosesOpenItem(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 119000}},
	})
	if err != nil {
		t.Fatalf("Zahlung: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 119000},
		{domain.SideCredit, domain.AccountBank, 119000},
	})

	items, _ := payments.OpenItems(ctx)
	if len(items) != 0 {
		t.Errorf("nach der Vollzahlung darf kein offener Posten bleiben, es sind aber %d", len(items))
	}
}

// Teilzahlung: der Rest bleibt stehen, der Status wechselt auf teilbezahlt.
func TestPartialPaymentLeavesRemainder(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 50000}},
	}); err != nil {
		t.Fatalf("erste Rate: %v", err)
	}

	items, _ := payments.OpenItems(ctx)
	if len(items) != 1 {
		t.Fatalf("erwartet 1 offener Posten, erhalten %d", len(items))
	}
	if items[0].OpenAmount != 69000 {
		t.Errorf("Restbetrag = %s €, erwartet 690,00", items[0].OpenAmount)
	}
	if items[0].Status() != "teilbezahlt" {
		t.Errorf("Status = %q, erwartet \"teilbezahlt\"", items[0].Status())
	}

	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-04-05",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 69000}},
	}); err != nil {
		t.Fatalf("zweite Rate: %v", err)
	}
	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("nach der Restzahlung darf kein offener Posten bleiben, es sind aber %d", len(items))
	}

	// Mehr als offen ist darf nicht zugeordnet werden.
	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-04-06",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 100}},
	}); err == nil {
		t.Error("ein bereits ausgeglichener Posten darf nicht erneut zugeordnet werden")
	}
}

// Skonto mindert nach § 17 UStG auch die Bemessungsgrundlage — die
// Steuerkorrektur ist der Teil, den eine reine Nettobuchung unterschlägt.
func TestSkontoCorrectsTheTaxBase(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	// 2 % Skonto auf 1.190,00 € brutto = 23,80 € (20,00 netto + 3,80 Steuer).
	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  invoice.ID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 2380,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Skonto: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 119000},
		{domain.SideCredit, "5736", 2000},                   // erhaltene Skonti 19 % Vorsteuer
		{domain.SideCredit, domain.AccountVorsteuer19, 380}, // Vorsteuerkorrektur
		{domain.SideCredit, domain.AccountBank, 116620},     // tatsächlich geflossen
	})

	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("mit Skonto ist der Posten vollständig ausgeglichen, es bleiben aber %d offen", len(items))
	}

	// Die Vorsteuer steht nach der Korrektur auf 190,00 − 3,80 = 186,20 €.
	turnovers, _ := env.journalRepo.AccountTurnovers(ctx, env.fiscalYear)
	vst := turnovers[domain.AccountVorsteuer19]
	if net := vst.Debit - vst.Credit; net != 18620 {
		t.Errorf("die abziehbare Vorsteuer muss nach dem Skonto 186,20 € betragen, ist aber %s €", net)
	}
}

// Gewährtes Skonto auf der Erlösseite, mit Umsatzsteuerkorrektur.
func TestGrantedSkontoOnReceivable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	customer := env.customer(t, "Kunde", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items:        []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 100000, TaxRate: domain.TaxRateStandard}},
	}
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  *inv.JournalEntryID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 2380,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlungseingang mit Skonto: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideCredit, customer.LedgerAccount, 119000},
		{domain.SideDebit, "4736", 2000},                      // gewährte Skonti 19 % USt
		{domain.SideDebit, domain.AccountUmsatzsteuer19, 380}, // Umsatzsteuerkorrektur
		{domain.SideDebit, domain.AccountBank, 116620},
	})
}

// Bankgebühr: die Bank hat mehr abgebucht als die Rechnung ausweist.
func TestBankFeeIsBookedSeparately(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  invoice.ID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceBankFee,
			DifferenceAmount: 500,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Gebühr: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 119000},
		{domain.SideDebit, domain.AccountNebenkostenGeld, 500},
		{domain.SideCredit, domain.AccountBank, 119500},
	})
}

// Kleinbetragsdifferenz: der Rest wird ausgebucht, statt als Cent-Leiche
// stehen zu bleiben.
func TestRoundingDifferenceClosesTheItem(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  invoice.ID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceRounding,
			DifferenceAmount: 3,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Rundungsdifferenz: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 119000},
		{domain.SideCredit, "4830", 3}, // sonstige betriebliche Erträge
		{domain.SideCredit, domain.AccountBank, 118997},
	})
	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("der Posten muss geschlossen sein, es bleiben aber %d offen", len(items))
	}
}

// Sammelüberweisung: eine Zahlung gleicht mehrere Belege aus.
func TestOnePaymentSettlesSeveralOpenItems(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")

	first := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)
	second := env.openPayable(t, vendor.ID, 50000, domain.TaxRateStandard)

	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations: []AllocationRequest{
			{OpenItemEntryID: first.ID, SettledAmount: 119000},
			{OpenItemEntryID: second.ID, SettledAmount: 59500},
		},
	})
	if err != nil {
		t.Fatalf("Sammelzahlung: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 119000},
		{domain.SideDebit, vendor.LedgerAccount, 59500},
		{domain.SideCredit, domain.AccountBank, 178500},
	})
	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("beide Posten müssen geschlossen sein, es bleiben aber %d offen", len(items))
	}
}

// Bei einer Bankzuordnung muss die Summe exakt dem Umsatz entsprechen —
// sonst wäre ein vertipptes Skonto eine stille Falschbuchung.
func TestBankAllocationMustMatchStatementAmount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	bankRepo := repository.NewBankRepository(env.db)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	if _, err := bankRepo.CreateBatch(ctx, []domain.BankTransaction{{
		FiscalYear: 2026, AccountIBAN: "DE02120300000000202051",
		BookingDate: "2026-03-20", ValueDate: "2026-03-20",
		Amount: -119000, Currency: "EUR", CounterpartyName: "Lieferant",
		EndToEndID: "E2E-1", MatchStatus: domain.MatchStatusUnmatched,
		LedgerAccount: domain.AccountBank,
	}}); err != nil {
		t.Fatalf("Bankumsatz anlegen: %v", err)
	}
	txs, _ := bankRepo.FindAll(ctx, 2026)
	if len(txs) != 1 {
		t.Fatalf("erwartet 1 Bankumsatz, erhalten %d", len(txs))
	}
	txID := txs[0].ID

	// Zu wenig zugeordnet: muss abgewiesen werden.
	if _, err := payments.Settle(ctx, PaymentRequest{
		BankTxID:    &txID,
		Allocations: []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 100000}},
	}); err == nil {
		t.Error("eine Zuordnung, die nicht dem Bankumsatz entspricht, muss abgewiesen werden")
	}

	// Passend zugeordnet: geht durch und markiert den Umsatz.
	if _, err := payments.Settle(ctx, PaymentRequest{
		BankTxID:    &txID,
		Allocations: []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 119000}},
	}); err != nil {
		t.Fatalf("passende Zuordnung: %v", err)
	}

	updated, _ := bankRepo.FindByID(ctx, txID)
	if updated.MatchStatus != domain.MatchStatusMatched {
		t.Errorf("der Bankumsatz muss als zugeordnet markiert sein, ist aber %q", updated.MatchStatus)
	}
}

// Ein stornierter Beleg darf keinen offenen Posten mehr erzeugen.
func TestReversedDocumentHasNoOpenItem(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	if _, err := env.journal.Reverse(ctx, invoice.ID, "Beleg doppelt erfasst"); err != nil {
		t.Fatalf("Storno: %v", err)
	}

	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("ein stornierter Beleg darf keinen offenen Posten hinterlassen, es sind aber %d", len(items))
	}
}
