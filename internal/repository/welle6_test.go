package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// rawInMemoryDB öffnet eine leere Datenbank ohne Schema.
//
// InitInMemoryDB migriert bereits; die Migrationstests brauchen aber genau den
// Zustand davor — eine Datei, die noch keine Schemaversion trägt.
func rawInMemoryDB() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func welle6DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	return db
}

// --- Änderungsprotokoll ---------------------------------------------------

func TestAuditLogChainsEntriesOnWrite(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := repo.Log(ctx, domain.AuditActionUpdate, "CONTACT", "7", "geändert"); err != nil {
			t.Fatalf("Protokolleintrag konnte nicht geschrieben werden: %v", err)
		}
	}

	entries, err := repo.FindAllAscending(ctx)
	if err != nil {
		t.Fatalf("Protokoll konnte nicht gelesen werden: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("%d Einträge, erwartet 3", len(entries))
	}

	if entries[0].PreviousHash != domain.GenesisHash {
		t.Errorf("der erste Eintrag muss am Genesis-Hash hängen, hat %q", entries[0].PreviousHash)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].PreviousHash != entries[i-1].EntryHash {
			t.Errorf("Eintrag %d hängt nicht an seinem Vorgänger", entries[i].ID)
		}
	}

	result := accounting.NewAuditChain().Verify(entries)
	if !result.IsValid {
		t.Errorf("die frisch geschriebene Kette muss gültig sein: %s", result.Message)
	}
}

func TestAuditLogDetectsTamperingInTheDatabase(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	_ = repo.Log(ctx, domain.AuditActionCreate, "JOURNAL", "1", "Buchung 2026-000001 erfasst")
	_ = repo.Log(ctx, domain.AuditActionUpdate, "JOURNAL", "1", "Betrag von 100 € auf 200 € geändert")
	_ = repo.Log(ctx, domain.AuditActionExport, "JOURNAL", "", "Journal exportiert")

	// Der Eingriff, gegen den die Kette schützt: jemand ändert die
	// Protokollzeile an der Datenbank vorbei am Programm.
	if err := db.Exec("UPDATE audit_log_entries SET details = ? WHERE id = 2", "nichts passiert").Error; err != nil {
		t.Fatalf("der Testeingriff ist fehlgeschlagen: %v", err)
	}

	entries, _ := repo.FindAllAscending(ctx)
	result := accounting.NewAuditChain().Verify(entries)
	if result.IsValid {
		t.Fatal("eine an der Datenbank geänderte Protokollzeile muss auffallen")
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != 2 {
		t.Errorf("der Bruch muss an Eintrag 2 gemeldet werden, bekommen %v", result.FirstBrokenID)
	}
}

func TestAuditLogStoresUTCActorAndVersion(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	before := time.Now().UTC().Add(-time.Second)
	if err := repo.Log(ctx, domain.AuditActionUpdate, "SETTINGS", "COMPANY", "geändert"); err != nil {
		t.Fatalf("Protokolleintrag konnte nicht geschrieben werden: %v", err)
	}

	entries, _ := repo.FindAll(ctx, 1)
	if len(entries) != 1 {
		t.Fatalf("%d Einträge, erwartet 1", len(entries))
	}
	entry := entries[0]

	// UTC: eine Ortszeit ohne Zone ist in der Nacht der Zeitumstellung
	// mehrdeutig, und die Reihenfolge des Protokolls hängt an ihr.
	if _, offset := entry.Timestamp.Zone(); offset != 0 {
		t.Errorf("der Zeitstempel steht nicht in UTC (Versatz %d Sekunden)", offset)
	}
	if entry.Timestamp.Before(before) {
		t.Errorf("der Zeitstempel %s liegt vor dem Schreiben", entry.Timestamp)
	}
	if entry.Actor == "" {
		t.Error("jeder Protokolleintrag braucht eine Bearbeiterkennung (UNV-04)")
	}
	if entry.AppVersion == "" {
		t.Error("jeder Protokolleintrag braucht die Programmfassung (UNV-06)")
	}
}

func TestLogChangeStoresOnlyChangedFields(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	before := &domain.Contact{ID: 7, Name: "Meier GmbH", Street: "Hauptstraße 1", LedgerAccount: "10001"}
	after := *before
	after.Street = "Nebenstraße 5"

	if err := repo.LogChange(ctx, domain.AuditActionUpdate, "CONTACT", "7",
		"Anschrift geändert", before, &after); err != nil {
		t.Fatalf("Änderung konnte nicht protokolliert werden: %v", err)
	}

	entries, _ := repo.FindAll(ctx, 1)
	entry := entries[0]
	if !strings.Contains(entry.Before, "Hauptstraße 1") {
		t.Errorf("das Vorher trägt die alte Anschrift nicht: %q", entry.Before)
	}
	if !strings.Contains(entry.After, "Nebenstraße 5") {
		t.Errorf("das Nachher trägt die neue Anschrift nicht: %q", entry.After)
	}
	if strings.Contains(entry.After, "Meier GmbH") {
		t.Errorf("der unveränderte Name gehört nicht ins Protokoll: %q", entry.After)
	}
}

func TestAuditFilterNarrowsTheLog(t *testing.T) {
	db := welle6DB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	_ = repo.Log(ctx, domain.AuditActionCreate, "CONTACT", "7", "angelegt")
	_ = repo.Log(ctx, domain.AuditActionUpdate, "CONTACT", "8", "geändert")
	_ = repo.Log(ctx, domain.AuditActionExport, "JOURNAL", "", "exportiert")

	byType, err := repo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "CONTACT"})
	if err != nil {
		t.Fatalf("gefiltertes Lesen ist fehlgeschlagen: %v", err)
	}
	if len(byType) != 2 {
		t.Errorf("%d Einträge zur Objektart CONTACT, erwartet 2", len(byType))
	}

	byID, _ := repo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "CONTACT", EntityID: "8"})
	if len(byID) != 1 {
		t.Errorf("%d Einträge zu Kontakt 8, erwartet 1", len(byID))
	}

	byAction, _ := repo.FindFiltered(ctx, 0, domain.AuditFilter{Action: domain.AuditActionExport})
	if len(byAction) != 1 {
		t.Errorf("%d Exporte, erwartet 1", len(byAction))
	}

	today := time.Now().UTC().Format("2006-01-02")
	byDay, _ := repo.FindFiltered(ctx, 0, domain.AuditFilter{From: today, To: today})
	if len(byDay) != 3 {
		t.Errorf("%d Einträge von heute, erwartet 3", len(byDay))
	}
}

