package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
)

func TestPostDerivesFiscalYearFromBookingDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := simpleEntry("6815", "1800", 10000)
	entry.BookingDate, entry.DocumentDate, entry.ServiceDateFrom, entry.ServiceDateTo =
		"2025-12-30", "2025-12-30", "2025-12-30", "2025-12-30"
	created, err := env.journal.Post(ctx, entry)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if created.FiscalYear != 2025 {
		t.Errorf("Geschäftsjahr %d, erwartet 2025 aus dem Buchungsdatum", created.FiscalYear)
	}

	wrong := simpleEntry("6815", "1800", 10000)
	wrong.FiscalYear = 2025
	_, err = env.journal.Post(ctx, wrong)
	if err == nil || !strings.Contains(err.Error(), "Geschäftsjahr 2026") {
		t.Fatalf("ein Geschäftsjahr, das nicht zum Buchungsdatum passt, muss abgewiesen werden; Fehler: %v", err)
	}
}

func TestPostRejectsInvalidDates(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	cases := []struct {
		name string
		set  func(e *domain.JournalEntry)
	}{
		{"Buchungsdatum mit Monat 13", func(e *domain.JournalEntry) { e.BookingDate = "2026-13-45" }},
		{"Belegdatum 30. Februar", func(e *domain.JournalEntry) { e.DocumentDate = "2026-02-30" }},
		{"Leistungsbeginn ohne führende Null", func(e *domain.JournalEntry) { e.ServiceDateFrom = "2026-3-1" }},
		{"Valuta mit Uhrzeit", func(e *domain.JournalEntry) { e.ValueDate = "2026-03-01T10:00:00Z" }},
		{"Fälligkeit im deutschen Format", func(e *domain.JournalEntry) { e.DueDate = "01.04.2026" }},
	}
	for _, c := range cases {
		entry := simpleEntry("6815", "1800", 10000)
		c.set(entry)
		if _, err := env.journal.Post(ctx, entry); err == nil || !strings.Contains(err.Error(), "kein gültiges Datum") {
			t.Errorf("%s: erwartet Abweisung als ungültiges Datum, Fehler: %v", c.name, err)
		}
	}
}

func TestPostSetsServiceStampsRegardlessOfInput(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	first, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatal(err)
	}

	committed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	festschreibung := uint(7)
	forged := simpleEntry("6815", "1800", 5000)
	forged.ID = 999
	forged.EntryNumber = "2026-999999"
	forged.Actor = "eve@elsewhere"
	forged.Channel = "agent:eve"
	forged.AppVersion = "0.0.1-fake"
	forged.PostingRuleVersion = "1999-01"
	forged.CommittedAt = &committed
	forged.FestschreibungID = &festschreibung
	forged.CorrectsEntryID = &first.ID
	forged.PreviousHash = strings.Repeat("a", 64)
	forged.EntryHash = strings.Repeat("b", 64)
	// Kennungen vorhandener Zeilen dürfen die Zeilen der ersten Buchung nicht
	// überschreiben.
	forged.Lines[0].ID = first.Lines[0].ID
	forged.Lines[0].EntryID = first.ID

	created, err := env.journal.Post(ctx, forged)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	stored, err := env.journalRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID == 999 || stored.EntryNumber == "2026-999999" {
		t.Errorf("Kennung und Nummer vergibt das Journal, gespeichert: %d / %s", stored.ID, stored.EntryNumber)
	}
	if stored.Actor != actor.Actor() || stored.AppVersion != buildinfo.Version {
		t.Errorf("Bearbeiter/Programmfassung aus der Eingabe übernommen: %q / %q", stored.Actor, stored.AppVersion)
	}
	if stored.Channel != string(actor.ChannelSystem) {
		t.Errorf("Erfassungsweg %q aus der Eingabe übernommen; ohne Angabe im Kontext gilt system", stored.Channel)
	}
	if stored.PostingRuleVersion != accounting.PostingRuleVersion {
		t.Errorf("Regelstand %q, erwartet %q", stored.PostingRuleVersion, accounting.PostingRuleVersion)
	}
	if stored.CommittedAt != nil || stored.FestschreibungID != nil {
		t.Error("Festschreibungsstempel dürfen nicht aus der Eingabe stammen")
	}
	if stored.CorrectsEntryID != nil {
		t.Error("der Verweis auf eine ersetzte Buchung entsteht nur über CorrectEntry")
	}

	reloaded, err := env.journalRepo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Lines) != 2 || reloaded.Lines[0].Amount != 10000 {
		t.Errorf("die Zeilen der ersten Buchung wurden verändert: %+v", reloaded.Lines)
	}
	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsValid {
		t.Errorf("Hash-Kette nach der Buchung gebrochen: %s", result.Message)
	}
}

