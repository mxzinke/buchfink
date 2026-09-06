package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Integritätsprüfung.
//
// Sie ist die Zusage der Unveränderbarkeit (§ 146 Abs. 4 AO, GoBD Rz. 107).
// Eine Prüfung, die nur das aktive Jahr ansieht oder die nur meldet, dass etwas
// nicht stimmt, löst sie nicht ein: der Aufbewahrungszeitraum ist zehn Jahre,
// und wer eine Unstimmigkeit beheben soll, braucht den Datensatz und beide
// Hashwerte.

// tamper verändert eine Buchung an der Datenbank vorbei — so, wie es jemand
// täte, der die Kette brechen will. Über den Dienst geht das nicht, und genau
// das ist der Punkt.
func tamper(t *testing.T, env *testEnv, entryID uint, newDescription string) {
	t.Helper()
	if err := env.db.Exec("UPDATE journal_entries SET description = ? WHERE id = ?",
		newDescription, entryID).Error; err != nil {
		t.Fatalf("Testmanipulation: %v", err)
	}
}

// Ein Bruch muss den Datensatz, den erwarteten und den tatsächlichen Hash
// nennen. Ohne die beiden Hashes kann außerhalb von Buchfink niemand prüfen,
// welche Seite recht hat.
func TestIntegrityBreakNamesExpectedAndActualHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	first, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}
	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 20000)); err != nil {
		t.Fatalf("zweite Buchung: %v", err)
	}

	tamper(t, env, first.ID, "Nachträglich geändert")

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if result.IsValid {
		t.Fatal("eine nachträglich geänderte Buchung muss auffallen")
	}
	if len(result.Breaks) == 0 {
		t.Fatal("das Ergebnis nennt keinen Bruch")
	}

	found := false
	for _, b := range result.Breaks {
		if b.EntryID != first.ID {
			continue
		}
		found = true
		if b.Reason != domain.IntegrityBreakContent {
			t.Errorf("Grund %q, erwartet %q", b.Reason, domain.IntegrityBreakContent)
		}
		if len(b.ExpectedHash) != 64 || len(b.ActualHash) != 64 {
			t.Errorf("erwarteter (%q) oder tatsächlicher Hash (%q) fehlt", b.ExpectedHash, b.ActualHash)
		}
		if b.ExpectedHash == b.ActualHash {
			t.Error("erwarteter und tatsächlicher Hash sind gleich — dann wäre nichts gebrochen")
		}
		if b.EntryNumber != first.EntryNumber {
			t.Errorf("Buchungsnummer %q, erwartet %q", b.EntryNumber, first.EntryNumber)
		}
	}
	if !found {
		t.Errorf("die geänderte Buchung %s steht nicht in der Liste der Brüche", first.EntryNumber)
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != first.ID {
		t.Error("der erste gebrochene Datensatz ist nicht benannt")
	}
}

// Nach dem ersten Bruch läuft die Prüfung weiter. Sonst verdeckte die erste
// geänderte Buchung jede spätere, und die Reparatur begänne nach jedem Lauf von
// vorn.
func TestIntegrityReportsEveryBreakNotOnlyTheFirst(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	var ids []uint
	for i := 0; i < 4; i++ {
		entry, err := env.journal.Post(ctx, simpleEntry("6815", "1800", domain.Cents(1000*(i+1))))
		if err != nil {
			t.Fatalf("Buchung %d: %v", i, err)
		}
		ids = append(ids, entry.ID)
	}

	tamper(t, env, ids[0], "erste Änderung")
	tamper(t, env, ids[2], "zweite Änderung")

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	broken := map[uint]bool{}
	for _, b := range result.Breaks {
		broken[b.EntryID] = true
	}
	if !broken[ids[0]] || !broken[ids[2]] {
		t.Errorf("erwartet Brüche bei %d und %d, gemeldet: %v", ids[0], ids[2], broken)
	}
	if broken[ids[1]] || broken[ids[3]] {
		t.Error("unveränderte Buchungen wurden als gebrochen gemeldet — die Prüfung setzt nach einem Bruch nicht richtig auf")
	}
}

