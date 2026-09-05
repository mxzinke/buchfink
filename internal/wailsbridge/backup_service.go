package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Sicherung und Wiederherstellung.
//
// Die Aufbewahrungspflicht des § 147 Abs. 1 AO läuft zehn Jahre. Eine
// Festplatte hält das selten aus, und niemand merkt den Verlust, bevor er
// zählt. Die Sicherung ist deshalb nicht ein Zubehör der Anwendung, sondern
// Teil der Ordnungsmäßigkeit (GoBD Rz. 103).

// GetBackupRuns liefert die letzten Sicherungs- und Prüfläufe.
func (b *BuchfinkBridge) GetBackupRuns() ([]domain.BackupRun, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.backupSvc == nil {
		return make([]domain.BackupRun, 0), nil
	}
	return emptyList(b.backupSvc.GetRuns(context.Background(), 20))
}

// SetBackupDir legt den Zielordner der Sicherung fest. Leer heißt: keine
// Sicherung eingerichtet — die Aufgabenliste weist dann darauf hin.
func (b *BuchfinkBridge) SetBackupDir(dir string) (domain.AppConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	tenant := b.activeTenantLocked()
	if tenant == nil {
		return b.appConfig, fmt.Errorf("kein aktiver Mandant gefunden")
	}
	if dir != "" {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return b.appConfig, fmt.Errorf("%s ist kein vorhandener Ordner", dir)
		}
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
		// Der Sicherungsordner darf nicht der Datenordner sein: eine Sicherung,
		// die neben den Daten liegt, geht mit ihnen zusammen verloren. Genau
		// davor soll sie schützen.
		if sameOrInside(dir, tenant.DataDir) {
			return b.appConfig, fmt.Errorf(
				"der Sicherungsordner darf nicht im Datenordner liegen — sonst geht die Sicherung mit den Daten zusammen verloren")
		}
	}

	tenant.BackupDir = dir
	b.appConfig.SyncActiveTenant(time.Now().Format("2006-01-02"))
	if err := b.appCfgRepo.Save(&b.appConfig); err != nil {
		return b.appConfig, err
	}
	if b.auditRepo != nil {
		_ = b.auditRepo.Log(context.Background(), domain.AuditActionUpdate, "BACKUP_DIR", tenant.ID,
			fmt.Sprintf("Sicherungsordner gesetzt: %s", orDefault(dir, "(keiner)")))
	}
	return b.appConfig, nil
}

// CreateBackup schreibt eine Sicherung von Hand.
func (b *BuchfinkBridge) CreateBackup() (*domain.BackupRun, error) {
	return b.runBackup(domain.BackupKindManual)
}

// runBackup führt einen Sicherungslauf aus, ohne die Bridge zu sperren.
//
// Unter b.mu wird nur eingesammelt, was der Lauf braucht; das Kopieren der
// Datenbank (VACUUM INTO, transaktional konsistent) und das Packen der Belege
// laufen ohne Sperre, und erst der Vermerk über den letzten Lauf nimmt sie
// wieder kurz. Andernfalls stünde die Oberfläche bei jedem Start still, bis
// die Sicherung eines belegreichen Mandanten fertig ist — und „läuft im
// Hintergrund" wäre eine Behauptung ohne Deckung.
func (b *BuchfinkBridge) runBackup(kind domain.BackupKind) (*domain.BackupRun, error) {
	b.mu.RLock()
	svc := b.backupSvc
	tenantID, targetDir := "", ""
	if tenant := b.activeTenantLocked(); tenant != nil {
		tenantID, targetDir = tenant.ID, tenant.BackupDir
	}
	b.mu.RUnlock()

	if svc == nil {
		return nil, fmt.Errorf("die Sicherung ist noch nicht initialisiert")
	}
	if tenantID == "" {
		return nil, fmt.Errorf("kein aktiver Mandant gefunden")
	}
	if targetDir == "" {
		return nil, fmt.Errorf("es ist kein Sicherungsordner eingerichtet. Bitte in den Einstellungen einen wählen")
	}

	// Zwei Läufe gleichzeitig schrieben zwei Kopien desselben Bestands und
	// stritten um den Arbeitsordner.
	b.backupMu.Lock()
	run, err := svc.CreateBackup(context.Background(), targetDir, kind)
	b.backupMu.Unlock()
	if err != nil {
		return run, err
	}

	b.mu.Lock()
	for i := range b.appConfig.Tenants {
		if b.appConfig.Tenants[i].ID == tenantID {
			b.appConfig.Tenants[i].LastBackupAt = run.FinishedAt.Format(time.RFC3339)
			break
		}
	}
	b.appConfig.SyncActiveTenant(time.Now().Format("2006-01-02"))
	_ = b.appCfgRepo.Save(&b.appConfig)
	b.mu.Unlock()
	return run, nil
}

