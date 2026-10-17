package wailsbridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Wiederherstellung an der Bridge.
//
// Der Test prüft, was dabei schiefgehen kann — vor allem, dass die
// vorgeschriebene Prüfung nach der Wiederherstellung auf den Büchern des vorher
// offenen Mandanten läuft und deren Ergebnis als Ergebnis der
// Wiederherstellung meldet.

// backupBridge legt eine Bridge mit einem Mandanten, einer Buchung und einer
// fertigen Sicherung an und liefert den Pfad der ZIP-Datei.
func backupBridge(t *testing.T) (*BuchfinkBridge, *domain.TenantConfig, string) {
	t.Helper()
	base := t.TempDir()
	b := newBridge(t)

	tenant, err := b.CreateTenant("Pfennig Ventures GmbH", filepath.Join(base, "mandant"),
		domain.CompanySettings{CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026})
	if err != nil {
		t.Fatalf("Mandant anlegen: %v", err)
	}
	if _, err := b.PostManualEntry(manualEntryRequest(t, b)); err != nil {
		t.Fatalf("Buchung anlegen: %v", err)
	}

	backupDir := filepath.Join(base, "sicherung")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatalf("Sicherungsordner anlegen: %v", err)
	}
	if _, err := b.SetBackupDir(backupDir); err != nil {
		t.Fatalf("Sicherungsordner setzen: %v", err)
	}
	run, err := b.CreateBackup()
	if err != nil || run == nil || !run.Success {
		t.Fatalf("Sicherung anlegen: %v (%+v)", err, run)
	}
	return b, tenant, run.Target
}

