package wailsbridge

import (
	"context"
	"errors"
	"fmt"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// Jede Bridge-Methode geht über den Kern: lesend über Tenant.Read, schreibend
// über Tenant.Write mit Schreibsperre, Prüfermodus und einer Transaktion je
// Vorgang. Der Erfassungsweg ist immer `ui`.
//
// Listen kommen leer und nie als nil aus der Bridge, auch ohne offenen
// Mandanten: ein nil-Slice wird über die Wails-Laufzeit zu `null`, und
// `null.map` nimmt im Render den ganzen Baum mit.

// uiContext ist der Kontext eines Aufrufs aus der Oberfläche.
func uiContext() context.Context {
	return actor.WithChannel(context.Background(), actor.ChannelUI)
}

type serviceFn[T any] func(ctx context.Context, s *core.Services) (T, error)

// read liest im offenen Mandanten.
func read[T any](b *BuchfinkBridge, fn serviceFn[T]) (T, error) {
	var out T
	err := b.host.Read(uiContext(), func(ctx context.Context, s *core.Services) error {
		var err error
		out, err = fn(ctx, s)
		return err
	})
	return out, err
}

// readList liest eine Liste; ohne offenen Mandanten ist sie leer.
func readList[E any](b *BuchfinkBridge, fn serviceFn[[]E]) ([]E, error) {
	out, err := read(b, fn)
	if errors.Is(err, core.ErrNoTenant) {
		err = nil
	}
	return emptyList(out, err)
}

// readOr liest; ohne offenen Mandanten antwortet fallback.
func readOr[T any](b *BuchfinkBridge, fallback func() (T, error), fn serviceFn[T]) (T, error) {
	out, err := read(b, fn)
	if errors.Is(err, core.ErrNoTenant) {
		return fallback()
	}
	return out, err
}

// write schreibt im offenen Mandanten als ein Vorgang.
func write[T any](b *BuchfinkBridge, fn serviceFn[T], opts ...core.WriteOption) (T, error) {
	var out T
	err := b.host.Write(uiContext(), func(ctx context.Context, s *core.Services) error {
		var err error
		out, err = fn(ctx, s)
		return err
	}, opts...)
	return out, err
}

// operate ruft einen Vorgang des Kerns, der selbst über Tenant.Write schreibt
// und dabei Netz und Schreibsperre trennt (Festschreibung, Bestätigungsanfrage,
// Vorabruf von Kursen). Für die Einordnung der Bridge-Methoden zählt er wie
// write.
func operate[T any](b *BuchfinkBridge, fn func(ctx context.Context, t *core.Tenant) (T, error)) (T, error) {
	var out T
	ctx := uiContext()
	t, release, err := b.host.Acquire(ctx)
	if err != nil {
		return out, err
	}
	defer release()
	return fn(ctx, t)
}

// prefetch holt vor einem Schreibvorgang, was er aus dem Netz braucht, ohne
// Schreibsperre. Ohne offenen Mandanten ist nichts zu holen; der
// Schreibvorgang meldet das.
func (b *BuchfinkBridge) prefetch(fn func(ctx context.Context, t *core.Tenant) error) error {
	_, err := operate(b, func(ctx context.Context, t *core.Tenant) (struct{}, error) {
		return struct{}{}, fn(ctx, t)
	})
	if errors.Is(err, core.ErrNoTenant) {
		return nil
	}
	return err
}

// inspect ruft einen lesenden Vorgang des Kerns.
func inspect[T any](b *BuchfinkBridge, fn func(ctx context.Context, t *core.Tenant) (T, error)) (T, error) {
	return operate(b, fn)
}

// change schreibt ohne Ergebnis.
func (b *BuchfinkBridge) change(fn func(ctx context.Context, s *core.Services) error, opts ...core.WriteOption) error {
	return b.host.Write(uiContext(), fn, opts...)
}

// viewYear ist das in der Kopfzeile gewählte Geschäftsjahr. Die Bridge reicht
// es an Ansichten durch; die Dienste kennen kein Jahr der Ansicht.
func (b *BuchfinkBridge) viewYear() int { return b.host.ViewYear() }

// yearOr nimmt das Jahr des Aufrufs, ohne Angabe das der Ansicht.
func (b *BuchfinkBridge) yearOr(year int) int {
	if year > 0 {
		return year
	}
	return b.viewYear()
}

// logAccess protokolliert die Herausgabe von Daten (QUE-02). Sie ist ein
// Lesevorgang; der Protokolleintrag ist der eine Schreibweg, den er hat.
func (b *BuchfinkBridge) logAccess(entityType, entityID, details string) error {
	_, err := read(b, func(ctx context.Context, s *core.Services) (struct{}, error) {
		return struct{}{}, s.Repos.Audit.Log(ctx, domain.AuditActionExport, entityType, entityID, details)
	})
	if err != nil {
		return fmt.Errorf("die Herausgabe ließ sich nicht protokollieren: %w", err)
	}
	return nil
}
