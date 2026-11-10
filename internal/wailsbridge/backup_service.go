package wailsbridge

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Sicherung und Wiederherstellung (§ 147 Abs. 1 AO, GoBD Rz. 103). Die Läufe
// selbst führt der Kern (core/backup.go): ohne Schreibsperre über ihre
// Laufzeit, der Vermerk eines Laufs als kurzer Schreibvorgang.

// GetBackupRuns liefert die letzten Sicherungs- und Prüfläufe.
func (b *BuchfinkBridge) GetBackupRuns() ([]domain.BackupRun, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.BackupRun, error) {
		return s.Backup.GetRuns(ctx, 20)
	})
}

// SetBackupDir legt den Zielordner der Sicherung fest. Leer heißt: keine
// Sicherung eingerichtet — die Aufgabenliste weist dann darauf hin.
func (b *BuchfinkBridge) SetBackupDir(dir string) (domain.AppConfig, error) {
	return b.host.SetBackupDir(uiContext(), dir)
}

// CreateBackup schreibt eine Sicherung von Hand.
func (b *BuchfinkBridge) CreateBackup() (*domain.BackupRun, error) {
	return b.host.RunBackup(uiContext(), domain.BackupKindManual)
}

// VerifyBackup ist der Wiederherstellungstest: eine vorhandene Sicherung wird
// in einen Temporärordner entpackt, geprüft und wieder abgeräumt.
func (b *BuchfinkBridge) VerifyBackup(zipPath string) (*domain.BackupRun, error) {
	return b.host.VerifyBackup(uiContext(), zipPath)
}

// RestoreFromBackup entpackt eine Sicherung in einen leeren Ordner, meldet ihn
// als Mandanten an und prüft Hash-Ketten und Belegdateien. Findet die Prüfung
// etwas, bleibt der Mandant angemeldet; die Meldung sagt, was sie fand.
func (b *BuchfinkBridge) RestoreFromBackup(zipPath, targetDir string) (*domain.TenantConfig, error) {
	return b.host.RestoreBackup(uiContext(), zipPath, targetDir)
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

// ServiceShutdown ist der Haken, den die Wails-Laufzeit beim Beenden ruft. Der
// MCP-Server hört zuerst auf, dann sichert der Kern, wenn seit der letzten
// Sicherung gearbeitet wurde (Host.Close); Wails wartet auf diesen Aufruf.
func (b *BuchfinkBridge) ServiceShutdown() error {
	ctx := actor.WithChannel(context.Background(), actor.ChannelSystem)
	return errors.Join(b.stopMCP(ctx), b.host.Close(ctx))
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