// --- Migrationsprotokoll --------------------------------------------------

func TestFirstOpenLogsAMigration(t *testing.T) {
	db, err := rawInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	ctx := context.Background()

	record, err := ApplyMigrations(ctx, db)
	if err != nil {
		t.Fatalf("die erste Migration ist fehlgeschlagen: %v", err)
	}
	if record == nil {
		t.Fatal("die erste Öffnung muss protokolliert werden")
	}
	if record.FromVersion != 0 || record.ToVersion != SchemaVersion {
		t.Errorf("Version %d → %d, erwartet 0 → %d", record.FromVersion, record.ToVersion, SchemaVersion)
	}
	if record.Result != domain.SchemaMigrationResultOK {
		t.Errorf("Ergebnis %q, erwartet %q", record.Result, domain.SchemaMigrationResultOK)
	}
	if record.AppVersion == "" || record.Tables == "" {
		t.Error("zum Protokolleintrag gehören Programmfassung und betroffene Tabellen")
	}
}

func TestSecondOpenWithoutChangeLogsNothing(t *testing.T) {
	db, err := rawInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank konnte nicht angelegt werden: %v", err)
	}
	ctx := context.Background()

	if _, err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("die erste Migration ist fehlgeschlagen: %v", err)
	}
	second, err := ApplyMigrations(ctx, db)
	if err != nil {
		t.Fatalf("die zweite Öffnung ist fehlgeschlagen: %v", err)
	}
	// Ohne diese Bedingung protokollierte jeder Start eine Migration, und das
	// Protokoll sagte nichts mehr aus.
	if second != nil {
		t.Errorf("eine Öffnung ohne Schemaänderung darf nichts protokollieren, bekommen %+v", second)
	}

	records, err := NewMigrationRepository(db).FindSchemaMigrations(ctx)
	if err != nil {
		t.Fatalf("das Protokoll konnte nicht gelesen werden: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("%d Protokolleinträge, erwartet 1", len(records))
	}
}

