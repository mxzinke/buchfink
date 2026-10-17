package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// bookedLine describes one expected Soll- oder Haben-Zeile.
type bookedLine struct {
	side    domain.Side
	account string
	amount  domain.Cents
}

func (b bookedLine) String() string {
	name := "SOLL"
	if b.side == domain.SideCredit {
		name = "HABEN"
	}
	return fmt.Sprintf("%s %s %s €", name, b.account, b.amount)
}

// assertLines compares a booking against the expected Buchungssatz, order
// independent, and always checks that Soll equals Haben.
func assertLines(t *testing.T, entry *domain.JournalEntry, want []bookedLine) {
	t.Helper()

	if !entry.IsBalanced() {
		t.Errorf("Buchung ist nicht ausgeglichen: Soll %s € / Haben %s €", entry.DebitTotal(), entry.CreditTotal())
	}

	got := make([]bookedLine, 0, len(entry.Lines))
	for _, l := range entry.Lines {
		got = append(got, bookedLine{side: l.Side, account: l.Account, amount: l.Amount})
	}

	key := func(b bookedLine) string { return fmt.Sprintf("%s|%s|%d", b.side, b.account, b.amount) }
	sortLines := func(s []bookedLine) { sort.Slice(s, func(i, j int) bool { return key(s[i]) < key(s[j]) }) }
	sortLines(got)
	sorted := append([]bookedLine(nil), want...)
	sortLines(sorted)

	if len(got) != len(sorted) {
		t.Fatalf("Buchung hat %d Zeilen, erwartet %d\nerhalten: %v\nerwartet: %v", len(got), len(sorted), got, sorted)
	}
	for i := range got {
		if got[i] != sorted[i] {
			t.Errorf("Zeile %d: erhalten %s, erwartet %s\nvollständig erhalten: %v", i+1, got[i], sorted[i], got)
		}
	}
}

// receipt files a Beleg away and returns the booking request that goes with it.
// Every Eingangsbeleg runs through a Beleg, so the tests do too — that is the
// path the application takes, sealing included.
func (e *testEnv) receipt(t *testing.T, vendorID uint, group string, net domain.Cents, rate domain.TaxRate, treatment domain.TaxTreatment) ReceiptRequest {
	t.Helper()
	filed := e.fileIncoming(t, "beleg.pdf")
	return ReceiptRequest{
		ContactID:       vendorID,
		ReceiptID:       filed.ID,
		BookingDate:     "2026-03-10",
		DocumentDate:    "2026-03-10",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-31",
		Description:     "Testbeleg",
		TaxTreatment:    treatment,
		Positions:       []ReceiptPosition{{PostingGroup: group, Net: net, TaxRate: rate}},
		Settlement:      SettlementOpen,
	}
}

// 7.1 Lieferantenrechnung auf Ziel, Inland, 19 %.
// SOLL 5906 Fremdleistungen 1.000,00 + SOLL 1406 Vorsteuer 190,00
// HABEN Kreditorenkonto 1.190,00
func TestIncomingDomesticInvoiceOnAccount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Agentur GmbH", "DE", "")

	entry, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "5906", 100000},
		{domain.SideDebit, domain.AccountVorsteuer19, 19000},
		{domain.SideCredit, vendor.LedgerAccount, 119000},
	})
}

// Reverse Charge nach § 13b UStG: der Empfänger schuldet die Steuer und zieht
// sie zugleich als Vorsteuer ab. Vier Zeilen, und an den Lieferanten ist nur der
// Nettobetrag zu zahlen.
func TestIncomingReverseChargeProducesFourLines(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Cloud Provider Ireland Ltd.", "IE", "IE6388047V")

	entry, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge))
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "5909", 100000},                          // Fremdleistungen ohne Vorsteuer § 13b
		{domain.SideDebit, domain.AccountVorsteuer13b19, 19000},     // 1407
		{domain.SideCredit, domain.AccountUmsatzsteuer13b19, 19000}, // 3837
		{domain.SideCredit, vendor.LedgerAccount, 100000},           // nur netto an den Lieferanten
	})
}

// Innergemeinschaftlicher Erwerb: Erwerbsteuer und Vorsteuer aus demselben
// Vorgang, Zahlung an den Lieferanten wieder netto.
func TestIncomingIntraCommunityAcquisition(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Hardware BV", "NL", "NL123456789B01")

	entry, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "wareneingang", 200000, domain.TaxRateStandard, domain.TaxTreatmentIntraCommunityAcquisition))
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "5400", 200000},
		{domain.SideDebit, domain.AccountVorsteuerIG19, 38000},     // 1404
		{domain.SideCredit, domain.AccountUmsatzsteuerIG19, 38000}, // 3804
		{domain.SideCredit, vendor.LedgerAccount, 200000},
	})
}

