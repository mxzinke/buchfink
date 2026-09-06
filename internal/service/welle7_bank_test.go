package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Zuordnungsvorschlag und der Prüfpfad.

// banks baut den Bankdienst mit Vorschlag und gelernten Regeln.
func (e *testEnv) banks(t *testing.T) *BankService {
	t.Helper()
	svc := NewBankService(
		repository.NewBankRepository(e.db), e.journal, repository.NewAuditRepository(e.db))
	svc.SetOpenItemSource(e.payments(t))
	svc.SetRuleRepo(repository.NewBankRuleRepository(e.db))
	return svc
}

// bankLine legt einen Bankumsatz an und liefert seinen Schlüssel.
func (e *testEnv) bankLine(
	t *testing.T, date string, amount domain.Cents, counterparty, purpose string,
) uint {
	t.Helper()
	ctx := context.Background()
	repo := repository.NewBankRepository(e.db)
	endToEnd := purpose + counterparty + date
	if _, err := repo.CreateBatch(ctx, []domain.BankTransaction{{
		FiscalYear: e.fiscalYear, AccountIBAN: "DE02120300000000202051",
		BookingDate: date, ValueDate: date, Amount: amount, Currency: "EUR",
		CounterpartyName: counterparty, RemittanceInfo: purpose,
		EndToEndID: endToEnd, MatchStatus: domain.MatchStatusUnmatched,
		LedgerAccount: domain.AccountBank,
	}}); err != nil {
		t.Fatalf("Bankumsatz anlegen: %v", err)
	}
	txs, err := repo.FindAll(ctx, e.fiscalYear)
	if err != nil {
		t.Fatalf("Bankumsätze lesen: %v", err)
	}
	for i := range txs {
		if txs[i].EndToEndID == endToEnd {
			return txs[i].ID
		}
	}
	t.Fatal("der angelegte Bankumsatz wurde nicht gefunden")
	return 0
}

// Der genaue Betrag und die Rechnungsnummer schlagen die Namensähnlichkeit: der
// beste Vorschlag ist der Posten, auf den beide zeigen.
func TestSuggestPrefersAmountAndInvoiceNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	right := env.customer(t, "Alpha Handel GmbH", "DE", "")
	wrong := env.customer(t, "Meier Bau GmbH", "DE", "")
	target := env.openReceivable(t, right, 119_000, "2026-03-01", "2026-03-15", "RE-2026-0100")
	env.openReceivable(t, wrong, 42_000, "2026-03-02", "2026-03-16", "RE-2026-0101")

	// Der Zahlungspartner heißt wie der falsche Kunde, der Betrag und die
	// Nummer gehören zum richtigen.
	txID := env.bankLine(t, "2026-03-20", 119_000, "Meier Bau GmbH", "Zahlung RE-2026-0100")

	suggestions, err := env.banks(t).Suggest(ctx, txID)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	if len(suggestions.Suggestions) < 2 {
		t.Fatalf("erwartet mindestens zwei Vorschläge, erhalten %d", len(suggestions.Suggestions))
	}
	best := suggestions.Suggestions[0]
	if len(best.Items) != 1 || best.Items[0].EntryID != target.ID {
		t.Fatalf("der beste Vorschlag zeigt nicht auf die Rechnung mit Betrag und Nummer: %+v", best)
	}
	if len(best.Reasons) < 2 {
		t.Errorf("der Vorschlag nennt seine Merkmale nicht: %v", best.Reasons)
	}
	if best.Score <= suggestions.Suggestions[1].Score {
		t.Errorf("Punktzahl %d ist nicht höher als die des nächsten Vorschlags (%d)",
			best.Score, suggestions.Suggestions[1].Score)
	}
}

