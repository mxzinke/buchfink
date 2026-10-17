package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Nachbesserungen der vierten Runde: was der Prüfer aus den Dateien der
// Überlassung nachrechnen können muss, und was der Fristenbericht nennen und
// verschweigen muss.

// --- Die Fälligkeit in der kanonischen Form --------------------------------

// Die Eröffnungsbuchung eines Umsteigers hat die übernommene Fälligkeit, und
// die geht in den Eigenhash ein. Steht sie nicht in journal.csv, lässt sich der
// Hash aus der Überlassung nicht nachrechnen — und die Zusage der
// Feldbeschreibung, das Verfahren stehe dort vollständig, wäre für genau diese
// Buchungen falsch.
func TestExportedOpeningBalanceLetsAnOutsiderRecomputeTheChain(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	closing := fileClosingStatement(t, env)
	if _, err := env.journal.BookOpeningBalance(ctx, openingRequest(t, env, customer.ID, closing.ID)); err != nil {
		t.Fatalf("die Eröffnungsbilanz ließ sich nicht buchen: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(ctx, env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	journal := readCSV(t, filepath.Join(dir, "journal.csv"))
	if len(journal) < 2 {
		t.Fatal("journal.csv enthält keine Daten")
	}
	meals := readCSV(t, filepath.Join(dir, "bewirtungen.csv"))
	gifts := readCSV(t, filepath.Join(dir, "geschenke.csv"))

	entries := groupJournalRows(t, journal)
	withDueDate := 0
	prev := domain.GenesisHash
	for _, entry := range entries {
		if entry.head["Faelligkeit"] != "" {
			withDueDate++
		}
		if got := entry.head["Vorgaengerhash"]; got != prev {
			t.Fatalf("Buchung %s: Vorgängerhash %q, erwartet %q",
				entry.head["Buchungsnummer"], got, prev)
		}
		if computed := recomputeEntryHash(entry, meals, gifts); computed != entry.head["Eigenhash"] {
			t.Fatalf("Buchung %s: nachgerechneter Hash %s, in der Datei steht %s",
				entry.head["Buchungsnummer"], computed, entry.head["Eigenhash"])
		}
		prev = entry.head["Eigenhash"]
	}
	if withDueDate == 0 {
		t.Fatal("keine exportierte Buchung hat eine Fälligkeit — der Test prüft dann nicht, wofür er da ist")
	}
}

// --- Die Kopfdaten des Belegs in der Überlassung ---------------------------

// belege.csv muss die Kopfdaten führen: Belegdatum, Aussteller und Betrag sind
// die Stammdaten des Belegs, und der Prüfer, der nur die Überlassung hat, sucht
// den Beleg über sie. Sie stehen außerdem im Beleg-Hash — eine Spalte
// „Prüfsumme über die Dateiliste“ ohne sie erklärte den Wert falsch.
func TestExportedReceiptsCarryTheirHeaderData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(ctx, env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	rows := readCSV(t, filepath.Join(dir, "belege.csv"))
	if len(rows) < 2 {
		t.Fatal("belege.csv enthält keine Daten")
	}
	for _, name := range []string{
		"Belegdatum", "Aussteller", "Bruttobetrag", "Steuerbetrag",
		"Waehrung", "Betreff", "Aufbewahrungsklasse", "Aufbewahrung_bis",
	} {
		columnIndex(t, rows[0], name)
	}

	receipts, err := env.receiptRepo.FindAll(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	byNumber := map[string]*domain.Receipt{}
	for i := range receipts {
		byNumber[receipts[i].ReceiptNumber] = &receipts[i]
	}

	number := columnIndex(t, rows[0], "Belegnummer")
	date := columnIndex(t, rows[0], "Belegdatum")
	class := columnIndex(t, rows[0], "Aufbewahrungsklasse")
	until := columnIndex(t, rows[0], "Aufbewahrung_bis")
	gross := columnIndex(t, rows[0], "Bruttobetrag")
	checked := 0
	for _, row := range rows[1:] {
		receipt, ok := byNumber[row[number]]
		if !ok {
			continue
		}
		if row[date] != receipt.DocumentDate {
			t.Errorf("Beleg %s: Belegdatum %q in der Datei, %q am Beleg",
				row[number], row[date], receipt.DocumentDate)
		}
		if row[class] != string(receipt.RetentionClass) {
			t.Errorf("Beleg %s: Aufbewahrungsklasse %q in der Datei, %q am Beleg",
				row[number], row[class], receipt.RetentionClass)
		}
		if row[until] != receipt.RetentionUntil {
			t.Errorf("Beleg %s: Aufbewahrung_bis %q in der Datei, %q am Beleg",
				row[number], row[until], receipt.RetentionUntil)
		}
		if receipt.GrossAmount != 0 && row[gross] == "0.00" {
			t.Errorf("Beleg %s: der Bruttobetrag fehlt in der Datei", row[number])
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("keine Zeile aus belege.csv ließ sich einem Beleg zuordnen")
	}

	// Und die Feldbeschreibung muss den Hash richtig erklären und sein
	// Verfahren nennen.
	doc, err := os.ReadFile(filepath.Join(dir, "feldbeschreibung.md"))
	if err != nil {
		t.Fatalf("die Feldbeschreibung fehlt: %v", err)
	}
	text := string(doc)
	for _, needle := range []string{
		"Den Beleg-Hash nachrechnen",
		"document_date",
		"Die Kette des Änderungsprotokolls nachrechnen",
		"entity_type",
		"due_date",
		"closing_period",
	} {
		if !strings.Contains(text, needle) {
			t.Errorf("die Feldbeschreibung erklärt %q nicht", needle)
		}
	}
	if strings.Contains(text, "Prüfsumme über die geordnete Dateiliste.") {
		t.Error("die Beschreibung der Spalte Beleg_SHA256 verschweigt die Kopfdaten")
	}

	// Die Aufbewahrungsklasse steht als Code in der Datei. Ohne ihre Bedeutung
	// im Schlüsselverzeichnis wäre „vouchers" für den Prüfer eine Zeichenkette
	// (GoBD Rz. 95).
	keys := readCSV(t, filepath.Join(dir, "schluesselverzeichnis.csv"))
	category := columnIndex(t, keys[0], "Kategorie")
	key := columnIndex(t, keys[0], "Schluessel")
	listed := map[string]bool{}
	for _, row := range keys[1:] {
		if row[category] == "Aufbewahrungsklasse" {
			listed[row[key]] = true
		}
	}
	for _, c := range domain.AllRetentionClasses() {
		if !listed[string(c)] {
			t.Errorf("das Schlüsselverzeichnis kennt die Aufbewahrungsklasse %q nicht", c)
		}
	}
}

// --- Die Kette des Änderungsprotokolls von außen ---------------------------

// Dasselbe für das Protokoll: es hat Vorgängerhash und Eigenhash, und wer nur
// aenderungsprotokoll.csv und die Feldbeschreibung hat, muss beide nachrechnen
// können. Sonst sind die zwei Spalten eine Behauptung.
func TestExportedAuditLogLetsAnOutsiderRecomputeTheChain(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(ctx, env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	rows := readCSV(t, filepath.Join(dir, "aenderungsprotokoll.csv"))
	if len(rows) < 2 {
		t.Fatal("aenderungsprotokoll.csv enthält keine Daten")
	}
	index := map[string]int{}
	for i, name := range rows[0] {
		index[name] = i
	}

	put := func(b *strings.Builder, name, value string) {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(len(value)))
		b.WriteByte(':')
		b.WriteString(value)
		b.WriteByte('\n')
	}

	prev := domain.GenesisHash
	checked := 0
	for _, row := range rows[1:] {
		if row[index["Eigenhash"]] == "" {
			continue
		}
		if got := row[index["Vorgaengerhash"]]; got != prev {
			t.Fatalf("Protokolleintrag %s: Vorgängerhash %q, erwartet %q",
				row[index["Protokoll_ID"]], got, prev)
		}
		var b strings.Builder
		put(&b, "prev", prev)
		put(&b, "timestamp", row[index["Zeitpunkt"]])
		put(&b, "action", row[index["Art"]])
		put(&b, "entity_type", row[index["Objektart"]])
		put(&b, "entity_id", row[index["Objekt_ID"]])
		put(&b, "details", row[index["Einzelheiten"]])
		put(&b, "before", row[index["Vorher"]])
		put(&b, "after", row[index["Nachher"]])
		put(&b, "actor", row[index["Bearbeiter"]])
		put(&b, "channel", row[index["Erfassungsweg"]])
		put(&b, "app_version", row[index["Programmfassung"]])
		sum := sha256.Sum256([]byte(b.String()))
		computed := hex.EncodeToString(sum[:])
		if computed != row[index["Eigenhash"]] {
			t.Fatalf("Protokolleintrag %s: nachgerechneter Hash %s, in der Datei steht %s",
				row[index["Protokoll_ID"]], computed, row[index["Eigenhash"]])
		}
		prev = row[index["Eigenhash"]]
		checked++
	}
	if checked == 0 {
		t.Fatal("kein exportierter Protokolleintrag hat einen Eigenhash")
	}
}

// --- Keine `null`-Listen in den Antworten der Welle 6 ----------------------

// Die Nachweisseite liest die Listen ohne Umweg. Ein nicht belegter Slice wird
// zu `null`, und `null.map` nimmt im Render den ganzen Baum mit — und zwar im
// Regelfall: die frische Buchhaltung ohne Bruch und ohne offenen Posten.
func TestWelle6AnswersMarshalEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	chain, err := NewAuditService(repository.NewAuditRepository(env.db)).VerifyChain(ctx)
	if err != nil {
		t.Fatalf("die Kettenprüfung ist fehlgeschlagen: %v", err)
	}
	assertNoNullLists(t, "Kette des Änderungsprotokolls", chain, "breaks")

	aging := accounting.AgeOpenItems(nil, "2026-06-30")
	assertNoNullLists(t, "Altersstruktur (ohne offene Posten)", aging, "sides")

	preview := &OpeningBalancePreview{}
	preview.EnsureLists()
	assertNoNullLists(t, "Eröffnungsbilanz (leere Vorschau)", preview,
		"entries", "messages")
}
