package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// changeLog sammelt die gemeldeten Vorgänge.
type changeLog struct {
	mu      sync.Mutex
	changes []Change
}

func (l *changeLog) add(c Change) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.changes = append(l.changes, c)
}

// take liefert die Meldungen seit dem letzten Aufruf.
func (l *changeLog) take() []Change {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.changes
	l.changes = nil
	return out
}

func newObservedHost(t *testing.T) (*Host, *changeLog) {
	t.Helper()
	log := &changeLog{}
	base := t.TempDir()
	h, err := NewHost(context.Background(), HostOptions{
		Config:   repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir:  base,
		Network:  OfflineNetwork(),
		OnChange: log.add,
	})
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	t.Cleanup(func() { _ = h.Close(context.Background()) })
	return h, log
}

// Jeder gelungene Vorgang meldet sich mit Mandant, Erfassungsweg und Art,
// gleich von wem; ein gescheiterter oder abgewiesener nicht.
func TestWriteReportsEachChange(t *testing.T) {
	h, log := newObservedHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	log.take()

	agent := agentCtx(t)
	for _, tcase := range []struct {
		name    string
		ctx     context.Context
		channel actor.Channel
	}{
		{"ui", uiCtx, actor.ChannelUI},
		{"agent", agent, actor.Channel("agent:claude-desktop")},
		{"system", systemCtx, actor.ChannelSystem},
	} {
		if err := tenant.Write(tcase.ctx, func(ctx context.Context, s *Services) error {
			_, err := post(ctx, s)
			return err
		}); err != nil {
			t.Fatalf("%s bucht: %v", tcase.name, err)
		}
		got := log.take()
		if len(got) != 1 {
			t.Fatalf("%s: %d Meldungen, erwartet eine: %+v", tcase.name, len(got), got)
		}
		c := got[0]
		if c.TenantID != tc.ID || c.Channel != tcase.channel ||
			!slices.Contains(c.Kinds, ChangeBookings) || !slices.Contains(c.Kinds, ChangeReceipts) {
			t.Errorf("%s: %+v", tcase.name, c)
		}
	}

	// Ein Vorgang in einem anderen gehört zu ihm: eine Meldung.
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		return tenant.Write(ctx, func(ctx context.Context, s *Services) error {
			_, err := s.Settings.SaveAgentSettings(ctx, domain.AgentSettings{DirectMaxGross: 5000})
			return err
		})
	}); err != nil {
		t.Fatal(err)
	}
	if got := log.take(); len(got) != 1 || !slices.Equal(got[0].Kinds, []ChangeKind{ChangeMasterData}) {
		t.Errorf("verschachtelt: %+v", got)
	}

	failed := errors.New("abgebrochen")
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := post(ctx, s); err != nil {
			return err
		}
		return failed
	}); !errors.Is(err, failed) {
		t.Fatalf("Fehlschlag: %v", err)
	}
	if err := tenant.Write(agent, func(ctx context.Context, s *Services) error {
		_, err := post(ctx, s)
		return err
	}, AppOnly()); !errors.Is(err, ErrAppOnly) {
		t.Fatalf("abgewiesen: %v", err)
	}
	if got := log.take(); len(got) != 0 {
		t.Errorf("Meldung ohne gelungenen Vorgang: %+v", got)
	}

	// Lesen meldet nichts, auch wenn es einen Zugriff protokolliert.
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		return s.Repos.Audit.Log(ctx, domain.AuditActionExport, "JOURNAL", "2026", "Journal herausgegeben")
	}); err != nil {
		t.Fatal(err)
	}
	if got := log.take(); len(got) != 0 {
		t.Errorf("Lesen hat gemeldet: %+v", got)
	}

	tenant.Notify(agent, ChangeInbox)
	if got := log.take(); len(got) != 1 || got[0].Channel != "agent:claude-desktop" ||
		!slices.Equal(got[0].Kinds, []ChangeKind{ChangeInbox}) {
		t.Errorf("Eingang: %+v", got)
	}
}

// Ein neuer Mandant bekommt seinen Eingang, voreingestellt im Datenordner.
func TestCreateTenantCreatesTheInbox(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	if info, err := os.Stat(filepath.Join(tc.DataDir, domain.DefaultInboxDir)); err != nil || !info.IsDir() {
		t.Fatalf("voreingestellter Eingang: %v", err)
	}
	tenant := openTenant(t, h)
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		inbox, err := s.Settings.InboxFolder(ctx)
		if err == nil && inbox != "" {
			t.Errorf("ohne Wahl gilt die Voreinstellung: %q", inbox)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}

	custom := filepath.Join(t.TempDir(), "Scans", "Buchfink")
	other, err := h.CreateTenant(uiCtx, "Zweite GmbH", filepath.Join(t.TempDir(), "daten"),
		domain.CompanySettings{CompanyName: "Zweite GmbH", FiscalYear: 2026}, WithInbox(custom))
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(custom); err != nil || !info.IsDir() {
		t.Fatalf("gewählter Eingang: %v", err)
	}
	if _, err := os.Stat(filepath.Join(other.DataDir, domain.DefaultInboxDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("mit eigenem Eingang entsteht keiner im Datenordner: %v", err)
	}
	if err := openTenant(t, h).Read(uiCtx, func(ctx context.Context, s *Services) error {
		inbox, err := s.Settings.InboxFolder(ctx)
		if err == nil && inbox != custom {
			t.Errorf("gespeicherter Eingang: %q", inbox)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.CreateTenant(uiCtx, "Dritte GmbH", filepath.Join(t.TempDir(), "daten"),
		domain.CompanySettings{CompanyName: "Dritte GmbH", FiscalYear: 2026}, WithInbox("relativ")); err == nil {
		t.Error("ein relativer Eingang wurde angenommen")
	}
}
