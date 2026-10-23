package domain

import (
	"testing"
)

// --- Migrationsprotokoll ---------------------------------------------------

func TestMigrationCountsBalance(t *testing.T) {
	// Eine Buchung ist immer ausgeglichen, also ist es auch die Summe aller.
	// Stimmen Soll und Haben nach einer Übernahme nicht überein, fehlt etwas.
	balanced := MigrationCounts{DebitTotal: 100000, CreditTotal: 100000}
	if !balanced.IsBalanced() {
		t.Error("gleiche Summen müssen als ausgeglichen gelten")
	}
	broken := MigrationCounts{DebitTotal: 100000, CreditTotal: 99900}
	if broken.IsBalanced() {
		t.Error("ungleiche Summen bedeuten eine unvollständige Übernahme")
	}
}
