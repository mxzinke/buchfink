package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// § 275 Abs. 2 HGB weist die Bestandsveränderung an zwei Stellen aus: die der
// Erzeugnisse als Nr. 2 hinter den Umsatzerlösen, die der Roh-, Hilfs- und
// Betriebsstoffe und der Waren im Materialaufwand (Nr. 5). Welches Gegenkonto
// die Buchung trägt, folgt deshalb aus dem Bestandskonto.
func TestInventoryChangeAccountFollowsTheStockAccount(t *testing.T) {
	cases := map[string]string{
		"1000": domain.AccountBestandRHBWaren,  // Roh-, Hilfs- und Betriebsstoffe
		"1140": domain.AccountBestandRHBWaren,  // Waren
		"1110": domain.AccountBestandFertige,   // Fertige Erzeugnisse
		"1050": domain.AccountBestandUnfertige, // Unfertige Erzeugnisse
	}
	for account, want := range cases {
		got, err := InventoryChangeAccount(account)
		if err != nil {
			t.Fatalf("%s: %v", account, err)
		}
		if got != want {
			t.Errorf("Konto %s bucht gegen %s — erwartet %s", account, got, want)
		}
	}
}

// Geleistete Anzahlungen auf Vorräte sind eine Vorleistung und kein Bestand;
// ihre Veränderung ist keine Bestandsveränderung.
func TestInventoryAccountsStopBeforeThePrepayments(t *testing.T) {
	for _, account := range []string{"1180", "1190", "1200", "1800"} {
		if IsInventoryAccount(account) {
			t.Errorf("Konto %s gilt als Vorratskonto, ist aber keines", account)
		}
	}
	if !IsInventoryAccount("1179") {
		t.Error("Konto 1179 ist das letzte Warenkonto und muss als Vorratskonto gelten")
	}
}

// Jede Gruppe trägt eine Bezeichnung — die Oberfläche zeigt sie statt der
// Kontonummer, weil „1140" niemandem sagt, dass es um Waren geht.
func TestInventoryGroupsAreLabelled(t *testing.T) {
	for _, group := range InventoryGroups() {
		if group.Label == "" || group.ChangeAccount == "" {
			t.Errorf("Gruppe %d–%d ohne Bezeichnung oder Gegenkonto", group.From, group.To)
		}
	}
	if InventoryGroupLabel("1140") != "Waren" {
		t.Errorf("Konto 1140 gehört zur Gruppe %q — erwartet \"Waren\"", InventoryGroupLabel("1140"))
	}
}
