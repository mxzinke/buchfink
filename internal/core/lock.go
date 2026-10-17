package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
)

// WriteOption ändert, was Tenant.Write vor dem Vorgang prüft.
type WriteOption func(*writeOptions)

type writeOptions struct {
	examinerExempt bool
	appOnly        bool
}

// ExaminerExempt lässt einen Schreibvorgang im Prüfermodus zu. Nur für das
// Ein- und Ausschalten des Modus selbst, den Vermerk eines Sicherungslaufs
// und ähnliche Nachweise, die die Buchführung nicht ändern.
func ExaminerExempt() WriteOption { return func(o *writeOptions) { o.examinerExempt = true } }

// AppOnly behält den Vorgang der Bedienung in der App vor: Festschreibung,
// Übermittlungsbestätigung, Aufstellen/Feststellen/Offenlegen, Stammdaten des
// Unternehmens, Prüfermodus. Jeder andere Erfassungsweg wird abgewiesen.
func AppOnly() WriteOption { return func(o *writeOptions) { o.appOnly = true } }

// writeLock ordnet die Schreiber eines Mandanten. Ein Kanal statt eines
// Mutex, damit ein Wartender über seinen Kontext abbrechen kann.
type writeLock chan struct{}

func newWriteLock() writeLock { return make(writeLock, 1) }

func (l writeLock) lock(ctx context.Context) error {
	select {
	case l <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l writeLock) unlock() { <-l }

// channelOf liest den Erfassungsweg eines Einstiegs in den Kern. Fehlt er oder
// ist er keiner der bekannten (ui, system, agent:<kurzname>), wird der Vorgang
// abgewiesen: was hier hereinkommt, steht an Buchung und Protokoll.
func channelOf(ctx context.Context) (actor.Channel, error) {
	channel, ok := actor.ChannelOf(ctx)
	if !ok {
		return "", ErrNoChannel
	}
	if _, err := actor.ParseChannel(string(channel)); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnknownChannel, err)
	}
	return channel, nil
}

type inWriteKey struct{}

// markInWrite kennzeichnet den Kontext eines laufenden Schreibvorgangs dieses
// Mandanten. Ein Write darin läuft in derselben Transaktion weiter, statt auf
// die eigene Sperre zu warten.
func markInWrite(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, inWriteKey{}, t)
}

func inWriteOf(ctx context.Context, t *Tenant) bool {
	owner, _ := ctx.Value(inWriteKey{}).(*Tenant)
	return owner == t
}

// InWrite meldet, ob der Kontext in einem Schreibvorgang läuft. Netzaufrufe
// gehören nicht dorthin: sie hielten die Schreibsperre für die Dauer eines
// Zeitlimits.
func InWrite(ctx context.Context) bool {
	_, ok := ctx.Value(inWriteKey{}).(*Tenant)
	return ok
}

// leases zählt die laufenden Vorgänge eines Mandanten. Nach Beginn des
// Schließens gibt es keine neue Lease mehr; Close wartet auf die laufenden.
type leases struct {
	mu      sync.Mutex
	active  int
	closing bool
	idle    chan struct{}
}

func (l *leases) acquire() (func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closing {
		return nil, ErrTenantClosing
	}
	l.active++
	var once sync.Once
	return func() { once.Do(l.release) }, nil
}

func (l *leases) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.active--
	if l.closing && l.active == 0 && l.idle != nil {
		close(l.idle)
		l.idle = nil
	}
}

// stop sperrt neue Leases.
func (l *leases) stop() {
	l.mu.Lock()
	l.closing = true
	l.mu.Unlock()
}

// drain sperrt neue Leases und wartet, bis die laufenden zurück sind oder die
// Frist abläuft. Es meldet, ob alle zurückkamen.
func (l *leases) drain(timeout time.Duration) bool {
	l.mu.Lock()
	l.closing = true
	if l.active == 0 {
		l.mu.Unlock()
		return true
	}
	if l.idle == nil {
		l.idle = make(chan struct{})
	}
	idle := l.idle
	l.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-idle:
		return true
	case <-timer.C:
		return false
	}
}