// Die Kette beginnt je Geschäftsjahr neu. Geprüft werden müssen deshalb alle
// Jahre: eine Prüfung des aktiven Jahres meldet Unversehrtheit, während im
// abgeschlossenen Vorjahr eine Zeile verändert wurde.
func TestIntegrityChecksEveryFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	old := simpleEntry("6815", "1800", 5000)
	old.BookingDate, old.DocumentDate = "2025-06-01", "2025-06-01"
	old.ServiceDateFrom, old.ServiceDateTo = "2025-06-01", "2025-06-01"
	env.journal.SetFiscalYear(2025)
	oldEntry, err := env.journal.Post(ctx, old)
	if err != nil {
		t.Fatalf("Buchung im Vorjahr: %v", err)
	}
	env.journal.SetFiscalYear(2026)
	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 7000)); err != nil {
		t.Fatalf("Buchung im laufenden Jahr: %v", err)
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if !result.IsValid {
		t.Fatalf("die unversehrte Buchführung wurde beanstandet: %s", result.Message)
	}
	if len(result.FiscalYears) != 2 {
		t.Errorf("geprüfte Geschäftsjahre: %v, erwartet 2025 und 2026", result.FiscalYears)
	}
	if result.TotalEntries != 2 {
		t.Errorf("geprüfte Buchungen: %d, erwartet 2", result.TotalEntries)
	}

	// Und jetzt das abgeschlossene Jahr manipulieren, während das aktive
	// unberührt bleibt.
	tamper(t, env, oldEntry.ID, "im Vorjahr geändert")
	result, err = env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if result.IsValid {
		t.Fatal("eine Änderung im Vorjahr muss auffallen, auch wenn das aktive Jahr sauber ist")
	}
	if len(result.Breaks) != 1 || result.Breaks[0].FiscalYear != 2025 {
		t.Errorf("der Bruch wird nicht dem Geschäftsjahr 2025 zugeordnet: %+v", result.Breaks)
	}
}

// Der Protokolleintrag der Prüfung nennt die Jahre lesbar. „GJ_[2025 2026]"
// stünde so im Änderungsprotokoll und in aenderungsprotokoll.csv und ließe sich
// dort weder lesen noch filtern.
func TestIntegrityCheckLogsReadableFiscalYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	old := simpleEntry("6815", "1800", 5000)
	old.BookingDate, old.DocumentDate = "2025-06-01", "2025-06-01"
	old.ServiceDateFrom, old.ServiceDateTo = "2025-06-01", "2025-06-01"
	env.journal.SetFiscalYear(2025)
	if _, err := env.journal.Post(ctx, old); err != nil {
		t.Fatalf("Buchung im Vorjahr: %v", err)
	}
	env.journal.SetFiscalYear(2026)
	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 7000)); err != nil {
		t.Fatalf("Buchung im laufenden Jahr: %v", err)
	}
	if _, err := env.journal.VerifyIntegrity(ctx); err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}

	logs, err := repository.NewAuditRepository(env.db).FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := ""
	for _, entry := range logs {
		if entry.EntityType == "HASH_CHAIN" {
			found = entry.EntityID
		}
	}
	if found != "GJ_2025,2026" {
		t.Errorf("die Objekt-ID der Prüfung lautet %q, erwartet GJ_2025,2026", found)
	}
}