// auditTypesAt liest die Arten der Protokolleinträge eines Datenordners, ohne
// ihn zu öffnen. So lässt sich sagen, in welcher Datenbank ein Vermerk steht.
func auditTypesAt(t *testing.T, dataDir string) map[string]bool {
	t.Helper()
	db, err := repository.OpenReadOnlyDB(filepath.Join(dataDir, "buchfink.sqlite"))
	if err != nil {
		t.Fatalf("Datenbank in %s öffnen: %v", dataDir, err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	var entries []string
	if err := db.Table("audit_log_entries").Pluck("entity_type", &entries).Error; err != nil {
		t.Fatalf("Protokoll in %s lesen: %v", dataDir, err)
	}
	types := make(map[string]bool, len(entries))
	for _, e := range entries {
		types[e] = true
	}
	return types
}

// Der wiederhergestellte Mandant ist danach der offene: seine Bücher sind
// lesbar, und die vorgeschriebene Prüfung wie der Vermerk darüber stehen in
// seiner Datenbank — nicht in der des vorher offenen Mandanten.
func TestRestoreFromBackupOpensAndChecksTheRestoredTenant(t *testing.T) {
	b, origin, zipPath := backupBridge(t)
	target := filepath.Join(t.TempDir(), "wiederhergestellt")

	restored, err := b.RestoreFromBackup(zipPath, target)
	if err != nil {
		t.Fatalf("Wiederherstellung: %v", err)
	}
	if restored == nil {
		t.Fatal("die Wiederherstellung liefert keinen Mandanten")
	}
	if !sameDir(restored.DataDir, target) {
		t.Errorf("der wiederhergestellte Mandant liegt in %s statt in %s", restored.DataDir, target)
	}
	if restored.ID == origin.ID {
		t.Error("der wiederhergestellte Mandant hat dieselbe Listenkennung wie sein Ursprung")
	}
	if restored.Name != origin.Name {
		t.Errorf("der wiederhergestellte Mandant heißt %q statt %q", restored.Name, origin.Name)
	}
	if got := b.GetAppConfig().ActiveTenantID; got != restored.ID {
		t.Errorf("aktiv ist %q und nicht der wiederhergestellte Mandant %q", got, restored.ID)
	}
	if dataDir := b.GetAppConfig().DataDir; !sameDir(dataDir, target) {
		t.Errorf("die Dienste arbeiten auf %s statt auf dem wiederhergestellten Ordner %s", dataDir, target)
	}

	entries, err := b.GetAllJournalEntries()
	if err != nil {
		t.Fatalf("Journal des wiederhergestellten Mandanten lesen: %v", err)
	}
	if len(entries) != 1 || entries[0].Description != "Testbuchung" {
		t.Errorf("das Journal des wiederhergestellten Mandanten gibt %d Buchungen und den Text %q",
			len(entries), firstDescription(entries))
	}

	restoredTypes := auditTypesAt(t, target)
	if !restoredTypes["RESTORE"] {
		t.Error("der Vermerk über die Wiederherstellung fehlt im Protokoll des wiederhergestellten Mandanten")
	}
	if !restoredTypes["HASH_CHAIN"] {
		t.Error("die Integritätsprüfung lief nicht auf dem wiederhergestellten Mandanten")
	}

	originTypes := auditTypesAt(t, origin.DataDir)
	if originTypes["RESTORE"] {
		t.Error("der Vermerk über die Wiederherstellung steht im Protokoll des vorherigen Mandanten")
	}
	if originTypes["HASH_CHAIN"] {
		t.Error("die Prüfung nach der Wiederherstellung lief auf den Büchern des vorherigen Mandanten")
	}
}

// Vom Startbildschirm aus ist kein Mandant offen — das ist der Hauptfall der
// Wiederherstellung („anderer Rechner"). Sie muss dort ohne offene Dienste
// durchlaufen und darf nicht abstürzen.
func TestRestoreFromBackupWithoutOpenTenant(t *testing.T) {
	_, _, zipPath := backupBridge(t)

	fresh := newBridge(t)
	target := filepath.Join(t.TempDir(), "wiederhergestellt")

	restored, err := fresh.RestoreFromBackup(zipPath, target)
	if err != nil {
		t.Fatalf("Wiederherstellung ohne offenen Mandanten: %v", err)
	}
	if restored == nil {
		t.Fatal("die Wiederherstellung liefert keinen Mandanten")
	}
	if !auditTypesAt(t, target)["HASH_CHAIN"] {
		t.Error("die vorgeschriebene Integritätsprüfung nach der Wiederherstellung fehlt")
	}
}

func firstDescription(entries []domain.JournalEntry) string {
	if len(entries) == 0 {
		return ""
	}
	return entries[0].Description
}

// Die Kette, die nach der Wiederherstellung neben dem Ursprung weitergeht:
// sichern, prüfen, noch einmal wiederherstellen.
func TestBackupOfARestoredTenantStaysVerifiableAndRestorable(t *testing.T) {
	b, origin, zipPath := backupBridge(t)
	base := t.TempDir()

	restored, err := b.RestoreFromBackup(zipPath, filepath.Join(base, "wiederhergestellt"))
	if err != nil {
		t.Fatalf("Wiederherstellung neben dem Ursprung: %v", err)
	}
	if restored.ID == origin.ID {
		t.Fatal("der wiederhergestellte Mandant hat dieselbe Listenkennung wie sein Ursprung")
	}

	// Der wiederhergestellte Mandant ist jetzt der offene. Er sichert sich
	// selbst.
	second := filepath.Join(base, "sicherung-2")
	if err := os.MkdirAll(second, 0o700); err != nil {
		t.Fatalf("zweiten Sicherungsordner anlegen: %v", err)
	}
	if _, err := b.SetBackupDir(second); err != nil {
		t.Fatalf("Sicherungsordner setzen: %v", err)
	}
	run, err := b.CreateBackup()
	if err != nil || run == nil || !run.Success {
		t.Fatalf("Sicherung des wiederhergestellten Mandanten: %v (%+v)", err, run)
	}

	meta, err := service.ReadBackupMeta(run.Target)
	if err != nil {
		t.Fatalf("backup.json lesen: %v", err)
	}
	if meta.TenantID != restored.ID {
		t.Errorf("backup.json nennt den Mandanten %q statt %q", meta.TenantID, restored.ID)
	}

	check, err := b.VerifyBackup(run.Target)
	if err != nil {
		t.Fatalf("Sicherung prüfen: %v", err)
	}
	if check == nil || !check.Success {
		t.Fatalf("die Prüfung meldet eine heile Sicherung als fehlerhaft: %+v", check)
	}

	if _, err := b.RestoreFromBackup(run.Target, filepath.Join(base, "wiederhergestellt-2")); err != nil {
		t.Fatalf("zweite Wiederherstellung: %v", err)
	}
	entries, err := b.GetAllJournalEntries()
	if err != nil || len(entries) != 1 || entries[0].Description != "Testbuchung" {
		t.Errorf("das Journal des zweiten wiederhergestellten Mandanten ist nicht lesbar: %v (%d Buchungen)",
			err, len(entries))
	}
}

// Der Wiederherstellungslauf gehört auch in die Bücher des wiederhergestellten
// Mandanten: die Oberfläche zeigt die Läufe je Mandant, und der eben entstandene
// hätte sonst keinen einzigen — mit der Auskunft „noch nie gesichert"
// ausgerechnet für den Bestand, dessen Herkunft eine Sicherung ist.
func TestRestoreIsRecordedWithTheRestoredTenant(t *testing.T) {
	b, _, zipPath := backupBridge(t)

	if _, err := b.RestoreFromBackup(zipPath, filepath.Join(t.TempDir(), "wiederhergestellt")); err != nil {
		t.Fatalf("Wiederherstellung: %v", err)
	}

	runs, err := b.GetBackupRuns()
	if err != nil {
		t.Fatalf("Läufe des wiederhergestellten Mandanten lesen: %v", err)
	}
	found := false
	for _, run := range runs {
		if run.Kind == domain.BackupKindRestore {
			found = true
			if !run.Success {
				t.Errorf("der Wiederherstellungslauf steht als fehlgeschlagen: %s", run.Message)
			}
		}
	}
	if !found {
		t.Errorf("in den Läufen des wiederhergestellten Mandanten steht keine Wiederherstellung: %+v", runs)
	}
}

// Der Jahreswechsel muss auch beim Export ankommen.
//
// Ruft die Oberfläche einen Export ohne Jahr auf, nimmt der Dienst sein eigenes
// — und das war das Jahr vom Öffnen des Mandanten, solange der Wechsel ihn
// überging. Der Prüfer bekäme dann die Bücher eines anderen Jahres, als auf dem
// Schirm steht.
func TestSetFiscalYearReachesTheExportService(t *testing.T) {
	b, _, _ := backupBridge(t)

	if err := b.SetFiscalYear(2027); err != nil {
		t.Fatalf("Geschäftsjahr wechseln: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "z3")
	result, err := b.ExportZ3(0, dir)
	if err != nil {
		t.Fatalf("Export ohne Jahresangabe: %v", err)
	}
	if result.FiscalYear != 2027 {
		t.Errorf("der Export ohne Jahresangabe überlässt das Geschäftsjahr %d statt 2027", result.FiscalYear)
	}
}

// Ein Vorgang, ein Protokolleintrag.
//
// Die Wiederherstellung schrieb zwei: einen beim Anmelden des Mandanten mit der
// Art „Übernahme" und einen danach mit der Art „Wiederherstellung". Das
// Migrationsprotokoll zählte damit jede Wiederherstellung doppelt und einmal
// unter dem falschen Namen — und die Frage „woher stammt dieser Bestand" hatte
// zwei Antworten.
func TestRestoreWritesExactlyOneMigrationRecord(t *testing.T) {
	b, _, zipPath := backupBridge(t)

	if _, err := b.RestoreFromBackup(zipPath, filepath.Join(t.TempDir(), "wiederhergestellt")); err != nil {
		t.Fatalf("Wiederherstellung: %v", err)
	}

	records, err := b.GetMigrationRecords()
	if err != nil {
		t.Fatalf("Migrationsprotokoll lesen: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("%d Protokolleinträge für eine Wiederherstellung, erwartet 1: %+v", len(records), records)
	}
	if records[0].Kind != domain.MigrationKindRestore {
		t.Errorf("Art %q, erwartet %q", records[0].Kind, domain.MigrationKindRestore)
	}
}

// Und das Öffnen einer vorhandenen Datei ist eine Öffnung und keine Übernahme.
func TestLoadingAnExistingDatabaseIsRecordedAsOpened(t *testing.T) {
	b, _, _ := backupBridge(t)

	existingDir := filepath.Join(t.TempDir(), "vorhanden")
	if _, err := repository.InitTenantDB(existingDir); err != nil {
		t.Fatalf("vorhandene Datei anlegen: %v", err)
	}
	if err := b.LoadExistingDatabase(filepath.Join(existingDir, "buchfink.sqlite")); err != nil {
		t.Fatalf("vorhandene Datei öffnen: %v", err)
	}

	records, err := b.GetMigrationRecords()
	if err != nil {
		t.Fatalf("Migrationsprotokoll lesen: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("%d Protokolleinträge für eine Öffnung, erwartet 1: %+v", len(records), records)
	}
	if records[0].Kind != domain.MigrationKindOpen {
		t.Errorf("Art %q, erwartet %q", records[0].Kind, domain.MigrationKindOpen)
	}
}
