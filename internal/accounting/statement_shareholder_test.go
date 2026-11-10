package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ausleihungen, Forderungen und Verbindlichkeiten gegenüber Gesellschaftern
// stehen in eigenen Posten (§ 42 Abs. 3 GmbHG) und werden nicht verrechnet
// (§ 246 Abs. 2 Satz 1 HGB).
func shareholderBalances(t *testing.T) []domain.Account {
	return accountsWith(t, map[string]domain.Cents{
		"0960": 1_000_000, // Ausleihung an den Gesellschafter
		"1307": 500_000,   // Verrechnungskonto im Soll
		"1250": 119_000,   // Forderung aus Lieferung an einen Gesellschafter
		"3640": -300_000,  // Auslagen des Gesellschafters
		"3519": -200_000,  // beschlossene Ausschüttung
		"1800": -1_119_000,
	})
}

func TestShareholderItemsHaveOwnPositions(t *testing.T) {
	stmt, err := BuildStatement(shareholderBalances(t), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	for key, want := range map[string]domain.Cents{
		"aktiva.A.III.gesellschafter": 1_000_000,
		"aktiva.B.II.gesellschafter":  619_000,
		"passiva.C.gesellschafter":    500_000,
		"aktiva.A.III.6":              0,
		"aktiva.B.II.4":               0,
		"passiva.C.8":                 0,
	} {
		if got := lineAmount(t, stmt, key); got != want {
			t.Errorf("%s = %s €, erwartet %s €", key, got, want)
		}
	}
	if line := stmt.Line("aktiva.B.II"); line == nil || len(line.Remarks) != 0 {
		t.Errorf("in der Vollgliederung steht der eigene Posten, kein Vermerk: %+v", line)
	}
}

// Ein Habensaldo auf dem Verrechnungskonto ist eine Verbindlichkeit gegenüber
// dem Gesellschafter, ein Sollsaldo auf 3640 eine Forderung gegen ihn.
func TestShareholderBalancesSwitchSides(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"1307": -400_000,
		"3640": 250_000,
		"1800": 150_000,
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if got := lineAmount(t, stmt, "passiva.C.gesellschafter"); got != 400_000 {
		t.Errorf("1307 im Haben: %s € unter den Verbindlichkeiten gegenüber Gesellschaftern, erwartet 4.000,00", got)
	}
	if got := lineAmount(t, stmt, "aktiva.B.II.gesellschafter"); got != 250_000 {
		t.Errorf("3640 im Soll: %s € unter den Forderungen gegen Gesellschafter, erwartet 2.500,00", got)
	}
}

// Die verkürzte Bilanz zeigt die Posten nicht; die Beträge stehen dann als
// Davon-Vermerk an der Zeile, in der sie aufgehen.
func TestShareholderRemarksInShortDepths(t *testing.T) {
	short, err := BuildStatement(shareholderBalances(t), nil, domain.DepthShort)
	if err != nil {
		t.Fatalf("verkürzte Bilanz: %v", err)
	}
	for key, want := range map[string]domain.Cents{
		"aktiva.A.III": 1_000_000,
		"aktiva.B.II":  619_000,
		"passiva.C":    500_000,
	} {
		line := short.Line(key)
		if line == nil || len(line.Remarks) != 1 || line.Remarks[0].Amount != want {
			t.Errorf("%s: Vermerk %+v, erwartet %s €", key, line, want)
			continue
		}
		if line.Remarks[0].Reference != "§ 42 Abs. 3 GmbHG" {
			t.Errorf("%s: Fundstelle %q", key, line.Remarks[0].Reference)
		}
	}

	letters, err := BuildStatement(shareholderBalances(t), nil, domain.DepthLetters)
	if err != nil {
		t.Fatalf("Buchstabengliederung: %v", err)
	}
	if line := letters.Line("aktiva.B"); line == nil || len(line.Remarks) != 1 || line.Remarks[0].Amount != 619_000 {
		t.Errorf("aktiva.B: Vermerk %+v, erwartet 6.190,00 €", line)
	}
	if line := letters.Line("aktiva.A"); line == nil || len(line.Remarks) != 1 || line.Remarks[0].Amount != 1_000_000 {
		t.Errorf("aktiva.A: Vermerk %+v, erwartet 10.000,00 €", line)
	}
}

// Ohne Beträge gegenüber Gesellschaftern gibt es keinen Vermerk.
func TestNoShareholderRemarkWithoutBalance(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"1400": 100_000, "1800": -100_000,
	}), nil, domain.DepthShort)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if line := stmt.Line("aktiva.B.II"); line == nil || len(line.Remarks) != 0 {
		t.Errorf("Vermerk ohne Betrag: %+v", line)
	}
}
