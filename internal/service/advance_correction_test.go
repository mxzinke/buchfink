package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Berichtigung einer Abschlagsrechnung und die Vereinnahmung aus dem
// Bankumsatz.
//
// Beides sind Stellen, an denen der Anzahlungsfall den gewöhnlichen
// Rechnungsweg nicht mitgehen darf: die Steuer entsteht mit der Vereinnahmung
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG), und das Geld kommt über den
// Kontoauszug.

// bankCredit legt einen unzugeordneten Zahlungseingang an und gibt seinen
// Schlüssel zurück.
func (e *testEnv) bankCredit(t *testing.T, date string, amount domain.Cents, endToEnd string) uint {
	t.Helper()
	ctx := context.Background()
	repo := repository.NewBankRepository(e.db)
	if _, err := repo.CreateBatch(ctx, []domain.BankTransaction{{
		FiscalYear: 2026, AccountIBAN: "DE02120300000000202051",
		BookingDate: date, ValueDate: date,
		Amount: amount, Currency: "EUR", CounterpartyName: "Kunde GmbH",
		EndToEndID: endToEnd, MatchStatus: domain.MatchStatusUnmatched,
		LedgerAccount: domain.AccountBank,
	}}); err != nil {
		t.Fatalf("Bankumsatz anlegen: %v", err)
	}
	txs, err := repo.FindAll(ctx, 2026)
	if err != nil {
		t.Fatalf("Bankumsätze lesen: %v", err)
	}
	for i := range txs {
		if txs[i].EndToEndID == endToEnd {
			return txs[i].ID
		}
	}
	t.Fatalf("der angelegte Bankumsatz %s wurde nicht gefunden", endToEnd)
	return 0
}

func (e *testEnv) bankTx(t *testing.T, id uint) *domain.BankTransaction {
	t.Helper()
	tx, err := repository.NewBankRepository(e.db).FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("Bankumsatz lesen: %v", err)
	}
	return tx
}

// Die berichtigte Abschlagsrechnung bleibt eine Abschlagsrechnung.
//
// Liefe sie über den gewöhnlichen Weg der Berichtigung (Typcode 384), würde sie
// beim Ausstellen gebucht: SOLL Debitor an HABEN 4400/3806. Damit entstünde die
// Umsatzsteuer vor der Vereinnahmung — gegen § 13 Abs. 1 Nr. 1 Buchst. a Satz 4
// UStG —, und das Ersatzdokument stünde außerhalb des Verbunds: ohne offenen
// Posten, ohne Anrechnung auf den Gesamtbetrag und ohne Absetzung in der
// Schlussrechnung.
func TestCorrectingAnAdvanceInvoiceStaysAnAdvanceWithoutBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	original, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}

	corrected, err := svc.CorrectInvoice(ctx, original.ID, "Abschlag zu hoch angesetzt", &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-20",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Abschlag auf Umbau Ladenlokal", QuantityMilli: 1000,
			Unit: domain.UnitCodeDefault, UnitPrice: 300000, TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung berichtigen: %v", err)
	}

	if corrected.ResolvedKind() != domain.InvoiceKindAdvance {
		t.Errorf("die berichtigte Abschlagsrechnung ist %q, erwartet %q",
			corrected.ResolvedKind(), domain.InvoiceKindAdvance)
	}
	if corrected.JournalEntryID != nil {
		t.Error("die berichtigte Abschlagsrechnung darf beim Ausstellen nicht gebucht werden")
	}
	if corrected.CorrectsInvoiceNumber != original.InvoiceNumber {
		t.Errorf("der Bezug auf die berichtigte Rechnung fehlt: %q", corrected.CorrectsInvoiceNumber)
	}

	// Kein Journal, keine Forderung, keine Umsatzsteuer: bis zur Vereinnahmung
	// gibt es nichts zu buchen — auch nicht über den Umweg der Berichtigung.
	entries, err := env.journalRepo.FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("erwartet keine Buchung, erhalten %d: %+v", len(entries), entries)
	}
	if got := env.accountBalance(t, customer.LedgerAccount); got != 0 {
		t.Errorf("die Berichtigung hat eine Forderung von %s erzeugt, erwartet null", got)
	}

	// Der Ersatz steht im Verbund: als offener Posten der Quelle „Abschlag",
	// gegen den Gesamtbetrag gerechnet, und der stornierte fällt heraus.
	open, err := svc.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].DocumentNumber != corrected.InvoiceNumber {
		t.Fatalf("erwartet den berichtigten Abschlag als einzigen offenen Posten, erhalten %+v", open)
	}
	reloaded, err := repository.NewInvoiceGroupRepository(env.db).FindByID(ctx, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.ComputeProgress().BilledNet; got != 300000 {
		t.Errorf("abgerechnet = %s, erwartet 3000,00 — der stornierte Abschlag zählt nicht mehr mit", got)
	}

	// Und er lässt sich vereinnahmen: erst hier entsteht die Steuer, auf dem
	// Anzahlungskonto und nicht auf dem Erlöskonto.
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: corrected.ID, PaymentDate: "2026-04-10", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("berichtigten Abschlag vereinnahmen: %v", err)
	}
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != -300000 {
		t.Errorf("Konto 3272 im Haben = %s, erwartet 3000,00", -got)
	}
}

