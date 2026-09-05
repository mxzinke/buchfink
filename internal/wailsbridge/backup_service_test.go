package wailsbridge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/security"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/zalando/go-keyring"
)

// Die Wiederherstellung an der Bridge.
//
// Der Regelfall ist ein verschlüsselter Mandant: jeder über CreateTenant
// angelegte hat eine Schlüsseldatei, die in der Sicherung liegt. Geprüft wird
// deshalb genau das, was dabei schiefgehen kann — dass der wiederhergestellte
// Ordner unter einer neuen Kennung angemeldet wird und der Schlüssel damit
// unauffindbar ist, und dass die vorgeschriebene Prüfung nach der
// Wiederherstellung auf den Büchern des vorher offenen Mandanten läuft und
// deren Ergebnis als Ergebnis der Wiederherstellung meldet.

// backupBridge legt eine Bridge mit einem verschlüsselten Mandanten, einer
// Buchung und einer fertigen Sicherung an und liefert den Pfad der ZIP-Datei.
func backupBridge(t *testing.T) (*BuchfinkBridge, *domain.TenantConfig, string) {
	t.Helper()
	// Schlüsselbund im Speicher: der Test soll keinen echten des Betriebssystems
	// anfassen, aber denselben Weg gehen wie der Betrieb.
	keyring.MockInit()

	base := t.TempDir()
	b := &BuchfinkBridge{
		appCfgRepo:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		currentYear: 2026,
	}

	tenant, err := b.CreateTenant("Pfennig Ventures GmbH", filepath.Join(base, "mandant"),
		domain.CompanySettings{CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026})
	if err != nil {
		t.Fatalf("Mandant anlegen: %v", err)
	}
	if !security.KeyfileExists(tenant.DataDir) {
		t.Fatal("der angelegte Mandant hat keine Schlüsseldatei — der Regelfall ist nicht abgebildet")
	}
	if _, err := b.PostJournalEntry(manualEntry()); err != nil {
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

	entries, err := repository.NewAuditRepository(db).FindAll(context.Background(), 0)
	if err != nil {
		t.Fatalf("Protokoll in %s lesen: %v", dataDir, err)
	}
	types := make(map[string]bool, len(entries))
	for _, e := range entries {
		types[e.EntityType] = true
	}
	return types
}

// Der wiederhergestellte Mandant ist danach der offene: er öffnet mit dem
// Schlüssel aus der Sicherung, seine Bücher sind lesbar, und die vorgeschriebene
// Prüfung wie der Vermerk darüber stehen in seiner Datenbank — nicht in der des
// vorher offenen Mandanten.
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
	if restored.VaultID() != origin.ID {
		t.Errorf("der wiederhergestellte Mandant sucht seinen Schlüssel unter %q statt unter %q — "+
			"unter dieser Kennung liegt er im Schlüsselbund", restored.VaultID(), origin.ID)
	}
	if b.IsLocked() {
		t.Fatal("der wiederhergestellte Mandant ist gesperrt, obwohl sein Schlüssel auf diesem Rechner liegt")
	}
	if got := b.GetAppConfig().ActiveTenantID; got != restored.ID {
		t.Errorf("aktiv ist %q und nicht der wiederhergestellte Mandant %q", got, restored.ID)
	}
	if !sameDir(b.dataDir, target) {
		t.Errorf("die Dienste arbeiten auf %s statt auf dem wiederhergestellten Ordner %s", b.dataDir, target)
	}

	// Die Buchung ist lesbar: der Text ist ein verschlüsseltes Feld, und im
	// Klartext zurück heißt, dass der richtige Schlüssel geöffnet hat.
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
	_, origin, zipPath := backupBridge(t)

	fresh := &BuchfinkBridge{
		appCfgRepo:  repository.NewAppConfigRepository(filepath.Join(t.TempDir(), "config")),
		currentYear: 2026,
	}
	target := filepath.Join(t.TempDir(), "wiederhergestellt")

	restored, err := fresh.RestoreFromBackup(zipPath, target)
	if err != nil {
		t.Fatalf("Wiederherstellung ohne offenen Mandanten: %v", err)
	}
	if restored == nil || restored.VaultID() != origin.ID {
		t.Fatalf("der wiederhergestellte Mandant trägt nicht die Kennung der Sicherung: %+v", restored)
	}
	if fresh.IsLocked() {
		t.Error("der wiederhergestellte Mandant ist gesperrt, obwohl sein Schlüssel auf diesem Rechner liegt")
	}
	if !auditTypesAt(t, target)["HASH_CHAIN"] {
		t.Error("die vorgeschriebene Integritätsprüfung nach der Wiederherstellung fehlt")
	}
}