// Eine Datenbank ohne jede Buchung und ohne gesetztes Geschäftsjahr darf keine
// Objektkennung „GJ_" ins Protokoll schreiben: eine Kennung, die nichts
// benennt, lässt sich in der Aufbewahrungsfrist nicht mehr deuten.
func TestIntegrityCheckLogsNamedObjectWithoutFiscalYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Ohne Geschäftsjahr und ohne Buchung: die Prüfung findet keine Jahre.
	empty := NewJournalService(
		repository.NewJournalRepository(env.db), repository.NewAccountRepository(env.db),
		repository.NewContactRepository(env.db), repository.NewAuditRepository(env.db),
		repository.NewSettingsRepository(env.db), 0)

	result, err := empty.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if len(result.FiscalYears) != 0 {
		t.Fatalf("die Prüfung meldet Geschäftsjahre %v — der Fall ist nicht der geprüfte", result.FiscalYears)
	}

	logs, err := repository.NewAuditRepository(env.db).FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := ""
	for _, entry := range logs {
		if entry.EntityType == "HASH_CHAIN" {
			found = entry.EntityID
		}
	}
	if found != "GJ_keine" {
		t.Errorf("die Objekt-ID der Prüfung lautet %q, erwartet GJ_keine", found)
	}
}

// --- Belegprüflauf ---------------------------------------------------------

// Die Hash-Chain sichert die Buchung, nicht die Datei. Ob der Beleg auf der
// Platte noch der gebuchte ist, sagt erst dieser Lauf.
func TestReceiptFileCheckFindsAChangedFile(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	if len(receipt.Files) != 1 {
		t.Fatalf("erwartet eine Belegdatei, erhalten %d", len(receipt.Files))
	}

	result, err := env.receipts.VerifyReceiptFiles(ctx)
	if err != nil {
		t.Fatalf("Belegprüflauf: %v", err)
	}
	if !result.IsValid || result.Checked != 1 || result.Intact != 1 {
		t.Fatalf("der unversehrte Beleg wurde beanstandet: %+v", result)
	}

	// Die Datei auf der Platte ändern — an Buchfink vorbei, wie es ein
	// Plattenfehler oder ein Zugriff von außen täte.
	abs := filepath.Join(env.dataDir, filepath.FromSlash(receipt.Files[0].StoredPath))
	if err := os.WriteFile(abs, []byte("%PDF-1.4 etwas anderes"), 0o600); err != nil {
		t.Fatalf("Testmanipulation: %v", err)
	}

	result, err = env.receipts.VerifyReceiptFiles(ctx)
	if err != nil {
		t.Fatalf("Belegprüflauf: %v", err)
	}
	if result.IsValid {
		t.Fatal("eine geänderte Belegdatei muss auffallen")
	}
	if result.Damaged != 1 || len(result.Issues) != 1 {
		t.Fatalf("erwartet genau eine Beanstandung, erhalten %+v", result)
	}
	issue := result.Issues[0]
	if issue.Reason != "damaged" || issue.ReceiptNumber != receipt.ReceiptNumber {
		t.Errorf("die Beanstandung benennt Grund oder Beleg nicht: %+v", issue)
	}
	if !strings.Contains(result.Message, "beschädigt") {
		t.Errorf("die Zusammenfassung nennt die Beschädigung nicht: %q", result.Message)
	}
}

// Eine fehlende Datei ist etwas anderes als eine veränderte, und die Meldung
// muss das unterscheiden: das eine ist ein Plattenfehler, das andere ein
// Eingriff.
func TestReceiptFileCheckFindsAMissingFile(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	abs := filepath.Join(env.dataDir, filepath.FromSlash(receipt.Files[0].StoredPath))
	if err := os.Remove(abs); err != nil {
		t.Fatalf("Testmanipulation: %v", err)
	}

	result, err := env.receipts.VerifyReceiptFiles(ctx)
	if err != nil {
		t.Fatalf("Belegprüflauf: %v", err)
	}
	if result.Missing != 1 || result.Damaged != 0 {
		t.Fatalf("erwartet eine fehlende und keine beschädigte Datei, erhalten %+v", result)
	}
	if result.Issues[0].Reason != "missing" {
		t.Errorf("Grund %q, erwartet \"missing\"", result.Issues[0].Reason)
	}
}
