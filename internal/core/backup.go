package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Sicherung und Wiederherstellung. Ein Lauf dauert Minuten und hält deshalb
// nur eine Lease, keine Schreibsperre: VACUUM INTO liest einen in sich
// stimmigen Stand, und erst der Vermerk des Laufs geht über Write.

// use führt fn mit einer Lease aus, ohne Sperre und ohne Transaktion. Nur für
// die langen Läufe dieser Datei, die ihre Vermerke selbst über Write schreiben.
func (t *Tenant) use(ctx context.Context, fn func(context.Context, *Services) error) error {
	if _, ok := actor.ChannelOf(ctx); !ok {
		return ErrNoChannel
	}
	release, err := t.leases.acquire()
	if err != nil {
		return err
	}
	defer release()
	return fn(ctx, t.svc)
}

// recordInWrite ist die Klammer, in der der Sicherungsdienst Läufe vermerkt.
func (t *Tenant) recordInWrite(ctx context.Context, fn func(context.Context) error) error {
	return t.Write(ctx, func(ctx context.Context, _ *Services) error { return fn(ctx) }, ExaminerExempt())
}

// SetBackupDir legt den Zielordner der Sicherung des offenen Mandanten fest.
// Leer heißt: keine Sicherung eingerichtet.
func (h *Host) SetBackupDir(ctx context.Context, dir string) (domain.AppConfig, error) {
	if err := requireApp(ctx); err != nil {
		return h.Config(), err
	}
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return h.Config(), fmt.Errorf("kein aktiver Mandant gefunden")
	}
	defer release()
	if dir != "" {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return h.Config(), fmt.Errorf("%s ist kein vorhandener Ordner", dir)
		}
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
		// Eine Sicherung neben den Daten geht mit ihnen verloren.
		if sameOrInside(dir, t.DataDir()) {
			return h.Config(), fmt.Errorf(
				"der Sicherungsordner darf nicht im Datenordner liegen — sonst geht die Sicherung mit den Daten zusammen verloren")
		}
	}

	previous := h.backupDirOf(t.ID())
	var cfg domain.AppConfig
	err = t.Write(ctx, func(ctx context.Context, s *Services) error {
		if err := s.Repos.Audit.LogChange(ctx, domain.AuditActionUpdate, "BACKUP_DIR", t.ID(),
			fmt.Sprintf("Sicherungsordner gesetzt: %s", orDefault(dir, "(keiner)")),
			map[string]string{"backupDir": previous}, map[string]string{"backupDir": dir}); err != nil {
			return err
		}
		var err error
		cfg, err = h.updateTenantConfig(t.ID(), func(tc *domain.TenantConfig) error {
			tc.BackupDir = dir
			return nil
		})
		return err
	}, ExaminerExempt())
	if err != nil {
		return h.Config(), err
	}
	return cfg, nil
}

// RunBackup sichert den offenen Mandanten in seinen Sicherungsordner.
func (h *Host) RunBackup(ctx context.Context, kind domain.BackupKind) (*domain.BackupRun, error) {
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	dir := h.backupDirOf(t.ID())
	if dir == "" {
		return nil, fmt.Errorf("es ist kein Sicherungsordner eingerichtet. Bitte in den Einstellungen einen wählen")
	}

	// Zwei Läufe gleichzeitig schrieben zwei Kopien desselben Bestands und
	// stritten um den Arbeitsordner.
	t.backupMu.Lock()
	var run *domain.BackupRun
	err = t.use(ctx, func(ctx context.Context, s *Services) error {
		var err error
		run, err = s.Backup.CreateBackup(ctx, dir, kind)
		return err
	})
	t.backupMu.Unlock()
	if err != nil {
		return run, err
	}
	if _, err := h.updateTenantConfig(t.ID(), func(tc *domain.TenantConfig) error {
		tc.LastBackupAt = run.FinishedAt.Format(time.RFC3339)
		return nil
	}); err != nil {
		h.logger.Warn("Zeitpunkt der Sicherung ließ sich nicht merken", "mandant", t.ID(), "fehler", err)
	}
	return run, nil
}

// RunDueBackup sichert, wenn eine Sicherung fällig ist: beim Start nach dem
// Abstand von 24 Stunden, beim Beenden zusätzlich, wenn sich seit der letzten
// Sicherung etwas geändert hat.
func (h *Host) RunDueBackup(ctx context.Context, onShutdown bool) {
	ctx = systemContext(ctx)
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return
	}
	defer release()
	if h.backupDirOf(t.ID()) == "" {
		return
	}
	due := false
	if err := t.Read(ctx, func(ctx context.Context, s *Services) error {
		due = s.Backup.IsDue(ctx, time.Now())
		if !due && onShutdown {
			last, err := s.Backup.LastSuccessful(ctx)
			if err != nil {
				last = nil
			}
			due = s.Backup.HasChangesSince(ctx, last)
		}
		return nil
	}); err != nil || !due {
		return
	}
	if _, err := h.RunBackup(ctx, domain.BackupKindAutomatic); err != nil {
		h.logger.Warn("die automatische Sicherung ist fehlgeschlagen", "mandant", t.ID(), "fehler", err)
	}
}

