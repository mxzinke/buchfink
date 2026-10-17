package core

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// fakeClock lässt die Takte der Hintergrundläufe von Hand ablaufen.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []chan time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.waiters = append(c.waiters, ch)
	return ch
}

// fire lässt jeden wartenden Takt ablaufen, sobald alle n Läufe warten.
func (c *fakeClock) fire(t *testing.T, n int) {
	t.Helper()
	eventually(t, "alle Läufe warten auf ihren Takt", func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return len(c.waiters) >= n
	})
	c.mu.Lock()
	waiters := c.waiters
	c.waiters = nil
	c.now = c.now.Add(25 * time.Hour)
	now := c.now
	c.mu.Unlock()
	for _, ch := range waiters {
		ch <- now
	}
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("nicht eingetreten: %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Die Hintergrundläufe laufen am Kern und nach seiner Uhr: Zeitstempel
// nachholen, Prüfermodus nach Mitternacht neu lesen, abgelaufene
// Idempotenzschlüssel löschen.
func TestJobsRunOnTheClockOfTheCore(t *testing.T) {
	var online atomic.Bool
	clock := &fakeClock{now: time.Now()}
	base := t.TempDir()
	h, err := NewHost(context.Background(), HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base,
		Jobs:    true,
		Clock:   clock,
		Network: Network{
			Timestamp: func(context.Context, string, string) (*timestamp.Result, error) {
				if !online.Load() {
					return nil, ErrOffline
				}
				return &timestamp.Result{Token: []byte("token"), GenTime: time.Now().UTC(), TSAName: "Test-TSA"}, nil
			},
			Rates: offlineRates{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close(context.Background()) })
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	if _, err := tenant.CommitPeriod(uiCtx, CommitRequest{
		PeriodType: "month", PeriodLabel: "März 2026", CutoffDate: "2026-03-31", OverrideReason: "Testlauf",
	}); err != nil {
		t.Fatal(err)
	}
	until := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if err := s.Repos.Examiner.Create(ctx, &domain.ExaminerPeriod{
			From: time.Now().Format("2006-01-02"), Until: until, Reason: "an der App vorbei",
		}); err != nil {
			return err
		}
		return repository.NewIdempotencyRepository(tenant.db).Create(ctx, &domain.IdempotencyRecord{
			ClientID: "agent", Key: "schluessel-1", Operation: "test", RequestHash: "x",
			ExpiresAt: time.Now().Add(-time.Hour),
		})
	}); err != nil {
		t.Fatal(err)
	}
	if tenant.Examiner().Active {
		t.Fatal("der Prüfermodus gilt, bevor der Lauf ihn gelesen hat")
	}

	online.Store(true)
	clock.fire(t, 4)

	eventually(t, "Zeitstempel nachgeholt", func() bool {
		var pending []domain.Festschreibung
		_ = tenant.Read(systemCtx, func(ctx context.Context, s *Services) error {
			var err error
			pending, err = s.Repos.Festschreibung.FindPendingTimestamp(ctx)
			return err
		})
		return len(pending) == 0
	})
	eventually(t, "Prüfermodus nach Mitternacht gelesen", func() bool { return tenant.Examiner().Active })
	eventually(t, "abgelaufener Schlüssel gelöscht", func() bool {
		var n int64
		_ = tenant.Read(systemCtx, func(ctx context.Context, s *Services) error {
			return tenant.reader.WithContext(ctx).Model(&domain.IdempotencyRecord{}).Count(&n).Error
		})
		return n == 0
	})
}

// Hintergrundläufe und Mandantenwechsel vertragen sich: die Läufe des alten
// Mandanten enden mit ihm. Unter `go test -race` belegt der Test, dass dabei
// niemand ungeschützt auf den Mandanten zugreift.
func TestJobsEndWithTheirTenant(t *testing.T) {
	base := t.TempDir()
	h, err := NewHost(context.Background(), HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base,
		Jobs:    true,
		Network: OfflineNetwork(),
	})
	if err != nil {
		t.Fatal(err)
	}
	first := newTenant(t, h, "Erste GmbH")
	if _, err := h.SetBackupDir(uiCtx, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	second := newTenant(t, h, "Zweite GmbH")
	for i := 0; i < 20; i++ {
		id := first.ID
		if i%2 == 1 {
			id = second.ID
		}
		if _, err := h.Switch(uiCtx, id); err != nil {
			t.Fatalf("Wechsel %d: %v", i, err)
		}
	}

	// Der erste Mandant hat ein Sicherungsziel: beim Öffnen ist seine
	// Sicherung fällig und läuft im Hintergrund.
	if _, err := h.Switch(uiCtx, first.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "Sicherung beim Öffnen", func() bool {
		var runs []domain.BackupRun
		_ = h.Read(systemCtx, func(ctx context.Context, s *Services) error {
			var err error
			runs, err = s.Backup.GetRuns(ctx, 50)
			return err
		})
		return len(runs) > 0 && runs[0].Success && runs[0].Kind == domain.BackupKindAutomatic
	})
	if err := h.Close(context.Background()); err != nil {
		t.Fatalf("Schließen: %v", err)
	}
}