// Ein Beleg mit zwei Steuersätzen — der Normalfall bei Hotel- und
// Bewirtungsrechnungen. Je Satz eine eigene Steuerzeile.
func TestIncomingReceiptWithTwoTaxRates(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Hotel Adler", "DE", "")

	req := env.receipt(t, vendor.ID, "reisekosten", 0, 0, domain.TaxTreatmentDomestic)
	req.Positions = []ReceiptPosition{
		{PostingGroup: "reisekosten", Net: 20000, TaxRate: domain.TaxRateReduced}, // Übernachtung 7 %
		{PostingGroup: "reisekosten", Net: 5000, TaxRate: domain.TaxRateStandard}, // Frühstück 19 %
	}

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "6650", 20000},
		{domain.SideDebit, "6650", 5000},
		{domain.SideDebit, domain.AccountVorsteuer7, 1400}, // 7 % von 200,00
		{domain.SideDebit, domain.AccountVorsteuer19, 950}, // 19 % von 50,00
		{domain.SideCredit, vendor.LedgerAccount, 27350},
	})
}

// „Sofort bezahlt" bucht gegen das gewählte Bankkonto.
func TestImmediatePaymentUsesTheChosenMeansOfPayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Schreibwaren Meier", "DE", "")

	for _, account := range []string{domain.AccountBank, "1810"} {
		req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		req.Settlement = SettlementPaid
		req.PaymentAccount = account

		entry, err := env.posting.PostIncomingReceipt(ctx, req)
		if err != nil {
			t.Fatalf("Zahlung über %s: %v", account, err)
		}
		assertLines(t, entry, []bookedLine{
			{domain.SideDebit, "6815", 5000},
			{domain.SideDebit, domain.AccountVorsteuer19, 950},
			{domain.SideCredit, account, 5950},
		})
	}

	// Ohne Angabe des Zahlungsmittels darf nicht gebucht werden.
	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Settlement = SettlementPaid
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("bei sofortiger Zahlung muss das Zahlungsmittel verlangt werden")
	}

	// Ein Aufwandskonto ist kein Zahlungsmittel.
	req.PaymentAccount = "6815"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("ein Aufwandskonto darf nicht als Zahlungsmittel akzeptiert werden")
	}

	// Eine Kasse führt Buchfink nicht: jede Zahlung läuft über ein Bankkonto.
	for _, cash := range []string{"1600", "1610", "1620"} {
		req.PaymentAccount = cash
		if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
			t.Errorf("die Kasse %s darf nicht als Zahlungsmittel akzeptiert werden", cash)
		}
	}
}

// 7.2 Ausgangsrechnung auf Ziel, Inland, 19 %.
func TestOutgoingDomesticInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde Alpha GmbH", "DE", "")

	inv := &domain.Invoice{
		ContactID:       customer.ID,
		Date:            "2026-03-15",
		ServiceDateFrom: "2026-03-01",
		ServiceDateTo:   "2026-03-31",
		TaxTreatment:    domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{
			{Description: "Beratungsleistung", QuantityMilli: 1000, UnitPrice: 200000, TaxRate: domain.TaxRateStandard},
		},
	}

	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if inv.InvoiceNumber != "RE-2026-0001" {
		t.Errorf("Rechnungsnummer = %q, erwartet RE-2026-0001", inv.InvoiceNumber)
	}
	if inv.GrossAmount != 238000 {
		t.Errorf("Bruttobetrag = %s €, erwartet 2.380,00", inv.GrossAmount)
	}

	entry, err := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	if err != nil {
		t.Fatalf("Buchung laden: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, customer.LedgerAccount, 238000},
		{domain.SideCredit, "4400", 200000},
		{domain.SideCredit, domain.AccountUmsatzsteuer19, 38000},
	})
}

