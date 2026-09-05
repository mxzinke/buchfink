package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Anzahlungsfall, von der Abschlagsrechnung bis zur Schlussrechnung.
//
// Er verlässt den gewöhnlichen Rechnungsweg an zwei Stellen: die Steuer
// entsteht mit der Vereinnahmung (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG), und
// die Schlussrechnung muss die berechneten und vereinnahmten Teilentgelte samt
// Steuer absetzen (§ 14 Abs. 5 Satz 2 UStG). Wer das Zweite vergisst, weist die
// Steuer zweimal aus und schuldet den Mehrbetrag.

func (e *testEnv) group(t *testing.T, svc *InvoiceService, customerID uint, totalNet domain.Cents) *domain.InvoiceGroup {
	t.Helper()
	group, err := svc.CreateInvoiceGroup(context.Background(), AdvanceGroupRequest{
		ContactID: customerID, Title: "Umbau Ladenlokal",
		TotalNet: totalNet, TaxRate: domain.TaxRateStandard,
	})
	if err != nil {
		t.Fatalf("Rechnungsverbund anlegen: %v", err)
	}
	return group
}

// lineAmount sucht den Betrag einer Buchungszeile auf einem Konto.
func lineAmount(t *testing.T, entry *domain.JournalEntry, account string, side domain.Side) domain.Cents {
	t.Helper()
	var sum domain.Cents
	found := false
	for _, l := range entry.Lines {
		if l.Account == account && l.Side == side {
			sum += l.Amount
			found = true
		}
	}
	if !found {
		t.Errorf("in der Buchung %s fehlt eine %s-Zeile auf Konto %s (%+v)",
			entry.EntryNumber, side, account, entry.Lines)
	}
	return sum
}

// Die Abschlagsrechnung wird ausgestellt, aber nicht gebucht — und erscheint
// trotzdem als offener Posten.
func TestAdvanceInvoiceIsIssuedWithoutBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	inv, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}

	if inv.Kind != domain.InvoiceKindAdvance {
		t.Errorf("Dokumentart = %q, erwartet %q", inv.Kind, domain.InvoiceKindAdvance)
	}
	if inv.InvoiceNumber == "" {
		t.Error("die Abschlagsrechnung braucht eine Nummer aus demselben Kreis (§ 14 Abs. 5 Satz 1 UStG)")
	}
	if inv.JournalEntryID != nil {
		t.Error("die Abschlagsrechnung darf beim Ausstellen nicht gebucht werden — die Steuer entsteht mit der Vereinnahmung")
	}

	entries, err := env.journalRepo.FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("es steht bereits %d Buchung(en) im Journal, erwartet keine", len(entries))
	}

	// Die OP-Liste kennt sie trotzdem: die zweite Quelle neben dem Journal.
	open, err := svc.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].DocumentNumber != inv.InvoiceNumber {
		t.Fatalf("erwartet den Abschlag in der OP-Liste, erhalten %+v", open)
	}
	if open[0].OpenAmount != inv.GrossAmount {
		t.Errorf("offener Betrag = %s, erwartet %s", open[0].OpenAmount, inv.GrossAmount)
	}
}

