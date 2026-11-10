package wailsbridge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/inbox"
	"github.com/buchfink/buchfink/internal/syncdir"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Der Eingang: ein Ordner je Mandant für Belege, E-Rechnungen und
// Kontoauszüge. Die App zeigt, was dort liegt, nimmt hineingezogene Dateien
// an und räumt nach der Ablage auf (policy.SettleInbox). Abgelegt wird auf
// Anweisung, nie von selbst.

// EventInboxAdded trägt ein InboxAdded, nachdem Dateien ins Fenster gezogen
// wurden.
const EventInboxAdded = "buchfink:eingang-abgelegt"

// InboxInfo ist der Stand des Eingangs mit den Ordnern der verbundenen Apps.
type InboxInfo struct {
	Path string `json:"path"`
	// Default: der Eingang ist der voreingestellte im Datenordner.
	Default      bool     `json:"default"`
	Count        int      `json:"count"`
	OldestDays   int      `json:"oldestDays"`
	ReadFolders  []string `json:"readFolders"`
	ExportFolder string   `json:"exportFolder"`
}

// InboxFile ist eine Datei im Eingang.
type InboxFile struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	// Kind ist beleg, e_rechnung, kontoauszug oder nicht_unterstuetzt.
	Kind string `json:"kind"`
	// ReceiptID nennt den Beleg, wenn derselbe Inhalt schon im Archiv liegt,
	// ReceiptYear sein Geschäftsjahr: die Belegliste zeigt nur eines.
	ReceiptID   *uint `json:"receiptId,omitempty"`
	ReceiptYear int   `json:"receiptYear,omitempty"`
}

// InboxAdded sind die Dateien, die in den Eingang kamen, und die übergangenen.
type InboxAdded struct {
	Names   []string        `json:"names"`
	Skipped []inbox.Skipped `json:"skipped"`
	Error   string          `json:"error,omitempty"`
}

// FolderCheck meldet einen Sync-Dienst über einem Ordner und den Hinweis dazu.
type FolderCheck struct {
	Sync    string `json:"sync"`
	Warning string `json:"warning"`
}

// inboxFolders sind Eingang, Lese- und Exportordner des offenen Mandanten.
func (b *BuchfinkBridge) inboxFolders(ctx context.Context, s *core.Services) (policy.Folders, error) {
	return policy.FoldersFor(ctx, s.AgentServices(), s.Tenant.DataDir, policy.ProtectedDirs(b.host.Config(), s.Tenant.DataDir))
}

func inboxInfo(f policy.Folders) (*InboxInfo, error) {
	f.EnsureInbox()
	n, oldest, err := f.InboxCount()
	if err != nil {
		return nil, fmt.Errorf("der Eingang %s ließ sich nicht lesen: %w", f.Inbox, err)
	}
	read, _ := emptyList(f.Read, nil)
	return &InboxInfo{Path: f.Inbox, Default: filepath.Clean(f.Inbox) == filepath.Join(f.DataDir, policy.InboxDir),
		Count: n, OldestDays: inbox.AgeDays(oldest, time.Now()), ReadFolders: read, ExportFolder: f.Export}, nil
}

// GetInbox liefert Pfad, Zahl der Dateien und Alter der ältesten im Eingang
// sowie Lese- und Exportordner.
func (b *BuchfinkBridge) GetInbox() (*InboxInfo, error) {
	return readOr(b, func() (*InboxInfo, error) { return &InboxInfo{ReadFolders: []string{}}, nil },
		func(ctx context.Context, s *core.Services) (*InboxInfo, error) {
			f, err := b.inboxFolders(ctx, s)
			if err != nil {
				return nil, err
			}
			return inboxInfo(f)
		})
}

// ListInbox liefert die Dateien im Eingang ohne Übernommen/, die älteste
// zuerst, mit Art und dem Beleg, falls derselbe Inhalt schon abgelegt ist.
func (b *BuchfinkBridge) ListInbox() ([]InboxFile, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]InboxFile, error) {
		f, err := b.inboxFolders(ctx, s)
		if err != nil {
			return nil, err
		}
		files, err := f.InboxFiles()
		if err != nil {
			return nil, fmt.Errorf("der Eingang %s ließ sich nicht lesen: %w", f.Inbox, err)
		}
		sums := make([]string, 0, len(files))
		for _, file := range files {
			if file.SHA256 != "" {
				sums = append(sums, file.SHA256)
			}
		}
		archived, err := s.Receipts.ReceiptIDsByFileHash(ctx, sums)
		if err != nil {
			return nil, err
		}
		out := make([]InboxFile, 0, len(files))
		for _, file := range files {
			row := InboxFile{Path: file.Path, Name: file.Name, Size: file.Size,
				ModifiedAt: file.ModTime.Format(time.RFC3339), Kind: string(file.Kind)}
			if id, ok := archived[file.SHA256]; ok && file.SHA256 != "" {
				row.ReceiptID = &id
				if r, err := s.Receipts.Get(ctx, id); err == nil {
					row.ReceiptYear = r.FiscalYear
				}
			}
			out = append(out, row)
		}
		return out, nil
	})
}

