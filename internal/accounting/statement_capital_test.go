package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Halbeinzahlung nach dem Musterprotokoll: 25.000 € gezeichnet, 12.500 € nicht
// eingefordert, 6.250 € eingefordert und noch nicht eingezahlt, 6.250 € auf der
// Bank. § 272 Abs. 1 Satz 2 HGB: der nicht eingeforderte Teil wird offen
// abgesetzt, der Rest heißt „Eingefordertes Kapital", die eingeforderte, noch
// nicht eingezahlte Einlage steht gesondert unter den Forderungen.
func TestUncalledCapitalIsDeductedOpenlyAndCalledCapitalIsAReceivable(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"2900": -2_500_000,
		"2910": 1_250_000,
		"1298": 625_000,
		"1800": 625_000,
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	capital := stmt.Line("passiva.A.I")
	if capital == nil || capital.Amount != 1_250_000 || capital.Label != "Eingefordertes Kapital" {
		t.Fatalf("Eigenkapital: %+v", capital)
	}
	if !strings.Contains(capital.Note, "25.000,00") || !strings.Contains(capital.Note, "12.500,00") {
		t.Errorf("die offene Absetzung fehlt im Vermerk: %s", capital.Note)
	}
	if got := lineAmount(t, stmt, "aktiva.B.II.einlagen"); got != 625_000 {
		t.Errorf("eingeforderte, noch nicht eingezahlte Einlagen %s € — erwartet 6.250,00 €", got)
	}
	if got := lineAmount(t, stmt, "aktiva.B.II.4"); got != 0 {
		t.Errorf("die eingeforderte Einlage steht unter den sonstigen Vermögensgegenständen: %s €", got)
	}
	if stmt.BalanceSheetTotal != 1_250_000 {
		t.Errorf("Bilanzsumme %s € — erwartet 12.500,00 €", stmt.BalanceSheetTotal)
	}

	// Ohne nicht eingeforderte Einlage bleibt es beim gezeichneten Kapital.
	plain, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"2900": -2_500_000, "1800": 2_500_000,
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if line := plain.Line("passiva.A.I"); line.Label != "Gezeichnetes Kapital" || line.Note != "" {
		t.Errorf("ohne Absetzung: %+v", line)
	}
}