// Die Vereinnahmung bucht 1800 an 3272 und 3806 — und die Steuer fällt in den
// Zeitraum der Zahlung, nicht in den der Leistung.
func TestAdvanceSettlementBooksTaxInThePaymentPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	inv, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}

	advance, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: inv.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	})
	if err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}
	if advance.SettlementEntryID == nil {
		t.Fatal("die Vereinnahmung muss eine Buchung erzeugen")
	}

	entry, err := env.journalRepo.FindByID(ctx, *advance.SettlementEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if got := lineAmount(t, entry, domain.AccountBank, domain.SideDebit); got != 476000 {
		t.Errorf("Bank im Soll = %s, erwartet 4760,00", got)
	}
	if got := lineAmount(t, entry, accounting.AccountErhalteneAnzahlungen19, domain.SideCredit); got != 400000 {
		t.Errorf("3272 im Haben = %s, erwartet 4000,00", got)
	}
	if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideCredit); got != 76000 {
		t.Errorf("3806 im Haben = %s, erwartet 760,00", got)
	}

	// Die Steuerzeile trägt den Schlüssel — sonst fiele sie aus der
	// Voranmeldung heraus — und der Zeitraum folgt der Zahlung.
	var taxLine domain.JournalLine
	for _, l := range entry.Lines {
		if l.TaxKey != "" {
			taxLine = l
		}
	}
	if taxLine.TaxKey != "UST19" {
		t.Fatalf("die Steuerzeile trägt den Schlüssel %q, erwartet UST19", taxLine.TaxKey)
	}
	if got := accounting.VatPeriodFor(entry, taxLine, ""); got != "2026-04-15" {
		t.Errorf("die Steuer fällt in den Zeitraum %q, erwartet den Zahlungsmonat 2026-04-15", got)
	}

	// Der vereinnahmte Abschlag ist nicht mehr offen.
	open, err := svc.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 0 {
		t.Errorf("der vereinnahmte Abschlag steht noch als offener Posten: %+v", open)
	}
	// Und ein zweites Mal geht es nicht.
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: inv.ID, PaymentDate: "2026-05-01", PaymentAccount: domain.AccountBank,
	}); err == nil {
		t.Error("eine bereits vereinnahmte Anzahlung darf nicht erneut gebucht werden")
	}
}

// Die Schlussrechnung setzt die vereinnahmten Anzahlungen ab: in der Buchung
// durch Auflösung von 3272 und 3806, im Dokument durch BT-113.
func TestFinalInvoiceSettlesTheAdvances(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	advanceInvoice, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advanceInvoice.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}

	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000,
			UnitPrice: 1000000, TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}

	if final.PrepaidAmount != 476000 {
		t.Errorf("abgesetzte Anzahlungen = %s, erwartet 4760,00", final.PrepaidAmount)
	}
	if final.OpenAmount() != 714000 {
		t.Errorf("Zahlbetrag = %s, erwartet 7140,00", final.OpenAmount())
	}
	if len(final.PrecedingRefs) != 1 || final.PrecedingRefs[0].Number != advanceInvoice.InvoiceNumber {
		t.Errorf("die abgesetzte Abschlagsrechnung fehlt im Bezug (BG-3): %+v", final.PrecedingRefs)
	}

	entry, err := env.journalRepo.FindByID(ctx, *final.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	// Der Gesamtbetrag als Erlös und Umsatzsteuer …
	if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideCredit); got != 190000 {
		t.Errorf("Umsatzsteuer im Haben = %s, erwartet 1900,00", got)
	}
	// … und die Auflösung der Anzahlung im Soll.
	if got := lineAmount(t, entry, accounting.AccountErhalteneAnzahlungen19, domain.SideDebit); got != 400000 {
		t.Errorf("3272 im Soll = %s, erwartet 4000,00", got)
	}
	if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideDebit); got != 76000 {
		t.Errorf("Steuer der Anzahlung im Soll = %s, erwartet 760,00", got)
	}
	// Als Forderung bleibt der Restbetrag.
	if got := lineAmount(t, entry, customer.LedgerAccount, domain.SideDebit); got != 714000 {
		t.Errorf("Forderung = %s, erwartet 7140,00", got)
	}

	// Der Verbund ist abgeschlossen und nimmt keinen weiteren Abschlag auf.
	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || !groups[0].Closed {
		t.Fatalf("der Verbund muss nach der Schlussrechnung abgeschlossen sein: %+v", groups)
	}
	if _, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-12-01", Net: 10000,
	}); err == nil {
		t.Error("nach der Schlussrechnung darf kein Abschlag mehr entstehen")
	}
	if _, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-12-01",
		Items: []domain.InvoiceItem{{Description: "nochmal", QuantityMilli: 1000, UnitPrice: 100, TaxRate: domain.TaxRateStandard}},
	}); err == nil {
		t.Error("ein Verbund hat höchstens eine Schlussrechnung")
	}
}

// Die Summe der Abschläge darf den vereinbarten Gesamtbetrag nicht
// überschreiten.
func TestAdvancesMayNotExceedTheAgreedTotal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	if _, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 700000,
	}); err != nil {
		t.Fatalf("erster Abschlag: %v", err)
	}
	_, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-04-01", Net: 400000,
	})
	if err == nil || !strings.Contains(err.Error(), "Gesamtbetrag") {
		t.Errorf("die Überschreitung muss abgewiesen werden, erhalten: %v", err)
	}
}