// Die Sammelzahlung: ein Betrag, der die Summe mehrerer Posten desselben Kunden
// trifft, wird als solche erkannt.
func TestSuggestFindsTheCollectivePayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Sammel GmbH", "DE", "")
	first := env.openReceivable(t, customer, 119_000, "2026-03-01", "2026-03-15", "RE-2026-0200")
	second := env.openReceivable(t, customer, 238_000, "2026-03-02", "2026-03-16", "RE-2026-0201")

	txID := env.bankLine(t, "2026-03-20", 357_000, "Sammel GmbH", "Sammelueberweisung")

	suggestions, err := env.banks(t).Suggest(ctx, txID)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	var collective *BankSuggestion
	for i := range suggestions.Suggestions {
		if suggestions.Suggestions[i].Kind == SuggestionCollective {
			collective = &suggestions.Suggestions[i]
			break
		}
	}
	if collective == nil {
		t.Fatalf("keine Sammelzahlung erkannt: %+v", suggestions.Suggestions)
	}
	if len(collective.Items) != 2 {
		t.Fatalf("erwartet zwei Posten in der Sammelzahlung, erhalten %d", len(collective.Items))
	}
	if collective.Amount != 357_000 {
		t.Errorf("Summe = %s €, erwartet 3.570,00", collective.Amount)
	}
	ids := map[uint]bool{collective.Items[0].EntryID: true, collective.Items[1].EntryID: true}
	if !ids[first.ID] || !ids[second.ID] {
		t.Errorf("die Sammelzahlung nennt die falschen Posten: %+v", collective.Items)
	}
	// Und sie steht oben: keiner der einzelnen Posten trifft den Betrag.
	if suggestions.Suggestions[0].Kind != SuggestionCollective {
		t.Errorf("der beste Vorschlag ist %q, erwartet die Sammelzahlung",
			suggestions.Suggestions[0].Kind)
	}
}

// Der wiederkehrende Umsatz ohne offenen Posten: die bestätigte Zuordnung wird
// gelernt und beim nächsten Mal vorgeschlagen.
func TestLearnedRuleIsSuggestedForARecurringTransaction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.banks(t)

	march := env.bankLine(t, "2026-03-01", -120_000, "Hausverwaltung Meier GmbH", "Miete Buero 03/2026")
	if _, err := svc.BookDirect(ctx, march, "6310", "Miete März"); err != nil {
		t.Fatalf("Bankumsatz buchen: %v", err)
	}

	rules, err := svc.Rules(ctx)
	if err != nil {
		t.Fatalf("Bankregeln: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("erwartet eine gelernte Regel, erhalten %d", len(rules))
	}
	if rules[0].CounterAccount != "6310" || rules[0].MoneyIn {
		t.Errorf("die Regel steht falsch: %+v", rules[0])
	}

	// Auch das Lernen steht im Protokoll: die Regel ist in den Einstellungen
	// sichtbar, und eine sichtbare Programmgewohnheit ohne Spur ihrer Entstehung
	// lässt sich im Prüfermodus nicht erklären.
	logs, err := repository.NewAuditRepository(env.db).FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	var learned *domain.AuditLogEntry
	for i := range logs {
		if logs[i].EntityType == "BANK_RULE" {
			learned = &logs[i]
			break
		}
	}
	if learned == nil {
		t.Fatalf("das Lernen der Regel steht nicht im Protokoll: %+v", logs)
	}
	if !strings.Contains(learned.Details, "6310") ||
		!strings.Contains(learned.Details, "Geldausgang") {
		t.Errorf("der Protokolleintrag nennt weder Gegenkonto noch Geldrichtung: %q", learned.Details)
	}

	april := env.bankLine(t, "2026-04-01", -120_000, "Hausverwaltung Meier GmbH", "Miete Buero 04/2026")
	suggestions, err := svc.Suggest(ctx, april)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	if len(suggestions.Suggestions) != 1 {
		t.Fatalf("erwartet einen Vorschlag aus der gelernten Regel, erhalten %d",
			len(suggestions.Suggestions))
	}
	rule := suggestions.Suggestions[0]
	if rule.Kind != SuggestionRule || rule.CounterAccount != "6310" {
		t.Errorf("der Vorschlag stammt nicht aus der Regel: %+v", rule)
	}

	// Ein Umsatz in der Gegenrichtung trägt dieselbe Beschreibung und ist
	// trotzdem ein anderer Vorgang.
	refund := env.bankLine(t, "2026-04-02", 50_000, "Hausverwaltung Meier GmbH", "Miete Buero Rueckzahlung")
	back, err := svc.Suggest(ctx, refund)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	for _, suggestion := range back.Suggestions {
		if suggestion.Kind == SuggestionRule {
			t.Errorf("die Regel eines Zahlungsausgangs darf einem Eingang nicht vorgeschlagen werden: %+v",
				suggestion)
		}
	}

	// Und sie lässt sich wieder löschen.
	if err := svc.DeleteRule(ctx, rules[0].ID); err != nil {
		t.Fatalf("Regel löschen: %v", err)
	}
	if remaining, err := svc.Rules(ctx); err != nil || len(remaining) != 0 {
		t.Errorf("nach dem Löschen bleiben %d Regeln (%v)", len(remaining), err)
	}
}