func TestSeedAddsMissingAccountsAndNeverDeletes(t *testing.T) {
	db := welle6DB(t)
	ctx := context.Background()

	// Ein Konto, das es im Kontenrahmen nicht gibt, und ein umbenanntes aus dem
	// Kontenrahmen: beides muss den Seed überleben. Vorher warf ein
	// `DELETE FROM accounts` bei weniger als hundert Konten alles weg.
	own := domain.Account{Number: "9999", Name: "Eigenes Konto", Type: domain.AccountTypeStatistical, Category: "Eigene"}
	renamed := domain.Account{Number: "1800", Name: "Geschäftskonto Sparkasse", Type: domain.AccountTypeAsset, Category: "Umlaufvermögen"}
	if err := db.Create(&own).Error; err != nil {
		t.Fatalf("eigenes Konto konnte nicht angelegt werden: %v", err)
	}
	if err := db.Create(&renamed).Error; err != nil {
		t.Fatalf("umbenanntes Konto konnte nicht angelegt werden: %v", err)
	}

	if err := SeedDefaultsIfEmpty(ctx, db, 2026); err != nil {
		t.Fatalf("der Seed ist fehlgeschlagen: %v", err)
	}

	var keptOwn domain.Account
	if err := db.Where("number = ?", "9999").First(&keptOwn).Error; err != nil {
		t.Fatalf("das eigene Konto wurde gelöscht: %v", err)
	}
	var keptRenamed domain.Account
	if err := db.Where("number = ?", "1800").First(&keptRenamed).Error; err != nil {
		t.Fatalf("das umbenannte Konto wurde gelöscht: %v", err)
	}
	if keptRenamed.Name != "Geschäftskonto Sparkasse" {
		t.Errorf("der geänderte Kontoname wurde überschrieben: %q", keptRenamed.Name)
	}

	var count int64
	db.Model(&domain.Account{}).Count(&count)
	if count < 100 {
		t.Errorf("die fehlenden Konten des Kontenrahmens wurden nicht ergänzt: %d Konten", count)
	}

	// Der Lauf steht im Änderungsprotokoll — sonst wäre nicht feststellbar,
	// wann der Kontenplan ergänzt wurde.
	entries, _ := NewAuditRepository(db).FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "ACCOUNT"})
	if len(entries) == 0 {
		t.Error("das Ergänzen des Kontenplans muss protokolliert werden")
	}
}

// --- Festschreibungszeitpunkt je Buchung ----------------------------------

func TestMarkCommittedStampsOnlyEntriesUpToTheCutoff(t *testing.T) {
	db := welle6DB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	march := welle6Entry(t, repo, "2026-03-15")
	june := welle6Entry(t, repo, "2026-06-15")

	stampedAt := time.Date(2026, 4, 10, 8, 30, 0, 0, time.UTC)
	count, err := repo.MarkCommitted(ctx, 2026, "2026-03-31", 42, stampedAt)
	if err != nil {
		t.Fatalf("die Festschreibung der Buchungen ist fehlgeschlagen: %v", err)
	}
	if count != 1 {
		t.Errorf("%d gestempelte Buchungen, erwartet 1", count)
	}

	committed, _ := repo.FindByID(ctx, march.ID)
	if committed.CommittedAt == nil {
		t.Fatal("die Buchung vor dem Stichtag muss einen Festschreibungszeitpunkt tragen")
	}
	if !committed.CommittedAt.UTC().Equal(stampedAt) {
		t.Errorf("Festschreibungszeitpunkt %s, erwartet %s", committed.CommittedAt.UTC(), stampedAt)
	}
	if committed.FestschreibungID == nil || *committed.FestschreibungID != 42 {
		t.Errorf("die Buchung muss auf die Festschreibung verweisen, hat %v", committed.FestschreibungID)
	}

	open, _ := repo.FindByID(ctx, june.ID)
	if open.CommittedAt != nil {
		t.Error("eine Buchung nach dem Stichtag darf nicht festgeschrieben sein")
	}
}

