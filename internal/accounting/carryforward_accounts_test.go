package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// TestCarryForwardAccountsArePostable prüft, dass der Saldenvortrag überhaupt
// gebucht werden kann.
//
// Er läuft über die Vortragskonten der Klasse 9 und über den Gewinn-/
// Verlustvortrag der Klasse 2. Klasse 9 ist die einzige statistische Klasse, die
// bebucht wird — Klasse 8 hält der SKR04 für künftige DATEV-Verwendung frei und
// EnsurePostable weist sie ab. Verwechselte man die beiden, ließe sich kein Jahr
// abschließen, und der Fehler zeigte sich erst beim ersten Vortrag.
func TestCarryForwardAccountsArePostable(t *testing.T) {
	chart := chartForTest(t)

	accounts := []string{
		domain.AccountSaldenvortraegeSachkonten,
		domain.AccountSaldenvortraegeDebitoren,
		domain.AccountSaldenvortraegeKreditoren,
		domain.AccountGewinnvortrag,
		domain.AccountVerlustvortrag,
	}
	for _, number := range accounts {
		if err := chart.EnsurePostable(number); err != nil {
			t.Errorf("auf %s muss der Saldenvortrag buchen können: %v", number, err)
		}
	}

	// Klasse 8 bleibt gesperrt: die Abgrenzung ist der Sinn der Prüfung.
	if err := chart.EnsurePostable("8400"); err == nil {
		t.Error("Kontenklasse 8 ist im SKR04 nicht bebuchbar")
	}
}

// Der Gewinn-/Verlustvortrag muss ein Bilanzkonto des Eigenkapitals sein, sonst
// stünde das Jahresergebnis im Folgejahr nicht in der Bilanz.
func TestResultAccountsAreEquityOnTheBalanceSheet(t *testing.T) {
	chart := chartForTest(t)

	for _, number := range []string{domain.AccountGewinnvortrag, domain.AccountVerlustvortrag} {
		acc, ok := chart.Lookup(number)
		if !ok {
			t.Fatalf("Konto %s fehlt im SKR04-Katalog", number)
		}
		if acc.StatementType != "Bilanz" {
			t.Errorf("Konto %s ist %q, erwartet „Bilanz\"", number, acc.StatementType)
		}
		if acc.Type != domain.AccountTypeEquity {
			t.Errorf("Konto %s ist %q, erwartet Eigenkapital", number, acc.Type)
		}
		if acc.HGBCode != "Passiva.A.IV" {
			t.Errorf("Konto %s trägt die HGB-Position %q, erwartet „Passiva.A.IV\" (Gewinn-/Verlustvortrag)",
				number, acc.HGBCode)
		}
	}

	// Die Vortragskonten selbst gehören nicht in die Bilanz: sie sind
	// statistisch und heben sich über die drei Vortragsbuchungen auf.
	for _, number := range []string{
		domain.AccountSaldenvortraegeSachkonten,
		domain.AccountSaldenvortraegeDebitoren,
		domain.AccountSaldenvortraegeKreditoren,
	} {
		acc, ok := chart.Lookup(number)
		if !ok {
			t.Fatalf("Konto %s fehlt im SKR04-Katalog", number)
		}
		if acc.StatementType != "Statistisch" {
			t.Errorf("Konto %s ist %q, erwartet „Statistisch\"", number, acc.StatementType)
		}
	}
}