// VerifyBackup ist der Wiederherstellungstest: eine vorhandene Sicherung wird
// in einen Temporärordner entpackt, geprüft und wieder abgeräumt.
func (b *BuchfinkBridge) VerifyBackup(zipPath string) (*domain.BackupRun, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.backupSvc == nil {
		return nil, fmt.Errorf("die Sicherung ist noch nicht initialisiert")
	}
	if zipPath == "" {
		return nil, fmt.Errorf("keine Sicherungsdatei gewählt")
	}
	return b.backupSvc.VerifyBackup(context.Background(), zipPath)
}

// RestoreFromBackup entpackt eine Sicherung in einen leeren Ordner und meldet
// ihn als Mandanten an.
//
// Danach laufen Integritätsprüfung und Belegprüflauf. Sie sind kein Zubehör:
// eine Wiederherstellung, deren Ergebnis niemand geprüft hat, ist eine
// Vermutung. Schlägt eine der beiden fehl, bleibt der Mandant trotzdem
// angemeldet — der Anwender entscheidet, was mit den Daten geschieht; sie
// stillschweigend wieder zu entfernen wäre der zweite Datenverlust.
func (b *BuchfinkBridge) RestoreFromBackup(zipPath, targetDir string) (*domain.TenantConfig, error) {
	b.mu.Lock()
	svc := b.backupSvc
	b.mu.Unlock()

	if svc == nil {
		// Ohne offenen Mandanten gibt es keinen Dienst — die Wiederherstellung
		// muss trotzdem möglich sein, denn das ist der Fall, in dem sie
		// gebraucht wird. Der Lauf wird dann erst in der wiederhergestellten
		// Datenbank protokolliert.
		svc = b.restoreOnlyService()
	}

	// Kennung und Name des gesicherten Mandanten stehen in backup.json und
	// werden gebraucht, bevor der Ordner angemeldet wird: die Sicherung bringt
	// die Schlüsseldatei des Ursprungsmandanten mit, und das Geheimnis dazu
	// liegt im Schlüsselbund unter dessen Kennung. Eine neu vergebene Kennung
	// fände den Schlüssel nicht — der wiederhergestellte Mandant wäre auf
	// demselben Rechner gesperrt, auf dem er eben noch offen war.
	meta, err := service.ReadBackupMeta(zipPath)
	if err != nil {
		return nil, err
	}

	run, err := svc.RestoreFromBackup(context.Background(), zipPath, targetDir)
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(targetDir, "buchfink.sqlite")
	b.mu.Lock()
	tenant, err := b.registerTenantLocked(dbPath, domain.TenantConfig{
		ID: meta.TenantID, Name: meta.TenantName,
	})
	locked := b.locked
	openDir, activeID := b.dataDir, b.appConfig.ActiveTenantID
	journal, receipts, audit := b.journalSvc, b.receiptSvc, b.auditRepo
	restoredBackup := b.backupSvc
	b.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("die Daten wurden wiederhergestellt, konnten aber nicht als Mandant angemeldet werden: %w", err)
	}

	// Verschlüsselt und ohne Schlüssel auf diesem Rechner: dann ist keine
	// Prüfung möglich, und es wäre die falsche Auskunft, eine zu melden. Der
	// Mandant bleibt angemeldet, damit er nach dem Einlesen der
	// Wiederherstellungsdatei weiterläuft.
	if locked {
		return tenant, fmt.Errorf(
			"die Daten wurden wiederhergestellt und der Mandant %q ist angemeldet, aber verschlüsselt: "+
				"der Schlüssel liegt auf diesem Rechner nicht im Schlüsselbund. "+
				"Bitte die Wiederherstellungsdatei (buchfink-recovery-*.json) einlesen; "+
				"danach werden Hash-Chain und Belegdateien geprüft",
			tenant.Name)
	}

	// Der Lauf kommt zusätzlich in die Bücher des wiederhergestellten
	// Mandanten. Ausgeführt hat ihn der vorher offene (oder gar keiner, wenn
	// die Wiederherstellung vom Startbildschirm kam); in der Oberfläche
	// erschiene er dann beim falschen Mandanten, und der wiederhergestellte
	// stünde ohne einen einzigen Lauf da.
	if restoredBackup != nil && sameDir(openDir, targetDir) && activeID == tenant.ID {
		restoredBackup.RecordRun(context.Background(), run)
	}

	// Geprüft wird der wiederhergestellte Mandant und niemand sonst. Zeigen die
	// Dienste noch auf den vorher offenen Ordner, liefe die Prüfung über dessen
	// Bücher und meldete deren Ergebnis als Ergebnis der Wiederherstellung.
	if !sameDir(openDir, targetDir) || activeID != tenant.ID || journal == nil || receipts == nil {
		return tenant, fmt.Errorf(
			"die Daten wurden wiederhergestellt und der Mandant ist angemeldet, geprüft wurde er aber nicht: " +
				"er ist nach der Anmeldung nicht der offene Mandant. " +
				"Bitte zu ihm wechseln und Integritätsprüfung und Belegprüflauf ausführen")
	}

	problems := make([]string, 0, 2)
	if result, err := journal.VerifyIntegrity(context.Background()); err != nil {
		problems = append(problems, fmt.Sprintf("Die Hash-Chain konnte nicht geprüft werden: %v", err))
	} else if !result.IsValid {
		problems = append(problems, result.Message)
	}
	if result, err := receipts.VerifyReceiptFiles(context.Background()); err != nil {
		problems = append(problems, fmt.Sprintf("Die Belegdateien konnten nicht geprüft werden: %v", err))
	} else if !result.IsValid {
		problems = append(problems, result.Message)
	}

	if audit != nil {
		outcome := "ohne Beanstandung"
		if len(problems) > 0 {
			outcome = "mit Beanstandungen: " + joinSentences(problems)
		}
		_ = audit.Log(context.Background(), domain.AuditActionImport, "RESTORE", tenant.ID,
			fmt.Sprintf("Sicherung %s nach %s wiederhergestellt und geprüft — %s",
				filepath.Base(zipPath), targetDir, outcome))
	}

	if len(problems) > 0 {
		return tenant, fmt.Errorf(
			"die Daten wurden wiederhergestellt und der Mandant ist angemeldet, die Prüfung hat aber etwas gefunden: %s",
			joinSentences(problems))
	}
	return tenant, nil
}

