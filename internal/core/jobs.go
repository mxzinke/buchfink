package core

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
)

// Clock ist die Uhr der Hintergrundläufe. Tests setzen eine eigene.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Takte der Hintergrundläufe.
const (
	timestampInterval   = 15 * time.Minute
	backupInterval      = time.Hour
	idempotencyInterval = 24 * time.Hour
)

// job ist ein Hintergrundlauf eines offenen Mandanten. Er läuft beim Start,
// wenn atStart gesetzt ist, und danach jeweils nach next.
type job struct {
	name    string
	atStart bool
	next    func(now time.Time) time.Duration
	run     func(ctx context.Context) error
}

func every(d time.Duration) func(time.Time) time.Duration {
	return func(time.Time) time.Duration { return d }
}

// untilMidnight ist die Zeit bis kurz nach der nächsten Mitternacht in Ortszeit:
// dann endet ein Prüfermodus, dessen letzter Tag gestern war.
func untilMidnight(now time.Time) time.Duration {
	y, m, d := now.Date()
	next := time.Date(y, m, d+1, 0, 0, 1, 0, now.Location())
	return next.Sub(now)
}

// jobs sind die Hintergrundläufe eines Mandanten.
type jobs struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   sync.WaitGroup
}

// cancelOnly bricht die Läufe ab, ohne auf sie zu warten.
func (j *jobs) cancelOnly() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cancel != nil {
		j.cancel()
	}
}

// stop beendet die Läufe und wartet, bis jeder zurück ist.
func (j *jobs) stop() {
	j.mu.Lock()
	cancel := j.cancel
	j.cancel = nil
	j.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	j.done.Wait()
}

// startJobs startet die Hintergrundläufe des Mandanten, der gerade offen
// wurde. Sie laufen im Erfassungsweg system und enden, wenn er schließt:
//
//   - Zeitstempel nachholen: beim Öffnen und alle 15 Minuten.
//   - Sicherung: beim Öffnen und stündlich, wenn sie fällig ist und ein Ziel
//     eingerichtet ist.
//   - Prüfermodus: nach jeder Mitternacht neu einlesen.
//   - Idempotenzschlüssel: beim Öffnen und täglich die abgelaufenen löschen.
func (h *Host) startJobs(t *Tenant) {
	if !h.opts.Jobs {
		return
	}
	ctx, cancel := context.WithCancel(actor.WithChannel(context.Background(), actor.ChannelSystem))
	t.jobs.mu.Lock()
	t.jobs.cancel = cancel
	t.jobs.mu.Unlock()

	for _, j := range []job{
		{name: "Zeitstempel nachholen", atStart: true, next: every(timestampInterval), run: func(ctx context.Context) error {
			_, err := t.RetryTimestamps(ctx)
			return err
		}},
		{name: "Sicherung", atStart: true, next: every(backupInterval), run: func(ctx context.Context) error {
			h.runDueBackup(ctx, t, false)
			return nil
		}},
		{name: "Prüfermodus", next: untilMidnight, run: t.loadExaminer},
		{name: "Idempotenzschlüssel", atStart: true, next: every(idempotencyInterval), run: func(ctx context.Context) error {
			return t.Write(ctx, func(ctx context.Context, s *Services) error {
				_, err := s.Idempotency.Prune(ctx)
				return err
			}, ExaminerExempt())
		}},
	} {
		t.jobs.done.Add(1)
		go func(j job) {
			defer t.jobs.done.Done()
			h.loop(ctx, t, j)
		}(j)
	}
}

func (h *Host) loop(ctx context.Context, t *Tenant, j job) {
	if j.atStart {
		h.runJob(ctx, t, j)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.clock.After(j.next(h.clock.Now())):
			h.runJob(ctx, t, j)
		}
	}
}

func (h *Host) runJob(ctx context.Context, t *Tenant, j job) {
	if ctx.Err() != nil {
		return
	}
	if err := j.run(ctx); err != nil && ctx.Err() == nil && !errors.Is(err, ErrTenantClosing) {
		h.logger.Warn("Hintergrundlauf fehlgeschlagen", "lauf", j.name, "mandant", t.ID(), "fehler", err)
	}
}