// Ohne Merkmal kein Vorschlag: eine Liste aller offenen Posten in anderer
// Reihenfolge hilft niemandem.
func TestSuggestSaysWhenNothingFits(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	txID := env.bankLine(t, "2026-03-20", -777, "Unbekannt", "ohne Bezug")

	suggestions, err := env.banks(t).Suggest(ctx, txID)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	if len(suggestions.Suggestions) != 0 {
		t.Errorf("erwartet keinen Vorschlag, erhalten %d", len(suggestions.Suggestions))
	}
	if suggestions.Note == "" {
		t.Error("ein leerer Kasten ohne Erklärung sieht aus wie ein Fehler")
	}
}

// --- Prüfpfad -------------------------------------------------------------

// Der Prüfpfad führt vom Beleg über die Buchung und die Zahlung zum Bankumsatz.
func TestAuditTrailLeadsFromReceiptToBankTransaction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100_000, domain.TaxRateStandard)

	txID := env.bankLine(t, "2026-03-20", -119_000, "Lieferant GmbH", "Rechnung")
	if _, err := env.payments(t).Settle(ctx, PaymentRequest{
		BankTxID:       &txID,
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-20",
		Allocations: []AllocationRequest{
			{OpenItemEntryID: invoice.ID, SettledAmount: 119_000},
		},
	}); err != nil {
		t.Fatalf("Zahlung buchen: %v", err)
	}

	receipt, err := env.receiptRepo.FindByJournalEntry(ctx, invoice.ID)
	if err != nil || receipt == nil {
		t.Fatalf("Beleg zur Buchung: %v", err)
	}

	svc := NewAuditTrailService(
		env.receiptRepo, env.journalRepo, repository.NewPaymentAllocationRepository(env.db),
		repository.NewBankRepository(env.db), repository.NewAuditRepository(env.db))

	trail, err := svc.Trail(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Prüfpfad: %v", err)
	}
	stages := make([]AuditTrailStage, 0, len(trail.Steps))
	for _, step := range trail.Steps {
		stages = append(stages, step.Stage)
	}
	want := []AuditTrailStage{TrailStageReceipt, TrailStageBooking, TrailStagePayment, TrailStageBank}
	if len(stages) != len(want) {
		t.Fatalf("Stufen = %v, erwartet %v", stages, want)
	}
	for i := range want {
		if stages[i] != want[i] {
			t.Errorf("Stufe %d = %q, erwartet %q", i+1, stages[i], want[i])
		}
	}

	// Und als CSV: dieselbe Kette in einer Datei.
	path := filepath.Join(t.TempDir(), "pruefpfad.csv")
	if _, err := svc.ExportCSV(ctx, receipt.ID, path); err != nil {
		t.Fatalf("Prüfpfad als CSV: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("CSV lesen: %v", err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV zerlegen: %v", err)
	}
	if len(rows) != len(want)+1 {
		t.Errorf("CSV hat %d Zeilen, erwartet %d samt Kopfzeile", len(rows), len(want)+1)
	}
}

// Ein Beleg ohne Buchung endet nach der ersten Stufe — und sagt das.
func TestAuditTrailStopsAtAnUnbookedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.fileIncoming(t, "rechnung.pdf")

	svc := NewAuditTrailService(
		env.receiptRepo, env.journalRepo, repository.NewPaymentAllocationRepository(env.db),
		repository.NewBankRepository(env.db), nil)

	trail, err := svc.Trail(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Prüfpfad: %v", err)
	}
	if len(trail.Steps) != 1 || trail.Steps[0].Stage != TrailStageReceipt {
		t.Fatalf("erwartet nur die Belegstufe, erhalten %+v", trail.Steps)
	}
	if !strings.Contains(trail.Note, "noch nicht gebucht") {
		t.Errorf("der Hinweis sagt nicht, wo der Pfad endet: %q", trail.Note)
	}
}