// restoreOnlyService baut einen Sicherungsdienst ohne Datenbank. Er kann
// entpacken und prüfen, aber nichts protokollieren — mehr braucht die
// Wiederherstellung auf einem leeren Rechner nicht.
func (b *BuchfinkBridge) restoreOnlyService() *service.BackupService {
	return service.NewBackupService(nil, nil, nil, "", "", "")
}

// SelectBackupDirDialog wählt den Sicherungsordner.
func (b *BuchfinkBridge) SelectBackupDirDialog(title string) (string, error) {
	if title == "" {
		title = "Ordner für die Sicherung wählen (am besten ein anderes Laufwerk)"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle(title).
		PromptForSingleSelection()
}

// SelectBackupFileDialog wählt eine Sicherungsdatei.
func (b *BuchfinkBridge) SelectBackupFileDialog(title string) (string, error) {
	if title == "" {
		title = "Buchfink-Sicherung auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Buchfink-Sicherungen (*.zip)", "*.zip").
		SetTitle(title).
		PromptForSingleSelection()
}

// ServiceShutdown ist der Haken, den die Wails-Laufzeit beim Beenden ruft.
//
// Die Sicherung beim Beenden ist der eigentliche Zeitpunkt: dann steht der
// Bestand des Arbeitstages vollständig in den Büchern. Sie läuft synchron —
// Wails wartet auf diesen Aufruf, und eine Sicherung, die das Programm nicht
// abwartet, hätte es nie gegeben. Der Lauf beim Start bleibt als Auffangnetz
// für den Fall, dass die Anwendung abstürzt oder der Rechner ausgeht.
func (b *BuchfinkBridge) ServiceShutdown() error {
	b.runDueBackup()
	return nil
}

// runDueBackup sichert, wenn die letzte Sicherung älter als einen Tag ist.
//
// Zweimal gerufen: beim Beenden über ServiceShutdown und beim Start. Der Start
// ist der Auffangfall — wer die Anwendung abwürgt, bekommt sonst nie eine
// Sicherung. Doppelt sichert sie deshalb nicht: IsDue fragt den letzten
// gelungenen Lauf, und unmittelbar nach einem ist keiner fällig.
func (b *BuchfinkBridge) runDueBackup() {
	b.mu.RLock()
	svc := b.backupSvc
	hasTarget := false
	if tenant := b.activeTenantLocked(); tenant != nil {
		hasTarget = tenant.BackupDir != ""
	}
	b.mu.RUnlock()

	if svc == nil || !hasTarget {
		return
	}
	if !svc.IsDue(context.Background(), time.Now()) {
		return
	}
	if _, err := b.runBackup(domain.BackupKindAutomatic); err != nil {
		fmt.Fprintf(os.Stderr, "Die automatische Sicherung ist fehlgeschlagen: %v\n", err)
	}
}

// sameDir meldet, ob zwei Pfade denselben Ordner bezeichnen.
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, err1 := filepath.Abs(a)
	absB, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

// sameOrInside meldet, ob a gleich b ist oder in b liegt.
func sameOrInside(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, err1 := filepath.Abs(a)
	absB, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absB, absA)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) &&
		!hasParentPrefix(rel))
}

func hasParentPrefix(rel string) bool {
	return len(rel) >= 2 && rel[0] == '.' && rel[1] == '.'
}

// joinSentences hängt Meldungen zu einem lesbaren Satz zusammen.
func joinSentences(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
