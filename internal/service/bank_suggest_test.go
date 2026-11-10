package service

import (
	"context"
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

	// Ein Umsatz in der Gegenrichtung hat dieselbe Beschreibung und ist
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
	// Zahlungspartner liefert den Vorschlag, aber schwächer bewertet.
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
