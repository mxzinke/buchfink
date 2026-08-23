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
	filed := e.fileIncoming(t, "lieferantenrechnung.pdf")
	entry, err := e.posting.PostIncomingReceipt(context.Background(), ReceiptRequest{
		ContactID:       vendorID,
		ReceiptID:       filed.ID,
		BookingDate:     "2026-03-01",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
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

// Ein gewährtes Skonto mindert nach § 17 Abs. 1 Satz 1 UStG den geschuldeten
// Steuerbetrag — und mit ihm die Bemessungsgrundlage, aus der er stammt. Beide
// müssen in derselben Voranmeldung in dieselbe Richtung gehen; eine gesunkene
// Steuer neben einem gestiegenen Umsatz sind zwei Zahlen, die einander
// widersprechen.
func TestGrantedSkontoReducesTheReportedBase(t *testing.T) {
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

	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  *inv.JournalEntryID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 2380,
		}},
	}); err != nil {
		t.Fatalf("Zahlungseingang mit Skonto: %v", err)
	}

	summary, err := NewVatService(env.journalRepo, env.fiscalYear).Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}
	if len(summary.TaxableRevenue) != 1 {
		t.Fatalf("erwartet eine Steuersatzgruppe, erhalten %d", len(summary.TaxableRevenue))
	}
	figure := summary.TaxableRevenue[0]

	// 1.000,00 € Umsatz abzüglich 20,00 € Skonto, 190,00 € Steuer abzüglich 3,80 €.
	if figure.Net != 98000 {
		t.Errorf("die Bemessungsgrundlage muss nach dem Skonto 980,00 € betragen, ist aber %s €", figure.Net)
	}
	if figure.Tax != 18620 {
		t.Errorf("die Umsatzsteuer muss nach dem Skonto 186,20 € betragen, ist aber %s €", figure.Tax)
	}
	// Die Probe, die den Fehler ursprünglich sichtbar macht: Steuer und
	// Grundlage müssen zueinander passen.
	if want := domain.TaxRateStandard.Tax(figure.Net); figure.Tax != want {
		t.Errorf("%s € Steuer passen nicht zu %s € Bemessungsgrundlage (erwartet %s €)", figure.Tax, figure.Net, want)
	}
	if summary.OutputTax != 18620 {
		t.Errorf("die vereinnahmte Umsatzsteuer muss 186,20 € betragen, ist aber %s €", summary.OutputTax)
	}
}

// § 17 Abs. 1 Satz 5 UStG erstreckt die Berichtigung auf die Fälle des § 13b:
// dort schuldet der Empfänger die Steuer und zieht sie zugleich ab, also sind
// beide Seiten zu berichtigen. Dass sie sich im Ergebnis ausgleichen, ist kein
// Grund, sie wegzulassen — UStAE 17.1 Abs. 3 sagt das ausdrücklich.
func TestSkontoOnReverseChargeCorrectsBothLegs(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Cloud Provider Ireland Ltd.", "IE", "IE6388047V")

	filed := env.fileIncoming(t, "cloud.pdf")
	invoice, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID:       vendor.ID,
		ReceiptID:       filed.ID,
		BookingDate:     "2026-03-01",
		DocumentDate:    "2026-03-01",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-01",
		Description:     "Cloud-Leistung",
		TaxTreatment:    domain.TaxTreatmentReverseCharge,
		Positions:       []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 100000, TaxRate: domain.TaxRateStandard}},
		Settlement:      SettlementOpen,
	})
	if err != nil {
		t.Fatalf("Eingangsrechnung: %v", err)
	}

	// Bei § 13b ist die Rechnung netto: offen sind 1.000,00 €, 2 % Skonto sind
	// 20,00 € — und die sind vollständig Bemessungsgrundlage, nicht brutto.
	entry, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  invoice.ID,
			SettledAmount:    100000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 2000,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung mit Skonto: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, vendor.LedgerAccount, 100000},
		{domain.SideCredit, "5730", 2000},                        // Erhaltene Skonti ohne Vorsteuer
		{domain.SideCredit, domain.AccountVorsteuer13b19, 380},   // Vorsteuerberichtigung
		{domain.SideDebit, domain.AccountUmsatzsteuer13b19, 380}, // Berichtigung der geschuldeten Steuer
		{domain.SideCredit, domain.AccountBank, 98000},
	})

	summary, err := NewVatService(env.journalRepo, env.fiscalYear).Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}
	if summary.ReverseChargeBase != 98000 {
		t.Errorf("die Bemessungsgrundlage nach § 13b muss 980,00 € betragen, ist aber %s €", summary.ReverseChargeBase)
	}
	if summary.ReverseChargeTax != 18620 {
		t.Errorf("die geschuldete Steuer nach § 13b muss 186,20 € betragen, ist aber %s €", summary.ReverseChargeTax)
	}
	if summary.InputTax != 18620 {
		t.Errorf("die abziehbare Vorsteuer muss 186,20 € betragen, ist aber %s €", summary.InputTax)
	}
	if summary.Payable != 0 {
		t.Errorf("bei § 13b gleichen sich Steuer und Vorsteuer aus, die Zahllast ist aber %s €", summary.Payable)
	}
}

