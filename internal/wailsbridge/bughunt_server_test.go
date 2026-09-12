//go:build bughunt && server

package wailsbridge

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/repository"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TestBugHuntServer serves the real UI and bridge with an isolated configuration.
// Run explicitly with -tags server,bughunt -run '^TestBugHuntServer$' -timeout 0.
func TestBugHuntServer(t *testing.T) {
	dir := os.Getenv("BUCHFINK_BUGHUNT_DIR")
	if dir == "" || !filepath.IsAbs(dir) {
		t.Skip("BUCHFINK_BUGHUNT_DIR must name an absolute test directory")
	}
	repo := repository.NewAppConfigRepository(dir)
	cfg, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsConfigured {
		cfg.DataDir = filepath.Join(dir, "company")
	}
	b := &BuchfinkBridge{appCfgRepo: repo, appConfig: *cfg, dataDir: cfg.DataDir, currentYear: time.Now().Year()}
	if cfg.IsConfigured {
		b.currentYear = cfg.LastFiscalYear
		if err := b.initTenant(b.activeTenantLocked()); err != nil {
			t.Fatal(err)
		}
	}
	app := application.New(application.Options{
		Name:     "Buchfink Bug Hunting",
		Services: []application.Service{application.NewService(b)},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(os.DirFS("../../frontend/dist"))},
		Server:   application.ServerOptions{Host: "127.0.0.1", Port: 9250},
	})
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
}