// Leistung an einen Unternehmer im übrigen Gemeinschaftsgebiet: die Steuer
// schuldet der Empfänger, eigenes Erlöskonto 4337, keine Steuerzeile.
func TestOutgoingEUService(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	inv := &domain.Invoice{
		ContactID:       customer.ID,
		Date:            "2026-03-15",
		ServiceDateFrom: "2026-03-15",
		ServiceDateTo:   "2026-03-15",
		TaxTreatment:    domain.TaxTreatmentReverseChargeSupply,
		Items: []domain.InvoiceItem{
			// Ohne Steuersatz: die Rechnung weist keine Steuer aus, und ein Satz
			// an der Position wäre ein Ausweis nach § 14c UStG (siehe
			// TestIssueRefusesUnlawfulTax).
			{Description: "Beratung", QuantityMilli: 1000, UnitPrice: 500000, TaxRate: domain.TaxRateNone},
		},
	}

	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.TaxAmount != 0 {
		t.Errorf("die Steuer schuldet der Empfänger, ausgewiesen wurden aber %s €", inv.TaxAmount)
	}

	entry, _ := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, customer.LedgerAccount, 500000},
		{domain.SideCredit, "4337", 500000},
	})
}

// Die Stammdaten müssen zum Steuerfall passen: ohne USt-IdNr. keine Rechnung
// nach § 13b UStG (§ 14a Abs. 1 UStG). Warenlieferungen ins Ausland kennt
// Buchfink nicht.
func TestTaxTreatmentIsValidatedAgainstMasterData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	withoutVatID := env.customer(t, "Client SARL", "FR", "")
	inv := &domain.Invoice{
		ContactID: withoutVatID.ID, Date: "2026-03-15",
		ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
		TaxTreatment: domain.TaxTreatmentReverseChargeSupply,
		Items:        []domain.InvoiceItem{{Description: "Beratung", QuantityMilli: 1000, UnitPrice: 10000, TaxRate: domain.TaxRateNone}},
	}
	err := env.invoices(t).Issue(ctx, inv)
	if err == nil {
		t.Fatal("ohne USt-IdNr. darf keine Rechnung nach § 13b UStG ausgestellt werden")
	}
	if !strings.Contains(err.Error(), "USt-IdNr") {
		t.Errorf("die Fehlermeldung sollte die fehlende USt-IdNr. benennen, lautet aber: %v", err)
	}

	withVatID := env.customer(t, "Client Lyon SARL", "FR", "FR98765432109")
	for _, treatment := range []domain.TaxTreatment{"intra_community_supply", "export"} {
		goods := &domain.Invoice{
			ContactID: withVatID.ID, Date: "2026-03-15",
			ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
			TaxTreatment: treatment,
			Items:        []domain.InvoiceItem{{Description: "Ware", QuantityMilli: 1000, UnitPrice: 10000, TaxRate: domain.TaxRateNone}},
		}
		if err := env.invoices(t).Issue(ctx, goods); err == nil {
			t.Errorf("der Steuerfall %q darf nicht ausgestellt werden", treatment)
		} else if !strings.Contains(err.Error(), "gibt es für Ausgangsrechnungen nicht") {
			t.Errorf("die Meldung sollte den unbekannten Steuerfall benennen: %v", err)
		}
	}
}

// Ein Lieferant kann keine Ausgangsrechnung erhalten und ein Kunde keinen
// Eingangsbeleg stellen — sonst landen Forderungen und Verbindlichkeiten auf der
// falschen Bilanzseite.
func TestDirectionMustMatchPartnerType(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde", "DE", "")
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, customer.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err == nil {
		t.Error("ein Kunde darf keinen Eingangsbeleg stellen")
	}

	vendor := env.vendor(t, "Lieferant", "DE", "")
	inv := &domain.Invoice{
		ContactID: vendor.ID, Date: "2026-03-15",
		ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
		Items: []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 10000, TaxRate: domain.TaxRateStandard}},
	}
	if err := env.invoices(t).Issue(ctx, inv); err == nil {
		t.Error("ein Lieferant darf keine Ausgangsrechnung erhalten")
	}
}

// Die Steuer wird einmal je Steuersatzgruppe gerundet, nicht je Position.
// Eine Rundung je Position ergäbe hier 3 × 0,63 = 1,89 statt 1,90 auf die Summe.
func TestTaxIsRoundedOncePerRateGroup(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 0, 0, domain.TaxTreatmentDomestic)
	req.Positions = []ReceiptPosition{
		{PostingGroup: "buerobedarf", Net: 333, TaxRate: domain.TaxRateStandard},
		{PostingGroup: "buerobedarf", Net: 333, TaxRate: domain.TaxRateStandard},
		{PostingGroup: "buerobedarf", Net: 333, TaxRate: domain.TaxRateStandard},
	}

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	var tax domain.Cents
	for _, l := range entry.Lines {
		if l.Account == domain.AccountVorsteuer19 {
			tax = l.Amount
		}
	}
	// 19 % von 9,99 € = 1,8981 € → 1,90 €
	if tax != 190 {
		t.Errorf("Vorsteuer = %s €, erwartet 1,90 (einmal auf die Summe gerundet, nicht je Position)", tax)
	}
	if !entry.IsBalanced() {
		t.Error("die Buchung muss trotz Rundung exakt ausgeglichen sein")
	}
}