// Ein offener Posten schließt nicht zum Jahreswechsel. § 252 Abs. 1 Nr. 5 HGB
// legt den Ertrag in das Jahr der Leistung und die Zahlung in das Jahr, in dem
// sie fließt — die Dezemberrechnung und ihr Ausgleich liegen also planmäßig in
// verschiedenen Jahren. Zählte nur das laufende Jahr, stünde die bezahlte
// Rechnung weiter offen und ließe sich ein zweites Mal ausgleichen.
func TestSettlementIsSeenAcrossTheFiscalYearBoundary(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")

	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	// Bezahlt im Januar des Folgejahres — die Buchung landet in 2027.
	payment, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2027-01-15",
		Allocations: []AllocationRequest{{
			OpenItemEntryID: invoice.ID,
			SettledAmount:   119000,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung im Folgejahr: %v", err)
	}
	if payment.FiscalYear != 2027 {
		t.Fatalf("die Zahlung gehört in das Wirtschaftsjahr 2027, gebucht wurde %d", payment.FiscalYear)
	}

	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	for _, item := range items {
		if item.EntryID == invoice.ID {
			t.Errorf("die Rechnung ist bezahlt, steht aber mit %s € weiter offen", item.OpenAmount)
		}
	}
}

// Und dieselbe Rechnung muss im Folgejahr überhaupt noch auswählbar sein:
// wäre die Liste auf das laufende Jahr begrenzt, ließe sie sich nicht mehr
// ausgleichen — sie stünde in keiner Auswahl.
func TestOpenItemsFromEarlierYearsStayVisible(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	payments := env.payments(t)
	payments.SetFiscalYear(2027)

	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	var found *domain.OpenItem
	for i := range items {
		if items[i].EntryID == invoice.ID {
			found = &items[i]
		}
	}
	if found == nil {
		t.Fatal("die Rechnung aus dem Vorjahr steht in keiner Auswahl und ließe sich nie mehr ausgleichen")
	}
	if found.OpenAmount != 119000 {
		t.Errorf("offen sind %s €, erwartet 1.190,00", found.OpenAmount)
	}

	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2027-02-01",
		Allocations: []AllocationRequest{{
			OpenItemEntryID: invoice.ID,
			SettledAmount:   119000,
		}},
	}); err != nil {
		t.Fatalf("die Rechnung aus dem Vorjahr ließ sich nicht ausgleichen: %v", err)
	}
}

// Wird eine Zahlung storniert, ist der Posten wieder offen. Die
// Zuordnungszeilen bleiben beim Storno stehen — zählten sie weiter, wäre die
// Rechnung ausgeglichen, ohne dass jemand gezahlt hätte.
func TestReversedPaymentReopensTheItem(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	payment, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations: []AllocationRequest{{
			OpenItemEntryID: invoice.ID,
			SettledAmount:   119000,
		}},
	})
	if err != nil {
		t.Fatalf("Zahlung: %v", err)
	}

	if _, err := env.journal.Reverse(ctx, payment.ID, "Zahlung war dem falschen Posten zugeordnet"); err != nil {
		t.Fatalf("Storno der Zahlung: %v", err)
	}

	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	var found *domain.OpenItem
	for i := range items {
		if items[i].EntryID == invoice.ID {
			found = &items[i]
		}
	}
	if found == nil {
		t.Fatal("nach dem Storno der Zahlung muss der Posten wieder offen sein")
	}
	if found.OpenAmount != 119000 {
		t.Errorf("offen sind %s €, erwartet wieder 1.190,00", found.OpenAmount)
	}
}

// Eine Generalumkehr wird auf den Tag der Korrektur datiert, nie zurück in den
// Zeitraum der falschen Buchung. Sie liegt damit regelmäßig in einem späteren
// Wirtschaftsjahr als die Buchung, die sie aufhebt. Wer den Blick auf das
// Vorjahr stellt, darf die stornierte Rechnung trotzdem nicht als offenen
// Posten wiederfinden — dort wäre sie Geld, das niemand mehr schuldet, und
// bezahlbar wäre sie auch noch.
func TestReversedInvoiceStaysClosedInAnEarlierYearView(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	filed := env.fileIncoming(t, "vorjahresrechnung.pdf")
	invoice, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID:       vendor.ID,
		ReceiptID:       filed.ID,
		BookingDate:     "2025-12-20",
		DocumentDate:    "2025-12-20",
		ServiceDateFrom: "2025-12-20",
		ServiceDateTo:   "2025-12-20",
		Description:     "Rechnung aus dem Vorjahr",
		TaxTreatment:    domain.TaxTreatmentDomestic,
		Positions:       []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 100000, TaxRate: domain.TaxRateStandard}},
		Settlement:      SettlementOpen,
	})
	if err != nil {
		t.Fatalf("Rechnung im Vorjahr: %v", err)
	}
	if invoice.FiscalYear != 2025 {
		t.Fatalf("die Rechnung gehört in das Wirtschaftsjahr 2025, gebucht wurde %d", invoice.FiscalYear)
	}

	// Storniert heute — die Generalumkehr landet damit in einem späteren Jahr.
	reversal, err := env.journal.Reverse(ctx, invoice.ID, "doppelt erfasst")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}
	if reversal.FiscalYear <= invoice.FiscalYear {
		t.Fatalf("der Test setzt voraus, dass das Storno in einem späteren Jahr liegt (%d, Rechnung %d)",
			reversal.FiscalYear, invoice.FiscalYear)
	}

	payments := env.payments(t)
	payments.SetFiscalYear(invoice.FiscalYear)

	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	for _, item := range items {
		if item.EntryID == invoice.ID {
			t.Errorf("die stornierte Rechnung steht mit %s € als offener Posten da", item.OpenAmount)
		}
	}
}