// Der Leistungsnachweis steht am Beleg, geht in den Prüfpfad — und ändert den
// Beleg-Hash nicht.
func TestServiceProofIsSavedOutsideTheReceiptHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.fileIncoming(t, "rechnung.pdf")
	before := receipt.ReceiptHash

	updated, err := env.receipts.SaveServiceProof(
		ctx, receipt.ID, "geprüft gegen Bestellung 4711 vom 03.03.2026", "2026-03-20")
	if err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}
	if updated.ServiceProof == "" || updated.ServiceProofAt != "2026-03-20" {
		t.Fatalf("der Vermerk steht nicht am Beleg: %+v", updated)
	}
	if updated.ReceiptHash != before {
		t.Errorf("der Beleg-Hash hat sich geändert (%s → %s) — das bräche die Kette jeder Buchung darauf",
			before, updated.ReceiptHash)
	}

	// Ohne Datum gilt der heutige Tag; ein unlesbares Datum wird abgewiesen.
	if _, err := env.receipts.SaveServiceProof(ctx, receipt.ID, "nochmal geprüft", "20.03.2026"); err == nil {
		t.Error("ein unlesbares Datum muss abgewiesen werden")
	}

	svc := NewAuditTrailService(
		env.receiptRepo, env.journalRepo, repository.NewPaymentAllocationRepository(env.db),
		repository.NewBankRepository(env.db), nil)
	trail, err := svc.Trail(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Prüfpfad: %v", err)
	}
	if !strings.Contains(trail.ServiceProof, "Bestellung 4711") {
		t.Errorf("der Prüfpfad nennt den Leistungsnachweis nicht: %q", trail.ServiceProof)
	}
}

// Die gelernte Regel greift auch, wenn der Verwendungszweck den Monat
// ausschreibt — und als Rückfall über den Zahlungspartner allein.
func TestLearnedRuleSurvivesWrittenMonthNames(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.banks(t)

	march := env.bankLine(t, "2026-03-01", -120_000, "Hausverwaltung Meier GmbH", "Miete Büro März 2026")
	if _, err := svc.BookDirect(ctx, march, "6310", "Miete März"); err != nil {
		t.Fatalf("Bankumsatz buchen: %v", err)
	}

	april := env.bankLine(t, "2026-04-01", -120_000, "Hausverwaltung Meier GmbH", "Miete Büro April 2026")
	suggestions, err := svc.Suggest(ctx, april)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	if len(suggestions.Suggestions) != 1 {
		t.Fatalf("erwartet einen Vorschlag aus der gelernten Regel, erhalten %d (%+v)",
			len(suggestions.Suggestions), suggestions.Suggestions)
	}
	full := suggestions.Suggestions[0]
	if full.Kind != SuggestionRule || full.CounterAccount != "6310" {
		t.Errorf("der Vorschlag stammt nicht aus der Regel: %+v", full)
	}

	// Derselbe Empfänger, ein anderer Verwendungszweck: der Rückfall über den
	// Zahlungspartner trägt den Vorschlag, aber schwächer bewertet.
	other := env.bankLine(t, "2026-04-15", -35_000, "Hausverwaltung Meier GmbH", "Nebenkostenabrechnung")
	fallback, err := svc.Suggest(ctx, other)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	if len(fallback.Suggestions) != 1 {
		t.Fatalf("erwartet einen Vorschlag über den Zahlungspartner, erhalten %d (%+v)",
			len(fallback.Suggestions), fallback.Suggestions)
	}
	partner := fallback.Suggestions[0]
	if partner.Kind != SuggestionRule || partner.CounterAccount != "6310" {
		t.Errorf("der Rückfall stammt nicht aus der Regel: %+v", partner)
	}
	if partner.Score >= full.Score {
		t.Errorf("der Rückfall (%d) darf nicht so schwer wiegen wie der volle Treffer (%d)",
			partner.Score, full.Score)
	}

	// Ein anderer Zahlungspartner erbt nichts.
	stranger := env.bankLine(t, "2026-04-16", -35_000, "Fremde Verwaltung AG", "Nebenkostenabrechnung")
	none, err := svc.Suggest(ctx, stranger)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	for _, suggestion := range none.Suggestions {
		if suggestion.Kind == SuggestionRule {
			t.Errorf("die Regel eines anderen Partners darf nicht vorgeschlagen werden: %+v", suggestion)
		}
	}
}