func TestMarkCommittedKeepsTheEarlierStamp(t *testing.T) {
	db := welle6DB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	entry := welle6Entry(t, repo, "2026-01-15")

	first := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	second := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	if _, err := repo.MarkCommitted(ctx, 2026, "2026-01-31", 1, first); err != nil {
		t.Fatalf("die erste Festschreibung ist fehlgeschlagen: %v", err)
	}
	count, err := repo.MarkCommitted(ctx, 2026, "2026-06-30", 2, second)
	if err != nil {
		t.Fatalf("die zweite Festschreibung ist fehlgeschlagen: %v", err)
	}
	// Die zweite Festschreibung reicht weiter, darf aber den Zeitpunkt der
	// ersten nicht überschreiben — sonst wäre der Abstand Erfassung →
	// Festschreibung nicht mehr auswertbar.
	if count != 0 {
		t.Errorf("%d neu gestempelte Buchungen, erwartet 0", count)
	}
	stored, _ := repo.FindByID(ctx, entry.ID)
	if !stored.CommittedAt.UTC().Equal(first) {
		t.Errorf("Festschreibungszeitpunkt %s, erwartet den der ersten Festschreibung %s",
			stored.CommittedAt.UTC(), first)
	}
	if *stored.FestschreibungID != 1 {
		t.Errorf("die Buchung verweist auf Festschreibung %d, erwartet 1", *stored.FestschreibungID)
	}
}

func TestMarkCommittedLeavesTheHashChainIntact(t *testing.T) {
	db := welle6DB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	welle6Entry(t, repo, "2026-02-01")
	welle6Entry(t, repo, "2026-02-02")

	if _, err := repo.MarkCommitted(ctx, 2026, "2026-12-31", 1, time.Now().UTC()); err != nil {
		t.Fatalf("die Festschreibung ist fehlgeschlagen: %v", err)
	}

	entries, _ := repo.FindAll(ctx, 2026)
	result := accounting.NewHashChain().VerifyChain(entries)
	if !result.IsValid {
		// Der Festschreibungszeitpunkt steht außerhalb der kanonischen Form.
		// Stünde er darin, bräche jede Festschreibung die Kette des Jahres.
		t.Fatalf("die Festschreibung darf die Hash-Kette nicht berühren: %s", result.Message)
	}
}