func TestPostAcceptsReversalOnlyThroughStorno(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatal(err)
	}

	forged := simpleEntry("6815", "1800", 10000)
	forged.Kind = domain.EntryKindReversal
	forged.ReversalOfID = &original.ID
	for i := range forged.Lines {
		forged.Lines[i].Amount = -forged.Lines[i].Amount
	}
	if _, err := env.journal.Post(ctx, forged); err == nil || !strings.Contains(err.Error(), "Storno") {
		t.Errorf("eine Generalumkehr außerhalb des Stornos muss abgewiesen werden, Fehler: %v", err)
	}

	pointer := simpleEntry("6815", "1800", 10000)
	pointer.ReversalOfID = &original.ID
	if _, err := env.journal.Post(ctx, pointer); err == nil {
		t.Error("ein Verweis auf eine stornierte Buchung entsteht nur über den Storno")
	}

	reversal, err := env.journal.Reverse(ctx, original.ID, "Testfall")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}
	if reversal.Kind != domain.EntryKindReversal || reversal.ReversalOfID == nil || *reversal.ReversalOfID != original.ID {
		t.Errorf("der Storno muss eine Generalumkehr mit Verweis sein: %+v", reversal)
	}
}

func TestCorrectEntryRefusesOpeningSource(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatal(err)
	}
	replacement := simpleEntry("6815", "1800", 9000)
	replacement.Source = domain.EntrySourceOpening
	if _, err := env.journal.CorrectEntry(ctx, original.ID, "falscher Betrag", replacement); err == nil {
		t.Fatal("die Neubuchung darf die Quelle „opening\" nicht tragen")
	}
	if existing, err := env.journalRepo.FindReversalOf(ctx, original.ID); err != nil || existing != nil {
		t.Errorf("nach der Abweisung darf kein Storno stehen (%v, %v)", existing, err)
	}

	replacement.Source = domain.EntrySourceManual
	result, err := env.journal.CorrectEntry(ctx, original.ID, "falscher Betrag", replacement)
	if err != nil {
		t.Fatalf("Korrektur: %v", err)
	}
	if result.Replacement.CorrectsEntryID == nil || *result.Replacement.CorrectsEntryID != original.ID {
		t.Error("die Neubuchung muss auf die ersetzte Buchung verweisen")
	}
}

func TestPostRefusesPartnershipAccountsForCorporations(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	_, err := env.journal.Post(ctx, simpleEntry("2100", "1800", 50000))
	if err == nil {
		t.Fatal("eine GmbH hat keine Privatentnahmen")
	}
	for _, hint := range []string{"1307", "3640", "verdeckte Gewinnausschüttung"} {
		if !strings.Contains(err.Error(), hint) {
			t.Errorf("die Meldung sollte den richtigen Weg nennen (%s): %v", hint, err)
		}
	}
	if _, err := env.journal.Post(ctx, simpleEntry("1307", "1800", 50000)); err != nil {
		t.Errorf("die Forderung gegen den Gesellschafter ist der richtige Weg: %v", err)
	}
}