// Ein stornierter, nicht vereinnahmter Abschlag fällt aus der Verrechnung
// heraus: mit ihm entfällt die Rechnung im Sinne des § 14 Abs. 5 Satz 2 UStG
// und damit der Grund für die Absetzung.
func TestCancelledAdvanceDropsOutOfTheSettlement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	first, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.CancelWithDocument(ctx, first.ID, "Auftrag geändert"); err != nil {
		t.Fatalf("Abschlag stornieren: %v", err)
	}

	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("erwartet einen Verbund, erhalten %d", len(groups))
	}
	if got := groups[0].DeductibleAdvances(); len(got) != 0 {
		t.Errorf("der stornierte Abschlag ist weiterhin abzusetzen: %+v", got)
	}
	if got := groups[0].ComputeProgress().BilledNet; got != 0 {
		t.Errorf("abgerechnet = %s, erwartet 0 — der Storno nimmt den Abschlag heraus", got)
	}
	// Und er steht nicht mehr als offener Posten: sonst forderte die
	// Schlussrechnung denselben Betrag ein zweites Mal.
	open, err := svc.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 0 {
		t.Errorf("der stornierte Abschlag steht noch als offener Posten: %+v", open)
	}

	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau", QuantityMilli: 1000, UnitPrice: 1000000, TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}
	if final.PrepaidAmount != 0 {
		t.Errorf("die Schlussrechnung setzt %s ab, erwartet nichts", final.PrepaidAmount)
	}
}

// Eine vereinnahmte Anzahlung wird zuerst zurückgezahlt und dann storniert.
//
// Der Grund ist die Steuer: sie ist mit der Vereinnahmung entstanden
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG) und wird erst mit der Rückzahlung
// des Entgelts berichtigt (§ 17 Abs. 2 Nr. 2 UStG, Abschn. 17.1 Abs. 7 UStAE).
// Ein Storno, das den Abschlag bloß aus der Verrechnung nähme, ließe 3272 und
// 3806 stehen und die Schlussrechnung die volle Steuer ein zweites Mal
// ausweisen — § 14c Abs. 1 UStG, dazu eine Forderung über einen längst
// gezahlten Betrag.
func TestSettledAdvanceIsRefundedBeforeItCanBeCancelled(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	advanceInvoice, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advanceInvoice.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}

	// Das Storno wird abgewiesen, solange das Geld beim Leistenden liegt.
	_, err = svc.CancelWithDocument(ctx, advanceInvoice.ID, "Auftrag geändert")
	if err == nil || !strings.Contains(err.Error(), "Rückzahlung") {
		t.Fatalf("das Storno einer vereinnahmten Anzahlung muss auf die Rückzahlung verweisen, erhalten: %v", err)
	}
	// Und die Nummer ist nicht verbraucht: es ist kein Stornodokument entstanden.
	if next, _ := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026); next != 2 {
		t.Errorf("der Nummernkreis steht auf %d, erwartet 2 — das abgewiesene Storno hat eine Nummer verbraucht", next)
	}

	if _, err := svc.RefundAdvance(ctx, RefundAdvanceRequest{
		AdvanceID: advanceInvoice.ID, RefundDate: "2026-05-02",
		PaymentAccount: domain.AccountBank, Reason: "Auftrag storniert",
	}); err != nil {
		t.Fatalf("Anzahlung zurückzahlen: %v", err)
	}
	if _, err := svc.CancelWithDocument(ctx, advanceInvoice.ID, "Auftrag geändert"); err != nil {
		t.Fatalf("Abschlag nach der Rückzahlung stornieren: %v", err)
	}

	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau", QuantityMilli: 1000, UnitPrice: 1000000, TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}
	if final.PrepaidAmount != 0 {
		t.Errorf("die Schlussrechnung setzt %s ab, erwartet nichts — die Anzahlung ist zurückgezahlt", final.PrepaidAmount)
	}

	// Die Probe auf das Ganze: nach Rückzahlung, Storno und Schlussrechnung
	// steht das Anzahlungskonto auf null, die Umsatzsteuer wird genau einmal auf
	// die Leistung von 10.000 € ausgewiesen, und die Forderung entspricht der
	// Schlussrechnung.
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != 0 {
		t.Errorf("Konto 3272 trägt %s, erwartet null — die Anzahlung ist aufgelöst", got)
	}
	if got := env.accountBalance(t, domain.AccountUmsatzsteuer19); got != -190000 {
		t.Errorf("Umsatzsteuer im Haben = %s, erwartet 1900,00 auf 10.000 netto", -got)
	}
	if got := env.accountBalance(t, domain.AccountBank); got != 0 {
		t.Errorf("Bank = %s, erwartet null — Zahlung und Rückzahlung heben sich auf", got)
	}
	if got := env.accountBalance(t, customer.LedgerAccount); got != 1190000 {
		t.Errorf("Forderung = %s, erwartet 11900,00 — nur die Schlussrechnung", got)
	}
}