func welle6Entry(t *testing.T, repo domain.JournalRepository, bookingDate string) *domain.JournalEntry {
	t.Helper()
	entry := &domain.JournalEntry{
		FiscalYear:      2026,
		BookingDate:     bookingDate,
		DocumentDate:    bookingDate,
		ServiceDateFrom: bookingDate,
		ServiceDateTo:   bookingDate,
		Description:     "Testbuchung " + bookingDate,
		Source:          domain.EntrySourceManual,
		Kind:            domain.EntryKindNormal,
		Currency:        "EUR",
		// Der Umrechnungskurs steht ausdrücklich da: die Spalte hat einen
		// Vorgabewert in der Datenbank, und ein nicht belegtes Feld käme mit
		// 1.000.000 zurück, während es mit 0 gehasht wurde — die Kette wäre
		// nach dem Lesen gebrochen, ohne dass jemand etwas geändert hätte.
		ExchangeRateMicros: 1_000_000,
		AppVersion:         "test",
		Actor:              "test@rechner",
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	if err := repo.Append(context.Background(), entry, accounting.NewHashChain().CalculateHash); err != nil {
		t.Fatalf("Buchung konnte nicht angehängt werden: %v", err)
	}
	return entry
}

// --- Aufbewahrung ---------------------------------------------------------

func TestRetentionHoldIsUniquePerYearAndSurvivesRelease(t *testing.T) {
	db := welle6DB(t)
	repo := NewRetentionRepository(db)
	ctx := context.Background()

	hold := &domain.RetentionHold{
		FiscalYear: 2020, Reason: domain.RetentionHoldAudit, SetBy: "pruefer@rechner",
	}
	if err := repo.CreateHold(ctx, hold); err != nil {
		t.Fatalf("die Aussetzung konnte nicht eingetragen werden: %v", err)
	}

	second := &domain.RetentionHold{FiscalYear: 2020, Reason: domain.RetentionHoldAppeal, SetBy: "x"}
	if err := repo.CreateHold(ctx, second); err == nil {
		t.Error("zwei geltende Aussetzungen für dasselbe Jahr wären nicht mehr auseinanderzuhalten")
	}

	active, err := repo.FindActiveHold(ctx, 2020)
	if err != nil || active == nil {
		t.Fatalf("die geltende Aussetzung wurde nicht gefunden: %v", err)
	}

	if err := repo.ReleaseHold(ctx, hold.ID, "pruefer@rechner", "Prüfung abgeschlossen", time.Now().UTC()); err != nil {
		t.Fatalf("die Aussetzung konnte nicht aufgehoben werden: %v", err)
	}
	if again, _ := repo.FindActiveHold(ctx, 2020); again != nil {
		t.Error("nach der Aufhebung gilt keine Aussetzung mehr")
	}

	// Aufgehoben heißt nicht gelöscht: dass eine Frist einmal ausgesetzt war,
	// gehört zur Geschichte der Daten.
	holds, _ := repo.FindHolds(ctx)
	if len(holds) != 1 || holds[0].ReleasedAt == nil {
		t.Errorf("die aufgehobene Aussetzung muss erhalten bleiben, bekommen %+v", holds)
	}
	if holds[0].ReleaseReason != "Prüfung abgeschlossen" {
		t.Errorf("der Aufhebungsgrund fehlt: %q", holds[0].ReleaseReason)
	}
}

func TestDeleteFiscalYearRemovesOnlyThatYear(t *testing.T) {
	db := welle6DB(t)
	journal := NewJournalRepository(db)
	retention := NewRetentionRepository(db)
	ctx := context.Background()

	welle6Entry(t, journal, "2026-01-10")
	old := &domain.JournalEntry{
		FiscalYear: 2015, BookingDate: "2015-01-10", DocumentDate: "2015-01-10",
		ServiceDateFrom: "2015-01-10", ServiceDateTo: "2015-01-10",
		Description: "Alte Buchung", Source: domain.EntrySourceManual,
		Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 5000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 5000},
		},
	}
	if err := journal.Append(ctx, old, accounting.NewHashChain().CalculateHash); err != nil {
		t.Fatalf("die alte Buchung konnte nicht angehängt werden: %v", err)
	}

	counts, err := retention.CountObjects(ctx, 2015)
	if err != nil {
		t.Fatalf("die Zählung ist fehlgeschlagen: %v", err)
	}
	if counts.JournalEntries != 1 || counts.JournalLines != 2 {
		t.Errorf("Zählung 2015: %d Buchungen mit %d Zeilen, erwartet 1 mit 2",
			counts.JournalEntries, counts.JournalLines)
	}

	deleted, orphans, err := retention.DeleteFiscalYear(ctx, 2015)
	if err != nil {
		t.Fatalf("das Löschen ist fehlgeschlagen: %v", err)
	}
	if deleted.JournalEntries != 1 {
		t.Errorf("%d gelöschte Buchungen, erwartet 1", deleted.JournalEntries)
	}
	if len(orphans) != 0 {
		t.Errorf("ohne Belegdateien gibt es keine verwaisten Pfade, bekommen %v", orphans)
	}

	remaining2015, _ := journal.FindAll(ctx, 2015)
	if len(remaining2015) != 0 {
		t.Errorf("das gelöschte Jahr trägt noch %d Buchungen", len(remaining2015))
	}
	remaining2026, _ := journal.FindAll(ctx, 2026)
	if len(remaining2026) != 1 {
		t.Errorf("das laufende Jahr wurde mitgelöscht: %d Buchungen übrig", len(remaining2026))
	}

	var lines int64
	db.Model(&domain.JournalLine{}).Count(&lines)
	if lines != 2 {
		t.Errorf("%d Buchungszeilen übrig, erwartet 2 (die des laufenden Jahres)", lines)
	}
}

// --- Fassungen der Verfahrensdokumentation --------------------------------

func TestProcedureDocumentationCountsPerDay(t *testing.T) {
	db := welle6DB(t)
	repo := NewProcedureDocumentationRepository(db)
	ctx := context.Background()

	for _, version := range []string{"2026-09-05-1", "2026-09-05-2", "2026-09-06-1"} {
		if err := repo.Create(ctx, &domain.ProcedureDocumentation{
			Version: version, FiscalYear: 2026, FileName: version + ".md",
		}); err != nil {
			t.Fatalf("Fassung %s konnte nicht abgelegt werden: %v", version, err)
		}
	}

	count, err := repo.CountForDay(ctx, "2026-09-05")
	if err != nil {
		t.Fatalf("die Zählung ist fehlgeschlagen: %v", err)
	}
	if count != 2 {
		t.Errorf("%d Fassungen am 5.9., erwartet 2", count)
	}

	all, _ := repo.FindAll(ctx)
	if len(all) != 3 {
		t.Errorf("%d Fassungen insgesamt, erwartet 3", len(all))
	}
}