// Der Steuersatz der berichtigten Abschlagsrechnung muss der des Verbunds sein.
//
// Er entscheidet über das Anzahlungskonto und über die Steuer der
// Vereinnahmung; ein abweichender Satz ergäbe eine Buchung, die etwas anderes
// sagt als das Dokument.
func TestCorrectedAdvanceKeepsTheGroupTaxRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	original, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}

	_, err = svc.CorrectInvoice(ctx, original.ID, "Steuersatz falsch", &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-20",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Abschlag", QuantityMilli: 1000, Unit: domain.UnitCodeDefault,
			UnitPrice: 300000, TaxRate: domain.TaxRateReduced,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "Rechnungsverbund") {
		t.Errorf("ein abweichender Steuersatz muss abgewiesen werden, erhalten: %v", err)
	}
}

// Die Vereinnahmung aus dem Bankumsatz: der Umsatz gilt danach als zugeordnet,
// und ein zweiter Zugriff darauf wird abgewiesen.
//
// Ohne das bliebe der Zahlungseingang im Import als offen stehen — und wer ihn
// dort zuordnete, buchte dasselbe Geld ein zweites Mal.
func TestAdvanceSettlementFromBankTransactionMarksItMatched(t *testing.T) {
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
	second, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-02", Net: 100000,
	})
	if err != nil {
		t.Fatalf("zweite Abschlagsrechnung ausstellen: %v", err)
	}

	txID := env.bankCredit(t, "2026-04-15", 476000, "E2E-ANZ-1")

	// Ein Umsatz über einen anderen Betrag gehört nicht zu diesem Abschlag:
	// eine Teilzahlung würde die Steuer des ganzen Abschlags in einen Zeitraum
	// bringen, in dem nur ein Teil zugeflossen ist.
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: second.ID, BankTxID: &txID,
	}); err == nil || !strings.Contains(err.Error(), "vollen Betrag") {
		t.Errorf("ein abweichender Betrag muss abgewiesen werden, erhalten: %v", err)
	}

	advance, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{AdvanceID: first.ID, BankTxID: &txID})
	if err != nil {
		t.Fatalf("Anzahlung aus dem Bankumsatz vereinnahmen: %v", err)
	}
	// Konto und Datum kommen aus dem Auszug und nicht aus der Eingabe.
	if advance.SettledAt != "2026-04-15" {
		t.Errorf("Vereinnahmung am %q, erwartet den Buchungstag des Auszugs 2026-04-15", advance.SettledAt)
	}
	if advance.SettlementEntryID == nil {
		t.Fatal("die Vereinnahmung muss gebucht sein")
	}
	entry, err := env.journalRepo.FindByID(ctx, *advance.SettlementEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.BankTxID == nil || *entry.BankTxID != txID {
		t.Errorf("die Buchung trägt den Bankumsatz %v, erwartet %d", entry.BankTxID, txID)
	}
	if got := env.bankTx(t, txID).MatchStatus; got != domain.MatchStatusMatched {
		t.Errorf("der Bankumsatz steht auf %q, erwartet %q", got, domain.MatchStatusMatched)
	}

	// Derselbe Umsatz ein zweites Mal: abgewiesen, bevor irgendetwas gebucht
	// wird.
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: second.ID, BankTxID: &txID,
	}); err == nil || !strings.Contains(err.Error(), "bereits zugeordnet") {
		t.Errorf("ein zugeordneter Bankumsatz darf nicht ein zweites Mal vereinnahmt werden, erhalten: %v", err)
	}

	// Eine Auszahlung ist keine Vereinnahmung.
	debitID := env.bankCredit(t, "2026-04-16", -119000, "E2E-ANZ-2")
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: second.ID, BankTxID: &debitID,
	}); err == nil || !strings.Contains(err.Error(), "Auszahlung") {
		t.Errorf("eine Auszahlung darf keine Anzahlung vereinnahmen, erhalten: %v", err)
	}
}

