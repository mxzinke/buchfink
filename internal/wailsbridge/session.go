package wailsbridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// prepared schreibt nach einem Vorabruf: prepare holt ohne Schreibsperre, was
// der Vorgang aus dem Netz braucht, danach schreibt fn auf demselben
// Mandanten. Wechselt die App dazwischen den Mandanten, endet der Vorgang mit
// core.ErrTenantClosing, statt im neuen Mandanten mit Angaben aus dem alten zu
// schreiben.
func prepared[T any](
	b *BuchfinkBridge, prepare func(ctx context.Context, t *core.Tenant) error, fn serviceFn[T],
) (T, error) {
	return operate(b, func(ctx context.Context, t *core.Tenant) (T, error) {
		var out T
		if err := prepare(ctx, t); err != nil {
			return out, err
		}
		err := t.Write(ctx, func(ctx context.Context, s *core.Services) (err error) {
			out, err = fn(ctx, s)
			return err
		})
		return out, err
	})
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

// handOut gibt eine Datei heraus und protokolliert das (QUE-02). Die Datei
// liegt erst nach dem Protokolleintrag unter ihrem Namen: scheitert der
// Eintrag, bleibt nichts herausgegeben.
func (b *BuchfinkBridge) handOut(path string, data []byte, entityType, entityID, details string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(path), err)
	}
	_, writeErr := tmp.Write(data)
	if err := errors.Join(writeErr, tmp.Close()); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(path), err)
	}
	if err := b.logAccess(entityType, entityID, details); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(path), err)
	}
	return nil
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