// Nach dem Storno einer Ausgangsrechnung darf auf dem Erlöskonto und beim
// Kunden nichts stehen bleiben.
func TestInvoiceCancellationClearsReceivableAndRevenue(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde Alpha", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-15",
		ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items:        []domain.InvoiceItem{{Description: "Leistung", QuantityMilli: 1000, UnitPrice: 100000, TaxRate: domain.TaxRateStandard}},
	}
	invoices := env.invoices(t)
	if err := invoices.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if err := invoices.Cancel(ctx, inv.ID, "Leistung nicht erbracht"); err != nil {
		t.Fatalf("Rechnung stornieren: %v", err)
	}

	turnovers, err := env.journalRepo.AccountTurnovers(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Verkehrszahlen: %v", err)
	}
	for _, account := range []string{"4400", domain.AccountUmsatzsteuer19, customer.LedgerAccount} {
		tn := turnovers[account]
		if tn.Debit != 0 || tn.Credit != 0 {
			t.Errorf("Konto %s: nach dem Storno müssen die Verkehrszahlen null sein, sind aber Soll %s / Haben %s",
				account, tn.Debit, tn.Credit)
		}
	}

	contacts, err := env.contacts.GetContacts(ctx, 2026)
	if err != nil {
		t.Fatalf("Kontakte: %v", err)
	}
	for _, c := range contacts {
		if c.ID == customer.ID && c.OpenAmount != 0 {
			t.Errorf("der offene Posten des Kunden muss nach dem Storno null sein, ist aber %s €", c.OpenAmount)
		}
	}
}

// Die Summen- und Saldenliste muss exakt ausgeglichen sein — ohne Toleranz.
func TestSuSaIsExactlyBalanced(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")
	euVendor := env.vendor(t, "SaaS Ireland Ltd.", "IE", "IE6388047V")
	customer := env.customer(t, "Kunde", "DE", "")

	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "buerobedarf", 3333, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
		t.Fatalf("Eingangsbeleg: %v", err)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, euVendor.ID, "software", 9999, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge)); err != nil {
		t.Fatalf("§ 13b-Beleg: %v", err)
	}

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-15",
		ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{
			{Description: "Leistung A", QuantityMilli: 1500, UnitPrice: 3333, TaxRate: domain.TaxRateStandard},
			{Description: "Leistung B", QuantityMilli: 1000, UnitPrice: 777, TaxRate: domain.TaxRateReduced},
		},
	}
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	susa, err := env.accounting.GetSuSaOverview(ctx, 2026)
	if err != nil {
		t.Fatalf("SuSa: %v", err)
	}
	if !susa.IsBalanced || susa.Difference != 0 {
		t.Errorf("die SuSa muss exakt ausgeglichen sein, Differenz ist aber %s € (Soll %s / Haben %s)",
			susa.Difference, susa.TotalDebit, susa.TotalCredit)
	}
}

// Ein fehlender Steuerfall wird abgewiesen, nicht auf "Inland" gesetzt.
//
// Der Fall entsteht real: liest der E-Rechnungs-Leser einen Kategoriecode, für
// den es keine deutsche Entsprechung gibt, oder mischt die Rechnung mehrere
// Kategorien, dann enthält er sich bewusst. Ein stiller Vorgabewert würde daraus
// eine Buchung mit Vorsteuerabzug machen, die richtig aussieht.
func TestIncomingReceiptWithoutTreatmentIsRefused(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Agentur GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.TaxTreatment = ""

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("eine Buchung ohne Steuerfall wurde angenommen")
	} else if !strings.Contains(err.Error(), "Steuerfall") {
		t.Fatalf("die Meldung nennt den Grund nicht: %v", err)
	}

	if _, err := env.posting.PreviewIncomingReceipt(ctx, req); err == nil {
		t.Fatal("die Vorschau rechnete ohne Steuerfall")
	}
}