// OpenInbox öffnet den Eingang im Dateimanager des Systems.
func (b *BuchfinkBridge) OpenInbox() error {
	_, err := inspect(b, func(ctx context.Context, t *core.Tenant) (struct{}, error) {
		var dir string
		if err := t.Read(ctx, func(ctx context.Context, s *core.Services) error {
			f, err := b.inboxFolders(ctx, s)
			if err != nil {
				return err
			}
			f.EnsureInbox()
			dir = f.Inbox
			return nil
		}); err != nil {
			return struct{}{}, err
		}
		app := application.Get()
		if app == nil || app.Env == nil {
			return struct{}{}, errors.New("der Dateimanager lässt sich von hier aus nicht öffnen")
		}
		return struct{}{}, app.Env.OpenFileManager(dir, false)
	})
	return err
}

// AddToInbox kopiert Dateien in den Eingang, etwa die ins Fenster gezogenen:
// nur Belegformate, ein vorhandener Name bekommt ein Suffix. Abgelegt wird
// dabei nichts.
func (b *BuchfinkBridge) AddToInbox(paths []string) (*InboxAdded, error) {
	return inspect(b, func(ctx context.Context, t *core.Tenant) (*InboxAdded, error) {
		var dir string
		if err := t.Read(ctx, func(ctx context.Context, s *core.Services) error {
			f, err := b.inboxFolders(ctx, s)
			dir = f.Inbox
			return err
		}); err != nil {
			return nil, err
		}
		names, skipped, err := inbox.Add(dir, paths)
		if err != nil {
			return nil, err
		}
		if len(names) > 0 {
			t.Notify(ctx, core.ChangeInbox)
		}
		return &InboxAdded{Names: names, Skipped: skipped}, nil
	})
}

// SetInboxFolder wählt den Eingang; leer heißt <Datenordner>/Eingang. Der
// Ordner wird angelegt. Er darf keinen Datenordner, keine Sicherung und
// keinen Leseordner berühren; im eigenen Datenordner ist nur der
// voreingestellte erlaubt.
func (b *BuchfinkBridge) SetInboxFolder(path string) (*InboxInfo, error) {
	path = domain.CleanFolder(path)
	return write(b, func(ctx context.Context, s *core.Services) (*InboxInfo, error) {
		stored, err := s.Settings.AgentSettings(ctx)
		if err != nil {
			return nil, err
		}
		protected := policy.ProtectedDirs(b.host.Config(), s.Tenant.DataDir)
		if err := policy.CheckFolders(path, stored.ExtraReadFolders, s.Tenant.DataDir, protected); err != nil {
			return nil, err
		}
		dir := path
		if dir == "" {
			dir = policy.DefaultInbox(s.Tenant.DataDir)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("der Eingang %s ließ sich nicht anlegen: %w", dir, err)
		}
		if _, err := s.Settings.SetInboxFolder(ctx, path); err != nil {
			return nil, err
		}
		f, err := b.inboxFolders(ctx, s)
		if err != nil {
			return nil, err
		}
		return inboxInfo(f)
	}, core.AppOnly())
}

// CheckFolder meldet, ob ein Ordner von einem Dienst synchronisiert wird,
// und den Hinweis dazu. Die Einrichtung zeigt ihn für die Ablage; blockiert
// wird nichts.
func (b *BuchfinkBridge) CheckFolder(path string) FolderCheck {
	service := syncdir.Detect(path)
	return FolderCheck{Sync: service, Warning: syncdir.Warning(service)}
}

// settleInbox verschiebt nach einer Ablage aus der App, was aus dem Eingang
// kam. Scheitert es, bleibt die Ablage gültig; der Grund steht im Log.
func (b *BuchfinkBridge) settleInbox(paths []string) {
	ctx := uiContext()
	t, release, err := b.host.Acquire(ctx)
	if err != nil {
		return
	}
	defer release()
	var f policy.Folders
	if err := t.Read(ctx, func(ctx context.Context, s *core.Services) (err error) {
		f, err = b.inboxFolders(ctx, s)
		return err
	}); err != nil {
		slog.Warn("Eingang nicht aufgeräumt", "fehler", err)
		return
	}
	_, hints := policy.SettleInbox(ctx, t, f, paths, time.Now())
	for _, h := range hints {
		slog.Warn("Eingang nicht aufgeräumt", "hinweis", h)
	}
}

// AttachFileDrop nimmt Dateien an, die auf eine Ablagezone des Fensters
// gezogen werden (ein Element mit dem Attribut data-file-drop-target), und
// kopiert sie in den Eingang. Ein Drop außerhalb einer Ablagezone bleibt ohne
// Wirkung. Das Ergebnis geht als EventInboxAdded an die Oberfläche.
func AttachFileDrop(w *application.WebviewWindow, b *BuchfinkBridge) {
	w.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		ctx := e.Context()
		if ctx == nil {
			return
		}
		handleDrop(b, ctx.DroppedFiles(), ctx.DropTargetDetails() != nil, func(added InboxAdded) {
			if app := application.Get(); app != nil {
				app.Event.Emit(EventInboxAdded, added)
			}
		})
	})
}

// handleDrop kopiert die abgelegten Dateien in den Eingang und meldet das
// Ergebnis, auch einen Fehler.
func handleDrop(b *BuchfinkBridge, files []string, onTarget bool, report func(InboxAdded)) {
	if !onTarget || len(files) == 0 {
		return
	}
	added, err := b.AddToInbox(files)
	if err != nil {
		report(InboxAdded{Names: []string{}, Skipped: []inbox.Skipped{}, Error: err.Error()})
		return
	}
	report(*added)
}