// Der fehlende Leistungsnachweis über der Grenze steht im Prüfbericht — und
// darunter nicht.
//
// Ohne diese Regel ließe sich eine Rechnung über tausend Euro ohne jeden
// Prüfvermerk buchen und festschreiben, ohne dass der Monatsabschluss oder das
// Prüferpaket es meldet: die Aufgabenliste allein sieht niemand, der den
// Bericht liest.
func TestCheckFindsMissingServiceProofAboveTheThreshold(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Die Voreinstellung der Grenze sind 1.000 €.
	small := env.fileWithAmount(t, "klein.pdf", 50_000, "2026-03-01")
	large := env.fileWithAmount(t, "gross.pdf", 500_000, "2026-03-02")

	run := runChecks(t, env.checksOn(t, "2026-04-01"), "2026-03-31")
	found := findingsFor(run, domain.CheckRuleServiceProofMissing)
	if len(found) != 1 {
		t.Fatalf("erwartet einen Befund, erhalten %d: %+v", len(found), found)
	}
	if found[0].ObjectID != fmt.Sprintf("%d", large.ID) {
		t.Errorf("der Befund zeigt auf Beleg %s, erwartet %d", found[0].ObjectID, large.ID)
	}
	if found[0].Severity != domain.CheckWarning {
		t.Errorf("Schwere = %q, erwartet einen Hinweis: die Grenze ist eine eigene Vorgabe",
			found[0].Severity)
	}
	if small.GrossAmount >= 100_000 {
		t.Fatal("der kleine Beleg liegt nicht unter der Grenze — dann prüft der Test nichts")
	}

	// Mit Vermerk verschwindet der Befund.
	if _, err := env.receipts.SaveServiceProof(
		ctx, large.ID, "geprüft gegen Bestellung 4711 vom 02.03.2026", "2026-03-05"); err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}
	after := runChecks(t, env.checksOn(t, "2026-04-01"), "2026-03-31")
	if got := findingsFor(after, domain.CheckRuleServiceProofMissing); len(got) != 0 {
		t.Errorf("nach dem Vermerk bleibt ein Befund stehen: %+v", got)
	}
}

// fileWithAmount legt einen Eingangsbeleg mit einem bestimmten Bruttobetrag ab.
func (e *testEnv) fileWithAmount(
	t *testing.T, name string, gross domain.Cents, documentDate string,
) *domain.Receipt {
	t.Helper()
	receipt, err := e.receipts.File(context.Background(), FileReceiptRequest{
		DocumentDate: documentDate, IssuerName: "Lieferant GmbH", GrossAmount: gross,
		Direction: domain.DirectionIncoming, ReceivedAt: documentDate,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: e.writeTempFile(t, name, minimalPDF)},
		},
	})
	if err != nil {
		t.Fatalf("Beleg %s ablegen: %v", name, err)
	}
	return receipt
}

// Der Bestellbezug geht ins Änderungsprotokoll — wie der Leistungsnachweis.
//
// Beide Felder stehen außerhalb des Beleg-Hashes; was sie unveränderbar macht,
// ist allein das Protokoll. Ein Bestellbezug, der sich still ändern ließe,
// entwertete den Prüfpfad, den er tragen soll.
func TestOrderReferenceIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.fileIncoming(t, "rechnung.pdf")

	updated, err := env.receipts.SaveOrderReference(ctx, receipt.ID, "Bestellung 4711")
	if err != nil {
		t.Fatalf("Bestellbezug: %v", err)
	}
	if updated.OrderReference != "Bestellung 4711" {
		t.Fatalf("der Bestellbezug steht nicht am Beleg: %+v", updated)
	}
	if updated.ReceiptHash != receipt.ReceiptHash {
		t.Errorf("der Beleg-Hash hat sich geändert (%s → %s)", receipt.ReceiptHash, updated.ReceiptHash)
	}

	entry := latestAudit(t, env, "RECEIPT")
	if !strings.Contains(entry.Details, "Bestellbezug") {
		t.Errorf("der Protokolleintrag nennt den Vorgang nicht: %q", entry.Details)
	}
	after := fields(t, entry.After)
	if after["orderReference"] != "Bestellung 4711" {
		t.Errorf("das Protokoll hält den neuen Stand nicht fest: %v", after)
	}
	before := fields(t, entry.Before)
	if before["orderReference"] != "" {
		t.Errorf("das Protokoll hält den alten Stand nicht fest: %v", before)
	}
}

