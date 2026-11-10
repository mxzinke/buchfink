package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Betrag und die Rechnungsnummer sind harte Merkmale: jedes für sich schlägt
// die Namensähnlichkeit.
func TestExactAmountAndDocumentNumberOutweighTheName(t *testing.T) {
	tx := MatchTransaction{
		Amount: 119000, RemittanceInfo: "Zahlung RE-2026-0001",
		CounterpartyName: "Muster Handels GmbH", BookingDate: "2026-03-20",
	}

	exact := MatchItem{DocumentNumber: "RE-2026-0001", ContactName: "Andere Firma",
		DueDate: "2026-03-15", OpenAmount: 119000}
	nameOnly := MatchItem{DocumentNumber: "RE-2026-0099", ContactName: "Muster Handels GmbH",
		DueDate: "2025-01-01", OpenAmount: 42000}

	exactScore, reasons := ScoreMatch(tx, exact)
	nameScore, _ := ScoreMatch(tx, nameOnly)
	if exactScore <= nameScore {
		t.Errorf("Betrag und Nummer (%d) müssen die Namensähnlichkeit (%d) schlagen", exactScore, nameScore)
	}
	if len(reasons) < 2 {
		t.Errorf("erwartet mindestens zwei Gründe, erhalten %v", reasons)
	}
	if exactScore < MatchScoreExactAmount+MatchScoreDocumentNumber {
		t.Errorf("Punktzahl = %d, erwartet mindestens %d",
			exactScore, MatchScoreExactAmount+MatchScoreDocumentNumber)
	}

	// Und jedes harte Merkmal allein reicht dafür schon.
	amountOnly, _ := ScoreMatch(tx, MatchItem{
		DocumentNumber: "RE-2026-0777", ContactName: "Ganz anders", OpenAmount: 119000})
	if amountOnly <= MatchScoreNameSimilar {
		t.Errorf("der genaue Betrag allein (%d) muss die Namensähnlichkeit (%d) schlagen",
			amountOnly, MatchScoreNameSimilar)
	}
}

// Die Rechnungsnummer wird auch dann gefunden, wenn der Verwendungszweck sie
// anders schreibt — Trennzeichen gehen beim Übertragen regelmäßig verloren.
func TestDocumentNumberIsFoundDespiteSeparators(t *testing.T) {
	tx := MatchTransaction{
		Amount: 5000, RemittanceInfo: "RG RE 2026 0042 vielen Dank",
		CounterpartyName: "", BookingDate: "2026-04-01",
	}
	score, _ := ScoreMatch(tx, MatchItem{DocumentNumber: "RE-2026-0042", OpenAmount: 9999})
	if score < MatchScoreDocumentNumber {
		t.Errorf("Punktzahl = %d, die Nummer wurde nicht erkannt", score)
	}
}

// Der Rechtsformzusatz allein ist keine Ähnlichkeit: sonst passte jede GmbH zu
// jeder anderen.
func TestLegalFormAloneIsNoNameMatch(t *testing.T) {
	tx := MatchTransaction{Amount: 100, CounterpartyName: "Alpha GmbH", BookingDate: "2026-01-01"}
	score, _ := ScoreMatch(tx, MatchItem{ContactName: "Beta GmbH", OpenAmount: 999})
	if score != 0 {
		t.Errorf("Punktzahl = %d, erwartet 0 — „GmbH“ ist kein Merkmal", score)
	}
}

// Die Sammelzahlung: drei Rechnungen, ein Betrag.
func TestAmountCombinationFindsTheCollectivePayment(t *testing.T) {
	values := []domain.Cents{11900, 23800, 5000, 7000}
	picked := FindAmountCombination(values, 30800, 8)
	if len(picked) != 2 {
		t.Fatalf("erwartet zwei Posten, erhalten %v", picked)
	}
	var sum domain.Cents
	for _, i := range picked {
		sum += values[i]
	}
	if sum != 30800 {
		t.Errorf("Summe = %s €, erwartet 308,00", sum)
	}

	// Was nicht aufgeht, geht nicht auf: ein „fast passender" Vorschlag wäre
	// schlimmer als keiner.
	if picked := FindAmountCombination(values, 12345, 8); picked != nil {
		t.Errorf("erwartet keine Kombination, erhalten %v", picked)
	}
}