// Ohne den Schlüssel auf diesem Rechner bleibt der Mandant angemeldet und
// gesperrt. Geprüft wird dann nichts — und vor allem wird nicht die Prüfung
// eines anderen Mandanten als seine ausgegeben.
func TestRestoreFromBackupWithoutKeyStaysLockedAndSkipsChecks(t *testing.T) {
	b, origin, zipPath := backupBridge(t)

	// Der Fall „anderer Rechner": die Daten sind da, das Geheimnis nicht.
	if err := security.DeleteTenantSecret(origin.ID); err != nil {
		t.Fatalf("Schlüsselbund-Geheimnis entfernen: %v", err)
	}
	target := filepath.Join(t.TempDir(), "wiederhergestellt")

	restored, err := b.RestoreFromBackup(zipPath, target)
	if err == nil {
		t.Fatal("ohne Schlüssel darf die Wiederherstellung nicht als geprüft gemeldet werden")
	}
	if !strings.Contains(err.Error(), "Wiederherstellungsdatei") {
		t.Errorf("die Meldung sagt nicht, was jetzt gebraucht wird: %v", err)
	}
	if restored == nil {
		t.Fatal("der Mandant muss trotz fehlenden Schlüssels angemeldet bleiben")
	}
	if !b.IsLocked() {
		t.Error("der wiederhergestellte Mandant müsste gesperrt sein")
	}
	if found := b.GetTenants(); len(found) != 2 {
		t.Errorf("die Mandantenliste führt %d Einträge statt zwei", len(found))
	}

	originTypes := auditTypesAt(t, origin.DataDir)
	if originTypes["HASH_CHAIN"] {
		t.Error("die Prüfung lief auf den Büchern des vorherigen Mandanten")
	}
	if originTypes["RESTORE"] {
		t.Error("der Vermerk über die Wiederherstellung steht im Protokoll des vorherigen Mandanten")
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
//
// Ein neben seinem Ursprung wiederhergestellter Mandant führt eine neue
// Listenkennung und behält den Schlüssel des Ursprungs. Trüge seine Sicherung
// die Listenkennung, suchte der Prüflauf das Geheimnis unter einer Kennung, zu
// der keines im Schlüsselbund liegt: eine heile Sicherung käme als unlesbar
// zurück, und die Wiederherstellung daraus ließe den Mandanten gesperrt.
func TestBackupOfARestoredTenantStaysVerifiableAndRestorable(t *testing.T) {
	b, origin, zipPath := backupBridge(t)
	base := t.TempDir()

	restored, err := b.RestoreFromBackup(zipPath, filepath.Join(base, "wiederhergestellt"))
	if err != nil {
		t.Fatalf("Wiederherstellung neben dem Ursprung: %v", err)
	}
	if restored.ID == origin.ID {
		t.Fatal("der wiederhergestellte Mandant trägt dieselbe Listenkennung wie sein Ursprung")
	}
	if restored.VaultID() != origin.ID {
		t.Fatalf("der wiederhergestellte Mandant sucht seinen Schlüssel unter %q statt unter %q",
			restored.VaultID(), origin.ID)
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

	// backup.json muss die Schlüsselkennung nennen und nicht die Listenkennung.
	meta, err := service.ReadBackupMeta(run.Target)
	if err != nil {
		t.Fatalf("backup.json lesen: %v", err)
	}
	if meta.TenantID != origin.ID {
		t.Errorf("backup.json nennt den Mandanten %q; im Schlüsselbund liegt das Geheimnis unter %q",
			meta.TenantID, origin.ID)
	}

	// „Sicherung prüfen" muss die Kette nachrechnen können — dazu braucht sie
	// den Schlüssel.
	check, err := b.VerifyBackup(run.Target)
	if err != nil {
		t.Fatalf("Sicherung prüfen: %v", err)
	}
	if check == nil || !check.Success {
		t.Fatalf("die Prüfung meldet eine heile Sicherung als fehlerhaft: %+v", check)
	}

	// Und die Wiederherstellung daraus darf den Mandanten nicht gesperrt
	// zurücklassen.
	again, err := b.RestoreFromBackup(run.Target, filepath.Join(base, "wiederhergestellt-2"))
	if err != nil {
		t.Fatalf("zweite Wiederherstellung: %v", err)
	}
	if b.IsLocked() {
		t.Error("der zum zweiten Mal wiederhergestellte Mandant ist gesperrt, obwohl sein Schlüssel vorliegt")
	}
	if again.VaultID() != origin.ID {
		t.Errorf("der zweite wiederhergestellte Mandant sucht seinen Schlüssel unter %q statt unter %q",
			again.VaultID(), origin.ID)
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
