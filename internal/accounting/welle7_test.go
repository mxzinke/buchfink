package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Rechnung, an der sich das Mahnwesen messen lässt: 10.000 € über 45 Tage
// bei einem Basiszinssatz von 1,27 % und neun Prozentpunkten Zuschlag sind
// 10,27 % im Jahr — 10.000 × 10,27 % × 45/365 = 126,6164… €, kaufmännisch
// 126,62 €.
func TestDefaultInterestIsCalculatedByTheDay(t *testing.T) {
	result, err := DefaultInterest(
		1_000_000, "2025-08-01", "2025-09-15", false, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.Days != 45 {
		t.Errorf("verzinste Tage = %d, erwartet 45", result.Days)
	}
	if result.Amount != 12662 {
		t.Errorf("Zinsen = %s €, erwartet 126,62", result.Amount)
	}
	if len(result.Segments) != 1 {
		t.Fatalf("erwartet einen Abschnitt, erhalten %d", len(result.Segments))
	}
	if result.Segments[0].TotalPoints != 1027 {
		t.Errorf("Zinssatz = %d Hundertstel Prozentpunkte, erwartet 1027 (1,27 %% + 9)",
			result.Segments[0].TotalPoints)
	}
}

// Gegenüber einem Verbraucher sind es fünf Prozentpunkte (§ 288 Abs. 1 BGB):
// 10.000 × 6,27 % × 45/365 = 77,3013… €.
func TestDefaultInterestUsesTheConsumerRate(t *testing.T) {
	result, err := DefaultInterest(
		1_000_000, "2025-08-01", "2025-09-15", true, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.Amount != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30", result.Amount)
	}
}

// Der Basiszinssatz wechselt zum 1. Januar und zum 1. Juli. Ein Verzug, der über
// den Wechsel läuft, wird abschnittsweise gerechnet — sonst trüge der ganze
// Zeitraum den Satz seines ersten Tages.
func TestDefaultInterestSplitsAtTheRateChange(t *testing.T) {
	result, err := DefaultInterest(
		1_000_000, "2026-06-01", "2026-08-01", false, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if len(result.Segments) != 2 {
		t.Fatalf("erwartet zwei Abschnitte über den Wechsel zum 1. Juli, erhalten %d", len(result.Segments))
	}
	first, second := result.Segments[0], result.Segments[1]
	if first.Days != 30 || second.Days != 31 {
		t.Errorf("Tage je Abschnitt = %d und %d, erwartet 30 und 31", first.Days, second.Days)
	}
	if first.TotalPoints != 1027 {
		t.Errorf("erster Abschnitt = %d, erwartet 1027 (1,27 %% + 9)", first.TotalPoints)
	}
	if second.TotalPoints != 1052 {
		t.Errorf("zweiter Abschnitt = %d, erwartet 1052 (1,52 %% + 9)", second.TotalPoints)
	}
	// 10.000 × 10,27 % × 30/365 = 84,4109… → 84,41;
	// 10.000 × 10,52 % × 31/365 = 89,3479… → 89,35.
	if first.Amount != 8441 || second.Amount != 8935 {
		t.Errorf("Beträge je Abschnitt = %s und %s €, erwartet 84,41 und 89,35",
			first.Amount, second.Amount)
	}
	if result.Amount != first.Amount+second.Amount {
		t.Errorf("die Summe %s € entspricht nicht den ausgewiesenen Abschnitten", result.Amount)
	}
}

// Der Verzug beginnt mit dem Ablauf von dreißig Tagen nach Fälligkeit; verzinst
// wird ab dem Tag danach (§ 286 Abs. 3 BGB).
func TestDefaultInterestStartIsThirtyDaysAfterTheDueDate(t *testing.T) {
	start, err := DefaultInterestStart("2026-01-31")
	if err != nil {
		t.Fatalf("Verzugsbeginn: %v", err)
	}
	if start != "2026-03-03" {
		t.Errorf("Verzugsbeginn = %s, erwartet 2026-03-03", start)
	}

	// Und vor diesem Tag läuft kein Zins: wer am dreißigsten Tag zahlt, zahlt
	// pünktlich.
	result, err := DefaultInterest(1_000_000, start, start, false, DefaultBaseRates())
	if err != nil {
		t.Fatalf("Verzugszinsen: %v", err)
	}
	if result.Amount != 0 || result.Days != 0 {
		t.Errorf("am Tag des Verzugsbeginns dürfen keine Zinsen laufen: %s € über %d Tage",
			result.Amount, result.Days)
	}
}

// Ein nachgetragener Satz gewinnt gegen den hinterlegten: er ist die
// Bekanntgabe, die jemand eingetragen hat.
func TestStoredBaseRateOverridesTheBuiltInOne(t *testing.T) {
	merged := MergeBaseRates([]BaseRatePeriod{
		{ValidFrom: "2026-07-01", BasisPoints: 200, Source: "nachgetragen"},
		{ValidFrom: "2027-01-01", BasisPoints: 250, Source: "nachgetragen"},
	})
	rate, err := BaseRateAt(merged, "2026-08-15")
	if err != nil {
		t.Fatalf("Basiszinssatz: %v", err)
	}
	if rate.BasisPoints != 200 {
		t.Errorf("Basiszinssatz = %d, erwartet den nachgetragenen 200", rate.BasisPoints)
	}
	// Der neue Stichtag kommt dazu und verdrängt den hinterlegten nicht.
	rate, err = BaseRateAt(merged, "2027-03-01")
	if err != nil {
		t.Fatalf("Basiszinssatz: %v", err)
	}
	if rate.BasisPoints != 250 {
		t.Errorf("Basiszinssatz = %d, erwartet 250", rate.BasisPoints)
	}
}

// Vor dem ersten hinterlegten Stichtag gibt es keinen Satz — und dann sagt die
// Tabelle das, statt null zu liefern.
func TestBaseRateBeforeTheFirstEntryIsAnError(t *testing.T) {
	if _, err := BaseRateAt(DefaultBaseRates(), "2001-01-01"); err == nil {
		t.Error("für einen Tag vor dem ersten Eintrag darf kein Satz zurückkommen")
	}
}

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

// Das Muster einer gelernten Regel überlebt die wechselnden Teile des
// Verwendungszwecks.
func TestBankRulePatternIgnoresTheChangingParts(t *testing.T) {
	first := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro 03/2026 Vertrag 4711", false)
	second := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro 04/2026 Vertrag 4711", false)
	if first != second {
		t.Errorf("die Muster unterscheiden sich:\n%q\n%q", first, second)
	}
	if first == "" {
		t.Error("das Muster ist leer")
	}

	other := BankRulePattern("Stadtwerke", "Abschlag Strom 03/2026", false)
	if other == first {
		t.Error("zwei verschiedene Vorgänge dürfen nicht dasselbe Muster tragen")
	}
}

// Der ausgeschriebene Monat wechselt und gehört deshalb nicht ins Muster.
//
// Der häufigste Fall überhaupt: Miete, Gehalt und Abschläge nennen den Monat im
// Klartext. Wer ihn ins Muster nähme, lernte jeden Monat eine neue Regel, die im
// Folgemonat nie wieder passt.
func TestBankRulePatternIgnoresWrittenMonthNames(t *testing.T) {
	march := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro März 2026", false)
	april := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro April 2026", false)
	if march != april {
		t.Errorf("die Muster unterscheiden sich:\n%q\n%q", march, april)
	}
	if march == "" {
		t.Fatal("das Muster ist leer")
	}
	// Abgekürzt ebenso, und der Vorgang bleibt im Muster stehen.
	if short := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro Sept 2026", false); short != march {
		t.Errorf("die abgekürzte Schreibweise ergibt ein anderes Muster:\n%q\n%q", short, march)
	}
	if !strings.Contains(march, "miete") {
		t.Errorf("das Muster nennt den Vorgang nicht: %q", march)
	}

	// Der Zahlungspartner lässt sich aus dem Muster zurücklesen — daran hängt
	// der Rückfall der Regelsuche.
	if got := BankRulePartnerOf(march); got != BankRulePartnerKey("Hausverwaltung Meier GmbH") {
		t.Errorf("Zahlungspartner aus dem Muster = %q, erwartet %q",
			got, BankRulePartnerKey("Hausverwaltung Meier GmbH"))
	}
	if BankRulePartnerOf(march) == march {
		t.Error("der Partnerteil darf nicht das ganze Muster sein")
	}

	// Die Geldrichtung gehört zum Schlüssel: die Rückerstattung desselben
	// Partners mit demselben Verwendungszweck ist ein anderer Vorgang und darf
	// die gelernte Regel des Ausgangs nicht überschreiben.
	incoming := BankRulePattern("Hausverwaltung Meier GmbH", "Miete Büro März 2026", true)
	if incoming == march {
		t.Errorf("Geldeingang und Geldausgang tragen dasselbe Muster: %q", march)
	}
	if BankRulePartnerOf(incoming) != BankRulePartnerKey("Hausverwaltung Meier GmbH") {
		t.Errorf("die Richtung verstellt den Partnerteil: %q", BankRulePartnerOf(incoming))
	}
	// Ohne Partner und ohne Verwendungszweck gibt es nichts zu lernen — die
	// Richtung allein ist kein Muster.
	if got := BankRulePattern("", "", true); got != "" {
		t.Errorf("Muster ohne kennzeichnenden Bestandteil = %q, erwartet leer", got)
	}
}
