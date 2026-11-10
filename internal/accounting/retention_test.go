package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Aufbewahrungsfristen -------------------------------------------------

func TestRetentionForClassifiesAndDates(t *testing.T) {
	cases := []struct {
		name         string
		kind         domain.RetentionKind
		year         int
		wantClass    domain.RetentionClass
		wantYears    int
		wantEnd      string
		wantDeletion string
	}{
		{
			// Das Vierte Bürokratieentlastungsgesetz: acht Jahre für den Beleg
			// aus 2025, gerechnet ab dem Schluss des Entstehungsjahres.
			name: "Beleg 2025", kind: domain.RetentionKindReceiptInvoice, year: 2025,
			wantClass: domain.RetentionClassVouchers, wantYears: 8,
			wantEnd: "2033-12-31", wantDeletion: "2034-01-01",
		},
		{
			// Das Journal ist ein Handelsbuch: zehn Jahre.
			name: "Journal 2025", kind: domain.RetentionKindJournal, year: 2025,
			wantClass: domain.RetentionClassBooks, wantYears: 10,
			wantEnd: "2035-12-31", wantDeletion: "2036-01-01",
		},
		{
			// Die Verkürzung wirkt auf laufende Fristen: der Beleg aus 2024 war
			// am 1.1.2025 noch aufzubewahren und bekommt deshalb acht Jahre.
			name: "Beleg 2024", kind: domain.RetentionKindReceiptInvoice, year: 2024,
			wantClass: domain.RetentionClassVouchers, wantYears: 8,
			wantEnd: "2032-12-31", wantDeletion: "2033-01-01",
		},
		{
			// Wessen Zehnjahresfrist vor dem 1.1.2025 abgelaufen war, bleibt bei
			// zehn Jahren: rückwirkend verkürzen ändert an einer beendeten
			// Aufbewahrung nichts, und die Auskunft soll die damals geltende
			// Frist nennen.
			name: "Beleg 2013", kind: domain.RetentionKindReceiptInvoice, year: 2013,
			wantClass: domain.RetentionClassVouchers, wantYears: 10,
			wantEnd: "2023-12-31", wantDeletion: "2024-01-01",
		},
		{
			name: "Handelsbrief 2025", kind: domain.RetentionKindReceiptLetter, year: 2025,
			wantClass: domain.RetentionClassLetters, wantYears: 6,
			wantEnd: "2031-12-31", wantDeletion: "2032-01-01",
		},
		{
			// Der Kontoauszug ist Buchungsbeleg.
			name: "Kontoauszug 2025", kind: domain.RetentionKindReceiptStatement, year: 2025,
			wantClass: domain.RetentionClassVouchers, wantYears: 8,
			wantEnd: "2033-12-31", wantDeletion: "2034-01-01",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := RetentionFor(tc.kind, tc.year)
			if info.Class != tc.wantClass {
				t.Errorf("Klasse: %q, erwartet %q", info.Class, tc.wantClass)
			}
			if info.Years != tc.wantYears {
				t.Errorf("Frist: %d Jahre, erwartet %d", info.Years, tc.wantYears)
			}
			if info.RetentionEnd != tc.wantEnd {
				t.Errorf("Fristende: %q, erwartet %q", info.RetentionEnd, tc.wantEnd)
			}
			if info.EarliestDeletion != tc.wantDeletion {
				t.Errorf("frühestes Löschdatum: %q, erwartet %q", info.EarliestDeletion, tc.wantDeletion)
			}
			if info.LegalBasis == "" {
				t.Error("zu jeder Frist gehört ihre Rechtsgrundlage")
			}
		})
	}
}

func TestRetentionEndAndEarliestDeletionAreOneDayApart(t *testing.T) {
	info := RetentionFor(domain.RetentionKindReceiptInvoice, 2025)

	// Am letzten Tag der Frist ist noch aufzubewahren.
	if info.RetentionEnd != "2033-12-31" {
		t.Errorf("letzter Aufbewahrungstag %s, erwartet 2033-12-31", info.RetentionEnd)
	}
	if info.EarliestDeletion != "2034-01-01" {
		t.Errorf("frühestes Löschdatum %s, erwartet 2034-01-01", info.EarliestDeletion)
	}
}