// Der Vereinnahmungszeitpunkt wird nicht nachträglich auf die ausgestellte
// Abschlagsrechnung geschrieben.
//
// Ihr Dokument liegt als Beleg im Archiv und trägt den Wert nicht; stünde er am
// Datensatz, ergäbe ein erneutes Rendern ein anderes XML als das abgelegte, und
// nach einer Rückzahlung bliebe er auf dem Stornodokument stehen. Geführt wird
// er am Abschlag.
func TestSettleAdvanceLeavesTheIssuedDocumentUnchanged(t *testing.T) {
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

	stored, err := env.invoiceRepoOf(t).FindByID(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PaymentReceivedAt != "" {
		t.Errorf("die Rechnung trägt nachträglich den Vereinnahmungszeitpunkt %q; das Dokument im Beleg "+
			"trägt ihn nicht", stored.PaymentReceivedAt)
	}
	if advance.SettledAt != "2026-04-15" {
		t.Errorf("der Vereinnahmungszeitpunkt gehört an den Abschlag, dort steht %q", advance.SettledAt)
	}

	// Steht er beim Ausstellen fest, kommt er aus der Anforderung und steht
	// auch auf dem Dokument (§ 14 Abs. 4 Nr. 6 UStG).
	paid, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-05-02", Net: 100000, PaymentReceivedAt: "2026-04-28",
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung nach Zahlung ausstellen: %v", err)
	}
	if paid.PaymentReceivedAt != "2026-04-28" {
		t.Errorf("Vereinnahmung auf der Rechnung = %q, erwartet 2026-04-28", paid.PaymentReceivedAt)
	}
}

// Jeder offene Posten sagt, woher er kommt — und damit, wie er auszugleichen
// ist.
//
// Der Abschlag hat keine Buchung und damit keine EntryID: über
// PaymentService.Settle wäre er nicht ausgleichbar („die Buchung 0 hat keinen
// offenen Posten"). Ohne das Kennzeichen müsste die Oberfläche die Herkunft aus
// zwei getrennten Abfragen erraten.
func TestOpenItemsNameTheirSource(t *testing.T) {
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
	ordinary := env.simpleInvoice(customer.ID, "2026-03-05", 50000)
	if err := svc.Issue(ctx, ordinary); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	advances, err := svc.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(advances) != 1 {
		t.Fatalf("erwartet einen Abschlag in der OP-Liste, erhalten %d", len(advances))
	}
	if advances[0].Source != domain.OpenItemSourceAdvance {
		t.Errorf("Quelle des Abschlags = %q, erwartet %q", advances[0].Source, domain.OpenItemSourceAdvance)
	}
	if advances[0].AdvanceInvoiceID != advanceInvoice.ID {
		t.Errorf("der Abschlag verweist auf Rechnung %d, erwartet %d",
			advances[0].AdvanceInvoiceID, advanceInvoice.ID)
	}

	items, err := env.payments(t).OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("erwartet einen Posten aus dem Journal, erhalten %d", len(items))
	}
	if items[0].Source != domain.OpenItemSourceJournal {
		t.Errorf("Quelle der Forderung = %q, erwartet %q", items[0].Source, domain.OpenItemSourceJournal)
	}
	if items[0].AdvanceInvoiceID != 0 {
		t.Errorf("ein Posten aus dem Journal gehört zu keiner Abschlagsrechnung, trägt aber %d",
			items[0].AdvanceInvoiceID)
	}
}