// Ab der Grenze wird ein Eingangsbeleg ohne Leistungsnachweis nicht gebucht
// (RECH-08).
//
// Der Vermerk ist Pflichtfeld und nicht Hinweis: ein Kontrollschritt, den man
// mit einem Klick übergeht, ist keiner. Geprüft wird deshalb der Buchungsweg
// selbst — die Aufgabenliste und der Prüfbericht melden denselben Beleg, aber
// erst, wenn er schon gebucht ist.
func TestBookingRefusesLargeReceiptWithoutServiceProof(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// Wie in der Anwendung (wailsbridge.initTenant): der Belegweg liest die
	// Grenze aus den Einstellungen. Voreinstellung sind 1.000 €.
	env.posting.SetSettingsSource(repository.NewSettingsRepository(env.db))
	vendor := env.vendor(t, "Grossauftrag GmbH", "DE", "")

	large := env.fileWithAmount(t, "gross.pdf", 595_000, "2026-03-02")
	req := ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: large.ID,
		BookingDate: "2026-03-10", DocumentDate: "2026-03-02",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:  []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement: SettlementOpen,
	}

	// Die Vorschau sagt es, bevor jemand bucht.
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchungsvorschau: %v", err)
	}
	var warned bool
	for _, w := range preview.Warnings {
		if w.Code == serviceProofWarningCode {
			warned = true
		}
	}
	if !warned {
		t.Errorf("die Vorschau nennt den fehlenden Leistungsnachweis nicht: %+v", preview.Warnings)
	}

	_, err = env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("ein Eingangsbeleg über der Grenze darf ohne Leistungsnachweis nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "Leistungsnachweis") || !strings.Contains(err.Error(), "1.000,00") {
		t.Errorf("die Meldung nennt weder den Vermerk noch die Grenze: %v", err)
	}
	booked, err := env.receipts.Get(ctx, large.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if booked.Status == domain.ReceiptStatusSealed || booked.JournalEntryID != nil {
		t.Errorf("der abgewiesene Beleg darf nicht gebucht sein: %+v", booked)
	}

	// Mit dem Vermerk läuft dieselbe Buchung durch.
	if _, err := env.receipts.SaveServiceProof(
		ctx, large.ID, "geprüft gegen Bestellung 4711 vom 28.02.2026", "2026-03-09"); err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg mit Leistungsnachweis buchen: %v", err)
	}
	if entry.ReceiptID == nil || *entry.ReceiptID != large.ID {
		t.Errorf("die Buchung zeigt nicht auf den Beleg: %+v", entry)
	}

	// Unter der Grenze verlangt nichts einen Vermerk: der Regelfall bleibt ein
	// Klick.
	small := env.fileWithAmount(t, "klein.pdf", 11_900, "2026-03-03")
	if _, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: small.ID,
		BookingDate: "2026-03-11", DocumentDate: "2026-03-03",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Bürobedarf", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:  []ReceiptPosition{{PostingGroup: "buerobedarf", Net: 10_000, TaxRate: domain.TaxRateStandard}},
		Settlement: SettlementOpen,
	}); err != nil {
		t.Fatalf("kleiner Beleg buchen: %v", err)
	}
}