func TestDeletionConceptCoversEveryClass(t *testing.T) {
	concept := DeletionConcept()
	if len(concept) == 0 {
		t.Fatal("das Löschkonzept ist leer")
	}
	seen := map[domain.RetentionClass]bool{}
	for _, row := range concept {
		if row.Category == "" || row.LegalBasis == "" || row.Years == 0 {
			t.Errorf("unvollständige Zeile des Löschkonzepts: %+v", row)
		}
		seen[row.Class] = true
	}
	for _, class := range domain.AllRetentionClasses() {
		if !seen[class] {
			t.Errorf("das Löschkonzept nennt die Klasse %q nicht", class)
		}
	}
}

func TestBlockedContactAnswerNamesTheNorms(t *testing.T) {
	answer := BlockedContactAnswer("Meier GmbH")
	for _, needle := range []string{"Meier GmbH", "§ 257 HGB", "§ 147 AO", "Art. 17 Abs. 3 Buchst. b DSGVO"} {
		if !strings.Contains(answer, needle) {
			t.Errorf("die Antwort an die betroffene Person nennt %q nicht:\n%s", needle, answer)
		}
	}
}

// Die Fristentabelle kommt aus der Ressource und rechnet die Verkürzung des
// Vierten Bürokratieentlastungsgesetzes richtig (ARC-01 K4).
func TestRetentionRulesComeFromTheResource(t *testing.T) {
	rules := LoadedRetentionRules()
	if rules.Version == "" || rules.Source == "" {
		t.Fatal("die Fristentabelle nennt weder ihre Fassung noch ihre Quelle")
	}
	// Der Gültigkeitsbeginn gehört zur Tabelle wie zu afa_rules.json: ohne ihn
	// sagt die Anzeige „Quelle: Gesetzesstand" nicht, welcher Stand gilt.
	if rules.ValidFrom != "2025-01-01" {
		t.Errorf("Gültigkeitsbeginn %q — erwartet den 1.1.2025, den Tag der Verkürzung",
			rules.ValidFrom)
	}
	if len(rules.Classes) != 3 {
		t.Fatalf("%d Klassen — erwartet Bücher, Belege und Handelsbriefe", len(rules.Classes))
	}

	// Ein Beleg aus 2013: die alte Zehnjahresfrist war am 1.1.2025 abgelaufen,
	// es bleibt bei ihr.
	old := RetentionFor(domain.RetentionKindReceiptInvoice, 2013)
	if old.Years != 10 || old.RetentionEnd != "2023-12-31" {
		t.Errorf("Beleg aus 2013: %d Jahre bis %s — erwartet 10 Jahre bis 2023-12-31",
			old.Years, old.RetentionEnd)
	}
	// Ein Beleg aus 2016: die alte Frist lief am 1.1.2025 noch, die kürzere
	// greift.
	running := RetentionFor(domain.RetentionKindReceiptInvoice, 2016)
	if running.Years != 8 {
		t.Errorf("Beleg aus 2016: %d Jahre — die Verkürzung wirkt auf laufende Fristen",
			running.Years)
	}
	books := RetentionFor(domain.RetentionKindJournal, 2026)
	if books.Years != 10 || books.Class != domain.RetentionClassBooks {
		t.Errorf("Handelsbücher: %d Jahre, Klasse %q", books.Years, books.Class)
	}
	letters := RetentionFor(domain.RetentionKindReceiptLetter, 2026)
	if letters.Years != 6 {
		t.Errorf("Handelsbrief: %d Jahre — erwartet sechs", letters.Years)
	}
	if RetentionForClass(domain.RetentionClassBooks, 2026).RetentionEnd != "2036-12-31" {
		t.Error("die ausdrücklich gewählte Klasse rechnet anders als die abgeleitete")
	}
}