func TestPostTakesTheReceiptHashFromTheFiledReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	filed := env.fileIncoming(t, "rechnung.pdf")

	entry := simpleEntry("6815", "1800", 10000)
	entry.ReceiptID = &filed.ID
	entry.ReceiptHash = strings.Repeat("f", 64)
	created, err := env.journal.Post(ctx, entry)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if created.ReceiptHash != filed.ReceiptHash {
		t.Errorf("Beleg-Hash %q, erwartet den des abgelegten Belegs %q", created.ReceiptHash, filed.ReceiptHash)
	}

	orphan := simpleEntry("6815", "1800", 10000)
	orphan.ReceiptHash = strings.Repeat("f", 64)
	created, err = env.journal.Post(ctx, orphan)
	if err != nil {
		t.Fatal(err)
	}
	if created.ReceiptHash != "" {
		t.Errorf("ohne Belegverweis darf die Buchung keinen Beleg-Hash tragen: %q", created.ReceiptHash)
	}

	missing := uint(9999)
	ghost := simpleEntry("6815", "1800", 10000)
	ghost.ReceiptID = &missing
	if _, err := env.journal.Post(ctx, ghost); err == nil || !strings.Contains(err.Error(), "existiert nicht") {
		t.Errorf("ein Verweis auf einen fehlenden Beleg muss abgewiesen werden, Fehler: %v", err)
	}
}

func TestIntegrityCheckComparesTheReceiptHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	filed := env.fileIncoming(t, "rechnung.pdf")
	entry := simpleEntry("6815", "1800", 10000)
	entry.ReceiptID = &filed.ID
	booked, err := env.journal.Post(ctx, entry)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil || !result.IsValid {
		t.Fatalf("vor der Änderung muss die Prüfung bestehen: %v / %s", err, result.Message)
	}

	// Der Betrag des Belegs wird an der Anwendung vorbei geändert.
	if err := env.db.Exec("UPDATE receipts SET gross_amount = gross_amount + 100 WHERE id = ?", filed.ID).Error; err != nil {
		t.Fatal(err)
	}
	result, err = env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsValid {
		t.Fatal("ein veränderter Beleg muss die Prüfung scheitern lassen")
	}
	found := false
	for _, b := range result.Breaks {
		if b.Reason == domain.IntegrityBreakReceipt && b.EntryID == booked.ID && b.ExpectedHash == booked.ReceiptHash {
			found = true
		}
	}
	if !found {
		t.Errorf("kein Belegbruch zur Buchung %s gemeldet: %+v", booked.EntryNumber, result.Breaks)
	}
}

// Der Erfassungsweg kommt aus dem Kontext des Aufrufers und steht an Buchung,
// Generalumkehr und Protokolleintrag.
func TestPostRecordsTheChannelOfTheCaller(t *testing.T) {
	env := newTestEnv(t)
	agent, err := actor.Agent("claude-desktop")
	if err != nil {
		t.Fatal(err)
	}
	byAgent := actor.WithChannel(context.Background(), agent)
	inApp := actor.WithChannel(context.Background(), actor.ChannelUI)

	created, err := env.journal.Post(byAgent, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if created.Channel != string(agent) {
		t.Errorf("Buchung trägt %q, erwartet %q", created.Channel, agent)
	}
	reversal, err := env.journal.Reverse(inApp, created.ID, "Fehlbuchung")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}
	if reversal.Channel != string(actor.ChannelUI) {
		t.Errorf("die Generalumkehr trägt %q; sie entstand in der App", reversal.Channel)
	}

	logs, err := repositoryAuditEntries(env)
	if err != nil {
		t.Fatal(err)
	}
	channels := map[string]bool{}
	for _, entry := range logs {
		channels[entry.Channel] = true
	}
	if !channels[string(agent)] || !channels[string(actor.ChannelUI)] {
		t.Errorf("die Protokolleinträge tragen die Wege %v, erwartet Agent und App", channels)
	}

	result, err := env.journal.VerifyIntegrity(inApp)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsValid {
		t.Errorf("Kette nach Buchungen mit Erfassungsweg gebrochen: %s", result.Message)
	}
}

func repositoryAuditEntries(env *testEnv) ([]domain.AuditLogEntry, error) {
	var out []domain.AuditLogEntry
	err := env.db.Order("id").Find(&out).Error
	return out, err
}