// accountBalance summiert Soll minus Haben eines Kontos über alle Buchungen des
// Jahres. Sie ist die Probe, die eine einzelne Buchungsprüfung nicht leistet:
// ob am Ende eines Vorgangs die Konten stimmen.
func (e *testEnv) accountBalance(t *testing.T, account string) domain.Cents {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Journal lesen: %v", err)
	}
	var balance domain.Cents
	for i := range entries {
		for _, l := range entries[i].Lines {
			if l.Account != account {
				continue
			}
			if l.Side == domain.SideDebit {
				balance += l.Amount
			} else {
				balance -= l.Amount
			}
		}
	}
	return balance
}

// Offene Abschläge überleben die Schlussrechnung nicht: sie werden nicht
// abgesetzt und stünden als zweite Forderung neben ihr.
func TestFinalInvoiceRefusesOpenAdvances(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	open, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}

	final := FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau", QuantityMilli: 1000, UnitPrice: 1000000, TaxRate: domain.TaxRateStandard,
		}},
	}
	_, err = svc.IssueFinalInvoice(ctx, final)
	if err == nil || !strings.Contains(err.Error(), open.InvoiceNumber) {
		t.Fatalf("die Schlussrechnung muss den offenen Abschlag benennen, erhalten: %v", err)
	}

	// Nach der Vereinnahmung geht sie durch — und danach nimmt der Verbund
	// keine Zahlung auf einen Abschlag mehr an.
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: open.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}
	if _, err := svc.IssueFinalInvoice(ctx, final); err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}

	second, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-12-01", Net: 10000,
	})
	if err == nil {
		t.Fatalf("nach der Schlussrechnung darf kein Abschlag mehr entstehen, erhalten %s", second.InvoiceNumber)
	}
}

// Nach der Schlussrechnung gehört eine Zahlung auf sie und nicht mehr auf einen
// Abschlag: eine dann gebuchte Anzahlung stünde auf 3272 und würde nie
// aufgelöst.
func TestSettleAdvanceRefusesAfterTheFinalInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	first, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: first.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}
	second, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-05-01", Net: 200000,
	})
	if err != nil {
		t.Fatalf("zweite Abschlagsrechnung: %v", err)
	}
	// Der zweite Abschlag wird storniert, damit die Schlussrechnung entstehen
	// kann; danach ist er weder zu vereinnahmen noch zu verrechnen.
	if _, err := svc.CancelWithDocument(ctx, second.ID, "doppelt gestellt"); err != nil {
		t.Fatalf("zweiten Abschlag stornieren: %v", err)
	}
	if _, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau", QuantityMilli: 1000, UnitPrice: 1000000, TaxRate: domain.TaxRateStandard,
		}},
	}); err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}

	_, err = svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: second.ID, PaymentDate: "2026-12-01", PaymentAccount: domain.AccountBank,
	})
	if err == nil {
		t.Fatal("nach der Schlussrechnung darf kein Abschlag mehr vereinnahmt werden")
	}
}