// Eine Rückzahlung lernt ihre eigene Regel; die Mietregel bleibt stehen.
//
// Ohne die Geldrichtung im Schlüssel trüge derselbe Partner mit demselben
// Verwendungszweck dasselbe Muster: die Rückerstattung schriebe die gelernte
// Mietregel auf „Geldeingang" um, und die nächste Mietzahlung fände nur noch den
// schwächeren Rückfall über den Zahlungspartner.
func TestLearnedRuleKeepsTheMoneyDirectionApart(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.banks(t)

	march := env.bankLine(t, "2026-03-01", -120_000, "Hausverwaltung Meier GmbH", "Miete Büro März 2026")
	if _, err := svc.BookDirect(ctx, march, "6310", "Miete März"); err != nil {
		t.Fatalf("Mietzahlung buchen: %v", err)
	}
	// Dieselbe Bezeichnung, andere Richtung: die Rückerstattung der Nebenkosten.
	refund := env.bankLine(t, "2026-03-20", 30_000, "Hausverwaltung Meier GmbH", "Miete Büro März 2026")
	if _, err := svc.BookDirect(ctx, refund, "6300", "Rückzahlung"); err != nil {
		t.Fatalf("Rückzahlung buchen: %v", err)
	}

	rules, err := svc.Rules(ctx)
	if err != nil {
		t.Fatalf("gelernte Regeln: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("erwartet zwei gelernte Regeln (eine je Richtung), erhalten %d: %+v", len(rules), rules)
	}
	for _, rule := range rules {
		switch rule.MoneyIn {
		case false:
			if rule.CounterAccount != "6310" {
				t.Errorf("die Regel des Geldausgangs zeigt auf %s, erwartet 6310", rule.CounterAccount)
			}
		case true:
			if rule.CounterAccount != "6300" {
				t.Errorf("die Regel des Geldeingangs zeigt auf %s, erwartet 6300", rule.CounterAccount)
			}
		}
	}

	// Die nächste Mietzahlung bekommt wieder den vollen Treffer auf 6310.
	april := env.bankLine(t, "2026-04-01", -120_000, "Hausverwaltung Meier GmbH", "Miete Büro April 2026")
	suggestions, err := svc.Suggest(ctx, april)
	if err != nil {
		t.Fatalf("Vorschläge: %v", err)
	}
	if len(suggestions.Suggestions) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d (%+v)",
			len(suggestions.Suggestions), suggestions.Suggestions)
	}
	best := suggestions.Suggestions[0]
	if best.Kind != SuggestionRule || best.CounterAccount != "6310" {
		t.Fatalf("der Vorschlag stammt nicht aus der Mietregel: %+v", best)
	}
	if best.Score != accounting.MatchScoreNameSimilar+accounting.MatchScoreDueNear {
		t.Errorf("Punktzahl = %d, erwartet den vollen Treffer und nicht den Rückfall über den Partner",
			best.Score)
	}
}

// Der Buchungsdialog schickt den Leistungsnachweis mit der Buchung — ein Ruf
// statt zwei.
//
// Ohne dieses Feld müsste die Maske erst SaveServiceProof rufen und dann buchen;
// bricht der zweite Ruf ab, stünde ein Vermerk an einem Beleg, der nie gebucht
// wurde. Der Vermerk geht trotzdem über den Belegdienst — mit derselben Prüfung
// des Datums, demselben Protokolleintrag und unverändertem Beleg-Hash.
func TestBookingTakesTheServiceProofWithTheRequest(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.posting.SetSettingsSource(repository.NewSettingsRepository(env.db))
	vendor := env.vendor(t, "Grossauftrag GmbH", "DE", "")
	large := env.fileWithAmount(t, "gross.pdf", 595_000, "2026-03-02")
	before := large.ReceiptHash

	entry, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: large.ID,
		BookingDate: "2026-03-10", DocumentDate: "2026-03-02",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:      []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement:     SettlementOpen,
		ServiceProof:   "geprüft gegen Bestellung 4711 vom 28.02.2026",
		ServiceProofAt: "2026-03-09",
	})
	if err != nil {
		t.Fatalf("Beleg mit mitgeschicktem Leistungsnachweis buchen: %v", err)
	}
	booked, err := env.receipts.Get(ctx, large.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if booked.ServiceProof == "" || booked.ServiceProofAt != "2026-03-09" {
		t.Errorf("der Vermerk steht nicht am Beleg: %+v", booked)
	}
	if booked.ReceiptHash != before {
		t.Errorf("der Beleg-Hash hat sich geändert (%s → %s)", before, booked.ReceiptHash)
	}
	if entry.ReceiptID == nil || *entry.ReceiptID != large.ID {
		t.Errorf("die Buchung zeigt nicht auf den Beleg: %+v", entry)
	}

	// Ein unlesbares Datum weist dieselbe Prüfung ab wie beim Nachtragen — und
	// dann wird auch nicht gebucht.
	second := env.fileWithAmount(t, "gross2.pdf", 595_000, "2026-03-03")
	if _, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: second.ID,
		BookingDate: "2026-03-11", DocumentDate: "2026-03-03",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:      []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement:     SettlementOpen,
		ServiceProof:   "geprüft gegen Bestellung 4712",
		ServiceProofAt: "03.03.2026",
	}); err == nil {
		t.Error("ein unlesbares Datum des Vermerks muss die Buchung abweisen")
	}
	unbooked, err := env.receipts.Get(ctx, second.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if unbooked.JournalEntryID != nil {
		t.Errorf("der abgewiesene Beleg darf nicht gebucht sein: %+v", unbooked)
	}
}