// VerifyBackup ist der Wiederherstellungstest einer vorhandenen Sicherung.
func (h *Host) VerifyBackup(ctx context.Context, zipPath string) (*domain.BackupRun, error) {
	if zipPath == "" {
		return nil, fmt.Errorf("keine Sicherungsdatei gewählt")
	}
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	var run *domain.BackupRun
	err = t.use(ctx, func(ctx context.Context, s *Services) error {
		var err error
		run, err = s.Backup.VerifyBackup(ctx, zipPath)
		return err
	})
	return run, err
}

// RestoreBackup entpackt eine Sicherung in einen leeren Ordner, meldet ihn als
// Mandanten an und prüft ihn: Hash-Ketten und Belegdateien. Findet die Prüfung
// etwas, bleibt der Mandant angemeldet und die Meldung sagt, was sie fand;
// die Daten wieder zu entfernen wäre der zweite Datenverlust.
func (h *Host) RestoreBackup(ctx context.Context, zipPath, targetDir string) (*domain.TenantConfig, error) {
	if err := requireApp(ctx); err != nil {
		return nil, err
	}
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()

	meta, err := service.ReadBackupMeta(zipPath)
	if err != nil {
		return nil, err
	}

	// Der Lauf wird beim vorher offenen Mandanten vermerkt; ohne offenen
	// Mandanten (Startbildschirm, anderer Rechner) nur beim wiederhergestellten.
	var run *domain.BackupRun
	if current, release, err := h.Acquire(ctx); err == nil {
		err = current.use(ctx, func(ctx context.Context, s *Services) error {
			var err error
			run, err = s.Backup.RestoreFromBackup(ctx, zipPath, targetDir)
			return err
		})
		release()
		if err != nil {
			return nil, err
		}
	} else {
		run, err = service.NewBackupService(nil, nil, nil, "", "", "").RestoreFromBackup(ctx, zipPath, targetDir)
		if err != nil {
			return nil, err
		}
	}

	tc, restored, err := h.register(ctx, filepath.Join(targetDir, "buchfink.sqlite"), meta.TenantName,
		domain.MigrationKindRestore)
	if err != nil {
		return nil, fmt.Errorf("die Daten wurden wiederhergestellt, konnten aber nicht als Mandant angemeldet werden: %w", err)
	}
	if err := restored.Write(ctx, func(ctx context.Context, s *Services) error {
		return s.Backup.RecordRun(ctx, run)
	}, ExaminerExempt()); err != nil {
		return tc, fmt.Errorf("die Daten wurden wiederhergestellt, der Lauf ließ sich aber nicht vermerken: %w", err)
	}

	problems := make([]string, 0, 2)
	_ = restored.Read(ctx, func(ctx context.Context, s *Services) error {
		if result, err := s.Journal.VerifyIntegrity(ctx); err != nil {
			problems = append(problems, fmt.Sprintf("Die Hash-Chain konnte nicht geprüft werden: %v", err))
		} else if !result.IsValid {
			problems = append(problems, result.Message)
		}
		if result, err := s.Receipts.VerifyReceiptFiles(ctx); err != nil {
			problems = append(problems, fmt.Sprintf("Die Belegdateien konnten nicht geprüft werden: %v", err))
		} else if !result.IsValid {
			problems = append(problems, result.Message)
		}
		return nil
	})

	outcome := "ohne Beanstandung"
	if len(problems) > 0 {
		outcome = "mit Beanstandungen: " + strings.Join(problems, " ")
	}
	if err := restored.Write(ctx, func(ctx context.Context, s *Services) error {
		return s.Repos.Audit.Log(ctx, domain.AuditActionImport, "RESTORE", tc.ID,
			fmt.Sprintf("Sicherung %s nach %s wiederhergestellt und geprüft — %s",
				filepath.Base(zipPath), targetDir, outcome))
	}, ExaminerExempt()); err != nil {
		return tc, fmt.Errorf("die Daten wurden wiederhergestellt, die Prüfung ließ sich aber nicht protokollieren: %w", err)
	}
	if len(problems) > 0 {
		return tc, fmt.Errorf(
			"die Daten wurden wiederhergestellt und der Mandant ist angemeldet, die Prüfung hat aber etwas gefunden: %s",
			strings.Join(problems, " "))
	}
	return tc, nil
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
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