// Die Ausbuchung eines uneinbringlichen Postens: Aufwand plus Steuerkorrektur
// nach § 17 Abs. 2 Nr. 1 UStG, mit Begründung.
func TestWriteOffUncollectibleReceivable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWired(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	payments := env.payments(t)
	if _, err := payments.WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: *inv.JournalEntryID, Date: "2026-09-01",
	}); err == nil {
		t.Error("eine Ausbuchung ohne Begründung darf nicht durchgehen")
	}

	entry, err := payments.WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: *inv.JournalEntryID, Date: "2026-09-01",
		Reason: "Insolvenz des Kunden, Forderung nicht durchsetzbar",
	})
	if err != nil {
		t.Fatalf("ausbuchen: %v", err)
	}

	if got := lineAmount(t, entry, accounting.AccountForderungsverluste19, domain.SideDebit); got != 100000 {
		t.Errorf("Forderungsverlust = %s, erwartet 1000,00", got)
	}
	if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideDebit); got != 19000 {
		t.Errorf("Steuerkorrektur = %s, erwartet 190,00", got)
	}
	if got := lineAmount(t, entry, customer.LedgerAccount, domain.SideCredit); got != 119000 {
		t.Errorf("Ausgleich des Personenkontos = %s, erwartet 1190,00", got)
	}
	// Die Steuerkorrektur trägt den Steuerschlüssel: ohne ihn fiele sie aus der
	// Voranmeldung heraus, und die Minderung käme nie an.
	hasTaxKey := false
	for _, l := range entry.Lines {
		if l.Account == domain.AccountUmsatzsteuer19 && l.TaxKey == "UST19" {
			hasTaxKey = true
		}
	}
	if !hasTaxKey {
		t.Error("die Steuerkorrektur muss den Steuerschlüssel tragen")
	}

	// Und der Posten ist geschlossen.
	open, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range open {
		if o.EntryID == *inv.JournalEntryID && o.OpenAmount != 0 {
			t.Errorf("der ausgebuchte Posten ist noch mit %s offen", o.OpenAmount)
		}
	}
}

// Eine Teil-Ausbuchung mit ungeradem Betrag geht auf: die Steuerkorrektur trägt
// die Rundungsdifferenz.
//
// Aus dem gerundeten Netto gerechnet ergäben drei Cent 0,03 € Aufwand und
// 0,01 € Steuer — vier Cent im Soll gegen drei im Haben, und die Buchung ginge
// gar nicht erst durch.
func TestPartialWriteOffAbsorbsTheRoundingDifference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWired(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	payments := env.payments(t)
	entry, err := payments.WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: *inv.JournalEntryID, Amount: 3, Date: "2026-09-01",
		Reason: "Restbetrag uneinbringlich",
	})
	if err != nil {
		t.Fatalf("Teilbetrag ausbuchen: %v", err)
	}

	var debit, credit domain.Cents
	for _, l := range entry.Lines {
		if l.Side == domain.SideDebit {
			debit += l.Amount
		} else {
			credit += l.Amount
		}
	}
	if debit != credit || debit != 3 {
		t.Errorf("Soll %s, Haben %s — erwartet drei Cent auf beiden Seiten", debit, credit)
	}
	if got := lineAmount(t, entry, customer.LedgerAccount, domain.SideCredit); got != 3 {
		t.Errorf("Ausgleich des Personenkontos = %s, erwartet 0,03", got)
	}
}

// Eine Verbindlichkeit ist kein Forderungsverlust. Ihr Wegfall ist ein Erlass
// oder eine Verjährung und wird als Ertrag gebucht — auf 6936 wäre er falsch.
func TestWriteOffRefusesPayables(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	entry := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	_, err := env.payments(t).WriteOffOpenItem(ctx, WriteOffRequest{
		OpenItemEntryID: entry.ID, Date: "2026-09-01", Reason: "verjährt",
	})
	if err == nil || !strings.Contains(err.Error(), "Verbindlichkeit") {
		t.Errorf("eine Verbindlichkeit darf nicht als Forderungsverlust ausgebucht werden, erhalten: %v", err)
	}
}

// Ohne verdrahteten Rechnungsverbund bleibt der Rechnungsdienst benutzbar: der
// Verbund ist eine Zusatzfunktion, kein Fundament.
func TestGroupsAreOptional(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoices(t)

	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatalf("ohne Verbund darf die Abfrage nicht scheitern: %v", err)
	}
	if groups == nil {
		t.Error("die Liste muss leer statt nil sein — sie geht als JSON ins Frontend")
	}
	if _, err := svc.CreateInvoiceGroup(ctx, AdvanceGroupRequest{}); err == nil {
		t.Error("ohne verdrahteten Verbund lässt sich keiner anlegen")
	}
	_ = repository.NewInvoiceGroupRepository(env.db)
}
